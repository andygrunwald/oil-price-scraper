package config

import (
	"reflect"
	"slices"
	"testing"
)

func TestDefaultOil(t *testing.T) {
	c := DefaultOil()

	if c.PostgresDSN != "" {
		t.Errorf("PostgresDSN = %q, want empty", c.PostgresDSN)
	}
	if c.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", c.LogLevel, "info")
	}
	if c.LogFormat != "json" {
		t.Errorf("LogFormat = %q, want %q", c.LogFormat, "json")
	}
	if c.StoreRawResponse {
		t.Error("StoreRawResponse = true, want false")
	}
	if c.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want %q", c.HTTPAddr, ":8080")
	}
	if c.ScrapeHour != 6 {
		t.Errorf("ScrapeHour = %d, want 6", c.ScrapeHour)
	}
	if want := []string{"heizoel24", "hoyer"}; !slices.Equal(c.Providers, want) {
		t.Errorf("Providers = %v, want %v", c.Providers, want)
	}
	if c.Backfill.Provider != "heizoel24" {
		t.Errorf("Backfill.Provider = %q, want %q", c.Backfill.Provider, "heizoel24")
	}
	if c.Backfill.MinDelay != 1 || c.Backfill.MaxDelay != 5 {
		t.Errorf("Backfill delays = %d/%d, want 1/5", c.Backfill.MinDelay, c.Backfill.MaxDelay)
	}
	if c.ZipCode != "" {
		t.Errorf("ZipCode = %q, want empty", c.ZipCode)
	}
	if c.OrderAmount != 3000 {
		t.Errorf("OrderAmount = %d, want 3000", c.OrderAmount)
	}
}

func TestDefaultWeather(t *testing.T) {
	c := DefaultWeather()

	if c.PostgresDSN != "" {
		t.Errorf("PostgresDSN = %q, want empty", c.PostgresDSN)
	}
	if c.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", c.LogLevel, "info")
	}
	if c.LogFormat != "json" {
		t.Errorf("LogFormat = %q, want %q", c.LogFormat, "json")
	}
	if c.StoreRawResponse {
		t.Error("StoreRawResponse = true, want false")
	}
	if c.HTTPAddr != ":8081" {
		t.Errorf("HTTPAddr = %q, want %q", c.HTTPAddr, ":8081")
	}
	if c.ScrapeHour != 7 {
		t.Errorf("ScrapeHour = %d, want 7", c.ScrapeHour)
	}
	if want := []string{"openmeteo"}; !slices.Equal(c.Providers, want) {
		t.Errorf("Providers = %v, want %v", c.Providers, want)
	}
	if c.Backfill.Provider != "openmeteo" {
		t.Errorf("Backfill.Provider = %q, want %q", c.Backfill.Provider, "openmeteo")
	}
	if c.Backfill.MinDelay != 1 || c.Backfill.MaxDelay != 5 {
		t.Errorf("Backfill delays = %d/%d, want 1/5", c.Backfill.MinDelay, c.Backfill.MaxDelay)
	}
	if c.Latitude != 0 || c.Longitude != 0 {
		t.Errorf("Latitude/Longitude = %v/%v, want 0/0", c.Latitude, c.Longitude)
	}
	if c.VisualCrossingAPIKey != "" || c.OpenWeatherAPIKey != "" {
		t.Error("API keys should default to empty")
	}
}

// TestCommonLoadFromEnvUnset verifies that an unset environment leaves every
// default untouched.
func TestCommonLoadFromEnvUnset(t *testing.T) {
	for _, name := range []string{
		"POSTGRES_DSN", "LOG_LEVEL", "LOG_FORMAT", "STORE_RAW_RESPONSE",
		"HTTP_ADDR", "SCRAPE_HOUR", "PROVIDERS",
	} {
		t.Setenv(name, "")
	}

	c := DefaultOil()
	c.LoadFromEnv()

	if !reflect.DeepEqual(c, DefaultOil()) {
		t.Errorf("LoadFromEnv() with unset environment changed the config: %+v", c)
	}
}

func TestCommonLoadFromEnv(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "postgres://localhost:5432/oil")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "console")
	t.Setenv("STORE_RAW_RESPONSE", "true")
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("SCRAPE_HOUR", "12")
	t.Setenv("PROVIDERS", "hoyer,heizoel24")

	c := DefaultOil()
	c.LoadFromEnv()

	if c.PostgresDSN != "postgres://localhost:5432/oil" {
		t.Errorf("PostgresDSN = %q", c.PostgresDSN)
	}
	if c.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", c.LogLevel, "debug")
	}
	if c.LogFormat != "console" {
		t.Errorf("LogFormat = %q, want %q", c.LogFormat, "console")
	}
	if !c.StoreRawResponse {
		t.Error("StoreRawResponse = false, want true")
	}
	if c.HTTPAddr != ":9090" {
		t.Errorf("HTTPAddr = %q, want %q", c.HTTPAddr, ":9090")
	}
	if c.ScrapeHour != 12 {
		t.Errorf("ScrapeHour = %d, want 12", c.ScrapeHour)
	}
	if want := []string{"hoyer", "heizoel24"}; !slices.Equal(c.Providers, want) {
		t.Errorf("Providers = %v, want %v", c.Providers, want)
	}
}

