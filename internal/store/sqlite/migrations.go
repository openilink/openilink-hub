package sqlite

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func runMigrations(db *sql.DB) error {
	// Migrate from old schema_version table to goose if needed
	if err := migrateFromLegacy(db); err != nil {
		return fmt.Errorf("legacy migration failed: %w", err)
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

	// Run idempotent post-migration fixups.
	// SQLite doesn't support IF NOT EXISTS for ALTER TABLE ADD COLUMN,
	// so we apply column additions here with error suppression.
	applyColumnIfMissing(db, "bots", "display_name", "TEXT NOT NULL DEFAULT ''")

	after, _ := goose.GetDBVersion(db)
	if after > before {
		slog.Info("migrations complete", "from", before, "to", after)
	}
	return nil
}

// applyColumnIfMissing adds a column to a table, ignoring "duplicate column" errors.
func applyColumnIfMissing(db *sql.DB, table, column, definition string) {
	_, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
	if err != nil && !strings.Contains(err.Error(), "duplicate column") {
		slog.Warn("column migration failed", "table", table, "column", column, "err", err)
	}
}

// migrateFromLegacy converts the old schema_version table to goose_db_version.
func migrateFromLegacy(db *sql.DB) error {
	var exists int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_version'").Scan(&exists); err != nil || exists == 0 {
		return nil
	}

	var gooseExists int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='goose_db_version'").Scan(&gooseExists); err != nil || gooseExists > 0 {
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

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS goose_db_version (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			version_id INTEGER NOT NULL,
			is_applied INTEGER NOT NULL DEFAULT 1,
			tstamp TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`); err != nil {
		return fmt.Errorf("create goose table: %w", err)
	}

	if _, err := tx.Exec("INSERT INTO goose_db_version (version_id, is_applied) VALUES (0, 1)"); err != nil {
		return fmt.Errorf("insert goose version 0: %w", err)
	}
	for v := 1; v <= maxVersion; v++ {
		if _, err := tx.Exec("INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, 1)", v); err != nil {
			return fmt.Errorf("insert goose version %d: %w", v, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit legacy migration: %w", err)
	}

	slog.Info("legacy migration complete", "versions_migrated", maxVersion)
	return nil
}
