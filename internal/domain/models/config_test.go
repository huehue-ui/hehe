package models

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Helper to set/unset env vars for tests
	setEnv := func(key, value string) {
		os.Setenv(key, value)
	}
	unsetEnv := func(key string) {
		os.Unsetenv(key)
	}

	originalEnv := make(map[string]string)
	saveEnv := func(keys ...string) {
		for _, k := range keys {
			if val, ok := os.LookupEnv(k); ok {
				originalEnv[k] = val
			}
		}
	}
	restoreEnv := func() {
		for k, v := range originalEnv {
			os.Setenv(k, v)
		}
		// Clear any keys that were set during the test but not originally present
		// This part is tricky if we don't know all keys set by tests.
		// A simpler approach for isolated tests is to unset specific keys used in the test.
	}

	envKeys := []string{
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME",
		"APP_PORT", "REDIS_ADDR", "REDIS_PASSWORD", "REDIS_DB",
	}

	t.Run("All environment variables set", func(t *testing.T) {
		saveEnv(envKeys...)
		defer restoreEnv()
		defer func() { // Ensure all test-specific envs are unset
			for _, k := range envKeys { unsetEnv(k) }
		}()


		setEnv("DB_HOST", "testhost")
		setEnv("DB_PORT", "1234")
		setEnv("DB_USER", "testuser")
		setEnv("DB_PASSWORD", "testpass")
		setEnv("DB_NAME", "testdb")
		setEnv("APP_PORT", "9090")
		setEnv("REDIS_ADDR", "testredis:6380")
		setEnv("REDIS_PASSWORD", "redispass")
		setEnv("REDIS_DB", "1")

		cfg, err := LoadConfig()
		assert.NoError(t, err)

		assert.Equal(t, "testhost", cfg.DBHost)
		assert.Equal(t, "1234", cfg.DBPort)
		assert.Equal(t, "testuser", cfg.DBUser)
		assert.Equal(t, "testpass", cfg.DBPassword)
		assert.Equal(t, "testdb", cfg.DBName)
		assert.Equal(t, "9090", cfg.AppPort)
		assert.Equal(t, "testredis:6380", cfg.RedisAddr)
		assert.Equal(t, "redispass", cfg.RedisPassword)
		assert.Equal(t, 1, cfg.RedisDB)
	})

	t.Run("Some environment variables missing, defaults apply", func(t *testing.T) {
		saveEnv(envKeys...)
		defer restoreEnv()
		defer func() { // Ensure all test-specific envs are unset
			for _, k := range envKeys { unsetEnv(k) }
		}()

		// Set some, leave others for defaults
		setEnv("DB_HOST", "anotherhost")
		setEnv("APP_PORT", "7070")
		// Unset others that might have been set by previous tests or environment
		unsetEnv("DB_PORT")
		unsetEnv("DB_USER")
		unsetEnv("DB_PASSWORD")
		unsetEnv("DB_NAME")
		unsetEnv("REDIS_ADDR")
		unsetEnv("REDIS_PASSWORD")
		unsetEnv("REDIS_DB")


		cfg, err := LoadConfig()
		assert.NoError(t, err)

		assert.Equal(t, "anotherhost", cfg.DBHost)        // Set value
		assert.Equal(t, "5432", cfg.DBPort)               // Default
		assert.Equal(t, "user", cfg.DBUser)               // Default
		assert.Equal(t, "password", cfg.DBPassword)       // Default
		assert.Equal(t, "campaigndb", cfg.DBName)         // Default
		assert.Equal(t, "7070", cfg.AppPort)              // Set value
		assert.Equal(t, "localhost:6379", cfg.RedisAddr)  // Default
		assert.Equal(t, "", cfg.RedisPassword)            // Default
		assert.Equal(t, 0, cfg.RedisDB)                   // Default
	})

	t.Run("Invalid REDIS_DB value, uses default", func(t *testing.T) {
		saveEnv(envKeys...)
		defer restoreEnv()
		defer func() { // Ensure all test-specific envs are unset
			for _, k := range envKeys { unsetEnv(k) }
		}()

		setEnv("REDIS_DB", "not-an-int")

		cfg, err := LoadConfig()
		assert.NoError(t, err)
		assert.Equal(t, 0, cfg.RedisDB) // Should use default
	})

	t.Run("REDIS_DB not set, uses default", func(t *testing.T) {
		saveEnv(envKeys...)
		defer restoreEnv()
		defer func() { // Ensure all test-specific envs are unset
			for _, k := range envKeys { unsetEnv(k) }
		}()

		unsetEnv("REDIS_DB")

		cfg, err := LoadConfig()
		assert.NoError(t, err)
		assert.Equal(t, 0, cfg.RedisDB) // Should use default
	})

	// Note: Testing with .env file presence would require creating a temporary .env file
	// or ensuring it's not present during the test run. For simplicity, these tests
	// focus on environment variables directly, as godotenv.Load() in LoadConfig
	// doesn't fail if the .env file is absent.
}
