// Package weatherapi provides the interface and types for weather API providers.
package weatherapi

import (
	"context"
	"time"

	"github.com/andygrunwald/oil-price-scraper/internal/models"
)

// Provider defines the interface for weather API providers.
type Provider interface {
	// Name returns the provider identifier.
	Name() string

	// FetchCurrentWeather fetches today's weather observation for the given location.
	FetchCurrentWeather(ctx context.Context, lat, lon float64) ([]models.WeatherResult, error)

	// FetchHistoricalWeather fetches weather observations for a date range (if supported).
	FetchHistoricalWeather(ctx context.Context, lat, lon float64, from, to time.Time) ([]models.WeatherResult, error)

	// SupportsBackfill returns true if the provider supports historical data.
	SupportsBackfill() bool

	// RequiresAPIKey returns true if the provider requires an API key.
	RequiresAPIKey() bool
}
