package models

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// AppConfig holds all configuration for the application
type AppConfig struct {
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	AppPort       string
	// RedisAddr     string // e.g., "localhost:6379" // Commented out
	// RedisPassword string // empty if no password    // Commented out
	// RedisDB       int    // e.g., 0                 // Commented out
}

// LoadConfig loads configuration from .env file and environment variables
func LoadConfig() (*AppConfig, error) {
	// Attempt to load .env file, but don't fail if it's not present
	// as environment variables might be set directly (e.g., in Docker)
	_ = godotenv.Load()

	cfg := &AppConfig{
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "user"),
		DBPassword:    getEnv("DB_PASSWORD", "password"),
		DBName:        getEnv("DB_NAME", "campaigndb"),
		AppPort:       getEnv("APP_PORT", "8080"),
		// RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"), // Commented out
		// RedisPassword: getEnv("REDIS_PASSWORD", ""),            // Commented out
		// RedisDB:       getEnvAsInt("REDIS_DB", 0),              // Commented out
	}
	return cfg, nil
}

// Helper function to get an environment variable or return a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("Environment variable %s not set, using default: %s", key, defaultValue)
	return defaultValue
}

// Helper function to get an environment variable as an integer or return a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	if valueStr != "" { // Only log if it was set but not parseable
		log.Printf("Warning: Environment variable %s (value: %s) is not a valid integer, using default: %d", key, valueStr, defaultValue)
	} else {
		// This case is covered by getEnv's log if the key wasn't found at all.
		// If getEnv returned its own default (empty string for non-int keys basically),
		// then we just use our int default.
	}
	return defaultValue
}
