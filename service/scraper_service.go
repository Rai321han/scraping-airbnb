package service

import (
	"context"
	"scraping-airbnb/internal/domain"
)


type ScraperService struct {
	scraper domain.Scraper
}

func NewScraperService(scraper domain.Scraper) *ScraperService {
	return &ScraperService{
		scraper: scraper,
	}
}


func (s *ScraperService) Run (ctx context.Context, url string) ([]domain.Product, error) {
	products, err := s.scraper.Scrape(ctx, url)

	if err != nil {
		return nil, err
	}

	return products, nil
}