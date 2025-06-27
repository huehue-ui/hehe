package main

import (
	"campaignservice/internal/api/handler"
	"campaignservice/internal/domain/models"
	"campaignservice/internal/infrastructure/db"
	"campaignservice/pkg/utils"
	"context" // Added for Redis
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall" // Added for graceful shutdown
	"time"    // Added for graceful shutdown

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9" // Added for Redis
)

func main() {
	cfg, err := models.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Database setup
	dbConnString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	d, err := db.Connect(dbConnString) // Assuming Connect takes the conn string
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer d.Close()

	// Redis client setup
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer rdb.Close()

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}
	log.Println("Successfully connected to Redis")

	// Prometheus and monitoring setup
	// Note: promhttp.Handler() is typically registered on a separate admin/metrics port
	// or using a separate ServeMux if your main router has middleware that might interfere.
	// For simplicity, keeping it on the main router for now.
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Println("Metrics server starting on :9091") // Example: run metrics on a different port
		if err := http.ListenAndServe(":9091", metricsMux); err != nil {
			log.Printf("Metrics server error: %s\n", err)
		}
	}()
	utils.InitMetrics()

	// Create the main router
	router := chi.NewRouter()

	// Initialize handlers with dependencies
	deliveryHandler := handler.NewDeliveryHandler(d, rdb) // Pass DB and Redis client

	// HTTP Metrics Middleware
	httpMetricsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := utils.NewResponseWriterWrapper(w) // To capture status code
			next.ServeHTTP(ww, r)
			duration := time.Since(start)
			utils.HTTPRequestDuration.With(prometheus.Labels{
				"path":   r.URL.Path, // Or use chi.RouteContext(r.Context()).RoutePattern() for patterned path
				"method": r.Method,
				"code":   fmt.Sprintf("%d", ww.StatusCode()),
			}).Observe(duration.Seconds())
		})
	}

	router.Use(httpMetricsMiddleware) // Apply middleware to all routes

	router.Route("/v1", func(v1 chi.Router) {
		v1.With(utils.MethodGuard("GET")).Get("/delivery", deliveryHandler.ServeHTTP) // Use the method from the handler instance
	})

	// Server setup
	server := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Server starting on port %s", cfg.AppPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
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
