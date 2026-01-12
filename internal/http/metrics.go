// Package http provides HTTP server functionality for the oil price scraper.
package http

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the scraper.
type Metrics struct {
	// API request metrics
	APIRequestsTotal   *prometheus.CounterVec
	APIRequestDuration *prometheus.HistogramVec

	// Scrape metrics
	LastScrapeTimestamp *prometheus.GaugeVec
	CurrentPriceEUR     *prometheus.GaugeVec

	// Database metrics
	DBOperationsTotal *prometheus.CounterVec
	PricesStoredTotal *prometheus.GaugeVec
}

// NewMetrics creates and registers Prometheus metrics.
func NewMetrics() *Metrics {
	return &Metrics{
		APIRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "oilscraper_api_requests_total",
				Help: "Total number of API requests by provider and status",
			},
			[]string{"provider", "status"},
		),
		APIRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "oilscraper_api_request_duration_seconds",
				Help:    "API request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"provider"},
		),
		LastScrapeTimestamp: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "oilscraper_last_scrape_timestamp",
				Help: "Timestamp of the last successful scrape",
			},
			[]string{"provider"},
		),
		CurrentPriceEUR: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "oilscraper_current_price_eur",
				Help: "Current oil price in EUR per 100L",
			},
			[]string{"provider", "scope", "product_type"},
		),
		DBOperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "oilscraper_db_operations_total",
				Help: "Total number of database operations by type and status",
			},
			[]string{"operation", "status"},
		),
		PricesStoredTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "oilscraper_prices_stored_total",
				Help: "Total number of prices stored in database by provider",
			},
			[]string{"provider"},
		),
	}
}

// RecordAPIRequest records an API request metric.
func (m *Metrics) RecordAPIRequest(provider, status string, duration float64) {
	m.APIRequestsTotal.WithLabelValues(provider, status).Inc()
	m.APIRequestDuration.WithLabelValues(provider).Observe(duration)
}

// RecordLastScrape records the last successful scrape timestamp.
func (m *Metrics) RecordLastScrape(provider string, timestamp float64) {
	m.LastScrapeTimestamp.WithLabelValues(provider).Set(timestamp)
}

// RecordCurrentPrice records the current oil price.
func (m *Metrics) RecordCurrentPrice(provider, scope, productType string, price float64) {
	m.CurrentPriceEUR.WithLabelValues(provider, scope, productType).Set(price)
}

// RecordDBOperation records a database operation metric.
func (m *Metrics) RecordDBOperation(operation, status string) {
	m.DBOperationsTotal.WithLabelValues(operation, status).Inc()
}

// RecordPricesStored records the total number of prices stored for a provider.
func (m *Metrics) RecordPricesStored(provider string, count float64) {
	m.PricesStoredTotal.WithLabelValues(provider).Set(count)
}
