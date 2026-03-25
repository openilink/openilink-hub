package sqlite

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"sort"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func loadMigrations() ([]string, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	sqls := make([]string, 0, len(entries))
	for _, e := range entries {
		data, err := migrationsFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return nil, err
		}
		sqls = append(sqls, string(data))
	}
	return sqls, nil
}

func runMigrations(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_version (
			version    INTEGER PRIMARY KEY,
			applied_at INTEGER NOT NULL DEFAULT (unixepoch())
		)
	`); err != nil {
		return fmt.Errorf("create schema_version: %w", err)
	}

	sqls, err := loadMigrations()
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}

	var current int
	db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&current)

	for i := current; i < len(sqls); i++ {
		version := i + 1
		slog.Info("running migration", "version", version)
		if _, err := db.Exec(sqls[i]); err != nil {
			return fmt.Errorf("migration %d failed: %w", version, err)
		}
		if _, err := db.Exec("INSERT INTO schema_version (version) VALUES (?)", version); err != nil {
			return fmt.Errorf("record migration %d: %w", version, err)
		}
	}

	if current < len(sqls) {
		slog.Info("migrations complete", "from", current, "to", len(sqls))
	}
	return nil
}
