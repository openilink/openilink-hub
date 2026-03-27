package openclaw

import (
	"encoding/json"

	"github.com/openilink/openilink-hub/internal/builtin"
)

func init() {
	builtin.Register(builtin.AppManifest{
		Slug:        "openclaw",
		Name:        "OpenClaw",
		Description: "通过 OpenClaw 接入 Bot，让 AI 助手处理微信消息",
		Icon:        "🦞",
		Readme:      "通过 OpenClaw 开源 AI 助手接入 Bot。OpenClaw 连接 26+ 个聊天平台，安装此 App 后，OpenClaw 可以通过 Hub 管理的微信 Bot 收发消息。",
		Guide: "## OpenClaw 接入指南\n\n你已在 Hub 上安装了 OpenClaw App。下面将 OpenClaw AI 连接到这个 Bot。\n\n### 1. 复制上方的 Token\n\n页面上方显示的 Token 是 OpenClaw 连接此 Bot 的凭证，请先复制。\n\n### 2. 安装插件\n\n```bash\nopenclaw plugins install openclaw-channel-openilink\n```\n\n### 3. 配置 Token\n\n将 Token 写入 OpenClaw 配置（`~/.openclaw/openclaw.json`）：\n\n```json\n{\n  \"channels\": {\n    \"openilink\": {\n      \"enabled\": true,\n      \"hub_url\": \"{hub_url}\",\n      \"app_token\": \"{your_token}\"\n    }\n  }\n}\n```\n\n### 4. 重启 Gateway\n\n```bash\nopenclaw gateway stop && openclaw gateway\n```\n\n完成！在微信中发送 `@claw 你好` 即可与 AI 对话。\n\n### 验证连接\n\n```bash\nopenclaw channels status\n```\n\n应看到 `OpeniLink Hub default: configured`。\n\n### 手动发送消息（API）\n\n```bash\ncurl -X POST {hub_url}/bot/v1/message/send \\\n  -H \"Authorization: Bearer {your_token}\" \\\n  -H \"Content-Type: application/json\" \\\n  -d '{\"content\":\"hello\"}'\n```\n\n### 更多信息\n\n- [OpenClaw 文档](https://docs.openclaw.ai)\n- [插件源码](https://github.com/openilink/openclaw-channel-openilink)",
		Scopes:      []string{"message:read", "message:write"},
		Events:      []string{"message"},
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {}
		}`),
	}, nil) // nil handler — events delivered via WS to OpenClaw plugin
}
