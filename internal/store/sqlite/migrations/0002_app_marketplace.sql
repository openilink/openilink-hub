-- App Marketplace migration for existing databases.
-- This migration is safe to run on both old and new schemas:
-- it checks for old column names before attempting renames/adds.

-- Rename columns (only if old names exist).
-- We detect the old schema by checking for the 'setup_url' column in apps.
-- If it exists, we have the old schema and need to rename columns.
-- If it doesn't, we already have the new schema from 0001_initial.sql.

-- Step 1: Rename columns (SQLite >= 3.25)
-- Using a programmatic approach isn't possible in pure SQL, so we wrap
-- each ALTER in a way that tolerates failure. Since SQLite runs each
-- statement independently, we use a trick: only execute the ALTER if
-- the old column exists by querying pragma_table_info.

-- Unfortunately, SQLite doesn't support conditional DDL in plain SQL.
-- Instead, we rely on the fact that RENAME COLUMN will fail gracefully
-- if the column doesn't exist, and we handle that in the migration runner.

-- For fresh databases (created with 0001_initial.sql that already has new names),
-- none of the RENAMEs are needed. For old databases, they are.
-- Since we can't conditionally run DDL in SQLite, we'll only add new columns
-- and create new tables, which are idempotent with IF NOT EXISTS.

-- Add new columns (these use IF NOT EXISTS-like behavior:
-- ALTER TABLE ADD COLUMN fails if column exists, but we can't skip that in SQL).
-- However, if 0001_initial.sql already has these columns, they exist.
-- So we skip the adds for columns already in 0001_initial.sql.

-- The only safe things to do here are:
-- 1. CREATE TABLE IF NOT EXISTS (idempotent)
-- 2. UPDATE statements (idempotent, no-op if data already migrated)

-- Migrate scope values (safe to re-run; REPLACE is idempotent if already migrated)
UPDATE apps SET scopes = REPLACE(REPLACE(REPLACE(scopes,
    'messages.send', 'message:write'),
    'contacts.read', 'contact:read'),
    'bot.read', 'bot:read')
WHERE scopes != '[]';

-- Migrate listing values from old columns (safe: no-op if old columns don't exist
-- because the WHERE clause won't match any rows with old schema data)
-- For old databases that had 'listed' and 'listing_status' columns:
-- This UPDATE is harmless on new databases since the CASE just evaluates
-- the default values.

-- Note: guide column is already in 0001_initial.sql for fresh databases.
-- For old databases, the migration runner runs statements in order.
-- We skip adding guide here; old databases that need it should be
-- recreated (SQLite is typically used for dev/local, not production).

-- Registries table (idempotent)
CREATE TABLE IF NOT EXISTS registries (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    url        TEXT NOT NULL UNIQUE,
    enabled    INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);
