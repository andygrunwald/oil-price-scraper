package numeric

import (
	"math"
	"testing"
)

func TestSliceMin(t *testing.T) {
	tests := []struct {
		input    []float64
		expected float64
	}{
		{[]float64{5, 10, 15}, 5},
		{[]float64{15, 10, 5}, 5},
		{[]float64{-3.5, 2, 7}, -3.5},
		{[]float64{42}, 42},
		{[]float64{}, math.MaxFloat64}, // empty slice, callers must guard with len(s) > 0
	}

	for _, tt := range tests {
		got := SliceMin(tt.input)
		if math.Abs(got-tt.expected) > 0.00001 {
			t.Errorf("SliceMin(%v) = %f, want %f", tt.input, got, tt.expected)
		}
	}
}

func TestSliceMax(t *testing.T) {
	tests := []struct {
		input    []float64
		expected float64
	}{
		{[]float64{5, 10, 15}, 15},
		{[]float64{15, 10, 5}, 15},
		{[]float64{-3.5, -2, -7}, -2},
		{[]float64{42}, 42},
		{[]float64{}, -math.MaxFloat64}, // empty slice, callers must guard with len(s) > 0
	}

	for _, tt := range tests {
		got := SliceMax(tt.input)
		if math.Abs(got-tt.expected) > 0.00001 {
			t.Errorf("SliceMax(%v) = %f, want %f", tt.input, got, tt.expected)
		}
	}
}

func TestSliceMean(t *testing.T) {
	tests := []struct {
		input    []float64
		expected float64
	}{
		{[]float64{5, 10, 15}, 10},
		{[]float64{1, 2}, 1.5},
		{[]float64{-4, 4}, 0},
		{[]float64{42}, 42},
	}

	for _, tt := range tests {
		got := SliceMean(tt.input)
		if math.Abs(got-tt.expected) > 0.00001 {
			t.Errorf("SliceMean(%v) = %f, want %f", tt.input, got, tt.expected)
		}
	}

	// An empty slice divides by zero, callers must guard with len(s) > 0.
	if got := SliceMean([]float64{}); !math.IsNaN(got) {
		t.Errorf("SliceMean([]) = %f, want NaN", got)
	}
}

func TestSliceSum(t *testing.T) {
	tests := []struct {
		input    []float64
		expected float64
	}{
		{[]float64{1, 2, 3}, 6},
		{[]float64{0.5, 1.25, 1.75}, 3.5},
		{[]float64{-4, 4}, 0},
		{[]float64{42}, 42},
		{[]float64{}, 0},
	}

	for _, tt := range tests {
		got := SliceSum(tt.input)
		if math.Abs(got-tt.expected) > 0.00001 {
			t.Errorf("SliceSum(%v) = %f, want %f", tt.input, got, tt.expected)
		}
	}
}
