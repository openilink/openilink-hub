-- Backfill installation scopes from app scopes for installations with empty scopes.
-- This implements the Slack model where scopes are locked at install time.
-- New installations will get scopes snapshotted from the app at install time;
-- this migration handles existing installations created before that change.
UPDATE app_installations SET scopes = (
    SELECT apps.scopes FROM apps WHERE apps.id = app_installations.app_id
) WHERE scopes = '[]' OR scopes = '' OR scopes IS NULL;
