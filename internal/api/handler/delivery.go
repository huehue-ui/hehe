package handler

import (
	"campaignservice/internal/domain/models"
	"campaignservice/internal/infrastructure/db"
	"campaignservice/pkg/utils"
	"encoding/json"
	"net/http"
	"strconv"
)

// DeliveryHandler handles incoming requests for campaign delivery.
// It expects 'app', 'os', and 'country' as query parameters.
// Optional 'page' and 'limit' parameters can be used for pagination.
func DeliveryHandler(w http.ResponseWriter, r *http.Request) {

	// Extract query parameters from the request URL
	q := r.URL.Query()
	app := q.Get("app")
	osParam := q.Get("os") // 'os' is a keyword in Go, so using osParam
	country := q.Get("country")

	// Validate required query parameters
	switch {
	case app == "":
		utils.ErrorJSON(w, http.StatusBadRequest, utils.ErrMissingApp)
		return
	case osParam == "":
		utils.ErrorJSON(w, http.StatusBadRequest, utils.ErrMissingOS)
		return
	case country == "":
		utils.ErrorJSON(w, http.StatusBadRequest, utils.ErrMissingCountry)
		return
	}

	// Parse pagination parameters
	// strconv.Atoi returns 0 if the string is not a valid number, which is handled by the logic below.
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Set default values for pagination if not provided or invalid
	if page < 1 {
		page = 1 // Default to page 1
	}
	if limit < 1 || limit > 100 { // Enforce a max limit of 100
		limit = utils.DefaultApiPageLimit // Default limit
	}
	// Calculate offset for database query
	offset := (page - 1) * limit

	// Establish a new database connection.
	// Note: This creates a new connection for each request.
	// For production, consider using a connection pool passed from main.
	dbConn := db.Connect()
	defer dbConn.Close() // Ensure the connection is closed after the handler finishes

	// Retrieve targeted campaigns from the database
	campaigns, err := db.GetTargetedCampaigns(dbConn, app, country, osParam, limit, offset)
	if err != nil {
		// If there's an error fetching campaigns, return an internal server error
		utils.ErrorJSON(w, http.StatusInternalServerError, utils.InternalServerError)
		return
	}

	// Prepare the response
	var response []models.DeliveryResponse
	for _, c := range campaigns {
		// Map database campaign model to the API response model
		response = append(response, models.DeliveryResponse{
			CID: c.CampaignID,
			Img: c.ImageURL,
			CTA: c.CallToAction,
		})
	}

	// Handle case where no campaigns are found
	if len(response) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // Return 200 OK with an empty array
		w.Write([]byte("[]"))        // Send an empty JSON array
		return
	}

	// Send the JSON response
	w.Header().Set("Content-Type", "application/json")
	// Encode the response slice directly to the response writer
	json.NewEncoder(w).Encode(response)
}
