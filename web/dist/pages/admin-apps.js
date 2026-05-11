import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { useState } from "react";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Badge } from "../components/ui/badge";
import { Dialog, DialogContent } from "../components/ui/dialog";
import { api } from "../lib/api";
import { Blocks, Trash2, X, Pencil } from "lucide-react";
import { useConfirm, usePrompt } from "@/components/ui/confirm-dialog";
import { useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "@/lib/query-keys";
import {
  useAdminApps,
  useSetAppListing,
  useReviewListing,
  useDeleteAdminApp,
} from "@/hooks/use-admin";
export function AdminAppsTab() {
  const queryClient = useQueryClient();
  const { data: apps = [] } = useAdminApps();
  const [selected, setSelected] = useState(null);
  const [editing, setEditing] = useState(false);
  const [error, setError] = useState("");
  const { confirm, ConfirmDialog } = useConfirm();
  const { prompt, PromptDialog } = usePrompt();
  const setListingMutation = useSetAppListing();
  const reviewMutation = useReviewListing();
  const deleteMutation = useDeleteAdminApp();
  function openDetail(app) {
    setSelected(app);
    setEditing(false);
  }
  async function toggleListing(e, app) {
    e.stopPropagation();
    const newListing = app.listing === "listed" ? "unlisted" : "listed";
    try {
      await setListingMutation.mutateAsync({ id: app.id, listing: newListing });
    } catch (err) {
      setError(err.message);
    }
  }
  async function handleDelete(app) {
    const ok = await confirm({
      title: "删除确认",
      description: `删除 App "${app.name}"？此操作不可恢复。`,
      confirmText: "删除",
      variant: "destructive",
    });
    if (!ok) return;
    try {
      await deleteMutation.mutateAsync(app.id);
      setSelected(null);
    } catch (err) {
      setError(err.message);
    }
  }
  async function handleApprove(app) {
    try {
      await reviewMutation.mutateAsync({ appId: app.id, approve: true });
    } catch (err) {
      setError(err.message);
    }
  }
  async function handleReject(app) {
    const reason = await prompt({
      title: "拒绝 App",
      description: "请输入拒绝原因",
      placeholder: "拒绝原因",
    });
    if (!reason) return;
    try {
      await reviewMutation.mutateAsync({ appId: app.id, approve: false, reason });
    } catch (err) {
      setError(err.message);
    }
  }
  return _jsxs("div", {
    className: "space-y-3",
    children: [
      ConfirmDialog,
      PromptDialog,
      error ? _jsx("p", { className: "text-xs text-destructive", children: error }) : null,
      _jsx("div", {
        className: "space-y-1",
        children: apps.map((app) =>
          _jsxs(
            "div",
            {
              onClick: () => openDetail(app),
              className: `flex items-center justify-between p-2.5 rounded-lg border cursor-pointer hover:border-primary/50 ${selected?.id === app.id ? "border-primary bg-primary/5" : "bg-card"}`,
              children: [
                _jsxs("div", {
                  className: "flex items-center gap-3",
                  children: [
                    _jsx("div", {
                      className:
                        "w-8 h-8 rounded-lg bg-secondary flex items-center justify-center text-base",
                      children:
                        app.icon || _jsx(Blocks, { className: "w-4 h-4 text-muted-foreground" }),
                    }),
                    _jsxs("div", {
                      children: [
                        _jsxs("div", {
                          className: "flex items-center gap-1.5",
                          children: [
                            _jsx("span", { className: "text-xs font-medium", children: app.name }),
                            _jsx("span", {
                              className: "text-xs text-muted-foreground font-mono",
                              children: app.slug,
                            }),
                            app.listing === "listed"
                              ? _jsx(Badge, {
                                  variant: "default",
                                  className: "text-[10px]",
                                  children: "\u5DF2\u4E0A\u67B6",
                                })
                              : null,
                            app.listing === "pending"
                              ? _jsx(Badge, {
                                  variant: "outline",
                                  className: "text-[10px] text-orange-500 border-orange-500",
                                  children: "\u5F85\u5BA1\u6838",
                                })
                              : null,
                            app.listing === "rejected"
                              ? _jsx(Badge, {
                                  variant: "destructive",
                                  className: "text-[10px]",
                                  children: "\u5DF2\u62D2\u7EDD",
                                })
                              : null,
                          ],
                        }),
                        _jsxs("p", {
                          className: "text-xs text-muted-foreground",
                          children: [
                            app.owner_name && `by ${app.owner_name} · `,
                            (app.tools || []).length,
                            " \u5DE5\u5177 \u00B7 ",
                            (app.events || []).length,
                            " \u4E8B\u4EF6",
                          ],
                        }),
                      ],
                    }),
                  ],
                }),
                _jsxs("div", {
                  className: "flex items-center gap-1 shrink-0",
                  children: [
                    app.listing === "pending"
                      ? _jsxs(_Fragment, {
                          children: [
                            _jsx(Button, {
                              size: "xs",
                              variant: "ghost",
                              className: "text-primary bg-primary/10 hover:bg-primary/20",
                              onClick: (e) => {
                                e.stopPropagation();
                                handleApprove(app);
                              },
                              children: "\u901A\u8FC7",
                            }),
                            _jsx(Button, {
                              size: "xs",
                              variant: "ghost",
                              className:
                                "text-destructive bg-destructive/10 hover:bg-destructive/20",
                              onClick: (e) => {
                                e.stopPropagation();
                                handleReject(app);
                              },
                              children: "\u62D2\u7EDD",
                            }),
                          ],
                        })
                      : null,
                    _jsx(Button, {
                      size: "xs",
                      variant: app.listing === "listed" ? "ghost" : "secondary",
                      className:
                        app.listing === "listed"
                          ? "text-primary bg-primary/10 hover:bg-primary/20"
                          : "",
                      onClick: (e) => toggleListing(e, app),
                      children: app.listing === "listed" ? "下架" : "上架",
                    }),
                  ],
                }),
              ],
            },
            app.id,
          ),
        ),
      }),
      apps.length === 0
        ? _jsx("p", {
            className: "text-center text-sm text-muted-foreground py-8",
            children: "\u6682\u65E0 App",
          })
        : null,
      _jsx(Dialog, {
        open: !!selected,
        onOpenChange: (open) => {
          if (!open) setSelected(null);
        },
        children: _jsx(DialogContent, {
          className: "max-w-lg max-h-[80vh] overflow-y-auto p-0",
          children: selected
            ? editing
              ? _jsx(AppEditForm, {
                  app: selected,
                  onSave: () => {
                    setEditing(false);
                    setSelected(null);
                    queryClient.invalidateQueries({ queryKey: queryKeys.admin.apps() });
                  },
                  onCancel: () => setEditing(false),
                })
              : _jsx(AppDetailView, {
                  app: selected,
                  onEdit: () => setEditing(true),
                  onDelete: () => handleDelete(selected),
                  onClose: () => setSelected(null),
                  onToggleListing: async () => {
                    const newListing = selected.listing === "listed" ? "unlisted" : "listed";
                    try {
                      await setListingMutation.mutateAsync({
                        id: selected.id,
                        listing: newListing,
                      });
                      setSelected({ ...selected, listing: newListing });
                    } catch {
                      // mutation error handled by react-query
                    }
                  },
                })
            : null,
        }),
      }),
    ],
  });
}
function AppDetailView({ app, onEdit, onDelete, onClose, onToggleListing }) {
  const tools = app.tools || [];
  const events = app.events || [];
  const scopes = app.scopes || [];
  return _jsxs(_Fragment, {
    children: [
      _jsxs("div", {
        className: "p-4 border-b",
        children: [
          _jsxs("div", {
            className: "flex items-center gap-2",
            children: [
              app.icon ? _jsx("span", { className: "text-lg", children: app.icon }) : null,
              _jsx("span", { className: "font-semibold", children: app.name }),
              _jsx("span", {
                className: "text-xs text-muted-foreground font-mono",
                children: app.slug,
              }),
            ],
          }),
          _jsx("p", {
            className: "text-xs text-muted-foreground mt-0.5",
            children: app.description || "无描述",
          }),
          _jsxs("div", {
            className: "flex gap-3 mt-1 text-xs text-muted-foreground",
            children: [
              app.owner_name
                ? _jsxs("span", { children: ["\u62E5\u6709\u8005: ", app.owner_name] })
                : null,
              app.homepage
                ? _jsx("a", {
                    href: app.homepage,
                    target: "_blank",
                    rel: "noopener",
                    className: "text-primary hover:underline",
                    children: "\u4E3B\u9875",
                  })
                : null,
              _jsx("span", { children: new Date(app.created_at * 1000).toLocaleDateString() }),
            ],
          }),
        ],
      }),
      tools.length > 0
        ? _jsxs("div", {
            className: "p-4 border-b space-y-2",
            children: [
              _jsxs("p", {
                className: "text-xs font-medium",
                children: ["\u5DE5\u5177 (", tools.length, ")"],
              }),
              tools.map((t, i) =>
                _jsxs(
                  "div",
                  {
                    className: "text-xs p-2 rounded border bg-card space-y-0.5",
                    children: [
                      _jsxs("div", {
                        className: "flex items-center gap-2",
                        children: [
                          _jsx("code", { className: "font-mono font-medium", children: t.name }),
                          t.command
                            ? _jsxs(Badge, {
                                variant: "outline",
                                className: "text-[10px] font-mono",
                                children: ["/", t.command],
                              })
                            : null,
                        ],
                      }),
                      _jsx("p", { className: "text-muted-foreground", children: t.description }),
                      t.parameters
                        ? _jsx("pre", {
                            className:
                              "text-[10px] font-mono text-muted-foreground mt-1 overflow-x-auto",
                            children:
                              typeof t.parameters === "string"
                                ? t.parameters
                                : JSON.stringify(t.parameters, null, 2),
                          })
                        : null,
                    ],
                  },
                  i,
                ),
              ),
            ],
          })
        : null,
      events.length > 0 || scopes.length > 0
        ? _jsxs("div", {
            className: "p-4 border-b space-y-2",
            children: [
              events.length > 0
                ? _jsxs("div", {
                    children: [
                      _jsx("p", {
                        className: "text-xs font-medium mb-1",
                        children: "\u4E8B\u4EF6\u8BA2\u9605",
                      }),
                      _jsx("div", {
                        className: "flex flex-wrap gap-1",
                        children: events.map((e) =>
                          _jsx(
                            Badge,
                            { variant: "outline", className: "text-[10px] font-mono", children: e },
                            e,
                          ),
                        ),
                      }),
                    ],
                  })
                : null,
              scopes.length > 0
                ? _jsxs("div", {
                    children: [
                      _jsx("p", {
                        className: "text-xs font-medium mb-1",
                        children: "\u6743\u9650",
                      }),
                      _jsx("div", {
                        className: "flex flex-wrap gap-1",
                        children: scopes.map((s) =>
                          _jsx(
                            Badge,
                            {
                              variant: "secondary",
                              className: "text-[10px] font-mono",
                              children: s,
                            },
                            s,
                          ),
                        ),
                      }),
                    ],
                  })
                : null,
            ],
          })
        : null,
      _jsxs("div", {
        className: "p-4 flex justify-between",
        children: [
          _jsxs("div", {
            className: "flex gap-2",
            children: [
              _jsxs(Button, {
                variant: "destructive",
                size: "sm",
                onClick: onDelete,
                children: [_jsx(Trash2, { className: "w-3.5 h-3.5 mr-1" }), " \u5220\u9664"],
              }),
              _jsx(Button, {
                variant: "outline",
                size: "sm",
                onClick: onToggleListing,
                children: app.listing === "listed" ? "下架" : "上架",
              }),
            ],
          }),
          _jsxs("div", {
            className: "flex gap-2",
            children: [
              _jsxs(Button, {
                variant: "outline",
                size: "sm",
                onClick: onEdit,
                children: [_jsx(Pencil, { className: "w-3.5 h-3.5 mr-1" }), " \u7F16\u8F91"],
              }),
              _jsx(Button, {
                variant: "outline",
                size: "sm",
                onClick: onClose,
                children: "\u5173\u95ED",
              }),
            ],
          }),
        ],
      }),
    ],
  });
}
function AppEditForm({ app, onSave, onCancel }) {
  const [form, setForm] = useState({
    name: app.name || "",
    description: app.description || "",
    icon: app.icon || "",
    homepage: app.homepage || "",
    tools: JSON.stringify(app.tools || [], null, 2),
    events: JSON.stringify(app.events || [], null, 2),
    scopes: JSON.stringify(app.scopes || [], null, 2),
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  async function handleSave() {
    setSaving(true);
    setError("");
    try {
      const tools = JSON.parse(form.tools);
      const events = JSON.parse(form.events);
      const scopes = JSON.parse(form.scopes);
      await api.updateApp(app.id, {
        name: form.name,
        description: form.description,
        icon: form.icon,
        homepage: form.homepage,
        tools,
        events,
        scopes,
      });
      onSave();
    } catch (err) {
      setError(err.message || "JSON 格式错误");
    }
    setSaving(false);
  }
  return _jsxs("div", {
    className: "p-4 space-y-3",
    children: [
      _jsxs("div", {
        className: "flex items-center justify-between",
        children: [
          _jsx("h3", { className: "text-sm font-medium", children: "\u7F16\u8F91 App" }),
          _jsx(Button, {
            variant: "ghost",
            size: "icon-xs",
            onClick: onCancel,
            children: _jsx(X, { className: "w-4 h-4" }),
          }),
        ],
      }),
      _jsxs("div", {
        className: "space-y-2",
        children: [
          _jsx(Input, {
            placeholder: "\u540D\u79F0",
            value: form.name,
            onChange: (e) => setForm({ ...form, name: e.target.value }),
            className: "h-8 text-xs",
          }),
          _jsx(Input, {
            placeholder: "\u63CF\u8FF0",
            value: form.description,
            onChange: (e) => setForm({ ...form, description: e.target.value }),
            className: "h-8 text-xs",
          }),
          _jsxs("div", {
            className: "flex gap-2",
            children: [
              _jsx(Input, {
                placeholder: "\u56FE\u6807\uFF08emoji\uFF09",
                value: form.icon,
                onChange: (e) => setForm({ ...form, icon: e.target.value }),
                className: "h-8 text-xs w-24",
              }),
              _jsx(Input, {
                placeholder: "\u4E3B\u9875 URL",
                value: form.homepage,
                onChange: (e) => setForm({ ...form, homepage: e.target.value }),
                className: "h-8 text-xs flex-1",
              }),
            ],
          }),
        ],
      }),
      _jsxs("div", {
        className: "space-y-1",
        children: [
          _jsx("label", { className: "text-xs font-medium", children: "\u5DE5\u5177 (JSON)" }),
          _jsx("textarea", {
            value: form.tools,
            onChange: (e) => setForm({ ...form, tools: e.target.value }),
            rows: 6,
            className:
              "w-full rounded-md border border-input bg-transparent px-2 py-1 text-[11px] font-mono placeholder:text-muted-foreground/40 focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 resize-none",
          }),
        ],
      }),
      _jsxs("div", {
        className: "space-y-1",
        children: [
          _jsx("label", { className: "text-xs font-medium", children: "\u4E8B\u4EF6 (JSON)" }),
          _jsx("textarea", {
            value: form.events,
            onChange: (e) => setForm({ ...form, events: e.target.value }),
            rows: 2,
            className:
              "w-full rounded-md border border-input bg-transparent px-2 py-1 text-[11px] font-mono placeholder:text-muted-foreground/40 focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 resize-none",
          }),
        ],
      }),
      _jsxs("div", {
        className: "space-y-1",
        children: [
          _jsx("label", { className: "text-xs font-medium", children: "\u6743\u9650 (JSON)" }),
          _jsx("textarea", {
            value: form.scopes,
            onChange: (e) => setForm({ ...form, scopes: e.target.value }),
            rows: 2,
            className:
              "w-full rounded-md border border-input bg-transparent px-2 py-1 text-[11px] font-mono placeholder:text-muted-foreground/40 focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 resize-none",
          }),
        ],
      }),
      error ? _jsx("p", { className: "text-xs text-destructive", children: error }) : null,
      _jsxs("div", {
        className: "flex justify-end gap-2",
        children: [
          _jsx(Button, {
            variant: "outline",
            size: "sm",
            onClick: onCancel,
            children: "\u53D6\u6D88",
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
  });
}
