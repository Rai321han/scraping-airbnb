package airbnb

import (
	"context"
	"encoding/json"
	"fmt"
	"scraping-airbnb/config"
	"scraping-airbnb/models"
	"scraping-airbnb/scraper"
	"scraping-airbnb/utils"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

type ChromedpScraper struct {
	allocatorCtx context.Context
	cfg          *config.Config
}

// NewChromedpScraper returns a ChromedpScraper using the default configuration.

func NewChromedpScraper(parent context.Context) *ChromedpScraper {
	cfg := config.Default()
	return &ChromedpScraper{
		allocatorCtx: scraper.NewAllocator(parent, &cfg.Browser),
		cfg:          cfg,
	}
}


func (s *ChromedpScraper) Scrape(ctx context.Context, baseURL string) ([]models.Property, error) {

	// Step 1: extract location links
	locationLinks, err := s.extractLocationLinks(baseURL)
	if err != nil {
		return nil, err
	}

	// Step 2: extract all card links concurrently
	propertyURLs := s.extractAllCardLinksConcurrent(locationLinks[:1])
	
	// Step 3: extract products concurrently via worker pool
	products := s.extractProductsWorkerPool(propertyURLs[:2], 5)

	return products, nil
}

// CARD LINKS CONCURRENT
func (s *ChromedpScraper) extractAllCardLinksConcurrent(locations []LocationLink) []string {

	var wg sync.WaitGroup
	var mu sync.Mutex

	sem := make(chan struct{}, 3)

	var allLinks []string

	for _, loc := range locations {

		wg.Add(1)

		go func(locationURL string) {

			defer wg.Done()

			sem <- struct{}{}
			links := s.extractCardLinks(locationURL)
			<-sem

			mu.Lock()
			allLinks = append(allLinks, links...)
			mu.Unlock()

		}(loc.URL)
	}

	wg.Wait()

	return allLinks
}


// WORKER POOL PRODUCT EXTRACTION
func (s *ChromedpScraper) extractProductsWorkerPool(
	cardLinks []string,
	workerCount int,
) []models.Property {

	jobs := make(chan string, len(cardLinks))
	results := make(chan models.Property, len(cardLinks))

	var wg sync.WaitGroup

	// workers
	for range workerCount {
		wg.Go(func() {
			for url := range jobs {

				product, err := s.extractProperty(url)

				if err == nil {
					results <- product
				}
			}
		})
	}

	// send jobs
	for _, link := range cardLinks {
		jobs <- link
	}

	close(jobs)

	wg.Wait()

	close(results)

	var products []models.Property

	for p := range results {
		products = append(products, p)
	}

	return products
}


type LocationLink struct {
	URL  string `json:"url"`
}

func (s *ChromedpScraper) extractLocationLinks(url string) ([]LocationLink, error) {
	tab, cancel := scraper.NewTab(s.allocatorCtx)
	defer cancel()

	var rawJSON string

	err := chromedp.Run(tab,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`h2`, chromedp.ByQuery),
		scraper.ScrollToBottom(&s.cfg.Timing, s.cfg.Scraper.ScrollStep),
		chromedp.Sleep(s.cfg.Timing.AfterScrollWait),
		chromedp.Evaluate(locationLinksJS, &rawJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("extractLocationLinks %s: %w", url, err)
	}

	var links []LocationLink
	if err := json.Unmarshal([]byte(rawJSON), &links); err != nil {
		return nil, fmt.Errorf("extractLocationLinks parse JSON: %w", err)
	}

	return links, nil
}

// extractCardLinks opens a location search page and collects listing hrefs.
// It scrolls to load all cards, then checks for a second page via pagination.
// A single tab is reused for both pages to avoid allocator pressure.
func (s *ChromedpScraper) extractCardLinks(locationURL string) []string {
	tab, cancel := scraper.NewTab(s.allocatorCtx)
	defer cancel()

	// Page 1
	page1 := s.scrapeCardPage(tab, locationURL)

	// Check for next page while still on page 1
	nextURL := s.findNextPageURL(tab)
	if nextURL == "" {
		return page1
	}

	// Page 2 (reuse same tab)
	page2 := s.scrapeCardPage(tab, nextURL)
	return append(page1, page2...)
}

// scrapeCardPage navigates to url in the given tab, scrolls, and returns card hrefs.
func (s *ChromedpScraper) scrapeCardPage(ctx context.Context, url string) []string {
	var links []string

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(s.cfg.Timing.PageLoadWait),
		scraper.ScrollToBottom(&s.cfg.Timing, s.cfg.Scraper.ScrollStep),
		chromedp.Sleep(s.cfg.Timing.AfterScrollWait),
		chromedp.Evaluate(cardLinksJS(s.cfg.Scraper.CardsPage1), &links),
	)
	if err != nil {
		fmt.Printf("[cards] scrapeCardPage error %s: %v\n", url, err)
	}

	return links
}

// findNextPageURL reads the current tab DOM and returns the "Next" page href,
// or an empty string if no pagination link is present.
func (s *ChromedpScraper) findNextPageURL(ctx context.Context) string {
	var nextURL string

	_ = chromedp.Run(ctx,
		chromedp.Evaluate(nextPageJS, &nextURL),
	)

	return nextURL
}

func (s *ChromedpScraper) extractProperty(url string) (models.Property, error) {
	tabCtx, cancel := context.WithTimeout(s.allocatorCtx, 30*time.Second)
	defer cancel()

	browserCtx, browserCancel := chromedp.NewContext(tabCtx)
	defer browserCancel()

	var title, priceText, location, ratingText string

	err := chromedp.Run(browserCtx,
		chromedp.Navigate(url),
		chromedp.Sleep(4*time.Second),
		chromedp.Evaluate(titleJS, &title),
		chromedp.Evaluate(priceJS, &priceText),
		chromedp.Evaluate(locationJS, &location),
		chromedp.Evaluate(ratingJS, &ratingText),
	)

	if err != nil {
		return models.Property{}, err
	}

	return models.Property{
		Title:    title,
		Price:    utils.ParsePrice(priceText),
		Location: location,
		URL:      url,
		Rating:   utils.ParseRating(ratingText),
	}, nil
}
