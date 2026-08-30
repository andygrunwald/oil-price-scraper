// Package weatherconfig provides configuration structures and loading for the weather scraper.
package weatherconfig

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the weather scraper.
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
	// Latitude for weather location
	Latitude float64
	// Longitude for weather location
	Longitude float64
	// Scrape hour (0-23)
	ScrapeHour int
	// Enabled providers
	Providers []string
	// Visual Crossing API key
	VisualCrossingAPIKey string
	// OpenWeather API key
	OpenWeatherAPIKey string
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
		StoreRawResponse: false,
		HTTPAddr:         ":8081",
		Latitude:         0,
		Longitude:        0,
		ScrapeHour:       7,
		Providers:        []string{"openmeteo"},
		Backfill: BackfillConfig{
			Provider: "openmeteo",
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
	if v := os.Getenv("LATITUDE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.Latitude = f
		}
	}
	if v := os.Getenv("LONGITUDE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.Longitude = f
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
	if v := os.Getenv("VISUAL_CROSSING_API_KEY"); v != "" {
		c.VisualCrossingAPIKey = v
	}
	if v := os.Getenv("OPENWEATHER_API_KEY"); v != "" {
		c.OpenWeatherAPIKey = v
	}
}
