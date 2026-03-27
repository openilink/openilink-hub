# OpeniLink Hub -- App Development Guide

> This document is for developers building Apps that integrate with OpeniLink Hub (a WeChat bot management platform). The App system follows a Slack-like model: your App is an external service that communicates with the platform via HTTP webhooks or WebSocket.

## Architecture

```
WeChat <-> OpeniLink Hub (Platform) <-> Your App (External Service)
```

Two communication directions:

1. **Platform -> App**: Platform POSTs events (messages, commands) to your App's Webhook URL (or pushes via WebSocket)
2. **App -> Platform**: Your App calls the Bot API with an `app_token` to send messages, read contacts, etc.

### App Types (by registry field)

- **Local App** (`registry: ""`): Created by developers on a specific Hub instance
- **Built-in Template** (`registry: "builtin"`): Hub built-in template (WebSocket App, Webhook App, OpenClaw Channel)
- **Marketplace App** (`registry: "https://..."`): Installed from an external App Registry

## Quick Start

### 1. Create an App

In the OpeniLink Hub dashboard -> Apps -> Create App:
- **Name**: Display name (e.g. "GitHub Integration")
- **Slug**: Unique identifier (e.g. `github`, lowercase alphanumeric + hyphens)
- **Tools**: Functions your App exposes (see below)
- **Events**: Event types your App subscribes to (e.g. `message.text`)
- **Scopes**: Permissions your App needs (e.g. `message:write`)

#### Tools

Tools define your App's capabilities. Each tool is a function that can be:
- Triggered by users via slash commands (e.g. `/pr`)
- Called by the platform's AI Agent via structured tool calling

| Field | Required | Description |
|---|---|---|
| `name` | Yes | Tool identifier (e.g. `list_prs`) |
| `description` | Yes | What this tool does (used by AI Agent for tool selection) |
| `command` | No | Slash command trigger without `/` prefix (e.g. `pr`) |
| `parameters` | No | JSON Schema defining structured parameters |

Example -- a GitHub App with tools:

```json
[
  {
    "name": "list_prs",
    "description": "List pull requests for a repository",
    "command": "pr",
    "parameters": {
      "type": "object",
      "properties": {
        "repo": {"type": "string", "description": "Repository (owner/repo)"},
        "state": {"type": "string", "enum": ["open", "closed", "all"]}
      },
      "required": ["repo"]
    }
  },
  {
    "name": "ping",
    "description": "Check if the service is alive",
    "command": "ping"
  }
]
```

##### How tools are triggered

**By user (slash command):** User sends `/pr openilink/openilink-hub` -> platform delivers:
```json
{"command": "pr", "text": "openilink/openilink-hub", "args": null}
```

**By AI Agent (tool calling):** AI decides to call `list_prs` -> platform delivers:
```json
{"command": "pr", "text": "", "args": {"repo": "openilink/openilink-hub", "state": "open"}}
```

**Via @handle:** User sends `@github /pr args` or `@github list my PRs` (AI Agent interprets).

Tools without a `command` field are only callable by the AI Agent.

### 2. Install to a Bot

Install your App to a Bot. You'll receive:
- **`app_token`**: Bearer token for calling the Bot API
- **`webhook_secret`**: Used to verify that events come from the platform

### 3. Set Webhook URL

Configure your App's HTTP endpoint. The platform will verify it with a challenge:

```json
POST {your_webhook_url}
{"v": 1, "type": "url_verification", "challenge": "random_string"}
```

Your server must respond:
```json
{"challenge": "random_string"}
```

### 4. Handle Events

Once verified, the platform will POST events to your Webhook URL.

## Event Delivery (Platform -> App)

### Event Envelope

All events share this envelope format:

```json
{
  "v": 1,
  "type": "event",
  "trace_id": "tr_abc123",
  "installation_id": "inst_xxx",
  "bot": {
    "id": "bot_xxx"
  },
  "event": {
    "type": "message.text",
    "id": "evt_xxx",
    "timestamp": 1711234567,
    "data": { ... }
  }
}
```

### Event Types

| Event Type | Trigger |
|---|---|
| `message` | Any message (wildcard, matches any `message.*`) |
| `message.text` | Text message |
| `message.image` | Image message |
| `message.voice` | Voice message |
| `message.video` | Video message |
| `message.file` | File message |
| `command` | Slash command or AI tool call |

