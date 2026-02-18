package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"scraping-airbnb/internal/domain"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

type ChromedpScraper struct {
	allocatorCtx context.Context
}

func NewChromedpScraper(parent context.Context) *ChromedpScraper {

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	allocCtx, _ := chromedp.NewExecAllocator(parent, opts...)

	return &ChromedpScraper{
		allocatorCtx: allocCtx,
	}
}

func (s *ChromedpScraper) Scrape(ctx context.Context, baseURL string) ([]domain.Product, error) {

	// Step 1: extract location links
	
	locationLinks, err := s.extractLocationLinks(baseURL)
	if err != nil {
		return nil, err
	}

	// Step 2: extract all card links concurrently
	propertyURLs := s.extractAllCardLinksConcurrent(locationLinks[:1])
	// // Step 3: extract products concurrently via worker pool
	products := s.extractProductsWorkerPool(propertyURLs[:2], 5)

	return products, nil
}

//
// LOCATION LINKS
//

type LocationLink struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

func (s *ChromedpScraper) extractLocationLinks(url string) ([]LocationLink, error) {
	
	tabCtx, cancel := chromedp.NewContext(s.allocatorCtx)
	defer cancel()

	var rawJSON string

	err := chromedp.Run(tabCtx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`h2`, chromedp.ByQuery),


// 	Scroll incrementally to trigger all lazy-loaded location cards
		chromedp.Evaluate(`
			(async function() {
				const delay = ms => new Promise(r => setTimeout(r, ms));
				const step = 400;
				const total = document.body.scrollHeight;
				for (let y = 0; y <= total; y += step) {
					window.scrollTo(0, y);
					await delay(1000);
				}
				// Final pause at bottom so last items can render
				await delay(3000);
				window.scrollTo(0, 0);
			})()
		`, nil),

		chromedp.Sleep(4*time.Second),

		chromedp.Evaluate(`
			JSON.stringify(
				Array.from(document.querySelectorAll('h2.skp76t2 > a'))
				.map(a => ({text:a.innerText,url:a.href}))
			)
		`, &rawJSON),
	)

	if err != nil {
		return nil, err
	}

	var links []LocationLink
	json.Unmarshal([]byte(rawJSON), &links)
	fmt.Printf("Got %d links from home page\n", len(links))
	return links, nil
}

//
// CARD LINKS CONCURRENT
//

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




