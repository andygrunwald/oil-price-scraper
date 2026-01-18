package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/andygrunwald/oil-price-scraper/internal/api/heizoel24"
	"github.com/andygrunwald/oil-price-scraper/internal/api/hoyer"
	"github.com/andygrunwald/oil-price-scraper/internal/database"
	"github.com/andygrunwald/oil-price-scraper/internal/scraper"
)

func backfillCmd() *cobra.Command {
	var fromStr, toStr string
	var provider string
	var minDelay, maxDelay int

	cmd := &cobra.Command{
		Use:   "backfill",
		Short: "Backfill historical data",
		Long:  "Backfills historical data from APIs that support it (e.g., HeizOel24).",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := setupLogger()

			if cfg.PostgresDSN == "" {
				return fmt.Errorf("--postgres-dsn is required")
			}

			if fromStr == "" {
				return fmt.Errorf("--from is required")
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
				Msg("starting backfill")

			// Connect to database
			db, err := database.New(cfg.PostgresDSN, logger)
			if err != nil {
				return fmt.Errorf("connecting to database: %w", err)
			}
			defer db.Close()

			// Create scraper
			s := scraper.New(db, cfg.StoreRawResponse, logger)

			// Register provider
			switch provider {
			case "heizoel24":
				s.RegisterProvider(heizoel24.New(logger))
			case "hoyer":
				s.RegisterProvider(hoyer.New(logger, cfg.ZipCode, cfg.OrderAmount))
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
	cmd.Flags().StringVar(&provider, "provider", "heizoel24", "Provider to backfill from")
	cmd.Flags().IntVar(&minDelay, "min-delay", 1, "Minimum delay between requests (seconds)")
	cmd.Flags().IntVar(&maxDelay, "max-delay", 5, "Maximum delay between requests (seconds)")

	return cmd
}
