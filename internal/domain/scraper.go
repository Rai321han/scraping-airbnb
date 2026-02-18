package domain

import (
	"context"
	"scraping-airbnb/models"
)

type Scraper interface {
	Scrape(ctx context.Context, baseUrl string) ([]models.Property, error)
}