func (s *ChromedpScraper) extractCardLinks(url string) []string {

	tabCtx, cancel := chromedp.NewContext(s.allocatorCtx)
	defer cancel()

	var page1Links []string
	var page2Links []string
	var nextPage string

	err := chromedp.Run(tabCtx,

		chromedp.Navigate(url),

		chromedp.Sleep(3*time.Second),

		chromedp.Evaluate(`
			(async function() {
				const delay = ms => new Promise(r => setTimeout(r, ms));
				const step = 400;
				const total = document.body.scrollHeight;
				for (let y = 0; y <= total; y += step) {
					window.scrollTo(0, y);
					await delay(2000);
				}
				// Final pause at bottom so last items can render
				await delay(3000);
				window.scrollTo(0, 0);
			})()
		`, nil),

		chromedp.Sleep(10*time.Second),

		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('.cy5jw6o > a'))
				.slice(0,5)
				.map(a=>a.href)
		`, &page1Links),

		chromedp.Evaluate(`
		(()=>{
			// Try multiple stable selectors
			const selectors = [
				'.p1uqa2vx > a[aria-label="Next page"]',
				'.p1j2gy66 > a[aria-label="Next"]'
			];
			for (const sel of selectors) {
				const el = document.querySelector(sel);
				if (el && el.href) return el.href;
			}
			return "";
		})()
		`, &nextPage),
	)

	if err != nil {
		return page1Links
	}

	// if next page exists
	if nextPage != "" {
		err = chromedp.Run(tabCtx,
			chromedp.Navigate(nextPage),
			chromedp.Sleep(3*time.Second),

			chromedp.Evaluate(`
			(async function() {
				const delay = ms => new Promise(r => setTimeout(r, ms));
				const step = 400;
				const total = document.body.scrollHeight;
				for (let y = 0; y <= total; y += step) {
					window.scrollTo(0, y);
					await delay(2000);
				}
				// Final pause at bottom so last items can render
				await delay(3000);
				window.scrollTo(0, 0);
				})()
			`, nil),

			chromedp.Sleep(4*time.Second),

			// extract 5 cards from second page
			chromedp.Evaluate(`
				Array.from(document.querySelectorAll('.cy5jw6o > a'))
					.slice(0,5)
					.map(a=>a.href)
			`, &page2Links),
		)

		if err != nil {
			return page1Links
		}
		return append(page1Links, page2Links...)
	}

	// no next page → take up to 10 from first page
	if len(page1Links) >= 10 {
		return page1Links[:10]
	}

	return page1Links
}


//
// WORKER POOL PRODUCT EXTRACTION
//

func (s *ChromedpScraper) extractProductsWorkerPool(
	cardLinks []string,
	workerCount int,
) []domain.Product {

	jobs := make(chan string, len(cardLinks))
	results := make(chan domain.Product, len(cardLinks))

	var wg sync.WaitGroup

	// workers
	for range workerCount {

		wg.Add(1)

		go func() {

			defer wg.Done()

			for url := range jobs {

				product, err := s.extractProduct(url)

				if err == nil {
					results <- product
				}
			}
		}()
	}

	// send jobs
	for _, link := range cardLinks {
		jobs <- link
	}

	close(jobs)

	wg.Wait()

	close(results)

	var products []domain.Product

	for p := range results {
		products = append(products, p)
	}

	return products
}

//
// PRODUCT EXTRACTION
//

func (s *ChromedpScraper) extractProduct(url string) (domain.Product, error) {

	// Timeout so it never hangs forever
	tabCtx, cancel := context.WithTimeout(s.allocatorCtx, 30*time.Second)
	defer cancel()

	browserCtx, browserCancel := chromedp.NewContext(tabCtx)
	defer browserCancel()

	var title, priceText, location, ratingText string

	err := chromedp.Run(browserCtx,
		chromedp.Navigate(url),
		chromedp.Sleep(4*time.Second),

		// Use Evaluate instead of Text - more resilient
		chromedp.Evaluate(`document.querySelector('h1')?.innerText || ""`, &title),
		chromedp.Evaluate(`
	(()=>{
		// first try u1opajno
		let el = document.querySelector('.u1opajno');
		if (el && el.textContent.trim()) {
			return el.textContent.trim();
		}

		// fallback to u174bpcy
		el = document.querySelector('.u174bpcy');
		if (el && el.textContent.trim()) {
			return el.textContent.trim();
		}

		return "";
	})()
	`, &priceText),

		chromedp.Evaluate(`
	(()=>{
		// first try
		let el = document.querySelector('._1t2xqmi > h3')
		if (el && el.textContent.trim()) {
			return el.textContent.trim();
		}

		// fallback to s1qk96pm
		el = document.querySelector('.s1qk96pm');
		if (el && el.textContent.trim()) {
			return el.textContent.trim();
		}

		return "";
	})()
	`, &location),

		chromedp.Evaluate(`
	(()=>{
		// first try
		let el = document.querySelector('[data-testid="pdp-reviews-highlight-banner-host-rating"] div[aria-hidden="true"]')
		if (el && el.textContent.trim()) {
			return el.textContent.trim();
		}

		// fallback to u174bpcy
		el = document.querySelector('.rmtgcc3');
		if (el && el.textContent.trim()) {
			return el.textContent.trim();
		}

		return "";
	})()
	`, &ratingText),
	)

	fmt.Printf("Scraped => title:%q price:%q location:%q rating:%q err:%v\n",
		title, priceText, location, ratingText, err)

	if err != nil {
		return domain.Product{}, err
	}

	return domain.Product{
		Title:    title,
		Price:    parsePrice(priceText),
		Location: location,
		URL:      url,
		Rating:   parseRating(ratingText),
	}, nil
}

//
// helpers
//

// Helper to safely get text without failing if selector missing
func safeText(sel string, val *string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		_ = chromedp.Text(sel, val, chromedp.ByQuery).Do(ctx)
		return nil // always nil — don't fail on missing element
	})
}

func parsePrice(price string) float32 {

	price = strings.ReplaceAll(price, "$", "")
	price = strings.ReplaceAll(price, ",", "")

	v, _ := strconv.ParseFloat(price, 32)

	return float32(v)
}

func parseRating(rating string) float32 {

	v, _ := strconv.ParseFloat(rating, 32)

	return float32(v)
}
