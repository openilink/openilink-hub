# OpeniLink Hub App Mock Server -- Development Guide

> This document is for AI coding assistants (Claude, Cursor, etc.) helping developers build Apps that integrate with OpeniLink Hub. The mock server lets you develop and test apps locally without a real Hub instance or WeChat bot.

## Quick Start

### 1. Start the Mock Server

```bash
# Basic
go run ./cmd/appmock

# With webhook delivery to your app
go run ./cmd/appmock --webhook-url http://localhost:8080/webhook

# Custom options
go run ./cmd/appmock --listen :9801 --app-token my-token --app-slug my-app --webhook-url http://localhost:8080/webhook
```

The server prints connection details on startup:

```
App Token:  mock_xxxx          # Use as Bearer token for Bot API calls
App Slug:   test-app           # Handle for @mention routing
Bot ID:     mock-bot
```

### 2. Wire Your App

Your app talks to the mock server exactly like it would talk to a real Hub:

- **Bot API base**: `http://localhost:9801/bot/v1`
- **Auth header**: `Authorization: Bearer {app_token}`
- **WebSocket**: `ws://localhost:9801/bot/v1/ws?token={app_token}`

### 3. Inject Test Events

Simulate a user sending a message to trigger your app:

```bash
# Text message (triggers message.text event)
curl -X POST http://localhost:9801/mock/event \
  -H "Content-Type: application/json" \
  -d '{"sender":"user_alice","content":"hello world"}'

# @mention (routes to your app specifically)
curl -X POST http://localhost:9801/mock/event \
  -d '{"sender":"user_alice","content":"@test-app what is the weather?"}'

# Slash command (matches app tools)
curl -X POST http://localhost:9801/mock/event \
  -d '{"sender":"user_alice","content":"/search hello world"}'

# Group message
curl -X POST http://localhost:9801/mock/event \
  -d '{"sender":"user_alice","content":"hello","group_id":"group_123"}'
```

### 4. Inspect State

```bash
# View messages your app sent via Bot API
curl http://localhost:9801/mock/messages

# View mock server config
curl http://localhost:9801/mock/config

# Reset all state
curl -X POST http://localhost:9801/mock/reset
```

## Architecture

```
Your App  <--- Bot API (HTTP/WS) --->  Mock Server  <--- /mock/event ---  Developer/Tests
                                        (real handlers,
                                         in-memory store,
                                         mock provider)
```

The mock server reuses the **real** Hub API handlers with an in-memory store and mock bot provider. This means all Bot API behavior (auth, scope checking, validation, error responses) is identical to production.

## Mock Server Endpoints

### Control Endpoints

| Method | Path | Description |
|---|---|---|
| POST | `/mock/event` | Inject an inbound message (triggers full dispatch pipeline) |
| GET | `/mock/messages` | View all messages sent by your app |
| GET | `/mock/config` | View mock server configuration (IDs, tokens) |
| POST | `/mock/reset` | Clear all recorded messages and logs |

### POST /mock/event

Simulates a WeChat user sending a message. The message flows through the real `bot.Manager` dispatch pipeline, triggering @mention routing, /command matching, and event delivery to your app.

```json
{
  "sender": "user_alice",       // optional, default "user_test"
  "content": "hello world",     // required
  "type": "text",               // optional, default "text"
  "group_id": ""                // optional, for group messages
}
```

Events are delivered to your app via:
1. WebSocket (if connected at `/bot/v1/ws`)
2. Webhook POST (if `--webhook-url` was provided)

### GET /mock/messages

Returns messages your app sent through the Bot API:

```json
{
  "store_messages": [
    {"id": 1, "to": "user_alice", "items": [{"type":"text","text":"hello"}], "created_at": 1711234567}
  ],
  "provider_messages": [
    {"to": "user_alice", "text": "hello"}
  ]
}
```

## Pre-configured Mock Data

| Entity | ID | Details |
|---|---|---|
| Bot | `mock-bot` | Status: connected, Provider: mock |
| App | `mock-app` | Slug from `--app-slug`, subscribes to `message` events |
| Installation | `mock-inst` | All scopes granted, handle = app slug |
| Contacts | `user_alice`, `user_bob` | Available via GET /bot/v1/contact |

Default scopes: `message:read`, `message:write`, `contact:read`, `bot:read`, `tools:write`

---

## App Development Reference

> The following sections describe the App protocol. Your app should implement these regardless of whether it talks to the mock server or a real Hub.

### Communication Model

```
WeChat <-> OpeniLink Hub (Platform) <-> Your App (External Service)
```

