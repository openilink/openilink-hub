import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { useState, useRef } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import {
  ArrowUpRight,
  Trash2,
  Bot as BotIcon,
  Cpu,
  Unplug,
  MessageSquare,
  Activity,
  Blocks,
  Download,
  RefreshCw,
  Sparkles,
  Pencil,
  Check,
  X,
} from "lucide-react";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { botDisplayName } from "../lib/api";
import {
  useBot,
  useBotApps,
  useDeleteBot,
  useSetBotAI,
  useSetBotAIModel,
  useUpdateBot,
} from "@/hooks/use-bots";
import { useApps } from "@/hooks/use-apps";
import { useAvailableModels } from "@/hooks/use-apps";
import { useBuiltinApps, useMarketplaceApps, useSyncMarketplaceApp } from "@/hooks/use-marketplace";
import { Card, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { useToast } from "@/hooks/use-toast";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { AppIcon } from "../components/app-icon";
import { parseTools } from "../components/tools-display";
import { useConfirm } from "@/components/ui/confirm-dialog";
const DEFAULT_MODEL = "__default__";
// ==================== Page ====================
function formatRelativeTime(ts) {
  if (!ts) return "—";
  const diff = Math.floor((Date.now() - ts * 1000) / 1000);
  if (diff < 0) return "刚刚";
  if (diff < 60) return `${diff}秒前`;
  if (diff < 3600) return `${Math.floor(diff / 60)}分钟前`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}小时前`;
  return `${Math.floor(diff / 86400)}天前`;
}
export function BotDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { toast } = useToast();
  const { confirm, ConfirmDialog } = useConfirm();
  // Server state via react-query
  const { data: bot, isLoading: loading } = useBot(id);
  const { data: installations = [] } = useBotApps(id);
  const { data: builtinApps = [] } = useBuiltinApps();
  const { data: listedAppsRaw = [] } = useApps({ listing: "listed" });
  const { data: marketplaceApps = [] } = useMarketplaceApps();
  const { data: availableModels = [] } = useAvailableModels();
  // Derived: listed apps excluding builtins, installed app IDs for this bot
  const builtinSlugs = new Set(builtinApps.map((a) => a.slug));
  const listedApps = listedAppsRaw.filter((a) => !builtinSlugs.has(a.slug));
  const installedOnBot = new Set(installations.map((inst) => inst.app_id));
  const marketplaceLoading = false; // All queries load in parallel, handled by isLoading above
  // Mutations
  const updateBotMutation = useUpdateBot();
  const deleteBotMutation = useDeleteBot();
  const setAIMutation = useSetBotAI();
  const setAIModelMutation = useSetBotAIModel();
  const syncAppMutation = useSyncMarketplaceApp();
  // Local UI state
  const [syncing, setSyncing] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [editingDisplayName, setEditingDisplayName] = useState(false);
  const [displayNameDraft, setDisplayNameDraft] = useState("");
  const marketplaceRef = useRef(null);
  const deleteInFlightRef = useRef(false);
  const handleDeleteBot = async () => {
    if (!bot || deleteInFlightRef.current) return;
    const finishDelete = () => {
      deleteInFlightRef.current = false;
      setIsDeleting(false);
    };
    deleteInFlightRef.current = true;
    setIsDeleting(true);
    try {
      const ok = await confirm({
        title: "删除确认",
        description: "确定要删除此账号？相关转发规则将停止工作。",
        confirmText: "删除",
        variant: "destructive",
      });
      if (!ok) {
        finishDelete();
        return;
      }
      deleteBotMutation.mutate(bot.id, {
        onSuccess: () => {
          toast({ title: "已删除账号" });
          navigate("/dashboard/accounts");
        },
        onError: (err) => {
          toast({ variant: "destructive", title: "删除失败", description: err.message });
        },
        onSettled: finishDelete,
      });
    } catch {
      finishDelete();
    }
  };
  const handleAutoRenewalChange = async (hours) => {
    updateBotMutation.mutate(
      { id: bot.id, data: { reminder_hours: hours } },
      {
        onSuccess: () => toast({ title: "已保存" }),
        onError: (e) =>
          toast({ variant: "destructive", title: "保存失败", description: e.message }),
      },
    );
  };
  const handleInstallApp = async (app) => {
    if (app.local_id) {
      navigate(`/dashboard/accounts/${id}/install/${app.local_id}`);
      return;
    }
    setSyncing(true);
    syncAppMutation.mutate(app.slug, {
      onSuccess: (synced) => navigate(`/dashboard/accounts/${id}/install/${synced.id}`),
      onError: (e) => toast({ variant: "destructive", title: "同步失败", description: e.message }),
      onSettled: () => setSyncing(false),
    });
  };
  if (loading)
    return _jsxs("div", {
      className: "space-y-6",
      children: [
        _jsx(Skeleton, { className: "h-20 w-full rounded-3xl" }),
        _jsx(Skeleton, { className: "h-96 w-full rounded-3xl" }),
      ],
    });
  if (!bot)
    return _jsxs("div", {
      className: "py-20 text-center space-y-4",
      children: [
        _jsx(Unplug, { className: "h-12 w-12 mx-auto opacity-20" }),
        _jsx("p", { className: "font-bold", children: "\u672A\u627E\u5230\u8D26\u53F7" }),
        _jsx(Button, {
          variant: "link",
          asChild: true,
          children: _jsx(Link, { to: "/dashboard/accounts", children: "\u8FD4\u56DE\u5217\u8868" }),
        }),
      ],
    });
  return _jsxs("div", {
    className: "flex flex-col gap-8 h-full",
    children: [
      ConfirmDialog,
      _jsxs("div", {
        className: "flex flex-col md:flex-row md:items-start justify-between gap-6",
        children: [
          _jsxs("div", {
            className: "flex items-center gap-4",
            children: [
              _jsx("div", {
                className:
                  "h-14 w-14 rounded-2xl bg-primary/10 flex items-center justify-center text-primary border border-primary/20 shrink-0",
                children: _jsx(BotIcon, { className: "h-7 w-7" }),
              }),
              _jsxs("div", {
                className: "space-y-1",
                children: [
                  _jsxs("div", {
                    className: "flex items-center gap-2 flex-wrap",
                    children: [
                      editingDisplayName
                        ? _jsxs("form", {
                            className: "flex items-center gap-1.5",
                            onSubmit: (e) => {
                              e.preventDefault();
                              updateBotMutation.mutate(
                                { id: bot.id, data: { display_name: displayNameDraft } },
                                {
                                  onSuccess: () => {
                                    toast({ title: "已保存" });
                                    setEditingDisplayName(false);
                                  },
                                  onError: (err) =>
                                    toast({
                                      variant: "destructive",
                                      title: "保存失败",
                                      description: err.message,
                                    }),
                                },
                              );
                            },
                            children: [
                              _jsx(Input, {
                                autoFocus: true,
                                className: "h-8 w-48 text-lg font-bold",
                                value: displayNameDraft,
                                onChange: (e) => setDisplayNameDraft(e.target.value),
                                placeholder: bot.name,
                              }),
                              _jsx(Button, {
                                type: "submit",
                                variant: "ghost",
                                size: "icon-sm",
                                children: _jsx(Check, { className: "h-4 w-4" }),
                              }),
                              _jsx(Button, {
                                type: "button",
                                variant: "ghost",
                                size: "icon-sm",
                                onClick: () => setEditingDisplayName(false),
                                children: _jsx(X, { className: "h-4 w-4" }),
                              }),
                            ],
                          })
                        : _jsxs("div", {
                            className: "flex items-center gap-1.5 group/name",
                            children: [
                              _jsx("h1", {
                                className: "text-2xl font-bold tracking-tight",
                                children: botDisplayName(bot),
                              }),
                              _jsx(Button, {
                                variant: "ghost",
                                size: "icon-sm",
                                className:
                                  "opacity-0 group-hover/name:opacity-100 transition-opacity",
                                onClick: () => {
                                  setDisplayNameDraft(bot.display_name || "");
                                  setEditingDisplayName(true);
                                },
                                children: _jsx(Pencil, { className: "h-3.5 w-3.5" }),
                              }),
                            ],
                          }),
                      _jsx(Badge, {
                        variant: bot.status === "connected" ? "default" : "destructive",
                        children:
                          bot.status === "connected"
                            ? "运行中"
                            : bot.status === "session_expired"
                              ? "授权过期"
                              : "离线",
                      }),
                      bot.can_send === false
                        ? _jsx(Badge, {
                            variant: "outline",
                            className: "text-orange-600 border-orange-300",
                            children: "\u4E0D\u53EF\u53D1\u9001",
                          })
                        : null,
                    ],
                  }),
                  _jsxs("div", {
                    className: "flex items-center gap-1.5 text-xs text-muted-foreground",
                    children: [
                      _jsx(Cpu, { className: "h-3 w-3" }),
                      _jsx("span", { className: "capitalize", children: bot.provider }),
                      _jsx("span", { className: "opacity-40", children: "\u00B7" }),
                      _jsxs("span", {
                        className: "font-mono",
                        children: [bot.id.slice(0, 12), "\u2026"],
                      }),
                    ],
                  }),
                  bot.send_disabled_reason
                    ? _jsx("p", {
                        className: "text-xs text-orange-600",
                        children: bot.send_disabled_reason,
                      })
                    : null,
                ],
              }),
            ],
          }),
          _jsxs("div", {
            className: "flex items-center gap-2 flex-wrap shrink-0",
            children: [
              _jsx(Button, {
                variant: "outline",
                size: "sm",
                asChild: true,
                children: _jsxs(Link, {
                  to: `/dashboard/accounts/${id}/console`,
                  children: [
                    _jsx(MessageSquare, { className: "h-3.5 w-3.5" }),
                    "\u6D88\u606F\u63A7\u5236\u53F0",
                  ],
                }),
              }),
              _jsx(Button, {
                variant: "outline",
                size: "sm",
                asChild: true,
                children: _jsxs(Link, {
                  to: `/dashboard/accounts/${id}/traces`,
                  children: [
                    _jsx(Activity, { className: "h-3.5 w-3.5" }),
                    "\u6D88\u606F\u8FFD\u8E2A",
                  ],
                }),
              }),
              _jsx(Separator, { orientation: "vertical", className: "h-6 mx-1" }),
              _jsxs("div", {
                className: "flex items-center gap-1.5",
                children: [
                  _jsxs(Label, {
                    htmlFor: `ai-toggle-${id}`,
                    className:
                      "text-xs text-muted-foreground flex items-center gap-1.5 cursor-pointer",
                    children: [
                      _jsx(Sparkles, { className: "h-3.5 w-3.5 text-primary" }),
                      "AI \u56DE\u590D",
                    ],
                  }),
                  _jsx(Switch, {
                    id: `ai-toggle-${id}`,
                    checked: bot.ai_enabled || false,
                    onCheckedChange: (enabled) => {
                      setAIMutation.mutate(
                        { botId: id, enabled },
                        {
                          onSuccess: () =>
                            toast({ title: enabled ? "AI 回复已开启" : "AI 回复已关闭" }),
                          onError: (err) =>
                            toast({
                              variant: "destructive",
                              title: "操作失败",
                              description: err.message,
                            }),
                        },
                      );
                    },
                  }),
                ],
              }),
              bot.ai_enabled &&
                availableModels.length > 0 &&
                _jsxs("div", {
                  className: "flex items-center gap-1.5",
                  children: [
                    _jsx(Label, {
                      className: "text-xs font-bold uppercase text-muted-foreground",
                      children: "\u6A21\u578B",
                    }),
                    _jsxs(Select, {
                      value: bot.ai_model || DEFAULT_MODEL,
                      onValueChange: (val) => {
                        const model = val === DEFAULT_MODEL ? "" : val;
                        setAIModelMutation.mutate(
                          { botId: id, model },
                          {
                            onSuccess: () =>
                              toast({
                                title: model ? `已切换到模型：${model}` : "已恢复全局默认模型",
                              }),
                            onError: (err) =>
                              toast({
                                variant: "destructive",
                                title: "操作失败",
                                description: err.message,
                              }),
                          },
                        );
                      },
                      children: [
                        _jsx(SelectTrigger, {
                          className: "h-7 text-xs w-48",
                          children: _jsx(SelectValue, {
                            placeholder: "\u4F7F\u7528\u5168\u5C40\u9ED8\u8BA4",
                          }),
                        }),
                        _jsxs(SelectContent, {
                          children: [
                            _jsx(SelectItem, {
                              value: DEFAULT_MODEL,
                              children: "\u4F7F\u7528\u5168\u5C40\u9ED8\u8BA4",
                            }),
                            availableModels
                              .filter(Boolean)
                              .map((m) => _jsx(SelectItem, { value: m, children: m }, m)),
                          ],
                        }),
                      ],
                    }),
                  ],
                }),
              _jsx(Separator, { orientation: "vertical", className: "h-6 mx-1" }),
              _jsxs("div", {
                className: "flex items-center gap-1.5",
                children: [
                  _jsx("span", {
                    className: "text-xs text-muted-foreground",
                    children: "\u81EA\u52A8\u7EED\u671F",
                  }),
                  _jsxs(Select, {
                    value: String(bot.reminder_hours || 0),
                    onValueChange: (v) => handleAutoRenewalChange(Number(v)),
                    children: [
                      _jsx(SelectTrigger, {
                        className: "h-7 w-28 text-xs",
                        children: _jsx(SelectValue, {}),
                      }),
                      _jsxs(SelectContent, {
                        children: [
                          _jsx(SelectItem, { value: "0", children: "\u4E0D\u63D0\u9192" }),
                          _jsx(SelectItem, {
                            value: "23",
                            children: "\u63D0\u524D 1 \u5C0F\u65F6",
                          }),
                          _jsx(SelectItem, {
                            value: "22",
                            children: "\u63D0\u524D 2 \u5C0F\u65F6",
                          }),
                        ],
                      }),
                    ],
                  }),
                  bot.reminder_hours > 0 &&
                    _jsxs(Tooltip, {
                      children: [
                        _jsx(TooltipTrigger, {
                          asChild: true,
                          children: _jsx("span", {
                            className: "text-[10px] text-muted-foreground/60 cursor-help",
                            children: bot.last_reminded_at
                              ? `上次 ${formatRelativeTime(bot.last_reminded_at)}`
                              : "尚未提醒",
                          }),
                        }),
                        _jsxs(TooltipContent, {
                          side: "bottom",
                          className: "text-xs space-y-1",
                          children: [
                            _jsxs("p", {
                              children: [
                                "\u4E0A\u6B21\u6D88\u606F:",
                                " ",
                                bot.last_msg_at
                                  ? new Date(bot.last_msg_at * 1000).toLocaleString()
                                  : "无",
                              ],
                            }),
                            _jsxs("p", {
                              children: [
                                "\u4E0A\u6B21\u63D0\u9192:",
                                " ",
                                bot.last_reminded_at
                                  ? new Date(bot.last_reminded_at * 1000).toLocaleString()
                                  : "无",
                              ],
                            }),
                            _jsxs("p", {
                              children: [
                                "\u4E0B\u6B21\u63D0\u9192:",
                                " ",
                                bot.last_msg_at
                                  ? new Date(
                                      Math.max(
                                        bot.last_msg_at + bot.reminder_hours * 3600,
                                        (bot.last_reminded_at || 0) + 3600,
                                      ) * 1000,
                                    ).toLocaleString()
                                  : "等待首条消息",
                              ],
                            }),
                          ],
                        }),
                      ],
                    }),
                ],
              }),
              _jsx(Separator, { orientation: "vertical", className: "h-6 mx-1" }),
              _jsx(Button, {
                variant: "outline",
                size: "sm",
                asChild: true,
                children: _jsx(Link, {
                  to: "/dashboard/accounts",
                  children: "\u8FD4\u56DE\u5217\u8868",
                }),
              }),
              _jsxs(Tooltip, {
                children: [
                  _jsx(TooltipTrigger, {
                    asChild: true,
                    children: _jsx(Button, {
                      variant: "destructive",
                      size: "icon-sm",
                      "aria-label": "\u5220\u9664\u8D26\u53F7",
                      disabled: isDeleting,
                      onClick: () => void handleDeleteBot(),
                      children: _jsx(Trash2, { className: "h-3.5 w-3.5" }),
                    }),
                  }),
                  _jsx(TooltipContent, { children: "\u5220\u9664\u8D26\u53F7" }),
                ],
              }),
            ],
          }),
        ],
      }),
      _jsxs(_Fragment, {
        children: [
          _jsxs("div", {
            className: "space-y-4",
            children: [
              _jsx("h2", {
                className: "text-sm font-semibold text-muted-foreground",
                children: "\u5DF2\u5B89\u88C5\u7684\u5E94\u7528",
              }),
              installations.length === 0
                ? _jsxs("div", {
                    className: "text-center py-16 space-y-3 border-2 border-dashed rounded-xl",
                    children: [
                      _jsx(Blocks, { className: "w-8 h-8 mx-auto text-muted-foreground/40" }),
                      _jsx("p", {
                        className: "text-sm text-muted-foreground",
                        children: "\u6682\u65E0\u5B89\u88C5\u7684\u5E94\u7528",
                      }),
                      _jsx(Button, {
                        variant: "outline",
                        size: "sm",
                        asChild: true,
                        children: _jsx(Link, {
                          to: "/dashboard/apps",
                          children: "\u53BB\u5E94\u7528\u5E02\u573A\u770B\u770B",
                        }),
                      }),
                    ],
                  })
                : _jsx("div", {
                    className: "grid gap-4 md:grid-cols-2 lg:grid-cols-3",
                    children: installations.map((inst) =>
                      _jsx(
                        Link,
                        {
                          to: `/dashboard/accounts/${id}/apps/${inst.id}`,
                          className: "group block",
                          children: _jsxs(Card, {
                            className:
                              "h-full border-border/50 transition-all hover:border-primary/30 hover:shadow-md",
                            children: [
                              _jsx(CardHeader, {
                                className: "pb-3",
                                children: _jsxs("div", {
                                  className: "flex items-start justify-between gap-2",
                                  children: [
                                    _jsxs("div", {
                                      className: "flex items-center gap-3 min-w-0",
                                      children: [
                                        _jsx(AppIcon, {
                                          icon: inst.app_icon,
                                          iconUrl: inst.app_icon_url,
                                          size: "h-9 w-9",
                                        }),
                                        _jsxs("div", {
                                          className: "min-w-0",
                                          children: [
                                            _jsx(CardTitle, {
                                              className:
                                                "text-sm font-semibold truncate group-hover:text-primary transition-colors",
                                              children: inst.app_name,
                                            }),
                                            inst.handle
                                              ? _jsxs("p", {
                                                  className:
                                                    "text-[11px] font-mono text-muted-foreground mt-0.5",
                                                  children: ["@", inst.handle],
                                                })
                                              : null,
                                          ],
                                        }),
                                      ],
                                    }),
                                    _jsx(Badge, {
                                      variant: inst.enabled ? "default" : "outline",
                                      className: "shrink-0 text-[10px]",
                                      children: inst.enabled ? "运行中" : "已停用",
                                    }),
                                  ],
                                }),
                              }),
                              _jsxs(CardFooter, {
                                className:
                                  "pt-2 pb-4 px-4 flex justify-between items-center border-t border-border/40",
                                children: [
                                  _jsx("span", {
                                    className: "text-[11px] font-mono text-muted-foreground/60",
                                    children: inst.app_slug,
                                  }),
                                  _jsx(ArrowUpRight, {
                                    className:
                                      "h-3.5 w-3.5 text-muted-foreground/40 group-hover:text-primary transition-colors",
                                  }),
                                ],
                              }),
                            ],
                          }),
                        },
                        inst.id,
                      ),
                    ),
                  }),
            ],
          }),
          _jsxs("div", {
            ref: marketplaceRef,
            className: "space-y-6",
            children: [
              _jsx("h2", {
                className: "text-sm font-semibold text-muted-foreground",
                children: "\u5E94\u7528\u5E02\u573A",
              }),
              !marketplaceLoading && builtinApps.length > 0
                ? _jsxs("div", {
                    className: "space-y-2",
                    children: [
                      _jsx("h3", {
                        className: "text-xs text-muted-foreground/60 px-1",
                        children: "\u5185\u7F6E\u5E94\u7528",
                      }),
                      _jsx("div", {
                        className:
                          "divide-y divide-border/50 rounded-xl border border-border/50 overflow-hidden",
                        children: builtinApps.map((app) =>
                          _jsxs(
                            "div",
                            {
                              className:
                                "group flex items-center gap-4 px-4 py-3.5 bg-card hover:bg-muted/40 transition-colors",
                              children: [
                                _jsx(AppIcon, {
                                  icon: app.icon,
                                  iconUrl: app.icon_url,
                                  size: "h-9 w-9",
                                }),
                                _jsxs("div", {
                                  className: "flex-1 min-w-0",
                                  children: [
                                    _jsxs("div", {
                                      className: "flex items-center gap-2",
                                      children: [
                                        _jsx("p", {
                                          className: "text-sm font-semibold leading-tight",
                                          children: app.name,
                                        }),
                                        installedOnBot.has(app.id)
                                          ? _jsx(Badge, {
                                              variant: "secondary",
                                              className: "text-[10px] shrink-0",
                                              children: "\u5DF2\u5B89\u88C5",
                                            })
                                          : null,
                                      ],
                                    }),
                                    _jsx("p", {
                                      className:
                                        "text-xs text-muted-foreground mt-0.5 line-clamp-1",
                                      children: app.description,
                                    }),
                                  ],
                                }),
                                parseTools(app.tools).length > 0
                                  ? _jsxs("span", {
                                      className:
                                        "text-[11px] text-muted-foreground/50 shrink-0 hidden sm:block",
                                      children: [
                                        parseTools(app.tools).length,
                                        " \u4E2A\u547D\u4EE4",
                                      ],
                                    })
                                  : null,
                                installedOnBot.has(app.id)
                                  ? _jsx("span", {
                                      className: "text-[11px] text-muted-foreground/50 shrink-0",
                                      children: "\u5DF2\u5B89\u88C5",
                                    })
                                  : _jsxs(Button, {
                                      size: "sm",
                                      variant: "outline",
                                      className: "shrink-0 gap-1.5",
                                      onClick: () =>
                                        navigate(`/dashboard/accounts/${id}/install/${app.id}`),
                                      children: [
                                        _jsx(Download, { className: "h-3.5 w-3.5" }),
                                        "\u5B89\u88C5",
                                      ],
                                    }),
                              ],
                            },
                            app.slug || app.id,
                          ),
                        ),
                      }),
                    ],
                  })
                : null,
              !marketplaceLoading && listedApps.length > 0
                ? _jsxs("div", {
                    className: "space-y-2",
                    children: [
                      _jsx("h3", {
                        className: "text-xs text-muted-foreground/60 px-1",
                        children: "\u63A8\u8350\u5E94\u7528",
                      }),
                      _jsx("div", {
                        className:
                          "divide-y divide-border/50 rounded-xl border border-border/50 overflow-hidden",
                        children: listedApps.map((app) =>
                          _jsxs(
                            "div",
                            {
                              className:
                                "group flex items-center gap-4 px-4 py-3.5 bg-card hover:bg-muted/40 transition-colors",
                              children: [
                                _jsx(AppIcon, {
                                  icon: app.icon,
                                  iconUrl: app.icon_url,
                                  size: "h-9 w-9",
                                }),
                                _jsxs("div", {
                                  className: "flex-1 min-w-0",
                                  children: [
                                    _jsxs("div", {
                                      className: "flex items-center gap-2",
                                      children: [
                                        _jsx("p", {
                                          className: "text-sm font-semibold leading-tight truncate",
                                          children: app.name,
                                        }),
                                        app.version
                                          ? _jsxs(Badge, {
                                              variant: "outline",
                                              className: "text-[10px] font-mono shrink-0",
                                              children: ["v", app.version],
                                            })
                                          : null,
                                        installedOnBot.has(app.id)
                                          ? _jsx(Badge, {
                                              variant: "secondary",
                                              className: "text-[10px] shrink-0",
                                              children: "\u5DF2\u5B89\u88C5",
                                            })
                                          : null,
                                      ],
                                    }),
                                    _jsx("p", {
                                      className:
                                        "text-xs text-muted-foreground mt-0.5 line-clamp-1",
                                      children: app.description,
                                    }),
                                  ],
                                }),
                                installedOnBot.has(app.id)
                                  ? _jsx("span", {
                                      className: "text-[11px] text-muted-foreground/50 shrink-0",
                                      children: "\u5DF2\u5B89\u88C5",
                                    })
                                  : _jsxs(Button, {
                                      size: "sm",
                                      variant: "outline",
                                      className: "shrink-0 gap-1.5",
                                      onClick: () =>
                                        navigate(`/dashboard/accounts/${id}/install/${app.id}`),
                                      children: [
                                        _jsx(Download, { className: "h-3.5 w-3.5" }),
                                        "\u5B89\u88C5",
                                      ],
                                    }),
                              ],
                            },
                            app.id,
                          ),
                        ),
                      }),
                    ],
                  })
                : null,
              !marketplaceLoading && marketplaceApps.length > 0
                ? _jsxs("div", {
                    className: "space-y-2",
                    children: [
                      _jsx("h3", {
                        className: "text-xs text-muted-foreground/60 px-1",
                        children: "\u8FDC\u7A0B\u5E02\u573A",
                      }),
                      _jsx("div", {
                        className:
                          "divide-y divide-border/50 rounded-xl border border-border/50 overflow-hidden",
                        children: marketplaceApps.map((app) =>
                          _jsxs(
                            "div",
                            {
                              className:
                                "group flex items-center gap-4 px-4 py-3.5 bg-card hover:bg-muted/40 transition-colors",
                              children: [
                                _jsx(AppIcon, {
                                  icon: app.icon,
                                  iconUrl: app.icon_url,
                                  size: "h-9 w-9",
                                }),
                                _jsxs("div", {
                                  className: "flex-1 min-w-0",
                                  children: [
                                    _jsxs("div", {
                                      className: "flex items-center gap-2",
                                      children: [
                                        _jsx("p", {
                                          className: "text-sm font-semibold leading-tight truncate",
                                          children: app.name,
                                        }),
                                        app.version
                                          ? _jsxs(Badge, {
                                              variant: "outline",
                                              className: "text-[10px] font-mono shrink-0",
                                              children: ["v", app.version],
                                            })
                                          : null,
                                        app.installed
                                          ? _jsx(Badge, {
                                              variant: "secondary",
                                              className: "text-[10px] shrink-0",
                                              children: "\u5DF2\u5B89\u88C5",
                                            })
                                          : null,
                                      ],
                                    }),
                                    _jsx("p", {
                                      className:
                                        "text-xs text-muted-foreground mt-0.5 line-clamp-1",
                                      children: app.description || "暂无描述",
                                    }),
                                  ],
                                }),
                                app.author
                                  ? _jsx("span", {
                                      className:
                                        "text-[11px] text-muted-foreground/50 shrink-0 hidden sm:block",
                                      children: app.author,
                                    })
                                  : null,
                                app.installed && app.update_available
                                  ? _jsxs(Button, {
                                      size: "sm",
                                      variant: "outline",
                                      className: "shrink-0 gap-1.5",
                                      disabled: syncing,
                                      onClick: () => handleInstallApp(app),
                                      children: [
                                        _jsx(RefreshCw, { className: "h-3.5 w-3.5" }),
                                        "\u66F4\u65B0",
                                      ],
                                    })
                                  : app.installed
                                    ? _jsx("span", {
                                        className: "text-[11px] text-muted-foreground/50 shrink-0",
                                        children: "\u5DF2\u5B89\u88C5",
                                      })
                                    : _jsxs(Button, {
                                        size: "sm",
                                        variant: "outline",
                                        className: "shrink-0 gap-1.5",
                                        disabled: syncing,
                                        onClick: () => handleInstallApp(app),
                                        children: [
                                          _jsx(Download, { className: "h-3.5 w-3.5" }),
                                          "\u5B89\u88C5",
                                        ],
                                      }),
                              ],
                            },
                            app.slug || app.id,
                          ),
                        ),
                      }),
                    ],
                  })
                : null,
            ],
          }),
        ],
      }),
    ],
  });
}
