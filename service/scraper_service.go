package service

import (
	"context"
	"fmt"
	"log"
	"math"
	"scraping-airbnb/config"
	"scraping-airbnb/internal/domain"
	"scraping-airbnb/models"
	"sort"
	"strings"
	"time"
)


type ScraperService struct {
	scraper domain.Scraper
	repo    domain.PropertyRepository
	cfg     *config.Config
}

func NewScraperService(
	s domain.Scraper,
	r domain.PropertyRepository,
	cfg *config.Config,
) *ScraperService {

	return &ScraperService{
		scraper: s,
		repo:    r,
		cfg:     cfg,
	}
}

func (s *ScraperService) Run (ctx context.Context, url string) ([]models.Property, error) {
	var property []models.Property

	// Scrape with retries
	err := s.retryWithBackoff(ctx, func() error {
		var scrapeErr error
		property, scrapeErr = s.scraper.Scrape(ctx, url)
		return scrapeErr
	})

	if err != nil {
		log.Printf("scrape failed after %d retries: %v", s.cfg.Retry.MaxRetries, err)
		return nil, err
	}

	// Save with retries
	err = s.retryWithBackoff(ctx, func() error {
		return s.repo.Save(ctx, property)
	})

	if err != nil {
		log.Printf("save failed after %d retries: %v", s.cfg.Retry.MaxRetries, err)
		return nil, err
	}

	// After successful save, print scraping insights
	printInsights(property)

	return property, nil
}

// retryWithBackoff executes fn with exponential backoff retries.
func (s *ScraperService) retryWithBackoff(ctx context.Context, fn func() error) error {
	maxRetries := s.cfg.Retry.MaxRetries
	initialBackoff := s.cfg.Retry.InitialBackoff
	maxBackoff := s.cfg.Retry.MaxBackoff

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("[retry] attempt #%d of %d...", attempt+1, maxRetries+1)
		}

		if err := fn(); err == nil {
			if attempt > 0 {
				log.Printf("[retry] ✅ attempt #%d succeeded", attempt+1)
			}
			return nil
		} else {
			lastErr = err
		}

		if attempt < maxRetries {
			// exponential backoff: backoff = initialBackoff * 2^attempt, capped at maxBackoff
			backoff := time.Duration(float64(initialBackoff) * math.Pow(2, float64(attempt)))
			if backoff > maxBackoff {
				backoff = maxBackoff
			}

			log.Printf("[retry] attempt #%d failed: %v; waiting %v before retry", attempt+1, lastErr, backoff)
			select {
			case <-time.After(backoff):
				// continue to next retry
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	log.Printf("[retry] ❌ all %d attempts failed", maxRetries+1)
	return fmt.Errorf("failed after %d attempts: %w", maxRetries+1, lastErr)
}

func parseCity(location string) string {
	parts := strings.Split(location, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	if len(parts) >= 2 {
		// second-last part
		return parts[len(parts)-2]
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return ""
}

func printInsights(property []models.Property) {
	total := len(property)
	if total == 0 {
		fmt.Println("No listings scraped.")
		return
	}

	var sumPrice float64
	minPrice := float64(property[0].Price)
	maxPrice := float64(property[0].Price)
	var mostExpensive models.Property
	mostExpensive = property[0]

	listingsPerLocation := make(map[string]int)
	platformCounts := make(map[string]int)

	for _, p := range property {
		price := float64(p.Price)
		sumPrice += price
		if price < minPrice {
			minPrice = price
		}
		if price > maxPrice {
			maxPrice = price
			mostExpensive = p
		}

		city := parseCity(p.Location)
		if city == "" {
			city = p.Location
		}
		listingsPerLocation[city]++

		platformCounts[p.Platform]++
	}

	avgPrice := sumPrice / float64(total)

	// sort locations by count desc
	type locCount struct{
		Loc string
		C int
	}
	var locs []locCount
	for k,v := range listingsPerLocation {
		locs = append(locs, locCount{Loc: k, C: v})
	}
	sort.Slice(locs, func(i,j int) bool { return locs[i].C > locs[j].C })

	// top 5 highest rated
	propertyByRating := make([]models.Property, len(property))
	copy(propertyByRating, property)
	sort.Slice(propertyByRating, func(i,j int) bool { return propertyByRating[i].Rating > propertyByRating[j].Rating })

	// print with clean formatting
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("                    SCRAPING INSIGHTS REPORT")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("\nSUMMARY STATISTICS")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("  Total Listings Scraped:  %d\n", total)
	fmt.Printf("  Airbnb Listings:         %d\n", platformCounts["Airbnb"])
	fmt.Printf("  Average Price:           $%.2f\n", avgPrice)
	fmt.Printf("  Minimum Price:           $%.0f\n", minPrice)
	fmt.Printf("  Maximum Price:           $%.0f\n", maxPrice)

	fmt.Println("\nMOST EXPENSIVE PROPERTY")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("  Title:                   %s\n", mostExpensive.Title)
	fmt.Printf("  Price:                   $%.0f\n", mostExpensive.Price)
	fmt.Printf("  Location:                %s\n", mostExpensive.Location)

	fmt.Println("\nLISTINGS PER LOCATION")
	fmt.Println(strings.Repeat("-", 60))
	for _, lc := range locs {
		fmt.Printf("  %-40s %d\n", lc.Loc+":", lc.C)
	}

	fmt.Println("\nTOP 5 HIGHEST RATED PROPERTIES")
	fmt.Println(strings.Repeat("-", 60))
	limit := 5
	if len(propertyByRating) < limit {
		limit = len(propertyByRating)
	}
	for i := 0; i < limit; i++ {
		p := propertyByRating[i]
		fmt.Printf("  %d. %s\n", i+1, p.Title)
		fmt.Printf("     Rating: %.2f ⭐\n", p.Rating)
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()
}