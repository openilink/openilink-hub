# OpeniLink 本地提示词同步 + Supabase 异步镜像实施计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 将微信聊天提示词依赖下沉到 OpeniLink 本地存储，实现低延迟回复；同时将聊天数据异步镜像到 Supabase，并保持 `admin-worker` 作为绑定关系与提示词快照的权威源。

**架构：** `admin-worker` 在绑定/变更时生成并下发 `full_prompt`；OpeniLink 本地持久化并在对话时仅读取 `full_prompt`。Supabase 同步采用 outbox 异步复制，不阻塞聊天主链路。

**技术栈：** Go 1.25、SQLite/PostgreSQL 双存储、Goose 迁移、现有 OpeniLink bot/AI 流水线、HTTP 事件集成、后台异步 worker。

---

## 范围与非目标

### 范围内
- OpeniLink 本地持久化提示词画像：`system_prompt`、`user_prompt`、`full_prompt`、`prompt_version`、`status`。
- 从 `admin-worker` 接收绑定快照与失效事件。
- AI 回复链路从本地 DB 读取 `full_prompt`。
- 换绑/解绑时失效本地绑定画像。
- 通过 outbox 异步镜像消息与画像变更到 Supabase。
- 增加 trace/metrics 用于排障与回滚。

### 非目标
- 不改变 `admin-worker` 的绑定主权。
- 不做 Supabase -> OpeniLink 的反向回写。
- 不在 OpeniLink 页面侧编辑提示词。
- 不在 OpeniLink 运行时做多层提示词合并。

---

## 高层设计

### 数据主权
- `admin-worker` 负责：
  - 绑定关系真值。
  - 提示词原始数据与 `full_prompt` 生成。
- `openilink-hub` 负责：
  - 本地低延迟读取模型（prompt profile snapshot）。
  - 消息处理与 AI 回复。
  - outbox 事件生产与 Supabase 异步同步。

### 运行时流程
1. 用户微信消息进入 OpeniLink。
2. OpeniLink 本地入库消息。
3. AI 阶段按 `(bot_id, sender_user_id)` 查询 active prompt profile。
4. 命中且 `full_prompt` 非空：直接作为 system prompt 使用。
5. 未命中或 `full_prompt` 为空：回退全局 `ai.system_prompt`。
6. 入站/出站消息写 outbox。
7. worker 批量消费 outbox 并 upsert 到 Supabase。
8. 收到换绑/解绑事件后，将对应 profile 设为 `inactive`。

### 一致性模型
- 聊天主链路：本地强可用，不依赖远程提示词接口。
- Supabase：最终一致，失败可重试。
- 绑定快照：`prompt_version` + `source_updated_at` 防回滚（last-write-wins）。

---

## 完整提示词（full_prompt）策略

### 规则
- 生成时机：仅上游（`admin-worker`）在首次绑定/后续变更时生成 `full_prompt`。
- 使用时机：仅 OpeniLink 对话时读取 `full_prompt`。
- OpeniLink 运行时禁止拼接 `system_prompt + user_prompt`。

### 防回滚
- 仅当 `prompt_version` 更大时允许覆盖。
- 若 `prompt_version` 相同，仅当 `source_updated_at` 更新时覆盖。
- 旧事件丢弃并记 trace。

### 安全与质量保护
- `full_prompt` 长度上限：`AI_FULL_PROMPT_MAX_BYTES`（默认 8192）。
- 全空白 `full_prompt` 视为无效快照。
- 超长策略：默认截断并记录 `prompt.truncated=true`。

---

## 文件结构规划

### 新增领域文件
- Create: `internal/store/prompt_profile.go`
- Create: `internal/store/sync_outbox.go`
- Create: `internal/sync/outbox_worker.go`
- Create: `internal/sync/supabase_client.go`
- Create: `internal/api/admin_binding_sync_handler.go`

### 新增迁移文件
- Create: `internal/store/sqlite/migrations/0009_prompt_profiles_and_outbox.sql`
- Create: `internal/store/postgres/migrations/0038_prompt_profiles_and_outbox.sql`

### 存储实现改造
- Modify: `internal/store/store.go`
- Modify: `internal/store/sqlite/sqlite.go`
- Modify: `internal/store/postgres/postgres.go`
- Create: `internal/store/sqlite/prompt_profile.go`
- Create: `internal/store/postgres/prompt_profile.go`
- Create: `internal/store/sqlite/sync_outbox.go`
- Create: `internal/store/postgres/sync_outbox.go`

### AI 流水线改造
- Modify: `internal/sink/ai.go`（注入本地 `full_prompt` 解析）
- Modify: `internal/ai/chat.go`（仅接收最终 system prompt）

