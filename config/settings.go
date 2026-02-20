package config

import "time"

// BrowserConfig controls headless Chrome flags.
type BrowserConfig struct {
	Headless   bool
	DisableGPU bool
	NoSandbox  bool
	DisableShm bool
	UserAgent  string
}

// TimingConfig controls all wait/sleep durations throughout the scraper.
type TimingConfig struct {
	// How long to wait after initial page navigation before interacting
	PageLoadWait time.Duration
	// Delay between each scroll step (keeps scroll synchronous)
	ScrollStepDelay time.Duration
	// Extra wait after reaching the bottom so lazy content can render
	ScrollBottomWait time.Duration
	// Wait after scrolling back to top before extracting data
	AfterScrollWait time.Duration
	// How long to wait on a product detail page before extracting
	ProductPageWait time.Duration
	// Hard timeout for a single product page extraction
	ProductTimeout time.Duration
}

// ConcurrencyConfig controls goroutine and worker pool limits.
type ConcurrencyConfig struct {
	// Max concurrent browser tabs when scraping location pages
	LocationWorkers int
	// Worker pool size when extracting individual product pages
	ProductWorkers int
}

// ScraperConfig controls extraction limits.
type ScraperConfig struct {
	// Cards to collect from page 1 of a location search
	CardsPage1 int
	// Cards to collect from page 2 (if pagination exists)
	CardsPage2 int
	// Pixels to advance per scroll step
	ScrollStep int
}

// RetryConfig controls retry behavior for resilience.
type RetryConfig struct {
	// Max number of retry attempts for failed operations
	MaxRetries int
	// Initial backoff duration before first retry
	InitialBackoff time.Duration
	// Max backoff duration (caps exponential growth)
	MaxBackoff time.Duration
}

// StealthConfig controls anti-detection and stealth behavior.
type StealthConfig struct {
	// Enable random delays between requests
	RandomDelayEnabled bool
	// Min random delay (ms)
	RandomDelayMin time.Duration
	// Max random delay (ms)
	RandomDelayMax time.Duration
	// Enable random user agent selection
	RandomUserAgentEnabled bool
	// Max requests per second (rate limiting; 0 = unlimited)
	MaxRequestsPerSecond float64
}

// Config is the root configuration passed into the scraper.
type Config struct {
	Browser     BrowserConfig
	Timing      TimingConfig
	Concurrency ConcurrencyConfig
	Scraper     ScraperConfig
	Retry       RetryConfig
	Stealth     StealthConfig
}

// Default returns a conservative production-ready configuration.
func Default() *Config {
	return &Config{
		Browser: BrowserConfig{
			Headless:   true,
			DisableGPU: true,
			NoSandbox:  true,
			DisableShm: true,
			UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		},
		Timing: TimingConfig{
			PageLoadWait:     5 * time.Second,
			ScrollStepDelay:  400 * time.Millisecond,
			ScrollBottomWait: 4 * time.Second,
			AfterScrollWait:  4 * time.Second,
			ProductPageWait:  4 * time.Second,
			ProductTimeout:   70 * time.Second,
		},
		Concurrency: ConcurrencyConfig{
			LocationWorkers: 3,
			ProductWorkers:  3,
		},
		Scraper: ScraperConfig{
			CardsPage1: 5,
			CardsPage2: 5,
			ScrollStep: 400,
		},
		Retry: RetryConfig{
			MaxRetries:     3,
			InitialBackoff: 2 * time.Second,
			MaxBackoff:     10 * time.Second,
		},
		Stealth: StealthConfig{
			RandomDelayEnabled:     true,
			RandomDelayMin:         4 * time.Second,
			RandomDelayMax:         6 * time.Second,
			RandomUserAgentEnabled: true,
			MaxRequestsPerSecond:   4.0,
		},
	}
}

// Dev returns a faster config suited for local development and testing.
func Dev() *Config {
	cfg := Default()
	cfg.Timing.ScrollStepDelay = 400 * time.Millisecond
	cfg.Timing.ScrollBottomWait = 2 * time.Second
	cfg.Timing.PageLoadWait = 4 * time.Second
	cfg.Timing.ProductPageWait = 3 * time.Second
	cfg.Concurrency.LocationWorkers = 1
	cfg.Concurrency.ProductWorkers = 2
	cfg.Stealth.RandomDelayMin = 2 * time.Second
	cfg.Stealth.RandomDelayMax = 4 * time.Second
	cfg.Stealth.MaxRequestsPerSecond = 10.0
	return cfg
}


// DefaultUserAgents returns a pool of realistic desktop browser user agents.
func DefaultUserAgents() []string {
	return []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
	}
}
