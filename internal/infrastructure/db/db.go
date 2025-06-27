package db

import (
	"campaignservice/internal/domain/models"
	"campaignservice/pkg/utils"
	"database/sql"
	"fmt"
	"log"
	// "os" // No longer needed for individual env vars here

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus" // Added for metrics
)

// Connect now accepts a connection string
func Connect(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("failed to open DB connection: %v", err)
		return nil, err
	}
	if err = db.Ping(); err != nil {
		log.Printf("DB not reachable: %v", err)
		// It might be better to return the error and let the caller decide to Fatal
		return nil, err
	}
	log.Println("Successfully connected to Database")
	return db, nil
}

// GetTargetedCampaigns retrieves campaigns based on targeting criteria.
// The SQL query logic needs careful review for correctness and efficiency,
// especially the GROUP BY and HAVING clauses for targeting.
// This version simplifies the argument passing to the query.
func GetTargetedCampaigns(db *sql.DB, appID, country, os string, limit int, offset int) ([]models.Campaign, error) {
	timer := prometheus.NewTimer(utils.DBOperationDuration.WithLabelValues("GetTargetedCampaigns"))
	defer timer.ObserveDuration()

	// The original query structure with bool_and and complex OR conditions inside HAVING
	// can be difficult to get right and maintain.
	// A common alternative is to find campaigns that DON'T match any exclusion rule
	// AND DO match all inclusion rules (or have no inclusion rule for a dimension).
	// However, sticking to the existing query structure for now, but using direct args.

	// Ensure limit and offset are sensible
	if limit <= 0 {
		limit = 10 // Default limit
	}
	if offset < 0 {
		offset = 0 // Default offset
	}

	// Note: The original query logic using LEFT JOIN and then complex HAVING clauses with bool_and
	// might not be the most efficient or straightforward way to implement the targeting.
	// It's trying to ensure that for each dimension, the campaign either:
	// 1. Has an 'include' rule matching the value.
	// 2. Does NOT have an 'exclude' rule matching the value.
	// 3. Has no specific rule for that dimension (implicitly included).

	// The query below is based on the one from the original file.
	// It is highly recommended to test this query thoroughly with various data scenarios.
	query := `
		SELECT DISTINCT c.campaign_id, c.campaign_name, c.image_url, c.call_to_action
		FROM campaigns c
		WHERE c.campaign_status = 'ACTIVE'
		  AND (
			EXISTS (SELECT 1 FROM targeting_rules tr WHERE tr.campaign_id = c.campaign_id AND tr.dimension = 'app_id' AND tr.type = 'include' AND tr.value = $1)
			OR NOT EXISTS (SELECT 1 FROM targeting_rules tr WHERE tr.campaign_id = c.campaign_id AND tr.dimension = 'app_id' AND tr.type = 'include')
		  )
		  AND NOT EXISTS (SELECT 1 FROM targeting_rules tr WHERE tr.campaign_id = c.campaign_id AND tr.dimension = 'app_id' AND tr.type = 'exclude' AND tr.value = $1)
		  AND (
			EXISTS (SELECT 1 FROM targeting_rules tr WHERE tr.campaign_id = c.campaign_id AND tr.dimension = 'country' AND tr.type = 'include' AND tr.value = $2)
			OR NOT EXISTS (SELECT 1 FROM targeting_rules tr WHERE tr.campaign_id = c.campaign_id AND tr.dimension = 'country' AND tr.type = 'include')
		  )
		  AND NOT EXISTS (SELECT 1 FROM targeting_rules tr WHERE tr.campaign_id = c.campaign_id AND tr.dimension = 'country' AND tr.type = 'exclude' AND tr.value = $2)
		  AND (
			EXISTS (SELECT 1 FROM targeting_rules tr WHERE tr.campaign_id = c.campaign_id AND tr.dimension = 'os' AND tr.type = 'include' AND tr.value = $3)
			OR NOT EXISTS (SELECT 1 FROM targeting_rules tr WHERE tr.campaign_id = c.campaign_id AND tr.dimension = 'os' AND tr.type = 'include')
		  )
		  AND NOT EXISTS (SELECT 1 FROM targeting_rules tr WHERE tr.campaign_id = c.campaign_id AND tr.dimension = 'os' AND tr.type = 'exclude' AND tr.value = $3)
		ORDER BY c.campaign_id -- Consistent ordering for pagination
		LIMIT $4 OFFSET $5;
	`
	// This revised query attempts to more directly implement the logic:
	// For each dimension:
	// - If there are 'include' rules for this dimension, one of them MUST match.
	// - No 'exclude' rule for this dimension can match.
	// If there are no 'include' rules for a dimension, it's considered a match for that dimension (unless excluded).

	args := []interface{}{appID, country, os, limit, offset}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("DB query failed for GetTargetedCampaigns: %v\nQuery: %s\nArgs: %v", err, query, args)
		return nil, err
	}
	defer rows.Close()

	var campaigns []models.Campaign
	for rows.Next() {
		var c models.Campaign
		// Ensure Campaign struct in models.go has these fields
		err := rows.Scan(&c.CampaignID, &c.CampaignName, &c.ImageURL, &c.CallToAction)
		if err != nil {
			log.Printf("Error scanning campaign row: %v", err)
			return nil, err
		}
		campaigns = append(campaigns, c)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating campaign rows: %v", err)
		return nil, err
	}

	// For debugging, log the number of campaigns found
	log.Printf("GetTargetedCampaigns found %d campaigns for appID: %s, country: %s, os: %s", len(campaigns), appID, country, os)
	return campaigns, nil
}

// GetCampaignByID retrieves a single campaign by its ID.
// This is a new function that might be useful.
func GetCampaignByID(db *sql.DB, campaignID string) (*models.Campaign, error) {
	timer := prometheus.NewTimer(utils.DBOperationDuration.WithLabelValues("GetCampaignByID"))
	defer timer.ObserveDuration()

	query := `
		SELECT campaign_id, campaign_name, image_url, call_to_action, campaign_status, cdate, udate
		FROM campaigns
		WHERE campaign_id = $1;`

	row := db.QueryRow(query, campaignID)
	var c models.Campaign
	err := row.Scan(
		&c.CampaignID, &c.CampaignName, &c.ImageURL, &c.CallToAction,
		&c.CampaignStatus, &c.CDate, &c.UDate,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Or a specific "not found" error
		}
		log.Printf("Error scanning campaign by ID %s: %v", campaignID, err)
		return nil, err
	}
	return &c, nil
}
