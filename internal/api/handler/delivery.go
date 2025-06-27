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

// ServeHTTP is the method that handles the HTTP requests
func (h *DeliveryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context() // Use request context

	q := r.URL.Query()
	appID := q.Get("app")
	osParam := q.Get("os")
	country := q.Get("country")

	switch {
	case appID == "":
		utils.ErrorJSON(w, http.StatusBadRequest, utils.ErrMissingApp)
		return
	case osParam == "":
		utils.ErrorJSON(w, http.StatusBadRequest, utils.ErrMissingOS)
		return
	case country == "":
		utils.ErrorJSON(w, http.StatusBadRequest, utils.ErrMissingCountry)
		return
	}

	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
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
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cachedData))
		utils.RecordCacheHit()
		return
	} else if err != redis.Nil {
		// Some other Redis error
		log.Printf("Redis GET error for key %s: %v. Proceeding to DB.", cacheKey, err)
		// Proceed to database, but log the error. Depending on policy, might return error.
	} else {
		log.Printf("Cache MISS for key: %s", cacheKey)
		utils.RecordCacheMiss()
	}
	w.Header().Set("X-Cache", "MISS")


	// Cache miss or Redis error, fetch from DB
	// Note: The original DeliveryHandler created a new DB connection for each request.
	// This is inefficient. The handler now uses the injected *sql.DB.
	campaigns, err := db.GetTargetedCampaigns(h.db, appID, country, osParam, limit, offset)
	if err != nil {
		// GetTargetedCampaigns already logs the error
		utils.ErrorJSON(w, http.StatusInternalServerError, utils.InternalServerError)
		return
	}

	var response []models.DeliveryResponse
	for _, c := range campaigns {
		response = append(response, models.DeliveryResponse{
			CID: c.CampaignID,
			Img: c.ImageURL,
			CTA: c.CallToAction,
		})
	}

	if len(response) == 0 {
		// Cache empty response as well to prevent repeated DB queries for non-existent data
		emptyResponseBytes, _ := json.Marshal([]models.DeliveryResponse{})
		errSet := h.rdb.Set(ctx, cacheKey, emptyResponseBytes, cacheTTLE).Err()
		if errSet != nil {
			log.Printf("Redis SET error for empty response (key %s): %v", cacheKey, errSet)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(emptyResponseBytes) // Return "[]"
		return
	}

	// Marshal response for caching and sending
	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling delivery response: %v", err)
		utils.ErrorJSON(w, http.StatusInternalServerError, utils.InternalServerError)
		return
	}

	// Store in cache
	errSet := h.rdb.Set(ctx, cacheKey, responseBytes, cacheTTLE).Err()
	if errSet != nil {
		// Log error but still serve the response from DB
		log.Printf("Redis SET error for key %s: %v", cacheKey, errSet)
	} else {
		log.Printf("Successfully cached response for key: %s", cacheKey)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}
