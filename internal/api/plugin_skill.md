# OpenILink Hub — Webhook Plugin Development Skill

> This document is designed for AI agents and developers to understand how to write webhook plugins for the OpenILink Hub plugin marketplace.

## What is a Webhook Plugin?

A webhook plugin is a JavaScript file that transforms how messages are forwarded from WeChat to external services via HTTP webhooks. Plugins run in a sandboxed JavaScript runtime (goja) on the server.

## Plugin Format

Every plugin is a `.js` file with a `==WebhookPlugin==` metadata block and exported functions:

```javascript
// ==WebhookPlugin==
// @name         Plugin Name (required)
// @namespace    github.com/yourname
// @version      1.0.0
// @description  What this plugin does
// @author       Your Name
// @license      MIT
// @homepage     https://github.com/yourname/plugin
// @icon         🔔
// @match        text,image
// @connect      api.example.com
// @grant        reply
// @config       webhook_url string "Target webhook URL"
// ==/WebhookPlugin==

function onRequest(ctx) {
  // Called BEFORE the HTTP request is sent
  // Modify ctx.req to transform the outgoing request
}

function onResponse(ctx) {
  // Called AFTER the HTTP response is received
  // Read ctx.res and optionally reply()
}
```

## Metadata Fields

| Field | Required | Default | Description |
|---|---|---|---|
| `@name` | Yes | — | Plugin display name |
| `@namespace` | No | — | Unique identifier (e.g. github.com/user) |
| `@version` | No | 1.0.0 | Semver version |
| `@description` | No | — | Short description |
| `@author` | No | — | Author name |
| `@license` | No | — | License (MIT, Apache-2.0, etc.) |
| `@homepage` | No | — | Project URL |
| `@icon` | No | — | Emoji icon for marketplace display |
| `@match` | No | `*` | Message types to trigger on (comma-separated: text,image,voice,video,file) |
| `@connect` | No | `*` | Allowed URL domains for ctx.req.url (comma-separated) |
| `@grant` | No | all | Required permissions: `reply`, `skip`, `none` (comma-separated) |
| `@config` | No | — | Configurable parameter (can have multiple) |

### @match — Message Type Filter

Controls which message types trigger the plugin. Default `*` means all types.

```
// @match  text              only text messages
// @match  text,image        text and image messages
// @match  *                 all message types (default)
```

### @connect — URL Domain Whitelist

Restricts which domains `ctx.req.url` can be changed to. Default `*` allows any domain.

```
// @connect  open.feishu.cn      only allow Feishu
// @connect  api.openai.com      only allow OpenAI
// @connect  *                   allow any domain (default, review carefully)
```

If a script modifies `ctx.req.url` to a domain not in the whitelist, execution is blocked.

### @grant — Permission Declaration

Declares which APIs the plugin needs. If `@grant` is not specified, all APIs are available (backward compatible). If specified, only declared APIs work.

```
// @grant  none              no side effects (reply/skip/forward all blocked)
// @grant  reply             can call reply()
// @grant  forward           can call forward()
// @grant  skip              can call skip()
// @grant  reply,forward     can reply text and forward binary
```

### @config Syntax

```
// @config <name> <type> "<description>"
```

Types: `string`, `string?` (optional), `number`, `bool`

## Context Object (ctx)

### ctx.msg — Inbound Message (read-only)

| Field | Type | Description |
|---|---|---|
| `ctx.msg.event` | string | Always `"message"` |
| `ctx.msg.channel_id` | string | Channel ID |
| `ctx.msg.bot_id` | string | Bot ID |
| `ctx.msg.seq_id` | number | Message sequence ID |
| `ctx.msg.sender` | string | Sender ID (e.g. `user@im.wechat`) |
| `ctx.msg.msg_type` | string | `text`, `image`, `voice`, `video`, `file` |
| `ctx.msg.content` | string | Text content or media description |
| `ctx.msg.timestamp` | number | Unix timestamp in milliseconds |
| `ctx.msg.items` | array | Message items (see below) |

#### ctx.msg.items[]

Each item has:
| Field | Type | Description |
|---|---|---|
| `type` | string | `text`, `image`, `voice`, `video`, `file` |
| `text` | string | Text content or voice transcription |
| `file_name` | string | Original file name |
| `media_url` | string | Download URL (if available) |
| `file_size` | number | File size in bytes |
| `play_time` | number | Voice duration in seconds |
| `ref_title` | string | Quoted message title |

### ctx.req — HTTP Request (modifiable)

| Field | Type | Description |
|---|---|---|
| `ctx.req.url` | string | Target URL (from channel webhook config) |
| `ctx.req.method` | string | HTTP method (default: `POST`) |
| `ctx.req.headers` | object | Request headers (key-value) |
| `ctx.req.body` | string | Request body (default: JSON of ctx.msg) |

