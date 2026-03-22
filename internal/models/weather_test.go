package models

import (
	"math"
	"testing"
)

func TestRoundCoord(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{51.455600, 51.4556},
		{6.762300, 6.7623},
		{51.45561234, 51.4556},
		{-33.86785, -33.8679},
		{0.0, 0.0},
		{180.0, 180.0},
		{-180.0, -180.0},
		{51.45555, 51.4556}, // rounds up
		{51.45554, 51.4555}, // rounds down
	}

	for _, tt := range tests {
		got := RoundCoord(tt.input)
		if math.Abs(got-tt.expected) > 0.00001 {
			t.Errorf("RoundCoord(%f) = %f, want %f", tt.input, got, tt.expected)
		}
	}
}

func TestFloat64Ptr(t *testing.T) {
	v := 42.5
	ptr := Float64Ptr(v)
	if ptr == nil {
		t.Fatal("Float64Ptr returned nil")
	}
	if *ptr != v {
		t.Errorf("expected %f, got %f", v, *ptr)
	}
}