**Important**: To receive message events, the App must declare the `message:read` scope.

### Message Events

```json
{
  "v": 1,
  "type": "event",
  "trace_id": "tr_abc123",
  "installation_id": "inst_xxx",
  "bot": {"id": "bot_xxx"},
  "event": {
    "type": "message.text",
    "id": "evt_xxx",
    "timestamp": 1711234567,
    "data": {
      "message_id": 12345,
      "sender": {"id": "wxid_abc", "name": "Zhang San"},
      "group": null,
      "content": "hello",
      "msg_type": "text",
      "items": []
    }
  }
}
```

Group messages include `group`:
```json
"group": {"id": "group_xxx", "name": "Tech Team"}
```

### Command Events

User-triggered:
```json
{
  "event": {
    "type": "command",
    "data": {
      "command": "pr",
      "text": "openilink/openilink-hub open",
      "args": null,
      "sender": {"id": "wxid_abc", "name": "Zhang San"},
      "group": null
    }
  }
}
```

AI Agent-triggered:
```json
{
  "event": {
    "type": "command",
    "data": {
      "command": "pr",
      "text": "",
      "args": {"repo": "openilink/openilink-hub", "state": "open"},
      "sender": {"id": "system", "name": "AI Agent"},
      "group": null
    }
  }
}
```

### Replying to Events

#### Method 1: Synchronous Reply (in HTTP response)

**Text reply:**
```json
{"reply": "Here are the open PRs:\n1. fix bug\n2. add feature"}
```

**Media reply:**
```json
{"reply_type": "image", "reply_url": "https://example.com/chart.png", "reply_name": "chart.png"}
```

| Field | Required | Description |
|---|---|---|
| `reply` | No | Text content (or fallback for failed media) |
| `reply_type` | No | `text` (default), `image`, `video`, `file` |
| `reply_url` | No | URL to media file |
| `reply_base64` | No | Base64-encoded media data |
| `reply_name` | No | Filename for the media |

#### Method 2: Asynchronous Reply (via Bot API)

For replies that take longer than 3 seconds:

```python
import requests
HUB = "https://hub.openilink.com"
headers = {"Authorization": f"Bearer {app_token}"}

requests.post(f"{HUB}/bot/v1/message/send", headers=headers,
    json={"type": "text", "content": "hello"})
```

### Request Signing

Every event POST includes these headers:

| Header | Description |
|---|---|
| `X-App-Id` | Your App's ID |
| `X-Installation-Id` | Installation instance ID |
| `X-Timestamp` | Unix timestamp (seconds) |
| `X-Signature` | `sha256={HMAC-SHA256 hex digest}` |
| `X-Trace-Id` | Trace ID for debugging |

**Verification algorithm**:
```
expected = HMAC-SHA256(webhook_secret, "{timestamp}:{request_body}")
```

### Retry Policy

| Attempt | Delay | Condition |
|---|---|---|
| 1 | Immediate | No response or non-2xx |
| 2 | 10 seconds | Same |
| 3 | 60 seconds | Same |

Your App must respond within **3 seconds**. Process asynchronously if needed.

## WebSocket Protocol

Apps can connect via WebSocket to receive events in real-time instead of webhook POSTs.

### Per-Installation WS

Each installation connects with its own `app_token`:

```
GET /bot/v1/ws?token={app_token}
```

Receives events for this one installation only. Suitable for self-hosted integrations, scripts, Bridge App.

### Per-App WS

Hosted apps can use a single connection to receive events for ALL installations on this Hub:

```
GET /bot/v1/app/ws?app_id={app_id}&secret={webhook_secret}
```

Each event includes `installation_id` to differentiate. Suitable for hosted third-party apps (e.g., GitHub Integration) that serve many installations.

### WS Protocol

**Server → Client:**
```jsonc
{"type":"init", "data":{"installation_id":"...", "bot_id":"...", "app_slug":"..."}}
{"type":"event", "v":1, "trace_id":"tr_xxx", "installation_id":"inst_xxx", "bot":{"id":"bot_xxx"}, "event":{...}}
{"type":"ack", "req_id":"r1", "ok":true}
{"type":"error", "req_id":"r1", "error":"..."}
{"type":"pong"}
```

