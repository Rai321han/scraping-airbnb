package utils

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/chromedp/chromedp"
)


func SafeText(sel string, val *string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		_ = chromedp.Text(sel, val, chromedp.ByQuery).Do(ctx)
		return nil // always nil — don't fail on missing element
	})
}

func ParsePrice(price string) float32 {

	price = strings.ReplaceAll(price, "$", "")
	price = strings.ReplaceAll(price, ",", "")

	v, _ := strconv.ParseFloat(price, 32)

	return float32(v)
}

func ParseRating(rating string) float32 {

	v, _ := strconv.ParseFloat(rating, 32)

	return float32(v)
}

func ParseNights(daysText string) int {
	// examples:
	// "for 3 nights"
	// "for 1 night"
	var nights int
	fmt.Sscanf(daysText, "for %d night", &nights)
	return nights
}