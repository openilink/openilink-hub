package hermes

import (
	"encoding/json"

	"github.com/openilink/openilink-hub/internal/builtin"
)

func init() {
	builtin.Register(builtin.AppManifest{
		Slug:        "hermes",
		Name:        "Hermes Agent",
		Description: "通过 Hermes Agent 接入 Bot，让 AI 助手处理微信消息",
		Icon:        "🪽",
		Readme:      "通过 Nous Research 的 Hermes Agent 接入 Bot。Hermes Agent 是一个具有自我学习能力的 AI 助手，支持跨平台消息处理。安装此 App 后，Hermes 可以通过 Hub 管理的微信 Bot 收发消息。",
		Guide: "## Hermes Agent 接入指南\n\n你已在 Hub 上安装了 Hermes Agent App。下面将 Hermes Agent 连接到这个 Bot。\n\n### 1. 复制上方的 Token\n\n页面上方显示的 Token 是 Hermes Agent 连接此 Bot 的凭证，请先复制。\n\n### 2. 安装 Gateway Adapter\n\n```bash\npip install hermes-gateway-openilink\n```\n\n### 3. 配置 Token\n\n将 Token 写入 Hermes 配置（`~/.hermes/config.yaml`）：\n\n```yaml\ngateway:\n  openilink:\n    enabled: true\n    hub_url: \"{hub_url}\"\n    app_token: \"{your_token}\"\n```\n\n### 4. 启动 Gateway\n\n```bash\nhermes gateway setup\nhermes gateway start\n```\n\n完成！在微信中发送消息即可与 Hermes AI 对话。\n\n### 手动发送消息（API）\n\n```bash\ncurl -X POST {hub_url}/bot/v1/message/send \\\n  -H \"Authorization: Bearer {your_token}\" \\\n  -H \"Content-Type: application/json\" \\\n  -d '{\"content\":\"hello\"}'\n```\n\n### 更多信息\n\n- [Hermes Agent 文档](https://hermes-agent.nousresearch.com)\n- [GitHub](https://github.com/nousresearch/hermes-agent)",
		Scopes:      []string{"message:read", "message:write"},
		Events:      []string{"message"},
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {}
		}`),
	}, nil) // nil handler — events delivered via WS to Hermes gateway adapter
}
