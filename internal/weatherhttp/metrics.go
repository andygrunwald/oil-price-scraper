// Package weatherhttp provides HTTP server functionality for the weather scraper.
package weatherhttp

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the weather scraper.
type Metrics struct {
	APIRequestsTotal       *prometheus.CounterVec
	APIRequestDuration     *prometheus.HistogramVec
	LastScrapeTimestamp     *prometheus.GaugeVec
	CurrentTemperature     *prometheus.GaugeVec
	DBOperationsTotal      *prometheus.CounterVec
	ObservationsStoredTotal *prometheus.GaugeVec
}

// NewMetrics creates and registers Prometheus metrics.
func NewMetrics() *Metrics {
	return &Metrics{
		APIRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "weatherscraper_api_requests_total",
				Help: "Total number of API requests by provider and status",
			},
			[]string{"provider", "status"},
		),
		APIRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "weatherscraper_api_request_duration_seconds",
				Help:    "API request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"provider"},
		),
		LastScrapeTimestamp: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "weatherscraper_last_scrape_timestamp",
				Help: "Timestamp of the last successful scrape",
			},
			[]string{"provider"},
		),
		CurrentTemperature: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "weatherscraper_current_temperature_celsius",
				Help: "Current temperature in Celsius",
			},
			[]string{"provider"},
		),
		DBOperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "weatherscraper_db_operations_total",
				Help: "Total number of database operations by type and status",
			},
			[]string{"operation", "status"},
		),
		ObservationsStoredTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "weatherscraper_observations_stored_total",
				Help: "Total number of observations stored in database by provider",
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

// RecordCurrentTemperature records the current temperature.
func (m *Metrics) RecordCurrentTemperature(provider string, temp float64) {
	m.CurrentTemperature.WithLabelValues(provider).Set(temp)
}

// RecordDBOperation records a database operation metric.
func (m *Metrics) RecordDBOperation(operation, status string) {
	m.DBOperationsTotal.WithLabelValues(operation, status).Inc()
}

// RecordObservationsStored records the total number of observations stored for a provider.
func (m *Metrics) RecordObservationsStored(provider string, count float64) {
	m.ObservationsStoredTotal.WithLabelValues(provider).Set(count)
}
