package postgres

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/openilink/openilink-hub/internal/store"
)

// DB implements store.Store for PostgreSQL.
type DB struct {
	*sql.DB
	clock store.Clock
}

// Verify interface compliance at compile time.
var _ store.Store = (*DB)(nil)

// Open connects to PostgreSQL and runs pending migrations.
func Open(dsn string) (*DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, err
	}

	slog.Info("PostgreSQL connected")
	return &DB{DB: db, clock: store.RealClock{}}, nil
}

// SetClock replaces the clock used by time-sensitive queries (e.g. reminders).
func (db *DB) SetClock(c store.Clock) { db.clock = c }

// now returns the current time from the configured clock.
func (db *DB) now() time.Time { return db.clock.Now() }
