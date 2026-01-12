// Package main provides the entry point for the oil price scraper CLI.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/andygrunwald/oil-price-scraper/internal/api/heizoel24"
	"github.com/andygrunwald/oil-price-scraper/internal/api/hoyer"
	"github.com/andygrunwald/oil-price-scraper/internal/config"
	"github.com/andygrunwald/oil-price-scraper/internal/database"
	"github.com/andygrunwald/oil-price-scraper/internal/http"
	"github.com/andygrunwald/oil-price-scraper/internal/scheduler"
	"github.com/andygrunwald/oil-price-scraper/internal/scraper"
)

var (
	// Version is set at build time.
	Version = "dev"
	// Commit is set at build time.
	Commit = "none"
	// BuildDate is set at build time.
	BuildDate = "unknown"
)

var cfg *config.Config

func main() {
	cfg = config.DefaultConfig()
	cfg.LoadFromEnv()

	rootCmd := &cobra.Command{
		Use:   "oilscraper",
		Short: "Oil Price Scraper - Never miss a dip in heating oil prices again",
		Long: `Oil Price Scraper is a service that scrapes heating oil prices from various
German APIs and stores them in a MySQL database for analysis and monitoring.

Features:
  - Multiple API providers (HeizOel24, Hoyer)
  - Daily automated scraping with configurable schedule
  - Historical data backfilling
  - Prometheus metrics endpoint
  - Status endpoint for operational visibility`,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfg.MySQLDSN, "mysql-dsn", cfg.MySQLDSN, "MySQL connection string")
	rootCmd.PersistentFlags().StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&cfg.LogFormat, "log-format", cfg.LogFormat, "Log format (json, console)")
	rootCmd.PersistentFlags().BoolVar(&cfg.StoreRawResponse, "store-raw-response", cfg.StoreRawResponse, "Store raw API responses in database")
	rootCmd.PersistentFlags().StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP server address for /metrics, /status")
	rootCmd.PersistentFlags().StringVar(&cfg.ZipCode, "zip-code", cfg.ZipCode, "Zip code for local price APIs")
	rootCmd.PersistentFlags().IntVar(&cfg.OrderAmount, "order-amount", cfg.OrderAmount, "Order amount in liters")

	// Add subcommands
	rootCmd.AddCommand(runCmd())
	rootCmd.AddCommand(scrapeCmd())
	rootCmd.AddCommand(backfillCmd())
	rootCmd.AddCommand(versionCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func setupLogger() zerolog.Logger {
	var logger zerolog.Logger

	// Set log level
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Set log format
	if cfg.LogFormat == "console" {
		logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
			With().
			Timestamp().
			Logger()
	} else {
		logger = zerolog.New(os.Stdout).
			With().
			Timestamp().
			Logger()
	}

	return logger
}

func runCmd() *cobra.Command {
	var scrapeHour int
	var providers string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start the continuous scraper service",
		Long:  "Starts the oil price scraper with an internal scheduler that runs daily at the specified hour.",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := setupLogger()

			if cfg.MySQLDSN == "" {
				return fmt.Errorf("--mysql-dsn is required")
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
				Msg("starting oil price scraper")

			// Connect to database
			db, err := database.New(cfg.MySQLDSN, logger)
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

			// Create scheduler
			sched := scheduler.New(s, scrapeHour, logger)

			// Create HTTP server
			httpServer := http.NewServer(cfg.HTTPAddr, s, sched, db, logger)

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

	cmd.Flags().IntVar(&scrapeHour, "scrape-hour", 6, "Hour of day (0-23) to scrape")
	cmd.Flags().StringVar(&providers, "providers", "heizoel24,hoyer", "Comma-separated list of providers")

	return cmd
}

func scrapeCmd() *cobra.Command {
	var providers string

	cmd := &cobra.Command{
		Use:   "scrape",
		Short: "Run a one-time scrape",
		Long:  "Runs a one-time scrape from the specified providers. Useful for testing.",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := setupLogger()

			if cfg.MySQLDSN == "" {
				return fmt.Errorf("--mysql-dsn is required")
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
			db, err := database.New(cfg.MySQLDSN, logger)
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

			if cfg.MySQLDSN == "" {
				return fmt.Errorf("--mysql-dsn is required")
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
			db, err := database.New(cfg.MySQLDSN, logger)
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

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Oil Price Scraper\n")
			fmt.Printf("  Version:    %s\n", Version)
			fmt.Printf("  Commit:     %s\n", Commit)
			fmt.Printf("  Build Date: %s\n", BuildDate)
		},
	}
}
