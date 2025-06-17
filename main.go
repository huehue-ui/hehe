package main

import (
	"campaignservice/db"
	"campaignservice/handler"
	"campaignservice/utils"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	_ = godotenv.Load()
	d := db.Connect()
	defer d.Close()

	// Prometheus and monitoring setup
	http.Handle("/metrics", promhttp.Handler())
	utils.InitMetrics()

	r := chi.NewRouter()
	r.Route("/v1", func(v1 chi.Router) {
		v1.With(utils.MethodGuard("GET")).Get("/delivery", handler.DeliveryHandler)
	})

	log.Println("server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
