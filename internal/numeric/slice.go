// Package numeric provides shared numeric helpers.
package numeric

import "math"

// SliceMin returns the smallest value in s.
// It returns math.MaxFloat64 if s is empty, so callers must guard with len(s) > 0.
func SliceMin(s []float64) float64 {
	m := math.MaxFloat64
	for _, v := range s {
		if v < m {
			m = v
		}
	}
	return m
}

// SliceMax returns the largest value in s.
// It returns -math.MaxFloat64 if s is empty, so callers must guard with len(s) > 0.
func SliceMax(s []float64) float64 {
	m := -math.MaxFloat64
	for _, v := range s {
		if v > m {
			m = v
		}
	}
	return m
}

// SliceMean returns the arithmetic mean of s.
// It returns NaN if s is empty, so callers must guard with len(s) > 0.
func SliceMean(s []float64) float64 {
	return SliceSum(s) / float64(len(s))
}

// SliceSum returns the sum of all values in s.
// It returns 0 if s is empty.
func SliceSum(s []float64) float64 {
	var sum float64
	for _, v := range s {
		sum += v
	}
	return sum
}
