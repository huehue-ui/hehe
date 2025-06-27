package handler

import (
	"campaignservice/internal/domain/models"
	"campaignservice/pkg/utils"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"campaignservice/internal/infrastructure/cache" // For MemoryCache
	"github.com/DATA-DOG/go-sqlmock"
	// "github.com/redis/go-redis/v9" // Commented out
	// "github.com/redis/go-redis/redismock/v9" // Commented out
	"github.com/stretchr/testify/assert"
	"github.com/gin-gonic/gin" // Added for Gin context
)

// Helper function to create a Gin context for testing
func setupGinTestContext(rr *httptest.ResponseRecorder, req *http.Request) *gin.Context {
	c, _ := gin.CreateTestContext(rr)
	c.Request = req
	return c
}


func TestDeliveryHandler_Gin(t *testing.T) { // Renamed test function
	// Setup common variables
	path := "/v1/delivery"

	t.Run("Missing 'app' query parameter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", path+"?os=ios&country=US", nil)
		rr := httptest.NewRecorder()
		c := setupGinTestContext(rr, req)

		// We don't need DB or Redis mocks for this validation error
		handler := NewDeliveryHandler(nil, nil)
		handler.ServeHTTPGin(c) // Call the Gin handler

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		expectedBody := `{"error":"` + utils.ErrMissingApp + `"}`
		assert.JSONEq(t, expectedBody, rr.Body.String())
	})

	t.Run("Missing 'os' query parameter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", path+"?app=app1&country=US", nil)
		rr := httptest.NewRecorder()
		c := setupGinTestContext(rr, req)
		handler := NewDeliveryHandler(nil, nil)
		handler.ServeHTTPGin(c)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		expectedBody := `{"error":"` + utils.ErrMissingOS + `"}`
		assert.JSONEq(t, expectedBody, rr.Body.String())
	})

	t.Run("Missing 'country' query parameter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", path+"?app=app1&os=ios", nil)
		rr := httptest.NewRecorder()
		c := setupGinTestContext(rr, req)
		handler := NewDeliveryHandler(nil, nil)
		handler.ServeHTTPGin(c)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		expectedBody := `{"error":"` + utils.ErrMissingCountry + `"}`
		assert.JSONEq(t, expectedBody, rr.Body.String())
	})

	t.Run("Successful response from cache", func(t *testing.T) {
		memCacheInstance := cache.NewMemoryCache()
		cacheKey := "delivery:app1:ios:US:page1:limit10"

		expectedCampaigns := []models.DeliveryResponse{
			{CID: "camp1", Img: "img1.url", CTA: "Action1"},
		}
		expectedBodyBytes, _ := json.Marshal(expectedCampaigns)

		// Pre-populate the in-memory cache
		memCacheInstance.Set(cacheKey, expectedBodyBytes, cacheTTLE)

		handler := NewDeliveryHandler(nil, memCacheInstance) // Pass nil for DB
		req, _ := http.NewRequest("GET", path+"?app=app1&os=ios&country=US", nil)
		rr := httptest.NewRecorder()
		c := setupGinTestContext(rr, req)
		handler.ServeHTTPGin(c)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.JSONEq(t, string(expectedBodyBytes), rr.Body.String())
		assert.Equal(t, "IN_MEMORY_HIT", rr.Header().Get("X-Cache-Type"))
		// No mock expectations to verify for MemoryCache in this direct way
	})

	t.Run("Cache miss, successful DB fetch, and cache SET", func(t *testing.T) {
		db, dbMock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		memCacheInstance := cache.NewMemoryCache()
		cacheKey := "delivery:app1:ios:US:page1:limit10"

		// Expected DB query and result
		rows := sqlmock.NewRows([]string{"campaign_id", "campaign_name", "image_url", "call_to_action"}).
			AddRow("camp1", "Campaign One", "img1.url", "Action1")

		dbMock.ExpectQuery("SELECT DISTINCT c.campaign_id, c.campaign_name, c.image_url, c.call_to_action").
									WithArgs("app1", "US", "ios", 10, 0).
									WillReturnRows(rows)

		// Cache should be empty initially for this key
		_, found := memCacheInstance.Get(cacheKey)
		assert.False(t, found, "Cache should be empty for %s before request", cacheKey)


		handler := NewDeliveryHandler(db, memCacheInstance)
		req, _ := http.NewRequest("GET", path+"?app=app1&os=ios&country=US", nil)
		rr := httptest.NewRecorder()
		c := setupGinTestContext(rr, req)
		handler.ServeHTTPGin(c)

		assert.Equal(t, http.StatusOK, rr.Code)
		expectedCampaigns := []models.DeliveryResponse{
			{CID: "camp1", Img: "img1.url", CTA: "Action1"},
		}
		expectedBodyBytes, _ := json.Marshal(expectedCampaigns)
		assert.JSONEq(t, string(expectedBodyBytes), rr.Body.String())
		assert.Equal(t, "MISS", rr.Header().Get("X-Cache-Type"))

		assert.NoError(t, dbMock.ExpectationsWereMet())

		// Verify item was cached
		cachedValue, foundAfter := memCacheInstance.Get(cacheKey)
		assert.True(t, foundAfter, "Value should be in cache after request for key %s", cacheKey)
		assert.JSONEq(t, string(expectedBodyBytes), string(cachedValue))
	})

	t.Run("Cache miss, DB error", func(t *testing.T) {
		db, dbMock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		memCacheInstance := cache.NewMemoryCache()
		cacheKey := "delivery:app1:ios:US:page1:limit10"

		dbMock.ExpectQuery("SELECT DISTINCT").
									WithArgs("app1", "US", "ios", 10, 0).
									WillReturnError(errors.New("DB error"))

		// Cache should be empty initially
		_, found := memCacheInstance.Get(cacheKey)
		assert.False(t, found, "Cache should be empty for %s before request", cacheKey)

		handler := NewDeliveryHandler(db, memCacheInstance)
		req, _ := http.NewRequest("GET", path+"?app=app1&os=ios&country=US", nil)
		rr := httptest.NewRecorder()
		c := setupGinTestContext(rr, req)
		handler.ServeHTTPGin(c)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.JSONEq(t, `{"error":"internal server error"}`, rr.Body.String())
		assert.Equal(t, "MISS", rr.Header().Get("X-Cache-Type"))

		assert.NoError(t, dbMock.ExpectationsWereMet())

		// Item should NOT be cached if DB error occurred
		_, foundAfter := memCacheInstance.Get(cacheKey)
		assert.False(t, foundAfter, "Value should NOT be in cache after DB error for key %s", cacheKey)
	})

	t.Run("Cache miss, DB returns no campaigns", func(t *testing.T) {
		db, dbMock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		memCacheInstance := cache.NewMemoryCache()
		cacheKey := "delivery:app1:ios:US:page1:limit10"

		rows := sqlmock.NewRows([]string{"campaign_id", "campaign_name", "image_url", "call_to_action"}) // No rows
		dbMock.ExpectQuery("SELECT DISTINCT").WithArgs("app1", "US", "ios", 10, 0).WillReturnRows(rows)

		// Cache should be empty initially
		_, found := memCacheInstance.Get(cacheKey)
		assert.False(t, found, "Cache should be empty for %s before request", cacheKey)


		handler := NewDeliveryHandler(db, memCacheInstance)
		req, _ := http.NewRequest("GET", path+"?app=app1&os=ios&country=US", nil)
		rr := httptest.NewRecorder()
		c := setupGinTestContext(rr, req)
		handler.ServeHTTPGin(c)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.JSONEq(t, "[]", rr.Body.String()) // Expect empty JSON array
		assert.Equal(t, "MISS", rr.Header().Get("X-Cache-Type"))

		assert.NoError(t, dbMock.ExpectationsWereMet())

		// Verify empty response was cached
		emptyResponseBytes, _ := json.Marshal([]models.DeliveryResponse{})
		cachedValue, foundAfter := memCacheInstance.Get(cacheKey)
		assert.True(t, foundAfter, "Empty response should be in cache after request for key %s", cacheKey)
		assert.JSONEq(t, string(emptyResponseBytes), string(cachedValue))
	})

}

