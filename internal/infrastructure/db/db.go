package db

import (
	"campaignservice/internal/domain/models"
	"campaignservice/pkg/utils"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func LoadDBConfig() models.DBConfig {
	return models.DBConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
	}
}

func Connect() *sql.DB {
	cfg := LoadDBConfig()
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName,
	)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("failed to connect to DB:", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal("DB not reachable:", err)
	}
	return db
}

func GetTargetedCampaigns(db *sql.DB, appID, country, os string, limit int, offset int) ([]models.Campaign, error) {
	dimensionValues := map[string]string{
		"app_id":  appID,
		"country": country,
		"os":      os,
	}

	var conditions []string
	var args []interface{}
	argIndex := 1

	for _, dim := range utils.TargetingDimensions {
		val := dimensionValues[dim]
		conditions = append(conditions,
			fmt.Sprintf("(tr.dimension = '%s' AND tr.type = 'include' AND tr.value != $%d)", dim, argIndex),
			fmt.Sprintf("(tr.dimension = '%s' AND tr.type = 'exclude' AND tr.value = $%d)", dim, argIndex),
		)
		args = append(args, val)
		argIndex++
	}

	query := `
		SELECT c.campaign_id, c.campaign_name, c.image_url, c.call_to_action
		FROM campaigns c
		LEFT JOIN targeting_rules tr ON tr.campaign_id = c.campaign_id
		WHERE c.campaign_status = 'ACTIVE'
		GROUP BY c.campaign_id, c.campaign_name, c.image_url, c.call_to_action
		HAVING bool_and(
			(tr.dimension != 'app_id' OR (tr.type = 'include' AND tr.value = $1) OR (tr.type = 'exclude' AND tr.value != $1))
		) AND bool_and(
			(tr.dimension != 'country' OR (tr.type = 'include' AND tr.value = $2) OR (tr.type = 'exclude' AND tr.value != $2))
		) AND bool_and(
			(tr.dimension != 'os' OR (tr.type = 'include' AND tr.value = $3) OR (tr.type = 'exclude' AND tr.value != $3))
		)`

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("DB query failed: %v\n", err)
		return nil, err
	}
	defer rows.Close()

	var campaigns []models.Campaign
	for rows.Next() {
		var c models.Campaign
		err := rows.Scan(&c.CampaignID, &c.CampaignName, &c.ImageURL, &c.CallToAction)
		if err != nil {
			return nil, err
		}
		campaigns = append(campaigns, c)
	}
	return campaigns, nil
}
