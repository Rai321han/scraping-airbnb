package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"scraping-airbnb/config"
	"scraping-airbnb/internal/domain"
	"scraping-airbnb/scraper/airbnb"
	"scraping-airbnb/service"

	_ "github.com/lib/pq"
)

	func main() {
		ctx := context.Background()

		// load config
		cfg := config.Default()
		log.Printf("scraper config: max_retries=%d, initial_backoff=%v, max_backoff=%v",
			cfg.Retry.MaxRetries, cfg.Retry.InitialBackoff, cfg.Retry.MaxBackoff)

		chromedpScraper := airbnb.NewChromedpScraper(ctx)

		// connect to postgres (defaults match docker-compose)
		dsn := os.Getenv("PG_DSN")
		if dsn == "" {
			dsn = "postgres://airbnb:airbnb_password@localhost:5432/airbnb_db?sslmode=disable"
		}

		db, err := sql.Open("postgres", dsn)
		if err != nil {
			log.Fatalf("failed to create db connection: %v", err)
		}
		defer db.Close()

		if err := db.PingContext(ctx); err != nil {
			log.Fatalf("failed to ping db: %v", err)
		}

		log.Println("db connection successful")

		repo := domain.NewPostgresRepository(db)
		scraperService := service.NewScraperService(chromedpScraper, repo, cfg)
		products, err := scraperService.Run(ctx, "https://www.airbnb.com/")

		if err != nil {
			log.Fatalf("scraping failed: %v", err)
		}

		fmt.Printf("✓ Scraping completed successfully: %d properties saved\n", len(products))
	}