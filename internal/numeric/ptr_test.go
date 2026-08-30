package numeric

import "testing"

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