### Bot/Manager 集成
- Modify: `internal/bot/manager.go`（消息保存节点写 outbox）

### API 路由与启动
- Modify: `internal/api/router.go`
- Modify: `main.go`

### 测试
- Create: `internal/store/storetest/prompt_profile_lifecycle.go`
- Create: `internal/store/storetest/sync_outbox.go`
- Modify: `internal/store/sqlite/storetest_test.go`
- Modify: `internal/store/postgres/storetest_test.go`
- Create: `internal/sync/outbox_worker_test.go`
- Create: `internal/api/admin_binding_sync_handler_test.go`
- Modify: `internal/sink/ai_test.go`
- Create: `internal/ai/full_prompt_test.go`

### 文档
- Modify: `README.md`
- Create: `docs/binding-prompt-sync.md`
- Create: `docs/full-prompt-flow.md`

---

## 数据模型

### `prompt_profiles`

字段：
- `id` (PK)
- `bot_id`（索引）
- `sender_user_id`（索引）
- `binding_id`（索引）
- `system_prompt`
- `user_prompt`
- `full_prompt`（运行时唯一读取）
- `full_prompt_hash`（可选，用于观测/排障）
- `prompt_version` (bigint)
- `source_updated_at`
- `status`（`active`/`inactive`）
- `created_at`、`updated_at`

约束：
- `(bot_id, sender_user_id)` 仅允许 1 条 active。
- `(bot_id, sender_user_id, binding_id)` 幂等 upsert。
- 写入时执行版本防回滚规则。

索引：
- `(bot_id, sender_user_id, status)`
- `(binding_id, status)`

### `sync_outbox`

字段：
- `id` (PK)
- `event_id`（唯一）
- `event_type`（`message_inbound` / `message_outbound` / `prompt_profile_changed` / `binding_invalidated`）
- `partition_key`（建议 `bot_id`）
- `payload`（json/jsonb）
- `status`（`pending`/`processing`/`sent`/`dead`）
- `retry_count`
- `next_retry_at`
- `last_error`
- `created_at`、`updated_at`、`sent_at`

索引：
- `(status, next_retry_at, id)`
- `(partition_key, id)`

---

## 与 `admin-worker` 的事件契约

### 入站事件：`binding_profile_snapshot`
字段：
- `event_id`
- `event_time`
- `bot_id`
- `sender_user_id`
- `binding_id`
- `system_prompt`
- `user_prompt`
- `full_prompt`
- `prompt_version`
- `source_updated_at`
- `gateway_status`（可选）

处理规则：
- 仅当版本更新（或同版本但更新时间更新）才覆盖本地。
- 状态置为 `active`。

### 入站事件：`binding_invalidated`
字段：
- `event_id`
- `event_time`
- `bot_id`
- `sender_user_id`
- `binding_id`
- `reason`（`rebind`/`unbind`/`admin_revoke`）

处理规则：
- 对应 profile 置为 `inactive`。

### 安全
- 使用共享密钥签名（HMAC）或 mTLS（二选一）。
- 基于 `event_id` 做重放去重。

---

## 提示词解析规则

AI 回复时：
1. 查询 `(bot_id, sender_user_id)` 的 active profile。
2. 命中且 `full_prompt` 非空：直接使用。
3. 否则回退全局 `ai.system_prompt`。
4. trace 记录：
   - `prompt.source=local_full_prompt|global_fallback`
   - `prompt.version=<n>`
   - `prompt.full_hash=<sha256_prefix>`
   - `prompt.truncated=true|false`

---

## Supabase 异步镜像设计

### 生产端（Producer）
- 入/出站消息保存成功后写 outbox。
- 理想情况同事务写消息与 outbox；如暂时无法同事务，先 best-effort + 指标告警。

### 消费端（Consumer）
- 固定间隔轮询 outbox。
- 批量领取 `pending 且 next_retry_at <= now`。
- 推送 Supabase 后：
  - 成功：`sent`
  - 暂时失败：指数退避 + `retry_count+1`
  - 超阈值：`dead`

### Upsert 主键建议
- 消息：`bot_id + direction + message_id_or_client_id`
- 提示词画像：`bot_id + sender_user_id + binding_id`

---

## 分任务执行清单

### Task 1: Store 合约与模型
**Files:**
- Create: `internal/store/prompt_profile.go`
- Create: `internal/store/sync_outbox.go`
- Modify: `internal/store/store.go`

