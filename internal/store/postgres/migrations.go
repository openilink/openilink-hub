package postgres

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func runMigrations(db *sql.DB) error {
	// Migrate from old schema_version table to goose if needed
	if err := migrateFromLegacy(db); err != nil {
		slog.Warn("legacy migration check", "err", err)
	}

	goose.SetBaseFS(migrationsFS)
	goose.SetLogger(goose.NopLogger())

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose set dialect: %w", err)
	}

	before, _ := goose.GetDBVersion(db)

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	after, _ := goose.GetDBVersion(db)
	if after > before {
		slog.Info("migrations complete", "from", before, "to", after)
	}
	return nil
}

// migrateFromLegacy converts the old schema_version table to goose_db_version.
// Old system used sequential position-based versioning (1, 2, 3...) which maps
// directly to the renumbered goose migration files (0001, 0002, 0003...).
func migrateFromLegacy(db *sql.DB) error {
	// Check if old table exists
	var exists bool
	err := db.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM information_schema.tables
		WHERE table_name = 'schema_version'
	)`).Scan(&exists)
	if err != nil || !exists {
		return nil
	}

	// Check if goose table already exists (already migrated)
	var gooseExists bool
	db.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM information_schema.tables
		WHERE table_name = 'goose_db_version'
	)`).Scan(&gooseExists)
	if gooseExists {
		return nil // already migrated
	}

	// Get current version from old table
	var maxVersion int
	if err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&maxVersion); err != nil {
		return err
	}
	if maxVersion == 0 {
		return nil
	}

	slog.Info("migrating from legacy schema_version to goose", "version", maxVersion)

	// Create goose table and insert version records
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS goose_db_version (
			id SERIAL PRIMARY KEY,
			version_id BIGINT NOT NULL,
			is_applied BOOLEAN NOT NULL DEFAULT TRUE,
			tstamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create goose table: %w", err)
	}

	// Insert initial row (goose expects version 0)
	db.Exec("INSERT INTO goose_db_version (version_id, is_applied) VALUES (0, true)")

	// Mark all existing versions as applied
	for v := 1; v <= maxVersion; v++ {
		if _, err := db.Exec("INSERT INTO goose_db_version (version_id, is_applied) VALUES ($1, true)", v); err != nil {
			return fmt.Errorf("insert goose version %d: %w", v, err)
		}
	}

	slog.Info("legacy migration complete", "versions_migrated", maxVersion)
	return nil
}
