import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { useState } from "react";
import {
  Users,
  Cpu,
  Globe,
  Blocks,
  Database,
  Trash2,
  Plus,
  UserPlus,
  KeyRound,
  Loader2,
  Pencil,
  Copy,
  Check,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useToast } from "@/hooks/use-toast";
import { useConfirm } from "@/components/ui/confirm-dialog";
import {
  useAdminStats,
  useAIConfig,
  useSaveAIConfig,
  useRegistrationConfig,
  useSetRegistrationConfig,
  useOIDCConfig,
  useSetOIDCConfig,
  useDeleteOIDCConfig,
  useRegistryConfig,
  useSetRegistryConfig,
  useRegistries,
  useCreateRegistry,
  useUpdateRegistry,
  useDeleteRegistry,
} from "@/hooks/use-admin";
const METRIC_CONFIG = [
  {
    label: "全站用户",
    key: "total_users",
    icon: Users,
    color: "text-blue-500",
    bg: "bg-blue-500/10",
  },
  {
    label: "微信账号",
    key: "total_bots",
    icon: Cpu,
    color: "text-emerald-500",
    bg: "bg-emerald-500/10",
  },
  {
    label: "已安装应用",
    key: "total_installations",
    icon: Globe,
    color: "text-violet-500",
    bg: "bg-violet-500/10",
  },
  {
    label: "活跃 App",
    key: "total_apps",
    icon: Blocks,
    color: "text-orange-500",
    bg: "bg-orange-500/10",
  },
];
function SkeletonCard() {
  return _jsx(Card, { className: "h-24 animate-pulse bg-muted/20 border-none" });
}
export function AdminOverviewPage() {
  const { data: stats, isLoading: loading } = useAdminStats();
  const { data: aiConfigData } = useAIConfig();
  const [aiConfig, setAIConfig] = useState(null);
  const saveAIMutation = useSaveAIConfig();
  const { toast } = useToast();
  // Sync query data into local state for form editing
  const effectiveAIConfig = aiConfig ?? aiConfigData;
  async function handleSaveAI() {
    if (!effectiveAIConfig) return;
    try {
      await saveAIMutation.mutateAsync(effectiveAIConfig);
      toast({ title: "全局 AI 配置已保存" });
    } catch (e) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
  }
  function updateAIConfig(patch) {
    setAIConfig((prev) => ({ ...(prev ?? aiConfigData), ...patch }));
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
              children: "\u7CFB\u7EDF\u6982\u89C8",
            }),
            _jsx("p", {
              className: "text-sm text-muted-foreground mt-0.5",
              children: "\u5E73\u53F0\u8FD0\u884C\u72B6\u6001\u4E0E\u914D\u7F6E\u3002",
            }),
          ],
        }),
      }),
      loading
        ? _jsxs("div", {
            className: "grid gap-6 md:grid-cols-2 lg:grid-cols-4",
            children: [
              _jsx(SkeletonCard, {}),
              _jsx(SkeletonCard, {}),
              _jsx(SkeletonCard, {}),
              _jsx(SkeletonCard, {}),
            ],
          })
        : _jsx("div", {
            className: "grid gap-6 md:grid-cols-2 lg:grid-cols-4",
            children: METRIC_CONFIG.map((m) =>
              _jsx(
                Card,
                {
                  className:
                    "border-border/50 bg-card/50 hover:bg-card transition-colors cursor-default",
                  children: _jsxs(CardContent, {
                    className: "p-5",
                    children: [
                      _jsx("div", {
                        className: "flex items-start justify-between mb-3",
                        children: _jsx("div", {
                          className: `h-8 w-8 rounded-lg ${m.bg} flex items-center justify-center ${m.color}`,
                          children: _jsx(m.icon, { className: "h-4 w-4" }),
                        }),
                      }),
                      _jsx("div", {
                        className: "text-2xl font-bold tabular-nums",
                        children: stats?.[m.key] || 0,
                      }),
                      _jsx("p", {
                        className: "text-xs font-semibold text-foreground/80",
                        children: m.label,
                      }),
                    ],
                  }),
                },
                m.label,
              ),
            ),
          }),
      _jsxs(Card, {
        className: "border-border/50 bg-card/30",
        children: [
          _jsx(CardHeader, { children: _jsx(CardTitle, { children: "\u7CFB\u7EDF\u72B6\u6001" }) }),
          _jsx(CardContent, {
            className: "space-y-4",
            children: _jsxs("div", {
              className: "grid gap-4 md:grid-cols-3",
              children: [
                _jsxs("div", {
                  className:
                    "p-4 rounded-2xl bg-muted/20 border border-border/50 flex items-center gap-4",
                  children: [
                    _jsx(Database, { className: "h-5 w-5 text-muted-foreground" }),
                    _jsxs("div", {
                      children: [
                        _jsx("p", {
                          className: "text-xs font-bold uppercase text-muted-foreground",
                          children: "PostgreSQL",
                        }),
                        _jsx("p", {
                          className: "text-sm font-bold",
                          children: "\u5DF2\u8FDE\u63A5",
                        }),
                      ],
                    }),
                  ],
                }),
                _jsxs("div", {
                  className:
                    "p-4 rounded-2xl bg-muted/20 border border-border/50 flex items-center gap-4",
                  children: [
                    _jsx(Globe, { className: "h-5 w-5 text-muted-foreground" }),
                    _jsxs("div", {
                      children: [
                        _jsx("p", {
                          className: "text-xs font-bold uppercase text-muted-foreground",
                          children: "WASM Runtime",
                        }),
                        _jsx("p", { className: "text-sm font-bold", children: "\u5C31\u7EEA" }),
                      ],
                    }),
                  ],
                }),
              ],
            }),
          }),
        ],
      }),
      _jsx(RegistrationConfigCard, {}),
      _jsx(OIDCConfigCard, {}),
      _jsxs("div", {
        className: "grid gap-8 md:grid-cols-2",
        children: [
          _jsxs(Card, {
            className: "border-border/50 bg-card/50",
            children: [
              _jsxs(CardHeader, {
                children: [
                  _jsx(CardTitle, { children: "AI \u914D\u7F6E" }),
                  _jsx(CardDescription, {
                    children: "\u6240\u6709\u8D26\u53F7\u7684\u9ED8\u8BA4 AI \u8BBE\u7F6E\u3002",
                  }),
                ],
              }),
              _jsx(CardContent, {
                children: _jsxs(Tabs, {
                  defaultValue: "basic",
                  children: [
                    _jsxs(TabsList, {
                      className: "mb-4",
                      children: [
                        _jsx(TabsTrigger, { value: "basic", children: "\u57FA\u7840" }),
                        _jsx(TabsTrigger, { value: "advanced", children: "\u9AD8\u7EA7" }),
                      ],
                    }),
                    _jsxs(TabsContent, {
                      value: "basic",
                      className: "space-y-4 mt-0",
                      children: [
                        _jsxs("div", {
                          className: "space-y-1.5",
                          children: [
                            _jsx(Label, {
                              className: "text-xs font-bold uppercase text-muted-foreground",
                              children: "\u63A5\u53E3\u5730\u5740",
                            }),
                            _jsx(Input, {
                              value: effectiveAIConfig?.base_url || "",
                              onChange: (e) => updateAIConfig({ base_url: e.target.value }),
                            }),
                          ],
                        }),
                        _jsxs("div", {
                          className: "space-y-1.5",
                          children: [
                            _jsx(Label, {
                              className: "text-xs font-bold uppercase text-muted-foreground",
                              children: "\u9ED8\u8BA4\u6A21\u578B",
                            }),
                            _jsx(Input, {
                              value: effectiveAIConfig?.model || "",
                              onChange: (e) => updateAIConfig({ model: e.target.value }),
                            }),
                          ],
                        }),
                        _jsxs("div", {
                          className: "space-y-1.5",
                          children: [
                            _jsx(Label, {
                              className: "text-xs font-bold uppercase text-muted-foreground",
                              children: "API Key",
                            }),
                            _jsx(Input, {
                              type: "password",
                              value: effectiveAIConfig?.api_key || "",
                              onChange: (e) => updateAIConfig({ api_key: e.target.value }),
                              placeholder: "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022",
                            }),
                          ],
                        }),
                        _jsxs("div", {
                          className: "space-y-1.5",
                          children: [
                            _jsx(Label, {
                              className: "text-xs font-bold uppercase text-muted-foreground",
                              children: "\u7CFB\u7EDF\u63D0\u793A\u8BCD",
                            }),
                            _jsx(Textarea, {
                              rows: 4,
                              value: effectiveAIConfig?.system_prompt || "",
                              onChange: (e) => updateAIConfig({ system_prompt: e.target.value }),
                              placeholder:
                                "\u8BBE\u7F6E AI \u7684\u7CFB\u7EDF\u89D2\u8272\u63D0\u793A\u8BCD",
                              className: "resize-y text-sm",
                            }),
                          ],
                        }),
                        _jsxs("div", {
                          className: "space-y-1.5",
                          children: [
                            _jsx(Label, {
                              className: "text-xs font-bold uppercase text-muted-foreground",
                              children: "\u5386\u53F2\u6D88\u606F\u8F6E\u6570",
                            }),
                            _jsx(Input, {
                              type: "number",
                              min: 0,
                              value: effectiveAIConfig?.max_history || "",
                              onChange: (e) => updateAIConfig({ max_history: e.target.value }),
                              placeholder: "\u9ED8\u8BA4 20 \u8F6E",
                            }),
                            _jsx("p", {
                              className: "text-xs text-muted-foreground",
                              children:
                                "AI \u5BF9\u8BDD\u65F6\u643A\u5E26\u7684\u5386\u53F2\u6D88\u606F\u8F6E\u6570\uFF0C0 \u8868\u793A\u4E0D\u643A\u5E26\u5386\u53F2\u3002",
                            }),
                          ],
                        }),
                      ],
                    }),
                    _jsxs(TabsContent, {
                      value: "advanced",
                      className: "space-y-4 mt-0",
                      children: [
                        _jsxs("div", {
                          className: "space-y-1.5",
                          children: [
                            _jsx(Label, {
                              className: "text-xs font-bold uppercase text-muted-foreground",
                              children: "\u53EF\u7528\u6A21\u578B\u5217\u8868",
                            }),
                            (() => {
                              let models = [];
                              try {
                                if (effectiveAIConfig?.available_models) {
                                  const parsed = JSON.parse(effectiveAIConfig.available_models);
                                  if (Array.isArray(parsed))
                                    models = parsed.filter((s) => typeof s === "string");
                                }
                              } catch {}
                              const setModels = (next) => {
                                setAIConfig((prev) => ({
                                  ...(prev ?? aiConfigData),
                                  available_models: JSON.stringify(next),
                                }));
                              };
                              return _jsxs("div", {
                                className: "space-y-2",
                                children: [
                                  models.length > 0 &&
                                    _jsx("div", {
                                      className: "flex flex-wrap gap-1.5",
                                      children: models.map((m, i) =>
                                        _jsxs(
                                          "span",
                                          {
                                            className:
                                              "inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-muted border text-xs font-mono",
                                            children: [
                                              m,
                                              _jsx("button", {
                                                type: "button",
                                                className:
                                                  "ml-0.5 text-muted-foreground hover:text-destructive",
                                                onClick: () =>
                                                  setModels(models.filter((_, j) => j !== i)),
                                                children: "\u00D7",
                                              }),
                                            ],
                                          },
                                          i,
                                        ),
                                      ),
                                    }),
                                  _jsx(Input, {
                                    placeholder:
                                      "\u8F93\u5165\u6A21\u578B\u540D\u79F0\uFF0C\u6309\u56DE\u8F66\u6DFB\u52A0",
                                    onKeyDown: (e) => {
                                      if (e.key === "Enter") {
                                        e.preventDefault();
                                        const v = e.target.value.trim();
                                        if (v && !models.includes(v)) {
                                          setModels([...models, v]);
                                          e.target.value = "";
                                        }
                                      }
                                    },
                                  }),
                                ],
                              });
                            })(),
                          ],
                        }),
                        _jsxs("div", {
                          className: "space-y-1.5",
                          children: [
                            _jsx(Label, {
                              className: "text-xs font-bold uppercase text-muted-foreground",
                              children: "\u81EA\u5B9A\u4E49 Headers",
                            }),
                            _jsx("p", {
                              className: "text-xs text-muted-foreground",
                              children:
                                "\u8C03\u7528 AI \u63A5\u53E3\u65F6\u9644\u52A0\u7684 HTTP \u8BF7\u6C42\u5934\uFF0C\u4F8B\u5982 OpenRouter \u5F52\u5C5E\u4FE1\u606F\u3002",
                            }),
                            _jsx("div", {
                              className: "space-y-2",
                              children: (() => {
                                let entries = [];
                                try {
                                  const raw = effectiveAIConfig?.custom_headers;
                                  if (raw) {
                                    const parsed = JSON.parse(raw);
                                    entries = Array.isArray(parsed)
                                      ? parsed
                                      : Object.entries(parsed);
                                  }
                                } catch {}
                                const sync = (next) => {
                                  updateAIConfig({
                                    custom_headers: next.length ? JSON.stringify(next) : "",
                                  });
                                };
                                return _jsxs(_Fragment, {
                                  children: [
                                    entries.map(([key, val], i) =>
                                      _jsxs(
                                        "div",
                                        {
                                          className: "flex gap-2 items-center",
                                          children: [
                                            _jsx(Input, {
                                              className: "flex-1",
                                              placeholder: "Header Name",
                                              value: key,
                                              onChange: (e) => {
                                                const next = [...entries];
                                                next[i] = [e.target.value, val];
                                                sync(next);
                                              },
                                            }),
                                            _jsx(Input, {
                                              className: "flex-1",
                                              placeholder: "Value",
                                              value: val,
                                              onChange: (e) => {
                                                const next = [...entries];
                                                next[i] = [key, e.target.value];
                                                sync(next);
                                              },
                                            }),
                                            _jsx(Button, {
                                              variant: "ghost",
                                              size: "icon",
                                              className:
                                                "shrink-0 h-8 w-8 text-muted-foreground hover:text-destructive",
                                              onClick: () =>
                                                sync(entries.filter((_, j) => j !== i)),
                                              children: _jsx(Trash2, { className: "h-3.5 w-3.5" }),
                                            }),
                                          ],
                                        },
                                        i,
                                      ),
                                    ),
                                    _jsxs(Button, {
                                      variant: "outline",
                                      size: "sm",
                                      className: "w-full",
                                      onClick: () => sync([...entries, ["", ""]]),
                                      children: [
                                        _jsx(Plus, { className: "h-3.5 w-3.5 mr-1" }),
                                        "\u6DFB\u52A0 Header",
                                      ],
                                    }),
                                  ],
                                });
                              })(),
                            }),
                          ],
                        }),
                        _jsxs("div", {
                          className:
                            "flex items-center justify-between p-3 rounded-xl bg-muted/20 border border-border/50",
                          children: [
                            _jsxs("div", {
                              children: [
                                _jsx("p", {
                                  className: "text-sm font-medium",
                                  children: "\u9690\u85CF\u601D\u8003\u8FC7\u7A0B",
                                }),
                                _jsx("p", {
                                  className: "text-xs text-muted-foreground",
                                  children:
                                    "\u542F\u7528\u540E\u4E0D\u4F1A\u5C06\u6A21\u578B\u7684\u601D\u8003\u5185\u5BB9\u53D1\u9001\u7ED9\u7528\u6237",
                                }),
                              ],
                            }),
                            _jsx(Switch, {
                              checked: effectiveAIConfig?.hide_thinking === "true",
                              onCheckedChange: (checked) =>
                                updateAIConfig({ hide_thinking: checked ? "true" : "false" }),
                            }),
                          ],
                        }),
                        _jsxs("div", {
                          className:
                            "flex items-center justify-between p-3 rounded-xl bg-muted/20 border border-border/50",
                          children: [
                            _jsxs("div", {
                              children: [
                                _jsx("p", {
                                  className: "text-sm font-medium",
                                  children: "Markdown \u8F6C\u7EAF\u6587\u672C",
                                }),
                                _jsx("p", {
                                  className: "text-xs text-muted-foreground",
                                  children:
                                    "\u542F\u7528\u540E\u5C06 AI \u56DE\u590D\u4E2D\u7684 Markdown \u683C\u5F0F\u8F6C\u4E3A\u7EAF\u6587\u672C",
                                }),
                              ],
                            }),
                            _jsx(Switch, {
                              checked: effectiveAIConfig?.strip_markdown === "true",
                              onCheckedChange: (checked) =>
                                updateAIConfig({ strip_markdown: checked ? "true" : "false" }),
                            }),
                          ],
                        }),
                      ],
                    }),
                  ],
                }),
              }),
              _jsx(CardFooter, {
                className: "flex justify-end",
                children: _jsx(Button, {
                  onClick: handleSaveAI,
                  disabled: saveAIMutation.isPending,
                  children: "\u4FDD\u5B58",
                }),
              }),
            ],
          }),
          _jsx(RegistryConfigCard, {}),
        ],
      }),
    ],
  });
}
// ==================== Registration Config ====================
function RegistrationConfigCard() {
  const { data: regConfig } = useRegistrationConfig();
  const setRegConfigMutation = useSetRegistrationConfig();
  const { toast } = useToast();
  async function handleToggle() {
    try {
      const newEnabled = regConfig?.enabled === "true" ? "false" : "true";
      await setRegConfigMutation.mutateAsync({ enabled: newEnabled });
      toast({ title: newEnabled === "true" ? "已开放注册" : "已关闭注册" });
    } catch (e) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
  }
  return _jsx(Card, {
    className: "border-border/50 bg-card/30",
    children: _jsx(CardContent, {
      className: "p-5",
      children: _jsxs("div", {
        className: "flex items-center justify-between",
        children: [
          _jsxs("div", {
            className: "flex items-center gap-4",
            children: [
              _jsx("div", {
                className:
                  "h-10 w-10 rounded-xl bg-amber-500/10 flex items-center justify-center text-amber-500",
                children: _jsx(UserPlus, { className: "h-5 w-5" }),
              }),
              _jsxs("div", {
                children: [
                  _jsx("p", {
                    className: "text-sm font-bold",
                    children: "\u5F00\u653E\u6CE8\u518C",
                  }),
                  _jsx("p", {
                    className: "text-xs text-muted-foreground",
                    children:
                      "\u5173\u95ED\u540E\uFF0C\u65B0\u7528\u6237\u65E0\u6CD5\u901A\u8FC7\u5BC6\u7801\u6CE8\u518C\u3001\u626B\u7801\u767B\u5F55\u6216 OAuth \u521B\u5EFA\u8D26\u53F7\u3002\u7BA1\u7406\u5458\u4ECD\u53EF\u624B\u52A8\u521B\u5EFA\u7528\u6237\u3002",
                  }),
                ],
              }),
            ],
          }),
          _jsx(Button, {
            variant: regConfig?.enabled === "true" ? "default" : "outline",
            size: "sm",
            onClick: handleToggle,
            disabled: setRegConfigMutation.isPending,
            children: regConfig?.enabled === "true" ? "已启用" : "已禁用",
          }),
        ],
      }),
    }),
  });
}
// ==================== Registry Config ====================
function RegistryConfigCard() {
  const { data: registryConfig } = useRegistryConfig();
  const { data: registries = [] } = useRegistries();
  const setRegistryConfigMutation = useSetRegistryConfig();
  const createRegistryMutation = useCreateRegistry();
  const updateRegistryMutation = useUpdateRegistry();
  const deleteRegistryMutation = useDeleteRegistry();
  const [newName, setNewName] = useState("");
  const [newUrl, setNewUrl] = useState("");
  const { toast } = useToast();
  const { confirm, ConfirmDialog } = useConfirm();
  const adding = createRegistryMutation.isPending;
  async function handleToggleExpose() {
    try {
      const newEnabled = registryConfig?.enabled === "true" ? "false" : "true";
      await setRegistryConfigMutation.mutateAsync({ enabled: newEnabled });
      toast({ title: "Registry 配置已保存" });
    } catch (e) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
  }
  async function handleAddRegistry() {
    if (!newName.trim() || !newUrl.trim()) return;
    try {
      await createRegistryMutation.mutateAsync({ name: newName.trim(), url: newUrl.trim() });
      setNewName("");
      setNewUrl("");
      toast({ title: "Registry 已添加" });
    } catch (e) {
      toast({ variant: "destructive", title: "添加失败", description: e.message });
    }
  }
  async function handleImportDefault() {
    try {
      await createRegistryMutation.mutateAsync({
        name: "OpeniLink Hub",
        url: "https://hub.openilink.com",
      });
      toast({ title: "已添加官方 Registry" });
    } catch (e) {
      toast({ variant: "destructive", title: "添加失败", description: e.message });
    }
  }
  async function handleToggleRegistry(reg) {
    try {
      await updateRegistryMutation.mutateAsync({ id: reg.id, data: { enabled: !reg.enabled } });
    } catch (e) {
      toast({ variant: "destructive", title: "操作失败", description: e.message });
    }
  }
  async function handleDeleteRegistry(reg) {
    const ok = await confirm({
      title: "删除确认",
      description: `确定删除 Registry "${reg.name}"？`,
      confirmText: "删除",
      variant: "destructive",
    });
    if (!ok) return;
    try {
      await deleteRegistryMutation.mutateAsync(reg.id);
      toast({ title: "已删除" });
    } catch (e) {
      toast({ variant: "destructive", title: "删除失败", description: e.message });
    }
  }
  return _jsxs(Card, {
    className: "border-border/50 bg-card/50",
    children: [
      ConfirmDialog,
      _jsxs(CardHeader, {
        children: [
          _jsx(CardTitle, { children: "Registry \u914D\u7F6E" }),
          _jsx(CardDescription, {
            children: "\u7BA1\u7406\u5E94\u7528\u5E02\u573A Registry \u6765\u6E90\u3002",
          }),
        ],
      }),
      _jsxs(CardContent, {
        className: "space-y-4",
        children: [
          _jsxs("div", {
            className:
              "flex items-center justify-between p-3 rounded-xl bg-muted/20 border border-border/50",
            children: [
              _jsxs("div", {
                children: [
                  _jsx("p", {
                    className: "text-sm font-medium",
                    children: "\u5BF9\u5916\u66B4\u9732 Registry",
                  }),
                  _jsx("p", {
                    className: "text-xs text-muted-foreground",
                    children:
                      "\u5141\u8BB8\u5176\u4ED6 Hub \u4ECE\u6B64\u5B9E\u4F8B\u62C9\u53D6\u5E94\u7528",
                  }),
                ],
              }),
              _jsx(Switch, {
                "aria-label": "\u5BF9\u5916\u66B4\u9732 Registry",
                checked: registryConfig?.enabled === "true",
                onCheckedChange: handleToggleExpose,
                disabled: setRegistryConfigMutation.isPending,
              }),
            ],
          }),
          _jsxs("div", {
            className: "space-y-2",
            children: [
              _jsx("p", {
                className: "text-xs font-bold uppercase tracking-widest text-muted-foreground",
                children: "Registry \u6765\u6E90",
              }),
              registries.length === 0
                ? _jsxs("div", {
                    className:
                      "flex items-center justify-between p-3 rounded-lg border border-dashed bg-muted/10",
                    children: [
                      _jsx("p", {
                        className: "text-sm text-muted-foreground",
                        children: "\u6682\u65E0 Registry \u6765\u6E90",
                      }),
                      window.location.origin !== "https://hub.openilink.com" &&
                        _jsxs(Button, {
                          size: "sm",
                          onClick: handleImportDefault,
                          disabled: adding,
                          children: [
                            _jsx(Globe, { className: "w-3.5 h-3.5 mr-1" }),
                            " \u4E00\u952E\u5BFC\u5165\u5B98\u65B9\u6E90",
                          ],
                        }),
                    ],
                  })
                : registries.map((reg) =>
                    _jsxs(
                      "div",
                      {
                        className:
                          "flex items-center justify-between p-2.5 rounded-lg border bg-background",
                        children: [
                          _jsxs("div", {
                            className: "min-w-0",
                            children: [
                              _jsx("p", {
                                className: "text-sm font-medium truncate",
                                children: reg.name,
                              }),
                              _jsx("p", {
                                className: "text-xs text-muted-foreground font-mono truncate",
                                children: reg.url,
                              }),
                            ],
                          }),
                          _jsxs("div", {
                            className: "flex items-center gap-2 shrink-0",
                            children: [
                              _jsx(Switch, {
                                "aria-label": `启用 ${reg.name}`,
                                checked: reg.enabled,
                                onCheckedChange: () => handleToggleRegistry(reg),
                                disabled: updateRegistryMutation.isPending,
                              }),
                              _jsxs(Tooltip, {
                                children: [
                                  _jsx(TooltipTrigger, {
                                    asChild: true,
                                    children: _jsx(Button, {
                                      variant: "ghost",
                                      size: "icon",
                                      className: "h-7 w-7 text-destructive hover:text-destructive",
                                      onClick: () => handleDeleteRegistry(reg),
                                      children: _jsx(Trash2, { className: "w-3.5 h-3.5" }),
                                    }),
                                  }),
                                  _jsx(TooltipContent, { children: "\u5220\u9664" }),
                                ],
                              }),
                            ],
                          }),
                        ],
                      },
                      reg.id,
                    ),
                  ),
            ],
          }),
          _jsxs("div", {
            className: "space-y-2 pt-2 border-t",
            children: [
              _jsx("p", {
                className: "text-xs font-bold uppercase tracking-widest text-muted-foreground",
                children: "\u6DFB\u52A0 Registry",
              }),
              _jsxs("div", {
                className: "flex gap-2",
                children: [
                  _jsx(Input, {
                    placeholder: "\u540D\u79F0",
                    value: newName,
                    onChange: (e) => setNewName(e.target.value),
                    className: "flex-1",
                  }),
                  _jsx(Input, {
                    placeholder: "https://hub.openilink.com",
                    value: newUrl,
                    onChange: (e) => setNewUrl(e.target.value),
                    className: "flex-[2]",
                  }),
                  _jsxs(Button, {
                    size: "sm",
                    onClick: handleAddRegistry,
                    disabled: adding || !newName.trim() || !newUrl.trim(),
                    children: [_jsx(Plus, { className: "w-3.5 h-3.5 mr-1" }), " \u6DFB\u52A0"],
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
function OIDCConfigCard() {
  const { data: providers = [] } = useOIDCConfig();
  const setOIDCMutation = useSetOIDCConfig();
  const deleteOIDCMutation = useDeleteOIDCConfig();
  const [editingSlug, setEditingSlug] = useState(null);
  const [slug, setSlug] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [issuerUrl, setIssuerUrl] = useState("");
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [scopes, setScopes] = useState("");
  const [copiedSlug, setCopiedSlug] = useState(null);
  const { toast } = useToast();
  const { confirm, ConfirmDialog } = useConfirm();
  const saving = setOIDCMutation.isPending;
  const isEditing = editingSlug !== null;
  function resetForm() {
    setEditingSlug(null);
    setSlug("");
    setDisplayName("");
    setIssuerUrl("");
    setClientId("");
    setClientSecret("");
    setScopes("");
  }
  function handleEdit(p) {
    setEditingSlug(p.slug);
    setSlug(p.slug);
    setDisplayName(p.display_name);
    setIssuerUrl(p.issuer_url);
    setClientId(p.client_id);
    setClientSecret("");
    setScopes(p.scopes || "");
  }
  async function handleSave() {
    const normalizedSlug = slug.trim();
    if (!normalizedSlug || !issuerUrl.trim() || !clientId.trim()) return;
    if (!isEditing && providers.some((p) => p.slug === normalizedSlug)) {
      toast({ variant: "destructive", title: "Slug 已存在", description: "请使用编辑功能修改。" });
      return;
    }
    try {
      await setOIDCMutation.mutateAsync({
        slug: normalizedSlug,
        data: {
          display_name: displayName.trim() || normalizedSlug,
          issuer_url: issuerUrl.trim(),
          client_id: clientId.trim(),
          client_secret: clientSecret.trim(),
          scopes: scopes.trim(),
        },
      });
      resetForm();
      toast({ title: isEditing ? "OIDC 提供商已更新" : "OIDC 提供商已添加" });
    } catch (e) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
  }
  async function handleDelete(s, name) {
    const ok = await confirm({
      title: "删除确认",
      description: `确定删除 OIDC 提供商 "${name}"？`,
      confirmText: "删除",
      variant: "destructive",
    });
    if (!ok) return;
    try {
      await deleteOIDCMutation.mutateAsync(s);
      if (editingSlug === s) resetForm();
      toast({ title: "已删除" });
    } catch (e) {
      toast({ variant: "destructive", title: "删除失败", description: e.message });
    }
  }
  async function handleCopyCallback(providerSlug) {
    const callbackUrl = `${window.location.origin}/api/auth/oidc/${providerSlug}/callback`;
    try {
      await navigator.clipboard.writeText(callbackUrl);
      setCopiedSlug(providerSlug);
      setTimeout(() => setCopiedSlug(null), 2000);
    } catch {
      toast({ variant: "destructive", title: "复制失败", description: "请手动复制回调地址" });
    }
  }
  return _jsxs(Card, {
    className: "border-border/50 bg-card/50",
    children: [
      ConfirmDialog,
      _jsxs(CardHeader, {
        children: [
          _jsx(CardTitle, { children: "OIDC \u8EAB\u4EFD\u63D0\u4F9B\u5546" }),
          _jsx(CardDescription, {
            children:
              "\u6DFB\u52A0\u81EA\u5B9A\u4E49 OIDC \u8EAB\u4EFD\u63D0\u4F9B\u5546\uFF08\u5982 Pocket-ID\u3001Keycloak\u3001Authentik \u7B49\uFF09\uFF0C\u7528\u6237\u53EF\u901A\u8FC7\u8FD9\u4E9B\u670D\u52A1\u767B\u5F55\u3002",
          }),
        ],
      }),
      _jsxs(CardContent, {
        className: "space-y-4",
        children: [
          providers.length > 0 &&
            _jsx("div", {
              className: "space-y-2",
              children: providers.map((p) =>
                _jsxs(
                  "div",
                  {
                    className: "space-y-1.5 p-3 rounded-lg border bg-background",
                    children: [
                      _jsxs("div", {
                        className: "flex items-center justify-between",
                        children: [
                          _jsxs("div", {
                            className: "min-w-0 flex items-center gap-3",
                            children: [
                              _jsx(KeyRound, {
                                className: "h-4 w-4 text-muted-foreground shrink-0",
                              }),
                              _jsxs("div", {
                                className: "min-w-0",
                                children: [
                                  _jsx("p", {
                                    className: "text-sm font-medium truncate",
                                    children: p.display_name,
                                  }),
                                  _jsx("p", {
                                    className: "text-xs text-muted-foreground font-mono truncate",
                                    children: p.issuer_url,
                                  }),
                                ],
                              }),
                            ],
                          }),
                          _jsxs("div", {
                            className: "flex items-center gap-1 shrink-0",
                            children: [
                              _jsxs(Tooltip, {
                                children: [
                                  _jsx(TooltipTrigger, {
                                    asChild: true,
                                    children: _jsx(Button, {
                                      variant: "ghost",
                                      size: "icon",
                                      className: "h-7 w-7",
                                      onClick: () => handleEdit(p),
                                      children: _jsx(Pencil, { className: "w-3.5 h-3.5" }),
                                    }),
                                  }),
                                  _jsx(TooltipContent, { children: "\u7F16\u8F91" }),
                                ],
                              }),
                              _jsxs(Tooltip, {
                                children: [
                                  _jsx(TooltipTrigger, {
                                    asChild: true,
                                    children: _jsx(Button, {
                                      variant: "ghost",
                                      size: "icon",
                                      className: "h-7 w-7 text-destructive hover:text-destructive",
                                      onClick: () => handleDelete(p.slug, p.display_name),
                                      children: _jsx(Trash2, { className: "w-3.5 h-3.5" }),
                                    }),
                                  }),
                                  _jsx(TooltipContent, { children: "\u5220\u9664" }),
                                ],
                              }),
                            ],
                          }),
                        ],
                      }),
                      _jsxs("div", {
                        className: "flex items-center gap-1.5 ml-7",
                        children: [
                          _jsxs("p", {
                            className: "text-xs text-muted-foreground font-mono truncate",
                            children: [
                              "\u56DE\u8C03\u5730\u5740: ",
                              window.location.origin,
                              "/api/auth/oidc/",
                              p.slug,
                              "/callback",
                            ],
                          }),
                          _jsxs(Tooltip, {
                            children: [
                              _jsx(TooltipTrigger, {
                                asChild: true,
                                children: _jsx(Button, {
                                  variant: "ghost",
                                  size: "icon",
                                  className: "h-5 w-5 shrink-0",
                                  onClick: () => handleCopyCallback(p.slug),
                                  children:
                                    copiedSlug === p.slug
                                      ? _jsx(Check, { className: "w-3 h-3 text-green-500" })
                                      : _jsx(Copy, { className: "w-3 h-3" }),
                                }),
                              }),
                              _jsx(TooltipContent, {
                                children: "\u590D\u5236\u56DE\u8C03\u5730\u5740",
                              }),
                            ],
                          }),
                        ],
                      }),
                    ],
                  },
                  p.slug,
                ),
              ),
            }),
          _jsxs("div", {
            className: "space-y-3 pt-2 border-t",
            children: [
              _jsxs("div", {
                className: "flex items-center justify-between",
                children: [
                  _jsx("p", {
                    className: "text-xs font-bold uppercase tracking-widest text-muted-foreground",
                    children: isEditing ? "编辑 OIDC 提供商" : "添加 OIDC 提供商",
                  }),
                  isEditing &&
                    _jsx(Button, {
                      variant: "ghost",
                      size: "sm",
                      className: "h-6 text-xs",
                      onClick: resetForm,
                      children: "\u53D6\u6D88\u7F16\u8F91",
                    }),
                ],
              }),
              _jsxs("div", {
                className: "grid gap-2 sm:grid-cols-2",
                children: [
                  _jsx(Input, {
                    placeholder: "\u6807\u8BC6 (slug, \u5982 pocket-id)",
                    value: slug,
                    onChange: (e) =>
                      setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, "")),
                    disabled: isEditing,
                  }),
                  _jsx(Input, {
                    placeholder: "\u663E\u793A\u540D\u79F0",
                    value: displayName,
                    onChange: (e) => setDisplayName(e.target.value),
                  }),
                ],
              }),
              _jsx(Input, {
                placeholder: "Issuer URL (\u5982 https://auth.example.com)",
                value: issuerUrl,
                onChange: (e) => setIssuerUrl(e.target.value),
                disabled: isEditing,
              }),
              _jsxs("div", {
                className: "grid gap-2 sm:grid-cols-2",
                children: [
                  _jsx(Input, {
                    placeholder: "Client ID",
                    value: clientId,
                    onChange: (e) => setClientId(e.target.value),
                  }),
                  _jsx(Input, {
                    type: "password",
                    placeholder: isEditing ? "Client Secret (留空保持不变)" : "Client Secret",
                    value: clientSecret,
                    onChange: (e) => setClientSecret(e.target.value),
                  }),
                ],
              }),
              _jsx(Input, {
                placeholder: "Scopes (\u9ED8\u8BA4: openid profile email)",
                value: scopes,
                onChange: (e) => setScopes(e.target.value),
              }),
              _jsxs(Button, {
                size: "sm",
                onClick: handleSave,
                disabled: saving || !slug.trim() || !issuerUrl.trim() || !clientId.trim(),
                children: [
                  saving
                    ? _jsx(Loader2, { className: "w-3.5 h-3.5 mr-1 animate-spin" })
                    : isEditing
                      ? _jsx(Check, { className: "w-3.5 h-3.5 mr-1" })
                      : _jsx(Plus, { className: "w-3.5 h-3.5 mr-1" }),
                  isEditing ? "保存" : "添加",
                ],
              }),
            ],
          }),
        ],
      }),
    ],
  });
}
