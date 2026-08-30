package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/andygrunwald/oil-price-scraper/internal/database"
	"github.com/andygrunwald/oil-price-scraper/internal/weatherapi/brightsky"
	"github.com/andygrunwald/oil-price-scraper/internal/weatherapi/dwdcdc"
	"github.com/andygrunwald/oil-price-scraper/internal/weatherapi/openmeteo"
	"github.com/andygrunwald/oil-price-scraper/internal/weatherapi/openweather"
	"github.com/andygrunwald/oil-price-scraper/internal/weatherapi/visualcrossing"
	"github.com/andygrunwald/oil-price-scraper/internal/weatherscraper"
)

func backfillCmd() *cobra.Command {
	var fromStr, toStr string
	var provider string
	var minDelay, maxDelay int
	var visualCrossingAPIKey string
	var openWeatherAPIKey string

	cmd := &cobra.Command{
		Use:   "backfill",
		Short: "Backfill historical weather data",
		Long:  "Backfills historical weather data from providers that support it.",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := setupLogger()

			if cfg.PostgresDSN == "" {
				return fmt.Errorf("--postgres-dsn is required")
			}

			if cfg.Latitude == 0 && cfg.Longitude == 0 {
				return fmt.Errorf("--latitude and --longitude are required")
			}

			if fromStr == "" {
				return fmt.Errorf("--from is required")
			}

			// Override config from flags
			if visualCrossingAPIKey != "" {
				cfg.VisualCrossingAPIKey = visualCrossingAPIKey
			}
			if openWeatherAPIKey != "" {
				cfg.OpenWeatherAPIKey = openWeatherAPIKey
			}

			from, err := time.Parse("2006-01-02", fromStr)
			if err != nil {
				return fmt.Errorf("parsing --from date: %w", err)
			}

			to := time.Now()
			if toStr != "" {
				to, err = time.Parse("2006-01-02", toStr)
				if err != nil {
					return fmt.Errorf("parsing --to date: %w", err)
				}
			}

			logger.Info().
				Str("provider", provider).
				Str("from", from.Format("2006-01-02")).
				Str("to", to.Format("2006-01-02")).
				Int("minDelay", minDelay).
				Int("maxDelay", maxDelay).
				Float64("latitude", cfg.Latitude).
				Float64("longitude", cfg.Longitude).
				Msg("starting weather backfill")

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

			// Register provider
			switch provider {
			case "openmeteo":
				s.RegisterProvider(openmeteo.New(logger))
			case "brightsky":
				s.RegisterProvider(brightsky.New(logger))
			case "visualcrossing":
				if cfg.VisualCrossingAPIKey == "" {
					return fmt.Errorf("--visual-crossing-api-key is required for Visual Crossing provider")
				}
				s.RegisterProvider(visualcrossing.New(logger, cfg.VisualCrossingAPIKey))
			case "openweather":
				if cfg.OpenWeatherAPIKey == "" {
					return fmt.Errorf("--openweather-api-key is required for OpenWeather provider")
				}
				s.RegisterProvider(openweather.New(logger, cfg.OpenWeatherAPIKey, minDelay, maxDelay))
			case "dwdcdc":
				s.RegisterProvider(dwdcdc.New(logger))
			default:
				return fmt.Errorf("unknown provider: %s", provider)
			}

			// Run backfill
			ctx := context.Background()
			if err := s.Backfill(ctx, provider, from, to, minDelay, maxDelay); err != nil {
				return fmt.Errorf("backfilling: %w", err)
			}

			logger.Info().Msg("backfill completed")
			return nil
		},
	}

	cmd.Flags().StringVar(&fromStr, "from", "", "Start date (YYYY-MM-DD, required)")
	cmd.Flags().StringVar(&toStr, "to", "", "End date (YYYY-MM-DD, defaults to today)")
	cmd.Flags().StringVar(&provider, "provider", "openmeteo", "Provider to backfill from")
	cmd.Flags().IntVar(&minDelay, "min-delay", 1, "Minimum delay between requests (seconds)")
	cmd.Flags().IntVar(&maxDelay, "max-delay", 5, "Maximum delay between requests (seconds)")
	cmd.Flags().StringVar(&visualCrossingAPIKey, "visual-crossing-api-key", "", "Visual Crossing API key")
	cmd.Flags().StringVar(&openWeatherAPIKey, "openweather-api-key", "", "OpenWeather API key")

	return cmd
}
