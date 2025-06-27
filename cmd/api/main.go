package main

import (
	"campaignservice/internal/api/handler"
	"campaignservice/internal/domain/models"
	"campaignservice/internal/infrastructure/db"
	"campaignservice/pkg/utils"
	// "context" // No longer needed here after commenting out Redis ping
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall" // Added for graceful shutdown
	"time"    // Added for graceful shutdown

	"campaignservice/internal/infrastructure/cache" // Added for MemoryCache
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	// "github.com/redis/go-redis/v9" // Commented out Redis
)

// GinPrometheusMiddleware creates a Gin middleware for Prometheus metrics.
func GinPrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		// No need for ResponseWriterWrapper with Gin, status is in c.Writer.Status()
		c.Next() // Process request

		duration := time.Since(start)
		path := c.FullPath() // Get matched route path
		if path == "" {      // Fallback for unmatched routes
			path = c.Request.URL.Path
		}
		utils.HTTPRequestDuration.With(prometheus.Labels{
			"path":   path,
			"method": c.Request.Method,
			"code":   fmt.Sprintf("%d", c.Writer.Status()),
		}).Observe(duration.Seconds())
	}
}

// MethodGuardGin creates a Gin middleware to guard HTTP methods.
func MethodGuardGin(allowedMethod string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Note: utils.RequestCount from Chi middleware is not used here directly.
		// HTTPRequestDuration above provides similar request counting via histogram.
		// If a simple counter is still desired, it can be added here or in GinPrometheusMiddleware.
		if c.Request.Method != allowedMethod {
			utils.ErrorJSONGin(c, http.StatusMethodNotAllowed, utils.ErrMethodNotAllowed)
			c.Abort() // Prevent pending handlers from being called
			return
		}
		c.Next()
	}
}


func main() {
	cfg, err := models.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Database setup
	dbConnString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	d, err := db.Connect(dbConnString)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer d.Close()

	// Tune DB connection pool (example values)
	d.SetMaxOpenConns(25) // Max number of open connections to the database
	d.SetMaxIdleConns(25) // Max number of connections in the idle connection pool
	d.SetConnMaxLifetime(5*time.Minute) // Max amount of time a connection may be reused

	log.Println("Successfully connected to Database and configured connection pool.")

	// In-memory cache setup
	memCache := cache.NewMemoryCache()
	log.Println("In-memory cache initialized.")

	// // Redis client setup (Commented out)
	// rdb := redis.NewClient(&redis.Options{
	// 	Addr:     cfg.RedisAddr,
	// 	Password: cfg.RedisPassword,
	// 	DB:       cfg.RedisDB,
	// })
	// defer rdb.Close()

	// ctxPing, cancelPing := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancelPing()
	// if _, err := rdb.Ping(ctxPing).Result(); err != nil {
	// 	log.Fatalf("Could not connect to Redis: %v", err)
	// }
	// log.Println("Successfully connected to Redis")

	// Prometheus metrics server (runs on a separate goroutine and port)
	go func() {
		metricsRouter := gin.New() // Use a separate Gin engine for metrics for isolation
		metricsRouter.GET("/metrics", gin.WrapH(promhttp.Handler()))
		log.Println("Metrics server starting on :9091")
		if err := http.ListenAndServe(":9091", metricsRouter); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %s\n", err)
		}
	}()
	utils.InitMetrics() // Registers custom metrics like CacheActionsTotal, DBOperationDuration

	// Main application router (Gin)
	// gin.SetMode(gin.ReleaseMode) // Uncomment for production
	router := gin.Default() // Default includes logger and recovery middleware

	// Apply custom Prometheus middleware
	router.Use(GinPrometheusMiddleware())

	// Initialize handlers (DeliveryHandler needs to be adapted for Gin)
	// For now, assuming DeliveryHandler is a struct with ServeHTTPForGin(c *gin.Context)
	deliveryHandler := handler.NewDeliveryHandler(d, memCache) // Pass memCache instead of rdb

	// Routes
	v1 := router.Group("/v1")
	{
		// Apply method guard middleware to specific routes or groups
		deliveryRoutes := v1.Group("/delivery")
		deliveryRoutes.Use(MethodGuardGin("GET"))
		{
			// Assuming DeliveryHandler.ServeHTTP is adapted to gin.HandlerFunc
			// e.g. by having a method like deliveryHandler.GetDelivery(c *gin.Context)
			// This will be refactored in the handler step.
			// For now, let's assume a placeholder or direct adaptation.
			deliveryRoutes.GET("", deliveryHandler.ServeHTTPGin)
		}
	}

	// Server setup
	server := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Main application server starting on port %s", cfg.AppPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error for main server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Server shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server gracefully stopped")
}