Two directions:

1. **Platform -> App**: Events are POSTed to your webhook URL or pushed via WebSocket
2. **App -> Platform**: Your app calls the Bot API with `Authorization: Bearer {app_token}`

### Event Delivery (Platform -> App)

#### Event Envelope

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
      "message_id": "12345",
      "sender": {"id": "user_alice", "role": "user"},
      "group": null,
      "content": "hello",
      "msg_type": "text",
      "items": []
    }
  }
}
```

#### Event Types

| Event Type | Description |
|---|---|
| `message` | Wildcard: matches any `message.*` |
| `message.text` | Text message |
| `message.image` | Image message |
| `message.voice` | Voice message |
| `message.video` | Video message |
| `message.file` | File message |
| `command` | Slash command or AI tool call |

#### Command Event Data

```json
{
  "event": {
    "type": "command",
    "data": {
      "command": "search",
      "text": "hello world",
      "args": null,
      "sender": {"id": "user_alice", "role": "user"},
      "group": null
    }
  }
}
```

`sender.role` is `"user"` for direct slash commands, `"agent"` for AI tool calls.

#### Routing Rules

- `@handle message` -> routes to the specific app installation with that handle
- `@handle /command args` -> command event to that installation
- `/command args` -> command event to all apps with matching tool
- Plain message -> `message.*` event to all subscribed apps

### Replying to Events

#### Synchronous Reply (in webhook HTTP response)

```json
{"reply": "Here is the answer"}
```

For media:
```json
{"reply_type": "image", "reply_url": "https://example.com/img.png", "reply_name": "chart.png"}
```

| Field | Description |
|---|---|
| `reply` | Text content |
| `reply_type` | `text` (default), `image`, `video`, `file` |
| `reply_url` | URL to media file |
| `reply_base64` | Base64-encoded media (supports `data:` URI prefix) |
| `reply_name` | Filename |
| `reply_async` | `true` = will push result later via Bot API |

#### Async Reply (via Bot API)

For long-running tasks, respond `{"reply_async": true}` then push the result:

```python
requests.post(f"{HUB}/bot/v1/message/send",
    headers={"Authorization": f"Bearer {app_token}"},
    json={"content": "Done!", "to": sender_id, "trace_id": trace_id})
```

### Bot API (App -> Platform)

Base URL: `http://localhost:9801/bot/v1` (mock) or `https://hub.example.com/bot/v1` (production)

Auth: `Authorization: Bearer {app_token}`

#### Send Message

```
POST /bot/v1/message/send
```

```json
{
  "type": "text",
  "content": "hello",
  "to": "user_alice",
  "trace_id": "tr_abc123"
}
```

| Field | Required | Description |
|---|---|---|
| `type` | No | `text` (default), `image`, `video`, `file` |
| `content` | Yes* | Text content (*required for text type) |
| `to` | No | Recipient user ID |
| `url` | No | Media URL (platform downloads it) |
| `base64` | No | Base64-encoded media data |
| `filename` | No | Filename for media |
| `trace_id` | No | Links reply to original message trace |

Response: `{"ok": true, "client_id": "msg_xxx", "trace_id": "tr_xxx"}`

#### List Contacts

```
GET /bot/v1/contact
```

Scope: `contact:read`

Response: `{"ok": true, "contacts": [{"user_id": "...", "last_msg_at": 123, "msg_count": 5}]}`

#### Get Bot Info

```
GET /bot/v1/info
```

Scope: `bot:read`

Response: `{"ok": true, "bot": {"id": "...", "name": "...", "status": "connected"}}`

#### Update Tools

```
PUT /bot/v1/app/tools              # App-level tools (shared across installations)
PUT /bot/v1/installation/tools     # Per-installation tools
```

Scope: `tools:write`

```json
{"tools": [{"name": "search", "description": "Search the web", "command": "search"}]}
```

#### Error Responses

```json
{"ok": false, "error": "error message"}
```

| Status | Meaning |
|---|---|
| 401 | Invalid or missing app_token |
| 403 | Missing required scope |
| 400 | Invalid request body |
| 404 | Bot not found |
| 502 | Bot send failed |
| 503 | Bot not connected or session expired |

### WebSocket Protocol

#### Connect

```
Per-installation: GET /bot/v1/ws?token={app_token}
Per-app:          GET /bot/v1/app/ws?app_id={app_id}&secret={webhook_secret}
```

#### Messages (Server -> Client)

