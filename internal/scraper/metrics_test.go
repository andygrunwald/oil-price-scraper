package scraper

import (
	"sync"
	"testing"
	"time"
)

// TestGetSnapshotCopiesEveryField guards the split of Metrics into an embedded
// commonMetrics plus a headline measurement: a field left out of
// commonMetrics.snapshot would silently read as its zero value on /status.
func TestGetSnapshotCopiesEveryField(t *testing.T) {
	at := time.Date(2026, 8, 30, 6, 0, 0, 0, time.UTC)
	errMsg := "boom"
	price := 99.5

	m := &Metrics{}
	m.TotalRequests = 7
	m.TotalErrors = 2
	m.LastScrapeAt = &at
	m.LastScrapeSuccess = true
	m.LastResponseTime = 250 * time.Millisecond
	m.LastError = &errMsg
	m.LastRawResponse = "{}"
	m.LastPrice = &price

	got := m.GetSnapshot()

	if got.TotalRequests != 7 || got.TotalErrors != 2 {
		t.Errorf("counters = %d/%d, want 7/2", got.TotalRequests, got.TotalErrors)
	}
	if got.LastScrapeAt != &at || !got.LastScrapeSuccess {
		t.Errorf("LastScrapeAt/Success = %v/%v", got.LastScrapeAt, got.LastScrapeSuccess)
	}
	if got.LastResponseTime != 250*time.Millisecond {
		t.Errorf("LastResponseTime = %v, want 250ms", got.LastResponseTime)
	}
	if got.LastError != &errMsg || got.LastRawResponse != "{}" {
		t.Errorf("LastError/LastRawResponse = %v/%q", got.LastError, got.LastRawResponse)
	}
	if got.LastPrice != &price {
		t.Errorf("LastPrice = %v, want %v", got.LastPrice, &price)
	}
}

// TestWeatherGetSnapshotCopiesEveryField is the weather half of the above.
func TestWeatherGetSnapshotCopiesEveryField(t *testing.T) {
	at := time.Date(2026, 8, 30, 7, 0, 0, 0, time.UTC)
	errMsg := "boom"
	temp := 21.5

	m := &WeatherMetrics{}
	m.TotalRequests = 7
	m.TotalErrors = 2
	m.LastScrapeAt = &at
	m.LastScrapeSuccess = true
	m.LastResponseTime = 250 * time.Millisecond
	m.LastError = &errMsg
	m.LastRawResponse = "{}"
	m.LastTemperature = &temp

	got := m.GetSnapshot()

	if got.TotalRequests != 7 || got.TotalErrors != 2 {
		t.Errorf("counters = %d/%d, want 7/2", got.TotalRequests, got.TotalErrors)
	}
	if got.LastScrapeAt != &at || !got.LastScrapeSuccess {
		t.Errorf("LastScrapeAt/Success = %v/%v", got.LastScrapeAt, got.LastScrapeSuccess)
	}
	if got.LastResponseTime != 250*time.Millisecond {
		t.Errorf("LastResponseTime = %v, want 250ms", got.LastResponseTime)
	}
	if got.LastError != &errMsg || got.LastRawResponse != "{}" {
		t.Errorf("LastError/LastRawResponse = %v/%q", got.LastError, got.LastRawResponse)
	}
	if got.LastTemperature != &temp {
		t.Errorf("LastTemperature = %v, want %v", got.LastTemperature, &temp)
	}
}

// TestGetSnapshotUnderConcurrentWrites is the reason commonMetrics.snapshot
// does not take the lock itself: GetSnapshot has to read the embedded fields
// and the headline measurement under one RLock, and sync.RWMutex is not
// reentrant. Run with -race, this fails if either half escapes the lock.
func TestGetSnapshotUnderConcurrentWrites(t *testing.T) {
	m := &Metrics{}
	w := &WeatherMetrics{}

	var writers, readers sync.WaitGroup
	stop := make(chan struct{})

	// Writers take the lock exactly the way ScrapeProvider does.
	writers.Add(2)
	go func() {
		defer writers.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			price := float64(i)
			now := time.Now()
			m.mu.Lock()
			m.TotalRequests++
			m.LastScrapeAt = &now
			m.LastPrice = &price
			m.mu.Unlock()
		}
	}()
	go func() {
		defer writers.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			temp := float64(i)
			now := time.Now()
			w.mu.Lock()
			w.TotalRequests++
			w.LastScrapeAt = &now
			w.LastTemperature = &temp
			w.mu.Unlock()
		}
	}()

	readers.Add(2)
	for range 2 {
		go func() {
			defer readers.Done()
			for range 2000 {
				_ = m.GetSnapshot()
				_ = w.GetSnapshot()
			}
		}()
	}

	readers.Wait()
	close(stop)
	writers.Wait()
}
