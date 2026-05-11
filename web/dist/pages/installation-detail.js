import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect, useState, useCallback, useRef, useMemo } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { marked } from "marked";
import DOMPurify from "dompurify";
import {
  ArrowLeft,
  Eye,
  EyeOff,
  Copy,
  Check,
  ArrowRight,
  Loader2,
  Trash2,
  RefreshCw,
  Key,
  ScrollText,
  Terminal,
  Sliders,
  ChevronRight,
  ShieldCheck,
  Zap,
  ExternalLink,
} from "lucide-react";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "../components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "../components/ui/dialog";
import { Skeleton } from "../components/ui/skeleton";
import { Switch } from "../components/ui/switch";
import { Label } from "../components/ui/label";
import { api, botDisplayName } from "../lib/api";
import { useBotApps, useBots } from "@/hooks/use-bots";
import { useApp } from "@/hooks/use-apps";
import { useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "@/lib/query-keys";
import { invalidateAllAppQueries } from "@/hooks/use-apps";
import { SCOPE_DESCRIPTIONS, EVENT_TYPES } from "../lib/constants";
import { useToast } from "@/hooks/use-toast";
import { AppIcon } from "../components/app-icon";
import { ToolsDisplay, parseTools } from "../components/tools-display";
function buildNavSections(app, inst) {
  const items = [{ key: "token", label: "Token & 使用", icon: Key }];
  if (parseTools(app?.tools).length > 0 || parseTools(inst?.tools).length > 0) {
    items.push({ key: "tools", label: "命令 / 工具", icon: Terminal });
  }
  const instScopes = inst?.scopes || [];
  const events = app?.events || [];
  if (instScopes.length > 0 || events.length > 0) {
    items.push({ key: "permissions", label: "权限", icon: ShieldCheck });
  }
  if (app?.config_schema) {
    let parsed = {};
    try {
      parsed =
        typeof app.config_schema === "string"
          ? JSON.parse(app.config_schema || "{}")
          : app.config_schema || {};
    } catch {}
    if (Object.keys(parsed.properties || {}).length > 0) {
      items.push({ key: "app-config", label: "应用配置", icon: Sliders });
    }
  }
  items.push({ key: "config", label: "危险操作", icon: Trash2 });
  items.push({ key: "event-logs", label: "事件日志", icon: ScrollText });
  items.push({ key: "api-logs", label: "API 日志", icon: ScrollText });
  return items;
}
// ==================== Page ====================
export function InstallationDetailPage() {
  const { id: botId, iid } = useParams();
  const navigate = useNavigate();
  const { toast } = useToast();
  const queryClient = useQueryClient();
  // Server state
  const { data: installations = [], isLoading: installationsLoading } = useBotApps(botId);
  const inst = installations.find((i) => i.id === iid) ?? null;
  const { data: app } = useApp(inst?.app_id ?? "");
  const { data: allBots = [] } = useBots();
  const bot = allBots.find((b) => b.id === botId);
  const botName = bot ? botDisplayName(bot) : "";
  const loading = installationsLoading;
  const refreshInstallations = () =>
    queryClient.invalidateQueries({ queryKey: queryKeys.bots.apps(botId) });
  // Local UI state
  const [section, setSection] = useState("token");
  const [handle, setHandle] = useState("");
  const [enabled, setEnabled] = useState(true);
  const [enablingPending, setEnablingPending] = useState(false);
  // Sync local state when inst loads
  useEffect(() => {
    if (inst) {
      setHandle(inst.handle || "");
      setEnabled(inst.enabled ?? true);
    }
  }, [inst?.id, inst?.handle, inst?.enabled]);
  if (loading) {
    return _jsxs("div", {
      className: "space-y-6",
      children: [
        _jsx(Skeleton, { className: "h-20 w-full rounded-3xl" }),
        _jsx(Skeleton, { className: "h-64 w-full rounded-3xl" }),
      ],
    });
  }
  if (!inst || !app) {
    return _jsxs("div", {
      className: "py-20 text-center space-y-4",
      children: [
        _jsx("p", {
          className: "font-bold",
          children: "\u672A\u627E\u5230\u5B89\u88C5\u5B9E\u4F8B",
        }),
        _jsx(Button, {
          variant: "link",
          asChild: true,
          children: _jsx(Link, {
            to: `/dashboard/accounts/${botId}`,
            children: "\u8FD4\u56DE\u8D26\u53F7",
          }),
        }),
      ],
    });
  }
  const navItems = buildNavSections(app, inst);
  async function handleSaveHandle(newHandle) {
    const trimmed = newHandle.trim();
    if (trimmed === handle) return;
    if (!trimmed) {
      toast({ variant: "destructive", title: "Handle 不能为空" });
      return;
    }
    try {
      await api.updateInstallation(inst.app_id, inst.id, { handle: trimmed });
      setHandle(trimmed);
      refreshInstallations();
      toast({ title: "Handle 已保存" });
    } catch (e) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
  }
  async function handleToggleEnabled(val) {
    if (enablingPending) return;
    setEnablingPending(true);
    setEnabled(val);
    try {
      await api.updateInstallation(inst.app_id, inst.id, { enabled: val });
      refreshInstallations();
    } catch (e) {
      setEnabled(!val);
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    } finally {
      setEnablingPending(false);
    }
  }
  return _jsxs("div", {
    className: "space-y-6",
    children: [
      _jsxs("div", {
        className: "space-y-4",
        children: [
          _jsx(Button, {
            variant: "ghost",
            size: "sm",
            asChild: true,
            className: "gap-1.5 text-muted-foreground hover:text-foreground",
            children: _jsxs(Link, {
              to: `/dashboard/accounts/${botId}`,
              children: [_jsx(ArrowLeft, { className: "h-4 w-4" }), botName || "返回"],
            }),
          }),
          _jsxs("div", {
            className: "flex items-start gap-4",
            children: [
              _jsx(AppIcon, {
                icon: inst.app_icon || app.icon,
                iconUrl: inst.app_icon_url || app.icon_url,
                size: "h-14 w-14",
              }),
              _jsxs("div", {
                className: "flex-1 min-w-0 space-y-1",
                children: [
                  _jsxs("div", {
                    className: "flex items-center gap-3 flex-wrap",
                    children: [
                      _jsx("h1", {
                        className: "text-2xl font-bold tracking-tight",
                        children: inst.app_name || app.name,
                      }),
                      _jsx(InlineHandleEditor, { value: handle, onSave: handleSaveHandle }),
                      app.registry && app.registry !== "builtin"
                        ? _jsx(Badge, {
                            variant: "outline",
                            className: "rounded-full font-bold",
                            children: "\u6765\u81EA\u5E94\u7528\u5E02\u573A",
                          })
                        : null,
                      app.registry === "builtin"
                        ? _jsx(Badge, {
                            variant: "outline",
                            className: "rounded-full font-bold",
                            children: "\u5185\u7F6E\u5E94\u7528",
                          })
                        : null,
                      app.homepage && app.registry
                        ? _jsxs("a", {
                            href: app.homepage,
                            target: "_blank",
                            rel: "noopener noreferrer",
                            className:
                              "inline-flex items-center gap-1 text-xs text-primary hover:underline",
                            children: [
                              _jsx(ExternalLink, { className: "h-3 w-3" }),
                              "\u5E94\u7528\u4E3B\u9875",
                            ],
                          })
                        : null,
                      _jsxs("div", {
                        className: "flex items-center gap-2",
                        children: [
                          _jsx(Switch, {
                            checked: enabled,
                            onCheckedChange: handleToggleEnabled,
                            disabled: enablingPending,
                            "aria-label": "\u542F\u7528\u72B6\u6001",
                          }),
                          _jsx("span", {
                            className: "text-sm text-muted-foreground",
                            children: enabled ? "运行中" : "已停用",
                          }),
                        ],
                      }),
                    ],
                  }),
                  app.description
                    ? _jsx("p", {
                        className: "text-sm text-muted-foreground",
                        children: app.description,
                      })
                    : null,
                ],
              }),
            ],
          }),
        ],
      }),
      _jsx("div", {
        className: "md:hidden",
        children: _jsx("select", {
          value: section,
          onChange: (e) => setSection(e.target.value),
          className: "w-full h-9 px-3 rounded-md border bg-background text-sm",
          "aria-label": "\u9009\u62E9\u9875\u9762",
          children: navItems.map((item) =>
            _jsx("option", { value: item.key, children: item.label }, item.key),
          ),
        }),
      }),
      _jsxs("div", {
        className: "flex gap-8",
        children: [
          _jsx("nav", {
            className: "hidden md:block w-48 shrink-0 space-y-1",
            children: navItems.map((item) =>
              _jsxs(
                Button,
                {
                  variant: "ghost",
                  size: "sm",
                  onClick: () => setSection(item.key),
                  className: `w-full justify-start gap-2 ${
                    section === item.key
                      ? "bg-primary/10 text-primary font-medium hover:bg-primary/10 hover:text-primary"
                      : "text-muted-foreground"
                  }`,
                  children: [_jsx(item.icon, { className: "h-4 w-4 shrink-0" }), item.label],
                },
                item.key,
              ),
            ),
          }),
          _jsxs("div", {
            className: "flex-1 min-w-0",
            children: [
              section === "token" && _jsx(TokenSection, { app: app, inst: inst }),
              section === "tools" &&
                (() => {
                  const appTools = parseTools(app?.tools);
                  const instTools = parseTools(inst?.tools);
                  return _jsxs("div", {
                    className: "space-y-6",
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
                              "\u6B64\u5B89\u88C5\u53EF\u7528\u7684\u547D\u4EE4\u548C\u5DE5\u5177\u3002",
                          }),
                        ],
                      }),
                      instTools.length > 0
                        ? _jsxs(Card, {
                            className: "p-5 space-y-3",
                            children: [
                              _jsx("h3", {
                                className:
                                  "text-xs font-bold uppercase tracking-wider text-muted-foreground",
                                children: "\u81EA\u5B9A\u4E49\u547D\u4EE4",
                              }),
                              _jsx(ToolsDisplay, { tools: instTools }),
                            ],
                          })
                        : null,
                      appTools.length > 0
                        ? _jsxs(Card, {
                            className: "p-5 space-y-3",
                            children: [
                              instTools.length > 0
                                ? _jsx("h3", {
                                    className:
                                      "text-xs font-bold uppercase tracking-wider text-muted-foreground",
                                    children: "\u5E94\u7528\u547D\u4EE4",
                                  })
                                : null,
                              _jsx(ToolsDisplay, { tools: appTools }),
                            ],
                          })
                        : null,
                    ],
                  });
                })(),
              section === "permissions" && _jsx(PermissionsSection, { app: app, inst: inst }),
              section === "app-config" &&
                _jsx(AppConfigForm, { app: app, inst: inst, onUpdate: refreshInstallations }),
              section === "config" &&
                _jsx(ConfigSection, {
                  inst: inst,
                  onUninstall: () => navigate(`/dashboard/accounts/${botId}`),
                  queryClient: queryClient,
                }),
              section === "event-logs" &&
                _jsx(EventLogsSection, {
                  appId: inst.app_id,
                  instId: inst.id,
                  botId: botId,
                  homepage: app.homepage,
                }),
              section === "api-logs" &&
                _jsx(ApiLogsSection, { appId: inst.app_id, instId: inst.id }),
            ],
          }),
        ],
      }),
    ],
  });
}
// ==================== Inline Handle Editor ====================
function InlineHandleEditor({ value, onSave }) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(value);
  const inputRef = useRef(null);
  // Prevents Enter keydown + subsequent onBlur from both calling onSave
  const committingRef = useRef(false);
  // Prevents Escape-triggered onBlur from calling onSave
  const cancelledRef = useRef(false);
  useEffect(() => {
    // Select all text when entering edit mode (autoFocus handles focus)
    if (editing) inputRef.current?.select();
    // Reset guards when editing state settles
    if (!editing) {
      committingRef.current = false;
      cancelledRef.current = false;
    }
  }, [editing]);
  // Keep draft in sync when external value changes (e.g. after save round-trip)
  useEffect(() => {
    if (!editing) setDraft(value);
  }, [value, editing]);
  function startEdit() {
    setDraft(value);
    setEditing(true);
  }
  function commit() {
    if (committingRef.current || cancelledRef.current) return;
    // Keep editor open on empty input so user can correct it
    if (!draft.trim()) {
      return;
    }
    committingRef.current = true;
    setEditing(false);
    onSave(draft);
  }
  function cancel() {
    cancelledRef.current = true;
    setEditing(false);
    setDraft(value);
  }
  function handleKeyDown(e) {
    if (e.key === "Enter") commit();
    if (e.key === "Escape") cancel();
  }
  if (editing) {
    return _jsxs("div", {
      className: "flex items-center gap-1",
      children: [
        _jsx("span", { className: "text-sm text-muted-foreground font-mono", children: "@" }),
        _jsx("input", {
          ref: inputRef,
          value: draft,
          onChange: (e) => setDraft(e.target.value),
          onBlur: commit,
          onKeyDown: handleKeyDown,
          className:
            "h-6 min-w-[8rem] max-w-[20rem] text-sm font-mono bg-transparent border-b border-primary outline-none px-0",
          placeholder: "handle",
          "aria-label": "\u7F16\u8F91 handle",
          autoFocus: true,
        }),
      ],
    });
  }
  return _jsx("button", {
    onClick: startEdit,
    className: `text-sm font-mono transition-colors cursor-text hover:text-foreground ${value ? "text-muted-foreground" : "text-muted-foreground/50 italic"}`,
    title: "\u70B9\u51FB\u7F16\u8F91 handle",
    children: value ? `@${value}` : "添加 handle",
  });
}
// ==================== Token & Usage Section ====================
function TokenSection({ app, inst }) {
  const [showToken, setShowToken] = useState(false);
  const [copied, setCopied] = useState(false);
  const { toast } = useToast();
  const token = inst.app_token || inst.token || "";
  const hubUrl = window.location.origin;
  function handleCopy(text) {
    if (!navigator.clipboard?.writeText) {
      toast({
        variant: "destructive",
        title: "复制失败",
        description: "当前浏览器不支持自动复制，请手动选中复制",
      });
      return;
    }
    navigator.clipboard
      .writeText(text)
      .then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      })
      .catch(() => {
        toast({ variant: "destructive", title: "复制失败", description: "请手动选中复制" });
      });
  }
  const maskedToken = token ? token.slice(0, 8) + "****" + token.slice(-4) : "---";
  function renderGuide() {
    if (app.guide) {
      return app.guide
        .replace(/\{hub_url\}/g, hubUrl)
        .replace(/\{your_token\}/g, token || "<your_token>");
    }
    return null;
  }
  const guideText = renderGuide();
  const guideHtml = useMemo(() => {
    if (!guideText) return "";
    // Open links in new tab to avoid navigating away from the SPA
    DOMPurify.addHook("afterSanitizeAttributes", (node) => {
      if (node.tagName === "A") {
        node.setAttribute("target", "_blank");
        node.setAttribute("rel", "noopener noreferrer");
      }
    });
    const html = DOMPurify.sanitize(marked.parse(guideText, { async: false }));
    DOMPurify.removeHook("afterSanitizeAttributes");
    return html;
  }, [guideText]);
  const guideRef = useRef(null);
  useEffect(() => {
    const el = guideRef.current;
    if (!el) return;
    const timeouts = [];
    el.querySelectorAll("pre").forEach((pre) => {
      if (pre.querySelector(".copy-btn")) return;
      // Read code text before appending button to avoid including button label
      const codeText = pre.querySelector("code")?.textContent || pre.textContent || "";
      const btn = document.createElement("button");
      btn.type = "button";
      btn.className = "copy-btn";
      btn.textContent = "复制";
      btn.setAttribute("aria-label", "复制代码");
      Object.assign(btn.style, {
        position: "absolute",
        top: "6px",
        right: "6px",
        fontSize: "11px",
        padding: "2px 8px",
        borderRadius: "4px",
        border: "1px solid var(--border)",
        background: "var(--background)",
        color: "var(--muted-foreground)",
        cursor: "pointer",
      });
      btn.addEventListener("click", () => {
        if (!navigator.clipboard?.writeText) {
          btn.textContent = "失败";
          toast({
            variant: "destructive",
            title: "复制失败",
            description: "当前浏览器不支持自动复制，请手动选中复制",
          });
          timeouts.push(
            setTimeout(() => {
              btn.textContent = "复制";
            }, 2000),
          );
          return;
        }
        navigator.clipboard
          .writeText(codeText)
          .then(() => {
            btn.textContent = "已复制";
            timeouts.push(
              setTimeout(() => {
                btn.textContent = "复制";
              }, 2000),
            );
          })
          .catch(() => {
            btn.textContent = "失败";
            toast({ variant: "destructive", title: "复制失败", description: "请手动选中复制" });
            timeouts.push(
              setTimeout(() => {
                btn.textContent = "复制";
              }, 2000),
            );
          });
      });
      pre.appendChild(btn);
    });
    return () => {
      timeouts.forEach(clearTimeout);
    };
  }, [guideHtml]);
  const showGenericGuide = !guideText && app.registry === "builtin";
  const showUsageGuide = guideText || showGenericGuide;
  return _jsxs("div", {
    className: "space-y-6",
    children: [
      _jsxs("div", {
        children: [
          _jsx("h2", {
            className: "text-base font-semibold",
            children: "Token & \u4F7F\u7528\u65B9\u5F0F",
          }),
          _jsx("p", {
            className: "text-sm text-muted-foreground mt-1",
            children: "\u5E94\u7528\u7684 Token \u548C\u63A5\u5165\u6307\u5357\u3002",
          }),
        ],
      }),
      _jsxs(Card, {
        children: [
          _jsx(CardHeader, { children: _jsx(CardTitle, { children: "Token" }) }),
          _jsx(CardContent, {
            children: _jsxs("div", {
              className: "flex items-center gap-2 p-2 rounded-md border bg-background",
              children: [
                _jsx("code", {
                  className: "text-xs font-mono flex-1 break-all select-all",
                  children: showToken ? token : maskedToken,
                }),
                _jsx(Button, {
                  type: "button",
                  variant: "ghost",
                  size: "icon-xs",
                  onClick: () => setShowToken(!showToken),
                  "aria-label": showToken ? "隐藏" : "显示",
                  children: showToken
                    ? _jsx(EyeOff, { className: "w-3.5 h-3.5" })
                    : _jsx(Eye, { className: "w-3.5 h-3.5" }),
                }),
                _jsx(Button, {
                  type: "button",
                  variant: "ghost",
                  size: "icon-xs",
                  onClick: () => handleCopy(token),
                  "aria-label": "\u590D\u5236",
                  children: copied
                    ? _jsx(Check, { className: "w-3.5 h-3.5 text-primary" })
                    : _jsx(Copy, { className: "w-3.5 h-3.5" }),
                }),
              ],
            }),
          }),
        ],
      }),
      guideText
        ? _jsxs(Card, {
            children: [
              _jsx(CardHeader, {
                children: _jsx(CardTitle, { children: "\u4F7F\u7528\u6307\u5357" }),
              }),
              _jsx(CardContent, {
                children: _jsx("div", {
                  ref: guideRef,
                  className: [
                    "p-3 rounded-md bg-muted/30 border overflow-x-auto text-sm text-muted-foreground leading-relaxed",
                    "[&_h1]:text-lg [&_h1]:font-bold [&_h1]:mb-2",
                    "[&_h2]:text-base [&_h2]:font-semibold [&_h2]:mb-2",
                    "[&_h3]:text-sm [&_h3]:font-semibold [&_h3]:mb-1",
                    "[&_p]:mb-2 [&_ul]:list-disc [&_ul]:pl-5 [&_ul]:mb-2 [&_ol]:list-decimal [&_ol]:pl-5 [&_ol]:mb-2 [&_li]:mb-1",
                    "[&_code]:bg-muted [&_code]:px-1 [&_code]:py-0.5 [&_code]:rounded [&_code]:text-xs [&_code]:font-mono",
                    "[&_pre]:bg-muted [&_pre]:p-3 [&_pre]:rounded-md [&_pre]:overflow-x-auto [&_pre]:mb-2 [&_pre]:relative [&_pre_code]:p-0 [&_pre_code]:bg-transparent",
                    "[&_a]:text-primary [&_a]:underline [&_a:hover]:text-primary/80",
                    "[&_table]:w-full [&_table]:border-collapse [&_table]:mb-2",
                    "[&_th]:border [&_th]:border-border [&_th]:px-2 [&_th]:py-1 [&_th]:text-left [&_th]:bg-muted",
                    "[&_td]:border [&_td]:border-border [&_td]:px-2 [&_td]:py-1",
                    "[&_blockquote]:border-l-2 [&_blockquote]:border-border [&_blockquote]:pl-3 [&_blockquote]:italic [&_blockquote]:mb-2",
                    "[&_hr]:border-border [&_hr]:my-3",
                  ].join(" "),
                  dangerouslySetInnerHTML: { __html: guideHtml },
                }),
              }),
            ],
          })
        : null,
      showGenericGuide
        ? _jsxs(Card, {
            children: [
              _jsx(CardHeader, {
                children: _jsx(CardTitle, { children: "\u63A5\u5165\u65B9\u5F0F" }),
              }),
              _jsxs(CardContent, {
                className: "space-y-3",
                children: [
                  _jsxs("details", {
                    className: "group",
                    children: [
                      _jsxs("summary", {
                        className:
                          "text-sm font-medium cursor-pointer flex items-center gap-2 select-none",
                        children: [
                          _jsx(ArrowRight, {
                            className: "h-3.5 w-3.5 transition-transform group-open:rotate-90",
                          }),
                          "WebSocket \u8FDE\u63A5",
                        ],
                      }),
                      _jsx("pre", {
                        className:
                          "mt-2 p-3 rounded-md bg-muted/30 border text-xs font-mono overflow-x-auto whitespace-pre-wrap",
                        children: `${window.location.protocol === "https:" ? "wss" : "ws"}://${window.location.host}/bot/v1/ws?token=${token || "<your_token>"}`,
                      }),
                    ],
                  }),
                  _jsxs("details", {
                    className: "group",
                    children: [
                      _jsxs("summary", {
                        className:
                          "text-sm font-medium cursor-pointer flex items-center gap-2 select-none",
                        children: [
                          _jsx(ArrowRight, {
                            className: "h-3.5 w-3.5 transition-transform group-open:rotate-90",
                          }),
                          "HTTP \u53D1\u6D88\u606F",
                        ],
                      }),
                      _jsx("pre", {
                        className:
                          "mt-2 p-3 rounded-md bg-muted/30 border text-xs font-mono overflow-x-auto whitespace-pre-wrap",
                        children: `curl -X POST ${hubUrl}/bot/v1/message/send \\\n  -H "Authorization: Bearer ${token || "<your_token>"}" \\\n  -H "Content-Type: application/json" \\\n  -d '{"content":"hello"}'`,
                      }),
                    ],
                  }),
                ],
              }),
            ],
          })
        : null,
      !showUsageGuide && app.webhook_url
        ? _jsx(Card, {
            children: _jsx(CardContent, {
              children: _jsxs("p", {
                className: "text-xs text-muted-foreground",
                children: [
                  "\u4E8B\u4EF6\u5C06\u63A8\u9001\u5230 ",
                  _jsx("code", { className: "font-mono", children: app.webhook_url }),
                ],
              }),
            }),
          })
        : null,
    ],
  });
}
// ==================== App Config Form (config_schema) ====================
function AppConfigForm({ app, inst, onUpdate }) {
  const { toast } = useToast();
  const [saving, setSaving] = useState(false);
  let parsed = {};
  try {
    parsed =
      typeof app.config_schema === "string"
        ? JSON.parse(app.config_schema || "{}")
        : app.config_schema || {};
  } catch {
    return null;
  }
  const properties = parsed.properties || {};
  if (Object.keys(properties).length === 0) return null;
  let currentConfig = {};
  try {
    currentConfig =
      typeof inst.config === "string" ? JSON.parse(inst.config || "{}") : inst.config || {};
  } catch {}
  const [form, setForm] = useState(currentConfig);
  async function handleSave() {
    setSaving(true);
    try {
      await api.updateInstallation(inst.app_id, inst.id, {
        config: form,
      });
      toast({ title: "配置已保存" });
      onUpdate();
    } catch (e) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
    setSaving(false);
  }
  return _jsxs("div", {
    className: "space-y-6",
    children: [
      _jsxs("div", {
        children: [
          _jsx("h2", {
            className: "text-base font-semibold",
            children: "\u5E94\u7528\u914D\u7F6E",
          }),
          _jsx("p", {
            className: "text-sm text-muted-foreground mt-1",
            children: "\u914D\u7F6E\u6B64\u5E94\u7528\u7684\u8FD0\u884C\u53C2\u6570\u3002",
          }),
        ],
      }),
      _jsx(Card, {
        children: _jsxs(CardContent, {
          className: "space-y-4 pt-6",
          children: [
            Object.entries(properties).map(([key, prop]) =>
              _jsxs(
                "div",
                {
                  className: "space-y-1.5",
                  children: [
                    _jsx(Label, {
                      className: "text-muted-foreground",
                      children: prop.title || key,
                    }),
                    _jsx(Input, {
                      value: form[key] || "",
                      onChange: (e) => setForm({ ...form, [key]: e.target.value }),
                      className: "h-8 text-xs font-mono",
                      placeholder: prop.description || "",
                    }),
                    prop.description
                      ? _jsx("p", {
                          className: "text-xs text-muted-foreground",
                          children: prop.description,
                        })
                      : null,
                  ],
                },
                key,
              ),
            ),
            _jsx("div", {
              className: "flex items-center gap-2 pt-2 border-t",
              children: _jsxs(Button, {
                size: "sm",
                onClick: handleSave,
                disabled: saving,
                children: [
                  saving && _jsx(Loader2, { className: "h-3 w-3 animate-spin mr-1" }),
                  "\u4FDD\u5B58\u914D\u7F6E",
                ],
              }),
            }),
          ],
        }),
      }),
    ],
  });
}
// ==================== Config Section ====================
function ConfigSection({ inst, onUninstall, queryClient }) {
  const { toast } = useToast();
  const [showUninstallDialog, setShowUninstallDialog] = useState(false);
  const [uninstalling, setUninstalling] = useState(false);
  async function handleUninstall() {
    setUninstalling(true);
    try {
      await api.deleteInstallation(inst.app_id, inst.id);
      toast({ title: "已卸载" });
      invalidateAllAppQueries(queryClient, inst.bot_id);
      onUninstall();
    } catch (e) {
      toast({ variant: "destructive", title: "卸载失败", description: e.message });
    } finally {
      setUninstalling(false);
    }
  }
  return _jsxs("div", {
    className: "space-y-6",
    children: [
      _jsxs("div", {
        children: [
          _jsx("h2", {
            className: "text-base font-semibold",
            children: "\u5371\u9669\u64CD\u4F5C",
          }),
          _jsx("p", {
            className: "text-sm text-muted-foreground mt-1",
            children:
              "\u4EE5\u4E0B\u64CD\u4F5C\u4E0D\u53EF\u64A4\u9500\uFF0C\u8BF7\u8C28\u614E\u64CD\u4F5C\u3002",
          }),
        ],
      }),
      _jsxs(Card, {
        children: [
          _jsxs(CardHeader, {
            children: [
              _jsx(CardTitle, { children: "\u5378\u8F7D\u5E94\u7528" }),
              _jsx(CardDescription, {
                children:
                  "\u5378\u8F7D\u540E\u5C06\u5220\u9664\u6B64\u5B89\u88C5\u5B9E\u4F8B\uFF0CToken \u5C06\u5931\u6548\uFF0C\u6B64\u64CD\u4F5C\u4E0D\u53EF\u64A4\u9500\u3002",
              }),
            ],
          }),
          _jsx(CardContent, {
            children: _jsxs(Button, {
              variant: "destructive",
              size: "sm",
              onClick: () => setShowUninstallDialog(true),
              children: [
                _jsx(Trash2, { className: "h-3.5 w-3.5 mr-1" }),
                "\u5378\u8F7D\u5E94\u7528",
              ],
            }),
          }),
        ],
      }),
      _jsx(Dialog, {
        open: showUninstallDialog,
        onOpenChange: setShowUninstallDialog,
        children: _jsxs(DialogContent, {
          className: "sm:max-w-md",
          children: [
            _jsxs(DialogHeader, {
              children: [
                _jsx(DialogTitle, { children: "\u786E\u8BA4\u5378\u8F7D" }),
                _jsx(DialogDescription, {
                  children:
                    "\u5378\u8F7D\u540E\u5C06\u5220\u9664\u6B64\u5B89\u88C5\u5B9E\u4F8B\uFF0CToken \u5C06\u5931\u6548\uFF0C\u6B64\u64CD\u4F5C\u4E0D\u53EF\u64A4\u9500\u3002",
                }),
              ],
            }),
            _jsxs("div", {
              className: "flex justify-end gap-2 pt-4",
              children: [
                _jsx(Button, {
                  variant: "ghost",
                  onClick: () => setShowUninstallDialog(false),
                  children: "\u53D6\u6D88",
                }),
                _jsxs(Button, {
                  variant: "destructive",
                  onClick: handleUninstall,
                  disabled: uninstalling,
                  children: [
                    uninstalling && _jsx(Loader2, { className: "h-3 w-3 animate-spin mr-1" }),
                    "\u786E\u8BA4\u5378\u8F7D",
                  ],
                }),
              ],
            }),
          ],
        }),
      }),
    ],
  });
}
// ==================== Event Logs Section ====================
function EventLogsSection({ appId, instId, botId, homepage }) {
  const navigate = useNavigate();
  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(true);
  const loadLogs = useCallback(async () => {
    try {
      const data = (await api.listEventLogs(appId, instId, 50)) || [];
      setLogs(data);
    } catch {
      setLogs([]);
    } finally {
      setLoading(false);
    }
  }, [appId, instId]);
  useEffect(() => {
    let timer;
    let cancelled = false;
    async function poll() {
      await loadLogs();
      if (!cancelled) timer = setTimeout(poll, 10000);
    }
    poll();
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [loadLogs]);
  return _jsxs("div", {
    className: "space-y-6",
    children: [
      _jsxs("div", {
        className: "flex items-center justify-between",
        children: [
          _jsxs("div", {
            children: [
              _jsx("h2", {
                className: "text-base font-semibold",
                children: "\u4E8B\u4EF6\u6295\u9012\u65E5\u5FD7",
              }),
              _jsx("p", {
                className: "text-sm text-muted-foreground mt-1",
                children:
                  "Hub \u63A8\u9001\u5230\u6B64\u5E94\u7528\u7684\u4E8B\u4EF6\u8BB0\u5F55\u3002",
              }),
            ],
          }),
          _jsxs(Button, {
            variant: "ghost",
            size: "sm",
            className: "h-7 text-xs gap-1",
            onClick: () => {
              setLoading(true);
              loadLogs();
            },
            children: [_jsx(RefreshCw, { className: "h-3 w-3" }), "\u5237\u65B0"],
          }),
        ],
      }),
      !loading &&
        logs.some((l) => {
          const code = l.status_code || l.status;
          return code >= 400 && code < 500;
        }) &&
        _jsxs("div", {
          className:
            "rounded-lg border border-orange-200 bg-orange-50 dark:border-orange-900 dark:bg-orange-950/30 p-3 text-xs text-muted-foreground space-y-1",
          children: [
            _jsx("p", {
              className: "font-medium text-orange-700 dark:text-orange-400",
              children: "\u90E8\u5206\u4E8B\u4EF6\u6295\u9012\u5931\u8D25",
            }),
            _jsx("p", {
              children:
                "\u5982\u679C\u5E94\u7528\u6765\u81EA\u8FDC\u7A0B\u5E02\u573A\uFF0C4xx \u9519\u8BEF\u901A\u5E38\u662F\u8FDC\u7A0B\u5E94\u7528\u670D\u52A1\u5668\u7684\u914D\u7F6E\u95EE\u9898\u3002\u8BF7\u8054\u7CFB\u5E94\u7528\u5F00\u53D1\u8005\u786E\u8BA4 Webhook \u5730\u5740\u548C\u6743\u9650\u914D\u7F6E\u662F\u5426\u6B63\u786E\u3002",
            }),
            homepage
              ? _jsxs("a", {
                  href: homepage,
                  target: "_blank",
                  rel: "noopener noreferrer",
                  className:
                    "inline-flex items-center gap-1 text-orange-700 hover:underline dark:text-orange-400",
                  children: [
                    _jsx(ExternalLink, { className: "h-3 w-3" }),
                    "\u524D\u5F80\u5E94\u7528\u4E3B\u9875",
                  ],
                })
              : null,
          ],
        }),
      _jsx(Card, {
        className: "overflow-hidden",
        children: loading
          ? _jsx("div", {
              className: "p-6 space-y-2",
              children: [1, 2, 3].map((i) => _jsx(Skeleton, { className: "h-8 w-full" }, i)),
            })
          : logs.length === 0
            ? _jsx("p", {
                className: "text-sm text-muted-foreground text-center py-8 px-4",
                children: "\u6682\u65E0\u4E8B\u4EF6\u65E5\u5FD7",
              })
            : _jsxs(Table, {
                children: [
                  _jsx(TableHeader, {
                    children: _jsxs(TableRow, {
                      children: [
                        _jsx(TableHead, { children: "\u65F6\u95F4" }),
                        _jsx(TableHead, { children: "\u4E8B\u4EF6\u7C7B\u578B" }),
                        _jsx(TableHead, { children: "Trace ID" }),
                        _jsx(TableHead, { children: "\u72B6\u6001\u7801" }),
                        _jsx(TableHead, { children: "\u8017\u65F6" }),
                        _jsx(TableHead, { children: "\u9519\u8BEF" }),
                        _jsx(TableHead, { className: "w-8" }),
                      ],
                    }),
                  }),
                  _jsx(TableBody, {
                    children: logs.map((log) =>
                      _jsxs(
                        TableRow,
                        {
                          className: log.trace_id ? "cursor-pointer focus-visible:bg-muted/50" : "",
                          tabIndex: log.trace_id ? 0 : undefined,
                          onClick: () =>
                            log.trace_id &&
                            navigate(`/dashboard/accounts/${botId}/traces/${log.trace_id}`),
                          onKeyDown: (e) => {
                            if (log.trace_id && (e.key === "Enter" || e.key === " ")) {
                              e.preventDefault();
                              navigate(`/dashboard/accounts/${botId}/traces/${log.trace_id}`);
                            }
                          },
                          children: [
                            _jsx(TableCell, {
                              className: "font-mono whitespace-nowrap",
                              children: formatTime(log.created_at),
                            }),
                            _jsx(TableCell, {
                              children: _jsx(Badge, {
                                variant: "outline",
                                className: "font-mono",
                                children: log.event_type,
                              }),
                            }),
                            _jsx(TableCell, {
                              className: "font-mono text-muted-foreground",
                              children: log.trace_id ? log.trace_id.slice(0, 12) + "…" : "-",
                            }),
                            _jsx(TableCell, {
                              children: _jsx(StatusBadge, {
                                status: log.status_code || log.status,
                              }),
                            }),
                            _jsx(TableCell, {
                              className: "font-mono",
                              children: log.duration_ms != null ? `${log.duration_ms}ms` : "-",
                            }),
                            _jsx(TableCell, {
                              className: "text-destructive max-w-48 truncate",
                              children: log.error || "-",
                            }),
                            _jsx(TableCell, {
                              className: "w-8 px-2",
                              children:
                                log.trace_id &&
                                _jsx(ChevronRight, { className: "h-4 w-4 text-muted-foreground" }),
                            }),
                          ],
                        },
                        log.id || log.trace_id + log.created_at,
                      ),
                    ),
                  }),
                ],
              }),
      }),
    ],
  });
}
// ==================== API Logs Section ====================
function ApiLogsSection({ appId, instId }) {
  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(true);
  const loadLogs = useCallback(async () => {
    try {
      const data = (await api.listApiLogs(appId, instId, 50)) || [];
      setLogs(data);
    } catch {
      setLogs([]);
    } finally {
      setLoading(false);
    }
  }, [appId, instId]);
  useEffect(() => {
    let timer;
    let cancelled = false;
    async function poll() {
      await loadLogs();
      if (!cancelled) timer = setTimeout(poll, 10000);
    }
    poll();
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [loadLogs]);
  return _jsxs("div", {
    className: "space-y-6",
    children: [
      _jsxs("div", {
        className: "flex items-center justify-between",
        children: [
          _jsxs("div", {
            children: [
              _jsx("h2", {
                className: "text-base font-semibold",
                children: "API \u8C03\u7528\u65E5\u5FD7",
              }),
              _jsx("p", {
                className: "text-sm text-muted-foreground mt-1",
                children:
                  "\u6B64\u5E94\u7528\u901A\u8FC7 Bot API \u53D1\u8D77\u7684\u8C03\u7528\u8BB0\u5F55\u3002",
              }),
            ],
          }),
          _jsxs(Button, {
            variant: "ghost",
            size: "sm",
            className: "h-7 text-xs gap-1",
            onClick: () => {
              setLoading(true);
              loadLogs();
            },
            children: [_jsx(RefreshCw, { className: "h-3 w-3" }), "\u5237\u65B0"],
          }),
        ],
      }),
      _jsx(Card, {
        className: "overflow-hidden",
        children: loading
          ? _jsx("div", {
              className: "p-6 space-y-2",
              children: [1, 2, 3].map((i) => _jsx(Skeleton, { className: "h-8 w-full" }, i)),
            })
          : logs.length === 0
            ? _jsx("p", {
                className: "text-sm text-muted-foreground text-center py-8 px-4",
                children: "\u6682\u65E0 API \u65E5\u5FD7",
              })
            : _jsxs(Table, {
                children: [
                  _jsx(TableHeader, {
                    children: _jsxs(TableRow, {
                      children: [
                        _jsx(TableHead, { children: "\u65F6\u95F4" }),
                        _jsx(TableHead, { children: "\u65B9\u6CD5" }),
                        _jsx(TableHead, { children: "\u8DEF\u5F84" }),
                        _jsx(TableHead, { children: "\u72B6\u6001\u7801" }),
                        _jsx(TableHead, { children: "\u8017\u65F6" }),
                      ],
                    }),
                  }),
                  _jsx(TableBody, {
                    children: logs.map((log, idx) =>
                      _jsxs(
                        TableRow,
                        {
                          children: [
                            _jsx(TableCell, {
                              className: "font-mono whitespace-nowrap",
                              children: formatTime(log.created_at),
                            }),
                            _jsx(TableCell, {
                              children: _jsx(Badge, {
                                variant: "outline",
                                className: "font-mono font-bold",
                                children: log.method,
                              }),
                            }),
                            _jsx(TableCell, {
                              className: "font-mono text-muted-foreground max-w-64 truncate",
                              children: log.path,
                            }),
                            _jsx(TableCell, {
                              children: _jsx(StatusBadge, {
                                status: log.status_code || log.status,
                              }),
                            }),
                            _jsx(TableCell, {
                              className: "font-mono",
                              children: log.duration_ms != null ? `${log.duration_ms}ms` : "-",
                            }),
                          ],
                        },
                        log.id || idx,
                      ),
                    ),
                  }),
                ],
              }),
      }),
    ],
  });
}
// ==================== Permissions Section ====================
function PermissionsSection({ app, inst }) {
  const scopes = inst?.scopes || [];
  const events = app?.events || [];
  const readScopes = scopes.filter((s) => s.endsWith(":read"));
  const writeScopes = scopes.filter((s) => s.endsWith(":write"));
  const otherScopes = scopes.filter((s) => !s.endsWith(":read") && !s.endsWith(":write"));
  function eventLabelEntry(key) {
    const found = EVENT_TYPES.find((e) => e.key === key);
    return found ? { label: found.label, showKey: true } : { label: key, showKey: false };
  }
  return _jsxs("div", {
    className: "space-y-6",
    children: [
      _jsxs("div", {
        children: [
          _jsx("h2", { className: "text-base font-semibold", children: "\u6743\u9650" }),
          _jsx("p", {
            className: "text-sm text-muted-foreground mt-1",
            children: "\u6B64\u5E94\u7528\u6240\u7533\u8BF7\u7684\u6743\u9650\u8303\u56F4\u3002",
          }),
        ],
      }),
      _jsx(Card, {
        children: _jsxs(CardContent, {
          className: "pt-6 space-y-4",
          children: [
            readScopes.length > 0 &&
              _jsxs("div", {
                className: "space-y-2",
                children: [
                  _jsx("p", {
                    className: "text-xs font-bold uppercase tracking-wider text-muted-foreground",
                    children: "\u8BFB\u53D6\u6743\u9650",
                  }),
                  _jsx("div", {
                    className: "space-y-1.5",
                    children: readScopes.map((scope) =>
                      _jsxs(
                        "div",
                        {
                          className: "flex items-center gap-2 text-sm text-muted-foreground",
                          children: [
                            _jsx(Eye, { className: "h-3.5 w-3.5 shrink-0" }),
                            _jsx("span", { children: SCOPE_DESCRIPTIONS[scope] || scope }),
                            _jsx("span", {
                              className: "font-mono text-xs ml-auto text-muted-foreground/60",
                              children: scope,
                            }),
                          ],
                        },
                        scope,
                      ),
                    ),
                  }),
                ],
              }),
            writeScopes.length > 0 &&
              _jsxs("div", {
                className: "space-y-2",
                children: [
                  _jsx("p", {
                    className: "text-xs font-bold uppercase tracking-wider text-muted-foreground",
                    children: "\u5199\u5165\u6743\u9650",
                  }),
                  _jsx("div", {
                    className: "space-y-1.5",
                    children: writeScopes.map((scope) =>
                      _jsxs(
                        "div",
                        {
                          className: "flex items-center gap-2 text-sm text-muted-foreground",
                          children: [
                            _jsx(Zap, { className: "h-3.5 w-3.5 shrink-0" }),
                            _jsx("span", { children: SCOPE_DESCRIPTIONS[scope] || scope }),
                            _jsx("span", {
                              className: "font-mono text-xs ml-auto text-muted-foreground/60",
                              children: scope,
                            }),
                          ],
                        },
                        scope,
                      ),
                    ),
                  }),
                ],
              }),
            otherScopes.length > 0 &&
              _jsxs("div", {
                className: "space-y-2",
                children: [
                  _jsx("p", {
                    className: "text-xs font-bold uppercase tracking-wider text-muted-foreground",
                    children: "\u5176\u4ED6\u6743\u9650",
                  }),
                  _jsx("div", {
                    className: "space-y-1.5",
                    children: otherScopes.map((scope) =>
                      _jsxs(
                        "div",
                        {
                          className: "flex items-center gap-2 text-sm text-muted-foreground",
                          children: [
                            _jsx(ShieldCheck, { className: "h-3.5 w-3.5 shrink-0" }),
                            _jsx("span", { children: SCOPE_DESCRIPTIONS[scope] || scope }),
                            _jsx("span", {
                              className: "font-mono text-xs ml-auto text-muted-foreground/60",
                              children: scope,
                            }),
                          ],
                        },
                        scope,
                      ),
                    ),
                  }),
                ],
              }),
            events.length > 0 &&
              _jsxs("div", {
                className: "space-y-2",
                children: [
                  _jsx("p", {
                    className: "text-xs font-bold uppercase tracking-wider text-muted-foreground",
                    children: "\u8BA2\u9605\u4E8B\u4EF6",
                  }),
                  _jsx("div", {
                    className: "flex flex-wrap gap-1.5",
                    children: events.map((event) => {
                      const { label, showKey } = eventLabelEntry(event);
                      return _jsxs(
                        Badge,
                        {
                          variant: "outline",
                          className: "text-xs",
                          children: [
                            label,
                            showKey &&
                              _jsxs("span", {
                                className: "font-mono text-muted-foreground/60 ml-1",
                                children: ["\u00B7 ", event],
                              }),
                          ],
                        },
                        event,
                      );
                    }),
                  }),
                ],
              }),
          ],
        }),
      }),
    ],
  });
}
// ==================== Helpers ====================
function StatusBadge({ status }) {
  if (status == null)
    return _jsx("span", { className: "text-xs text-muted-foreground", children: "-" });
  const n = typeof status === "string" ? parseInt(status, 10) : status;
  if (isNaN(n))
    return _jsx("span", { className: "text-xs text-muted-foreground", children: status });
  const variant = n >= 200 && n < 300 ? "default" : n >= 400 ? "destructive" : "outline";
  return _jsx(Badge, { variant: variant, className: "font-mono", children: n });
}
function formatTime(ts) {
  if (!ts) return "-";
  try {
    const d = new Date(ts * 1000);
    return d.toLocaleString("zh-CN", {
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  } catch {
    return "-";
  }
}
