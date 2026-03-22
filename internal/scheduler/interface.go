package scheduler

import "context"

// ScraperInterface defines what the scheduler needs from a scraper.
// Both the oil scraper and weather scraper implement this interface.
type ScraperInterface interface {
	// ScrapeAll scrapes data from all registered providers.
	ScrapeAll(ctx context.Context) error

	// GetProviderNames returns the names of all registered providers.
	GetProviderNames() []string

	// HasScrapedToday checks if a provider has already been scraped today.
	HasScrapedToday(ctx context.Context, providerName string) (bool, error)

	// ScrapeProvider scrapes data from a single provider.
	ScrapeProvider(ctx context.Context, providerName string) error
}
