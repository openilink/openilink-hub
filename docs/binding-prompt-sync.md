# Binding Prompt Sync (admin-worker -> OpeniLink)

## Endpoint

- Method: `POST`
- Path: `/api/internal/admin/sync/binding`
- Auth: `X-Admin-Signature` HMAC-SHA256 (`sha256=<hex>` or raw hex)
- Secret source: `ADMIN_SYNC_SHARED_SECRET`

## Envelope

```json
{
  "type": "binding_prebind | binding_profile_snapshot | binding_invalidated",
  "data": {"...": "..."}
}
```

## Event: `binding_prebind`

Required fields:

- `event_id`
- `binding_id`
- `role_id`
- `bot_id` (provider bot id)
- `session_id`

Behavior:

1. `event_id` is persisted in `admin_sync_inbox` for idempotency.
2. Resolve local bot by `FindBotByProviderID("ilink", bot_id)`.
3. Create local pending row in `wechat_pending_bindings` (`pending_finalize`, 10 min TTL).
4. First inbound message triggers callback to Admin finalize endpoint (configured by env).

## Event: `binding_profile_snapshot`

Required fields:

- `event_id`
- `bot_id`
- `sender_user_id`
- `binding_id`
- `full_prompt`
- `prompt_version`
- `source_updated_at`

Behavior:

1. `event_id` is persisted in `admin_sync_inbox` for idempotency.
2. `full_prompt` is truncated by `AI_FULL_PROMPT_MAX_BYTES` (default `8192`).
3. Blank prompt is rejected.
4. Upsert is accepted only when:
   - newer `prompt_version`, or
   - same `prompt_version` with newer `source_updated_at`.
5. Active profile is switched to the new binding snapshot.
6. Emits outbox event `prompt_profile_changed`.

## Event: `binding_invalidated`

Required fields:

- `event_id`
- `bot_id`
- `sender_user_id`

Optional fields:

- `binding_id`
- `reason`

Behavior:

1. `event_id` idempotency check.
2. Mark matching active profile `inactive`.
3. Emits outbox event `binding_invalidated`.

## Notes

- Missing `ADMIN_SYNC_SHARED_SECRET` disables this endpoint (returns 404 style error payload).
- Endpoint is intentionally internal and should only be exposed behind trusted network boundaries.
- Finalize callback envs:
  - `ADMIN_FINALIZE_URL` (e.g. `https://admin-worker.../api/account/platforms/wechat/inbound-finalize`)
  - `ADMIN_FINALIZE_SECRET` (Bearer token; defaults to `ADMIN_SYNC_SHARED_SECRET`)
