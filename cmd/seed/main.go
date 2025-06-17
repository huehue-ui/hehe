package main

import (
	"campaignservice/db"
	"campaignservice/utils"
	"database/sql"
	"flag"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

func SeedTestData(db *sql.DB, count, concurrency int) error {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	dimensions := utils.TargetingDimensions

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	errCh := make(chan error, count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()

			cid := fmt.Sprintf("camp_%d_%d", time.Now().UnixNano(), r.Intn(10000))
			name := fmt.Sprintf("Campaign %d", i)
			image := fmt.Sprintf("https://cdn.example.com/img%d.png", i)
			cta := "Install"
			status := "ACTIVE"

			_, err := db.Exec(`
				INSERT INTO campaigns (campaign_id, campaign_name, image_url, call_to_action, campaign_status)
				VALUES ($1, $2, $3, $4, $5)
			`, cid, name, image, cta, status)
			if err != nil {
				errCh <- err
				return
			}

			for _, dim := range dimensions {
				ruleType := []string{"include", "exclude"}[r.Intn(2)]
				val := fmt.Sprintf("%s_val_%d", dim, r.Intn(5))
				_, err := db.Exec(`
					INSERT INTO targeting_rules (campaign_id, dimension, type, value)
					VALUES ($1, $2, $3, $4)
				`, cid, dim, ruleType, val)
				if err != nil {
					errCh <- err
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		return err
	}
	return nil
}

func main() {
	records := flag.Int("records", 10, "Number of records to seed")
	workers := flag.Int("workers", 10, "Number of concurrent workers")
	flag.Parse()

	_ = godotenv.Load()
	conn := db.Connect()
	defer conn.Close()

	if err := SeedTestData(conn, *records, *workers); err != nil {
		fmt.Println("Error seeding data:", err)
		return
	}
	fmt.Printf("Inserted %d campaigns.\n", *records)
}
