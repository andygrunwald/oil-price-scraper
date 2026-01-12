// Package models provides shared data types for the oil price scraper.
package models

import (
	"time"
)

// PriceScope indicates the geographical scope of the price.
type PriceScope string

const (
	// PriceScopeLocal indicates a zip code specific price.
	PriceScopeLocal PriceScope = "local"
	// PriceScopeNational indicates a nationwide average price.
	PriceScopeNational PriceScope = "national"
)

// PriceResult is the unified return type for all providers.
type PriceResult struct {
	// Date is the date the price is valid for.
	Date time.Time
	// PricePer100L is the price in EUR per 100 liters.
	PricePer100L float64
	// Currency is the currency code (EUR).
	Currency string
	// Provider is the provider name (e.g., "heizoel24", "hoyer").
	Provider string
	// ProductType is the product variant (e.g., "standard", "bestpreis", "eco", "express").
	ProductType string
	// Scope indicates whether the price is local (zip code) or national.
	Scope PriceScope
	// ZipCode is only set if Scope is local.
	ZipCode string
	// RawResponse is the original API response (JSON).
	RawResponse []byte
	// FetchedAt is when the data was fetched.
	FetchedAt time.Time
}

// OilPrice represents a stored oil price record from the database.
type OilPrice struct {
	ID           uint64
	Provider     string
	ProductType  string
	PriceDate    time.Time
	PricePer100L float64
	Currency     string
	Scope        PriceScope
	ZipCode      *string
	RawResponse  []byte
	FetchedAt    time.Time
	CreatedAt    time.Time
}

// ProviderStatus holds the operational status of a provider.
type ProviderStatus struct {
	Enabled            bool       `json:"enabled"`
	LastScrapeAt       *time.Time `json:"last_scrape_at"`
	LastScrapeSuccess  bool       `json:"last_scrape_success"`
	LastResponseTimeMs int64      `json:"last_response_time_ms"`
	LastPrice          *float64   `json:"last_price"`
	LastError          *string    `json:"last_error"`
	TotalRequests      int64      `json:"total_requests"`
	TotalErrors        int64      `json:"total_errors"`
	LastRawResponse    string     `json:"last_raw_response,omitempty"`
}

// StatusResponse is the response for the /status endpoint.
type StatusResponse struct {
	Status                string                    `json:"status"`
	UptimeSeconds         int64                     `json:"uptime_seconds"`
	SchedulerRunning      bool                      `json:"scheduler_running"`
	NextScrapeAt          *time.Time                `json:"next_scrape_at,omitempty"`
	LastScheduledScrapeAt *time.Time                `json:"last_scheduled_scrape_at,omitempty"`
	Providers             map[string]ProviderStatus `json:"providers"`
	Database              DatabaseStatus            `json:"database"`
}

// DatabaseStatus holds the database connection status.
type DatabaseStatus struct {
	Connected         bool  `json:"connected"`
	TotalPricesStored int64 `json:"total_prices_stored"`
}
