package airbnb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"scraping-airbnb/config"
	"scraping-airbnb/models"
	"scraping-airbnb/scraper"
	"scraping-airbnb/utils"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/chromedp"
)

type ChromedpScraper struct {
	allocatorCtx context.Context
	cfg          *config.Config
	rateLimiter  *time.Ticker
	requestMutex sync.Mutex
	userAgents   []string
}

// NewChromedpScraper returns a ChromedpScraper using the default configuration.

func NewChromedpScraper(parent context.Context) *ChromedpScraper {
	cfg := config.Default()
	log.SetFlags(log.LstdFlags)
	log.Printf("chromedp scraper created")

	// initialize rate limiter
	var ticker *time.Ticker
	if cfg.Stealth.MaxRequestsPerSecond > 0 {
		interval := time.Duration(float64(time.Second) / cfg.Stealth.MaxRequestsPerSecond)
		ticker = time.NewTicker(interval)
	}

	s := &ChromedpScraper{
		allocatorCtx: scraper.NewAllocator(parent, &cfg.Browser),
		cfg:          cfg,
		rateLimiter:  ticker,
		userAgents:   config.DefaultUserAgents(),
	}

	// log stealth settings
	if cfg.Stealth.RandomDelayEnabled {
		log.Printf("stealth: random delays enabled (%v-%v)", cfg.Stealth.RandomDelayMin, cfg.Stealth.RandomDelayMax)
	}
	if cfg.Stealth.RandomUserAgentEnabled {
		log.Printf("stealth: random user agent enabled")
	}
	if cfg.Stealth.MaxRequestsPerSecond > 0 {
		log.Printf("stealth: rate limit enabled (%.1f req/sec)", cfg.Stealth.MaxRequestsPerSecond)
	}

	return s
}

// runWithRetry executes chromedp.Run with exponential backoff retries.
func (s *ChromedpScraper) runWithRetry(ctx context.Context, actions ...chromedp.Action) error {
	return s.retryWithBackoff(ctx, func() error {
		return chromedp.Run(ctx, actions...)
	})
}

