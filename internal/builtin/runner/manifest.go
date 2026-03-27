package runner

import (
	"encoding/json"

	"github.com/openilink/openilink-hub/internal/builtin"
)

func init() {
	builtin.Register(builtin.AppManifest{
		Slug:        "runner",
		Name:        "Runner",
		Description: "将本地命令桥接到 Bot，微信发命令即可执行",
		Icon:        "⚡",
		Readme:      "将本地 CLI 命令桥接到微信 Bot。安装后，你可以在微信中发送命令（如 /hn、/weather），Runner 在你的电脑上执行对应程序并返回结果。",
		Guide: "## Runner 使用指南\n\n你已在 Hub 上安装了 Runner App。下面将你的本地命令连接到这个 Bot。\n\n### 1. 复制上方的 Token\n\n页面上方显示的 Token 是 Runner 连接此 Bot 的凭证，请先复制。\n\n### 2. 安装 Runner CLI\n\n```bash\nnpm install -g openilink-app-runner\n```\n\n### 3. 初始化配置\n\n```bash\nopenilink-app-runner init --hub-url {hub_url} --token {your_token}\n```\n\n### 4. 添加命令\n\n```bash\n# 添加 HackerNews 热门\nopenilink-app-runner add hn \"opencli hackernews top --format json\" --desc \"HackerNews 热门\"\n\n# 添加天气查询\nopenilink-app-runner add weather \"curl -s 'wttr.in/${args}?format=3'\" --desc \"查天气\"\n\n# 查看已添加的命令\nopenilink-app-runner list\n```\n\n命令会自动同步到 Hub，无需手动配置。\n\n### 5. 启动\n\n```bash\nopenilink-app-runner start\n```\n\n启动后，在微信中发送 `@runner /hn` 或 `@runner /weather 上海` 即可执行。\n\n### CLI 命令\n\n| 命令 | 说明 |\n|------|------|\n| `init` | 初始化配置 |\n| `add <name> <exec>` | 添加命令 |\n| `remove <name>` | 删除命令 |\n| `list` | 查看命令列表 |\n| `sync` | 手动同步到 Hub |\n| `start` | 启动 Runner |\n\n### 更多信息\n\n- [Runner 源码](https://github.com/openilink/openilink-app-runner)\n- [npm 包](https://www.npmjs.com/package/openilink-app-runner)",
		Scopes:      []string{"message:read", "message:write", "tools:write"},
		Events:      []string{"command"},
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {}
		}`),
	}, nil) // nil handler — events delivered via WS to local runner
}
