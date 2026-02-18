package domain

import (
	"context"
	"database/sql"
	"scraping-airbnb/models"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Save(ctx context.Context, products []models.Product) error {

	query := `
	INSERT INTO products (title, price, location, url, rating, details)
	VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, p := range products {

		_, err := r.db.ExecContext(
			ctx,
			query,
			p.Title,
			p.Price,
			p.Location,
			p.URL,
			p.Rating,
			p.Details,
		)

		if err != nil {
			return err
		}
	}

	return nil
}
