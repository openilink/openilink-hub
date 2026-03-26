package bridge

import (
	"encoding/json"

	"github.com/openilink/openilink-hub/internal/builtin"
)

func init() {
	builtin.Register(builtin.AppManifest{
		Slug:        "bridge",
		Name:        "Bridge",
		Description: "双向桥接 Bot 与外部系统",
		Icon:        "🔗",
		Readme:      "双向桥接 Bot 与外部系统。Bot 收到的消息会自动转发到配置的 URL，外部系统也可以通过 Token 向 Bot 发送消息。",
		Guide:       "## Bridge\n\n### 接收消息\n\nBot 收到的消息会自动 POST 到你配置的转发地址。\n\n### 发送消息\n\n```bash\ncurl -X POST {hub_url}/bot/v1/message/send \\\n  -H \"Authorization: Bearer {your_token}\" \\\n  -H \"Content-Type: application/json\" \\\n  -d '{\"content\":\"hello\"}'\n```",
		Scopes:      []string{"message:read", "message:write"},
		Events:      []string{"message"},
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"forward_url": {
					"type": "string",
					"format": "uri",
					"title": "转发地址",
					"description": "Bot 收到的消息将 POST 到此地址"
				}
			},
			"required": ["forward_url"]
		}`),
	}, &Handler{})
}
