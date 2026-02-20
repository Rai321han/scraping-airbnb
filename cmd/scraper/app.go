package application

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
)

func NewApp(cfg *config.Config) *App {
	return &App{cfg: cfg}
}

type App struct {
	cfg *config.Config
}

func (a *App) Run(ctx context.Context, url string) error {
	log.Printf("scraper config: max_retries=%d, initial_backoff=%v, max_backoff=%v",
		a.cfg.Retry.MaxRetries, a.cfg.Retry.InitialBackoff, a.cfg.Retry.MaxBackoff)

	chromedpScraper := airbnb.NewChromedpScraper(ctx)

	// connect to postgres (defaults match docker-compose)
	dsn := os.Getenv("PG_DSN")
	if dsn == "" {
		return fmt.Errorf("db connection string not found")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to create db connection: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping db: %w", err)
	}

	log.Println("db connection successful")

	repo := domain.NewPostgresRepository(db)
	scraperService := service.NewScraperService(chromedpScraper, repo, a.cfg)
	products, err := scraperService.Run(ctx, url)

	if err != nil {
		return fmt.Errorf("scraping failed: %w", err)
	}

	fmt.Printf("✓ Scraping completed successfully: %d properties saved\n", len(products))
	return nil
}