**Client → Server:**
```jsonc
{"type":"ping"}
{"type":"send", "req_id":"r1", "content":"hello"}
```

### Event Delivery Priority

```
1. Installation WS connected → push via WS
2. App-level WS connected → push via WS
3. webhook_url configured → POST via HTTP
4. None → event dropped
```

## OAuth Install Flow (PKCE)

For Apps that need user authorization during installation:

1. Hub redirects user to your `oauth_setup_url` (in a popup window):
   ```
   {oauth_setup_url}?hub={hub_url}&app_id={app_id}&bot_id={bot_id}&state={state}&return_url={return_url}
   ```
   **Save `return_url`** — you will redirect to it after the OAuth exchange.

2. Your App generates a PKCE `code_verifier` and `code_challenge`:
   ```
   code_verifier = random(43-128 chars)
   code_challenge = base64url(sha256(code_verifier))
   ```

3. Your App redirects user to Hub's authorize endpoint:
   ```
   {hub_url}/api/apps/{app_id}/oauth/authorize?bot_id={bot_id}&state={state}&code_challenge={code_challenge}
   ```

4. Hub redirects back to your `oauth_redirect_url` with a temporary code:
   ```
   {oauth_redirect_url}?code={code}&state={state}
   ```

5. Your App exchanges the code for credentials:
   ```
   POST {hub_url}/api/apps/{app_id}/oauth/exchange
   {"code": "{code}", "code_verifier": "{code_verifier}"}
   ```

   Response:
   ```json
   {
     "installation_id": "inst_xxx",
     "app_token": "tok_xxx",
     "webhook_secret": "sec_xxx",
     "bot_id": "bot_xxx"
   }
   ```

6. **After exchange, redirect to `return_url`** to close the popup and return the user to Hub:
   ```python
   return_url = request.args.get("return_url")
   # ... after exchange succeeds ...
   return redirect(return_url)
   ```
   This is required. The `return_url` points to Hub's OAuth complete page, which sends a `postMessage` to the opener window and closes the popup automatically. If you skip this step, the user will be stuck in the popup window.

If no `code_challenge` was provided during authorize, the code exchange succeeds without PKCE verification (backward compatible).

## Bot API (App -> Platform)

**Base URL**: `{hub_origin}/bot/v1`

**Authentication**: `Authorization: Bearer {app_token}`

### Send Message

```
POST /bot/v1/message/send
```

| Field | Required | Description |
|---|---|---|
| `type` | No | `text` (default), `image`, `video`, `file` |
| `content` | Yes* | Text content (*required for text type) |
| `to` | No | 指定接收人（默认发给 Bot 自身，由 Bot 转发） |
| `url` | No | Media URL |
| `base64` | No | Base64-encoded media data |
| `filename` | No | Filename for media |
| `trace_id` | No | Links reply to original message trace |

Response:
```json
{"ok": true, "client_id": "msg_xxx", "trace_id": "tr_xxx"}
```

### List Contacts

```
GET /bot/v1/contact
```

**Required scope**: `contact:read`

Response:
```json
{"ok": true, "contacts": [...]}
```

### Get Bot Info

```
GET /bot/v1/info
```

**Required scope**: `bot:read`

Response:
```json
{
  "ok": true,
  "bot": {"id": "bot_xxx", "name": "My Bot", "provider": "wechat", "status": "connected"}
}
```

### Update App Tools

```
PUT /bot/v1/app/tools
Authorization: Bearer {app_token}
```

Dynamically update the **app-level** tools/commands at runtime. Requires `tools:write` scope. Only works for local apps (not marketplace or builtin). App-level tools are shared across all installations of the app.

| Field | Required | Description |
|---|---|---|
| `tools` | Yes | JSON array of tool definitions |

Response:
```json
{"ok": true, "tool_count": 5}
```

### Update Installation Tools

```
PUT /bot/v1/installation/tools
Authorization: Bearer {app_token}
```

Update **per-installation** tools/commands. Requires `tools:write` scope. Installation-level tools supplement the app-level tools -- during command matching, both sets are checked.

Use this when different installations of the same app need different commands. For example, one installation might expose `/hn` while another exposes `/weather`.

| Field | Required | Description |
|---|---|---|
| `tools` | Yes | JSON array of tool definitions |