// Helper to get the complex query string for GetTargetedCampaigns if needed, though sqlmock.AnyArg() can be used.
// func getExpectedTargetedCampaignsQuery() string {
// 	return `
// 		SELECT DISTINCT c.campaign_id, c.campaign_name, c.image_url, c.call_to_action
// 		FROM campaigns c
// 		LEFT JOIN targeting_rules tr_app_id ON c.campaign_id = tr_app_id.campaign_id AND tr_app_id.dimension = 'app_id'
// 		LEFT JOIN targeting_rules tr_country ON c.campaign_id = tr_country.campaign_id AND tr_country.dimension = 'country'
// 		LEFT JOIN targeting_rules tr_os ON c.campaign_id = tr_os.campaign_id AND tr_os.dimension = 'os'
// 		WHERE c.campaign_status = 'ACTIVE'
// 		  AND (
// 			EXISTS (SELECT 1 FROM targeting_rules tr WHERE tr.campaign_id = c.campaign_id AND tr.dimension = 'app_id' AND tr.type = 'include' AND tr.value = $1)
// 			OR NOT EXISTS (SELECT 1 FROM targeting_rules tr WHERE tr.campaign_id = c.campaign_id AND tr.dimension = 'app_id' AND tr.type = 'include')
// 		  )
// 		  AND NOT EXISTS (SELECT 1 FROM targeting_rules tr WHERE tr.campaign_id = c.campaign_id AND tr.dimension = 'app_id' AND tr.type = 'exclude' AND tr.value = $1)
// 		  AND (
// 			EXISTS (SELECT 1 FROM targeting_rules tr WHERE tr.campaign_id = c.campaign_id AND tr.dimension = 'country' AND tr.type = 'include' AND tr.value = $2)
// 			OR NOT EXISTS (SELECT 1 FROM targeting_rules tr WHERE tr.campaign_id = c.campaign_id AND tr.dimension = 'country' AND tr.type = 'include')
// 		  )
// 		  AND NOT EXISTS (SELECT 1 FROM targeting_rules tr WHERE tr.campaign_id = c.campaign_id AND tr.dimension = 'country' AND tr.type = 'exclude' AND tr.value = $2)
// 		  AND (
// 			EXISTS (SELECT 1 FROM targeting_rules tr WHERE tr.campaign_id = c.campaign_id AND tr.dimension = 'os' AND tr.type = 'include' AND tr.value = $3)
// 			OR NOT EXISTS (SELECT 1 FROM targeting_rules tr WHERE tr.campaign_id = c.campaign_id AND tr.dimension = 'os' AND tr.type = 'include')
// 		  )
// 		  AND NOT EXISTS (SELECT 1 FROM targeting_rules tr WHERE tr.campaign_id = c.campaign_id AND tr.dimension = 'os' AND tr.type = 'exclude' AND tr.value = $3)
// 		ORDER BY c.campaign_id
// 		LIMIT $4 OFFSET $5;
// 	`
// }

// Need to import "github.com/redis/go-redis/v9/redismock"
// The file is getting long, so I'll stop here and add more test cases or refine in subsequent steps.
// Key learning: direct dependency on *redis.Client makes mocking harder than an interface.
// `go-redis/redismock/v9` is the way for *redis.Client.
// The MockRedisCmdable with testify/mock is not the right pattern for go-redis v9 client.
// I've switched to using redismock in the test cases.
// The sqlmock.ExpectQuery will need the exact query string or a more robust matcher if the query changes often.
// Using a substring match like "SELECT DISTINCT c.campaign_id" is a pragmatic way to make it less brittle.
// The args matching for WithArgs is crucial.
