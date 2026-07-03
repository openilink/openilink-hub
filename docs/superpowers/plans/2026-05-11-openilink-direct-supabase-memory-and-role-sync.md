# OpeniLink Direct Supabase Memory & Role Sync Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在不破坏现有 admin/应用/Webhook 能力的前提下，让 `openilink-hub` 直接完成微信 AI 回复，并将角色提示词与记忆统一接入 Supabase。  

**Architecture:** `openilink-hub` 保留现有本地消息链路与 outbox，同步新增一个运行时 Supabase 客户端：对话前拉取角色提示词与向量记忆，对话后异步写入消息记忆。首次扫码绑定成功后，立即从 Supabase 拉取角色提示词快照写入本地 `prompt_profiles`，确保首条消息即带角色人设。  

**Tech Stack:** Go 1.25、OpenAI-compatible Chat API、Supabase REST/RPC、现有 SQLite/Postgres 存储层。

---

## 1. 需求总览（确认版）

- 保留现有 admin 功能（用户、应用、Webhook、配置、市场等）不回退。
- 聊天回复由 `openilink-hub` 直接执行，不依赖中转 worker 才能回复。
- 首次扫码绑定成功后：按 `(bot_id, sender_user_id)` 从 Supabase 拉取角色提示词并写入本地 prompt profile。
- 日常聊天：
  - 对话前从 Supabase 召回历史记忆（优先向量检索，失败降级关键词/最近消息）。
  - 对话后异步写入用户消息与助手回复到 Supabase 记忆表。
- Supabase 不可用时不阻塞主回复链路：AI 仍可按本地/全局 prompt 回复。

## 2. 数据与职责边界

- `openilink-hub` 本地：仍负责消息收发、会话上下文、频道分发、admin 配置、prompt profile 本地缓存。
- Supabase：
  - 角色提示词权威源（读取）
  - 记忆持久化与向量召回（读取 + 写入）
- 迁移策略：先“增强直连能力”，不删除现有 outbox/sync worker。

## 3. 本轮实现范围

- 新增运行时 Supabase 客户端：
  - 拉取绑定提示词快照（用于首次绑定 bootstrap）
  - 拉取 active 提示词（用于运行时 fallback）
  - 向量召回记忆（RPC `match_memories`）与降级召回
  - 写入记忆记录（inbound/outbound）
- AI sink 接入：
  - `resolveRuntimePrompt` 支持本地优先 + Supabase fallback
  - 在系统提示词中注入记忆片段
  - 回复完成后异步写入 Supabase 记忆
- 绑定链路接入：
  - 扫码绑定成功后触发 prompt bootstrap（best effort）
- 配置扩展：
  - 记忆开关、TopK、表名/RPC 名、embedding 模型等可配置项

## 4. 非目标（本轮不做）

- 不重构现有 outbox 消息镜像表结构。
- 不在本轮新增 Supabase SQL migration（假设目标环境已有 `bl_memories` / `match_memories` / 角色相关表）。
- 不改前端页面交互。

## 5. 验收标准

- AI 正常回复：在 Supabase 不可用情况下依然回复成功。
- 首次绑定后本地 `prompt_profiles` 可看到已同步的 `full_prompt`。
- 聊天后可观测到 Supabase 记忆写入请求触发。
- 当召回命中时，AI 链路 trace 有 `memory.*` 属性。

## 6. 风险与待确认

- Supabase 实际表/RPC 名称在不同环境可能有差异：通过配置项可覆盖。
- `sender_user_id` 与 `bl_tool_bindings.external_chat_id` 是否稳定一致：若不一致需补映射表。
- 嵌入模型兼容性：默认 `text-embedding-3-small`，可通过环境变量调整。
