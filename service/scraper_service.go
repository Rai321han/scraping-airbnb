package service

import (
	"context"
	"scraping-airbnb/internal/domain"
	"scraping-airbnb/models"
)


type ScraperService struct {
	scraper domain.Scraper
	repo    domain.ProductRepository
}

func NewScraperService(
	s domain.Scraper,
	r domain.ProductRepository,
) *ScraperService {

	return &ScraperService{
		scraper: s,
		repo:    r,
	}
}

func (s *ScraperService) Run (ctx context.Context, url string) ([]models.Property, error) {
	products, err := s.scraper.Scrape(ctx, url)

	if err != nil {
		return nil, err
	}

	err = s.repo.Save(ctx, products)
	if err != nil {
		return nil, err
	}

	return products, nil
}