import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { HexagonBackground } from "../components/ui/hexagon-background";
import { cn } from "../lib/utils";
import { Bot, Puzzle, Webhook, Cable, Shield, Zap } from "lucide-react";
import { api } from "../lib/api";

const features = [
  { icon: Bot, title: "多账号管理", desc: "同时管理多个微信账号，独立运行、独立配置" },
  {
    icon: Cable,
    title: "消息转发",
    desc: "通过 @提及 将消息转发到不同服务，支持 WebSocket 和 HTTP API",
  },
  {
    icon: Webhook,
    title: "Webhook 推送",
    desc: "收到消息自动推送到你的服务，支持自定义脚本中间件",
  },
  {
    icon: Puzzle,
    title: "插件市场",
    desc: "社区驱动的 Webhook 插件市场，一键安装飞书、Slack 等通知集成",
  },
  { icon: Zap, title: "AI 自动回复", desc: "内置 OpenAI 兼容的 AI 回复，按规则独立开关" },
  {
    icon: Shield,
    title: "安全沙箱",
    desc: "插件脚本在安全沙箱中执行，5 秒超时、栈深限制、禁止危险 API",
  },
];

export function HomePage() {
  const [isScrolled, setIsScrolled] = useState(false);
  const [loggedIn, setLoggedIn] = useState<boolean | null>(null);

  useEffect(() => {
    api
      .me()
      .then(() => setLoggedIn(true))
      .catch(() => setLoggedIn(false));
  }, []);

  useEffect(() => {
    const updateScrollState = () => {
      setIsScrolled(window.scrollY > 32);
    };

    updateScrollState();
    window.addEventListener("scroll", updateScrollState, { passive: true });

    return () => window.removeEventListener("scroll", updateScrollState);
  }, []);

  return (
    <div className="relative isolate flex min-h-screen flex-col overflow-x-hidden bg-background">
      <HexagonBackground className="opacity-90" hexagonSize={84} hexagonMargin={5} />

      <div
        className={cn(
          "absolute inset-x-0 top-0 h-[34rem] bg-[radial-gradient(circle_at_top,rgba(255,255,255,0.18),transparent_42%),radial-gradient(circle_at_50%_12%,rgba(255,255,255,0.08),transparent_50%)] transition-opacity duration-500",
          isScrolled ? "opacity-40" : "opacity-100",
        )}
      />
      <div className="absolute inset-0 bg-[linear-gradient(to_bottom,rgba(0,0,0,0.04),transparent_20%,transparent_80%,rgba(0,0,0,0.18))]" />

      <header className="sticky top-0 z-20 px-6 py-4 sm:px-8 lg:px-12">
        <div
          className={cn(
            "mx-auto flex max-w-7xl items-center justify-between rounded-full px-4 sm:px-5 transition-all duration-300",
            isScrolled
              ? "border border-white/8 bg-[linear-gradient(to_bottom,rgba(10,10,10,0.76),rgba(18,18,18,0.56))] py-2.5 shadow-[0_14px_44px_rgba(0,0,0,0.34)] backdrop-blur-xl"
              : "border border-white/5 bg-[linear-gradient(to_bottom,rgba(255,255,255,0.05),rgba(255,255,255,0.015))] py-3 shadow-[0_10px_30px_rgba(0,0,0,0.16)] backdrop-blur-md",
          )}
        >
          <span className="text-base font-semibold tracking-tight">OpeniLink Hub</span>
          <div className="flex items-center gap-3">
            {loggedIn ? (
              <Link to="/dashboard">
                <Button size="sm" className="px-3 text-sm">
                  进入控制台
                </Button>
              </Link>
            ) : (
              <Link to="/login">
                <Button size="sm" className="px-3 text-sm">
                  登录
                </Button>
              </Link>
            )}
          </div>
        </div>
        <div
          className={cn(
            "mx-auto mt-3 h-px max-w-6xl bg-gradient-to-r from-transparent to-transparent transition-all duration-300",
            isScrolled ? "via-white/16 opacity-100" : "via-white/10 opacity-70",
          )}
        />
      </header>

      <main className="relative z-10 flex-1 space-y-8 pb-20 sm:space-y-10 sm:pb-24">
        {/* Hero */}
        <section className="mx-auto max-w-3xl px-6 py-20 text-center sm:px-8 sm:py-24 lg:py-28">
          <h1 className="text-4xl font-bold tracking-tight sm:text-5xl">OpeniLink Hub</h1>
          <p className="mt-5 text-base leading-8 text-muted-foreground sm:text-lg">
            开源的微信 Bot 管理与消息中继平台。连接你的微信，通过 WebSocket、HTTP API 或 Webhook
            接收和发送消息。
          </p>
          <div className="mt-8 flex justify-center gap-4 sm:mt-10">
            <Link to={loggedIn ? "/dashboard" : "/login"}>
              <Button size="lg" className="px-5 text-sm">
                开始使用
              </Button>
            </Link>
            <a href="https://github.com/openilink/openilink-hub" target="_blank" rel="noopener">
              <Button variant="outline" size="lg" className="px-5 text-sm">
                GitHub
              </Button>
            </a>
          </div>
        </section>

        {/* Features */}
        <section className="mx-auto max-w-6xl px-6 sm:px-8">
          <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 sm:gap-6 lg:grid-cols-3">
            {features.map((f) => (
              <Card
                key={f.title}
                className="rounded-2xl border border-white/8 bg-card/75 p-6 backdrop-blur-sm sm:p-7 space-y-4"
              >
                <div className="flex items-center gap-3">
                  <f.icon className="h-5 w-5 text-primary" />
                  <h3 className="text-base font-semibold tracking-tight">{f.title}</h3>
                </div>
                <p className="text-sm leading-7 text-muted-foreground">{f.desc}</p>
              </Card>
            ))}
          </div>
        </section>

        {/* How it works */}
        <section className="mx-auto max-w-3xl px-6 pt-6 sm:px-8 sm:pt-8">
          <h2 className="mb-10 text-center text-2xl font-semibold tracking-tight">工作流程</h2>
          <div className="space-y-7 text-base">
            {[
              {
                step: "1",
                title: "扫码登录",
                desc: "在 Hub 中扫码登录你的微信账号",
              },
              {
                step: "2",
                title: "创建转发规则",
                desc: "创建转发规则，每条规则有独立的 API Key",
              },
              {
                step: "3",
                title: "接入集成",
                desc: "通过 WebSocket 实时监听，HTTP API 轮询，或 Webhook 推送接收消息",
              },
              {
                step: "4",
                title: "安装插件",
                desc: "从插件市场一键安装飞书、Slack 等通知插件",
              },
            ].map((s) => (
              <div key={s.step} className="flex items-start gap-4 sm:gap-5">
                <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary text-sm font-bold text-primary-foreground sm:h-9 sm:w-9">
                  {s.step}
                </span>
                <div className="space-y-1.5 pt-0.5">
                  <p className="text-lg font-semibold tracking-tight">{s.title}</p>
                  <p className="text-sm leading-7 text-muted-foreground sm:text-base">{s.desc}</p>
                </div>
              </div>
            ))}
          </div>
        </section>
      </main>

      <footer className="relative z-10 border-t border-white/8 px-6 py-6 text-center text-sm text-muted-foreground sm:px-8">
        <a
          href="https://github.com/openilink/openilink-hub"
          target="_blank"
          rel="noopener"
          className="hover:text-primary"
        >
          OpeniLink Hub
        </a>
        {" · "}开源微信消息平台
      </footer>
    </div>
  );
}
