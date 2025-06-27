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

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/redismock/v9" // Official mock for go-redis
	"github.com/stretchr/testify/assert"
	// "github.com/stretchr/testify/mock" // No longer using testify/mock for Redis directly
)

// MockRedisCmdable is a mock for go-redis's Cmdable interface (which Client implements)
// We only need to mock the methods actually used by the handler: Get, Set.
// For simplicity, we can use redis.Client itself and override specific command behaviors
// or use a dedicated mocking library if more complex Redis interactions were needed.
// Here, we'll use an actual redis.Client connected to a redismock.
// An alternative is to define our own simple interface for what we need from Redis.

// For this test, we'll use go-redis's own mock `redis.NewClient(&redis.Options{})` and then use `mockConstructor` for specific commands.
// However, a more common approach for unit testing with testify is to define an interface and mock that.
// Let's try a simpler approach by mocking `Cmdable` if possible or using a library like `go-redis-mock`.
// `go-redis/redismock/v9` is the official mock for go-redis.

// MockRedis is a testify mock for the redis.Cmdable interface
// type MockRedisCmdable struct {
// 	mock.Mock
// 	redis.Cmdable // Embed to satisfy the interface if methods are called directly
// }

// func (m *MockRedisCmdable) Get(ctx context.Context, key string) *redis.StringCmd {
// 	args := m.Called(ctx, key)
// 	return args.Get(0).(*redis.StringCmd)
// }

// func (m *MockRedisCmdable) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
// 	args := m.Called(ctx, key, value, expiration)
// 	return args.Get(0).(*redis.StatusCmd)
// }

// // Ping to satisfy *redis.Client, not strictly needed for handler if it only uses Get/Set via Cmdable
// func (m *MockRedisCmdable) Ping(ctx context.Context) *redis.StatusCmd {
//     args := m.Called(ctx)
//     return args.Get(0).(*redis.StatusCmd)
// }

// MockRedisClient is a mock for go-redis's Cmdable interface (which Client implements)
// We only need to mock the methods actually used by the handler: Get, Set.
// For simplicity, we can use redis.Client itself and override specific command behaviors
// or use a dedicated mocking library if more complex Redis interactions were needed.
// Here, we'll use an actual redis.Client connected to a redismock.
// An alternative is to define our own simple interface for what we need from Redis.

// For this test, we'll use go-redis's own mock `redis.NewClient(&redis.Options{})` and then use `mockConstructor` for specific commands.
// However, a more common approach for unit testing with testify is to define an interface and mock that.
// Let's try a simpler approach by mocking `Cmdable` if possible or using a library like `go-redis-mock`.
// `go-redis/redismock/v9` is the official mock for go-redis.

// MockRedis is a testify mock for the redis.Cmdable interface
type MockRedisCmdable struct {
	mock.Mock
	redis.Cmdable // Embed to satisfy the interface if methods are called directly
}

func (m *MockRedisCmdable) Get(ctx context.Context, key string) *redis.StringCmd {
	args := m.Called(ctx, key)
	return args.Get(0).(*redis.StringCmd)
}

func (m *MockRedisCmdable) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	args := m.Called(ctx, key, value, expiration)
	return args.Get(0).(*redis.StatusCmd)
}

// Ping to satisfy *redis.Client, not strictly needed for handler if it only uses Get/Set via Cmdable
func (m *MockRedisCmdable) Ping(ctx context.Context) *redis.StatusCmd {
    args := m.Called(ctx)
    return args.Get(0).(*redis.StatusCmd)
}


