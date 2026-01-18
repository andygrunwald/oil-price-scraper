// Package config provides configuration structures and loading for the oil price scraper.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the oil price scraper.
type Config struct {
	// PostgreSQL connection string
	PostgresDSN string
	// Log level (debug, info, warn, error)
	LogLevel string
	// Log format (json, console)
	LogFormat string
	// Store raw API responses in database
	StoreRawResponse bool
	// HTTP server address
	HTTPAddr string
	// Zip code for local price APIs
	ZipCode string
	// Order amount in liters
	OrderAmount int
	// Scrape hour (0-23)
	ScrapeHour int
	// Enabled providers
	Providers []string
	// Backfill settings
	Backfill BackfillConfig
}

// BackfillConfig holds configuration for backfilling historical data.
type BackfillConfig struct {
	// Start date for backfill
	From time.Time
	// End date for backfill
	To time.Time
	// Provider to backfill from
	Provider string
	// Minimum delay between requests in seconds
	MinDelay int
	// Maximum delay between requests in seconds
	MaxDelay int
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		PostgresDSN:      "",
		LogLevel:         "info",
		LogFormat:        "json",
		StoreRawResponse: true,
		HTTPAddr:         ":8080",
		ZipCode:          "47259",
		OrderAmount:      3000,
		ScrapeHour:       6,
		Providers:        []string{"heizoel24", "hoyer"},
		Backfill: BackfillConfig{
			Provider: "heizoel24",
			MinDelay: 1,
			MaxDelay: 5,
		},
	}
}

// LoadFromEnv loads configuration from environment variables.
func (c *Config) LoadFromEnv() {
	if v := os.Getenv("POSTGRES_DSN"); v != "" {
		c.PostgresDSN = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		c.LogFormat = v
	}
	if v := os.Getenv("STORE_RAW_RESPONSE"); v != "" {
		c.StoreRawResponse = strings.ToLower(v) == "true"
	}
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		c.HTTPAddr = v
	}
	if v := os.Getenv("ZIP_CODE"); v != "" {
		c.ZipCode = v
	}
	if v := os.Getenv("ORDER_AMOUNT"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.OrderAmount = i
		}
	}
	if v := os.Getenv("SCRAPE_HOUR"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i >= 0 && i <= 23 {
			c.ScrapeHour = i
		}
	}
	if v := os.Getenv("PROVIDERS"); v != "" {
		c.Providers = strings.Split(v, ",")
	}
}
