// Package scraper provides orchestration for scraping oil prices and weather
// data from multiple providers.
//
// commonMetrics, commonSnapshot and commonPrometheusMetrics hold what the two
// scrapers share. Scraper and WeatherScraper, in oil.go and weather.go, add the
// parts that are genuinely domain-specific: the provider interface they drive,
// the records they persist, and their headline measurement.
package scraper

import (
	"sync"
	"time"
)

// commonPrometheusMetrics is the part of the Prometheus recorder interface both
// scrapers use. PrometheusMetrics and WeatherPrometheusMetrics embed it and add
// the two recorders whose shape is domain-specific.
type commonPrometheusMetrics interface {
	RecordAPIRequest(provider, status string, duration float64)
	RecordLastScrape(provider string, timestamp float64)
	RecordDBOperation(operation, status string)
}

// commonMetrics holds the per-provider scrape metrics both scrapers record.
// Metrics and WeatherMetrics embed it and add their headline measurement, the
// last price or the last temperature.
type commonMetrics struct {
	mu                sync.RWMutex
	TotalRequests     int64
	TotalErrors       int64
	LastScrapeAt      *time.Time
	LastScrapeSuccess bool
	LastResponseTime  time.Duration
	LastError         *string
	LastRawResponse   string
}

// commonSnapshot is the shared half of a metrics snapshot. MetricsSnapshot and
// WeatherMetricsSnapshot embed it, so callers still read every field directly
// off the snapshot they were handed.
type commonSnapshot struct {
	TotalRequests     int64
	TotalErrors       int64
	LastScrapeAt      *time.Time
	LastScrapeSuccess bool
	LastResponseTime  time.Duration
	LastError         *string
	LastRawResponse   string
}

// snapshot copies the shared fields.
//
// The caller must already hold m.mu. That is why this is unexported and why
// the exported GetSnapshot lives on the embedding type: the headline
// measurement has to be read under the same lock as the rest, and RWMutex is
// not reentrant, so this method must not take the lock itself.
func (m *commonMetrics) snapshot() commonSnapshot {
	return commonSnapshot{
		TotalRequests:     m.TotalRequests,
		TotalErrors:       m.TotalErrors,
		LastScrapeAt:      m.LastScrapeAt,
		LastScrapeSuccess: m.LastScrapeSuccess,
		LastResponseTime:  m.LastResponseTime,
		LastError:         m.LastError,
		LastRawResponse:   m.LastRawResponse,
	}
}
