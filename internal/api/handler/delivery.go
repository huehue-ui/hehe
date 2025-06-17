package handler

import (
	"campaignservice/internal/domain/models"
	"campaignservice/internal/infrastructure/db"
	"campaignservice/pkg/utils"
	"encoding/json"
	"net/http"
	"strconv"
)

func DeliveryHandler(w http.ResponseWriter, r *http.Request) {

	q := r.URL.Query()
	app := q.Get("app")
	osParam := q.Get("os")
	country := q.Get("country")

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

	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = utils.DefaultApiPageLimit
	}
	offset := (page - 1) * limit

	dbConn := db.Connect()
	defer dbConn.Close()

	campaigns, err := db.GetTargetedCampaigns(dbConn, app, country, osParam, limit, offset)
	if err != nil {
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