- [ ] **Step 1: 写编译失败的接口约束检查**
- [ ] **Step 2: 定义模型与最小接口**
- [ ] **Step 3: 扩展总 `Store` 接口**
- [ ] **Step 4: 运行 `go test ./internal/store/...` 验证失败（预期）**
- [ ] **Step 5: Commit**

### Task 2: DB 迁移（SQLite + Postgres）
**Files:**
- Create: `internal/store/sqlite/migrations/0009_prompt_profiles_and_outbox.sql`
- Create: `internal/store/postgres/migrations/0038_prompt_profiles_and_outbox.sql`

- [ ] **Step 1: 写 DDL（表/约束/索引）**
- [ ] **Step 2: 校验迁移版本连续与 goose 兼容**
- [ ] **Step 3: 跑 store 测试验证迁移可启动**
- [ ] **Step 4: Commit**

### Task 3: Prompt Profile 存储实现
**Files:**
- Create: `internal/store/sqlite/prompt_profile.go`
- Create: `internal/store/postgres/prompt_profile.go`
- Test: `internal/store/storetest/prompt_profile_lifecycle.go`
- Modify: `internal/store/sqlite/storetest_test.go`
- Modify: `internal/store/postgres/storetest_test.go`

- [ ] **Step 1: 先写 storetest（upsert/版本保护/active 查询/invalidate）**
- [ ] **Step 2: 跑测试确认失败**
- [ ] **Step 3: 实现 SQLite**
- [ ] **Step 4: 实现 Postgres**
- [ ] **Step 5: 跑测试通过**
- [ ] **Step 6: Commit**

### Task 4: Outbox 存储实现
**Files:**
- Create: `internal/store/sqlite/sync_outbox.go`
- Create: `internal/store/postgres/sync_outbox.go`
- Test: `internal/store/storetest/sync_outbox.go`
- Modify: `internal/store/sqlite/storetest_test.go`
- Modify: `internal/store/postgres/storetest_test.go`

- [ ] **Step 1: 先写 storetest（enqueue/claim/sent/retry/dead）**
- [ ] **Step 2: 实现双库 SQL**
- [ ] **Step 3: 验证重试状态迁移**
- [ ] **Step 4: Commit**

### Task 5: Admin 同步 API（绑定/提示词事件）
**Files:**
- Create: `internal/api/admin_binding_sync_handler.go`
- Modify: `internal/api/router.go`
- Test: `internal/api/admin_binding_sync_handler_test.go`

- [ ] **Step 1: 定义 payload + 签名校验**
- [ ] **Step 2: 实现 `event_id` 幂等处理**
- [ ] **Step 3: 实现 snapshot upsert 与 invalidation**
- [ ] **Step 4: 注册内部路由**
- [ ] **Step 5: 增加 replay/乱序/失效用例**
- [ ] **Step 6: Commit**

### Task 6: AI 接入本地 full_prompt
**Files:**
- Modify: `internal/sink/ai.go`
- Modify: `internal/ai/chat.go`
- Modify: `internal/sink/ai_test.go`

- [ ] **Step 1: 在 AI sink 注入 prompt resolver**
- [ ] **Step 2: 在 BuildMessages 前查询本地 profile**
- [ ] **Step 3: 仅注入 `full_prompt`（禁止运行时合并）**
- [ ] **Step 4: 增加 trace 属性**
- [ ] **Step 5: 覆盖命中/未命中/inactive 用例**
- [ ] **Step 6: Commit**

### Task 7: 消息链路写 Outbox
**Files:**
- Modify: `internal/bot/manager.go`
- Optional Create: `internal/sync/event_builder.go`

- [ ] **Step 1: 在入/出站保存节点挂 outbox enqueue**
- [ ] **Step 2: 统一事件 payload 结构**
- [ ] **Step 3: enqueue 异常不影响回复主链路**
- [ ] **Step 4: 补充关键路径测试**
- [ ] **Step 5: Commit**

### Task 8: Supabase Outbox Worker
**Files:**
- Create: `internal/sync/supabase_client.go`
- Create: `internal/sync/outbox_worker.go`
- Test: `internal/sync/outbox_worker_test.go`
- Modify: `main.go`

- [ ] **Step 1: Supabase 客户端抽象（URL/Key/表映射）**
- [ ] **Step 2: 实现批处理消费与 ack/retry**
- [ ] **Step 3: `main.go` 接入生命周期（启动/优雅停止）**
- [ ] **Step 4: 补充暂时失败/永久失败测试**
- [ ] **Step 5: Commit**

### Task 9: 配置与文档
**Files:**
- Modify: `README.md`
- Create: `docs/binding-prompt-sync.md`
- Create: `docs/full-prompt-flow.md`

