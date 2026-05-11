import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Card, CardContent } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { api } from "../lib/api";
import { useToast } from "@/hooks/use-toast";
import { useConfirm } from "@/components/ui/confirm-dialog";
import { useBotApps } from "@/hooks/use-bots";
import { useUninstallApp } from "@/hooks/use-apps";
import { queryKeys } from "@/lib/query-keys";
import { Blocks, Plus, Trash2, Loader2, Eye, Zap, Search } from "lucide-react";
import { AppIcon } from "../components/app-icon";
import { SCOPE_DESCRIPTIONS } from "../lib/constants";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
export function BotAppsTab({ botId }) {
  const { data: installations = [] } = useBotApps(botId);
  const [showInstall, setShowInstall] = useState(false);
  const { toast } = useToast();
  const { confirm, ConfirmDialog } = useConfirm();
  const qc = useQueryClient();
  const uninstallMutation = useUninstallApp();
  async function handleUninstall(appId, instId) {
    const ok = await confirm({
      title: "卸载确认",
      description: "确定要卸载此应用？",
      confirmText: "卸载",
      variant: "destructive",
    });
    if (!ok) return;
    try {
      await uninstallMutation.mutateAsync({ appId, instId });
      toast({ title: "已卸载" });
    } catch (e) {
      toast({ variant: "destructive", title: "卸载失败", description: e.message });
    }
  }
  async function handleToggle(inst) {
    try {
      await api.updateInstallation(inst.app_id, inst.id, { enabled: !inst.enabled });
      qc.invalidateQueries({ queryKey: queryKeys.bots.apps(botId) });
    } catch {}
  }
  return _jsxs("div", {
    className: "space-y-6 mt-4",
    children: [
      ConfirmDialog,
      _jsxs("div", {
        className: "flex items-center justify-between",
        children: [
          _jsxs("div", {
            children: [
              _jsx("h3", {
                className: "text-sm font-semibold",
                children: "\u5DF2\u5B89\u88C5\u7684\u5E94\u7528",
              }),
              _jsx("p", {
                className: "text-xs text-muted-foreground mt-0.5",
                children:
                  "\u7BA1\u7406\u6B64\u8D26\u53F7\u4E0A\u5B89\u88C5\u7684\u5E94\u7528\uFF0C\u63A7\u5236\u6743\u9650\u548C\u72B6\u6001\u3002",
              }),
            ],
          }),
          _jsxs(Button, {
            variant: "outline",
            size: "sm",
            onClick: () => setShowInstall(true),
            children: [_jsx(Plus, { className: "w-3.5 h-3.5 mr-1" }), " \u5B89\u88C5\u5E94\u7528"],
          }),
        ],
      }),
      installations.length === 0
        ? _jsxs("div", {
            className: "text-center py-16 space-y-3 border-2 border-dashed rounded-2xl",
            children: [
              _jsx(Blocks, { className: "w-10 h-10 mx-auto text-muted-foreground/40" }),
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
                  children: "\u6D4F\u89C8\u5E94\u7528\u5E02\u573A",
                }),
              }),
            ],
          })
        : _jsx("div", {
            className: "space-y-3",
            children: installations.map((inst) =>
              _jsx(
                Card,
                {
                  children: _jsx(CardContent, {
                    className: "py-4",
                    children: _jsxs("div", {
                      className: "flex items-start justify-between",
                      children: [
                        _jsxs("div", {
                          className: "flex items-start gap-3",
                          children: [
                            _jsx(AppIcon, {
                              icon: inst.app_icon,
                              iconUrl: inst.app_icon_url,
                              size: "h-10 w-10",
                            }),
                            _jsxs("div", {
                              className: "space-y-0.5",
                              children: [
                                _jsxs("div", {
                                  className: "flex items-center gap-2",
                                  children: [
                                    _jsx("span", {
                                      className: "text-sm font-semibold",
                                      children: inst.app_name,
                                    }),
                                    inst.handle
                                      ? _jsxs(Badge, {
                                          variant: "secondary",
                                          className: "text-[10px] font-mono",
                                          children: ["@", inst.handle],
                                        })
                                      : null,
                                  ],
                                }),
                                _jsx("p", {
                                  className: "text-xs text-muted-foreground font-mono",
                                  children: inst.app_slug,
                                }),
                              ],
                            }),
                          ],
                        }),
                        _jsxs("div", {
                          className: "flex items-center gap-2",
                          children: [
                            _jsx(Badge, {
                              variant: inst.enabled ? "default" : "outline",
                              className: "text-[10px]",
                              children: inst.enabled ? "运行中" : "已停用",
                            }),
                            _jsx(Button, {
                              variant: "outline",
                              size: "sm",
                              className: "h-7 text-xs",
                              onClick: () => handleToggle(inst),
                              children: inst.enabled ? "停用" : "启用",
                            }),
                            _jsx(Button, {
                              variant: "ghost",
                              size: "sm",
                              className: "h-7 text-xs text-destructive",
                              "aria-label": "\u5378\u8F7D",
                              onClick: () => handleUninstall(inst.app_id, inst.id),
                              children: _jsx(Trash2, { className: "w-3 h-3" }),
                            }),
                          ],
                        }),
                      ],
                    }),
                  }),
                },
                inst.id,
              ),
            ),
          }),
      _jsx(InstallDialog, {
        botId: botId,
        open: showInstall,
        onOpenChange: setShowInstall,
        onInstalled: () => {
          qc.invalidateQueries({ queryKey: queryKeys.bots.apps(botId) });
          qc.invalidateQueries({ queryKey: queryKeys.bots.all() });
          qc.invalidateQueries({ queryKey: queryKeys.marketplace.apps() });
          qc.invalidateQueries({ queryKey: queryKeys.marketplace.builtin() });
          qc.invalidateQueries({ queryKey: queryKeys.apps.all({ listing: "listed" }) });
        },
      }),
    ],
  });
}
// ==================== Unified Install Dialog ====================
function InstallDialog({ botId, open, onOpenChange, onInstalled }) {
  const [apps, setApps] = useState([]);
  const [loading, setLoading] = useState(false);
  const [confirmApp, setConfirmApp] = useState(null);
  const [handle, setHandle] = useState("");
  const [installing, setInstalling] = useState(false);
  const [error, setError] = useState("");
  const [search, setSearch] = useState("");
  const { toast } = useToast();
  useEffect(() => {
    if (!open) {
      setConfirmApp(null);
      setSearch("");
      return;
    }
    setLoading(true);
    Promise.all([api.listApps(), api.listApps({ listing: "listed" })])
      .then(([my, listed]) => {
        const seen = new Set();
        const merged = [];
        for (const a of [...(my || []), ...(listed || [])]) {
          if (!seen.has(a.id)) {
            seen.add(a.id);
            merged.push(a);
          }
        }
        setApps(merged);
      })
      .finally(() => setLoading(false));
  }, [open]);
  async function doInstall() {
    if (!confirmApp) return;
    setInstalling(true);
    setError("");
    try {
      await api.installApp(confirmApp.id, { bot_id: botId, handle: handle.trim() || undefined });
      toast({ title: "安装成功", description: `已安装 ${confirmApp.name}。` });
      onOpenChange(false);
      onInstalled();
    } catch (e) {
      setError(e.message);
    }
    setInstalling(false);
  }
  // Step 2: Confirm permissions + handle
  if (confirmApp) {
    const tools = confirmApp.tools || [];
    const events = confirmApp.events || [];
    const scopes = confirmApp.scopes || [];
    const readScopes = scopes.filter((s) => s.includes("read"));
    const writeScopes = scopes.filter((s) => !s.includes("read"));
    return _jsx(Dialog, {
      open: open,
      onOpenChange: onOpenChange,
      children: _jsxs(DialogContent, {
        className: "sm:max-w-2xl",
        children: [
          _jsxs(DialogHeader, {
            className: "sr-only",
            children: [
              _jsxs(DialogTitle, { children: ["\u5B89\u88C5 ", confirmApp.name] }),
              _jsx(DialogDescription, {
                children: "\u67E5\u770B\u6743\u9650\u5E76\u786E\u8BA4\u5B89\u88C5\u3002",
              }),
            ],
          }),
          _jsxs("div", {
            className: "py-2",
            children: [
              _jsxs("div", {
                className: "flex flex-col sm:flex-row gap-6",
                children: [
                  _jsxs("div", {
                    className: "sm:w-2/5 space-y-4 sm:border-r sm:pr-6",
                    children: [
                      _jsxs("div", {
                        className: "flex items-center gap-3",
                        children: [
                          _jsx(AppIcon, {
                            icon: confirmApp.icon,
                            iconUrl: confirmApp.icon_url,
                            size: "h-14 w-14",
                          }),
                          _jsxs("div", {
                            children: [
                              _jsx("h3", {
                                className: "text-lg font-bold",
                                children: confirmApp.name,
                              }),
                              _jsx("p", {
                                className: "text-xs text-muted-foreground font-mono",
                                children: confirmApp.slug,
                              }),
                            ],
                          }),
                        ],
                      }),
                      confirmApp.description
                        ? _jsx("p", {
                            className: "text-sm text-muted-foreground leading-relaxed",
                            children: confirmApp.description,
                          })
                        : null,
                    ],
                  }),
                  _jsxs("div", {
                    className: "sm:w-3/5 space-y-5",
                    children: [
                      _jsxs("div", {
                        className: "space-y-3",
                        children: [
                          _jsx("h4", {
                            className:
                              "text-xs font-bold uppercase tracking-wider text-muted-foreground",
                            children: "\u6B64\u5E94\u7528\u5C06\u80FD\u591F\uFF1A",
                          }),
                          readScopes.length > 0
                            ? _jsxs("div", {
                                className: "space-y-1.5",
                                children: [
                                  _jsx("p", {
                                    className:
                                      "text-[10px] font-medium text-muted-foreground uppercase tracking-wide",
                                    children: "\u67E5\u770B",
                                  }),
                                  readScopes.map((s) =>
                                    _jsxs(
                                      "div",
                                      {
                                        className: "flex items-start gap-2 text-sm",
                                        children: [
                                          _jsx(Eye, {
                                            className:
                                              "h-3.5 w-3.5 mt-0.5 text-muted-foreground shrink-0",
                                          }),
                                          _jsx("span", { children: SCOPE_DESCRIPTIONS[s] || s }),
                                        ],
                                      },
                                      s,
                                    ),
                                  ),
                                ],
                              })
                            : null,
                          writeScopes.length > 0
                            ? _jsxs("div", {
                                className: "space-y-1.5",
                                children: [
                                  _jsx("p", {
                                    className:
                                      "text-[10px] font-medium text-muted-foreground uppercase tracking-wide",
                                    children: "\u64CD\u4F5C",
                                  }),
                                  writeScopes.map((s) =>
                                    _jsxs(
                                      "div",
                                      {
                                        className: "flex items-start gap-2 text-sm",
                                        children: [
                                          _jsx(Zap, {
                                            className: "h-3.5 w-3.5 mt-0.5 text-primary shrink-0",
                                          }),
                                          _jsx("span", { children: SCOPE_DESCRIPTIONS[s] || s }),
                                        ],
                                      },
                                      s,
                                    ),
                                  ),
                                ],
                              })
                            : null,
                          tools.length > 0
                            ? _jsxs("div", {
                                className: "space-y-1.5",
                                children: [
                                  _jsx("p", {
                                    className:
                                      "text-[10px] font-medium text-muted-foreground uppercase tracking-wide",
                                    children: "\u547D\u4EE4",
                                  }),
                                  _jsx("div", {
                                    className: "flex flex-wrap gap-1.5",
                                    children: tools.map((t) =>
                                      _jsxs(
                                        Badge,
                                        {
                                          variant: "secondary",
                                          className: "font-mono text-xs",
                                          children: ["/", t.command || t.name],
                                        },
                                        t.name,
                                      ),
                                    ),
                                  }),
                                ],
                              })
                            : null,
                          events.length > 0
                            ? _jsxs("div", {
                                className: "space-y-1.5",
                                children: [
                                  _jsx("p", {
                                    className:
                                      "text-[10px] font-medium text-muted-foreground uppercase tracking-wide",
                                    children: "\u4E8B\u4EF6\u8BA2\u9605",
                                  }),
                                  _jsx("div", {
                                    className: "flex flex-wrap gap-1.5",
                                    children: events.map((e) =>
                                      _jsx(
                                        Badge,
                                        {
                                          variant: "outline",
                                          className: "font-mono text-[10px]",
                                          children: e,
                                        },
                                        e,
                                      ),
                                    ),
                                  }),
                                ],
                              })
                            : null,
                          scopes.length === 0 && tools.length === 0 && events.length === 0
                            ? _jsx("p", {
                                className: "text-sm text-muted-foreground",
                                children:
                                  "\u63A5\u6536 @mention \u6D88\u606F\u5E76\u6267\u884C\u54CD\u5E94\u3002",
                              })
                            : null,
                        ],
                      }),
                      _jsxs("div", {
                        className: "space-y-3 pt-2 border-t",
                        children: [
                          _jsxs("div", {
                            className: "space-y-1.5",
                            children: [
                              _jsx("label", {
                                htmlFor: "bot-install-handle",
                                className: "text-xs font-medium",
                                children: "Handle",
                              }),
                              _jsx(Input, {
                                id: "bot-install-handle",
                                value: handle,
                                onChange: (e) => setHandle(e.target.value),
                                placeholder: "\u5982 notify-prod",
                                className: "h-9 font-mono",
                              }),
                              _jsxs("p", {
                                className: "text-[10px] text-muted-foreground",
                                children: [
                                  "\u7528\u6237\u53D1\u9001 @",
                                  handle || "handle",
                                  " \u89E6\u53D1\u6B64\u5E94\u7528",
                                ],
                              }),
                            ],
                          }),
                          error
                            ? _jsx("p", { className: "text-xs text-destructive", children: error })
                            : null,
                        ],
                      }),
                    ],
                  }),
                ],
              }),
              _jsxs("div", {
                className: "flex justify-end gap-2 pt-4 mt-4 border-t",
                children: [
                  _jsx(Button, {
                    variant: "ghost",
                    onClick: () => setConfirmApp(null),
                    children: "\u8FD4\u56DE",
                  }),
                  _jsxs(Button, {
                    onClick: doInstall,
                    disabled: installing || !handle.trim(),
                    className: "px-6",
                    children: [
                      installing
                        ? _jsx(Loader2, { className: "h-4 w-4 animate-spin mr-1.5" })
                        : null,
                      "\u5141\u8BB8\u5E76\u5B89\u88C5",
                    ],
                  }),
                ],
              }),
            ],
          }),
        ],
      }),
    });
  }
  // Step 1: Pick an app
  const filtered = apps.filter(
    (a) =>
      !search ||
      a.name.toLowerCase().includes(search.toLowerCase()) ||
      (a.slug || "").toLowerCase().includes(search.toLowerCase()),
  );
  return _jsx(Dialog, {
    open: open,
    onOpenChange: onOpenChange,
    children: _jsxs(DialogContent, {
      className: "sm:max-w-lg",
      children: [
        _jsxs(DialogHeader, {
          children: [
            _jsx(DialogTitle, { children: "\u5B89\u88C5\u5E94\u7528" }),
            _jsx(DialogDescription, {
              children: "\u9009\u62E9\u8981\u5B89\u88C5\u7684\u5E94\u7528\u3002",
            }),
          ],
        }),
        loading
          ? _jsx("div", {
              className: "flex justify-center py-8",
              children: _jsx(Loader2, { className: "h-6 w-6 animate-spin text-muted-foreground" }),
            })
          : apps.length === 0
            ? _jsxs("div", {
                className: "text-center py-8 space-y-2",
                children: [
                  _jsx(Blocks, { className: "w-8 h-8 mx-auto text-muted-foreground/40" }),
                  _jsx("p", {
                    className: "text-xs text-muted-foreground",
                    children: "\u6CA1\u6709\u53EF\u7528\u7684\u5E94\u7528",
                  }),
                ],
              })
            : _jsxs("div", {
                className: "space-y-3",
                children: [
                  _jsxs("div", {
                    className: "relative",
                    children: [
                      _jsx(Search, {
                        className: "absolute left-3 top-2.5 h-3.5 w-3.5 text-muted-foreground",
                      }),
                      _jsx(Input, {
                        placeholder: "\u641C\u7D22\u5E94\u7528...",
                        value: search,
                        onChange: (e) => setSearch(e.target.value),
                        className: "pl-9 h-9 text-xs",
                        "aria-label": "\u641C\u7D22\u5E94\u7528",
                      }),
                    ],
                  }),
                  _jsxs("div", {
                    className: "space-y-1.5 max-h-72 overflow-y-auto",
                    children: [
                      filtered.map((app) =>
                        _jsxs(
                          "div",
                          {
                            className:
                              "flex items-center justify-between p-2.5 rounded-lg border bg-background hover:bg-muted/30 transition-colors",
                            children: [
                              _jsxs("div", {
                                className: "flex items-center gap-3 min-w-0",
                                children: [
                                  _jsx(AppIcon, {
                                    icon: app.icon,
                                    iconUrl: app.icon_url,
                                    size: "h-9 w-9",
                                  }),
                                  _jsxs("div", {
                                    className: "min-w-0",
                                    children: [
                                      _jsx("div", {
                                        className: "flex items-center gap-1.5",
                                        children: _jsx("span", {
                                          className: "text-sm font-medium",
                                          children: app.name,
                                        }),
                                      }),
                                      app.description
                                        ? _jsx("p", {
                                            className: "text-xs text-muted-foreground truncate",
                                            children: app.description,
                                          })
                                        : null,
                                    ],
                                  }),
                                ],
                              }),
                              _jsx(Button, {
                                size: "sm",
                                variant: "outline",
                                onClick: () => {
                                  setConfirmApp(app);
                                  setHandle(app.slug || "");
                                  setError("");
                                },
                                children: "\u5B89\u88C5",
                              }),
                            ],
                          },
                          app.id,
                        ),
                      ),
                      filtered.length === 0 &&
                        _jsx("p", {
                          className: "text-center text-xs text-muted-foreground py-4",
                          children: "\u6CA1\u6709\u5339\u914D\u7684\u5E94\u7528",
                        }),
                    ],
                  }),
                ],
              }),
      ],
    }),
  });
}
