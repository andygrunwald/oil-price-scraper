package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/andygrunwald/oil-price-scraper/internal/database"
	"github.com/andygrunwald/oil-price-scraper/internal/models"
	"github.com/andygrunwald/oil-price-scraper/internal/scheduler"
	"github.com/andygrunwald/oil-price-scraper/internal/scraper"
)

// StatusHandler handles the /status endpoint.
type StatusHandler struct {
	scraper   *scraper.Scraper
	scheduler *scheduler.Scheduler
	db        *database.DB
	startTime time.Time
}

// NewStatusHandler creates a new StatusHandler.
func NewStatusHandler(s *scraper.Scraper, sched *scheduler.Scheduler, db *database.DB) *StatusHandler {
	return &StatusHandler{
		scraper:   s,
		scheduler: sched,
		db:        db,
		startTime: time.Now(),
	}
}

// ServeHTTP implements the http.Handler interface.
func (h *StatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	response := models.StatusResponse{
		Status:        "healthy",
		UptimeSeconds: int64(time.Since(h.startTime).Seconds()),
		Providers:     make(map[string]models.ProviderStatus),
	}

	// Get scheduler status
	if h.scheduler != nil {
		response.SchedulerRunning = h.scheduler.IsRunning()
		response.LastScheduledScrapeAt = h.scheduler.LastScrapeAt()
		nextScrape := h.scheduler.NextScrapeAt()
		if !nextScrape.IsZero() {
			response.NextScrapeAt = &nextScrape
		}
	}

	// Get provider statuses
	for _, provider := range h.scraper.GetProviders() {
		metrics := h.scraper.GetMetrics(provider.Name())
		if metrics == nil {
			continue
		}

		snapshot := metrics.GetSnapshot()
		providerStatus := models.ProviderStatus{
			Enabled:            true,
			LastScrapeAt:       snapshot.LastScrapeAt,
			LastScrapeSuccess:  snapshot.LastScrapeSuccess,
			LastResponseTimeMs: snapshot.LastResponseTime.Milliseconds(),
			LastPrice:          snapshot.LastPrice,
			LastError:          snapshot.LastError,
			TotalRequests:      snapshot.TotalRequests,
			TotalErrors:        snapshot.TotalErrors,
			LastRawResponse:    snapshot.LastRawResponse,
		}

		response.Providers[provider.Name()] = providerStatus
	}

	// Get database status
	response.Database = h.getDatabaseStatus(ctx)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *StatusHandler) getDatabaseStatus(ctx context.Context) models.DatabaseStatus {
	status := models.DatabaseStatus{
		Connected: false,
	}

	if h.db == nil {
		return status
	}

	// Check database connection
	if err := h.db.Ping(); err != nil {
		return status
	}
	status.Connected = true

	// Get total prices count
	count, err := h.db.GetTotalPricesCount(ctx)
	if err == nil {
		status.TotalPricesStored = count
	}

	return status
}