// retryWithBackoff executes fn with exponential backoff.
func (s *ChromedpScraper) retryWithBackoff(ctx context.Context, fn func() error) error {
	maxRetries := s.cfg.Retry.MaxRetries
	initialBackoff := s.cfg.Retry.InitialBackoff
	maxBackoff := s.cfg.Retry.MaxBackoff

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("[chromedp-retry] attempt #%d of %d...", attempt+1, maxRetries+1)
		}

		if err := fn(); err == nil {
			if attempt > 0 {
				log.Printf("[chromedp-retry] ✅ attempt #%d succeeded", attempt+1)
			}
			return nil
		} else {
			lastErr = err
		}

		if attempt < maxRetries {
			backoff := time.Duration(float64(initialBackoff) * math.Pow(2, float64(attempt)))
			if backoff > maxBackoff {
				backoff = maxBackoff
			}

			log.Printf("[chromedp-retry] attempt #%d failed: %v; waiting %v before retry", attempt+1, lastErr, backoff)
			select {
			case <-time.After(backoff):
				// continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	log.Printf("[chromedp-retry] ❌ all %d attempts failed", maxRetries+1)
	return fmt.Errorf("chromedp failed after %d attempts: %w", maxRetries+1, lastErr)
}

// applyRateLimit waits if necessary to respect the configured max requests per second.
func (s *ChromedpScraper) applyRateLimit() {
	if s.rateLimiter == nil {
		return
	}
	s.requestMutex.Lock()
	defer s.requestMutex.Unlock()
	<-s.rateLimiter.C
}

// randomDelay applies a random sleep if stealth mode is enabled.
func (s *ChromedpScraper) randomDelay() {
	if !s.cfg.Stealth.RandomDelayEnabled {
		return
	}
	minMs := s.cfg.Stealth.RandomDelayMin.Milliseconds()
	maxMs := s.cfg.Stealth.RandomDelayMax.Milliseconds()
	if minMs >= maxMs {
		return
	}
	randMs := rand.Int63n(maxMs - minMs) + minMs
	time.Sleep(time.Duration(randMs) * time.Millisecond)
}

// getRandomUserAgent returns a random user agent from the pool if enabled.
func (s *ChromedpScraper) getRandomUserAgent() string {
	if !s.cfg.Stealth.RandomUserAgentEnabled || len(s.userAgents) == 0 {
		return s.cfg.Browser.UserAgent
	}
	return s.userAgents[rand.Intn(len(s.userAgents))]
}

func (s *ChromedpScraper) Scrape(ctx context.Context, baseURL string) ([]models.Property, error) {

	start := time.Now()
	log.Printf("scrape: start %s", baseURL)

	// Step 1: extract location links
	locationLinks, err := s.extractLocationLinks(baseURL)
	if err != nil {
		return nil, err
	}
	log.Printf("scrape: found %d locations", len(locationLinks))

	// Step 2: extract all card links concurrently
	propertyURLs := s.extractAllCardLinksConcurrent(locationLinks)
	log.Printf("scrape: collected %d property URLs", len(propertyURLs))

	// Step 3: extract products concurrently via worker pool
	products := s.extractProductsWorkerPool(propertyURLs, s.cfg.Concurrency.ProductWorkers)

	duration := time.Since(start)
	failed := len(propertyURLs) - len(products)
	if failed < 0 {
		failed = 0
	}

	log.Printf("scrape: finished — locations=%d urls=%d fetched=%d failed=%d duration=%s",
		len(locationLinks), len(propertyURLs), len(products), failed, duration)

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

	log.Printf("workerpool: starting %d workers for %d jobs", workerCount, len(cardLinks))

	var fetchedCount int32
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for url := range jobs {
				product, err := s.extractProperty(url)
				if err != nil {
					log.Printf("[property] worker %d: failed %s: %v", id, url, err)
					continue
				}
				n := atomic.AddInt32(&fetchedCount, 1)
				log.Printf("[property] #%d fetched: %s", n, product.Title)
				results <- product
			}
		}(i)
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

	err := s.runWithRetry(tab,
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
	s.applyRateLimit()
	s.randomDelay()

	var links []string

	err := s.runWithRetry(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(s.cfg.Timing.PageLoadWait),
		scraper.ScrollToBottom(&s.cfg.Timing, s.cfg.Scraper.ScrollStep),
		chromedp.Sleep(s.cfg.Timing.AfterScrollWait),
		chromedp.Evaluate(cardLinksJS(s.cfg.Scraper.CardsPage1), &links),
	)
	if err != nil {
		log.Printf("[cards] scrapeCardPage error %s: %v", url, err)
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
	s.applyRateLimit()
	s.randomDelay()

	// Create the browser context FIRST, then wrap it with timeout
    // so the timeout applies to the tab's operations, not the allocator lifetime
    browserCtx, browserCancel := chromedp.NewContext(s.allocatorCtx)
    defer browserCancel()

    tabCtx, cancel := context.WithTimeout(browserCtx, s.cfg.Timing.ProductTimeout)
    defer cancel()

    var title, priceText, location, ratingText, description string

    err := s.runWithRetry(tabCtx,
        chromedp.Navigate(url),
        chromedp.WaitVisible(`div[data-plugin-in-point-id="TITLE_DEFAULT"]`, chromedp.ByQuery),
        chromedp.Evaluate(titleJS, &title),
        chromedp.Evaluate(priceJS, &priceText),
        chromedp.Evaluate(ratingJS, &ratingText),
        chromedp.WaitVisible(`div[data-section-id="LOCATION_DEFAULT"]`, chromedp.ByQuery),
        chromedp.Evaluate(locationJS, &location),
        chromedp.Evaluate(`
            (() => {
                const btn = document.querySelector('button[aria-label="Show more about this place"]');
                if (btn) btn.click();
            })()
        `, nil),
        chromedp.Evaluate(descriptionJS, &description),
    )
	if err != nil {
		return models.Property{}, err
	}

	property := models.Property{
		Platform: "Airbnb",
		Title:    title,
		Price:    utils.ParsePrice(priceText),
		Location: location,
		URL:      url,
		Rating:   utils.ParseRating(ratingText),
		Description:  description,
	}

	log.Printf("[property] fetched: %s", property.URL)
	return property, nil
}
