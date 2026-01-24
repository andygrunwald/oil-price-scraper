package http

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"

	"github.com/andygrunwald/oil-price-scraper/internal/database"
	"github.com/andygrunwald/oil-price-scraper/internal/scheduler"
	"github.com/andygrunwald/oil-price-scraper/internal/scraper"
)

// Server represents the HTTP server for metrics and status endpoints.
type Server struct {
	server  *http.Server
	logger  zerolog.Logger
	metrics *Metrics
}

// NewServer creates a new HTTP server.
func NewServer(addr string, s *scraper.Scraper, sched *scheduler.Scheduler, db *database.DB, logger zerolog.Logger) *Server {
	mux := http.NewServeMux()
	metrics := NewMetrics()

	// Register handlers
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/status", NewStatusHandler(s, sched, db))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			panic(err)
		}
	})

	return &Server{
		server: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		logger:  logger.With().Str("component", "http").Logger(),
		metrics: metrics,
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.logger.Info().Str("addr", s.server.Addr).Msg("starting HTTP server")
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("shutting down HTTP server")
	return s.server.Shutdown(ctx)
}

// Metrics returns the Prometheus metrics.
func (s *Server) Metrics() *Metrics {
	return s.metrics
}