func TestStoreRawResponseFromEnv(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"false", false},
		{"1", false},
		{"yes", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Setenv("STORE_RAW_RESPONSE", tt.value)

			c := DefaultOil()
			c.LoadFromEnv()

			if c.StoreRawResponse != tt.want {
				t.Errorf("STORE_RAW_RESPONSE=%q gives %v, want %v", tt.value, c.StoreRawResponse, tt.want)
			}
		})
	}
}

// TestScrapeHourFromEnv covers the 0-23 range guard: out-of-range and
// unparsable values are ignored and keep the default.
func TestScrapeHourFromEnv(t *testing.T) {
	tests := []struct {
		value string
		want  int
	}{
		{"0", 0},
		{"12", 12},
		{"23", 23},
		{"24", 6},
		{"-1", 6},
		{"abc", 6},
		{"6.5", 6},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Setenv("SCRAPE_HOUR", tt.value)

			c := DefaultOil()
			c.LoadFromEnv()

			if c.ScrapeHour != tt.want {
				t.Errorf("SCRAPE_HOUR=%q gives %d, want %d", tt.value, c.ScrapeHour, tt.want)
			}
		})
	}
}

func TestOilLoadFromEnv(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "postgres://localhost:5432/oil")
	t.Setenv("ZIP_CODE", "47259")
	t.Setenv("ORDER_AMOUNT", "1500")

	c := DefaultOil()
	c.LoadFromEnv()

	// The shared field proves the embedded Common.LoadFromEnv still runs.
	if c.PostgresDSN != "postgres://localhost:5432/oil" {
		t.Errorf("PostgresDSN = %q, embedded Common.LoadFromEnv did not run", c.PostgresDSN)
	}
	if c.ZipCode != "47259" {
		t.Errorf("ZipCode = %q, want %q", c.ZipCode, "47259")
	}
	if c.OrderAmount != 1500 {
		t.Errorf("OrderAmount = %d, want 1500", c.OrderAmount)
	}
}

func TestOrderAmountFromEnv(t *testing.T) {
	tests := []struct {
		value string
		want  int
	}{
		{"1500", 1500},
		{"abc", 3000},
		{"1.5", 3000},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Setenv("ORDER_AMOUNT", tt.value)

			c := DefaultOil()
			c.LoadFromEnv()

			if c.OrderAmount != tt.want {
				t.Errorf("ORDER_AMOUNT=%q gives %d, want %d", tt.value, c.OrderAmount, tt.want)
			}
		})
	}
}

func TestWeatherLoadFromEnv(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "postgres://localhost:5432/oil")
	t.Setenv("LATITUDE", "51.4556")
	t.Setenv("LONGITUDE", "6.7623")
	t.Setenv("VISUAL_CROSSING_API_KEY", "vc-key")
	t.Setenv("OPENWEATHER_API_KEY", "ow-key")

	c := DefaultWeather()
	c.LoadFromEnv()

	// The shared field proves the embedded Common.LoadFromEnv still runs.
	if c.PostgresDSN != "postgres://localhost:5432/oil" {
		t.Errorf("PostgresDSN = %q, embedded Common.LoadFromEnv did not run", c.PostgresDSN)
	}
	if c.Latitude != 51.4556 {
		t.Errorf("Latitude = %v, want 51.4556", c.Latitude)
	}
	if c.Longitude != 6.7623 {
		t.Errorf("Longitude = %v, want 6.7623", c.Longitude)
	}
	if c.VisualCrossingAPIKey != "vc-key" {
		t.Errorf("VisualCrossingAPIKey = %q, want %q", c.VisualCrossingAPIKey, "vc-key")
	}
	if c.OpenWeatherAPIKey != "ow-key" {
		t.Errorf("OpenWeatherAPIKey = %q, want %q", c.OpenWeatherAPIKey, "ow-key")
	}
}

func TestCoordinatesFromEnv(t *testing.T) {
	tests := []struct {
		value string
		want  float64
	}{
		{"51.4556", 51.4556},
		{"-6.7623", -6.7623},
		{"0", 0},
		{"abc", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Setenv("LATITUDE", tt.value)
			t.Setenv("LONGITUDE", tt.value)

			c := DefaultWeather()
			c.LoadFromEnv()

			if c.Latitude != tt.want {
				t.Errorf("LATITUDE=%q gives %v, want %v", tt.value, c.Latitude, tt.want)
			}
			if c.Longitude != tt.want {
				t.Errorf("LONGITUDE=%q gives %v, want %v", tt.value, c.Longitude, tt.want)
			}
		})
	}
}
