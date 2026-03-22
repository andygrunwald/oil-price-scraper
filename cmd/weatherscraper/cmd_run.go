package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/andygrunwald/oil-price-scraper/internal/database"
	"github.com/andygrunwald/oil-price-scraper/internal/scheduler"
	"github.com/andygrunwald/oil-price-scraper/internal/weatherapi/brightsky"
	"github.com/andygrunwald/oil-price-scraper/internal/weatherapi/openmeteo"
	"github.com/andygrunwald/oil-price-scraper/internal/weatherapi/openweather"
	"github.com/andygrunwald/oil-price-scraper/internal/weatherapi/visualcrossing"
	"github.com/andygrunwald/oil-price-scraper/internal/weatherhttp"
	"github.com/andygrunwald/oil-price-scraper/internal/weatherscraper"
)

func runCmd() *cobra.Command {
	var scrapeHour int
	var providers string
	var visualCrossingAPIKey string
	var openWeatherAPIKey string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start the continuous weather scraper service",
		Long:  "Starts the weather scraper with an internal scheduler that runs daily at the specified hour.",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := setupLogger()

			if cfg.PostgresDSN == "" {
				return fmt.Errorf("--postgres-dsn is required")
			}

			if cfg.Latitude == 0 && cfg.Longitude == 0 {
				return fmt.Errorf("--latitude and --longitude are required")
			}

			// Override config from flags
			if visualCrossingAPIKey != "" {
				cfg.VisualCrossingAPIKey = visualCrossingAPIKey
			}
			if openWeatherAPIKey != "" {
				cfg.OpenWeatherAPIKey = openWeatherAPIKey
			}

			// Parse providers
			providerList := strings.Split(providers, ",")
			for i := range providerList {
				providerList[i] = strings.TrimSpace(providerList[i])
			}

			logger.Info().
				Str("version", Version).
				Str("commit", Commit).
				Str("buildDate", BuildDate).
				Str("httpAddr", cfg.HTTPAddr).
				Int("scrapeHour", scrapeHour).
				Strs("providers", providerList).
				Float64("latitude", cfg.Latitude).
				Float64("longitude", cfg.Longitude).
				Msg("starting weather scraper")

			// Connect to database
			db, err := database.New(cfg.PostgresDSN, logger)
			if err != nil {
				return fmt.Errorf("connecting to database: %w", err)
			}
			defer func() {
				if err := db.Close(); err != nil {
					panic(err)
				}
			}()

			// Create scraper
			s := weatherscraper.New(db, cfg.StoreRawResponse, cfg.Latitude, cfg.Longitude, logger)

			// Register providers
			for _, p := range providerList {
				switch p {
				case "openmeteo":
					s.RegisterProvider(openmeteo.New(logger))
				case "brightsky":
					s.RegisterProvider(brightsky.New(logger))
				case "visualcrossing":
					if cfg.VisualCrossingAPIKey == "" {
						logger.Warn().Msg("Visual Crossing API key not set, skipping provider")
						continue
					}
					s.RegisterProvider(visualcrossing.New(logger, cfg.VisualCrossingAPIKey))
				case "openweather":
					if cfg.OpenWeatherAPIKey == "" {
						logger.Warn().Msg("OpenWeather API key not set, skipping provider")
						continue
					}
					s.RegisterProvider(openweather.New(logger, cfg.OpenWeatherAPIKey, 1, 5))
				default:
					logger.Warn().Str("provider", p).Msg("unknown provider, skipping")
				}
			}

			// Create scheduler
			sched := scheduler.New(s, scrapeHour, logger)

			// Create HTTP server
			httpServer := weatherhttp.NewServer(cfg.HTTPAddr, s, sched, db, logger)

			// Wire Prometheus metrics to scraper
			s.SetPrometheusMetrics(httpServer.Metrics())

			// Setup signal handling
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			// Start HTTP server in goroutine
			go func() {
				if err := httpServer.Start(); err != nil {
					logger.Error().Err(err).Msg("HTTP server error")
					cancel()
				}
			}()

			// Start scheduler in goroutine
			go func() {
				if err := sched.Start(ctx); err != nil && err != context.Canceled {
					logger.Error().Err(err).Msg("scheduler error")
					cancel()
				}
			}()

			// Wait for signal
			select {
			case sig := <-sigCh:
				logger.Info().Str("signal", sig.String()).Msg("received signal, shutting down")
			case <-ctx.Done():
			}

			// Graceful shutdown
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer shutdownCancel()

			if err := httpServer.Shutdown(shutdownCtx); err != nil {
				logger.Error().Err(err).Msg("HTTP server shutdown error")
			}

			logger.Info().Msg("shutdown complete")
			return nil
		},
	}

	cmd.Flags().IntVar(&scrapeHour, "scrape-hour", 7, "Hour of day (0-23) to scrape")
	cmd.Flags().StringVar(&providers, "providers", "openmeteo", "Comma-separated list of providers")
	cmd.Flags().StringVar(&visualCrossingAPIKey, "visual-crossing-api-key", "", "Visual Crossing API key")
	cmd.Flags().StringVar(&openWeatherAPIKey, "openweather-api-key", "", "OpenWeather API key")

	return cmd
}
