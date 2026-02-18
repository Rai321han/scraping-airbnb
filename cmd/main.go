package main

import (
	"context"
	"fmt"
	"scraping-airbnb/scraper"
	"scraping-airbnb/service"
)

func main() {
	ctx := context.Background()
	chromedpScraper := scraper.NewChromedpScraper(ctx)
	scraperService := service.NewScraperService(chromedpScraper)
	products, err := scraperService.Run(ctx, "https://www.airbnb.com/")

	if err != nil {
		panic(err)
	}

	fmt.Println("Products:", len(products))

	for _, p := range products {
		fmt.Println(p.Title, p.Price)
	}
}