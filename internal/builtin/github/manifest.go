package github

import (
	"encoding/json"

	"github.com/openilink/openilink-hub/internal/builtin"
)

func init() {
	builtin.Register(builtin.AppManifest{
		Slug:        "github",
		Name:        "GitHub",
		Description: "接收 GitHub 仓库事件通知（Push、PR、Issue、CI 等）",
		Icon:        "🐙",
		Readme:      "将 GitHub 仓库的 Webhook 事件推送到微信。支持 Push、Pull Request、Issues、CI 等常见事件。",
		Guide: `## GitHub 通知

### 1. 在 GitHub 仓库添加 Webhook

进入你的 GitHub 仓库 → **Settings** → **Webhooks** → **Add webhook**，填写以下信息：

- **Payload URL**:

` + "```" + `
{hub_url}/api/hooks/github?token={your_token}
` + "```" + `

- **Content type**: ` + "`application/json`" + `
- **Secret**: 如需签名验证，在下方「配置」中填写相同的 Secret（可选）
- **Which events**: 推荐选择 "Let me select individual events" 然后勾选：
  - Pushes、Pull requests、Issues、Issue comments
  - Releases、Workflow runs

点击 **Add webhook** 保存。

### 2. 验证连接

添加后 GitHub 会自动发送一个 ` + "`ping`" + ` 事件，你的微信应收到：

> 🔔 [owner/repo] Webhook 连接成功

如果没有收到，请检查 Payload URL 是否正确、Bot 是否在线。

### 3. 手动测试（可选）

` + "```bash" + `
curl -X POST "{hub_url}/api/hooks/github?token={your_token}" \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: ping" \
  -d '{"zen":"It works!","repository":{"full_name":"test/repo"},"sender":{"login":"you"}}'
` + "```" + `

### 支持的事件

| 事件 | 推送内容 |
|------|----------|
| push | 📦 代码推送（含 commit 列表） |
| pull_request | 🔀 PR 创建/合并/关闭 |
| issues | 📋 Issue 创建/关闭 |
| issue_comment | 💬 Issue/PR 评论 |
| release | 🚀 发布新版本 |
| workflow_run | ✅/❌ CI 执行结果 |
| create / delete | 🌿/🗑️ 分支/标签操作 |
| star / fork | ⭐/🍴 Star / Fork |`,
		Scopes: []string{"message:write"},
		Events: []string{},
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"secret": {
					"type": "string",
					"title": "Webhook Secret",
					"description": "GitHub Webhook Secret，用于验证请求签名（可选）"
				}
			}
		}`),
	}, nil)
}
