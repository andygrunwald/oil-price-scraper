// Package database provides PostgreSQL database operations
// for oil prices and weather observations.
//
// DB, New, Close and Ping hold the shared connection
// handling. The domain-specific queries live in oil.go
// and weather.go.
package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog"
)

// DB wraps the PostgreSQL database connection.
type DB struct {
	db     *sql.DB
	logger zerolog.Logger
}

// New creates a new database connection.
func New(dsn string, logger zerolog.Logger) (*DB, error) {
	db, err := sql.Open("pgx", dsn)
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
