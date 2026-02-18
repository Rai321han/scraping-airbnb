package domain

import (
	"context"
)

type Scraper interface {
	Scrape(ctx context.Context, baseUrl string) ([]Product, error)
}