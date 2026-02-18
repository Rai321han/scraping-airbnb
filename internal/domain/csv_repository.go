package domain

import (
	"context"
	"encoding/csv"
	"os"
	"scraping-airbnb/models"
	"strconv"
)

type CSVRepository struct {
	filePath string
}

func NewCSVRepository(filePath string) *CSVRepository {
	return &CSVRepository{
		filePath: filePath,
	}
}

func (r *CSVRepository) Save(ctx context.Context, products []models.Property) error {

	file, err := os.Create(r.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// header
	writer.Write([]string{
		"Title",
		"Price",
		"Location",
		"URL",
		"Rating",
	})

	for _, p := range products {

		writer.Write([]string{
			p.Title,
			strconv.FormatFloat(float64(p.Price), 'f', 2, 32),
			p.Location,
			p.URL,
		})
	}

	return nil
}
