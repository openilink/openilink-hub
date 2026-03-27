package sqlite

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

	if err := goose.SetDialect("sqlite3"); err != nil {
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
func migrateFromLegacy(db *sql.DB) error {
	var exists int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_version'").Scan(&exists)
	if err != nil || exists == 0 {
		return nil
	}

	// Check if goose table already exists
	var gooseExists int
	db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='goose_db_version'").Scan(&gooseExists)
	if gooseExists > 0 {
		return nil
	}

	var maxVersion int
	if err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&maxVersion); err != nil {
		return err
	}
	if maxVersion == 0 {
		return nil
	}

	slog.Info("migrating from legacy schema_version to goose", "version", maxVersion)

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS goose_db_version (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			version_id INTEGER NOT NULL,
			is_applied INTEGER NOT NULL DEFAULT 1,
			tstamp TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`); err != nil {
		return fmt.Errorf("create goose table: %w", err)
	}

	db.Exec("INSERT INTO goose_db_version (version_id, is_applied) VALUES (0, 1)")

	for v := 1; v <= maxVersion; v++ {
		if _, err := db.Exec("INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, 1)", v); err != nil {
			return fmt.Errorf("insert goose version %d: %w", v, err)
		}
	}

	slog.Info("legacy migration complete", "versions_migrated", maxVersion)
	return nil
}