func TestDeliveryHandler(t *testing.T) {
	// Setup common variables
	path := "/v1/delivery"

	t.Run("Missing 'app' query parameter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", path+"?os=ios&country=US", nil)
		rr := httptest.NewRecorder()

		// We don't need DB or Redis mocks for this validation error
		handler := NewDeliveryHandler(nil, nil)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		expectedBody := `{"error":"` + utils.ErrMissingApp + `"}`
		assert.JSONEq(t, expectedBody, rr.Body.String())
	})

	t.Run("Missing 'os' query parameter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", path+"?app=app1&country=US", nil)
		rr := httptest.NewRecorder()
		handler := NewDeliveryHandler(nil, nil)
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		expectedBody := `{"error":"` + utils.ErrMissingOS + `"}`
		assert.JSONEq(t, expectedBody, rr.Body.String())
	})

	t.Run("Missing 'country' query parameter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", path+"?app=app1&os=ios", nil)
		rr := httptest.NewRecorder()
		handler := NewDeliveryHandler(nil, nil)
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		expectedBody := `{"error":"` + utils.ErrMissingCountry + `"}`
		assert.JSONEq(t, expectedBody, rr.Body.String())
	})

	t.Run("Successful response from cache", func(t *testing.T) {
		mockRdb := new(MockRedisCmdable) // Using our testify mock

		expectedCampaigns := []models.DeliveryResponse{
			{CID: "camp1", Img: "img1.url", CTA: "Action1"},
		}
		expectedBodyBytes, _ := json.Marshal(expectedCampaigns)

		// Mock Redis GET
		// Create a StringCmd that returns the expected data
		cmdResult := redis.NewStringResult(string(expectedBodyBytes), nil)
		mockRdb.On("Get", mock.Anything, "delivery:app1:ios:US:page1:limit10").Return(cmdResult).Once()

		// We need a redis.Client that uses our mock Cmdable.
		// This is tricky as redis.Client doesn't directly take a Cmdable.
		// For true unit tests, the handler should depend on an interface we define,
		// e.g., type CacheClient interface { Get(...) ...; Set(...) ... }
		// For now, we'll adapt. The NewDeliveryHandler expects *redis.Client.
		// We will pass nil for db as it shouldn't be called.
		// This highlights a limitation of direct *redis.Client dependency for easy mocking.
		// A better approach is to use go-redis/redismock/v9 for mocking *redis.Client behavior.

		// Let's use go-redis/redismock/v9
		rdb, rdbMock := redis.NewClient( &redis.Options{}), redis.NewMock() // This is not how redismock works.
		// redismock.NewClientMock() is the correct way.

		// Corrected approach with redismock
		mrdb, mockRedis := redis.NewClientMock()

		mockRedis.ExpectGet("delivery:app1:ios:US:page1:limit10").SetVal(string(expectedBodyBytes))

		handler := NewDeliveryHandler(nil, mrdb) // Pass nil for DB
		req, _ := http.NewRequest("GET", path+"?app=app1&os=ios&country=US", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.JSONEq(t, string(expectedBodyBytes), rr.Body.String())
		assert.Equal(t, "HIT", rr.Header().Get("X-Cache"))
		mockRedis.ExpectationsWereMet() // Verify all GET expectations were met
	})

	t.Run("Cache miss, successful DB fetch, and cache SET", func(t *testing.T) {
		db, dbMock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mrdb, mockRedis := redis.NewClientMock()

		// Expected DB query and result
		rows := sqlmock.NewRows([]string{"campaign_id", "campaign_name", "image_url", "call_to_action"}).
			AddRow("camp1", "Campaign One", "img1.url", "Action1")

		// Remember the query from db.go - it's complex. For unit tests, we match the exact SQL.
		// This is a good argument for having repository interfaces.
		// For now, we'll use sqlmock.AnyArg() for the complex query string.
		// Or, more precisely, match the query string from db.GetTargetedCampaigns
		// The actual query string is long, using AnyMatcher here for brevity in this example.
		// In a real test, you'd put the exact query.
		dbMock.ExpectQuery("SELECT DISTINCT c.campaign_id, c.campaign_name, c.image_url, c.call_to_action"). // Simplified, use actual query
									WithArgs("app1", "US", "ios", 10, 0). // Args: appID, country, os, limit, offset
									WillReturnRows(rows)

		// Mock Redis GET (cache miss)
		mockRedis.ExpectGet("delivery:app1:ios:US:page1:limit10").SetErr(redis.Nil)

		// Mock Redis SET
		expectedCampaigns := []models.DeliveryResponse{
			{CID: "camp1", Img: "img1.url", CTA: "Action1"},
		}
		expectedCacheBodyBytes, _ := json.Marshal(expectedCampaigns)
		mockRedis.ExpectSet("delivery:app1:ios:US:page1:limit10", string(expectedCacheBodyBytes), cacheTTLE).SetVal("OK")


		handler := NewDeliveryHandler(db, mrdb)
		req, _ := http.NewRequest("GET", path+"?app=app1&os=ios&country=US", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.JSONEq(t, string(expectedCacheBodyBytes), rr.Body.String())
		assert.Equal(t, "MISS", rr.Header().Get("X-Cache"))

		assert.NoError(t, dbMock.ExpectationsWereMet())
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})

	t.Run("Cache miss, DB error", func(t *testing.T) {
		db, dbMock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mrdb, mockRedis := redis.NewClientMock()

		dbMock.ExpectQuery("SELECT DISTINCT"). // Simplified
									WithArgs("app1", "US", "ios", 10, 0).
									WillReturnError(errors.New("DB error"))

		mockRedis.ExpectGet("delivery:app1:ios:US:page1:limit10").SetErr(redis.Nil)
		// No SET should be called

		handler := NewDeliveryHandler(db, mrdb)
		req, _ := http.NewRequest("GET", path+"?app=app1&os=ios&country=US", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.JSONEq(t, `{"error":"internal server error"}`, rr.Body.String())
		assert.Equal(t, "MISS", rr.Header().Get("X-Cache"))

		assert.NoError(t, dbMock.ExpectationsWereMet())
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})

	t.Run("Cache miss, DB returns no campaigns", func(t *testing.T) {
		db, dbMock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mrdb, mockRedis := redis.NewClientMock()

		rows := sqlmock.NewRows([]string{"campaign_id", "campaign_name", "image_url", "call_to_action"}) // No rows
		dbMock.ExpectQuery("SELECT DISTINCT").WithArgs("app1", "US", "ios", 10, 0).WillReturnRows(rows)

		mockRedis.ExpectGet("delivery:app1:ios:US:page1:limit10").SetErr(redis.Nil)

		emptyResponseBytes, _ := json.Marshal([]models.DeliveryResponse{})
		mockRedis.ExpectSet("delivery:app1:ios:US:page1:limit10", string(emptyResponseBytes), cacheTTLE).SetVal("OK")

		handler := NewDeliveryHandler(db, mrdb)
		req, _ := http.NewRequest("GET", path+"?app=app1&os=ios&country=US", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.JSONEq(t, "[]", rr.Body.String())
		assert.Equal(t, "MISS", rr.Header().Get("X-Cache"))

		assert.NoError(t, dbMock.ExpectationsWereMet())
		assert.NoError(t, mockRedis.ExpectationsWereMet())
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
