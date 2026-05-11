import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect, useState } from "react";
import { useParams, useNavigate, useSearchParams } from "react-router-dom";
import {
  ArrowLeft,
  Eye,
  Zap,
  Loader2,
  ExternalLink,
  ShieldCheck,
  Terminal,
  Sliders,
} from "lucide-react";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Card, CardContent } from "../components/ui/card";
import { Skeleton } from "../components/ui/skeleton";
import { Label } from "../components/ui/label";
import { useQueryClient } from "@tanstack/react-query";
import { api, botDisplayName } from "../lib/api";
import { invalidateAllAppQueries, useApp } from "@/hooks/use-apps";
import { useBots } from "@/hooks/use-bots";
import { useToast } from "@/hooks/use-toast";
import { AppIcon } from "../components/app-icon";
import { SCOPE_DESCRIPTIONS, EVENT_TYPES } from "../lib/constants";
import { ToolsDisplay, parseTools } from "../components/tools-display";
export function InstallAppPage() {
  const { id: botId, appId } = useParams();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { toast } = useToast();
  const qc = useQueryClient();
  const invalidateAppQueries = () => invalidateAllAppQueries(qc, botId);
  const { data: app, isLoading: appLoading } = useApp(appId);
  const { data: allBots = [] } = useBots();
  const bot = allBots.find((b) => b.id === botId);
  const botName = bot ? botDisplayName(bot) : "";
  const loading = appLoading;
  const [handle, setHandle] = useState(searchParams.get("handle") || "");
  const [configForm, setConfigForm] = useState(() => {
    const prefill = {};
    searchParams.forEach((value, key) => {
      if (key.startsWith("config.")) {
        prefill[key.slice(7)] = value;
      }
    });
    return prefill;
  });
  const [installing, setInstalling] = useState(false);
  const [waitingForOAuth, setWaitingForOAuth] = useState(false);
  const [oauthPopup, setOAuthPopup] = useState(null);
  const [tab, setTab] = useState("permissions");
  // Save config for an installation if the app has a config_schema and the user filled values.
  async function saveConfigIfNeeded(installationId) {
    if (!app?.config_schema || !installationId) return;
    const hasConfig = Object.values(configForm).some((v) => v !== "");
    if (!hasConfig) return;
    try {
      await api.updateInstallation(appId, installationId, { config: configForm });
    } catch {
      toast({ title: "配置保存失败", description: "应用已安装，但配置未保存。" });
    }
  }
  // Listen for OAuth completion from popup
  useEffect(() => {
    if (!waitingForOAuth) return;
    const handleMessage = async (event) => {
      if (event.data?.type === "oauth_complete") {
        setWaitingForOAuth(false);
        if (oauthPopup && !oauthPopup.closed) oauthPopup.close();
        // Look up the installation to save config (OAuth flow creates it server-side).
        try {
          const installations = await api.listBotApps(botId);
          const found = installations?.find((i) => i.app_id === appId);
          if (found) await saveConfigIfNeeded(found.id);
        } catch {}
        invalidateAppQueries();
        toast({ title: "安装成功" });
        navigate(`/dashboard/accounts/${botId}`);
      }
    };
    window.addEventListener("message", handleMessage);
    const interval = setInterval(async () => {
      try {
        const installations = await api.listBotApps(botId);
        const found = installations?.find((i) => i.app_id === appId);
        if (found) {
          clearInterval(interval);
          setWaitingForOAuth(false);
          if (oauthPopup && !oauthPopup.closed) oauthPopup.close();
          await saveConfigIfNeeded(found.id);
          invalidateAppQueries();
          toast({ title: "安装成功" });
          navigate(`/dashboard/accounts/${botId}`);
        }
      } catch {}
    }, 3000);
    return () => {
      window.removeEventListener("message", handleMessage);
      clearInterval(interval);
    };
  }, [waitingForOAuth, oauthPopup, botId, appId, navigate, toast]);
  async function handleInstall() {
    setInstalling(true);
    try {
      const result = await api.installApp(appId, {
        bot_id: botId,
        handle: handle.trim(),
        scopes: app.scopes || [],
      });
      if (result?.needs_oauth && result?.oauth_redirect) {
        setWaitingForOAuth(true);
        const popup = window.open(
          result.oauth_redirect,
          "oauth_popup",
          "width=600,height=700,scrollbars=yes",
        );
        setOAuthPopup(popup);
        setInstalling(false);
        return;
      }
      const installationId = result?.id || result?.installation_id;
      if (installationId) await saveConfigIfNeeded(installationId);
      toast({ title: "安装成功" });
      invalidateAppQueries();
      navigate(`/dashboard/accounts/${botId}/apps/${installationId}`);
    } catch (e) {
      const description =
        e.message === "HTTP 403" && app?.homepage
          ? `远端应用返回了 403。请先检查该应用的接入配置，或前往应用主页联系开发者：${app.homepage}`
          : e.message;
      toast({ variant: "destructive", title: "安装失败", description });
    } finally {
      setInstalling(false);
    }
  }
  if (waitingForOAuth) {
    return _jsxs("div", {
      className: "max-w-xl mx-auto py-20 text-center space-y-4",
      children: [
        _jsx(Loader2, { className: "h-8 w-8 animate-spin mx-auto text-primary" }),
        _jsx("h2", {
          className: "text-lg font-bold",
          children: "\u7B49\u5F85\u6388\u6743\u5B8C\u6210",
        }),
        _jsx("p", {
          className: "text-sm text-muted-foreground",
          children:
            "\u8BF7\u5728\u5F39\u51FA\u7A97\u53E3\u4E2D\u5B8C\u6210\u5E94\u7528\u6388\u6743\u3002\u5B8C\u6210\u540E\u6B64\u9875\u9762\u5C06\u81EA\u52A8\u66F4\u65B0\u3002",
        }),
        _jsx(Button, {
          variant: "outline",
          size: "sm",
          onClick: () => {
            setWaitingForOAuth(false);
            if (oauthPopup && !oauthPopup.closed) oauthPopup.close();
          },
          children: "\u53D6\u6D88",
        }),
      ],
    });
  }
  if (loading) {
    return _jsxs("div", {
      className: "space-y-6",
      children: [
        _jsx(Skeleton, { className: "h-20 w-full rounded-3xl" }),
        _jsx(Skeleton, { className: "h-64 w-full rounded-3xl" }),
      ],
    });
  }
  if (!app) {
    return _jsxs("div", {
      className: "py-20 text-center space-y-4",
      children: [
        _jsx("p", { className: "font-bold", children: "\u672A\u627E\u5230\u5E94\u7528" }),
        _jsx(Button, {
          variant: "link",
          onClick: () => navigate(`/dashboard/accounts/${botId}`),
          children: "\u8FD4\u56DE\u8D26\u53F7",
        }),
      ],
    });
  }
  // Parse scopes
  const scopes = app.scopes || [];
  const readScopes = scopes.filter((s) => s.endsWith(":read"));
  const writeScopes = scopes.filter((s) => s.endsWith(":write"));
  const otherScopes = scopes.filter((s) => !s.endsWith(":read") && !s.endsWith(":write"));
  const events = app.events || [];
  const hasPermissions =
    readScopes.length > 0 || writeScopes.length > 0 || otherScopes.length > 0 || events.length > 0;
  // Parse config_schema
  let schemaProperties = {};
  if (app.config_schema) {
    try {
      const parsed =
        typeof app.config_schema === "string" ? JSON.parse(app.config_schema) : app.config_schema;
      schemaProperties = parsed.properties || {};
    } catch {
      // ignore
    }
  }
  const tools = parseTools(app.tools);
  // Build tabs — permissions only shown when app has scopes or events
  const tabs = [];
  if (hasPermissions) {
    tabs.push({ key: "permissions", label: "权限", icon: ShieldCheck });
  }
  if (tools.length > 0) {
    tabs.push({ key: "tools", label: "命令 / 工具", icon: Terminal });
  }
  if (Object.keys(schemaProperties).length > 0) {
    tabs.push({ key: "config", label: "配置", icon: Sliders });
  }
  const activeTab = tabs.find((t) => t.key === tab) ? tab : tabs[0]?.key;
  const hasTabs = tabs.length > 0;
  return _jsxs("div", {
    className: "space-y-6",
    children: [
      _jsxs("button", {
        onClick: () => navigate(`/dashboard/accounts/${botId}`),
        className:
          "flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors cursor-pointer",
        children: [_jsx(ArrowLeft, { className: "h-4 w-4" }), botName || "返回"],
      }),
      _jsxs("div", {
        className: "flex items-start gap-4",
        children: [
          _jsx(AppIcon, { icon: app.icon, iconUrl: app.icon_url, size: "h-14 w-14" }),
          _jsxs("div", {
            className: "flex-1 min-w-0 space-y-1",
            children: [
              _jsxs("div", {
                className: "flex items-center gap-3 flex-wrap",
                children: [
                  _jsx("h1", {
                    className: "text-2xl font-bold tracking-tight",
                    children: app.name,
                  }),
                  app.slug &&
                    _jsx("span", {
                      className: "text-sm text-muted-foreground font-mono",
                      children: app.slug,
                    }),
                  app.homepage &&
                    _jsxs("a", {
                      href: app.homepage,
                      target: "_blank",
                      rel: "noopener noreferrer",
                      className:
                        "inline-flex items-center gap-1.5 text-sm text-primary hover:underline",
                      children: [_jsx(ExternalLink, { className: "h-3.5 w-3.5" }), "\u4E3B\u9875"],
                    }),
                ],
              }),
              app.description &&
                _jsx("p", {
                  className: "text-sm text-muted-foreground",
                  children: app.description,
                }),
              _jsxs("div", {
                className: "flex items-center gap-3 pt-2",
                children: [
                  _jsxs("div", {
                    className: "flex items-center gap-2",
                    children: [
                      _jsx(Input, {
                        id: "install-handle",
                        value: handle,
                        onChange: (e) => setHandle(e.target.value),
                        className: "h-8 text-xs font-mono w-40",
                        placeholder: "\u5982 notify-prod",
                      }),
                      _jsxs("span", {
                        className: "text-xs text-muted-foreground font-mono shrink-0",
                        children: ["@", handle || "handle"],
                      }),
                    ],
                  }),
                  _jsxs(Button, {
                    size: "sm",
                    onClick: handleInstall,
                    disabled: installing,
                    children: [
                      installing && _jsx(Loader2, { className: "h-4 w-4 animate-spin mr-2" }),
                      "\u5141\u8BB8\u5E76\u5B89\u88C5",
                    ],
                  }),
                ],
              }),
            ],
          }),
        ],
      }),
      hasTabs &&
        tabs.length > 1 &&
        _jsx("div", {
          className: "md:hidden",
          children: _jsx("select", {
            value: activeTab,
            onChange: (e) => setTab(e.target.value),
            className: "w-full h-9 px-3 rounded-md border bg-background text-sm",
            children: tabs.map((t) => _jsx("option", { value: t.key, children: t.label }, t.key)),
          }),
        }),
      hasTabs &&
        _jsxs("div", {
          className: "flex gap-8",
          children: [
            tabs.length > 1 &&
              _jsx("nav", {
                className: "hidden md:block w-48 shrink-0 space-y-1",
                children: tabs.map((t) =>
                  _jsxs(
                    Button,
                    {
                      variant: "ghost",
                      size: "sm",
                      onClick: () => setTab(t.key),
                      className: `w-full justify-start gap-2 ${
                        activeTab === t.key
                          ? "bg-primary/10 text-primary font-medium hover:bg-primary/10 hover:text-primary"
                          : "text-muted-foreground"
                      }`,
                      children: [_jsx(t.icon, { className: "h-4 w-4 shrink-0" }), t.label],
                    },
                    t.key,
                  ),
                ),
              }),
            _jsxs("div", {
              className: "flex-1 min-w-0",
              children: [
                activeTab === "permissions" &&
                  _jsxs("div", {
                    className: "space-y-6",
                    children: [
                      _jsxs("div", {
                        children: [
                          _jsx("h2", {
                            className: "text-base font-semibold",
                            children: "\u6743\u9650",
                          }),
                          _jsx("p", {
                            className: "text-sm text-muted-foreground mt-1",
                            children:
                              "\u5B89\u88C5\u540E\u6B64\u5E94\u7528\u5C06\u83B7\u5F97\u4EE5\u4E0B\u6743\u9650\u3002",
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
                                    className:
                                      "text-xs font-bold uppercase tracking-wider text-muted-foreground",
                                    children: "\u8BFB\u53D6\u6743\u9650",
                                  }),
                                  _jsx("div", {
                                    className: "space-y-1.5",
                                    children: readScopes.map((scope) =>
                                      _jsxs(
                                        "div",
                                        {
                                          className:
                                            "flex items-center gap-2 text-sm text-muted-foreground",
                                          children: [
                                            _jsx(Eye, { className: "h-3.5 w-3.5 shrink-0" }),
                                            _jsx("span", {
                                              children: SCOPE_DESCRIPTIONS[scope] || scope,
                                            }),
                                            _jsx("span", {
                                              className:
                                                "font-mono text-xs ml-auto text-muted-foreground/60",
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
                                    className:
                                      "text-xs font-bold uppercase tracking-wider text-muted-foreground",
                                    children: "\u5199\u5165\u6743\u9650",
                                  }),
                                  _jsx("div", {
                                    className: "space-y-1.5",
                                    children: writeScopes.map((scope) =>
                                      _jsxs(
                                        "div",
                                        {
                                          className:
                                            "flex items-center gap-2 text-sm text-muted-foreground",
                                          children: [
                                            _jsx(Zap, { className: "h-3.5 w-3.5 shrink-0" }),
                                            _jsx("span", {
                                              children: SCOPE_DESCRIPTIONS[scope] || scope,
                                            }),
                                            _jsx("span", {
                                              className:
                                                "font-mono text-xs ml-auto text-muted-foreground/60",
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
                                    className:
                                      "text-xs font-bold uppercase tracking-wider text-muted-foreground",
                                    children: "\u5176\u4ED6\u6743\u9650",
                                  }),
                                  _jsx("div", {
                                    className: "space-y-1.5",
                                    children: otherScopes.map((scope) =>
                                      _jsxs(
                                        "div",
                                        {
                                          className:
                                            "flex items-center gap-2 text-sm text-muted-foreground",
                                          children: [
                                            _jsx(ShieldCheck, {
                                              className: "h-3.5 w-3.5 shrink-0",
                                            }),
                                            _jsx("span", {
                                              children: SCOPE_DESCRIPTIONS[scope] || scope,
                                            }),
                                            _jsx("span", {
                                              className:
                                                "font-mono text-xs ml-auto text-muted-foreground/60",
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
                                    className:
                                      "text-xs font-bold uppercase tracking-wider text-muted-foreground",
                                    children: "\u8BA2\u9605\u4E8B\u4EF6",
                                  }),
                                  _jsx("div", {
                                    className: "flex flex-wrap gap-1.5",
                                    children: events.map((event) => {
                                      const found = EVENT_TYPES.find((e) => e.key === event);
                                      return _jsxs(
                                        Badge,
                                        {
                                          variant: "outline",
                                          className: "text-xs",
                                          children: [
                                            found ? found.label : event,
                                            found &&
                                              _jsxs("span", {
                                                className:
                                                  "font-mono text-muted-foreground/60 ml-1",
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
                  }),
                activeTab === "tools" &&
                  tools.length > 0 &&
                  _jsxs("div", {
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
                              "\u6B64\u5E94\u7528\u63D0\u4F9B\u7684\u547D\u4EE4\u548C\u5DE5\u5177\u3002",
                          }),
                        ],
                      }),
                      _jsx(Card, {
                        children: _jsx(CardContent, {
                          className: "pt-6",
                          children: _jsx(ToolsDisplay, { tools: tools }),
                        }),
                      }),
                    ],
                  }),
                activeTab === "config" &&
                  Object.keys(schemaProperties).length > 0 &&
                  _jsxs("div", {
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
                            children:
                              "\u5B89\u88C5\u524D\u586B\u5199\u6B64\u5E94\u7528\u6240\u9700\u7684\u914D\u7F6E\u3002",
                          }),
                        ],
                      }),
                      _jsx(Card, {
                        children: _jsx(CardContent, {
                          className: "space-y-4 pt-6",
                          children: Object.entries(schemaProperties).map(([key, prop]) =>
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
                                    value: configForm[key] || "",
                                    onChange: (e) =>
                                      setConfigForm({ ...configForm, [key]: e.target.value }),
                                    className: "h-8 text-xs font-mono",
                                    placeholder: prop.description || "",
                                  }),
                                  prop.description &&
                                    _jsx("p", {
                                      className: "text-xs text-muted-foreground",
                                      children: prop.description,
                                    }),
                                ],
                              },
                              key,
                            ),
                          ),
                        }),
                      }),
                    ],
                  }),
              ],
            }),
          ],
        }),
    ],
  });
}
