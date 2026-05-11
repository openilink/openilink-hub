# Full Prompt Runtime Flow

## Runtime rule

At reply time, AI system prompt comes from exactly one source:

1. local active `prompt_profiles.full_prompt`, or
2. global fallback `ai.system_prompt`

No runtime merge of `system_prompt + user_prompt` is performed.

## Resolution steps

1. AI sink loads global config.
2. AI sink queries local profile by `(bot_id, sender_user_id)`.
3. If profile exists and `full_prompt` is non-blank:
   - use local `full_prompt` as system prompt
   - trace tags:
     - `prompt.source=local_full_prompt`
     - `prompt.version=<prompt_version>`
     - `prompt.full_hash=<sha256_prefix>`
4. Otherwise:
   - fallback to global `ai.system_prompt`
   - trace tag: `prompt.source=global_fallback`

## Safety guards

- `AI_FULL_PROMPT_MAX_BYTES` truncation happens on admin sync write path.
- blank `full_prompt` is rejected on sync API.
- profile version guard prevents stale event rollback.
