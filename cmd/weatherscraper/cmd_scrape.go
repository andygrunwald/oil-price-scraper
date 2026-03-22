package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/andygrunwald/oil-price-scraper/internal/database"
	"github.com/andygrunwald/oil-price-scraper/internal/weatherapi/brightsky"
	"github.com/andygrunwald/oil-price-scraper/internal/weatherapi/dwdcdc"
	"github.com/andygrunwald/oil-price-scraper/internal/weatherapi/openmeteo"
	"github.com/andygrunwald/oil-price-scraper/internal/weatherapi/openweather"
	"github.com/andygrunwald/oil-price-scraper/internal/weatherapi/visualcrossing"
	"github.com/andygrunwald/oil-price-scraper/internal/weatherscraper"
)

func scrapeCmd() *cobra.Command {
	var providers string
	var visualCrossingAPIKey string
	var openWeatherAPIKey string

	cmd := &cobra.Command{
		Use:   "scrape",
		Short: "Run a one-time weather scrape",
		Long:  "Runs a one-time scrape from the specified providers. Useful for testing.",
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
				Strs("providers", providerList).
				Float64("latitude", cfg.Latitude).
				Float64("longitude", cfg.Longitude).
				Msg("running one-time weather scrape")

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
				case "dwdcdc":
					s.RegisterProvider(dwdcdc.New(logger))
				default:
					logger.Warn().Str("provider", p).Msg("unknown provider, skipping")
				}
			}

			// Run scrape
			ctx := context.Background()
			if err := s.ScrapeAll(ctx); err != nil {
				return fmt.Errorf("scraping: %w", err)
			}

			logger.Info().Msg("scrape completed")
			return nil
		},
	}

	cmd.Flags().StringVar(&providers, "providers", "openmeteo", "Comma-separated list of providers")
	cmd.Flags().StringVar(&visualCrossingAPIKey, "visual-crossing-api-key", "", "Visual Crossing API key")
	cmd.Flags().StringVar(&openWeatherAPIKey, "openweather-api-key", "", "OpenWeather API key")

	return cmd
}
