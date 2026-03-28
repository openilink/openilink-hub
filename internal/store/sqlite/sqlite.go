package sqlite

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/openilink/openilink-hub/internal/store"
	_ "modernc.org/sqlite"
)

// DB implements store.Store for SQLite.
type DB struct {
	*sql.DB
	clock store.Clock
}

// Verify interface compliance at compile time.
var _ store.Store = (*DB)(nil)

// Open connects to SQLite and runs pending migrations.
func Open(dsn string) (*DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)

	// Enable WAL mode, busy timeout, and foreign keys.
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("%s: %w", pragma, err)
		}
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, err
	}

	slog.Info("SQLite connected")
	return &DB{DB: db, clock: store.RealClock{}}, nil
}

// SetClock replaces the clock used by time-sensitive queries (e.g. reminders).
// Intended for testing with a fake clock.
func (db *DB) SetClock(c store.Clock) { db.clock = c }

// now returns the current unix epoch from the configured clock.
func (db *DB) now() int64 { return db.clock.Now().Unix() }
