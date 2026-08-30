package openmeteo

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/rs/zerolog"
)

func TestParseResponse(t *testing.T) {
	responseJSON := `{
		"latitude": 51.44,
		"longitude": 6.76,
		"timezone": "Europe/Berlin",
		"daily": {
			"time": ["2024-03-22", "2024-03-23"],
			"temperature_2m_max": [15.2, 16.5],
			"temperature_2m_min": [8.1, 9.3],
			"temperature_2m_mean": [11.6, 12.9],
			"precipitation_sum": [0.0, 2.4],
			"wind_speed_10m_max": [25.0, 20.5],
			"wind_gusts_10m_max": [40.0, 35.0],
			"sunshine_duration": [28800.0, 14400.0],
			"cloud_cover_mean": [45.0, 75.0],
			"relative_humidity_2m_mean": [65.0, 80.0],
			"surface_pressure_mean": [1013.0, 1010.5]
		},
		"daily_units": {
			"temperature_2m_max": "°C",
			"precipitation_sum": "mm"
		}
	}`

	var resp apiResponse
	if err := json.Unmarshal([]byte(responseJSON), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	p := New(logger)

	results, err := p.parseResponse(resp, 51.4556, 6.7623, []byte(responseJSON))
	if err != nil {
		t.Fatalf("parseResponse error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Check first day
	r := results[0]
	if r.Date.Format("2006-01-02") != "2024-03-22" {
		t.Errorf("expected date 2024-03-22, got %s", r.Date.Format("2006-01-02"))
	}
	if r.Provider != "openmeteo" {
		t.Errorf("expected provider openmeteo, got %s", r.Provider)
	}
	assertFloat64Ptr(t, "TemperatureMaxC", r.TemperatureMaxC, 15.2)
	assertFloat64Ptr(t, "TemperatureMinC", r.TemperatureMinC, 8.1)
	assertFloat64Ptr(t, "TemperatureMeanC", r.TemperatureMeanC, 11.6)
	assertFloat64Ptr(t, "PrecipitationMmSum", r.PrecipitationMmSum, 0.0)
	assertFloat64Ptr(t, "WindSpeedMaxKmh", r.WindSpeedMaxKmh, 25.0)
	assertFloat64Ptr(t, "WindGustMaxKmh", r.WindGustMaxKmh, 40.0)
	assertFloat64Ptr(t, "SunshineDurationS", r.SunshineDurationS, 28800.0)
	assertFloat64Ptr(t, "CloudCoverPercent", r.CloudCoverPercent, 45.0)
	assertFloat64Ptr(t, "HumidityPercent", r.HumidityPercent, 65.0)
	assertFloat64Ptr(t, "PressureHpa", r.PressureHpa, 1013.0)

	// Check second day
	r2 := results[1]
	assertFloat64Ptr(t, "PrecipitationMmSum day2", r2.PrecipitationMmSum, 2.4)
}

func TestParseResponseWithNulls(t *testing.T) {
	responseJSON := `{
		"latitude": 51.44,
		"longitude": 6.76,
		"timezone": "Europe/Berlin",
		"daily": {
			"time": ["2024-03-22"],
			"temperature_2m_max": [15.2],
			"temperature_2m_min": [null],
			"temperature_2m_mean": [11.6],
			"precipitation_sum": [null],
			"wind_speed_10m_max": [25.0],
			"wind_gusts_10m_max": [null],
			"sunshine_duration": [28800.0],
			"cloud_cover_mean": [45.0],
			"relative_humidity_2m_mean": [65.0],
			"surface_pressure_mean": [1013.0]
		}
	}`

	var resp apiResponse
	if err := json.Unmarshal([]byte(responseJSON), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	p := New(logger)

	results, err := p.parseResponse(resp, 51.4556, 6.7623, []byte(responseJSON))
	if err != nil {
		t.Fatalf("parseResponse error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	assertFloat64Ptr(t, "TemperatureMaxC", r.TemperatureMaxC, 15.2)
	if r.TemperatureMinC != nil {
		t.Errorf("expected TemperatureMinC to be nil, got %f", *r.TemperatureMinC)
	}
	if r.PrecipitationMmSum != nil {
		t.Errorf("expected PrecipitationMmSum to be nil, got %f", *r.PrecipitationMmSum)
	}
}

func assertFloat64Ptr(t *testing.T, name string, got *float64, want float64) {
	t.Helper()
	if got == nil {
		t.Errorf("%s: expected %f, got nil", name, want)
		return
	}
	if math.Abs(*got-want) > 0.01 {
		t.Errorf("%s: expected %f, got %f", name, want, *got)
	}
}
