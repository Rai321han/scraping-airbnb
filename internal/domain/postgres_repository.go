package domain

import (
	"context"
	"database/sql"
	"fmt"
	"scraping-airbnb/models"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Save inserts properties in a single transaction using a prepared statement.
func (r *PostgresRepository) Save(ctx context.Context, properties []models.Property) error {
	if len(properties) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO properties (platform, title, price, location, url, rating, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (url) DO UPDATE SET
			title = EXCLUDED.title,
			price = EXCLUDED.price,
			location = EXCLUDED.location,
			rating = EXCLUDED.rating,
			description = EXCLUDED.description
	`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("prepare stmt: %w", err)
	}
	defer stmt.Close()

	for _, p := range properties {
		if _, err := stmt.ExecContext(ctx,
			p.Platform,
			p.Title,
			p.Price,
			p.Location,
			p.URL,
			p.Rating,
			p.Description,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("exec insert: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
