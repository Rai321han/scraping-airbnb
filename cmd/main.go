package main

import (
	"context"
	"fmt"
	"scraping-airbnb/internal/domain"
	"scraping-airbnb/scraper/airbnb"
	"scraping-airbnb/service"
)

func main() {
	ctx := context.Background()
	chromedpScraper := airbnb.NewChromedpScraper(ctx)
	repo := domain.NewCSVRepository("products.csv")
	scraperService := service.NewScraperService(chromedpScraper, repo)
	products, err := scraperService.Run(ctx, "https://www.airbnb.com/")

	if err != nil {
		panic(err)
	}

	fmt.Println("Products:", len(products))

	for _, p := range products {
		fmt.Println(p.Title, p.Price)
	}
}