- [ ] **Step 1: 文档化 admin-worker 事件契约**
- [ ] **Step 2: 文档化 Supabase 配置与重试参数**
- [ ] **Step 3: 文档化 full_prompt 下发、覆盖、失效规则**
- [ ] **Step 4: Commit**

### Task 10: 端到端验证
**Files:**
- 基于现有 `integration_test.go` 或新增聚焦集成测试

- [ ] **Step 1: 模拟 snapshot -> 收消息 -> 验证 full_prompt 生效**
- [ ] **Step 2: 模拟换绑/解绑 -> 验证 profile 失效**
- [ ] **Step 3: 验证 outbox 生成与 Supabase mock 同步**
- [ ] **Step 4: 验证 Supabase 故障不影响回复延迟**
- [ ] **Step 5: Commit**

---

## 配置项

新增配置：
- `ADMIN_SYNC_SHARED_SECRET`
- `ADMIN_SYNC_ALLOWLIST`（可选）
- `SUPABASE_URL`
- `SUPABASE_SERVICE_ROLE_KEY`
- `SUPABASE_SCHEMA`（可选）
- `OUTBOX_BATCH_SIZE`（默认 100）
- `OUTBOX_POLL_INTERVAL_MS`（默认 500）
- `OUTBOX_MAX_RETRIES`（默认 10）
- `AI_FULL_PROMPT_MAX_BYTES`（默认 8192）

约定：
- Supabase 配置缺失时，仅禁用 outbox worker；聊天功能正常。
- `full_prompt` 缺失时，降级到全局 `ai.system_prompt`。

---

## 发布策略

1. 先发版 schema + 代码（worker 默认关闭）。
2. 打开 admin sync API，验证本地 prompt 快照可用。
3. 打开 AI 本地 `full_prompt` 解析，观察 trace。
4. 小流量 bot 灰度开启。
5. 灰度打开 outbox worker。
6. 监控 dead-letter、重试、同步延迟、full_prompt 命中率。
7. 全量开启。

---

## 可观测性与 SLO

### 指标
- `prompt_profile_lookup_hit_ratio`
- `prompt_profile_lookup_latency_ms`
- `prompt_full_hit_count`
- `prompt_full_truncated_count`
- `outbox_pending_count`
- `outbox_dead_count`
- `outbox_retry_count`
- `supabase_mirror_lag_seconds`
- `chat_reply_latency_p95/p99`

### Trace 属性
- `prompt.source`
- `prompt.version`
- `prompt.full_hash`
- `prompt.truncated`
- `binding.id`
- `outbox.event_id`

### 告警
- `outbox_dead_count` 超阈值
- `supabase_mirror_lag_seconds` 持续偏高
- `prompt_profile_lookup_hit_ratio` 突降
- `prompt_full_truncated_count` 异常升高

---

## 风险与缓解

- 风险：snapshot 与 invalidation 事件乱序。
- 缓解：版本/时间戳保护 + 幂等处理。

- 风险：Supabase 故障导致 outbox 堆积。
- 缓解：指数退避 + dead-letter + 监控。

- 风险：提示词敏感信息泄漏。
- 缓解：接口签名、日志脱敏、必要时本地加密存储。

- 风险：上游下发空/脏 `full_prompt` 导致回复质量下降。
- 缓解：长度/空值校验 + 全局提示词兜底 + trace 告警。

---

## 验收标准

- AI 回复链路不再按消息调用远程提示词接口。
- 已绑定用户提示词可本地命中，换绑/解绑后及时失效。
- 对话阶段仅使用本地 `full_prompt`（无运行时合并）。
- 超长/空 `full_prompt` 触发保护且可观测。
- Supabase 同步异步化，Supabase 故障时回复延迟不显著劣化。
- outbox 重试与 dead-letter 行为可预测、可观测。
- SQLite/PostgreSQL 双存储迁移与测试通过。

---

## 建议提交顺序

1. `feat(store): add prompt profile and outbox store interfaces`
2. `feat(db): add prompt profile and outbox migrations`
3. `feat(store): implement prompt profile persistence for sqlite/postgres`
4. `feat(store): implement outbox persistence for sqlite/postgres`
5. `feat(api): add admin binding/profile sync endpoint`
6. `feat(ai): use local full_prompt only for runtime system prompt`
7. `feat(bot): enqueue mirror events to outbox`
8. `feat(sync): add supabase outbox worker`
9. `docs: add binding prompt sync and full prompt flow runbook`
10. `test: add e2e coverage for prompt invalidation and full_prompt fallback`