Response:
```json
{"ok": true, "tool_count": 2}
```

#### App-level vs Installation-level Tools

```
App.Tools = [/echo, /ping]              -- shared by all installations
Installation.Tools = [/hn, /weather]     -- per-installation only

Command matching merges both sets:
  /echo    -> matched (from App)
  /weather -> matched (from Installation)
```

### WebSocket (Per-Installation)

```
GET /bot/v1/ws?token={app_token}
```

Real-time bidirectional connection. See **WebSocket Protocol** section above.

### WebSocket (Per-App)

```
GET /bot/v1/app/ws?app_id={app_id}&secret={webhook_secret}
```

Single connection receives events for ALL installations of this app. See **WebSocket Protocol** section above.

### Error Responses

```json
{"ok": false, "error": "error message"}
```

| Status | Meaning |
|---|---|
| 401 | Invalid or missing app_token |
| 403 | Missing required scope |
| 400 | Invalid request body |
| 404 | Bot or resource not found |
| 501 | Not implemented (WebSocket) |
| 502 | Bot send failed |
| 503 | Bot not connected or session expired |

## Scopes

| Scope | Capability |
|---|---|
| `message:write` | Send messages via the Bot |
| `message:read` | Receive message events |
| `contact:read` | Read the Bot's contact list |
| `bot:read` | Read Bot info (name, status, etc.) |
| `tools:write` | Dynamically update app tools/commands |

Declare only the scopes your App needs. Users see the requested scopes when installing.

**Scope locking (Slack model):** Scopes are snapshotted at installation time. If you add new scopes to your App later, existing installations are **not** automatically updated. Users must explicitly re-authorize to grant new scopes:

```
POST /api/apps/{id}/installations/{iid}/reauthorize
```

This updates the installation's scopes to the App's current scopes.

## Marketplace

### Installing from a Registry

The Hub can connect to external App Registries. Browse available apps at:

```
GET /api/marketplace
```

Sync a marketplace app to the latest version:

```
POST /api/marketplace/sync/{slug}
```

### Publishing to the Registry

1. Create your App on a Hub instance
2. Ensure all required fields are set:
   - **name**, **description**, **readme**, **version**
   - **webhook_url** (must be set and verified, unless `registry: "builtin"`)
   - At least one **tool** or **event subscription**
   - At least one **scope**
3. Request listing: `POST /api/apps/{id}/request-listing`
4. An admin reviews and approves (or rejects with a reason)
5. Once listed, the app appears in the Hub's registry (if enabled)

#### Pending Review Rules

While a listing request is pending:
- **Cosmetic fields** (name, description, icon, readme, guide) can still be updated
- **Core fields** (webhook_url, tools, events, scopes, config_schema) are frozen -- changes are blocked
- To modify core fields, withdraw the request first: `POST /api/apps/{id}/withdraw-listing`
- After withdrawing, the app reverts to "unlisted" and you can make changes before re-submitting

#### Auto-Revert on Core Changes

If a **listed** app's core fields are updated (tools, events, scopes, config_schema, or webhook_url), the listing is automatically reverted to "pending" for re-review. Cosmetic changes do not trigger a re-review.

#### Withdraw a Listing Request

```
POST /api/apps/{id}/withdraw-listing
```

Reverts a pending listing request back to "unlisted". Only works when listing is "pending".

### Registry API

Any Hub can act as an App Registry. Enable it in admin settings.

```
GET /api/registry/v1/apps.json
```

Returns a manifest:
```json
{
  "version": 1,
  "updated_at": "2026-03-26T00:00:00Z",
  "apps": [
    {
      "slug": "github-bot",
      "name": "GitHub Bot",
      "description": "GitHub notifications",
      "version": "1.0.0",
      "author": "developer",
      "icon_url": "https://...",
      "tools": [...],
      "events": [...],
      "scopes": [...]
    }
  ]
}
```

### Admin: Managing Registries

```
GET    /api/admin/registries          -- list registry sources
POST   /api/admin/registries          -- add a registry source
PUT    /api/admin/registries/{id}     -- enable/disable
DELETE /api/admin/registries/{id}     -- remove
```

Enable this Hub as a registry:
```
PUT /api/admin/config/registry
{"enabled": "true"}
```

## API Endpoints Summary

