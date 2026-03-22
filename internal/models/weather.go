package models

import (
	"math"
	"time"
)

// RoundCoord rounds a coordinate to 4 decimal places (~11m precision).
func RoundCoord(v float64) float64 {
	return math.Round(v*10000) / 10000
}

// WeatherResult is the unified return type for all weather providers.
type WeatherResult struct {
	// Date is the date the observation is valid for.
	Date time.Time
	// Provider is the provider name (e.g., "openmeteo", "brightsky").
	Provider string
	// Latitude is the latitude of the observation location.
	Latitude float64
	// Longitude is the longitude of the observation location.
	Longitude float64
	// TemperatureMinC is the minimum temperature in Celsius.
	TemperatureMinC *float64
	// TemperatureMaxC is the maximum temperature in Celsius.
	TemperatureMaxC *float64
	// TemperatureMeanC is the mean temperature in Celsius.
	TemperatureMeanC *float64
	// PrecipitationMmSum is the total precipitation in millimeters.
	PrecipitationMmSum *float64
	// WindSpeedMaxKmh is the maximum wind speed in km/h.
	WindSpeedMaxKmh *float64
	// WindGustMaxKmh is the maximum wind gust speed in km/h.
	WindGustMaxKmh *float64
	// SunshineDurationS is the sunshine duration in seconds.
	SunshineDurationS *float64
	// CloudCoverPercent is the mean cloud cover in percent.
	CloudCoverPercent *float64
	// HumidityPercent is the mean relative humidity in percent.
	HumidityPercent *float64
	// PressureHpa is the mean sea-level pressure in hPa.
	PressureHpa *float64
	// RawResponse is the original API response (JSON).
	RawResponse []byte
	// FetchedAt is when the data was fetched.
	FetchedAt time.Time
}

// WeatherObservation represents a stored weather observation from the database.
type WeatherObservation struct {
	ID                 uint64
	Provider           string
	ObservationDate    time.Time
	Latitude           float64
	Longitude          float64
	TemperatureMinC    *float64
	TemperatureMaxC    *float64
	TemperatureMeanC   *float64
	PrecipitationMmSum *float64
	WindSpeedMaxKmh    *float64
	WindGustMaxKmh     *float64
	SunshineDurationS  *float64
	CloudCoverPercent  *float64
	HumidityPercent    *float64
	PressureHpa        *float64
	RawResponse        []byte
	FetchedAt          time.Time
	CreatedAt          time.Time
}

// WeatherProviderStatus holds the operational status of a weather provider.
type WeatherProviderStatus struct {
	Enabled            bool       `json:"enabled"`
	LastScrapeAt       *time.Time `json:"last_scrape_at"`
	LastScrapeSuccess  bool       `json:"last_scrape_success"`
	LastResponseTimeMs int64      `json:"last_response_time_ms"`
	LastTemperature    *float64   `json:"last_temperature,omitempty"`
	LastError          *string    `json:"last_error"`
	TotalRequests      int64      `json:"total_requests"`
	TotalErrors        int64      `json:"total_errors"`
	LastRawResponse    string     `json:"last_raw_response,omitempty"`
}

// WeatherStatusResponse is the response for the weather /status endpoint.
type WeatherStatusResponse struct {
	Status                string                           `json:"status"`
	UptimeSeconds         int64                            `json:"uptime_seconds"`
	SchedulerRunning      bool                             `json:"scheduler_running"`
	NextScrapeAt          *time.Time                       `json:"next_scrape_at,omitempty"`
	LastScheduledScrapeAt *time.Time                       `json:"last_scheduled_scrape_at,omitempty"`
	Providers             map[string]WeatherProviderStatus `json:"providers"`
	Database              WeatherDatabaseStatus            `json:"database"`
}

// WeatherDatabaseStatus holds the database connection status for weather.
type WeatherDatabaseStatus struct {
	Connected               bool  `json:"connected"`
	TotalObservationsStored int64 `json:"total_observations_stored"`
}

// Float64Ptr returns a pointer to the given float64 value.
func Float64Ptr(v float64) *float64 {
	return &v
}
