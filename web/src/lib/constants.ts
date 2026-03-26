export const SCOPE_DESCRIPTIONS: Record<string, string> = {
  "message:write": "通过 Bot 发送微信消息",
  "message:read": "接收 Bot 的微信消息事件",
  "contact:read": "读取 Bot 的联系人列表",
  "bot:read": "读取 Bot 的状态和基本信息",
};

export const SCOPES = [
  { key: "message:write", label: "发送消息", description: "通过 Bot 发送微信消息", category: "write" as const },
  { key: "message:read", label: "接收消息", description: "接收 Bot 的微信消息事件", category: "read" as const },
  { key: "contact:read", label: "读取联系人", description: "读取 Bot 的联系人列表", category: "read" as const },
  { key: "bot:read", label: "读取 Bot 信息", description: "读取 Bot 的状态和基本信息", category: "read" as const },
];

export const EVENT_TYPES = [
  { key: "message.text", label: "文本消息", description: "接收文本消息事件" },
  { key: "message.image", label: "图片消息", description: "接收图片消息事件" },
  { key: "message.video", label: "视频消息", description: "接收视频消息事件" },
  { key: "message.voice", label: "语音消息", description: "接收语音消息事件" },
  { key: "message.file", label: "文件消息", description: "接收文件消息事件" },
  { key: "message.location", label: "位置消息", description: "接收位置消息事件" },
  { key: "contact.added", label: "新增联系人", description: "新联系人添加事件" },
  { key: "group.join", label: "入群", description: "有人加入群组事件" },
  { key: "group.leave", label: "退群", description: "有人退出群组事件" },
];

export const APP_TEMPLATES = [
  {
    id: "websocket-app",
    emoji: "📡",
    name: "WebSocket App",
    description: "通过 WebSocket 实时收发 Bot 消息",
    scopes: ["message:write", "message:read", "contact:read", "bot:read"],
    events: ["message"],
    readme: "通过 WebSocket 实时收发 Bot 消息，支持双向通信和事件订阅。",
    guide: `## WebSocket App

连接 WebSocket 实时收发消息。

### 连接方式

\`\`\`
wss://{hub_url}/bot/v1/ws?token={your_token}
\`\`\`

### 发送消息

通过 WebSocket 发送：
\`\`\`json
{"type":"send","to":"wxid_xxx","content":"hello"}
\`\`\`

或通过 HTTP：
\`\`\`bash
curl -X POST {hub_url}/bot/v1/message/send \\
  -H "Authorization: Bearer {your_token}" \\
  -d '{"to":"wxid_xxx","content":"hello"}'
\`\`\``,
  },
  {
    id: "webhook-app",
    emoji: "🔗",
    name: "Webhook App",
    description: "通过 HTTP API 向 Bot 发送消息",
    scopes: ["message:write"],
    events: [],
    readme: "通过 HTTP API 向 Bot 发送消息，适合 CI/CD、监控告警等场景。",
    guide: `## Webhook App

通过 HTTP API 发送消息。

### 发送消息

\`\`\`bash
curl -X POST {hub_url}/bot/v1/message/send \\
  -H "Authorization: Bearer {your_token}" \\
  -H "Content-Type: application/json" \\
  -d '{"to":"wxid_xxx","content":"hello"}'
\`\`\`

### 发送图片

\`\`\`bash
curl -X POST {hub_url}/bot/v1/message/send \\
  -H "Authorization: Bearer {your_token}" \\
  -d '{"to":"wxid_xxx","type":"image","url":"https://example.com/img.png"}'
\`\`\``,
  },
  {
    id: "openclaw-channel",
    emoji: "🦞",
    name: "OpenClaw Channel",
    description: "通过 OpenClaw 协议接入 Bot",
    scopes: ["message:write", "message:read", "contact:read", "bot:read"],
    events: ["message"],
    readme: "通过 OpenClaw 协议接入 Bot，实现跨平台消息互通。",
    guide: `## OpenClaw Channel

通过 OpenClaw Channel Plugin 接入 Bot。

### 安装 Plugin

请参考 [OpenClaw Channel Plugin 文档](https://github.com/nicepkg/openclaw) 安装和配置。

### 配置

在 OpenClaw 配置中填入以下信息：

- **Hub URL**: \`{hub_url}\`
- **Token**: \`{your_token}\``,
  },
];
