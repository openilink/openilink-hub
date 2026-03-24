<div align="center">

# OpenILink Hub

**微信 ClawBot iLink 协议的开源消息管理平台**<br>
**Open-source message management platform for WeChat ClawBot (iLink protocol)**

扫码绑定微信号，消息实时转发到你的服务 —— 支持 WebSocket / Webhook / AI 自动回复<br>
多 Bot 集中管理 · JavaScript 插件引擎 · 7 种语言 SDK · Passkey 无密码登录

[![License](https://img.shields.io/github/license/openilink/openilink-hub?style=flat-square)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![React](https://img.shields.io/badge/React-19-61DAFB?style=flat-square&logo=react&logoColor=black)](https://react.dev)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat-square&logo=docker&logoColor=white)](docker-compose.yml)
[![GitHub Stars](https://img.shields.io/github/stars/openilink/openilink-hub?style=flat-square&logo=github)](https://github.com/openilink/openilink-hub/stargazers)

[在线体验](https://hub.openilink.com) · [快速开始](#快速开始) · [SDK 文档](#sdk-生态) · [插件市场](#插件系统) · [English](#english)

</div>

---

## 这是什么？

2026 年 3 月，微信正式推出 **ClawBot 插件**，底层协议叫 **iLink（智联）**，接入域名为 `ilinkai.weixin.qq.com` —— 微信官方首次开放个人号的 Bot API，你可以**合法地**让程序收发微信消息了。

**但 iLink 只是一个消息通道**：你扫码、收消息、发回复，仅此而已。要真正用起来，你还需要管理多个 Bot、路由消息到不同服务、处理媒体文件、配置过滤规则……

**OpenILink Hub 就是干这个的。** 它把 iLink 的原始能力包装成一个完整的消息管理平台：

```mermaid
flowchart TD
    subgraph WeChat
        A[微信 ClawBot 插件]
    end
    
    A -->|iLink 协议| B[OpenILink SDK]
    subgraph OpeniLink Capabilities
      C[OpenILink Hub]
    end
    B --> C[OpenILink Hub]
    
    C --> D[WebSocket]
    C --> E[Webhook]
    C --> F[AI 自动回复]
    
    D --> G[实时客户端]
    E --> H[第三方服务]
    F --> I[LLM]
    
    G --> J[业务系统]
    H --> J
    I --> J
    
```

<details>
<summary><b>和 OpenClaw 是什么关系？</b></summary>

OpenClaw 是一个 AI Agent Gateway 框架，微信 ClawBot 插件原生支持对接 OpenClaw。

OpenILink Hub **不依赖 OpenClaw**，它是一个独立的、更通用的消息管理平台。你可以通过 [openclaw-channel-openilink](https://github.com/openilink/openclaw-channel-openilink) 适配器将两者打通，也可以完全不用 OpenClaw，直接用 Hub 对接你自己的服务。

简单说：**OpenClaw 专注 AI Agent，OpenILink Hub 专注消息管理和分发**，两者互补但互不依赖。

</details>

## 为什么选择 OpenILink Hub？

| | OpenILink Hub | 传统方案 |
|---|---|---|
| **部署方式** | `docker compose up -d` 一键启动 | 需要复杂的依赖配置 |
| **多 Bot 管理** | 扫码绑定，集中管理多个微信号 | 通常只支持单个 Bot |
| **消息下发** | WebSocket + Webhook + AI 三通道并行 | 单一通道 |
| **认证方式** | Passkey 无密码 + OAuth + 传统密码 | 仅密码认证 |
| **可扩展性** | JavaScript 插件引擎 + 7 种语言 SDK | 硬编码，难以扩展 |
| **开源协议** | MIT，无商业限制 | 部分闭源或限制商用 |

## 核心特性

**多 Bot 集中管理**
扫描二维码即可绑定微信号，支持同时管理多个 Bot，统一面板监控在线状态与消息统计。

**三通道消息下发**
- **WebSocket** — 毫秒级实时推送，适合需要即时响应的场景
- **Webhook** — HTTP 回调 + JavaScript 中间件，灵活对接任意服务
- **AI 自动回复** — 接入 OpenAI 兼容 API，Bot 自动与用户对话

**JavaScript 插件引擎**
内置 JS 运行时（goja），Webhook 支持 `onRequest` / `onResponse` 两阶段钩子，自定义消息过滤、转换、路由逻辑。提供插件市场，社区贡献开箱即用。

**现代化认证体系**
支持 Passkey（WebAuthn）生物识别 / 硬件密钥无密码登录，同时集成 GitHub、LinuxDo OAuth，多因素安全保障。

**Channel 精细化控制**
每个 Bot 下可创建多个 Channel，独立 API Key、独立过滤规则（用户 / 关键词 / 消息类型），实现消息的精确路由。

**完善的管理后台**
用户管理、角色权限、OAuth 配置、AI 全局设置，管理员一站式掌控。

## 架构总览

```mermaid
flowchart TB
    subgraph Client["客户端"]
        direction TB
        WX[微信号<br/>扫码绑定] --> Provider[Provider<br/>iLink]
        UI[Web UI<br/>React] --> API[REST API<br/>Go]
    end
    
    Provider --> Broker[Message Broker]
    API --> Broker
    
    Broker --> WS[WebSocket<br/>实时客户端]
    Broker --> WH[Webhook<br/>JS 中间件]
    Broker --> AI[AI Sink<br/>自动回复]
    
    subgraph Storage["存储"]
        PG[(PostgreSQL<br/>数据)]
        MinIO[(MinIO<br/>媒体存储)]
        PK[Passkey<br/>WebAuthn]
    end
```

## 快速开始

### Docker Compose（推荐）

```bash
docker compose up -d
```

访问 `http://localhost:9800`，**首个注册用户自动成为管理员**。

### 从源码构建

```bash
# 构建前端
cd web && npm ci && npm run build && cd ..

# 构建并运行
go build -o openilink-hub .
DATABASE_URL="postgres://user:pass@localhost:5432/openilink" \
SECRET="$(openssl rand -hex 32)" \
./openilink-hub
```

### Docker Compose 完整配置

```yaml
services:
  postgres:
    image: postgres:17-alpine
    environment:
      POSTGRES_USER: openilink
      POSTGRES_PASSWORD: <改为强密码>
      POSTGRES_DB: openilink
    volumes:
      - pgdata:/var/lib/postgresql/data

  hub:
    build: .
    ports:
      - "9800:9800"
    environment:
      DATABASE_URL: postgres://openilink:<密码>@postgres:5432/openilink?sslmode=disable
      RP_ORIGIN: https://hub.example.com
      RP_ID: hub.example.com
      SECRET: <随机字符串>
    depends_on:
      - postgres

volumes:
  pgdata:
```

前置 Nginx / Caddy 做 HTTPS 反向代理，将 443 端口转发到 9800。

## SDK 生态

OpenILink 提供 **7 种语言的官方 SDK**，方便你用熟悉的技术栈快速接入：

| 语言 | 仓库 | 安装方式 |
|------|------|---------|
| **Go** | [openilink-sdk-go](https://github.com/openilink/openilink-sdk-go) | `go get github.com/openilink/openilink-sdk-go` |
| **Node.js** | [openilink-sdk-node](https://github.com/openilink/openilink-sdk-node) | `npm install @openilink/openilink-sdk-node` |
| **Python** | [openilink-sdk-python](https://github.com/openilink/openilink-sdk-python) | `pip install openilink-sdk-python` |
| **PHP** | [openilink-sdk-php](https://github.com/openilink/openilink-sdk-php) | `composer require openilink/openilink-sdk-php` |
| **Java** | [openilink-sdk-java](https://github.com/openilink/openilink-sdk-java) | 从源码构建 |
| **C#** | [openilink-sdk-csharp](https://github.com/openilink/openilink-sdk-csharp) | 开发中 |
| **Lua** | [openilink-sdk-lua](https://github.com/openilink/openilink-sdk-lua) | 从源码引入 |

### 相关项目

| 项目 | 说明 |
|------|------|
| [openilink-tg](https://github.com/openilink/openilink-tg) | Telegram Bot 集成，微信消息转发到 Telegram |
| [openilink-webhook-plugins](https://github.com/openilink/openilink-webhook-plugins) | 官方 Webhook 插件仓库，社区贡献的开箱即用插件 |
| [openclaw-channel-openilink](https://github.com/openilink/openclaw-channel-openilink) | OpenClaw 平台的 OpenILink 适配器 |

## 插件系统

Webhook 支持 JavaScript 插件，通过 `onRequest` / `onResponse` 钩子实现消息的自定义处理：

```javascript
// @name         消息转发到飞书
// @description  将微信消息格式化后转发到飞书群
// @match        text
// @connect      open.feishu.cn
// @grant        skip

function onRequest(ctx) {
  ctx.method = "POST";
  ctx.url = "https://open.feishu.cn/open-apis/bot/v2/hook/xxx";
  ctx.headers["Content-Type"] = "application/json";
  ctx.body = JSON.stringify({
    msg_type: "text",
    content: { text: ctx.message.text }
  });
}
```

**插件权限声明**：`@match` 控制消息类型过滤，`@connect` 限制可访问的域名，`@grant` 声明功能权限（`reply` | `skip` | `none`）。

内置插件市场支持社区提交、管理员审核，详见 [Webhook 插件开发文档](docs/webhook-plugin-skill.md)。

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `LISTEN` | `:9800` | 监听地址 |
| `DATABASE_URL` | `postgres://localhost:5432/openilink` | PostgreSQL 连接串 |
| `RP_ORIGIN` | `http://localhost:9800` | 站点源地址（必须与浏览器访问地址一致） |
| `RP_ID` | `localhost` | WebAuthn RP ID，通常为域名 |
| `SECRET` | `change-me-in-production` | 服务端密钥，**生产环境必须修改** |
| `GITHUB_CLIENT_ID` | — | GitHub OAuth Client ID |
| `GITHUB_CLIENT_SECRET` | — | GitHub OAuth Client Secret |
| `LINUXDO_CLIENT_ID` | — | LinuxDo OAuth Client ID |
| `LINUXDO_CLIENT_SECRET` | — | LinuxDo OAuth Client Secret |
| `STORAGE_ENDPOINT` | — | MinIO / S3 兼容存储端点 |
| `STORAGE_ACCESS_KEY` | — | 存储访问密钥 |
| `STORAGE_SECRET_KEY` | — | 存储密钥 |
| `STORAGE_BUCKET` | — | 存储桶名称 |
| `STORAGE_PUBLIC_URL` | — | 存储公开访问 URL |

## 配置 OAuth 登录

OAuth 为可选功能，配置后用户可使用第三方账号登录或绑定到已有账号。

<details>
<summary><b>GitHub OAuth</b></summary>

1. 前往 [GitHub Developer Settings](https://github.com/settings/developers) → OAuth Apps → New OAuth App
2. 填写：
   - **Homepage URL**: `https://hub.example.com`
   - **Authorization callback URL**: `https://hub.example.com/api/auth/oauth/github/callback`
3. 获取 Client ID 和 Client Secret，设置对应环境变量

</details>

<details>
<summary><b>LinuxDo OAuth</b></summary>

1. 前往 [connect.linux.do](https://connect.linux.do) 创建应用
2. 回调地址：`https://hub.example.com/api/auth/oauth/linuxdo/callback`
3. 获取 Client ID 和 Client Secret，设置对应环境变量

</details>

> 回调地址格式：`{RP_ORIGIN}/api/auth/oauth/{provider}/callback`，`RP_ORIGIN` 必须与实际访问地址完全一致。

## Provider 扩展

Bot 连接通过 Provider 接口抽象（`internal/provider/`），当前实现了 iLink Provider。新增 Provider 只需三步：

1. 在 `internal/provider/<name>/` 下实现 `provider.Provider` 接口
2. 在 `init()` 中调用 `provider.Register("name", factory)`
3. 在 `main.go` 中 `import _ ".../<name>"` 注册

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.25, PostgreSQL 17, gorilla/websocket, goja (JS VM) |
| 前端 | React 19, Vite, TypeScript, Tailwind CSS |
| 认证 | WebAuthn (Passkey), OAuth 2.0, 密码 |
| 存储 | MinIO / S3 兼容对象存储 |
| 部署 | Docker, Docker Compose |

## 参与贡献

欢迎提交 Issue 和 Pull Request！

- 插件贡献请提交到 [openilink-webhook-plugins](https://github.com/openilink/openilink-webhook-plugins)
- SDK 问题请到对应语言的仓库反馈

## License

[MIT](LICENSE) — 自由使用，无商业限制。

---

<div align="center">

**[OpenILink](https://openilink.com)** · 让微信 Bot 接入更简单

</div>

---

<a name="english"></a>

## English

**OpenILink Hub** is a self-hosted, open-source WeChat Bot management and message relay platform built on top of the **iLink protocol** — the official WeChat ClawBot Bot API launched in March 2026.

It turns WeChat's raw messaging capability into a manageable, routable, and extensible system: bind multiple WeChat accounts via QR code, then forward messages to your services through WebSocket, Webhook (with JavaScript middleware), or AI auto-reply. It works independently and can also integrate with OpenClaw via the [openclaw-channel-openilink](https://github.com/openilink/openclaw-channel-openilink) adapter.

**Key Features:** Multi-bot management, Passkey (WebAuthn) passwordless login, GitHub/LinuxDo OAuth, JavaScript plugin engine, plugin marketplace, and official SDKs for Go, Node.js, Python, PHP, Java, C#, and Lua.

**Quick Start:** `docker compose up -d` → visit `http://localhost:9800`

For full documentation, see the Chinese sections above or visit [hub.openilink.com](https://hub.openilink.com).
