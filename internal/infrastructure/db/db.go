package db

import (
	"campaignservice/internal/domain/models"
	"campaignservice/pkg/utils" // Used for TargetingDimensions, though the query logic might need review
	"database/sql"
	"fmt"
	"log"
	"os"

	// Import PostgreSQL driver, the blank identifier _ means it's imported for its side effects (registering the driver)
	_ "github.com/lib/pq"
)

// LoadDBConfig loads database configuration from environment variables.
func LoadDBConfig() models.DBConfig {
	return models.DBConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
	}
}

// Connect establishes a connection to the PostgreSQL database.
// It uses configuration loaded by LoadDBConfig.
// The application will terminate (log.Fatal) if the connection fails.
func Connect() *sql.DB {
	cfg := LoadDBConfig()
	// Construct the connection string for PostgreSQL. SSL mode is disabled.
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName,
	)
	// Open a new database connection.
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("failed to connect to DB: %v", err) // Use Fatalf for formatted message
	}
	// Ping the database to verify the connection is alive.
	if err = db.Ping(); err != nil {
		log.Fatalf("DB not reachable: %v", err) // Use Fatalf for formatted message
	}
	log.Println("Successfully connected to the database.")
	return db
}

// GetTargetedCampaigns retrieves campaigns from the database that match the specified targeting criteria.
// It filters by appID, country, and os, and supports pagination with limit and offset.
func GetTargetedCampaigns(db *sql.DB, appID, country, os string, limit int, offset int) ([]models.Campaign, error) {
	// This map and loop seem to be an incomplete or alternative way to build conditions.
	// The actual query uses hardcoded parameter placeholders $1, $2, $3.
	// Keeping it here for context, but it doesn't directly affect the executed query.
	/*
		dimensionValues := map[string]string{
			"app_id":  appID,
			"country": country,
			"os":      os,
		}

		var conditions []string
		var queryArgsForDynamicBuild []interface{} // Renamed to avoid confusion with 'args' used later
		argIndex := 1

		for _, dim := range utils.TargetingDimensions {
			val, ok := dimensionValues[dim] // Check if the dimension exists in the input
			if ok {
				// Example of how dynamic conditions might be built (currently not used by the main query)
				// This logic would need careful construction to be correct and secure (SQL injection).
				conditions = append(conditions,
					fmt.Sprintf("(tr.dimension = '%s' AND tr.type = 'include' AND tr.value != $%d)", dim, argIndex), // This logic for include seems inverted
					fmt.Sprintf("(tr.dimension = '%s' AND tr.type = 'exclude' AND tr.value = $%d)", dim, argIndex),   // This logic for exclude seems inverted
				)
				queryArgsForDynamicBuild = append(queryArgsForDynamicBuild, val)
				argIndex++
			}
		}
	*/

	// Arguments for the hardcoded query below.
	// The order must match the placeholders $1, $2, $3, $4, $5 in the query.
	args := []interface{}{appID, country, os, limit, offset}

	// SQL query to select campaigns based on targeting rules.
	// Explanation:
	// - Selects basic campaign details.
	// - Joins with targeting_rules to filter based on dimensions.
	// - Filters for ACTIVE campaigns.
	// - Groups by campaign to apply aggregate conditions in HAVING.
	// - The HAVING clause uses bool_and to ensure all conditions are met for a campaign to be selected.
	//   - For each dimension (app_id, country, os):
	//     - (tr.dimension != 'dim_name' OR ...): This part means if a campaign has NO specific rule for 'dim_name', it's considered a match for this part of the condition for that dimension.
	//     - (tr.type = 'include' AND tr.value = $N): If it's an 'include' rule for the dimension, the value must match.
	//     - (tr.type = 'exclude' AND tr.value != $N): If it's an 'exclude' rule for the dimension, the value must NOT match.
	// - LIMIT and OFFSET are used for pagination.
	query := `
		SELECT c.campaign_id, c.campaign_name, c.image_url, c.call_to_action
		FROM campaigns c
		LEFT JOIN targeting_rules tr ON tr.campaign_id = c.campaign_id AND tr.dimension IN ('app_id', 'country', 'os')
		WHERE c.campaign_status = 'ACTIVE'
		GROUP BY c.campaign_id, c.campaign_name, c.image_url, c.call_to_action
		HAVING bool_and(
			(tr.dimension != 'app_id' OR (tr.type = 'include' AND tr.value = $1) OR (tr.type = 'exclude' AND tr.value != $1))
		) AND bool_and(
			(tr.dimension != 'country' OR (tr.type = 'include' AND tr.value = $2) OR (tr.type = 'exclude' AND tr.value != $2))
		) AND bool_and(
			(tr.dimension != 'os' OR (tr.type = 'include' AND tr.value = $3) OR (tr.type = 'exclude' AND tr.value != $3))
		)
		ORDER BY c.campaign_id -- Added for consistent ordering, good for pagination
		LIMIT $4 OFFSET $5`

	// Execute the query
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("DB query failed for GetTargetedCampaigns: %v\nQuery: %s\nArgs: %v", err, query, args)
		return nil, fmt.Errorf("querying targeted campaigns: %w", err)
	}
	defer rows.Close()

	var campaigns []models.Campaign
	// Iterate over the returned rows
	for rows.Next() {
		var c models.Campaign
		// Scan the row data into the Campaign struct fields
		// Note: Only CampaignID, CampaignName, ImageURL, CallToAction are selected by the query.
		// Other fields of models.Campaign (like CampaignStatus, CDate, UDate) will remain empty/zero-valued.
		err := rows.Scan(&c.CampaignID, &c.CampaignName, &c.ImageURL, &c.CallToAction)
		if err != nil {
			log.Printf("Error scanning campaign row: %v", err)
			return nil, fmt.Errorf("scanning campaign row: %w", err)
		}
		campaigns = append(campaigns, c)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating campaign rows: %v", err)
		return nil, fmt.Errorf("iterating campaign rows: %w", err)
	}

	return campaigns, nil
}
