package config

import (
	"os"
	"strconv"
)

// Weather holds all configuration for the weather scraper.
type Weather struct {
	Common
	// Latitude for weather location
	Latitude float64
	// Longitude for weather location
	Longitude float64
	// Visual Crossing API key
	VisualCrossingAPIKey string
	// OpenWeather API key
	OpenWeatherAPIKey string
}

// DefaultWeather returns a Weather configuration with default values.
func DefaultWeather() *Weather {
	return &Weather{
		Common: Common{
			PostgresDSN:      "",
			LogLevel:         "info",
			LogFormat:        "json",
			StoreRawResponse: false,
			HTTPAddr:         ":8081",
			ScrapeHour:       7,
			Providers:        []string{"openmeteo"},
			Backfill: BackfillConfig{
				Provider: "openmeteo",
				MinDelay: 1,
				MaxDelay: 5,
			},
		},
		Latitude:  0,
		Longitude: 0,
	}
}

// LoadFromEnv loads the weather scraper configuration from environment variables.
func (c *Weather) LoadFromEnv() {
	c.Common.LoadFromEnv()

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
	if v := os.Getenv("VISUAL_CROSSING_API_KEY"); v != "" {
		c.VisualCrossingAPIKey = v
	}
	if v := os.Getenv("OPENWEATHER_API_KEY"); v != "" {
		c.OpenWeatherAPIKey = v
	}
}