### ctx.res — HTTP Response (read-only, only in onResponse)

| Field | Type | Description |
|---|---|---|
| `ctx.res.status` | number | HTTP status code |
| `ctx.res.headers` | object | Response headers |
| `ctx.res.body` | string/null | Response body (null for binary responses) |
| `ctx.res.content_type` | string | Response Content-Type header |
| `ctx.res.size` | number | Response body size in bytes |

When the response is binary (image, audio, video, PDF), `ctx.res.body` is `null`.
Use `ctx.res.content_type` to detect the type and call `forward()` to send it to the user.

## Global Functions

| Function | Description |
|---|---|
| `reply(text)` | Send a text message back to the sender via the bot (max 10 per execution) |
| `forward()` | Forward the binary HTTP response as a media message to the sender (image, audio, video, file) |
| `skip()` | Cancel this webhook delivery (no HTTP request will be made) |
| `JSON.parse(str)` | Parse JSON string |
| `JSON.stringify(obj)` | Serialize to JSON string |

## Sandbox Restrictions

- **5-second timeout** — script is terminated if it runs too long
- **Max call stack depth: 64** — prevents stack overflow
- **No `eval()` or `new Function()`** — code injection prevention
- **No `require()`, `process`, `fs`, `net`** — no system access
- **`reply()` limited to 10 calls** — prevents message spam
- **HTTP sent by Hub** — scripts cannot make their own network requests; they only modify `ctx.req` which Hub sends

## Examples

### 1. Notification Forward (onRequest only)

Transforms the request body. Works for Feishu, DingTalk, Slack, Discord, WeCom — just change the JSON structure.

```javascript
// ==WebhookPlugin==
// @name         Feishu Notification
// @namespace    github.com/openilink
// @version      1.0.0
// @description  Forward WeChat messages to Feishu group bot
// @author       openilink
// @icon         🔔
// @match        text,image,file
// @connect      open.feishu.cn
// @grant        none
// ==/WebhookPlugin==

function onRequest(ctx) {
  ctx.req.body = JSON.stringify({
    msg_type: "text",
    content: {
      text: "[" + ctx.msg.msg_type + "] " + ctx.msg.sender + ": " + ctx.msg.content
    }
  });
}
```

### 2. AI Auto-Reply (onRequest + onResponse + reply)

Full two-phase flow: transform request, parse response, reply through bot.

```javascript
// ==WebhookPlugin==
// @name         ChatGPT Auto-Reply
// @namespace    github.com/openilink
// @version      1.0.0
// @description  Forward to OpenAI API and auto-reply
// @author       openilink
// @icon         🤖
// @match        text
// @connect      api.openai.com
// @grant        reply
// @config       api_key string "OpenAI API Key"
// ==/WebhookPlugin==

function onRequest(ctx) {
  ctx.req.url = "https://api.openai.com/v1/chat/completions";
  ctx.req.headers["Authorization"] = "Bearer YOUR_API_KEY";
  ctx.req.body = JSON.stringify({
    model: "gpt-4o-mini",
    messages: [
      { role: "system", content: "You are a helpful assistant." },
      { role: "user", content: ctx.msg.content }
    ]
  });
}

function onResponse(ctx) {
  var data = JSON.parse(ctx.res.body);
  if (data.choices && data.choices[0]) {
    reply(data.choices[0].message.content);
  }
}
```

### 3. Conditional Filter (skip)

Uses `skip()` to cancel delivery when conditions aren't met.

```javascript
// ==WebhookPlugin==
// @name         Keyword Filter
// @namespace    github.com/openilink
// @version      1.0.0
// @description  Only forward messages containing keywords
// @author       openilink
// @icon         🔍
// @match        text
// @grant        skip
// @config       keywords string "Keywords, comma-separated"
// ==/WebhookPlugin==

function onRequest(ctx) {
  var keywords = ["urgent", "bug", "error", "help"];
  var found = false;
  for (var i = 0; i < keywords.length; i++) {
    if (ctx.msg.content.toLowerCase().indexOf(keywords[i]) >= 0) {
      found = true;
      break;
    }
  }
  if (!found) {
    skip();
    return;
  }
  ctx.req.body = JSON.stringify({
    text: "[ALERT] " + ctx.msg.sender + ": " + ctx.msg.content
  });
}
```

### 4. Media Forward (forward)

Sends a request to an API that returns binary (image/file), then forwards the response to the user.

