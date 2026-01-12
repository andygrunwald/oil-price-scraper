// Package api provides the interface and types for oil price API providers.
package api

import (
	"context"
	"time"

	"github.com/andygrunwald/oil-price-scraper/internal/models"
)

// Provider defines the interface for oil price API providers.
type Provider interface {
	// Name returns the provider identifier.
	Name() string

	// FetchCurrentPrices fetches today's prices (may return multiple for different product types).
	FetchCurrentPrices(ctx context.Context) ([]models.PriceResult, error)

	// FetchHistoricalPrices fetches prices for a date range (if supported).
	FetchHistoricalPrices(ctx context.Context, from, to time.Time) ([]models.PriceResult, error)

	// SupportsBackfill returns true if the provider supports historical data.
	SupportsBackfill() bool

	// PriceScope returns whether the price is local (zip code) or nationwide.
	PriceScope() models.PriceScope
}
