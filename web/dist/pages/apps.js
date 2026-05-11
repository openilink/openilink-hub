import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Badge } from "../components/ui/badge";
import { Blocks, Download, Loader2, Search, RefreshCw } from "lucide-react";
import { botDisplayName } from "../lib/api";
import { useApps } from "@/hooks/use-apps";
import { useBots } from "@/hooks/use-bots";
import { useMarketplaceApps, useSyncMarketplaceApp } from "@/hooks/use-marketplace";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useToast } from "@/hooks/use-toast";
import { AppIcon } from "../components/app-icon";
import { parseTools } from "../components/tools-display";
// ==================== Page ====================
export function AppsPage() {
  const navigate = useNavigate();
  const { toast } = useToast();
  const { data: listedApps = [], isLoading: listedLoading } = useApps({ listing: "listed" });
  const { data: registryApps = [], isLoading: registryLoading } = useMarketplaceApps();
  const { data: bots = [] } = useBots();
  const syncAppMutation = useSyncMarketplaceApp();
  const loading = listedLoading || registryLoading;
  const [search, setSearch] = useState("");
  const [pendingApp, setPendingApp] = useState(null);
  const [selectedBotId, setSelectedBotId] = useState("");
  // Auto-select first bot when bots load
  if (bots.length > 0 && !selectedBotId) setSelectedBotId(bots[0].id);
  // Group registry apps by registry_name
  const registryGroups = useMemo(() => {
    const groups = {};
    for (const app of registryApps) {
      const name = app.registry_name || app.registry_url || "Registry";
      if (!groups[name]) groups[name] = [];
      groups[name].push(app);
    }
    return groups;
  }, [registryApps]);
  const registryNames = Object.keys(registryGroups);
  const showTabs = registryNames.length > 0;
  const syncing = syncAppMutation.isPending;
  function handleInstallConfirm() {
    if (!pendingApp || !selectedBotId) return;
    const appId = pendingApp.id || pendingApp.local_id;
    if (appId) {
      navigate(`/dashboard/accounts/${selectedBotId}/install/${appId}`);
      setPendingApp(null);
      return;
    }
    syncAppMutation.mutate(pendingApp.slug, {
      onSuccess: (synced) => {
        navigate(`/dashboard/accounts/${selectedBotId}/install/${synced.id}`);
        setPendingApp(null);
      },
      onError: (e) => toast({ variant: "destructive", title: "同步失败", description: e.message }),
    });
  }
  function filterApps(apps) {
    if (!search) return apps;
    const q = search.toLowerCase();
    return apps.filter(
      (a) => a.name?.toLowerCase().includes(q) || (a.slug || "").toLowerCase().includes(q),
    );
  }
  return _jsxs("div", {
    className: "space-y-6",
    children: [
      _jsx("div", {
        className: "flex items-start justify-between gap-4",
        children: _jsxs("div", {
          children: [
            _jsx("h1", {
              className: "text-2xl font-bold tracking-tight",
              children: "\u5E94\u7528\u5E02\u573A",
            }),
            _jsx("p", {
              className: "text-sm text-muted-foreground mt-0.5",
              children: "\u6D4F\u89C8\u548C\u5B89\u88C5\u5E94\u7528\u3002",
            }),
          ],
        }),
      }),
      _jsxs("div", {
        className: "relative max-w-sm",
        children: [
          _jsx(Search, {
            className: "absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground",
          }),
          _jsx(Input, {
            placeholder: "\u641C\u7D22\u5E94\u7528...",
            value: search,
            onChange: (e) => setSearch(e.target.value),
            className: "pl-9",
            "aria-label": "\u641C\u7D22\u5E94\u7528",
          }),
        ],
      }),
      loading
        ? _jsx("div", {
            className:
              "divide-y divide-border/50 rounded-xl border border-border/50 overflow-hidden",
            children: [1, 2, 3, 4].map((i) =>
              _jsxs(
                "div",
                {
                  className: "flex items-center gap-4 px-4 py-3.5 animate-pulse",
                  children: [
                    _jsx("div", { className: "h-9 w-9 rounded-lg bg-muted shrink-0" }),
                    _jsxs("div", {
                      className: "flex-1 space-y-1.5",
                      children: [
                        _jsx("div", { className: "h-3.5 w-32 rounded bg-muted" }),
                        _jsx("div", { className: "h-3 w-48 rounded bg-muted" }),
                      ],
                    }),
                    _jsx("div", { className: "h-7 w-14 rounded-md bg-muted shrink-0" }),
                  ],
                },
                i,
              ),
            ),
          })
        : showTabs
          ? _jsxs(Tabs, {
              defaultValue: "local",
              children: [
                _jsxs(TabsList, {
                  children: [
                    _jsx(TabsTrigger, { value: "local", children: "\u672C\u7AD9" }),
                    registryNames.map((name) =>
                      _jsx(TabsTrigger, { value: name, children: name }, name),
                    ),
                  ],
                }),
                _jsx(TabsContent, {
                  value: "local",
                  className: "mt-6",
                  children: _jsx(AppGrid, {
                    apps: filterApps(listedApps),
                    search: search,
                    onInstall: setPendingApp,
                  }),
                }),
                registryNames.map((name) =>
                  _jsx(
                    TabsContent,
                    {
                      value: name,
                      className: "mt-6",
                      children: _jsx(AppGrid, {
                        apps: filterApps(registryGroups[name]),
                        search: search,
                        onInstall: setPendingApp,
                      }),
                    },
                    name,
                  ),
                ),
              ],
            })
          : _jsx(AppGrid, {
              apps: filterApps(listedApps),
              search: search,
              onInstall: setPendingApp,
            }),
      _jsx(Dialog, {
        open: !!pendingApp,
        onOpenChange: (o) => !o && setPendingApp(null),
        children: _jsxs(DialogContent, {
          className: "sm:max-w-md",
          children: [
            _jsxs(DialogHeader, {
              children: [
                _jsx(DialogTitle, { children: "\u9009\u62E9\u8D26\u53F7" }),
                _jsxs(DialogDescription, {
                  children: [
                    "\u9009\u62E9\u8981\u5B89\u88C5\u300C",
                    pendingApp?.name,
                    "\u300D\u7684\u8D26\u53F7\u3002",
                  ],
                }),
              ],
            }),
            bots.length === 0
              ? _jsx("p", {
                  className: "text-sm text-muted-foreground py-4",
                  children: "\u8BF7\u5148\u521B\u5EFA\u4E00\u4E2A\u8D26\u53F7\u3002",
                })
              : _jsxs("div", {
                  className: "space-y-4 pt-2",
                  children: [
                    _jsxs(Select, {
                      value: selectedBotId,
                      onValueChange: setSelectedBotId,
                      children: [
                        _jsx(SelectTrigger, {
                          className: "w-full",
                          children: _jsx(SelectValue, { placeholder: "\u9009\u62E9\u8D26\u53F7" }),
                        }),
                        _jsx(SelectContent, {
                          children: bots.map((b) =>
                            _jsx(SelectItem, { value: b.id, children: botDisplayName(b) }, b.id),
                          ),
                        }),
                      ],
                    }),
                    _jsxs(Button, {
                      className: "w-full",
                      disabled: syncing,
                      onClick: handleInstallConfirm,
                      children: [
                        syncing ? _jsx(Loader2, { className: "h-4 w-4 animate-spin mr-2" }) : null,
                        "\u7EE7\u7EED\u5B89\u88C5",
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
// ==================== App Grid ====================
function AppGrid({ apps, search, onInstall }) {
  if (apps.length === 0) {
    return _jsxs("div", {
      className: "text-center py-16 space-y-3 border-2 border-dashed rounded-xl",
      children: [
        _jsx(Blocks, { className: "w-8 h-8 mx-auto text-muted-foreground/40" }),
        _jsx("p", {
          className: "text-sm text-muted-foreground",
          children: search ? "没有匹配的应用" : "暂无应用",
        }),
      ],
    });
  }
  return _jsx("div", {
    className: "divide-y divide-border/50 rounded-xl border border-border/50 overflow-hidden",
    children: apps.map((app) =>
      _jsxs(
        "div",
        {
          className:
            "group flex items-center gap-4 px-4 py-3.5 bg-card hover:bg-muted/40 transition-colors",
          children: [
            _jsx(AppIcon, { icon: app.icon, iconUrl: app.icon_url, size: "h-9 w-9" }),
            _jsxs("div", {
              className: "flex-1 min-w-0",
              children: [
                _jsxs("div", {
                  className: "flex items-center gap-2 flex-wrap",
                  children: [
                    _jsx("p", {
                      className: "text-sm font-semibold leading-tight",
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
                  className: "text-xs text-muted-foreground mt-0.5 line-clamp-1",
                  children: app.description || "暂无描述",
                }),
              ],
            }),
            app.author || app.owner_name
              ? _jsx("span", {
                  className: "text-[11px] text-muted-foreground/50 shrink-0 hidden sm:block",
                  children: app.author || app.owner_name,
                })
              : null,
            parseTools(app.tools).length > 0
              ? _jsxs("span", {
                  className: "text-[11px] text-muted-foreground/50 shrink-0 hidden md:block",
                  children: [parseTools(app.tools).length, " \u4E2A\u547D\u4EE4"],
                })
              : null,
            app.installed && app.update_available
              ? _jsxs(Button, {
                  size: "sm",
                  variant: "outline",
                  className: "shrink-0 gap-1.5",
                  onClick: () => onInstall(app),
                  children: [_jsx(RefreshCw, { className: "h-3.5 w-3.5" }), "\u66F4\u65B0"],
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
                    onClick: () => onInstall(app),
                    children: [_jsx(Download, { className: "h-3.5 w-3.5" }), "\u5B89\u88C5"],
                  }),
          ],
        },
        `${app.registry || "local"}-${app.slug || app.id}`,
      ),
    ),
  });
}
