package postgres_test

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/openilink/openilink-hub/internal/store/postgres"
	"github.com/openilink/openilink-hub/internal/store/storetest"
)

func testPGStore(t *testing.T) *postgres.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping PG store tests")
	}
	// Drop all tables for clean migration run
	preDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skipf("skip: %v", err)
	}
	preDB.Exec(`DO $$ DECLARE r RECORD;
		BEGIN FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public')
		LOOP EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE'; END LOOP; END $$`)
	preDB.Close()

	db, err := postgres.Open(dsn)
	if err != nil {
		t.Skipf("skip: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Truncate all tables (faster than DELETE, resets sequences)
	db.Exec(`TRUNCATE
		app_api_logs, app_event_logs, app_oauth_codes, app_installations,
		apps, trace_spans, webhook_logs, messages, channels, bots,
		plugin_installs, plugin_versions, plugins,
		oauth_accounts, sessions, credentials, users, system_config
		RESTART IDENTITY CASCADE`)
	return db
}

func TestPostgresStore(t *testing.T) {
	db := testPGStore(t)
	storetest.RunAll(t, db)
}
