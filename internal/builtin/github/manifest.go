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

### 1. 安装应用

在上方页面点击安装到你的 Bot。

### 2. 配置 Webhook Secret（可选）

在安装配置中填写一个 Secret，用于验证 GitHub 发来的请求。

### 3. 在 GitHub 仓库添加 Webhook

进入你的 GitHub 仓库 → Settings → Webhooks → Add webhook：

- **Payload URL**: ` + "`" + `{hub_url}/api/hooks/github?token={your_token}` + "`" + `
- **Content type**: ` + "`application/json`" + `
- **Secret**: 填写你在第 2 步配置的 Secret（如未配置则留空）
- **Events**: 选择你想接收的事件，推荐 "Let me select individual events" 然后勾选：
  - Pushes
  - Pull requests
  - Issues
  - Issue comments
  - Releases
  - Workflow runs

### 4. 完成

GitHub 事件将自动推送到你的微信。

### 支持的事件

| 事件 | 说明 |
|------|------|
| push | 代码推送 |
| pull_request | PR 创建/合并/关闭 |
| issues | Issue 创建/关闭 |
| issue_comment | Issue/PR 评论 |
| release | 发布新版本 |
| workflow_run | CI/CD 完成 |
| create | 创建分支/标签 |
| delete | 删除分支/标签 |
| star | 仓库 Star |
| fork | 仓库 Fork |
| ping | Webhook 测试 |`,
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
