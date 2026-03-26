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
    id: "custom-integration",
    emoji: "⚡",
    name: "自定义集成",
    description: "创建自定义 App，通过 Token 调用 Bot API",
    scopes: ["message:write", "message:read", "contact:read", "bot:read"],
    events: ["message"],
    readme: "自定义集成，通过 Token 调用 Bot API 实现任意功能。",
    guide: `## 自定义集成

### WebSocket 连接

\`\`\`
wss://{hub_url}/bot/v1/ws?token={your_token}
\`\`\`

### HTTP 发消息

\`\`\`bash
curl -X POST {hub_url}/bot/v1/message/send \\
  -H "Authorization: Bearer {your_token}" \\
  -H "Content-Type: application/json" \\
  -d '{"content":"hello"}'
\`\`\``,
  },
];
