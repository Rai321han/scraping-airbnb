package domain

import (
	"context"
	"scraping-airbnb/models"
)

type ProductRepository interface {
	Save(ctx context.Context, products []models.Property) error
}
