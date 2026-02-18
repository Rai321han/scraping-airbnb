package airbnb

import "fmt"

// ── Location page JS ──────────────────────────────────────────────────────────

// locationLinksJS collects all location card hrefs from the Airbnb homepage.
const locationLinksJS = `
JSON.stringify(
	Array.from(document.querySelectorAll('h2.skp76t2 > a'))
		.map(a => ({ url: a.href }))
)
`

// ── Listing search page JS ────────────────────────────────────────────────────

// cardLinksJS returns JS that collects up to `limit` listing card hrefs.
func cardLinksJS(limit int) string {
	return fmt.Sprintf(`
		Array.from(document.querySelectorAll('.cy5jw6o > a'))
			.slice(0, %d)
			.map(a => a.href)
	`, limit)
}

// nextPageJS finds Airbnb's pagination "Next" anchor using multiple strategies.
// Strategy 1 — aria-label selectors (stable across class name changes).
// Strategy 2 — cursor= param in href (Airbnb's pagination mechanism).
const nextPageJS = `
(()=>{
	const labeled = [
		'a[aria-label="Next"]',
		'a[aria-label="Next page"]',
		'.p1uqa2vx > a[aria-label="Next page"]',
		'.p1j2gy66 > a[aria-label="Next"]',
	];
	for (const sel of labeled) {
		const el = document.querySelector(sel);
		if (el?.href) return el.href;
	}
	const cursorEl = Array.from(document.querySelectorAll('a'))
		.find(a => a.href.includes('cursor=') && a.href.includes('pagination_search=true'));
	return cursorEl?.href || "";
})()
`

// debugPaginationJS dumps pagination state to stdout for troubleshooting.
const debugPaginationJS = `
JSON.stringify({
	nextAria:   !!document.querySelector('a[aria-label="Next"]'),
	nextHref:   (document.querySelector('a[aria-label="Next"]')?.href || "").slice(0, 80),
	cursorLink: (Array.from(document.querySelectorAll('a'))
		.find(a => a.href.includes('cursor='))?.href || "").slice(0, 100),
	cardCount:  document.querySelectorAll('.cy5jw6o > a').length,
})
`

// ── Product detail page JS ────────────────────────────────────────────────────
// Each selector tries the most specific/stable target first, then falls back.

// titleJS extracts the listing title from the h1.
const titleJS = `document.querySelector('h1')?.innerText?.trim() || ""`

// priceJS extracts the nightly price, trying known class names in order.
const priceJS = `
(()=>{
	for (const sel of ['.u1opajno', '.u174bpcy']) {
		const text = document.querySelector(sel)?.textContent?.trim();
		if (text) return text;
	}
	return "";
})()
`

// locationJS extracts the listing location/neighbourhood.
const locationJS = `
(()=>{
	for (const sel of ['._1t2xqmi > h3', '.s1qk96pm']) {
		const text = document.querySelector(sel)?.textContent?.trim();
		if (text) return text;
	}
	return "";
})()
`

// ratingJS extracts the host/listing rating.
const ratingJS = `
(()=>{
	for (const sel of [
		'[data-testid="pdp-reviews-highlight-banner-host-rating"] div[aria-hidden="true"]',
		'.rmtgcc3',
	]) {
		const text = document.querySelector(sel)?.textContent?.trim();
		if (text) return text;
	}
	return "";
})()
`