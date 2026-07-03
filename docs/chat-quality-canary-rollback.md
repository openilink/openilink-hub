# Chat Quality Canary & Rollback Runbook

[PROTOCOL]: 变更时更新此头部，然后检查 CLAUDE.md

## 目标

为对话质量链路（planner-lite + guard + emotion/tone + summary）提供可执行的灰度与回滚操作手册。

## 灰度步骤

1. 5% 流量：观察 20-30 分钟。  
2. 20% 流量：观察 20-30 分钟。  
3. 50% 流量：观察 20-30 分钟。  
4. 100% 全量。

## 关键观测指标

1. `chat.error_rate`  
2. `chat.p95_ms`  
3. `dialogue.fallback_rate`  
4. `dialogue.guard_block_rate`  
5. `emoji_suppressed_by_emotion_policy`（审计字段）

## Stop-the-line 条件

1. `chat.p95_ms` 连续 10 分钟 > 基线 1.3x  
2. `dialogue.fallback_rate` 连续 10 分钟 > 基线 + 8%  
3. `dialogue.guard_block_rate` 连续 10 分钟 > 20%

## 开关与降级路径

1. `planner_only=true`：仅保留结构化计划回复。  
2. `guard_soft_mode=true`：守卫仅审计不拦截。  
3. `fallback_fast_path=true`：快速降级为保守回复路径。

## 回滚流程

1. 立即开启 `guard_soft_mode=true`，观察 5 分钟。  
2. 若仍异常，开启 `fallback_fast_path=true`。  
3. 若仍异常，开启 `planner_only=true` 并冻结放量。  
4. 导出最近 1 小时 `bl_dialogue_events` 与 `bl_platform_audit_logs` 样本复盘。

## 发布前检查

1. 必须通过 `TestChatQualityGate50Cases`。  
2. 必须通过 `go test ./internal/supamemory ./internal/sink`。  
3. 灰度变更记录需包含阈值、窗口、结论。

