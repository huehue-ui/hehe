package handler

import (
	"campaignservice/internal/domain/models"
	"campaignservice/internal/infrastructure/db"
	"campaignservice/pkg/utils"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"campaignservice/internal/infrastructure/cache" // Added for MemoryCache
	"github.com/gin-gonic/gin"
	// "github.com/redis/go-redis/v9" // Commented out Redis
)

const (
	cacheTTLE = 5 * time.Minute // Cache TTL for delivery responses (reused for memory cache)
)

// DeliveryHandler struct now holds dependencies
type DeliveryHandler struct {
	db    *sql.DB
	memCache *cache.MemoryCache // Changed rdb to memCache
	// rdb *redis.Client // Commented out Redis client
}

// NewDeliveryHandler creates a new DeliveryHandler with dependencies
func NewDeliveryHandler(db *sql.DB, memCache *cache.MemoryCache) *DeliveryHandler {
	return &DeliveryHandler{db: db, memCache: memCache}
}

// ServeHTTPGin is the method that handles the HTTP requests for Gin
func (h *DeliveryHandler) ServeHTTPGin(c *gin.Context) {
	// ctx := c.Request.Context() // Context not directly used by MemoryCache Get/Set

	appID := c.Query("app")
	osParam := c.Query("os")
	country := c.Query("country")

	switch {
	case appID == "":
		utils.ErrorJSONGin(c, http.StatusBadRequest, utils.ErrMissingApp)
		return
	case osParam == "":
		utils.ErrorJSONGin(c, http.StatusBadRequest, utils.ErrMissingOS)
		return
	case country == "":
		utils.ErrorJSONGin(c, http.StatusBadRequest, utils.ErrMissingCountry)
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", strconv.Itoa(utils.DefaultApiPageLimit))

	page, errPage := strconv.Atoi(pageStr)
	if errPage != nil || page < 1 {
		page = 1
	}

	limit, errLimit := strconv.Atoi(limitStr)
	if errLimit != nil || limit < 1 || limit > 100 { // Max limit of 100
		limit = utils.DefaultApiPageLimit
	}
	offset := (page - 1) * limit

	// Create a cache key
	cacheKey := fmt.Sprintf("delivery:%s:%s:%s:page%d:limit%d", appID, osParam, country, page, limit)

	// Try to get from in-memory cache first
	cachedData, found := h.memCache.Get(cacheKey)
	if found {
		// Cache hit
		log.Printf("In-memory Cache HIT for key: %s", cacheKey)
		c.Header("X-Cache-Type", "IN_MEMORY_HIT")
		c.Data(http.StatusOK, "application/json; charset=utf-8", cachedData) // cachedData is []byte
		utils.RecordCacheHit() // Assuming this metric is generic for any cache type
		return
	}

	log.Printf("In-memory Cache MISS for key: %s", cacheKey)
	utils.RecordCacheMiss() // Assuming this metric is generic
	c.Header("X-Cache-Type", "MISS")


	// Cache miss, fetch from DB
	dbCampaigns, errDb := db.GetTargetedCampaigns(h.db, appID, country, osParam, limit, offset)
	if errDb != nil {
		utils.ErrorJSONGin(c, http.StatusInternalServerError, utils.InternalServerError)
		return
	}

	var response []models.DeliveryResponse
	for _, camp := range dbCampaigns {
		response = append(response, models.DeliveryResponse{
			CID: camp.CampaignID,
			Img: camp.ImageURL,
			CTA: camp.CallToAction,
		})
	}

	// Marshal response for caching and sending
	var responseBytesToCache []byte
	var marshalErr error

	if len(response) == 0 {
		// Cache empty response as well
		responseBytesToCache, marshalErr = json.Marshal([]models.DeliveryResponse{})
	} else {
		responseBytesToCache, marshalErr = json.Marshal(response)
	}

	if marshalErr != nil {
		log.Printf("Error marshalling delivery response: %v", marshalErr)
		utils.ErrorJSONGin(c, http.StatusInternalServerError, utils.InternalServerError)
		return
	}

	// Store in in-memory cache
	h.memCache.Set(cacheKey, responseBytesToCache, cacheTTLE)
	log.Printf("Successfully stored response in in-memory cache for key: %s", cacheKey)

	c.Data(http.StatusOK, "application/json; charset=utf-8", responseBytesToCache)
}
