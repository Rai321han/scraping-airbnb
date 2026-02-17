package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

type LocationLink struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

func main() {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	var rawJSON string

	err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.airbnb.com/"),
		chromedp.Sleep(5*time.Second),
		chromedp.WaitVisible(`h2`, chromedp.ByQuery),

		// Scroll incrementally to trigger all lazy-loaded location cards
		chromedp.Evaluate(`
			(async function() {
				const delay = ms => new Promise(r => setTimeout(r, ms));
				const step = 400;
				const total = document.body.scrollHeight;
				for (let y = 0; y <= total; y += step) {
					window.scrollTo(0, y);
					await delay(250);
				}
				// Final pause at bottom so last items can render
				await delay(1000);
				window.scrollTo(0, 0);
			})()
		`, nil),

		// Wait for newly rendered elements to settle
		chromedp.Sleep(3*time.Second),

		chromedp.Evaluate(`
			(function() {
				let anchors = document.querySelectorAll('h2.skp76t2 > a');
				const results = [];
				anchors.forEach(a => {
					results.push({
						text: a.innerText.trim(),
						url: a.href
					});
				});

				return JSON.stringify(results);
			})()
		`, &rawJSON),
	)

	// for each link
	// 1. get 5 + 5 card link from first and second pagination (last child of div with .p1j2gy66 is the next button) page store those links
	// 2. go to each card and extract data
	//	title -> .tglziin > h1
	//	Price (per night price, numeric or string form) -> url has a check_in and a check_out query setting, set check_out date to next date of check_in and get price from div with ".u1opajno"
	//	Location (city, region, or country) -> div with .s1qk96pm
	//	Rating (if the page makes it available) -> div with .rmtgcc3
	//	URL - we can get this from card url
	//	Details
	// 3. export to a csv file

	if err != nil {
		log.Printf("Error during scraping: %v", err)
	}

	var links []LocationLink
	if err := json.Unmarshal([]byte(rawJSON), &links); err != nil {
		log.Printf("Failed to parse JSON: %v\nRaw: %s", err, rawJSON)
		return
	}

	if len(links) == 0 {
		fmt.Println("No links found.")
		return
	}

	fmt.Printf("Found %d location-based URLs:\n\n", len(links))
	for i, link := range links {
		fmt.Printf("[%d] Text: %s\n    URL:  %s\n\n", i+1, link.Text, link.URL)
	}
}