package openclaw

import (
	"encoding/json"

	"github.com/openilink/openilink-hub/internal/builtin"
)

func init() {
	builtin.Register(builtin.AppManifest{
		Slug:        "openclaw",
		Name:        "OpenClaw",
		Description: "通过 OpenClaw 协议接入 Bot",
		Icon:        "🦞",
		Readme:      "通过 OpenClaw 协议接入 Bot，实现跨平台消息互通。",
		Guide:       "## OpenClaw\n\n🚧 开发中（WIP）\n\n请关注后续更新。",
		Scopes:      []string{"message:read", "message:write"},
		Events:      []string{"message"},
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {}
		}`),
	}, nil) // nil handler = WIP, events will be dropped
}
