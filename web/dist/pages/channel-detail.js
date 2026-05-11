import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState } from "react";
import { useParams, Link, useNavigate, useLocation } from "react-router-dom";
import {
  Copy,
  Cable,
  Bot as BotIcon,
  Webhook,
  RotateCw,
  Trash2,
  Activity,
  Terminal,
  Loader2,
  Info,
  ChevronRight,
  Filter,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { api, botDisplayName } from "../lib/api";
import { useBot, useBotChannels, useWebhookLogs } from "@/hooks/use-bots";
import { useBotPush } from "@/lib/ws";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useToast } from "@/hooks/use-toast";
import { Skeleton } from "@/components/ui/skeleton";
import { useConfirm } from "@/components/ui/confirm-dialog";
export function ChannelDetailPage() {
  const { id: botId, cid: channelId } = useParams();
  const navigate = useNavigate();
  const location = useLocation();
  const { toast } = useToast();
  const { confirm, ConfirmDialog } = useConfirm();
  const activeTab = location.pathname.split("/").pop() || "overview";
  const { data: bot, isLoading: botLoading, isError: botError } = useBot(botId || "");
  const {
    data: channels,
    isLoading: channelsLoading,
    isError: channelsError,
    refetch: refetchChannels,
  } = useBotChannels(botId || "");
  const channel = (channels || []).find((c) => c.id === channelId) || null;
  const loading = botLoading || channelsLoading;
  const fetchError = botError || channelsError;
  function load() {
    refetchChannels();
  }
  async function handleDelete() {
    const ok = await confirm({
      title: "删除确认",
      description: "确定要删除此转发规则？API Key 将失效。",
      confirmText: "删除",
      variant: "destructive",
    });
    if (!ok) return;
    try {
      await api.deleteChannel(botId, channelId);
      toast({ title: "已删除" });
      navigate(`/dashboard/accounts/${botId}/channels`);
    } catch (e) {
      toast({ variant: "destructive", title: "删除失败", description: e.message });
    }
  }
  async function handleToggle() {
    try {
      const nextStatus = !channel.enabled;
      await api.updateChannel(botId, channelId, { enabled: nextStatus });
      toast({ title: nextStatus ? "已启用" : "已停用" });
      load();
    } catch (e) {
      toast({ variant: "destructive", title: "操作失败", description: e.message });
    }
  }
  if (loading)
    return _jsxs("div", {
      className: "space-y-6",
      children: [
        _jsx(Skeleton, { className: "h-12 w-[300px]" }),
        _jsx(Skeleton, { className: "h-64 w-full" }),
      ],
    });
  if (!channel || !bot)
    return _jsxs("div", {
      className: "flex flex-col items-center justify-center py-20 text-center",
      children: [
        _jsx(Info, { className: "h-10 w-10 text-muted-foreground opacity-20 mb-4" }),
        _jsx("p", {
          className: "text-sm font-medium text-muted-foreground",
          children: fetchError ? "加载失败" : "未找到",
        }),
        _jsx(Button, {
          variant: "link",
          asChild: true,
          children: _jsx(Link, {
            to: `/dashboard/accounts/${botId}/channels`,
            children: "\u8FD4\u56DE\u5217\u8868",
          }),
        }),
      ],
    });
  return _jsxs("div", {
    className: "flex flex-col gap-6",
    children: [
      ConfirmDialog,
      _jsxs("div", {
        className: "flex flex-col gap-4 md:flex-row md:items-center md:justify-between",
        children: [
          _jsxs("div", {
            className: "flex items-center gap-4",
            children: [
              _jsx("div", {
                className:
                  "flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10 text-primary shrink-0 shadow-sm border border-primary/20",
                children: _jsx(Cable, { className: "h-6 w-6" }),
              }),
              _jsxs("div", {
                className: "space-y-1",
                children: [
                  _jsxs("div", {
                    className: "flex items-center gap-2",
                    children: [
                      _jsx("h1", {
                        className: "text-2xl font-bold tracking-tight",
                        children: channel.name,
                      }),
                      _jsx(Button, {
                        variant: channel.enabled ? "default" : "outline",
                        size: "sm",
                        className: "h-6 px-2 text-[10px] uppercase font-bold rounded-full",
                        onClick: handleToggle,
                        children: channel.enabled ? "已启用" : "已停用",
                      }),
                    ],
                  }),
                  _jsxs("div", {
                    className: "flex items-center gap-2 text-xs text-muted-foreground font-mono",
                    children: [
                      _jsxs(Link, {
                        to: `/dashboard/accounts/${botId}`,
                        className: "hover:text-primary transition-colors flex items-center gap-1",
                        children: [
                          _jsx(BotIcon, { className: "h-3 w-3" }),
                          " ",
                          botDisplayName(bot),
                        ],
                      }),
                      _jsx(ChevronRight, { className: "h-3 w-3 opacity-30" }),
                      _jsxs("span", { children: ["\u89C4\u5219 ID: ", channel.id.slice(0, 8)] }),
                    ],
                  }),
                ],
              }),
            ],
          }),
          _jsxs("div", {
            className: "flex items-center gap-2",
            children: [
              _jsx(Button, {
                variant: "outline",
                size: "sm",
                onClick: load,
                className: "h-9 w-9 p-0",
                children: _jsx(RotateCw, { className: "h-4 w-4" }),
              }),
              _jsxs(Button, {
                variant: "outline",
                size: "sm",
                onClick: handleDelete,
                className:
                  "h-9 gap-2 text-destructive border-destructive/20 hover:bg-destructive/10",
                children: [
                  _jsx(Trash2, { className: "h-4 w-4" }),
                  " ",
                  _jsx("span", { className: "hidden sm:inline", children: "\u5220\u9664" }),
                ],
              }),
            ],
          }),
        ],
      }),
      _jsxs(Tabs, {
        value: activeTab,
        onValueChange: (v) => navigate(`/dashboard/accounts/${botId}/channel/${channelId}/${v}`),
        className: "space-y-6",
        children: [
          _jsxs(TabsList, {
            className: "bg-muted/50 p-1",
            children: [
              _jsxs(TabsTrigger, {
                value: "overview",
                className: "gap-2 px-4",
                children: [_jsx(Activity, { className: "h-4 w-4" }), " \u6982\u89C8"],
              }),
              _jsxs(TabsTrigger, {
                value: "webhook",
                className: "gap-2 px-4",
                children: [_jsx(Webhook, { className: "h-4 w-4" }), " Webhook"],
              }),
              _jsxs(TabsTrigger, {
                value: "ai",
                className: "gap-2 px-4",
                children: [_jsx(BotIcon, { className: "h-4 w-4" }), " AI \u56DE\u590D"],
              }),
              _jsxs(TabsTrigger, {
                value: "filter",
                className: "gap-2 px-4",
                children: [_jsx(Filter, { className: "h-4 w-4" }), " \u8FC7\u6EE4\u6761\u4EF6"],
              }),
              _jsxs(TabsTrigger, {
                value: "logs",
                className: "gap-2 px-4",
                children: [_jsx(Terminal, { className: "h-4 w-4" }), " \u8BF7\u6C42\u65E5\u5FD7"],
              }),
            ],
          }),
          _jsxs(TabsContent, {
            value: "overview",
            className: "m-0 space-y-6",
            children: [
              _jsxs("div", {
                className: "grid gap-6 md:grid-cols-2 lg:grid-cols-3",
                children: [
                  _jsxs(Card, {
                    className: "border-border/50 bg-card/50",
                    children: [
                      _jsx(CardHeader, {
                        className: "pb-3",
                        children: _jsx(CardTitle, {
                          className:
                            "text-xs font-bold uppercase tracking-widest text-muted-foreground",
                          children: "\u57FA\u672C\u4FE1\u606F",
                        }),
                      }),
                      _jsxs(CardContent, {
                        className: "space-y-4",
                        children: [
                          _jsxs("div", {
                            className: "space-y-1",
                            children: [
                              _jsx("p", {
                                className: "text-xs text-muted-foreground",
                                children: "\u540D\u79F0",
                              }),
                              _jsx("p", {
                                className: "text-sm font-semibold",
                                children: channel.name,
                              }),
                            ],
                          }),
                          _jsxs("div", {
                            className: "space-y-1",
                            children: [
                              _jsx("p", {
                                className: "text-xs text-muted-foreground",
                                children: "@\u63D0\u53CA",
                              }),
                              _jsxs("div", {
                                className: "flex items-center gap-2",
                                children: [
                                  _jsxs("code", {
                                    className: "text-xs bg-muted px-1.5 py-0.5 rounded font-mono",
                                    children: ["@", channel.handle || "null"],
                                  }),
                                  _jsx("p", {
                                    className: "text-[10px] text-muted-foreground italic",
                                    children: channel.handle ? "仅匹配此提及" : "接收所有消息",
                                  }),
                                ],
                              }),
                            ],
                          }),
                        ],
                      }),
                    ],
                  }),
                  _jsxs(Card, {
                    className: "border-border/50 bg-card/50",
                    children: [
                      _jsx(CardHeader, {
                        className: "pb-3",
                        children: _jsx(CardTitle, {
                          className:
                            "text-xs font-bold uppercase tracking-widest text-muted-foreground",
                          children: "API Key",
                        }),
                      }),
                      _jsxs(CardContent, {
                        className: "space-y-4",
                        children: [
                          _jsxs("div", {
                            className: "space-y-1",
                            children: [
                              _jsx("p", {
                                className: "text-xs text-muted-foreground",
                                children: "\u9891\u9053\u5BC6\u94A5",
                              }),
                              _jsxs("div", {
                                className: "flex items-center gap-2",
                                children: [
                                  _jsx("code", {
                                    className:
                                      "text-xs bg-muted px-1.5 py-0.5 rounded font-mono truncate max-w-[140px]",
                                    children: channel.api_key,
                                  }),
                                  _jsx(Button, {
                                    variant: "ghost",
                                    size: "icon",
                                    className: "h-6 w-6",
                                    onClick: () => {
                                      if (!navigator.clipboard?.writeText) {
                                        toast({
                                          variant: "destructive",
                                          title: "复制失败",
                                          description: "当前浏览器不支持自动复制，请手动选中复制",
                                        });
                                        return;
                                      }
                                      navigator.clipboard
                                        .writeText(channel.api_key)
                                        .then(() => {
                                          toast({ title: "已复制 API Key" });
                                        })
                                        .catch(() => {
                                          toast({
                                            variant: "destructive",
                                            title: "复制失败",
                                            description: "请手动选中复制",
                                          });
                                        });
                                    },
                                    children: _jsx(Copy, { className: "h-3.5 w-3.5" }),
                                  }),
                                ],
                              }),
                            ],
                          }),
                          _jsxs("div", {
                            className: "space-y-1",
                            children: [
                              _jsx("p", {
                                className: "text-xs text-muted-foreground",
                                children: "\u63A5\u53E3\u5730\u5740",
                              }),
                              _jsxs("p", {
                                className: "text-[10px] font-mono opacity-70 truncate",
                                children: [window.location.origin, "/api/v1/channels"],
                              }),
                            ],
                          }),
                        ],
                      }),
                    ],
                  }),
                  _jsxs(Card, {
                    className: "border-border/50 bg-card/50 border-primary/20",
                    children: [
                      _jsx(CardHeader, {
                        className: "pb-3",
                        children: _jsx(CardTitle, {
                          className:
                            "text-xs font-bold uppercase tracking-widest text-muted-foreground",
                          children: "\u72B6\u6001",
                        }),
                      }),
                      _jsx(CardContent, {
                        children: _jsxs("div", {
                          className: "flex flex-col gap-2",
                          children: [
                            _jsxs("div", {
                              className: "flex items-center justify-between text-xs",
                              children: [
                                _jsx("span", {
                                  className: "text-muted-foreground",
                                  children: "\u6700\u540E\u6D3B\u8DC3",
                                }),
                                _jsx("span", {
                                  className: "font-medium",
                                  children: channel.updated_at
                                    ? new Date(channel.updated_at * 1000).toLocaleString()
                                    : "从未",
                                }),
                              ],
                            }),
                            _jsxs("div", {
                              className: "flex items-center justify-between text-xs",
                              children: [
                                _jsx("span", {
                                  className: "text-muted-foreground",
                                  children: "Webhook \u72B6\u6001",
                                }),
                                _jsx(Badge, {
                                  variant: channel.webhook_config?.url ? "default" : "secondary",
                                  className: "h-4 text-[9px]",
                                  children: channel.webhook_config?.url ? "已配置" : "未开启",
                                }),
                              ],
                            }),
                            _jsxs("div", {
                              className: "flex items-center justify-between text-xs",
                              children: [
                                _jsx("span", {
                                  className: "text-muted-foreground",
                                  children: "AI \u589E\u5F3A",
                                }),
                                _jsx(Badge, {
                                  variant: channel.ai_config?.enabled ? "default" : "secondary",
                                  className: "h-4 text-[9px]",
                                  children: channel.ai_config?.enabled ? "活跃" : "关闭",
                                }),
                              ],
                            }),
                          ],
                        }),
                      }),
                    ],
                  }),
                ],
              }),
              _jsxs(Card, {
                className: "border-border/50",
                children: [
                  _jsxs(CardHeader, {
                    children: [
                      _jsx(CardTitle, {
                        className: "text-sm",
                        children: "\u63A5\u5165\u65B9\u5F0F",
                      }),
                      _jsx(CardDescription, {
                        children:
                          "\u901A\u8FC7 WebSocket \u6216 HTTP \u63A5\u6536\u6D88\u606F\u3002",
                      }),
                    ],
                  }),
                  _jsx(CardContent, {
                    className: "space-y-4",
                    children: _jsxs("div", {
                      className:
                        "rounded-lg bg-muted/50 p-4 border font-mono text-[11px] space-y-4 leading-relaxed",
                      children: [
                        _jsxs("div", {
                          children: [
                            _jsx("p", {
                              className: "text-primary font-bold mb-1",
                              children: "// WebSocket\uFF08\u5B9E\u65F6\u63A8\u9001\uFF09",
                            }),
                            _jsx("code", {
                              className: "block bg-background p-2 rounded border",
                              children: `${window.location.protocol === "https:" ? "wss" : "ws"}://${window.location.host}/api/v1/channels/connect?key=${channel.api_key.slice(0, 12)}...`,
                            }),
                          ],
                        }),
                        _jsxs("div", {
                          children: [
                            _jsx("p", {
                              className: "text-primary font-bold mb-1",
                              children: "// HTTP POST\uFF08\u53D1\u9001\u56DE\u590D\uFF09",
                            }),
                            _jsx("code", {
                              className: "block bg-background p-2 rounded border",
                              children: `POST ${window.location.origin}/api/v1/channels/send?key=${channel.api_key.slice(0, 12)}...`,
                            }),
                          ],
                        }),
                      ],
                    }),
                  }),
                ],
              }),
            ],
          }),
          _jsx(TabsContent, {
            value: "webhook",
            className: "m-0",
            children: _jsx(WebhookTab, { channel: channel, botId: botId, onRefresh: load }),
          }),
          _jsx(TabsContent, {
            value: "ai",
            className: "m-0",
            children: _jsx(AITab, { channel: channel, botId: botId, onRefresh: load }),
          }),
          _jsx(TabsContent, {
            value: "filter",
            className: "m-0",
            children: _jsx(FilterTab, { channel: channel, botId: botId, onRefresh: load }),
          }),
          _jsx(TabsContent, {
            value: "logs",
            className: "m-0",
            children: _jsx(WebhookLogsTab, { channel: channel, botId: botId }),
          }),
        ],
      }),
    ],
  });
}
// ==================== Sub-Tabs Components (Standardized) ====================
function WebhookTab({ channel, botId, onRefresh }) {
  const cfg = channel.webhook_config || {};
  const [form, setForm] = useState({
    url: cfg.url || "",
    authType: cfg.auth?.type || "none",
    authToken: cfg.auth?.token || "",
    authName: cfg.auth?.name || "",
    authValue: cfg.auth?.value || cfg.auth?.secret || "",
    script: cfg.script || "",
  });
  const [saving, setSaving] = useState(false);
  const { toast } = useToast();
  async function handleSave() {
    setSaving(true);
    try {
      let auth = null;
      if (form.authType === "bearer") auth = { type: "bearer", token: form.authToken };
      else if (form.authType === "header")
        auth = { type: "header", name: form.authName, value: form.authValue };
      else if (form.authType === "hmac") auth = { type: "hmac", secret: form.authValue };
      await api.updateChannel(botId, channel.id, {
        webhook_config: {
          url: form.url,
          auth,
          script: form.script || undefined,
        },
      });
      toast({ title: "Webhook 配置已更新" });
      onRefresh();
    } catch (e) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
    setSaving(false);
  }
  return _jsxs("div", {
    className: "grid gap-6 md:grid-cols-2",
    children: [
      _jsxs(Card, {
        className: "border-border/50",
        children: [
          _jsxs(CardHeader, {
            children: [
              _jsx(CardTitle, { className: "text-lg", children: "Webhook \u8BBE\u7F6E" }),
              _jsx(CardDescription, {
                children: "\u6D88\u606F\u5C06\u8F6C\u53D1\u5230\u6B64\u5730\u5740\u3002",
              }),
            ],
          }),
          _jsxs(CardContent, {
            className: "space-y-4",
            children: [
              _jsxs("div", {
                className: "space-y-2",
                children: [
                  _jsx("label", { className: "text-xs font-medium", children: "Webhook URL" }),
                  _jsx(Input, {
                    placeholder: "https://...",
                    value: form.url,
                    onChange: (e) => setForm({ ...form, url: e.target.value }),
                    className: "font-mono",
                  }),
                ],
              }),
              _jsxs("div", {
                className: "space-y-3 pt-2",
                children: [
                  _jsx("p", {
                    className: "text-xs font-medium",
                    children: "\u8BA4\u8BC1\u65B9\u5F0F",
                  }),
                  _jsx("div", {
                    className: "flex flex-wrap gap-2",
                    children: ["none", "bearer", "header", "hmac"].map((t) =>
                      _jsx(
                        Button,
                        {
                          variant: form.authType === t ? "default" : "outline",
                          className: "h-7 px-3 py-1 uppercase text-[10px] rounded-full",
                          onClick: () => setForm({ ...form, authType: t }),
                          children: t,
                        },
                        t,
                      ),
                    ),
                  }),
                  _jsxs("div", {
                    className: "pt-2",
                    children: [
                      form.authType === "bearer"
                        ? _jsx(Input, {
                            placeholder: "Token",
                            value: form.authToken,
                            onChange: (e) => setForm({ ...form, authToken: e.target.value }),
                            className: "h-9 font-mono",
                          })
                        : null,
                      form.authType === "header"
                        ? _jsxs("div", {
                            className: "flex gap-2",
                            children: [
                              _jsx(Input, {
                                placeholder: "Name",
                                value: form.authName,
                                onChange: (e) => setForm({ ...form, authName: e.target.value }),
                                className: "h-9",
                              }),
                              _jsx(Input, {
                                placeholder: "Value",
                                value: form.authValue,
                                onChange: (e) => setForm({ ...form, authValue: e.target.value }),
                                className: "h-9",
                              }),
                            ],
                          })
                        : null,
                      form.authType === "hmac"
                        ? _jsx(Input, {
                            placeholder: "Secret",
                            value: form.authValue,
                            onChange: (e) => setForm({ ...form, authValue: e.target.value }),
                            className: "h-9 font-mono",
                          })
                        : null,
                    ],
                  }),
                ],
              }),
            ],
          }),
          _jsx(CardFooter, {
            className: "bg-muted/30 pt-4 flex justify-end",
            children: _jsxs(Button, {
              onClick: handleSave,
              disabled: saving,
              size: "sm",
              children: [
                saving ? _jsx(Loader2, { className: "mr-2 h-3 w-3 animate-spin" }) : null,
                "\u4FDD\u5B58\u8BBE\u7F6E",
              ],
            }),
          }),
        ],
      }),
      _jsxs(Card, {
        className: "border-border/50",
        children: [
          _jsxs(CardHeader, {
            children: [
              _jsx(CardTitle, { className: "text-lg", children: "\u811A\u672C\u5904\u7406" }),
              _jsx(CardDescription, {
                children: "\u5728\u8F6C\u53D1\u524D\u5BF9\u6D88\u606F\u505A\u5904\u7406\u3002",
              }),
            ],
          }),
          _jsx(CardContent, {
            className: "space-y-4",
            children: _jsxs("div", {
              className: "space-y-2",
              children: [
                _jsx("textarea", {
                  placeholder: `function onRequest(ctx) {\n  // 转换消息格式...\n}`,
                  value: form.script,
                  onChange: (e) => setForm({ ...form, script: e.target.value }),
                  className:
                    "w-full h-40 bg-muted/30 border rounded-md p-3 font-mono text-[11px] focus:outline-none",
                }),
                _jsx("p", {
                  className: "text-[10px] text-muted-foreground",
                  children:
                    "\u811A\u672C\u5728\u5B89\u5168\u6C99\u7BB1\u4E2D\u8FD0\u884C\uFF0C\u652F\u6301 reply() \u548C\u4FEE\u6539 ctx.body\u3002",
                }),
              ],
            }),
          }),
        ],
      }),
    ],
  });
}
function AITab({ channel, botId, onRefresh }) {
  const cfg = channel.ai_config || {};
  const [form, setForm] = useState({
    enabled: cfg.enabled || false,
    source: cfg.source || "builtin",
    baseUrl: cfg.base_url || "",
    apiKey: "",
    model: cfg.model || "",
    prompt: cfg.system_prompt || "",
    history: cfg.max_history || 20,
  });
  const [saving, setSaving] = useState(false);
  const { toast } = useToast();
  async function handleSave() {
    setSaving(true);
    try {
      await api.updateChannel(botId, channel.id, {
        ai_config: {
          ...form,
          base_url: form.baseUrl,
          api_key: form.apiKey || undefined,
          system_prompt: form.prompt,
          max_history: form.history,
        },
      });
      toast({ title: "AI 配置已保存" });
      onRefresh();
    } catch (e) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
    setSaving(false);
  }
  return _jsxs(Card, {
    className: "border-border/50 max-w-3xl",
    children: [
      _jsx(CardHeader, {
        children: _jsxs("div", {
          className: "flex items-center justify-between",
          children: [
            _jsxs("div", {
              className: "space-y-1",
              children: [
                _jsx(CardTitle, { children: "AI \u81EA\u52A8\u56DE\u590D" }),
                _jsx(CardDescription, {
                  children:
                    "\u6536\u5230\u6D88\u606F\u540E\u81EA\u52A8\u7528 AI \u56DE\u590D\u3002",
                }),
              ],
            }),
            _jsx(Button, {
              variant: form.enabled ? "default" : "outline",
              size: "sm",
              onClick: () => setForm({ ...form, enabled: !form.enabled }),
              children: form.enabled ? "已开启" : "已关闭",
            }),
          ],
        }),
      }),
      form.enabled
        ? _jsxs(CardContent, {
            className: "space-y-6 pt-2 animate-in fade-in slide-in-from-top-2",
            children: [
              _jsxs("div", {
                className: "space-y-3",
                children: [
                  _jsx("p", {
                    className: "text-xs font-medium",
                    children: "\u6A21\u578B\u6765\u6E90",
                  }),
                  _jsxs("div", {
                    className: "flex gap-2",
                    children: [
                      _jsx(Button, {
                        variant: form.source === "builtin" ? "default" : "outline",
                        size: "sm",
                        onClick: () => setForm({ ...form, source: "builtin" }),
                        children: "\u7CFB\u7EDF\u5185\u7F6E",
                      }),
                      _jsx(Button, {
                        variant: form.source === "custom" ? "default" : "outline",
                        size: "sm",
                        onClick: () => setForm({ ...form, source: "custom" }),
                        children: "\u81EA\u5B9A\u4E49\u63A5\u53E3 (OpenAI \u534F\u8BAE)",
                      }),
                    ],
                  }),
                ],
              }),
              form.source === "custom"
                ? _jsxs("div", {
                    className: "grid gap-4 sm:grid-cols-2 border rounded-xl p-4 bg-muted/20",
                    children: [
                      _jsxs("div", {
                        className: "sm:col-span-2 space-y-1.5",
                        children: [
                          _jsx("label", {
                            className: "text-[11px] font-bold uppercase",
                            children: "\u63A5\u53E3\u5730\u5740",
                          }),
                          _jsx(Input, {
                            placeholder: "https://api.openai.com/v1",
                            value: form.baseUrl,
                            onChange: (e) => setForm({ ...form, baseUrl: e.target.value }),
                            className: "h-9 font-mono text-xs",
                          }),
                        ],
                      }),
                      _jsxs("div", {
                        className: "space-y-1.5",
                        children: [
                          _jsx("label", {
                            className: "text-[11px] font-bold uppercase",
                            children: "API Key",
                          }),
                          _jsx(Input, {
                            type: "password",
                            placeholder: "sk-...",
                            value: form.apiKey,
                            onChange: (e) => setForm({ ...form, apiKey: e.target.value }),
                            className: "h-9 font-mono text-xs",
                          }),
                        ],
                      }),
                      _jsxs("div", {
                        className: "space-y-1.5",
                        children: [
                          _jsx("label", {
                            className: "text-[11px] font-bold uppercase",
                            children: "\u6A21\u578B",
                          }),
                          _jsx(Input, {
                            placeholder: "gpt-4o",
                            value: form.model,
                            onChange: (e) => setForm({ ...form, model: e.target.value }),
                            className: "h-9 font-mono text-xs",
                          }),
                        ],
                      }),
                    ],
                  })
                : null,
              _jsxs("div", {
                className: "space-y-2",
                children: [
                  _jsx("label", {
                    className: "text-xs font-medium",
                    children: "\u7CFB\u7EDF\u63D0\u793A\u8BCD",
                  }),
                  _jsx("textarea", {
                    value: form.prompt,
                    onChange: (e) => setForm({ ...form, prompt: e.target.value }),
                    className:
                      "w-full h-24 bg-muted/30 border rounded-md p-3 text-xs leading-relaxed focus:outline-none",
                    placeholder: "\u4F60\u662F\u4E00\u4E2A\u667A\u80FD\u52A9\u7406...",
                  }),
                ],
              }),
              _jsxs("div", {
                className: "flex items-center gap-4",
                children: [
                  _jsxs("div", {
                    className: "space-y-1",
                    children: [
                      _jsx("label", {
                        className: "text-xs font-medium",
                        children: "\u4E0A\u4E0B\u6587\u6DF1\u5EA6",
                      }),
                      _jsx(Input, {
                        type: "number",
                        value: form.history,
                        onChange: (e) =>
                          setForm({ ...form, history: parseInt(e.target.value) || 20 }),
                        className: "w-24 h-9",
                      }),
                    ],
                  }),
                  _jsx("p", {
                    className: "text-[10px] text-muted-foreground italic flex-1",
                    children:
                      "\u8F83\u5927\u7684\u6DF1\u5EA6\u4F1A\u6D88\u8017\u66F4\u591A Token\uFF0C\u4F46\u5BF9\u8BDD\u8FDE\u8D2F\u6027\u66F4\u597D\u3002",
                  }),
                ],
              }),
            ],
          })
        : null,
      _jsx(CardFooter, {
        className: "bg-muted/30 pt-4 flex justify-end",
        children: _jsx(Button, {
          onClick: handleSave,
          disabled: saving,
          size: "sm",
          children: "\u4FDD\u5B58 AI \u914D\u7F6E",
        }),
      }),
    ],
  });
}
function FilterTab({ channel, botId, onRefresh }) {
  const rule = channel.filter_rule || {};
  const [vals, setVals] = useState({
    uids: (rule.user_ids || []).join(", "),
    words: (rule.keywords || []).join(", "),
    types: (rule.message_types || []).join(", "),
  });
  const [saving, setSaving] = useState(false);
  const { toast } = useToast();
  async function handleSave() {
    setSaving(true);
    const parse = (s) =>
      s
        .split(",")
        .map((v) => v.trim())
        .filter(Boolean);
    try {
      await api.updateChannel(botId, channel.id, {
        filter_rule: {
          user_ids: parse(vals.uids),
          keywords: parse(vals.words),
          message_types: parse(vals.types),
        },
      });
      toast({ title: "过滤规则已更新" });
      onRefresh();
    } catch (e) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
    setSaving(false);
  }
  return _jsxs(Card, {
    className: "border-border/50 max-w-2xl",
    children: [
      _jsxs(CardHeader, {
        children: [
          _jsx(CardTitle, { children: "\u8FC7\u6EE4\u6761\u4EF6" }),
          _jsx(CardDescription, {
            children: "\u8BBE\u7F6E\u54EA\u4E9B\u6D88\u606F\u4F1A\u88AB\u8F6C\u53D1\u3002",
          }),
        ],
      }),
      _jsxs(CardContent, {
        className: "space-y-4",
        children: [
          _jsxs("div", {
            className: "space-y-1.5",
            children: [
              _jsx("label", {
                className: "text-xs font-medium",
                children: "\u7528\u6237\u767D\u540D\u5355 (User ID)",
              }),
              _jsx(Input, {
                value: vals.uids,
                onChange: (e) => setVals({ ...vals, uids: e.target.value }),
                placeholder: "u123, u456...",
                className: "h-9 font-mono text-xs",
              }),
            ],
          }),
          _jsxs("div", {
            className: "space-y-1.5",
            children: [
              _jsx("label", {
                className: "text-xs font-medium",
                children: "\u5173\u952E\u5B57\u5305\u542B",
              }),
              _jsx(Input, {
                value: vals.words,
                onChange: (e) => setVals({ ...vals, words: e.target.value }),
                placeholder: "help, \u83DC\u5355...",
                className: "h-9 text-xs",
              }),
            ],
          }),
          _jsxs("div", {
            className: "space-y-1.5",
            children: [
              _jsx("label", {
                className: "text-xs font-medium",
                children: "\u6D88\u606F\u7C7B\u578B\u9650\u5236",
              }),
              _jsx(Input, {
                value: vals.types,
                onChange: (e) => setVals({ ...vals, types: e.target.value }),
                placeholder: "text, image, voice...",
                className: "h-9 font-mono text-xs",
              }),
            ],
          }),
          _jsxs("div", {
            className:
              "mt-4 p-3 bg-primary/5 rounded-lg border border-primary/10 flex gap-3 items-start",
            children: [
              _jsx(Info, { className: "h-4 w-4 text-primary shrink-0 mt-0.5" }),
              _jsx("p", {
                className: "text-[10px] text-muted-foreground leading-relaxed",
                children:
                  "\u7528\u82F1\u6587\u9017\u53F7\u5206\u9694\u591A\u4E2A\u503C\u3002\u7559\u7A7A\u8868\u793A\u5339\u914D\u6240\u6709\u6D88\u606F\u3002\u6EE1\u8DB3\u4EFB\u4E00\u6761\u4EF6\u5373\u8F6C\u53D1\u3002",
              }),
            ],
          }),
        ],
      }),
      _jsx(CardFooter, {
        className: "bg-muted/30 pt-4 flex justify-end",
        children: _jsx(Button, {
          onClick: handleSave,
          disabled: saving,
          size: "sm",
          children: "\u4FDD\u5B58\u8FC7\u6EE4\u89C4\u5219",
        }),
      }),
    ],
  });
}
function WebhookLogsTab({ channel, botId }) {
  const {
    data: logs = [],
    isLoading: loading,
    isError: logsError,
    refetch,
  } = useWebhookLogs(botId, channel.id);
  // Push-driven updates: useBotPush subscribes to events; PushProvider
  // auto-invalidates webhook-logs cache on message_new events.
  useBotPush(botId);
  return _jsxs("div", {
    className: "space-y-4",
    children: [
      _jsxs("div", {
        className: "flex items-center justify-between",
        children: [
          _jsx("p", {
            className: "text-xs text-muted-foreground",
            children: "\u6700\u8FD1 50 \u6761\u8BF7\u6C42\u8BB0\u5F55",
          }),
          _jsx(Button, {
            variant: "ghost",
            size: "sm",
            className: "h-7 text-xs",
            onClick: () => refetch(),
            children: "\u5237\u65B0",
          }),
        ],
      }),
      _jsx("div", {
        className: "rounded-xl border overflow-hidden",
        children: _jsxs(Table, {
          children: [
            _jsx(TableHeader, {
              className: "bg-muted/30",
              children: _jsxs(TableRow, {
                children: [
                  _jsx(TableHead, { children: "\u72B6\u6001" }),
                  _jsx(TableHead, { children: "\u65B9\u6CD5" }),
                  _jsx(TableHead, { children: "\u8017\u65F6" }),
                  _jsx(TableHead, {
                    className: "text-right",
                    children: "\u8BF7\u6C42\u65F6\u95F4",
                  }),
                ],
              }),
            }),
            _jsx(TableBody, {
              children:
                loading && logs.length === 0
                  ? _jsx(TableRow, {
                      children: _jsx(TableCell, {
                        colSpan: 4,
                        className: "h-24 text-center",
                        children: _jsx(Loader2, {
                          className: "h-4 w-4 animate-spin mx-auto opacity-20",
                        }),
                      }),
                    })
                  : logsError
                    ? _jsx(TableRow, {
                        children: _jsx(TableCell, {
                          colSpan: 4,
                          className: "h-24 text-center text-destructive italic",
                          children: "\u52A0\u8F7D\u5931\u8D25",
                        }),
                      })
                    : logs.length === 0
                      ? _jsx(TableRow, {
                          children: _jsx(TableCell, {
                            colSpan: 4,
                            className: "h-24 text-center text-muted-foreground italic",
                            children: "\u6682\u65E0\u8BB0\u5F55",
                          }),
                        })
                      : logs.map((log) =>
                          _jsxs(
                            TableRow,
                            {
                              className: "text-xs group hover:bg-muted/30",
                              children: [
                                _jsx(TableCell, {
                                  children: _jsx(Badge, {
                                    variant: log.status === "success" ? "default" : "destructive",
                                    className: "h-4 text-[9px] uppercase",
                                    children: log.response_status || log.status,
                                  }),
                                }),
                                _jsx(TableCell, {
                                  className: "font-mono text-muted-foreground uppercase",
                                  children: log.request_method,
                                }),
                                _jsxs(TableCell, {
                                  className: "font-mono text-muted-foreground",
                                  children: [log.duration_ms, "ms"],
                                }),
                                _jsx(TableCell, {
                                  className: "text-right text-muted-foreground opacity-60",
                                  children: new Date(log.created_at * 1000).toLocaleTimeString(),
                                }),
                              ],
                            },
                            log.id,
                          ),
                        ),
            }),
          ],
        }),
      }),
    ],
  });
}