### Dashboard API (User Management)

| Method | Path | Description |
|---|---|---|
| POST | `/api/apps` | Create App |
| GET | `/api/apps` | List my Apps |
| GET | `/api/apps?listing=listed` | List publicly listed Apps |
| GET | `/api/apps/{id}` | Get App detail |
| PUT | `/api/apps/{id}` | Update App |
| DELETE | `/api/apps/{id}` | Delete App |
| POST | `/api/bots/{id}/apps` | Install App to Bot (unified) |
| POST | `/api/apps/{id}/request-listing` | Request listing review |
| POST | `/api/apps/{id}/withdraw-listing` | Withdraw pending listing request |
| POST | `/api/apps/{id}/verify-url` | Verify webhook URL |
| GET | `/api/apps/{id}/installations` | List installations |
| GET | `/api/apps/{id}/installations/{iid}` | Installation detail |
| PUT | `/api/apps/{id}/installations/{iid}` | Update installation |
| DELETE | `/api/apps/{id}/installations/{iid}` | Uninstall |
| POST | `/api/apps/{id}/installations/{iid}/regenerate-token` | Regenerate token |
| POST | `/api/apps/{id}/installations/{iid}/reauthorize` | Re-authorize (update scopes to current app scopes) |
| GET | `/api/apps/{id}/installations/{iid}/event-logs` | Event delivery logs |
| GET | `/api/apps/{id}/installations/{iid}/api-logs` | API call logs |

### OAuth Flow

| Method | Path | Description |
|---|---|---|
| GET | `/api/apps/{id}/oauth/setup` | Redirect to app's setup URL |
| GET | `/api/apps/{id}/oauth/authorize` | Authorize (with PKCE code_challenge) |
| POST | `/api/apps/{id}/oauth/exchange` | Exchange code (with PKCE code_verifier) |
| GET | `/oauth/complete` | OAuth callback page (closes popup, returns user to Hub) |

### Marketplace

| Method | Path | Description |
|---|---|---|
| GET | `/api/marketplace` | Browse available apps |
| POST | `/api/marketplace/sync/{slug}` | Sync to latest version |

### Bot API (App Calls)

| Method | Path | Scope | Description |
|---|---|---|---|
| POST | `/bot/v1/message/send` | `message:write` | Send message |
| GET | `/bot/v1/contact` | `contact:read` | List contacts |
| GET | `/bot/v1/info` | `bot:read` | Get bot info |
| PUT | `/bot/v1/app/tools` | `tools:write` | Update app-level tools dynamically |
| PUT | `/bot/v1/installation/tools` | `tools:write` | Update per-installation tools |
| GET | `/bot/v1/ws` | - | WebSocket (per-installation) |
| GET | `/bot/v1/app/ws` | - | WebSocket (per-app, all installations) |

Legacy paths (`/bot/v1/messages/send`, `/bot/v1/contacts`, `/bot/v1/bot`) are supported for backward compatibility.

### Admin

| Method | Path | Description |
|---|---|---|
| GET | `/api/admin/apps` | List all apps |
| PUT | `/api/admin/apps/{id}/review-listing` | Approve/reject listing |
| GET | `/api/admin/registries` | List registry sources |
| POST | `/api/admin/registries` | Add registry source |
| PUT | `/api/admin/registries/{id}` | Update registry |
| DELETE | `/api/admin/registries/{id}` | Delete registry |
| GET | `/api/admin/config/registry` | Get registry config |
| PUT | `/api/admin/config/registry` | Set registry config |

### Registry (Public)

| Method | Path | Description |
|---|---|---|
| GET | `/api/registry/v1/apps.json` | App manifest (if enabled) |

## Tracing

Every event includes a `trace_id`. Pass it when calling the Bot API to link async replies to the original message trace:

```python
requests.post(f"{HUB}/bot/v1/message/send",
    headers={"Authorization": f"Bearer {token}"},
    json={"content": "Done!", "trace_id": trace_id})
```

## Tips

1. Always handle URL verification (`"type": "url_verification"`) first
2. Verify the `X-Signature` on every event
3. Respond within 3 seconds -- process asynchronously if needed
4. Pass `trace_id` from events when calling the Bot API
5. Declare minimum required scopes
6. Handle retry gracefully -- use `event.id` for deduplication
