import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Link } from "react-router-dom";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Bot,
  MessageSquare,
  Zap,
  Plus,
  ArrowRight,
  Wifi,
  Workflow,
  Cpu,
  AlertTriangle,
  TrendingUp,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useBotStats } from "@/hooks/use-bots";
const STEPS = [
  {
    step: "01",
    title: "添加微信账号",
    desc: "扫码登录你的微信，连接到平台。",
    link: "/dashboard/accounts",
    icon: Cpu,
    color: "text-blue-500",
    bg: "bg-blue-500/10",
  },
  {
    step: "02",
    title: "创建转发规则",
    desc: "设置消息转发到你的服务器或 AI。",
    link: "/dashboard/accounts",
    icon: Workflow,
    color: "text-violet-500",
    bg: "bg-violet-500/10",
  },
  {
    step: "03",
    title: "安装应用",
    desc: "从市场安装现成的扩展功能。",
    link: "/dashboard/apps",
    icon: Zap,
    color: "text-orange-500",
    bg: "bg-orange-500/10",
  },
];
const QUICK_LINKS = [
  { label: "全部账号", icon: Bot, link: "/dashboard/accounts" },
  { label: "应用市场", icon: Workflow, link: "/dashboard/apps" },
  { label: "系统设置", icon: Cpu, link: "/dashboard/settings/profile" },
];
export function DashboardOverviewPage() {
  const { data: stats, isLoading: loading } = useBotStats();
  if (loading)
    return _jsxs("div", {
      className: "space-y-6",
      children: [
        _jsxs("div", {
          className: "flex justify-between items-center",
          children: [
            _jsx(Skeleton, { className: "h-5 w-48" }),
            _jsxs("div", {
              className: "flex gap-2",
              children: [
                _jsx(Skeleton, { className: "h-9 w-24" }),
                _jsx(Skeleton, { className: "h-9 w-24" }),
              ],
            }),
          ],
        }),
        _jsx("div", {
          className: "grid gap-4 md:grid-cols-4",
          children: Array.from({ length: 4 }).map((_, i) =>
            _jsx(Skeleton, { className: "h-24 w-full" }, i),
          ),
        }),
        _jsxs("div", {
          className: "grid gap-6 lg:grid-cols-7",
          children: [
            _jsx(Skeleton, { className: "h-64 lg:col-span-4" }),
            _jsx(Skeleton, { className: "h-64 lg:col-span-3" }),
          ],
        }),
      ],
    });
  return _jsxs("div", {
    className: "space-y-6",
    children: [
      _jsxs("div", {
        className: "flex items-start justify-between gap-4",
        children: [
          _jsxs("div", {
            children: [
              _jsx("h1", {
                className: "text-2xl font-bold tracking-tight",
                children: "\u6982\u89C8",
              }),
              _jsx("p", {
                className: "text-sm text-muted-foreground mt-0.5",
                children:
                  "\u67E5\u770B\u8D26\u53F7\u72B6\u6001\u548C\u6D88\u606F\u7EDF\u8BA1\u3002",
              }),
            ],
          }),
          _jsxs("div", {
            className: "flex items-center gap-2 shrink-0",
            children: [
              _jsx(Button, {
                variant: "outline",
                size: "sm",
                className: "h-9 px-4 font-medium",
                asChild: true,
                children: _jsx(Link, {
                  to: "/dashboard/accounts",
                  children: "\u8D26\u53F7\u7BA1\u7406",
                }),
              }),
              _jsx(Button, {
                size: "sm",
                className: "h-9 px-4 gap-1.5 font-medium shadow-sm",
                asChild: true,
                children: _jsxs(Link, {
                  to: "/dashboard/accounts",
                  children: [_jsx(Plus, { className: "h-3.5 w-3.5" }), " \u6DFB\u52A0\u8D26\u53F7"],
                }),
              }),
            ],
          }),
        ],
      }),
      _jsx("div", {
        className: "grid gap-4 md:grid-cols-2 lg:grid-cols-4",
        children: [
          {
            label: "在线账号",
            value: stats?.online_bots ?? 0,
            sub: `共 ${stats?.total_bots ?? 0} 个`,
            icon: Bot,
            color: "text-blue-500",
            bg: "bg-blue-500/10",
            badge: (stats?.online_bots ?? 0) > 0,
            link: "/dashboard/accounts",
          },
          {
            label: "消息总量",
            value: stats?.total_messages ?? 0,
            sub: "历史累计",
            icon: MessageSquare,
            color: "text-emerald-500",
            bg: "bg-emerald-500/10",
            badge: false,
            link: null,
          },
          {
            label: "已安装应用",
            value: stats?.total_installations ?? 0,
            sub: "个插件",
            icon: Workflow,
            color: "text-violet-500",
            bg: "bg-violet-500/10",
            badge: false,
            link: null,
          },
          {
            label: "WebSocket 连接",
            value: stats?.connected_ws ?? 0,
            sub: "活跃连接",
            icon: Wifi,
            color: "text-orange-500",
            bg: "bg-orange-500/10",
            badge: false,
            link: null,
          },
        ].map((m, i) =>
          _jsx(
            Card,
            {
              className: `border-border/50 bg-card/50 hover:bg-card transition-colors${m.link ? " cursor-pointer" : " cursor-default"}`,
              ...(m.link ? { onClick: undefined } : {}),
              children: m.link
                ? _jsx(Link, {
                    to: m.link,
                    className: "block",
                    children: _jsxs(CardContent, {
                      className: "p-5",
                      children: [
                        _jsxs("div", {
                          className: "flex items-start justify-between mb-3",
                          children: [
                            _jsx("div", {
                              className: `h-8 w-8 rounded-lg ${m.bg} flex items-center justify-center ${m.color}`,
                              children: _jsx(m.icon, { className: "h-4 w-4" }),
                            }),
                            m.badge
                              ? _jsx(Badge, {
                                  variant: "outline",
                                  className:
                                    "text-[10px] h-5 px-1.5 text-emerald-600 border-emerald-200 bg-emerald-50 dark:bg-emerald-950/30 dark:border-emerald-800 dark:text-emerald-400",
                                  children: "\u5728\u7EBF",
                                })
                              : null,
                          ],
                        }),
                        _jsxs("div", {
                          className: "space-y-0.5",
                          children: [
                            _jsx("div", {
                              className: "text-2xl font-bold tabular-nums",
                              children: m.value.toLocaleString(),
                            }),
                            _jsxs("div", {
                              className: "flex items-baseline gap-1.5",
                              children: [
                                _jsx("p", {
                                  className: "text-xs font-semibold text-foreground/80",
                                  children: m.label,
                                }),
                                _jsx("span", {
                                  className: "text-[10px] text-muted-foreground",
                                  children: m.sub,
                                }),
                              ],
                            }),
                          ],
                        }),
                      ],
                    }),
                  })
                : _jsxs(CardContent, {
                    className: "p-5",
                    children: [
                      _jsx("div", {
                        className: "flex items-start justify-between mb-3",
                        children: _jsx("div", {
                          className: `h-8 w-8 rounded-lg ${m.bg} flex items-center justify-center ${m.color}`,
                          children: _jsx(m.icon, { className: "h-4 w-4" }),
                        }),
                      }),
                      _jsxs("div", {
                        className: "space-y-0.5",
                        children: [
                          _jsx("div", {
                            className: "text-2xl font-bold tabular-nums",
                            children: m.value.toLocaleString(),
                          }),
                          _jsxs("div", {
                            className: "flex items-baseline gap-1.5",
                            children: [
                              _jsx("p", {
                                className: "text-xs font-semibold text-foreground/80",
                                children: m.label,
                              }),
                              _jsx("span", {
                                className: "text-[10px] text-muted-foreground",
                                children: m.sub,
                              }),
                            ],
                          }),
                        ],
                      }),
                    ],
                  }),
            },
            i,
          ),
        ),
      }),
      (stats?.online_bots ?? 0) === 0
        ? _jsxs("div", {
            className:
              "flex items-center gap-3 p-4 rounded-lg border border-destructive/20 bg-destructive/5",
            children: [
              _jsx(AlertTriangle, { className: "h-4 w-4 text-destructive shrink-0" }),
              _jsxs("div", {
                className: "flex-1 min-w-0",
                children: [
                  _jsx("p", {
                    className: "text-sm font-medium text-destructive",
                    children: "\u6682\u65E0\u5728\u7EBF\u8D26\u53F7",
                  }),
                  _jsx("p", {
                    className: "text-xs text-destructive/70 mt-0.5",
                    children:
                      "\u8FD8\u6CA1\u6709\u5728\u7EBF\u7684\u5FAE\u4FE1\u8D26\u53F7\uFF0C\u8BF7\u5148\u6DFB\u52A0\u4E00\u4E2A\u3002",
                  }),
                ],
              }),
              _jsx(Button, {
                size: "sm",
                variant: "destructive",
                className: "shrink-0 h-8 px-3 text-xs",
                asChild: true,
                children: _jsx(Link, {
                  to: "/dashboard/accounts",
                  children: "\u7ACB\u5373\u6DFB\u52A0",
                }),
              }),
            ],
          })
        : null,
      _jsxs("div", {
        className: "grid gap-6 lg:grid-cols-7",
        children: [
          _jsxs(Card, {
            className: "lg:col-span-4 border-border/50",
            children: [
              _jsx(CardHeader, {
                className: "pb-3",
                children: _jsxs("div", {
                  className: "flex items-center justify-between",
                  children: [
                    _jsxs("div", {
                      children: [
                        _jsx(CardTitle, {
                          className: "text-base font-semibold",
                          children: "\u5FEB\u901F\u5F00\u59CB",
                        }),
                        _jsx(CardDescription, {
                          className: "text-xs mt-0.5",
                          children: "\u4E09\u6B65\u5F00\u59CB\u4F7F\u7528 OpeniLink",
                        }),
                      ],
                    }),
                    _jsx("div", {
                      className:
                        "h-8 w-8 rounded-lg bg-primary/10 flex items-center justify-center text-primary",
                      children: _jsx(TrendingUp, { className: "h-4 w-4" }),
                    }),
                  ],
                }),
              }),
              _jsx(CardContent, {
                className: "p-0",
                children: _jsx("div", {
                  className: "divide-y divide-border/50",
                  children: STEPS.map((item, i) =>
                    _jsxs(
                      Link,
                      {
                        to: item.link,
                        className:
                          "flex items-center gap-4 px-6 py-4 hover:bg-muted/40 transition-colors group",
                        children: [
                          _jsx("span", {
                            className:
                              "text-xs font-bold tabular-nums text-muted-foreground/30 group-hover:text-muted-foreground/60 transition-colors w-5 shrink-0",
                            children: item.step,
                          }),
                          _jsx("div", {
                            className: `h-9 w-9 rounded-lg ${item.bg} flex items-center justify-center shrink-0 transition-colors`,
                            children: _jsx(item.icon, { className: `h-4 w-4 ${item.color}` }),
                          }),
                          _jsxs("div", {
                            className: "flex-1 min-w-0",
                            children: [
                              _jsx("h4", {
                                className:
                                  "text-sm font-semibold group-hover:text-primary transition-colors",
                                children: item.title,
                              }),
                              _jsx("p", {
                                className: "text-xs text-muted-foreground mt-0.5",
                                children: item.desc,
                              }),
                            ],
                          }),
                          _jsx(ArrowRight, {
                            className:
                              "h-4 w-4 text-muted-foreground/30 group-hover:text-primary group-hover:translate-x-0.5 transition-all shrink-0",
                          }),
                        ],
                      },
                      i,
                    ),
                  ),
                }),
              }),
            ],
          }),
          _jsx("div", {
            className: "lg:col-span-3 space-y-4",
            children: _jsxs(Card, {
              className: "border-border/50",
              children: [
                _jsxs(CardHeader, {
                  className: "pb-3",
                  children: [
                    _jsx(CardTitle, {
                      className: "text-base font-semibold",
                      children: "\u5FEB\u6377\u5165\u53E3",
                    }),
                    _jsx(CardDescription, {
                      className: "text-xs",
                      children: "\u5E38\u7528\u529F\u80FD\u76F4\u8FBE",
                    }),
                  ],
                }),
                _jsx(CardContent, {
                  className: "grid grid-cols-2 gap-2 pt-0",
                  children: QUICK_LINKS.map((item) =>
                    _jsx(
                      Button,
                      {
                        variant: "outline",
                        className: "h-auto flex-col gap-2 py-4 justify-start items-center",
                        asChild: true,
                        children: _jsxs(Link, {
                          to: item.link,
                          children: [
                            _jsx(item.icon, { className: "h-5 w-5 text-muted-foreground" }),
                            _jsx("span", {
                              className: "text-xs font-medium",
                              children: item.label,
                            }),
                          ],
                        }),
                      },
                      item.link,
                    ),
                  ),
                }),
              ],
            }),
          }),
        ],
      }),
    ],
  });
}
