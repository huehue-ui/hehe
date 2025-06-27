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

	"github.com/gin-gonic/gin" // Added Gin
	"github.com/redis/go-redis/v9"
)

const (
	cacheTTLE = 5 * time.Minute // Cache TTL for delivery responses
)

// DeliveryHandler struct now holds dependencies
type DeliveryHandler struct {
	db  *sql.DB
	rdb *redis.Client
}

// NewDeliveryHandler creates a new DeliveryHandler with dependencies
func NewDeliveryHandler(db *sql.DB, rdb *redis.Client) *DeliveryHandler {
	return &DeliveryHandler{db: db, rdb: rdb}
}

// ServeHTTPGin is the method that handles the HTTP requests for Gin
func (h *DeliveryHandler) ServeHTTPGin(c *gin.Context) {
	ctx := c.Request.Context() // Use Gin context's underlying request context

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

	// Try to get from cache first
	cachedData, err := h.rdb.Get(ctx, cacheKey).Result()
	if err == nil && cachedData != "" {
		// Cache hit
		log.Printf("Cache HIT for key: %s", cacheKey)
		c.Header("X-Cache", "HIT")
		// Data in Redis is already JSON, so write it directly as raw JSON
		c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(cachedData))
		utils.RecordCacheHit()
		return
	} else if err != redis.Nil {
		log.Printf("Redis GET error for key %s: %v. Proceeding to DB.", cacheKey, err)
	} else {
		log.Printf("Cache MISS for key: %s", cacheKey)
		utils.RecordCacheMiss()
	}
	c.Header("X-Cache", "MISS")

	campaigns, err := db.GetTargetedCampaigns(h.db, appID, country, osParam, limit, offset)
	if err != nil {
		utils.ErrorJSONGin(c, http.StatusInternalServerError, utils.InternalServerError)
		return
	}

	var response []models.DeliveryResponse
	for _, camp := range campaigns { // Renamed loop variable to avoid conflict
		response = append(response, models.DeliveryResponse{
			CID: camp.CampaignID,
			Img: camp.ImageURL,
			CTA: camp.CallToAction,
		})
	}

	if len(response) == 0 {
		emptyResponseBytes, _ := json.Marshal([]models.DeliveryResponse{})
		errSet := h.rdb.Set(ctx, cacheKey, emptyResponseBytes, cacheTTLE).Err()
		if errSet != nil {
			log.Printf("Redis SET error for empty response (key %s): %v", cacheKey, errSet)
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", emptyResponseBytes)
		return
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling delivery response: %v", err)
		utils.ErrorJSONGin(c, http.StatusInternalServerError, utils.InternalServerError)
		return
	}

	errSet := h.rdb.Set(ctx, cacheKey, responseBytes, cacheTTLE).Err()
	if errSet != nil {
		log.Printf("Redis SET error for key %s: %v", cacheKey, errSet)
	} else {
		log.Printf("Successfully cached response for key: %s", cacheKey)
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", responseBytes)
}
