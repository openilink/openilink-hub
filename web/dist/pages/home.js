import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { HexagonBackground } from "../components/ui/hexagon-background";
import { cn } from "../lib/utils";
import { Bot, Webhook, Cable, Shield, Zap } from "lucide-react";
import { useUser } from "@/hooks/use-auth";
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
  { icon: Zap, title: "AI 自动回复", desc: "内置 OpenAI 兼容的 AI 回复，按规则独立开关" },
  {
    icon: Shield,
    title: "安全沙箱",
    desc: "脚本在安全沙箱中执行，5 秒超时、栈深限制、禁止危险 API",
  },
];
export function HomePage() {
  const [isScrolled, setIsScrolled] = useState(false);
  const { data: user, isLoading } = useUser();
  const loggedIn = isLoading ? null : !!user;
  useEffect(() => {
    const updateScrollState = () => {
      setIsScrolled(window.scrollY > 32);
    };
    updateScrollState();
    window.addEventListener("scroll", updateScrollState, { passive: true });
    return () => window.removeEventListener("scroll", updateScrollState);
  }, []);
  return _jsxs("div", {
    className: "relative isolate flex min-h-screen flex-col overflow-x-hidden bg-background",
    children: [
      _jsx(HexagonBackground, { className: "opacity-90", hexagonSize: 84, hexagonMargin: 5 }),
      _jsx("div", {
        className: cn(
          "absolute inset-x-0 top-0 h-[34rem] bg-[radial-gradient(circle_at_top,rgba(255,255,255,0.18),transparent_42%),radial-gradient(circle_at_50%_12%,rgba(255,255,255,0.08),transparent_50%)] transition-opacity duration-500",
          isScrolled ? "opacity-40" : "opacity-100",
        ),
      }),
      _jsx("div", {
        className:
          "absolute inset-0 bg-[linear-gradient(to_bottom,rgba(0,0,0,0.04),transparent_20%,transparent_80%,rgba(0,0,0,0.18))]",
      }),
      _jsxs("header", {
        className: "sticky top-0 z-20 px-6 py-4 sm:px-8 lg:px-12",
        children: [
          _jsxs("div", {
            className: cn(
              "mx-auto flex max-w-7xl items-center justify-between rounded-full px-4 sm:px-5 transition-all duration-300",
              isScrolled
                ? "border border-white/8 bg-[linear-gradient(to_bottom,rgba(10,10,10,0.76),rgba(18,18,18,0.56))] py-2.5 shadow-[0_14px_44px_rgba(0,0,0,0.34)] backdrop-blur-xl"
                : "border border-white/5 bg-[linear-gradient(to_bottom,rgba(255,255,255,0.05),rgba(255,255,255,0.015))] py-3 shadow-[0_10px_30px_rgba(0,0,0,0.16)] backdrop-blur-md",
            ),
            children: [
              _jsx("span", {
                className: "text-base font-semibold tracking-tight",
                children: "OpeniLink Hub",
              }),
              _jsx("div", {
                className: "flex items-center gap-3",
                children: loggedIn
                  ? _jsx(Link, {
                      to: "/dashboard",
                      children: _jsx(Button, {
                        size: "sm",
                        className: "px-3 text-sm",
                        children: "\u8FDB\u5165\u63A7\u5236\u53F0",
                      }),
                    })
                  : _jsx(Link, {
                      to: "/login",
                      children: _jsx(Button, {
                        size: "sm",
                        className: "px-3 text-sm",
                        children: "\u767B\u5F55",
                      }),
                    }),
              }),
            ],
          }),
          _jsx("div", {
            className: cn(
              "mx-auto mt-3 h-px max-w-6xl bg-gradient-to-r from-transparent to-transparent transition-all duration-300",
              isScrolled ? "via-white/16 opacity-100" : "via-white/10 opacity-70",
            ),
          }),
        ],
      }),
      _jsxs("main", {
        className: "relative z-10 flex-1 space-y-8 pb-20 sm:space-y-10 sm:pb-24",
        children: [
          _jsxs("section", {
            className: "mx-auto max-w-3xl px-6 py-20 text-center sm:px-8 sm:py-24 lg:py-28",
            children: [
              _jsx("h1", {
                className: "text-4xl font-bold tracking-tight sm:text-5xl",
                children: "OpeniLink Hub",
              }),
              _jsx("p", {
                className: "mt-5 text-base leading-8 text-muted-foreground sm:text-lg",
                children:
                  "\u5F00\u6E90\u7684\u5FAE\u4FE1 Bot \u7BA1\u7406\u4E0E\u6D88\u606F\u4E2D\u7EE7\u5E73\u53F0\u3002\u8FDE\u63A5\u4F60\u7684\u5FAE\u4FE1\uFF0C\u901A\u8FC7 WebSocket\u3001HTTP API \u6216 Webhook \u63A5\u6536\u548C\u53D1\u9001\u6D88\u606F\u3002",
              }),
              _jsxs("div", {
                className: "mt-8 flex justify-center gap-4 sm:mt-10",
                children: [
                  _jsx(Link, {
                    to: loggedIn ? "/dashboard" : "/login",
                    children: _jsx(Button, {
                      size: "lg",
                      className: "px-5 text-sm",
                      children: "\u5F00\u59CB\u4F7F\u7528",
                    }),
                  }),
                  _jsx("a", {
                    href: "https://github.com/openilink/openilink-hub",
                    target: "_blank",
                    rel: "noopener",
                    children: _jsx(Button, {
                      variant: "outline",
                      size: "lg",
                      className: "px-5 text-sm",
                      children: "GitHub",
                    }),
                  }),
                ],
              }),
            ],
          }),
          _jsx("section", {
            className: "mx-auto max-w-6xl px-6 sm:px-8",
            children: _jsx("div", {
              className: "grid grid-cols-1 gap-5 sm:grid-cols-2 sm:gap-6 lg:grid-cols-3",
              children: features.map((f) =>
                _jsxs(
                  Card,
                  {
                    className:
                      "rounded-2xl border border-white/8 bg-card/75 p-6 backdrop-blur-sm sm:p-7 space-y-4",
                    children: [
                      _jsxs("div", {
                        className: "flex items-center gap-3",
                        children: [
                          _jsx(f.icon, { className: "h-5 w-5 text-primary" }),
                          _jsx("h3", {
                            className: "text-base font-semibold tracking-tight",
                            children: f.title,
                          }),
                        ],
                      }),
                      _jsx("p", {
                        className: "text-sm leading-7 text-muted-foreground",
                        children: f.desc,
                      }),
                    ],
                  },
                  f.title,
                ),
              ),
            }),
          }),
          _jsxs("section", {
            className: "mx-auto max-w-3xl px-6 pt-6 sm:px-8 sm:pt-8",
            children: [
              _jsx("h2", {
                className: "mb-10 text-center text-2xl font-semibold tracking-tight",
                children: "\u5DE5\u4F5C\u6D41\u7A0B",
              }),
              _jsx("div", {
                className: "space-y-7 text-base",
                children: [
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
                ].map((s) =>
                  _jsxs(
                    "div",
                    {
                      className: "flex items-start gap-4 sm:gap-5",
                      children: [
                        _jsx("span", {
                          className:
                            "flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary text-sm font-bold text-primary-foreground sm:h-9 sm:w-9",
                          children: s.step,
                        }),
                        _jsxs("div", {
                          className: "space-y-1.5 pt-0.5",
                          children: [
                            _jsx("p", {
                              className: "text-lg font-semibold tracking-tight",
                              children: s.title,
                            }),
                            _jsx("p", {
                              className: "text-sm leading-7 text-muted-foreground sm:text-base",
                              children: s.desc,
                            }),
                          ],
                        }),
                      ],
                    },
                    s.step,
                  ),
                ),
              }),
            ],
          }),
        ],
      }),
      _jsxs("footer", {
        className:
          "relative z-10 border-t border-white/8 px-6 py-6 text-center text-sm text-muted-foreground sm:px-8",
        children: [
          _jsx("a", {
            href: "https://github.com/openilink/openilink-hub",
            target: "_blank",
            rel: "noopener",
            className: "hover:text-primary",
            children: "OpeniLink Hub",
          }),
          " · ",
          "\u5F00\u6E90\u5FAE\u4FE1\u6D88\u606F\u5E73\u53F0",
        ],
      }),
    ],
  });
}
