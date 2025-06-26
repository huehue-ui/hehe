package main

import (
	"campaignservice/internal/infrastructure/db"
	"campaignservice/pkg/utils"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/joho/godotenv" // For loading .env files
)

// SeedTestData generates and inserts a specified number of campaigns and their targeting rules into the database.
// It uses goroutines to perform insertions concurrently, limited by the 'concurrency' parameter.
func SeedTestData(dbConn *sql.DB, count, concurrency int) error {
	// Initialize a new random number generator.
	// Seeding with the current time ensures different data on each run.
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// Get the list of dimensions to create targeting rules for (e.g., "app_id", "country", "os").
	dimensions := utils.TargetingDimensions

	var wg sync.WaitGroup                     // Used to wait for all goroutines to complete.
	sem := make(chan struct{}, concurrency)   // Semaphore to limit the number of concurrent goroutines.
	errCh := make(chan error, count*len(dimensions)) // Buffered channel to collect errors from goroutines. Max possible errors.

	log.Printf("Starting to seed %d campaigns with %d workers...\n", count, concurrency)

	// Loop to create 'count' number of campaigns.
	for i := 0; i < count; i++ {
		wg.Add(1)      // Increment the WaitGroup counter.
		sem <- struct{}{} // Acquire a spot in the semaphore; blocks if 'concurrency' limit is reached.

		// Launch a goroutine to insert one campaign and its rules.
		go func(campaignIndex int) {
			defer wg.Done()             // Decrement the WaitGroup counter when the goroutine finishes.
			defer func() { <-sem }() // Release the spot in the semaphore.

			// Generate unique campaign ID and other campaign details.
			// Using campaignIndex for some determinism in naming if needed, along with random elements.
			cid := fmt.Sprintf("camp_%d_%d", time.Now().UnixNano(), r.Intn(100000))
			name := fmt.Sprintf("Awesome Campaign %d", campaignIndex)
			image := fmt.Sprintf("https://via.placeholder.com/300x250/0000FF/FFFFFF?text=Campaign+%d", campaignIndex)
			cta := []string{"Learn More", "Sign Up", "Shop Now", "Get Started"}[r.Intn(4)] // Random CTA
			status := "ACTIVE"                                                           // All seeded campaigns are active.

			// Insert the campaign into the 'campaigns' table.
			_, err := dbConn.Exec(`
				INSERT INTO campaigns (campaign_id, campaign_name, image_url, call_to_action, campaign_status)
				VALUES ($1, $2, $3, $4, $5)
			`, cid, name, image, cta, status)
			if err != nil {
				errCh <- fmt.Errorf("error inserting campaign %s: %w", cid, err)
				return // Exit goroutine if campaign insertion fails.
			}

			// For each dimension, create a random targeting rule.
			for _, dim := range dimensions {
				ruleType := []string{"include", "exclude"}[r.Intn(2)] // Randomly pick 'include' or 'exclude'.
				// Generate a somewhat realistic value for the dimension.
				var val string
				switch dim {
				case "country":
					val = []string{"US", "CA", "GB", "DE", "FR"}[r.Intn(5)]
				case "os":
					val = []string{"android", "ios"}[r.Intn(2)]
				case "app_id":
					val = fmt.Sprintf("app%d", r.Intn(3)+1) // e.g., app1, app2, app3
				default:
					val = fmt.Sprintf("%s_val_%d", dim, r.Intn(5))
				}

				// Insert the targeting rule into the 'targeting_rules' table.
				_, err := dbConn.Exec(`
					INSERT INTO targeting_rules (campaign_id, dimension, type, value)
					VALUES ($1, $2, $3, $4)
				`, cid, dim, ruleType, val)
				if err != nil {
					errCh <- fmt.Errorf("error inserting rule for campaign %s, dimension %s: %w", cid, dim, err)
					// Continue to try inserting other rules for this campaign, or return if critical.
				}
			}
			if campaignIndex%100 == 0 && campaignIndex > 0 { // Log progress
				log.Printf("Seeded campaign %d and its rules...\n", campaignIndex)
			}
		}(i) // Pass 'i' to the goroutine to ensure each gets a unique index.
	}

	wg.Wait()   // Wait for all goroutines to finish.
	close(errCh) // Close the error channel once all producers are done.

	// Check if any errors were reported by the goroutines.
	// This will return the first error encountered. For multiple errors, more complex handling would be needed.
	for err := range errCh {
		if err != nil { // Should always be non-nil here as we only send errors
			return err // Return the first error.
		}
	}
	return nil // No errors reported.
}

func main() {
	// Define command-line flags for number of records and concurrency.
	records := flag.Int("records", 100, "Number of campaign records to seed.") // Increased default
	workers := flag.Int("workers", 20, "Number of concurrent workers for seeding.") // Increased default
	flag.Parse()

	// Load environment variables from a .env file (if present).
	// Useful for local development to set DB_HOST, DB_USER, etc.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading it, relying on existing environment variables.")
	}

	// Establish a connection to the database.
	// db.Connect() reads configuration from environment variables.
	log.Println("Connecting to database...")
	conn := db.Connect()
	defer conn.Close() // Ensure the database connection is closed when main exits.
	log.Println("Database connection successful.")

	log.Printf("Attempting to seed %d records with %d workers.\n", *records, *workers)
	startTime := time.Now()

	// Call SeedTestData to perform the seeding.
	if err := SeedTestData(conn, *records, *workers); err != nil {
		log.Fatalf("Error seeding data: %v\n", err) // Use log.Fatalf to exit on error.
		return
	}

	duration := time.Since(startTime)
	log.Printf("Successfully inserted %d campaigns in %s.\n", *records, duration)
}
