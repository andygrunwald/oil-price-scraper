package dwdcdc

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
)

func TestProviderInterface(t *testing.T) {
	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	p := New(logger)

	if p.Name() != "dwdcdc" {
		t.Errorf("expected name 'dwdcdc', got '%s'", p.Name())
	}
	if !p.SupportsBackfill() {
		t.Error("expected SupportsBackfill() = true")
	}
	if p.RequiresAPIKey() {
		t.Error("expected RequiresAPIKey() = false")
	}
}
