// Package scheduler provides a daily scheduler for oil price scraping.
package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/andygrunwald/oil-price-scraper/internal/scraper"
)

// Scheduler manages the daily scraping schedule.
type Scheduler struct {
	scraper    *scraper.Scraper
	scrapeHour int
	logger     zerolog.Logger

	mu           sync.RWMutex
	nextScrapeAt time.Time
	lastScrapeAt *time.Time
	running      bool
}

// New creates a new Scheduler.
func New(s *scraper.Scraper, scrapeHour int, logger zerolog.Logger) *Scheduler {
	return &Scheduler{
		scraper:    s,
		scrapeHour: scrapeHour,
		logger:     logger.With().Str("component", "scheduler").Logger(),
	}
}

// Start starts the scheduler and blocks until the context is cancelled.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	// Run initial scrape if needed
	s.logger.Info().Int("scrapeHour", s.scrapeHour).Msg("starting scheduler")

	// Check if we should scrape immediately (if we haven't scraped today yet)
	s.runIfNeeded(ctx)

	// Calculate time until next scrape
	nextScrape := s.calculateNextScrapeTime()
	s.mu.Lock()
	s.nextScrapeAt = nextScrape
	s.mu.Unlock()

	s.logger.Info().
		Time("nextScrape", nextScrape).
		Dur("duration", time.Until(nextScrape)).
		Msg("next scrape scheduled")

	timer := time.NewTimer(time.Until(nextScrape))
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info().Msg("scheduler stopped")
			return ctx.Err()
		case <-timer.C:
			s.runScrape(ctx)

			// Calculate next scrape time (24 hours from now)
			nextScrape = s.calculateNextScrapeTime()
			s.mu.Lock()
			s.nextScrapeAt = nextScrape
			s.mu.Unlock()

			s.logger.Info().
				Time("nextScrape", nextScrape).
				Msg("next scrape scheduled")

			timer.Reset(time.Until(nextScrape))
		}
	}
}

// calculateNextScrapeTime calculates the next scrape time based on the scrape hour.
func (s *Scheduler) calculateNextScrapeTime() time.Time {
	now := time.Now()

	// Create a time for today at the scrape hour
	nextScrape := time.Date(now.Year(), now.Month(), now.Day(), s.scrapeHour, 0, 0, 0, now.Location())

	// If the scrape time has already passed today, schedule for tomorrow
	if now.After(nextScrape) {
		nextScrape = nextScrape.Add(24 * time.Hour)
	}

	return nextScrape
}

// runIfNeeded checks if scraping is needed and runs it.
func (s *Scheduler) runIfNeeded(ctx context.Context) {
	providers := s.scraper.GetProviders()

	for _, provider := range providers {
		hasScraped, err := s.scraper.HasScrapedToday(ctx, provider.Name())
		if err != nil {
			s.logger.Error().
				Err(err).
				Str("provider", provider.Name()).
				Msg("failed to check if scraped today")
			continue
		}

		if !hasScraped {
			s.logger.Info().
				Str("provider", provider.Name()).
				Msg("no scrape for today, running initial scrape")

			if err := s.scraper.ScrapeProvider(ctx, provider.Name()); err != nil {
				s.logger.Error().
					Err(err).
					Str("provider", provider.Name()).
					Msg("initial scrape failed")
			}
		} else {
			s.logger.Info().
				Str("provider", provider.Name()).
				Msg("already scraped today, skipping initial scrape")
		}
	}
}

// runScrape runs the scraper for all providers.
func (s *Scheduler) runScrape(ctx context.Context) {
	s.logger.Info().Msg("running scheduled scrape")

	now := time.Now()
	s.mu.Lock()
	s.lastScrapeAt = &now
	s.mu.Unlock()

	if err := s.scraper.ScrapeAll(ctx); err != nil {
		s.logger.Error().Err(err).Msg("scheduled scrape failed")
	} else {
		s.logger.Info().Msg("scheduled scrape completed")
	}
}

// NextScrapeAt returns the time of the next scheduled scrape.
func (s *Scheduler) NextScrapeAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nextScrapeAt
}

// LastScrapeAt returns the time of the last scrape.
func (s *Scheduler) LastScrapeAt() *time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastScrapeAt
}

// IsRunning returns whether the scheduler is currently running.
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}
