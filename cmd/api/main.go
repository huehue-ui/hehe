package main

import (
	"campaignservice/internal/api/handler"
	"campaignservice/internal/infrastructure/db"
	"campaignservice/pkg/utils"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Load environment variables from .env file
	_ = godotenv.Load()

	// Establish database connection
	d := db.Connect()
	// Defer closing the database connection until the main function exits
	defer d.Close()

	// Prometheus and monitoring setup
	// Expose the /metrics endpoint for Prometheus to scrape
	http.Handle("/metrics", promhttp.Handler())
	// Initialize custom application metrics
	utils.InitMetrics()

	// Create a new Chi router
	r := chi.NewRouter()

	// Define API routes
	r.Route("/v1", func(v1 chi.Router) {
		// Apply MethodGuard middleware to allow only GET requests for the /delivery endpoint
		v1.With(utils.MethodGuard("GET")).Get("/delivery", handler.DeliveryHandler)
	})

	// Start the HTTP server
	log.Println("server started at :8080")
	// ListenAndServe will block until the server is stopped or an error occurs
	log.Fatal(http.ListenAndServe(":8080", r))
}
