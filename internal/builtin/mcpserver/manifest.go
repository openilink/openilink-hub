package mcpserver

import (
	"encoding/json"

	"github.com/openilink/openilink-hub/internal/builtin"
)

func init() {
	builtin.Register(builtin.AppManifest{
		Slug:        "mcp-server",
		Name:        "MCP Server",
		Description: "将 Bot 暴露为 MCP Server，方便 AI 调用发消息",
		Icon:        "🤖",
		Readme:      "将 Bot 暴露为标准的 MCP (Model Context Protocol) Server。安装后，AI 助手（如 Claude、Cursor 等）可以通过 MCP 协议连接并调用 Bot 的能力发送消息、查看联系人等。",
		Guide: "## MCP Server\n\n安装此 App 后，你的 Bot 将暴露为一个 MCP Server，支持 Streamable HTTP 传输。\n\n### 连接方式\n\n在 AI 客户端的 MCP 配置中添加：\n\n```json\n{\n  \"mcpServers\": {\n    \"openilink\": {\n      \"url\": \"{hub_url}/mcp\",\n      \"headers\": {\n        \"Authorization\": \"Bearer {your_token}\"\n      }\n    }\n  }\n}\n```\n\n### 可用工具\n\n| 工具 | 说明 |\n|------|------|\n| `send_message` | 通过 Bot 发送消息 |\n| `list_contacts` | 列出最近联系人 |\n| `get_bot_info` | 获取 Bot 信息 |",
		Scopes: []string{"message:write", "contact:read", "bot:read"},
		Events: []string{},
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {}
		}`),
	}, nil) // nil handler — MCP endpoint handles everything
}
