// Package database provides MySQL database operations for the oil price scraper.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog"

	"github.com/andygrunwald/oil-price-scraper/internal/models"
)

// DB wraps the MySQL database connection and provides operations for oil prices.
type DB struct {
	db     *sql.DB
	logger zerolog.Logger
}

// New creates a new database connection.
func New(dsn string, logger zerolog.Logger) (*DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &DB{
		db:     db,
		logger: logger.With().Str("component", "database").Logger(),
	}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// Ping checks if the database connection is alive.
func (d *DB) Ping() error {
	return d.db.Ping()
}

// InsertPrice inserts a new oil price record into the database.
func (d *DB) InsertPrice(ctx context.Context, price models.PriceResult, storeRawResponse bool) error {
	query := `
		INSERT INTO oil_prices (provider, product_type, price_date, price_per_100l, currency, scope, zip_code, raw_response, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			price_per_100l = VALUES(price_per_100l),
			raw_response = VALUES(raw_response),
			fetched_at = VALUES(fetched_at)
	`

	var rawResponse []byte
	if storeRawResponse {
		rawResponse = price.RawResponse
	}

	var zipCode *string
	if price.ZipCode != "" {
		zipCode = &price.ZipCode
	}

	_, err := d.db.ExecContext(ctx, query,
		price.Provider,
		price.ProductType,
		price.Date.Format("2006-01-02"),
		price.PricePer100L,
		price.Currency,
		string(price.Scope),
		zipCode,
		rawResponse,
		price.FetchedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting price: %w", err)
	}

	d.logger.Debug().
		Str("provider", price.Provider).
		Str("product_type", price.ProductType).
		Str("date", price.Date.Format("2006-01-02")).
		Float64("price", price.PricePer100L).
		Msg("inserted price record")

	return nil
}

// ExistsForDate checks if a price record exists for the given provider, product type, date, and zip code.
func (d *DB) ExistsForDate(ctx context.Context, provider, productType string, date time.Time, zipCode string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM oil_prices
		WHERE provider = ? AND product_type = ? AND price_date = ?
		AND (zip_code = ? OR (zip_code IS NULL AND ? IS NULL))
	`

	var zipCodePtr *string
	if zipCode != "" {
		zipCodePtr = &zipCode
	}

	var count int
	err := d.db.QueryRowContext(ctx, query,
		provider,
		productType,
		date.Format("2006-01-02"),
		zipCodePtr,
		zipCodePtr,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking existence: %w", err)
	}

	return count > 0, nil
}

// GetTotalPricesCount returns the total number of price records in the database.
func (d *DB) GetTotalPricesCount(ctx context.Context) (int64, error) {
	var count int64
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM oil_prices").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting prices: %w", err)
	}
	return count, nil
}

// GetPricesCountByProvider returns the number of price records for a specific provider.
func (d *DB) GetPricesCountByProvider(ctx context.Context, provider string) (int64, error) {
	var count int64
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM oil_prices WHERE provider = ?", provider).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting prices for provider: %w", err)
	}
	return count, nil
}

// GetLatestPrice returns the most recent price for a provider.
func (d *DB) GetLatestPrice(ctx context.Context, provider string) (*models.OilPrice, error) {
	query := `
		SELECT id, provider, product_type, price_date, price_per_100l, currency, scope, zip_code, raw_response, fetched_at, created_at
		FROM oil_prices
		WHERE provider = ?
		ORDER BY price_date DESC, created_at DESC
		LIMIT 1
	`

	var price models.OilPrice
	var zipCode sql.NullString
	var rawResponse []byte
	var scope string

	err := d.db.QueryRowContext(ctx, query, provider).Scan(
		&price.ID,
		&price.Provider,
		&price.ProductType,
		&price.PriceDate,
		&price.PricePer100L,
		&price.Currency,
		&scope,
		&zipCode,
		&rawResponse,
		&price.FetchedAt,
		&price.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting latest price: %w", err)
	}

	price.Scope = models.PriceScope(scope)
	if zipCode.Valid {
		price.ZipCode = &zipCode.String
	}
	price.RawResponse = rawResponse

	return &price, nil
}

// GetPricesForDateRange returns all prices for a provider within a date range.
func (d *DB) GetPricesForDateRange(ctx context.Context, provider string, from, to time.Time) ([]models.OilPrice, error) {
	query := `
		SELECT id, provider, product_type, price_date, price_per_100l, currency, scope, zip_code, raw_response, fetched_at, created_at
		FROM oil_prices
		WHERE provider = ? AND price_date >= ? AND price_date <= ?
		ORDER BY price_date ASC
	`

	rows, err := d.db.QueryContext(ctx, query, provider, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("querying prices: %w", err)
	}
	defer rows.Close()

	var prices []models.OilPrice
	for rows.Next() {
		var price models.OilPrice
		var zipCode sql.NullString
		var rawResponse []byte
		var scope string

		err := rows.Scan(
			&price.ID,
			&price.Provider,
			&price.ProductType,
			&price.PriceDate,
			&price.PricePer100L,
			&price.Currency,
			&scope,
			&zipCode,
			&rawResponse,
			&price.FetchedAt,
			&price.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}

		price.Scope = models.PriceScope(scope)
		if zipCode.Valid {
			price.ZipCode = &zipCode.String
		}
		price.RawResponse = rawResponse
		prices = append(prices, price)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	return prices, nil
}
