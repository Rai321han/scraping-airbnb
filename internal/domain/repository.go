package domain

import (
	"context"
	"scraping-airbnb/models"
)

type PropertyRepository interface {
	Save(ctx context.Context, property []models.Property) error
}
