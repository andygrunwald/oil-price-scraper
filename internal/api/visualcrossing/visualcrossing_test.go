package visualcrossing

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/rs/zerolog"
)

func TestParseResponse(t *testing.T) {
	responseJSON := `{
		"queryCost": 1,
		"latitude": 51.4556,
		"longitude": 6.7623,
		"resolvedAddress": "Duisburg, NRW, Germany",
		"timezone": "Europe/Berlin",
		"days": [
			{
				"datetime": "2024-03-22",
				"tempmax": 15.2,
				"tempmin": 8.1,
				"temp": 11.6,
				"humidity": 65.0,
				"precip": 2.4,
				"windspeed": 25.0,
				"windgust": 40.0,
				"winddir": 240,
				"cloudcover": 75.0,
				"pressure": 1013.0,
				"uvindex": 3.2,
				"sunrise": "06:43:00",
				"sunset": "18:52:00",
				"conditions": "Partly cloudy",
				"description": "Partly cloudy throughout the day"
			},
			{
				"datetime": "2024-03-23",
				"tempmax": 16.5,
				"tempmin": 9.3,
				"temp": 12.9,
				"humidity": 70.0,
				"precip": 0.0,
				"windspeed": 20.0,
				"windgust": 35.0,
				"winddir": 180,
				"cloudcover": 40.0,
				"pressure": 1015.0,
				"uvindex": 4.0,
				"sunrise": "06:41:00",
				"sunset": "18:54:00",
				"conditions": "Clear",
				"description": "Clear conditions throughout the day"
			}
		]
	}`

	var resp apiResponse
	if err := json.Unmarshal([]byte(responseJSON), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	p := New(logger, "test-key")

	results, err := p.parseResponse(resp, 51.4556, 6.7623, []byte(responseJSON))
	if err != nil {
		t.Fatalf("parseResponse error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	r := results[0]
	if r.Date.Format("2006-01-02") != "2024-03-22" {
		t.Errorf("expected date 2024-03-22, got %s", r.Date.Format("2006-01-02"))
	}
	if r.Provider != "visualcrossing" {
		t.Errorf("expected provider visualcrossing, got %s", r.Provider)
	}

	assertFloat64Ptr(t, "TemperatureMaxC", r.TemperatureMaxC, 15.2)
	assertFloat64Ptr(t, "TemperatureMinC", r.TemperatureMinC, 8.1)
	assertFloat64Ptr(t, "TemperatureMeanC", r.TemperatureMeanC, 11.6)
	assertFloat64Ptr(t, "PrecipitationMmSum", r.PrecipitationMmSum, 2.4)
	assertFloat64Ptr(t, "WindSpeedMaxKmh", r.WindSpeedMaxKmh, 25.0)
	assertFloat64Ptr(t, "WindGustMaxKmh", r.WindGustMaxKmh, 40.0)
	assertFloat64Ptr(t, "CloudCoverPercent", r.CloudCoverPercent, 75.0)
	assertFloat64Ptr(t, "HumidityPercent", r.HumidityPercent, 65.0)
	assertFloat64Ptr(t, "PressureHpa", r.PressureHpa, 1013.0)

	// Sunshine duration is not provided by Visual Crossing
	if r.SunshineDurationS != nil {
		t.Errorf("expected SunshineDurationS to be nil")
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
