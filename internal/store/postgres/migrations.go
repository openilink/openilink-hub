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
		return fmt.Errorf("legacy migration failed: %w", err)
	}

	// Advisory lock to prevent concurrent migration runs (multiple app instances)
	if _, err := db.Exec("SELECT pg_advisory_lock(1)"); err != nil {
		return fmt.Errorf("advisory lock: %w", err)
	}
	defer db.Exec("SELECT pg_advisory_unlock(1)")

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
// Wrapped in a transaction to ensure atomicity — partial migration would leave
// goose in a broken state on next startup.
func migrateFromLegacy(db *sql.DB) error {
	// Check if old table exists
	var exists bool
	if err := db.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM information_schema.tables
		WHERE table_name = 'schema_version'
	)`).Scan(&exists); err != nil || !exists {
		return nil
	}

	// Check if goose table already exists (already migrated)
	var gooseExists bool
	if err := db.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM information_schema.tables
		WHERE table_name = 'goose_db_version'
	)`).Scan(&gooseExists); err != nil || gooseExists {
		return nil
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

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS goose_db_version (
			id SERIAL PRIMARY KEY,
			version_id BIGINT NOT NULL,
			is_applied BOOLEAN NOT NULL DEFAULT TRUE,
			tstamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create goose table: %w", err)
	}

	// Insert initial row (goose expects version 0) + all existing versions
	if _, err := tx.Exec("INSERT INTO goose_db_version (version_id, is_applied) VALUES (0, true)"); err != nil {
		return fmt.Errorf("insert goose version 0: %w", err)
	}
	for v := 1; v <= maxVersion; v++ {
		if _, err := tx.Exec("INSERT INTO goose_db_version (version_id, is_applied) VALUES ($1, true)", v); err != nil {
			return fmt.Errorf("insert goose version %d: %w", v, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit legacy migration: %w", err)
	}

	slog.Info("legacy migration complete", "versions_migrated", maxVersion)
	return nil
}
