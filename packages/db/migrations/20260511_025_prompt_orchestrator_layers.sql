-- 20260511_025_prompt_orchestrator_layers.sql
-- 补充：系统提示词分层编排表（非用户提示词表）
-- 说明：该迁移用于提示词编排层，不承载用户个人画像或用户角色提示词。

CREATE TABLE IF NOT EXISTS prompt_orchestrator_layers (
  id                TEXT PRIMARY KEY,
  layer_key          TEXT NOT NULL UNIQUE,
  layer_name         TEXT NOT NULL,
  scope              TEXT NOT NULL DEFAULT 'system',
  priority           INTEGER NOT NULL,
  enabled            INTEGER NOT NULL DEFAULT 1,
  merge_mode         TEXT NOT NULL DEFAULT 'append',
  content            TEXT NOT NULL DEFAULT '',
  content_hash       TEXT NOT NULL DEFAULT '',
  source             TEXT NOT NULL DEFAULT 'local',
  version            BIGINT NOT NULL DEFAULT 1,
  metadata           TEXT NOT NULL DEFAULT '{}',
  created_at         BIGINT NOT NULL,
  updated_at         BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_prompt_orchestrator_layers_enabled_priority
  ON prompt_orchestrator_layers(enabled, priority);

CREATE INDEX IF NOT EXISTS idx_prompt_orchestrator_layers_scope
  ON prompt_orchestrator_layers(scope);
