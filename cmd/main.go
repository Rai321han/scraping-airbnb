package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	application "scraping-airbnb/cmd/scraper"
	"scraping-airbnb/config"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)


func init() {
	// load .env file from project root
	envPath := filepath.Join(".", ".env")
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("warning: .env file not found at %s; using environment variables", envPath)
	}
}

func main() {
	ctx := context.Background()

	// load config
	cfg := config.Default()

	// initialize app
	app := application.NewApp(cfg)

	// get URL from environment or use default
	url := os.Getenv("SCRAPER_URL")
	if url == "" {
		log.Fatal("SCRAPER_URL environment variable not set")
	}

	// run the scraper
	if err := app.Run(ctx, url); err != nil {
		log.Fatalf("application failed: %v", err)
	}
}