```jsonc
{"type": "init", "data": {"installation_id": "...", "bot_id": "...", "app_name": "...", "app_slug": "..."}}
{"type": "event", "v": 1, "trace_id": "tr_xxx", "installation_id": "inst_xxx", "bot": {"id": "bot_xxx"}, "event": {...}}
{"type": "ack", "req_id": "r1", "ok": true}
{"type": "error", "req_id": "r1", "error": "..."}
{"type": "pong"}
```

#### Messages (Client -> Server)

```jsonc
{"type": "ping"}
{"type": "send", "req_id": "r1", "to": "user_alice", "content": "hello", "msg_type": "text"}
```

### Webhook Verification

When you set a webhook URL, the platform sends a challenge:

```json
POST {webhook_url}
{"v": 1, "type": "url_verification", "challenge": "random_string"}
```

Respond with:
```json
{"challenge": "random_string"}
```

### Request Signing

Event POSTs include these headers:

| Header | Description |
|---|---|
| `X-App-Id` | App ID |
| `X-Installation-Id` | Installation ID |
| `X-Timestamp` | Unix timestamp |
| `X-Signature` | `sha256={HMAC-SHA256(webhook_secret, "{timestamp}:{body}")}` |
| `X-Trace-Id` | Trace ID |

The mock server uses `mock-webhook-secret` as the webhook secret.

### Tools Definition

Tools define slash commands and AI-callable functions:

```json
[
  {
    "name": "search",
    "description": "Search the web for information",
    "command": "search",
    "parameters": {
      "type": "object",
      "properties": {
        "query": {"type": "string", "description": "Search query"}
      },
      "required": ["query"]
    }
  }
]
```

- `name`: Tool identifier
- `description`: What it does (used by AI for tool selection)
- `command`: Slash command trigger without `/` prefix (optional)
- `parameters`: JSON Schema for structured parameters (optional)

### Scopes

| Scope | Capability |
|---|---|
| `message:write` | Send messages via Bot API |
| `message:read` | Receive message events |
| `contact:read` | Read contact list |
| `bot:read` | Read bot info |
| `tools:write` | Update tools dynamically |

## Example: Building a Simple Echo App (Python)

```python
from flask import Flask, request, jsonify

app = Flask(__name__)

@app.route("/webhook", methods=["POST"])
def webhook():
    data = request.json

    # Handle URL verification
    if data.get("type") == "url_verification":
        return jsonify({"challenge": data["challenge"]})

    # Handle events
    if data.get("type") == "event":
        event = data["event"]
        if event["type"] in ("message.text", "command"):
            content = event["data"].get("content") or event["data"].get("text", "")
            return jsonify({"reply": f"Echo: {content}"})

    return jsonify({"ok": True})

if __name__ == "__main__":
    app.run(port=8080)
```

Test it:

```bash
# Terminal 1: Start your app
python app.py

# Terminal 2: Start mock server pointing to your app
go run ./cmd/appmock --webhook-url http://localhost:8080/webhook

# Terminal 3: Send a test message
curl -X POST http://localhost:9801/mock/event \
  -d '{"sender":"alice","content":"hello world"}'

# Check what the mock bot sent back (echo reply via sync response)
curl http://localhost:9801/mock/messages
```

## Example: WebSocket App (Python)

```python
import asyncio, json, websockets, httpx

APP_TOKEN = "test-token-123"
HUB = "http://localhost:9801"

async def main():
    uri = f"ws://localhost:9801/bot/v1/ws?token={APP_TOKEN}"
    async with websockets.connect(uri) as ws:
        # Receive init message
        init = json.loads(await ws.recv())
        print(f"Connected: {init}")

        async for raw in ws:
            msg = json.loads(raw)
            if msg["type"] == "event":
                event = msg["event"]
                content = event["data"].get("content", "")
                sender = event["data"]["sender"]["id"]
                trace_id = msg.get("trace_id", "")

                # Reply via Bot API
                async with httpx.AsyncClient() as client:
                    await client.post(
                        f"{HUB}/bot/v1/message/send",
                        headers={"Authorization": f"Bearer {APP_TOKEN}"},
                        json={"content": f"Echo: {content}", "to": sender, "trace_id": trace_id},
                    )
            elif msg["type"] == "pong":
                pass

asyncio.run(main())
```

Test it:

```bash
# Terminal 1: Start mock server
go run ./cmd/appmock --app-token test-token-123

# Terminal 2: Start your WebSocket app
python ws_app.py

# Terminal 3: Inject a message
curl -X POST http://localhost:9801/mock/event \
  -d '{"sender":"alice","content":"hello"}'

# Check replies
curl http://localhost:9801/mock/messages
```
