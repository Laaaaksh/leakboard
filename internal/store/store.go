// Package store persists Leakboard's data in Postgres: users and sessions,
// connected GitHub organizations, tracked repos, scan runs, findings, the
// shared allowlist, and outbound webhooks.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Store wraps a database connection pool with Leakboard's query methods.
type Store struct {
	db *sql.DB
}

// Open connects to Postgres at databaseURL and applies any pending
// migrations before returning.
func Open(ctx context.Context, databaseURL string) (*Store, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetConnMaxLifetime(time.Hour)

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if err := Migrate(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &Store{db: db}, nil
}

// Close releases the underlying connection pool.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB exposes the underlying pool for callers (tests, admin tooling) that
// need direct access.
func (s *Store) DB() *sql.DB {
	return s.db
}
