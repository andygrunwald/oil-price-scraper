package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/andygrunwald/oil-price-scraper/internal/api/heizoel24"
	"github.com/andygrunwald/oil-price-scraper/internal/api/hoyer"
	"github.com/andygrunwald/oil-price-scraper/internal/database"
	"github.com/andygrunwald/oil-price-scraper/internal/scraper"
)

func scrapeCmd() *cobra.Command {
	var providers string

	cmd := &cobra.Command{
		Use:   "scrape",
		Short: "Run a one-time scrape",
		Long:  "Runs a one-time scrape from the specified providers. Useful for testing.",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := setupLogger()

			if cfg.PostgresDSN == "" {
				return fmt.Errorf("--postgres-dsn is required")
			}

			if cfg.ZipCode == "" {
				return fmt.Errorf("--zip-code is required")
			}

			// Parse providers
			providerList := strings.Split(providers, ",")
			for i := range providerList {
				providerList[i] = strings.TrimSpace(providerList[i])
			}

			logger.Info().
				Strs("providers", providerList).
				Msg("running one-time scrape")

			// Connect to database
			db, err := database.New(cfg.PostgresDSN, logger)
			if err != nil {
				return fmt.Errorf("connecting to database: %w", err)
			}
			defer db.Close()

			// Create scraper
			s := scraper.New(db, cfg.StoreRawResponse, logger)

			// Register providers
			for _, p := range providerList {
				switch p {
				case "heizoel24":
					s.RegisterProvider(heizoel24.New(logger))
				case "hoyer":
					s.RegisterProvider(hoyer.New(logger, cfg.ZipCode, cfg.OrderAmount))
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

	cmd.Flags().StringVar(&providers, "providers", "heizoel24,hoyer", "Comma-separated list of providers")

	return cmd
}
