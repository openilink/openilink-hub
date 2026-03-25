package postgres

import (
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/openilink/openilink-hub/internal/store"
)

// DB implements store.Store for PostgreSQL.
type DB struct {
	*sql.DB
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
	return &DB{db}, nil
}
