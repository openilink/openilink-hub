# OpeniLink Hub — App Development Guide

> This document is for developers building Apps that integrate with OpeniLink Hub (a WeChat bot management platform). The App system follows a Slack-like model: your App is an external service that communicates with the platform via HTTP.

## Architecture

```
WeChat ←→ OpeniLink Hub (Platform) ←→ Your App (External Service)
```

Two communication directions:

1. **Platform → App**: Platform POSTs events (messages, commands) to your App's Request URL
2. **App → Platform**: Your App calls the Bot API with an `app_token` to send messages, read contacts, etc.

## Quick Start

### 1. Create an App

In the OpeniLink Hub dashboard → Apps → Create App:
- **Name**: Display name (e.g. "GitHub Integration")
- **Slug**: Unique identifier (e.g. `github`, lowercase alphanumeric + hyphens)
- **Tools**: Functions your App exposes (see below)
- **Events**: Event types your App subscribes to (e.g. `message.text`)
- **Scopes**: Permissions your App needs (e.g. `messages.send`)

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

Example — a GitHub App with tools:

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
        "state": {"type": "string", "enum": ["open", "closed", "all"], "description": "Filter state"}
      },
      "required": ["repo"]
    }
  },
  {
    "name": "create_issue",
    "description": "Create a new GitHub issue",
    "command": "issue",
    "parameters": {
      "type": "object",
      "properties": {
        "repo": {"type": "string", "description": "Repository (owner/repo)"},
        "title": {"type": "string", "description": "Issue title"},
        "body": {"type": "string", "description": "Issue body"}
      },
      "required": ["repo", "title"]
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

**By user (slash command):** User sends `/pr openilink/openilink-hub` → platform delivers:
```json
{"command": "pr", "text": "openilink/openilink-hub", "args": null}
```

**By AI Agent (tool calling):** AI decides to call `list_prs` → platform delivers:
```json
{"command": "pr", "text": "", "args": {"repo": "openilink/openilink-hub", "state": "open"}}
```

**Via @handle:** User sends `@github /pr args` or `@github list my PRs` (AI Agent interprets).

Your App should handle both: check `args` first (structured), fall back to parsing `text` (free-form).

Tools without a `command` field are only callable by the AI Agent, not by users directly.

### 2. Install to a Bot

Install your App to a Bot. You'll receive:
- **`app_token`**: Bearer token for calling the Bot API
- **`signing_secret`**: Used to verify that events come from the platform

### 3. Set Request URL

Configure your App's HTTP endpoint. The platform will verify it with a challenge:

```json
POST {your_request_url}
{"v": 1, "type": "url_verification", "challenge": "random_string"}
```

Your server must respond:
```json
{"challenge": "random_string"}
```

### 4. Handle Events

Once verified, the platform will POST events to your Request URL.

## Event Delivery (Platform → App)

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

### Message Events

When a WeChat message matches your App's subscription:

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
      "sender": {
        "id": "wxid_abc",
        "name": "Zhang San"
      },
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
"group": {
  "id": "group_xxx",
  "name": "Tech Team"
}
```

### Tool / Command Events

When a user sends `/command args` or the AI Agent calls a tool, the platform routes to the App.

**User-triggered (free-form text):**
```json
{
  "v": 1,
  "type": "event",
  "trace_id": "tr_abc123",
  "installation_id": "inst_xxx",
  "bot": {"id": "bot_xxx"},
  "event": {
    "type": "command",
    "id": "evt_xxx",
    "timestamp": 1711234567,
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

**AI Agent-triggered (structured args):**
```json
{
  "v": 1,
  "type": "event",
  "trace_id": "tr_abc123",
  "installation_id": "inst_xxx",
  "bot": {"id": "bot_xxx"},
  "event": {
    "type": "command",
    "id": "evt_xxx",
    "timestamp": 1711234567,
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

Your App should handle both modes:

```python
@app.route("/webhook", methods=["POST"])
def handle():
    data = request.json

    # URL verification
    if data.get("type") == "url_verification":
        return jsonify({"challenge": data["challenge"]})

    # Route by event.type
    event_type = data.get("event", {}).get("type", "")
    event_data = data.get("event", {}).get("data", {})

    if event_type == "command":
        args = event_data.get("args") or {}   # structured (from AI Agent)
        text = event_data.get("text", "")       # free-form (from user)

        if event_data["command"] == "pr":
            repo = args.get("repo") or (text.split()[0] if text else None)
            state = args.get("state", "open")
            return list_prs(repo, state)
        elif event_data["command"] == "issue":
            repo = args.get("repo") or (text.split()[0] if text else None)
            title = args.get("title") or " ".join(text.split()[1:])
            return create_issue(repo, title, args.get("body"))

    elif event_type.startswith("message."):
        # Handle message events (delivered to all enabled installations)
        content = event_data.get("content", "")
        sender = event_data.get("sender", {})
        # ...process message...

    return jsonify({"ok": True})
```

### Replying to Events

There are two ways to send replies: synchronous (in the HTTP response) and asynchronous (via Bot API).

#### Method 1: Synchronous Reply (in HTTP response)

Return a JSON body with your reply. The platform sends it to the user immediately.

**Text reply:**
```json
{"reply": "Here are the open PRs:\n1. fix bug\n2. add feature"}
```

**Image reply:**
```json
{"reply_type": "image", "reply_url": "https://example.com/chart.png", "reply_name": "chart.png"}
```

**Video reply:**
```json
{"reply_type": "video", "reply_url": "https://example.com/demo.mp4", "reply_name": "demo.mp4"}
```

**File reply:**
```json
{"reply_type": "file", "reply_url": "https://example.com/report.pdf", "reply_name": "report.pdf"}
```

**Base64 media (no URL needed):**
```json
{"reply_type": "image", "reply_base64": "iVBORw0KGgo...", "reply_name": "chart.png"}
```

**Fallback:** If `reply_type` is media but URL/base64 fails, `reply` text is sent as fallback.

| Field | Required | Description |
|---|---|---|
| `reply` | No | Text message content (or fallback for failed media) |
| `reply_type` | No | `text` (default), `image`, `video`, `file` |
| `reply_url` | No | URL to media file (platform downloads and sends) |
| `reply_base64` | No | Base64-encoded media data (no download needed) |
| `reply_name` | No | Filename for the media (e.g. `report.pdf`) |

#### Method 2: Asynchronous Reply (via Bot API)

For replies that take longer than 3 seconds, or when you need to send multiple messages, use the Bot API:

```python
import requests, base64

HUB = "https://hub.openilink.com"
headers = {"Authorization": f"Bearer {app_token}"}

# Text
requests.post(f"{HUB}/bot/v1/messages/send", headers=headers,
    json={"to": to, "type": "text", "content": "hello"})

# Image via URL
requests.post(f"{HUB}/bot/v1/messages/send", headers=headers,
    json={"to": to, "type": "image", "url": "https://example.com/pic.png", "filename": "pic.png"})

# Image via base64
with open("chart.png", "rb") as f:
    b64 = base64.b64encode(f.read()).decode()
requests.post(f"{HUB}/bot/v1/messages/send", headers=headers,
    json={"to": to, "type": "image", "base64": b64, "filename": "chart.png"})

# File
requests.post(f"{HUB}/bot/v1/messages/send", headers=headers,
    json={"to": to, "type": "file", "url": "https://example.com/report.pdf", "filename": "report.pdf"})
```

`POST /bot/v1/messages/send` fields:

| Field | Required | Description |
|---|---|---|
| `to` | Yes | Recipient WeChat ID |
| `type` | No | `text` (default), `image`, `video`, `file` |
| `content` | Yes* | Text content (*required for text type) |
| `url` | No | Media URL (platform downloads) |
| `base64` | No | Base64-encoded media data |
| `filename` | No | Filename for media |

Use async when:
- Processing takes more than 3 seconds
- You need to send multiple messages
- You need to send different media types in sequence

### Event Types

| Event Type | Trigger |
|---|---|
| `message` | Any WeChat message (wildcard) |
| `message.text` | Text message |
| `message.image` | Image message |
| `message.voice` | Voice message |
| `message.video` | Video message |
| `message.file` | File message |

**Important**: Message events are delivered to all enabled installations whose App subscribes to the event type. If your App has subscribed to an event in its configuration, all enabled installations will receive it.

### Request Signing

Every event POST includes these headers:

| Header | Description |
|---|---|
| `X-App-Id` | Your App's ID |
| `X-Installation-Id` | Installation instance ID |
| `X-Timestamp` | Unix timestamp (seconds) |
| `X-Signature` | `sha256={HMAC-SHA256 hex digest}` |
| `X-Trace-Id` | Trace ID for debugging |
| `Content-Type` | `application/json` |

**Verification algorithm**:
```
expected = HMAC-SHA256(signing_secret, "{timestamp}:{request_body}")
```

Verify:
1. `X-Timestamp` is within 5 minutes of current time
2. Computed signature matches `X-Signature` (after removing `sha256=` prefix)

### Retry Policy

| Attempt | Delay | Condition |
|---|---|---|
| 1 | Immediate | No response or non-2xx |
| 2 | 10 seconds | Same |
| 3 | 60 seconds | Same |

Your App must respond with HTTP 2xx within **3 seconds**. If processing takes longer, respond immediately with 200 and process asynchronously.

## Bot API (App → Platform)

Your App calls these endpoints to interact with the Bot.

**Base URL**: `{hub_origin}/bot/v1`

**Authentication**: `Authorization: Bearer {app_token}`

**Optional headers**:
- `X-Trace-Id`: Your trace ID (links to event trace for debugging)

### Send Message

```
POST /bot/v1/messages/send
Authorization: Bearer {app_token}

{
  "to": "wxid_xxx",
  "type": "text",
  "content": "New PR: fix login bug #123",
  "trace_id": "tr_xxx"
}
```

Response:
```json
{"ok": true, "client_id": "msg_xxx", "trace_id": "tr_xxx"}
```

| Field | Required | Description |
|---|---|---|
| `to` | Yes | Recipient WeChat ID or group ID |
| `type` | No | `text` (default), `image`, `video`, `file` |
| `content` | Yes* | Text content (*required for text type) |
| `url` | No | Media URL (platform downloads and sends) |
| `base64` | No | Base64-encoded media data |
| `filename` | No | Filename for media |
| `trace_id` | No | Pass the `trace_id` from the event to link this reply to the original message trace |

**Important**: Pass the `trace_id` you received in the event envelope. This links your async reply back to the original message in the trace chain, making it visible in the "链路追踪" dashboard.

Examples:
```json
{"to": "wxid_xxx", "type": "text", "content": "hello"}
{"to": "wxid_xxx", "type": "image", "url": "https://example.com/img.png", "filename": "img.png"}
{"to": "wxid_xxx", "type": "image", "base64": "iVBORw0KGgo...", "filename": "chart.png"}
{"to": "wxid_xxx", "type": "file", "url": "https://example.com/report.pdf", "filename": "report.pdf"}
```

### List Contacts

```
GET /bot/v1/contacts
Authorization: Bearer {app_token}
```

Response:
```json
{
  "ok": true,
  "contacts": [
    {"user_id": "wxid_abc", "display_name": "Zhang San", "last_seen": 1711234567},
    ...
  ]
}
```

**Required scope**: `contacts.read`

### Get Bot Info

```
GET /bot/v1/bot
Authorization: Bearer {app_token}
```

Response:
```json
{
  "ok": true,
  "bot": {
    "id": "bot_xxx",
    "name": "My Bot",
    "provider": "wechat",
    "status": "connected",
    "msg_count": 1234,
    "created_at": 1711234567,
    "updated_at": 1711234567
  }
}
```

**Required scope**: `bot.read`

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
| 502 | Bot send failed |
| 503 | Bot not connected or session expired |

## Scopes

| Scope | Capability |
|---|---|
| `messages.send` | Send messages via the Bot |
| `contacts.read` | Read the Bot's contact list |
| `bot.read` | Read Bot info (name, status, etc.) |

Declare only the scopes your App needs. Users see the requested scopes when installing.

## Full Example: GitHub Notification Bot

A complete App that:
- Receives `/github subscribe owner/repo` commands from WeChat
- Receives GitHub webhook events and notifies WeChat

### 1. App Configuration

```
Name: GitHub Bot
Slug: github-bot
Tools: [{"name": "github_cmd", "description": "GitHub commands", "command": "github"}]
Events: []
Scopes: ["messages.send", "contacts.read"]
```

### 2. Server (Python example)

```python
import hmac, hashlib, json, time
from flask import Flask, request, jsonify
import requests

app = Flask(__name__)

SIGNING_SECRET = "your_signing_secret"
APP_TOKEN = "app_xxx"
HUB_URL = "https://your-hub.example.com"

def verify_signature(req):
    timestamp = req.headers.get("X-Timestamp", "")
    signature = req.headers.get("X-Signature", "").removeprefix("sha256=")
    if abs(time.time() - int(timestamp)) > 300:
        return False
    body = req.get_data()
    expected = hmac.new(
        SIGNING_SECRET.encode(), f"{timestamp}:".encode() + body, hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(expected, signature)

# Handle events from OpeniLink Hub
@app.route("/webhook", methods=["POST"])
def handle_event():
    data = request.json

    # URL verification
    if data.get("type") == "url_verification":
        return jsonify({"challenge": data["challenge"]})

    # Verify signature
    if not verify_signature(request):
        return "invalid signature", 401

    # Route by event.type
    event_type = data.get("event", {}).get("type", "")
    event_data = data.get("event", {}).get("data", {})

    if event_type == "command":
        if event_data["command"] == "github":
            return jsonify({"reply": f"GitHub command received: {event_data['text']}"})

    elif event_type.startswith("message."):
        # Received if the App subscribes to this event type
        pass

    return jsonify({"ok": True})

# Handle GitHub webhook events
@app.route("/github-webhook", methods=["POST"])
def handle_github():
    event = request.headers.get("X-GitHub-Event")
    data = request.json

    if event == "pull_request":
        action = data["action"]
        pr = data["pull_request"]
        text = f"PR {action}: {pr['title']}\n{pr['html_url']}"

        # Send to WeChat via Bot API
        requests.post(
            f"{HUB_URL}/bot/v1/messages/send",
            headers={"Authorization": f"Bearer {APP_TOKEN}"},
            json={"to": "group_xxx", "content": text}
        )

    return "ok"
```

### 3. Server (Go example)

```go
package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strconv"
    "strings"
    "time"
)

const (
    signingSecret = "your_signing_secret"
    appToken      = "app_xxx"
    hubURL        = "https://your-hub.example.com"
)

func verifySignature(r *http.Request, body []byte) bool {
    ts := r.Header.Get("X-Timestamp")
    sig := strings.TrimPrefix(r.Header.Get("X-Signature"), "sha256=")
    tsInt, _ := strconv.ParseInt(ts, 10, 64)
    if abs(time.Now().Unix()-tsInt) > 300 {
        return false
    }
    mac := hmac.New(sha256.New, []byte(signingSecret))
    mac.Write([]byte(ts + ":"))
    mac.Write(body)
    expected := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(sig))
}

func abs(n int64) int64 { if n < 0 { return -n }; return n }

func sendToWeChat(to, content string) {
    body, _ := json.Marshal(map[string]string{"to": to, "content": content})
    req, _ := http.NewRequest("POST", hubURL+"/bot/v1/messages/send", strings.NewReader(string(body)))
    req.Header.Set("Authorization", "Bearer "+appToken)
    req.Header.Set("Content-Type", "application/json")
    http.DefaultClient.Do(req)
}

func handleEvent(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    var envelope map[string]any
    json.Unmarshal(body, &envelope)

    // URL verification
    if envelope["type"] == "url_verification" {
        json.NewEncoder(w).Encode(map[string]string{
            "challenge": envelope["challenge"].(string),
        })
        return
    }

    // Verify signature
    if !verifySignature(r, body) {
        http.Error(w, "invalid signature", 401)
        return
    }

    // Route by event.type
    event := envelope["event"].(map[string]any)
    eventType, _ := event["type"].(string)
    data, _ := event["data"].(map[string]any)

    if eventType == "command" {
        json.NewEncoder(w).Encode(map[string]string{
            "reply": fmt.Sprintf("GitHub command: %s", data["text"]),
        })
        return
    }

    w.Write([]byte(`{"ok":true}`))
}

func handleGitHub(w http.ResponseWriter, r *http.Request) {
    event := r.Header.Get("X-GitHub-Event")
    var data map[string]any
    json.NewDecoder(r.Body).Decode(&data)

    if event == "pull_request" {
        pr := data["pull_request"].(map[string]any)
        text := fmt.Sprintf("PR %s: %s\n%s",
            data["action"], pr["title"], pr["html_url"])
        sendToWeChat("group_xxx", text)
    }

    w.Write([]byte("ok"))
}

func main() {
    http.HandleFunc("/webhook", handleEvent)
    http.HandleFunc("/github-webhook", handleGitHub)
    http.ListenAndServe(":8080", nil)
}
```

## API Endpoints Summary

### Dashboard API (User Management)

| Method | Path | Description |
|---|---|---|
| POST | `/api/apps` | Create App |
| GET | `/api/apps` | List my Apps |
| GET | `/api/apps/{id}` | Get App detail |
| PUT | `/api/apps/{id}` | Update App |
| DELETE | `/api/apps/{id}` | Delete App |
| POST | `/api/apps/{id}/install` | Install to Bot |
| GET | `/api/apps/{id}/installations` | List installations |
| GET | `/api/apps/{id}/installations/{iid}` | Installation detail |
| PUT | `/api/apps/{id}/installations/{iid}` | Update installation |
| DELETE | `/api/apps/{id}/installations/{iid}` | Uninstall |
| POST | `/api/apps/{id}/installations/{iid}/regenerate-token` | Regenerate token |
| POST | `/api/apps/{id}/verify-url` | Verify request URL |
| GET | `/api/apps/{id}/installations/{iid}/event-logs` | Event delivery logs |
| GET | `/api/apps/{id}/installations/{iid}/api-logs` | API call logs |

### Bot API (App Calls)

| Method | Path | Scope | Description |
|---|---|---|---|
| POST | `/bot/v1/messages/send` | `messages.send` | Send message |
| GET | `/bot/v1/contacts` | `contacts.read` | List contacts |
| GET | `/bot/v1/bot` | `bot.read` | Get bot info |

## Tracing

Every event the platform delivers to your App includes a `trace_id` in the envelope:

```json
{
  "v": 1,
  "type": "event",
  "trace_id": "tr_550e8400-e29b-41d4-a716-446655440000",
  ...
}
```

This `trace_id` links back to the original WeChat message. The platform records a trace for every message showing the full processing chain:

```
receive → store → match_command → deliver_app → reply → async_reply
```

**To make your async replies appear in the trace**, pass `trace_id` when calling the Bot API:

```python
def handle_event(data):
    trace_id = data["trace_id"]  # save this

    # Sync reply (automatically traced)
    return jsonify({"reply": "Processing..."})

    # Later, async reply (pass trace_id to link back)
    requests.post(f"{HUB}/bot/v1/messages/send",
        headers={"Authorization": f"Bearer {token}"},
        json={
            "to": sender_id,
            "content": "Done! Here are the results...",
            "trace_id": trace_id  # ← links to original message trace
        })
```

Bot owners can view the full trace in **Bot detail → 链路追踪** tab.

## Tips for AI Agents

When building an App:

1. Always handle URL verification (`"type": "url_verification"`) first
2. Verify the `X-Signature` on every event to ensure it comes from the platform
3. Respond to events within 3 seconds — process asynchronously if needed
4. **Pass `trace_id` from events when calling the Bot API** for end-to-end tracing
5. Declare minimum required scopes
6. Handle retry gracefully — use `event.id` for deduplication
7. Commands are triggered by `/command` or `@handle` in WeChat messages
8. The `to` field in send message must be a valid WeChat contact ID on the Bot