```javascript
// ==WebhookPlugin==
// @name         Image Generator
// @namespace    github.com/openilink
// @version      1.0.0
// @description  Generate image from text and send to user
// @author       openilink
// @icon         🎨
// @match        text
// @connect      api.example.com
// @grant        forward,reply
// ==/WebhookPlugin==

function onRequest(ctx) {
  ctx.req.url = "https://api.example.com/generate";
  ctx.req.body = JSON.stringify({ prompt: ctx.msg.content });
}

function onResponse(ctx) {
  if (ctx.res.content_type && ctx.res.content_type.indexOf("image/") === 0) {
    forward(); // send image to user
    reply("Image generated (" + ctx.res.size + " bytes)");
  } else if (ctx.res.body) {
    var data = JSON.parse(ctx.res.body);
    if (data.error) reply("Error: " + data.error);
  }
}
```

## Publishing a Plugin

### Step 1: Create a GitHub Repository

Create a dedicated public repository for your plugin (recommended) or add it to an existing repo.

Recommended repo structure:

```
my-webhook-plugin/
├── plugin.js          ← your plugin file
├── README.md          ← usage instructions
└── LICENSE            ← open source license (MIT, Apache-2.0, etc.)
```

The plugin file should contain the full `==WebhookPlugin==` metadata block.

### Step 2: Write and Test Your Plugin

Write your plugin following the format above. Before submitting, verify:

- [ ] `// @name` is present (required)
- [ ] `// @match` is set to the message types you actually need (avoid `*` if possible)
- [ ] `// @connect` is limited to the domains you actually call (avoid `*` if possible)
- [ ] `// @grant` declares only the permissions you use (`none` if no reply/skip)
- [ ] No ES6+ syntax (no `const`, `let`, `=>`, template literals)
- [ ] No infinite loops or excessive recursion
- [ ] `JSON.stringify()` is used to set `ctx.req.body`

### Step 3: Submit to the Marketplace

**Option A: Submit via GitHub URL (recommended)**

1. Push your plugin to GitHub
2. Go to the OpenILink Hub plugin marketplace → "Submit Plugin" tab
3. Paste the GitHub blob URL, e.g.:
   ```
   https://github.com/yourname/my-webhook-plugin/blob/main/plugin.js
   ```
4. The system automatically:
   - Fetches the script content
   - Pins the exact commit hash (ensures what's reviewed is what runs)
   - Parses all `@metadata` fields
5. Click "Submit for Review"

**Option B: Submit via direct paste**

1. Go to the plugin marketplace → "Submit Plugin" tab
2. Switch to "Paste Script" mode
3. Paste your full plugin code
4. Click "Submit for Review"

### Step 4: Admin Review

An administrator will review your plugin:

- **Security analysis**: automatic checks for risky patterns (infinite loops, prototype pollution, wildcard domains)
- **Permission review**: @grant, @connect, @match declarations
- **Code review**: manual inspection of the script logic
- **Approve**: plugin appears in the marketplace for all users
- **Reject**: you'll see the rejection reason and can fix + resubmit

### Step 5: Users Install Your Plugin

Once approved, users can install your plugin in two ways:

1. **From the marketplace**: click "Install" to copy the script
2. **From channel config**: go to Bot → Channel → Webhook → "Plugin Marketplace" → select your plugin → one-click install

The channel's `webhook_config.plugin_id` is set to your plugin's ID. At runtime, the system fetches the script from the database — no manual copy needed.

### Updating Your Plugin

To publish a new version:

1. Update your plugin code (bump `@version`)
2. Push to GitHub
3. Submit the new URL (same repo, new commit)
4. Admin reviews the new version
5. Users can upgrade their channels to the new version from channel settings

Each version has a separate plugin ID. Channels pin to a specific version until the user explicitly upgrades.

## API Endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/api/webhook-plugins` | No | List approved plugins |
| GET | `/api/webhook-plugins?status=pending` | Admin | List pending plugins |
| GET | `/api/webhook-plugins/{id}` | No | Plugin detail |
| POST | `/api/webhook-plugins/submit` | Yes | Submit plugin (github_url or script) |
| POST | `/api/webhook-plugins/{id}/install` | Yes | Install (get script + increment count) |
| PUT | `/api/admin/webhook-plugins/{id}/review` | Admin | Approve or reject |
| DELETE | `/api/admin/webhook-plugins/{id}` | Admin | Delete plugin |

## Tips for AI Agents

When generating a plugin:

1. Always include `// @name` — submission will fail without it
2. Use `JSON.stringify()` to set `ctx.req.body` — it must be a string
3. Use `JSON.parse()` to read `ctx.res.body` in onResponse (check for null first — binary responses have body=null)
4. Call `skip()` to conditionally cancel delivery
5. Call `reply(text)` to send a text message back through the bot
6. Call `forward()` to forward binary HTTP responses (images, files) to the user — check `ctx.res.content_type` first
7. Don't use ES6+ syntax (no arrow functions, no const/let, no template literals) — the runtime is ES5
8. Don't try to access external resources — the sandbox blocks all I/O
9. Keep the script simple and focused — complex logic should live in the webhook receiver
10. Declare `@grant forward` if the plugin needs to forward binary responses
