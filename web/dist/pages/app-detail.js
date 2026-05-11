import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect, useState } from "react";
import { useParams, Link, useNavigate, useLocation } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeft,
  Plus,
  Trash2,
  ShieldCheck,
  Eye,
  EyeOff,
  Copy,
  Check,
  ExternalLink,
  Loader2,
  Settings,
  Download,
  Globe,
  Radio,
  Terminal,
  Shield,
  Zap,
} from "lucide-react";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { api, botDisplayName } from "../lib/api";
import { invalidateAllAppQueries, useApp } from "@/hooks/use-apps";
import { queryKeys } from "@/lib/query-keys";
import { useToast } from "@/hooks/use-toast";
import { useConfirm } from "@/components/ui/confirm-dialog";
import { AppIcon } from "../components/app-icon";
import { EVENT_TYPES, SCOPES } from "../lib/constants";
const NAV_SECTIONS = [
  {
    group: "设置",
    items: [
      { key: "basic-info", label: "基本信息", icon: Settings },
      { key: "install-app", label: "安装管理", icon: Download },
      { key: "distribution", label: "分发管理", icon: Globe },
    ],
  },
  {
    group: "功能",
    items: [
      { key: "event-subscriptions", label: "事件订阅", icon: Radio },
      { key: "commands", label: "命令 / 工具", icon: Terminal },
      { key: "oauth-permissions", label: "OAuth 权限", icon: Shield },
    ],
  },
];
export function AppDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const location = useLocation();
  const queryClient = useQueryClient();
  const { data: app, isError } = useApp(id);
  const [section, setSection] = useState("basic-info");
  const isDeveloper = location.pathname.startsWith("/dashboard/developer/");
  const backPath = isDeveloper ? "/dashboard/developer/apps" : "/dashboard/apps";
  // Convenience: invalidate app detail cache after mutations
  const refreshApp = () => queryClient.invalidateQueries({ queryKey: queryKeys.apps.detail(id) });
  if (isError) {
    navigate(backPath);
    return null;
  }
  if (!app) return null;
  return _jsxs("div", {
    className: "space-y-6",
    children: [
      _jsxs("div", {
        className: "flex items-center gap-3",
        children: [
          _jsx(Link, {
            to: backPath,
            className: "text-muted-foreground hover:text-foreground",
            "aria-label": "\u8FD4\u56DE\u6211\u7684\u5E94\u7528",
            children: _jsx(ArrowLeft, { className: "w-4 h-4" }),
          }),
          _jsx(AppIcon, { icon: app.icon, iconUrl: app.icon_url, size: "h-8 w-8" }),
          _jsxs("div", {
            className: "flex-1 min-w-0",
            children: [
              _jsx("h1", { className: "text-2xl font-bold tracking-tight", children: app.name }),
              _jsx("p", {
                className: "text-xs text-muted-foreground font-mono",
                children: app.slug,
              }),
            ],
          }),
          _jsx(Badge, {
            variant: app.status === "active" ? "default" : "outline",
            children: app.status === "active" ? "已启用" : "草稿",
          }),
          app.registry &&
            _jsx(Badge, { variant: "outline", children: "\u6765\u81EA\u5E94\u7528\u5E02\u573A" }),
          app.listing === "listed"
            ? _jsx(Badge, { variant: "default", children: "\u5DF2\u4E0A\u67B6" })
            : app.listing === "pending"
              ? _jsx(Badge, { variant: "outline", children: "\u5BA1\u6838\u4E2D" })
              : app.listing === "rejected"
                ? _jsx(Badge, { variant: "destructive", children: "\u5DF2\u62D2\u7EDD" })
                : null,
        ],
      }),
      _jsx("div", {
        className: "md:hidden",
        children: _jsx("select", {
          value: section,
          onChange: (e) => setSection(e.target.value),
          className: "w-full h-9 px-3 rounded-md border bg-background text-sm",
          "aria-label": "\u9009\u62E9\u8BBE\u7F6E\u9875\u9762",
          children: NAV_SECTIONS.flatMap((g) => g.items).map((item) =>
            _jsx("option", { value: item.key, children: item.label }, item.key),
          ),
        }),
      }),
      _jsxs("div", {
        className: "flex gap-8",
        children: [
          _jsx("nav", {
            className: "hidden md:block w-52 shrink-0 space-y-6",
            children: NAV_SECTIONS.map((group) =>
              _jsxs(
                "div",
                {
                  className: "space-y-1",
                  children: [
                    _jsx("p", {
                      className:
                        "text-xs font-bold uppercase tracking-widest text-muted-foreground px-2 mb-2",
                      children: group.group,
                    }),
                    group.items.map((item) =>
                      _jsxs(
                        Button,
                        {
                          variant: "ghost",
                          onClick: () => setSection(item.key),
                          className: `w-full justify-start gap-2 ${
                            section === item.key
                              ? "bg-primary/10 text-primary font-medium hover:bg-primary/10 hover:text-primary"
                              : "text-muted-foreground"
                          }`,
                          children: [
                            _jsx(item.icon, { className: "h-4 w-4 shrink-0" }),
                            item.label,
                          ],
                        },
                        item.key,
                      ),
                    ),
                  ],
                },
                group.group,
              ),
            ),
          }),
          _jsxs("div", {
            className: "flex-1 min-w-0",
            children: [
              section === "basic-info" &&
                _jsx(
                  BasicInfoSection,
                  { app: app, onUpdate: refreshApp, backPath: backPath },
                  app.updated_at,
                ),
              section === "install-app" && _jsx(InstallAppSection, { appId: id }),
              section === "distribution" &&
                _jsx(DistributionSection, { app: app, onUpdate: refreshApp }),
              section === "event-subscriptions" &&
                _jsx(EventSubscriptionsSection, { app: app, onUpdate: refreshApp }),
              section === "commands" && _jsx(ToolsEditor, { app: app, onUpdate: refreshApp }),
              section === "oauth-permissions" &&
                _jsx(OAuthPermissionsSection, { app: app, onUpdate: refreshApp }),
            ],
          }),
        ],
      }),
    ],
  });
}
// ==================== Basic Information (merged Settings + Credentials) ====================
function BasicInfoSection({ app, onUpdate, backPath }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { confirm, ConfirmDialog } = useConfirm();
  const [form, setForm] = useState({
    name: app.name || "",
    description: app.description || "",
    icon: app.icon || "",
    homepage: app.homepage || "",
    version: app.version || "",
    readme: app.readme || "",
    guide: app.guide || "",
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  async function handleSave(e) {
    e.preventDefault();
    setError("");
    setSuccess("");
    setSaving(true);
    try {
      await api.updateApp(app.id, form);
      setSuccess("已保存");
      onUpdate();
    } catch (err) {
      setError(err.message);
    }
    setSaving(false);
  }
  async function handleDelete() {
    const ok = await confirm({
      title: "删除确认",
      description: "确定删除此 App？所有安装也将被移除。",
      confirmText: "删除",
      variant: "destructive",
    });
    if (!ok) return;
    try {
      await api.deleteApp(app.id);
      invalidateAllAppQueries(queryClient);
      queryClient.removeQueries({ queryKey: queryKeys.apps.detail(app.id) });
      navigate(backPath);
    } catch {}
  }
  return _jsxs("div", {
    className: "space-y-6",
    children: [
      ConfirmDialog,
      _jsxs("div", {
        children: [
          _jsx("h2", {
            className: "text-base font-semibold",
            children: "\u57FA\u672C\u4FE1\u606F",
          }),
          _jsx("p", {
            className: "text-sm text-muted-foreground mt-1",
            children: "\u5E94\u7528\u7684\u57FA\u672C\u4FE1\u606F\u548C\u51ED\u8BC1\u3002",
          }),
        ],
      }),
      _jsxs(Card, {
        children: [
          _jsx(CardHeader, { children: _jsx(CardTitle, { children: "\u5C55\u793A\u4FE1\u606F" }) }),
          _jsx(CardContent, {
            children: _jsxs("form", {
              onSubmit: handleSave,
              className: "space-y-2",
              children: [
                _jsx(Input, {
                  placeholder: "\u540D\u79F0",
                  value: form.name,
                  onChange: (e) => setForm((f) => ({ ...f, name: e.target.value })),
                  className: "h-8 text-xs",
                  disabled: !!app.registry,
                }),
                _jsx(Input, {
                  placeholder: "\u63CF\u8FF0",
                  value: form.description,
                  onChange: (e) => setForm((f) => ({ ...f, description: e.target.value })),
                  className: "h-8 text-xs",
                  disabled: !!app.registry,
                }),
                _jsx(Input, {
                  placeholder: "\u56FE\u6807 (emoji \u6216 URL)",
                  value: form.icon,
                  onChange: (e) => setForm((f) => ({ ...f, icon: e.target.value })),
                  className: "h-8 text-xs",
                  disabled: !!app.registry,
                }),
                _jsx(Input, {
                  placeholder: "\u4E3B\u9875 URL",
                  value: form.homepage,
                  onChange: (e) => setForm((f) => ({ ...f, homepage: e.target.value })),
                  className: "h-8 text-xs",
                  disabled: !!app.registry,
                }),
                _jsx(Input, {
                  placeholder: "\u7248\u672C\u53F7 (\u5982 1.0.0)",
                  value: form.version,
                  onChange: (e) => setForm((f) => ({ ...f, version: e.target.value })),
                  className: "h-8 text-xs",
                  disabled: !!app.registry,
                }),
                _jsx("textarea", {
                  placeholder: "\u8BF4\u660E\u6587\u6863 (Readme)",
                  value: form.readme,
                  onChange: (e) => setForm((f) => ({ ...f, readme: e.target.value })),
                  className:
                    "w-full min-h-[80px] px-3 py-2 rounded-md border bg-background text-xs font-mono resize-y",
                  disabled: !!app.registry,
                }),
                _jsx("textarea", {
                  placeholder: "\u4F7F\u7528\u6307\u5357 (Guide)",
                  value: form.guide,
                  onChange: (e) => setForm((f) => ({ ...f, guide: e.target.value })),
                  className:
                    "w-full min-h-[80px] px-3 py-2 rounded-md border bg-background text-xs font-mono resize-y",
                  disabled: !!app.registry,
                }),
                !app.registry
                  ? _jsxs("div", {
                      className: "flex items-center justify-between",
                      children: [
                        _jsxs("div", {
                          children: [
                            error
                              ? _jsx("span", {
                                  className: "text-xs text-destructive",
                                  children: error,
                                })
                              : null,
                            success
                              ? _jsx("span", {
                                  className: "text-xs text-primary",
                                  children: success,
                                })
                              : null,
                          ],
                        }),
                        _jsx(Button, {
                          type: "submit",
                          size: "sm",
                          disabled: saving,
                          children: saving ? "..." : "保存",
                        }),
                      ],
                    })
                  : null,
              ],
            }),
          }),
        ],
      }),
      app.registry
        ? _jsx(Card, {
            children: _jsx(CardContent, {
              children: _jsxs("div", {
                className: "flex items-center gap-2",
                children: [
                  _jsx(Badge, {
                    variant: "outline",
                    children: "\u6765\u81EA\u5E94\u7528\u5E02\u573A",
                  }),
                  _jsx("span", {
                    className: "text-xs text-muted-foreground",
                    children:
                      "\u6B64\u5E94\u7528\u6765\u81EA\u5E94\u7528\u5E02\u573A Registry\uFF0C\u914D\u7F6E\u4E0D\u53EF\u7F16\u8F91\u3002",
                  }),
                ],
              }),
            }),
          })
        : null,
      app.registry === "builtin" ? _jsx(IntegrationTokenGuide, { app: app }) : null,
      app.readme
        ? _jsxs(Card, {
            children: [
              _jsx(CardHeader, {
                children: _jsx(CardTitle, { children: "\u8BF4\u660E\u6587\u6863" }),
              }),
              _jsx(CardContent, {
                children: _jsx("div", {
                  className:
                    "text-sm text-muted-foreground whitespace-pre-wrap font-mono leading-relaxed",
                  children: app.readme
                    .replace(/\{hub_url\}/g, window.location.origin)
                    .replace(/\{your_token\}/g, "<your_token>"),
                }),
              }),
            ],
          })
        : null,
      app.registry === "builtin" && !app.readme
        ? _jsxs(Card, {
            children: [
              _jsxs(CardHeader, {
                children: [
                  _jsx(CardTitle, { children: "\u4F7F\u7528\u8BF4\u660E" }),
                  _jsx(CardDescription, {
                    children:
                      "\u6B64\u5E94\u7528\u4E3A Integration \u7C7B\u578B\uFF0C\u4F7F\u7528 Token \u8FDB\u884C API \u8C03\u7528\u3002\u8BF7\u5728\u5B89\u88C5\u7BA1\u7406\u4E2D\u67E5\u770B Token\u3002",
                  }),
                ],
              }),
              _jsxs(CardContent, {
                className: "space-y-2 text-xs font-mono text-muted-foreground",
                children: [
                  _jsx("p", {
                    className: "font-sans text-xs font-medium text-foreground",
                    children: "HTTP \u53D1\u6D88\u606F",
                  }),
                  _jsx("pre", {
                    className:
                      "p-2 rounded-md bg-muted/30 border overflow-x-auto whitespace-pre-wrap",
                    children: `curl -X POST ${window.location.origin}/bot/v1/message/send \\
  -H "Authorization: Bearer <your_token>" \\
  -d '{"content":"hello"}'`,
                  }),
                  _jsx("p", {
                    className: "font-sans text-xs font-medium text-foreground",
                    children: "WebSocket \u8FDE\u63A5",
                  }),
                  _jsx("pre", {
                    className:
                      "p-2 rounded-md bg-muted/30 border overflow-x-auto whitespace-pre-wrap",
                    children: `${window.location.protocol === "https:" ? "wss" : "ws"}://${window.location.host}/bot/v1/ws?token=<your_token>`,
                  }),
                ],
              }),
            ],
          })
        : null,
      _jsxs(Card, {
        children: [
          _jsxs(CardHeader, {
            children: [
              _jsx(CardTitle, { children: "\u5E94\u7528\u51ED\u8BC1" }),
              _jsx(CardDescription, {
                children:
                  "\u8FD9\u4E9B\u51ED\u8BC1\u7528\u4E8E\u4F60\u7684 App \u4E0E Hub \u4E4B\u95F4\u7684\u5B89\u5168\u901A\u4FE1\u3002\u8BF7\u59A5\u5584\u4FDD\u7BA1\uFF0C\u4E0D\u8981\u6CC4\u9732\u3002",
              }),
            ],
          }),
          _jsx(CardContent, {
            children: app.webhook_secret
              ? _jsx(SecretField, {
                  label: "Webhook Secret",
                  value: app.webhook_secret,
                  description:
                    "Hub \u4F7F\u7528\u6B64\u5BC6\u94A5\u5BF9\u63A8\u9001\u4E8B\u4EF6\u7B7E\u540D\uFF0CApp \u7528\u5B83\u9A8C\u8BC1\u8BF7\u6C42\u6765\u6E90",
                })
              : _jsx("p", {
                  className: "text-xs text-muted-foreground italic",
                  children: "\u51ED\u8BC1\u4EC5\u5BF9 App \u6240\u6709\u8005\u53EF\u89C1\u3002",
                }),
          }),
        ],
      }),
      _jsxs(Card, {
        children: [
          _jsxs(CardHeader, {
            children: [
              _jsx(CardTitle, {
                className: "text-destructive",
                children: "\u5220\u9664\u5E94\u7528",
              }),
              _jsx(CardDescription, {
                children:
                  "\u5220\u9664\u540E\u6240\u6709\u5B89\u88C5\u4E5F\u5C06\u88AB\u79FB\u9664\uFF0C\u6B64\u64CD\u4F5C\u4E0D\u53EF\u64A4\u9500\u3002",
              }),
            ],
          }),
          _jsx(CardContent, {
            children: _jsxs(Button, {
              variant: "destructive",
              size: "sm",
              onClick: handleDelete,
              children: [_jsx(Trash2, { className: "w-3.5 h-3.5 mr-1" }), " \u5220\u9664 App"],
            }),
          }),
        ],
      }),
    ],
  });
}
function IntegrationTokenGuide({ app }) {
  const hubUrl = window.location.origin;
  return _jsxs(Card, {
    children: [
      _jsxs(CardHeader, {
        children: [
          _jsx(CardTitle, { children: "Integration Token \u4F7F\u7528\u6307\u5357" }),
          _jsx(CardDescription, {
            children:
              "\u6B64\u5E94\u7528\u4E3A Integration \u7C7B\u578B\u3002\u5B89\u88C5\u5B9E\u4F8B\u7684 Token \u53EF\u5728\u300C\u5B89\u88C5\u7BA1\u7406\u300D\u4E2D\u67E5\u770B\u3002",
          }),
        ],
      }),
      _jsxs(CardContent, {
        className: "space-y-3 text-xs",
        children: [
          _jsxs("div", {
            className: "space-y-1",
            children: [
              _jsx("p", {
                className: "font-medium text-foreground",
                children: "HTTP \u53D1\u6D88\u606F",
              }),
              _jsx("pre", {
                className:
                  "p-2 rounded-md bg-muted/30 border font-mono overflow-x-auto whitespace-pre-wrap",
                children: `curl -X POST ${hubUrl}/bot/v1/message/send \\
  -H "Authorization: Bearer {token}" \\
  -d '{"content":"hello"}'`,
              }),
            ],
          }),
          _jsxs("div", {
            className: "space-y-1",
            children: [
              _jsx("p", {
                className: "font-medium text-foreground",
                children: "WebSocket \u8FDE\u63A5",
              }),
              _jsx("pre", {
                className:
                  "p-2 rounded-md bg-muted/30 border font-mono overflow-x-auto whitespace-pre-wrap",
                children: `${window.location.protocol === "https:" ? "wss" : "ws"}://${window.location.host}/bot/v1/ws?token={token}`,
              }),
            ],
          }),
        ],
      }),
    ],
  });
}
function SecretField({ label, value, description }) {
  const [show, setShow] = useState(false);
  const [copied, setCopied] = useState(false);
  const { toast } = useToast();
  const masked = value ? value.slice(0, 8) + "..." + value.slice(-4) : "---";
  function handleCopy() {
    if (!navigator.clipboard?.writeText) {
      toast({
        variant: "destructive",
        title: "复制失败",
        description: "当前浏览器不支持自动复制，请手动选中复制",
      });
      return;
    }
    navigator.clipboard
      .writeText(value)
      .then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      })
      .catch(() => {
        toast({ variant: "destructive", title: "复制失败", description: "请手动选中复制" });
      });
  }
  return _jsxs("div", {
    className: "space-y-1",
    children: [
      _jsx("p", { className: "text-xs font-medium", children: label }),
      description &&
        _jsx("p", { className: "text-xs text-muted-foreground", children: description }),
      _jsxs("div", {
        className: "flex items-center gap-2 p-2 rounded-md border bg-background",
        children: [
          _jsx("code", {
            className: "text-xs font-mono flex-1 break-all",
            children: show ? value : masked,
          }),
          _jsx(Button, {
            variant: "ghost",
            size: "icon-xs",
            onClick: () => setShow(!show),
            "aria-label": show ? "隐藏" : "显示",
            children: show
              ? _jsx(EyeOff, { className: "w-3.5 h-3.5" })
              : _jsx(Eye, { className: "w-3.5 h-3.5" }),
          }),
          _jsx(Button, {
            variant: "ghost",
            size: "icon-xs",
            onClick: handleCopy,
            "aria-label": "\u590D\u5236",
            children: copied
              ? _jsx(Check, { className: "w-3.5 h-3.5 text-primary" })
              : _jsx(Copy, { className: "w-3.5 h-3.5" }),
          }),
        ],
      }),
    ],
  });
}
// ==================== Install App ====================
function InstallAppSection({ appId }) {
  const [installations, setInstallations] = useState([]);
  const [bots, setBots] = useState([]);
  const [botId, setBotId] = useState("");
  const [handle, setHandle] = useState("");
  const [installing, setInstalling] = useState(false);
  const { toast } = useToast();
  const { confirm, ConfirmDialog } = useConfirm();
  const queryClient = useQueryClient();
  async function load() {
    try {
      setInstallations((await api.listInstallations(appId)) || []);
    } catch {}
  }
  useEffect(() => {
    load();
    api.listBots().then((l) => {
      const items = l || [];
      setBots(items);
      if (items.length) setBotId(items[0].id);
    });
  }, [appId]);
  async function handleInstall() {
    if (!botId || !handle.trim()) return;
    setInstalling(true);
    try {
      await api.installApp(appId, { bot_id: botId, handle: handle.trim() });
      toast({ title: "安装成功" });
      setHandle("");
      load();
      invalidateAllAppQueries(queryClient, botId);
    } catch (e) {
      toast({ variant: "destructive", title: "安装失败", description: e.message });
    }
    setInstalling(false);
  }
  async function handleDelete(instId) {
    const ok = await confirm({
      title: "卸载确认",
      description: "确定卸载此安装？",
      confirmText: "卸载",
      variant: "destructive",
    });
    if (!ok) return;
    try {
      await api.deleteInstallation(appId, instId);
      toast({ title: "已卸载" });
      load();
      invalidateAllAppQueries(queryClient);
    } catch (e) {
      toast({ variant: "destructive", title: "卸载失败", description: e.message });
    }
  }
  return _jsxs("div", {
    className: "space-y-6",
    children: [
      ConfirmDialog,
      _jsxs("div", {
        children: [
          _jsx("h2", {
            className: "text-base font-semibold",
            children: "\u5B89\u88C5\u7BA1\u7406",
          }),
          _jsx("p", {
            className: "text-sm text-muted-foreground mt-1",
            children:
              "\u6240\u6709\u5B89\u88C5\u4E86\u6B64\u5E94\u7528\u7684\u8D26\u53F7\u3002\u6BCF\u4E2A\u5B89\u88C5\u5B9E\u4F8B\u6709\u72EC\u7ACB\u7684 app_token \u548C handle\u3002",
          }),
        ],
      }),
      _jsxs(Card, {
        children: [
          _jsx(CardHeader, {
            children: _jsx(CardTitle, { children: "\u5B89\u88C5\u5230\u8D26\u53F7" }),
          }),
          _jsx(CardContent, {
            children:
              bots.length === 0
                ? _jsx("p", {
                    className: "text-sm text-muted-foreground",
                    children:
                      "\u8BF7\u5148\u521B\u5EFA\u4E00\u4E2A\u8D26\u53F7\uFF0C\u7136\u540E\u518D\u5B89\u88C5\u5E94\u7528\u3002",
                  })
                : _jsxs("div", {
                    className: "flex gap-2 items-end",
                    children: [
                      _jsxs("div", {
                        className: "flex-1 space-y-1",
                        children: [
                          _jsx("label", {
                            htmlFor: "install-bot-select",
                            className: "text-xs text-muted-foreground",
                            children: "\u8D26\u53F7",
                          }),
                          _jsx("select", {
                            id: "install-bot-select",
                            value: botId,
                            onChange: (e) => setBotId(e.target.value),
                            className:
                              "w-full h-8 px-2 rounded-md border bg-background text-xs outline-none",
                            children: bots.map((b) =>
                              _jsx("option", { value: b.id, children: botDisplayName(b) }, b.id),
                            ),
                          }),
                        ],
                      }),
                      _jsxs("div", {
                        className: "flex-1 space-y-1",
                        children: [
                          _jsx("label", {
                            htmlFor: "install-handle-input",
                            className: "text-xs text-muted-foreground",
                            children: "Handle",
                          }),
                          _jsx(Input, {
                            id: "install-handle-input",
                            value: handle,
                            onChange: (e) => setHandle(e.target.value),
                            placeholder: "\u5982 notify",
                            className: "h-8 text-xs font-mono",
                          }),
                        ],
                      }),
                      _jsxs(Button, {
                        size: "sm",
                        onClick: handleInstall,
                        disabled: installing || !botId || !handle.trim(),
                        className: "h-8",
                        children: [
                          installing && _jsx(Loader2, { className: "h-3 w-3 animate-spin mr-1" }),
                          "\u5B89\u88C5",
                        ],
                      }),
                    ],
                  }),
          }),
        ],
      }),
      installations.length === 0
        ? _jsx("p", {
            className: "text-center text-sm text-muted-foreground py-8",
            children: "\u6682\u65E0\u5B89\u88C5",
          })
        : _jsx("div", {
            className: "space-y-2",
            children: installations.map((ins) =>
              _jsx(
                Card,
                {
                  children: _jsxs(CardContent, {
                    className: "flex items-center justify-between",
                    children: [
                      _jsxs("div", {
                        className: "space-y-0.5",
                        children: [
                          _jsxs("div", {
                            className: "flex items-center gap-2",
                            children: [
                              _jsx("span", {
                                className: "text-sm font-medium",
                                children: ins.bot_name || ins.bot_id,
                              }),
                              ins.handle
                                ? _jsxs(Badge, {
                                    variant: "outline",
                                    className: "text-xs font-mono",
                                    children: ["@", ins.handle],
                                  })
                                : null,
                            ],
                          }),
                          _jsx("p", {
                            className: "text-xs text-muted-foreground font-mono",
                            children: ins.id,
                          }),
                        ],
                      }),
                      _jsxs("div", {
                        className: "flex items-center gap-2",
                        children: [
                          _jsx(Badge, {
                            variant: ins.enabled ? "default" : "outline",
                            children: ins.enabled ? "启用" : "禁用",
                          }),
                          _jsx(Button, {
                            variant: "ghost",
                            size: "sm",
                            className: "h-7 text-xs text-destructive",
                            "aria-label": "\u5378\u8F7D",
                            onClick: () => handleDelete(ins.id),
                            children: _jsx(Trash2, { className: "w-3 h-3" }),
                          }),
                        ],
                      }),
                    ],
                  }),
                },
                ins.id,
              ),
            ),
          }),
    ],
  });
}
// ==================== Manage Distribution ====================
const ACTION_LABELS = {
  request: "申请上架",
  approve: "通过",
  reject: "拒绝",
  withdraw: "撤回",
  auto_revert: "自动回退",
  admin_set: "管理员操作",
};
const ACTION_VARIANTS = {
  request: "outline",
  approve: "default",
  reject: "destructive",
  withdraw: "secondary",
  auto_revert: "secondary",
  admin_set: "outline",
};
function DistributionSection({ app, onUpdate }) {
  const [loading, setLoading] = useState(false);
  const [reviews, setReviews] = useState([]);
  const { toast } = useToast();
  useEffect(() => {
    api
      .listAppReviews(app.id)
      .then(setReviews)
      .catch(() => {});
  }, [app.id]);
  async function handleRequestListing() {
    setLoading(true);
    try {
      await api.requestListing(app.id);
      onUpdate();
      api
        .listAppReviews(app.id)
        .then(setReviews)
        .catch(() => {});
    } catch (e) {
      toast({ variant: "destructive", title: "提交审核失败", description: e.message });
    }
    setLoading(false);
  }
  return _jsxs("div", {
    className: "space-y-6",
    children: [
      _jsxs("div", {
        children: [
          _jsx("h2", {
            className: "text-base font-semibold",
            children: "\u5206\u53D1\u7BA1\u7406",
          }),
          _jsx("p", {
            className: "text-sm text-muted-foreground mt-1",
            children:
              "\u7BA1\u7406\u5E94\u7528\u7684\u4E0A\u67B6\u72B6\u6001\uFF0C\u4E0A\u67B6\u540E\u5176\u4ED6\u7528\u6237\u53EF\u4EE5\u641C\u7D22\u5E76\u5B89\u88C5\u3002",
          }),
        ],
      }),
      _jsxs(Card, {
        children: [
          _jsx(CardHeader, { children: _jsx(CardTitle, { children: "\u5E94\u7528\u5E02\u573A" }) }),
          _jsx(CardContent, {
            className: "space-y-3",
            children:
              app.listing === "listed"
                ? _jsxs("div", {
                    className: "flex items-center gap-2",
                    children: [
                      _jsx(Badge, { variant: "default", children: "\u5DF2\u4E0A\u67B6" }),
                      _jsx("span", {
                        className: "text-xs text-muted-foreground",
                        children:
                          "\u4F60\u7684\u5E94\u7528\u5DF2\u5728\u5E94\u7528\u5E02\u573A\u4E2D\u5C55\u793A\u3002",
                      }),
                    ],
                  })
                : app.listing === "pending"
                  ? _jsx("div", {
                      className: "space-y-2",
                      children: _jsxs("div", {
                        className: "flex items-center gap-2",
                        children: [
                          _jsx(Badge, { variant: "outline", children: "\u5BA1\u6838\u4E2D" }),
                          _jsx("span", {
                            className: "text-xs text-muted-foreground",
                            children:
                              "\u4E0A\u67B6\u7533\u8BF7\u5DF2\u63D0\u4EA4\uFF0C\u7B49\u5F85\u7BA1\u7406\u5458\u5BA1\u6838\u3002",
                          }),
                        ],
                      }),
                    })
                  : app.listing === "rejected"
                    ? _jsxs("div", {
                        className: "space-y-3",
                        children: [
                          _jsxs("div", {
                            className: "flex items-center gap-2",
                            children: [
                              _jsx(Badge, {
                                variant: "destructive",
                                children: "\u5DF2\u62D2\u7EDD",
                              }),
                              app.listing_reject_reason
                                ? _jsxs("span", {
                                    className: "text-xs text-destructive",
                                    children: ["\u539F\u56E0\uFF1A", app.listing_reject_reason],
                                  })
                                : null,
                            ],
                          }),
                          _jsx(Button, {
                            size: "sm",
                            variant: "outline",
                            disabled: loading,
                            onClick: handleRequestListing,
                            children: loading ? "..." : "重新申请",
                          }),
                        ],
                      })
                    : _jsxs("div", {
                        className: "space-y-3",
                        children: [
                          _jsx("p", {
                            className: "text-xs text-muted-foreground",
                            children:
                              "\u4F60\u7684\u5E94\u7528\u5C1A\u672A\u4E0A\u67B6\u3002\u4E0A\u67B6\u540E\u5176\u4ED6\u7528\u6237\u53EF\u4EE5\u641C\u7D22\u5E76\u5B89\u88C5\u3002",
                          }),
                          _jsx(Button, {
                            size: "sm",
                            variant: "outline",
                            disabled: loading,
                            onClick: handleRequestListing,
                            children: loading ? "..." : "申请上架",
                          }),
                        ],
                      }),
          }),
        ],
      }),
      reviews.length > 0 &&
        _jsxs(Card, {
          children: [
            _jsx(CardHeader, {
              children: _jsx(CardTitle, { children: "\u5BA1\u6838\u8BB0\u5F55" }),
            }),
            _jsx(CardContent, {
              children: _jsx("div", {
                className: "space-y-3",
                children: reviews.map((review) =>
                  _jsxs(
                    "div",
                    {
                      className: "flex items-start gap-3 text-sm",
                      children: [
                        _jsx("span", {
                          className: "text-xs text-muted-foreground whitespace-nowrap mt-0.5",
                          children: new Date(review.created_at * 1000).toLocaleString(),
                        }),
                        _jsx(Badge, {
                          variant: ACTION_VARIANTS[review.action] || "outline",
                          className: "shrink-0",
                          children: ACTION_LABELS[review.action] || review.action,
                        }),
                        review.version &&
                          _jsxs("span", {
                            className: "text-xs text-muted-foreground",
                            children: ["v", review.version],
                          }),
                        review.reason &&
                          _jsx("span", {
                            className: "text-xs text-muted-foreground truncate",
                            children: review.reason,
                          }),
                      ],
                    },
                    review.id,
                  ),
                ),
              }),
            }),
          ],
        }),
    ],
  });
}
// ==================== Event Subscriptions ====================
function EventSubscriptionsSection({ app, onUpdate }) {
  const [webhookUrl, setWebhookUrl] = useState(app.webhook_url || "");
  const [events, setEvents] = useState(app.events || []);
  const [saving, setSaving] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const { toast } = useToast();
  function toggleEvent(key) {
    setEvents((prev) => (prev.includes(key) ? prev.filter((e) => e !== key) : [...prev, key]));
  }
  async function handleSave() {
    setSaving(true);
    try {
      await api.updateApp(app.id, { webhook_url: webhookUrl, events });
      toast({ title: "已保存" });
      onUpdate();
    } catch (e) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
    setSaving(false);
  }
  async function handleVerify() {
    setVerifying(true);
    try {
      if (webhookUrl !== (app.webhook_url || "")) {
        await api.updateApp(app.id, { webhook_url: webhookUrl });
      }
      await api.verifyAppUrl(app.id);
      toast({ title: "URL 验证成功" });
      onUpdate();
    } catch (e) {
      toast({ variant: "destructive", title: "验证失败", description: e.message });
    }
    setVerifying(false);
  }
  return _jsxs("div", {
    className: "space-y-6",
    children: [
      _jsxs("div", {
        children: [
          _jsx("h2", {
            className: "text-base font-semibold",
            children: "\u4E8B\u4EF6\u8BA2\u9605",
          }),
          _jsx("p", {
            className: "text-sm text-muted-foreground mt-1",
            children:
              "\u914D\u7F6E\u4E8B\u4EF6\u63A8\u9001 URL \u548C\u8BA2\u9605\u7684\u4E8B\u4EF6\u7C7B\u578B\u3002",
          }),
        ],
      }),
      _jsxs(Card, {
        children: [
          _jsxs(CardHeader, {
            children: [
              _jsx(CardTitle, { children: "\u8F6C\u53D1\u5730\u5740" }),
              _jsx(CardDescription, {
                children: "Bot \u6536\u5230\u7684\u6D88\u606F\u5C06 POST \u5230\u6B64\u5730\u5740",
              }),
            ],
          }),
          _jsxs(CardContent, {
            className: "space-y-3",
            children: [
              _jsxs("div", {
                className: "flex gap-2",
                children: [
                  _jsx(Input, {
                    placeholder: "https://your-app.example.com/webhook",
                    value: webhookUrl,
                    onChange: (e) => setWebhookUrl(e.target.value),
                    className: "h-8 text-xs font-mono flex-1",
                  }),
                  _jsxs(Button, {
                    size: "sm",
                    variant: "outline",
                    onClick: handleVerify,
                    disabled: verifying || !webhookUrl.trim(),
                    className: "h-8",
                    children: [
                      verifying
                        ? _jsx(Loader2, { className: "h-3 w-3 animate-spin" })
                        : _jsx(ExternalLink, { className: "h-3 w-3 mr-1" }),
                      "\u9A8C\u8BC1",
                    ],
                  }),
                ],
              }),
              app.url_verified
                ? _jsxs("div", {
                    className: "flex items-center gap-1 text-xs text-primary",
                    children: [
                      _jsx(ShieldCheck, { className: "w-3 h-3" }),
                      " URL \u5DF2\u9A8C\u8BC1",
                    ],
                  })
                : null,
            ],
          }),
        ],
      }),
      _jsxs(Card, {
        children: [
          _jsx(CardHeader, { children: _jsx(CardTitle, { children: "\u8BA2\u9605\u4E8B\u4EF6" }) }),
          _jsxs(CardContent, {
            className: "space-y-3",
            children: [
              _jsx("div", {
                className: "grid grid-cols-2 gap-2",
                children: EVENT_TYPES.map((et) =>
                  _jsxs(
                    "label",
                    {
                      className: "flex items-center gap-2 cursor-pointer",
                      children: [
                        _jsx("input", {
                          type: "checkbox",
                          checked: events.includes(et.key),
                          onChange: () => toggleEvent(et.key),
                          className: "w-3.5 h-3.5 accent-primary",
                        }),
                        _jsx("span", { className: "text-xs", children: et.label }),
                        _jsx("span", {
                          className: "text-xs text-muted-foreground font-mono",
                          children: et.key,
                        }),
                      ],
                    },
                    et.key,
                  ),
                ),
              }),
              _jsx(Button, {
                size: "sm",
                onClick: handleSave,
                disabled: saving,
                children: saving ? "..." : "保存",
              }),
            ],
          }),
        ],
      }),
    ],
  });
}
// ==================== Commands / Tools ====================
function ToolsEditor({ app, onUpdate }) {
  const [tools, setTools] = useState(
    (app.tools || []).map((t) => ({
      ...t,
      parameters: t.parameters ? JSON.stringify(t.parameters, null, 2) : "",
    })),
  );
  const [saving, setSaving] = useState(false);
  const [mcpUrl, setMcpUrl] = useState("");
  const [mcpHeaders, setMcpHeaders] = useState("");
  const [importing, setImporting] = useState(false);
  const [showMcpImport, setShowMcpImport] = useState(false);
  const { toast } = useToast();
  const { confirm, ConfirmDialog } = useConfirm();
  function addTool() {
    setTools([...tools, { name: "", description: "", command: "", parameters: "" }]);
  }
  function removeTool(index) {
    setTools(tools.filter((_, i) => i !== index));
  }
  function updateTool(index, field, value) {
    setTools(tools.map((t, i) => (i === index ? { ...t, [field]: value } : t)));
  }
  async function handleImportMCP() {
    if (!mcpUrl.trim()) return;
    setImporting(true);
    try {
      let headers;
      if (mcpHeaders.trim()) {
        headers = {};
        for (const line of mcpHeaders.split("\n")) {
          const idx = line.indexOf(":");
          if (idx > 0) headers[line.slice(0, idx).trim()] = line.slice(idx + 1).trim();
        }
      }
      const result = await api.importMCP({ url: mcpUrl.trim(), headers });
      if (result.tools.length === 0) {
        toast({
          variant: "destructive",
          title: "未发现工具",
          description: "MCP 服务器未返回任何工具定义",
        });
      } else {
        if (tools.length > 0) {
          const ok = await confirm({
            title: "替换现有工具",
            description: `将从 MCP 服务器导入 ${result.tools.length} 个工具，替换当前全部 ${tools.length} 个工具。确定继续？`,
            confirmText: "替换",
          });
          if (!ok) {
            setImporting(false);
            return;
          }
        }
        const imported = result.tools.map((t) => ({
          name: t.name,
          description: t.description || "",
          command: "",
          parameters: t.parameters ? JSON.stringify(t.parameters, null, 2) : "",
        }));
        setTools(imported);
        setShowMcpImport(false);
        const serverInfo = result.server_name
          ? `${result.server_name}${result.server_version ? ` v${result.server_version}` : ""}`
          : "MCP 服务器";
        let desc = `来自 ${serverInfo}`;
        if (result.truncated) desc += "（工具数量超限，已截断）";
        toast({ title: `已导入 ${result.tools.length} 个工具`, description: desc });
      }
    } catch (e) {
      toast({ variant: "destructive", title: "导入失败", description: e.message });
    }
    setImporting(false);
  }
  async function handleSave() {
    setSaving(true);
    try {
      const payload = tools.map((t) => {
        const tool = { name: t.name, description: t.description };
        if (t.command) tool.command = t.command.replace(/^\//, "");
        if (t.parameters?.trim()) tool.parameters = JSON.parse(t.parameters);
        return tool;
      });
      await api.updateApp(app.id, { tools: payload });
      onUpdate();
    } catch {}
    setSaving(false);
  }
  return _jsxs("div", {
    className: "space-y-6",
    children: [
      ConfirmDialog,
      _jsxs("div", {
        className: "flex items-center justify-between",
        children: [
          _jsxs("div", {
            children: [
              _jsx("h2", {
                className: "text-base font-semibold",
                children: "\u547D\u4EE4 / \u5DE5\u5177",
              }),
              _jsx("p", {
                className: "text-sm text-muted-foreground mt-1",
                children:
                  "\u5B9A\u4E49\u5E94\u7528\u7684\u5DE5\u5177\u548C\u547D\u4EE4\uFF0C\u7528\u6237\u901A\u8FC7 /command \u89E6\u53D1\u3002",
              }),
            ],
          }),
          _jsxs("div", {
            className: "flex gap-2",
            children: [
              _jsxs(Button, {
                variant: "outline",
                size: "sm",
                className: "h-7 text-xs",
                onClick: () => setShowMcpImport(!showMcpImport),
                children: [
                  _jsx(Download, { className: "w-3 h-3 mr-1" }),
                  " \u4ECE MCP \u5BFC\u5165",
                ],
              }),
              _jsxs(Button, {
                variant: "outline",
                size: "sm",
                className: "h-7 text-xs",
                onClick: addTool,
                children: [_jsx(Plus, { className: "w-3 h-3 mr-1" }), " \u6DFB\u52A0"],
              }),
            ],
          }),
        ],
      }),
      showMcpImport &&
        _jsxs(Card, {
          children: [
            _jsxs(CardHeader, {
              children: [
                _jsx(CardTitle, { children: "\u4ECE MCP \u670D\u52A1\u5668\u5BFC\u5165" }),
                _jsx(CardDescription, {
                  children:
                    "\u8F93\u5165 MCP \u670D\u52A1\u5668\u7684 Streamable HTTP \u5730\u5740\uFF0C\u81EA\u52A8\u53D1\u73B0\u5E76\u5BFC\u5165\u5DE5\u5177\u5B9A\u4E49\u3002\u5BFC\u5165\u4F1A\u66FF\u6362\u5F53\u524D\u6240\u6709\u5DE5\u5177\u3002",
                }),
              ],
            }),
            _jsxs(CardContent, {
              className: "space-y-3",
              children: [
                _jsx(Input, {
                  placeholder: "https://your-mcp-server.example.com/mcp",
                  value: mcpUrl,
                  onChange: (e) => setMcpUrl(e.target.value),
                  className: "h-8 text-xs font-mono",
                  "aria-label": "MCP \u670D\u52A1\u5668\u5730\u5740",
                }),
                _jsx("textarea", {
                  placeholder:
                    "\u81EA\u5B9A\u4E49\u8BF7\u6C42\u5934\uFF08\u53EF\u9009\uFF0C\u6BCF\u884C\u4E00\u4E2A\uFF0C\u683C\u5F0F Key: Value\uFF09",
                  value: mcpHeaders,
                  onChange: (e) => setMcpHeaders(e.target.value),
                  rows: 2,
                  className:
                    "w-full rounded-md border border-input bg-transparent px-2 py-1 text-xs font-mono placeholder:text-muted-foreground/40 focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 resize-none",
                  "aria-label": "\u81EA\u5B9A\u4E49\u8BF7\u6C42\u5934",
                }),
                _jsxs("div", {
                  className: "flex gap-2",
                  children: [
                    _jsxs(Button, {
                      size: "sm",
                      onClick: handleImportMCP,
                      disabled: importing || !mcpUrl.trim(),
                      className: "h-7 text-xs",
                      children: [
                        importing && _jsx(Loader2, { className: "h-3 w-3 animate-spin mr-1" }),
                        importing ? "正在发现..." : "发现工具",
                      ],
                    }),
                    _jsx(Button, {
                      variant: "ghost",
                      size: "sm",
                      onClick: () => setShowMcpImport(false),
                      className: "h-7 text-xs",
                      children: "\u53D6\u6D88",
                    }),
                  ],
                }),
              ],
            }),
          ],
        }),
      tools.length === 0
        ? _jsx("p", {
            className: "text-xs text-muted-foreground",
            children:
              "\u6682\u65E0\u5DE5\u5177\u3002\u70B9\u51FB\u53F3\u4E0A\u89D2\u6DFB\u52A0\u3002",
          })
        : null,
      tools.map((tool, i) =>
        _jsx(
          Card,
          {
            children: _jsx(CardContent, {
              children: _jsxs("div", {
                className: "flex items-start gap-2",
                children: [
                  _jsxs("div", {
                    className: "flex-1 space-y-1",
                    children: [
                      _jsxs("div", {
                        className: "flex gap-1",
                        children: [
                          _jsx(Input, {
                            placeholder: "\u5DE5\u5177\u540D\uFF08\u5982 list_prs\uFF09",
                            value: tool.name,
                            onChange: (e) => updateTool(i, "name", e.target.value),
                            className: "h-7 text-xs font-mono flex-1",
                          }),
                          _jsx(Input, {
                            placeholder: "\u547D\u4EE4\u89E6\u53D1\uFF08\u5982 pr\uFF09",
                            value: tool.command,
                            onChange: (e) => updateTool(i, "command", e.target.value),
                            className: "h-7 text-xs font-mono w-36",
                          }),
                        ],
                      }),
                      _jsx(Input, {
                        placeholder: "\u63CF\u8FF0",
                        value: tool.description,
                        onChange: (e) => updateTool(i, "description", e.target.value),
                        className: "h-7 text-xs",
                      }),
                      _jsx("textarea", {
                        placeholder: "\u53C2\u6570 JSON Schema\uFF08\u53EF\u9009\uFF09",
                        value: tool.parameters,
                        onChange: (e) => updateTool(i, "parameters", e.target.value),
                        rows: 2,
                        className:
                          "w-full rounded-md border border-input bg-transparent px-2 py-1 text-xs font-mono placeholder:text-muted-foreground/40 focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 resize-none",
                      }),
                    ],
                  }),
                  _jsx(Button, {
                    type: "button",
                    variant: "ghost",
                    size: "icon-sm",
                    onClick: () => removeTool(i),
                    className: "mt-1 text-destructive hover:text-destructive",
                    "aria-label": "\u5220\u9664\u5DE5\u5177",
                    children: _jsx(Trash2, { className: "w-3.5 h-3.5" }),
                  }),
                ],
              }),
            }),
          },
          i,
        ),
      ),
      tools.length > 0
        ? _jsx(Button, {
            size: "sm",
            onClick: handleSave,
            disabled: saving,
            children: saving ? "..." : "保存工具",
          })
        : null,
    ],
  });
}
// ==================== OAuth & Permissions ====================
function OAuthPermissionsSection({ app, onUpdate }) {
  const [scopes, setScopes] = useState(app.scopes || []);
  const [saving, setSaving] = useState(false);
  const readScopes = SCOPES.filter((s) => s.category === "read");
  const writeScopes = SCOPES.filter((s) => s.category === "write");
  function toggleScope(key) {
    setScopes((prev) => (prev.includes(key) ? prev.filter((s) => s !== key) : [...prev, key]));
  }
  async function handleSave() {
    setSaving(true);
    try {
      await api.updateApp(app.id, { scopes });
      onUpdate();
    } catch {}
    setSaving(false);
  }
  return _jsxs("div", {
    className: "space-y-6",
    children: [
      _jsxs("div", {
        children: [
          _jsx("h2", { className: "text-base font-semibold", children: "OAuth \u6743\u9650" }),
          _jsx("p", {
            className: "text-sm text-muted-foreground mt-1",
            children:
              "\u7BA1\u7406\u5E94\u7528\u901A\u8FC7 Bot API \u8C03\u7528\u65F6\u6240\u9700\u7684\u6743\u9650\u8303\u56F4\u3002",
          }),
        ],
      }),
      _jsxs(Card, {
        children: [
          _jsxs(CardHeader, {
            children: [
              _jsx(CardTitle, { children: "\u6743\u9650\u8303\u56F4" }),
              _jsx(CardDescription, {
                children:
                  "\u5B9A\u4E49\u5E94\u7528\u80FD\u591F\u8BBF\u95EE\u548C\u6267\u884C\u7684\u64CD\u4F5C\u3002\u5B89\u88C5\u65F6\u7528\u6237\u5C06\u770B\u5230\u8FD9\u4E9B\u6743\u9650\u63CF\u8FF0\u3002",
              }),
            ],
          }),
          _jsxs(CardContent, {
            className: "space-y-4",
            children: [
              _jsxs("div", {
                className: "space-y-2",
                children: [
                  _jsxs("p", {
                    className: "text-xs font-medium flex items-center gap-1.5",
                    children: [
                      _jsx(Eye, { className: "h-3.5 w-3.5 text-muted-foreground" }),
                      " \u67E5\u770B\u4FE1\u606F",
                    ],
                  }),
                  readScopes.map((s) =>
                    _jsxs(
                      "label",
                      {
                        className:
                          "flex items-start gap-3 p-2 rounded-md border bg-background cursor-pointer hover:bg-muted/30 transition-colors",
                        children: [
                          _jsx("input", {
                            type: "checkbox",
                            checked: scopes.includes(s.key),
                            onChange: () => toggleScope(s.key),
                            className: "mt-0.5 accent-primary",
                          }),
                          _jsxs("div", {
                            children: [
                              _jsx("span", { className: "text-sm font-medium", children: s.label }),
                              _jsx("span", {
                                className: "text-xs text-muted-foreground font-mono ml-2",
                                children: s.key,
                              }),
                              _jsx("p", {
                                className: "text-xs text-muted-foreground mt-0.5",
                                children: s.description,
                              }),
                            ],
                          }),
                        ],
                      },
                      s.key,
                    ),
                  ),
                ],
              }),
              _jsxs("div", {
                className: "space-y-2",
                children: [
                  _jsxs("p", {
                    className: "text-xs font-medium flex items-center gap-1.5",
                    children: [
                      _jsx(Zap, { className: "h-3.5 w-3.5 text-primary" }),
                      " \u6267\u884C\u64CD\u4F5C",
                    ],
                  }),
                  writeScopes.map((s) =>
                    _jsxs(
                      "label",
                      {
                        className:
                          "flex items-start gap-3 p-2 rounded-md border bg-background cursor-pointer hover:bg-muted/30 transition-colors",
                        children: [
                          _jsx("input", {
                            type: "checkbox",
                            checked: scopes.includes(s.key),
                            onChange: () => toggleScope(s.key),
                            className: "mt-0.5 accent-primary",
                          }),
                          _jsxs("div", {
                            children: [
                              _jsx("span", { className: "text-sm font-medium", children: s.label }),
                              _jsx("span", {
                                className: "text-xs text-muted-foreground font-mono ml-2",
                                children: s.key,
                              }),
                              _jsx("p", {
                                className: "text-xs text-muted-foreground mt-0.5",
                                children: s.description,
                              }),
                            ],
                          }),
                        ],
                      },
                      s.key,
                    ),
                  ),
                ],
              }),
              _jsx(Button, {
                size: "sm",
                onClick: handleSave,
                disabled: saving,
                children: saving ? "..." : "保存更改",
              }),
            ],
          }),
        ],
      }),
    ],
  });
}
