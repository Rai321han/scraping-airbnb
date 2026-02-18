package scraper

import (
	"context"
	"fmt"
	"scraping-airbnb/config"
	"time"

	"github.com/chromedp/chromedp"
)

// newAllocator creates a shared Chrome process from the given browser config.
// All tabs (contexts) must be created from the returned context.
func NewAllocator(parent context.Context, cfg *config.BrowserConfig) context.Context {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", cfg.Headless),
		chromedp.Flag("disable-gpu", cfg.DisableGPU),
		chromedp.Flag("no-sandbox", cfg.NoSandbox),
		chromedp.Flag("disable-setuid-sandbox", cfg.NoSandbox),
		chromedp.Flag("disable-dev-shm-usage", cfg.DisableShm),
		chromedp.UserAgent(cfg.UserAgent),
	)
	allocCtx, _ := chromedp.NewExecAllocator(parent, opts...)
	return allocCtx
}

// newTab opens a new browser tab from the allocator context.
func NewTab(allocCtx context.Context) (context.Context, context.CancelFunc) {
	return chromedp.NewContext(allocCtx)
}

// newTabWithTimeout opens a browser tab that auto-cancels after the given duration.
func NewTabWithTimeout(allocCtx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	tCtx, tCancel := context.WithTimeout(allocCtx, timeout)
	bCtx, bCancel := chromedp.NewContext(tCtx)
	return bCtx, func() {
		bCancel()
		tCancel()
	}
}

// scrollToBottom incrementally scrolls the page so lazy-loaded content renders.
// It stays at the bottom when done — call scrollToTop separately if needed.
// Using ActionFunc (not async JS) ensures each step actually blocks.
func ScrollToBottom(cfg *config.TimingConfig, scrollStep int) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		var height int
		if err := chromedp.Evaluate(`document.body.scrollHeight`, &height).Do(ctx); err != nil {
			return fmt.Errorf("scrollToBottom: get height: %w", err)
		}

		for y := 0; y <= height; y += scrollStep {
			if err := chromedp.Evaluate(
				fmt.Sprintf(`window.scrollTo(0, %d)`, y), nil,
			).Do(ctx); err != nil {
				return fmt.Errorf("scrollToBottom: scroll to %d: %w", y, err)
			}
			time.Sleep(cfg.ScrollStepDelay)
		}

		// Final pause so last lazy-loaded items have time to render
		time.Sleep(cfg.ScrollBottomWait)
		return nil
	}
}