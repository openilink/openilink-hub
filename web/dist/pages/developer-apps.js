import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Blocks, Loader2, Plus } from "lucide-react";
import { useApps, useCreateApp } from "@/hooks/use-apps";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useToast } from "@/hooks/use-toast";
import { AppIcon } from "@/components/app-icon";
import { ListingBadge } from "@/components/listing-badge";
// ==================== Page ====================
export function DeveloperAppsPage() {
  const navigate = useNavigate();
  const { toast } = useToast();
  const { data: apps = [], isLoading: loading } = useApps();
  const createAppMutation = useCreateApp();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [newName, setNewName] = useState("");
  function openDialog() {
    setNewName("");
    setDialogOpen(true);
  }
  function handleDialogClose(open) {
    setDialogOpen(open);
    if (!open) setNewName("");
  }
  const creating = createAppMutation.isPending;
  function handleCreate() {
    const name = newName.trim();
    if (!name) return;
    createAppMutation.mutate(
      { name },
      {
        onSuccess: (app) => {
          setDialogOpen(false);
          setNewName("");
          navigate(`/dashboard/developer/apps/${app.id}`);
        },
        onError: (e) =>
          toast({ variant: "destructive", title: "创建失败", description: e.message }),
      },
    );
  }
  function handleKeyDown(e) {
    if (creating) return;
    if (e.key === "Enter") {
      handleCreate();
    }
  }
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
                children: "\u6211\u7684\u5E94\u7528",
              }),
              _jsx("p", {
                className: "text-sm text-muted-foreground mt-0.5",
                children: "\u7BA1\u7406\u4F60\u521B\u5EFA\u7684\u5E94\u7528\u3002",
              }),
            ],
          }),
          _jsxs(Button, {
            onClick: openDialog,
            className: "gap-1.5 shrink-0",
            children: [_jsx(Plus, { className: "h-4 w-4" }), "\u65B0\u5EFA\u5E94\u7528"],
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
                    _jsx("div", { className: "h-5 w-14 rounded-md bg-muted shrink-0" }),
                  ],
                },
                i,
              ),
            ),
          })
        : apps.length === 0
          ? _jsxs("div", {
              className: "text-center py-16 space-y-3 border-2 border-dashed rounded-xl",
              children: [
                _jsx(Blocks, { className: "w-8 h-8 mx-auto text-muted-foreground/40" }),
                _jsx("p", {
                  className: "text-sm text-muted-foreground",
                  children: "\u8FD8\u6CA1\u6709\u521B\u5EFA\u8FC7\u5E94\u7528",
                }),
                _jsxs(Button, {
                  variant: "outline",
                  onClick: openDialog,
                  className: "gap-1.5",
                  children: [
                    _jsx(Plus, { className: "h-4 w-4" }),
                    "\u65B0\u5EFA\u7B2C\u4E00\u4E2A\u5E94\u7528",
                  ],
                }),
              ],
            })
          : _jsx("div", {
              className:
                "divide-y divide-border/50 rounded-xl border border-border/50 overflow-hidden",
              children: apps.map((app) =>
                _jsxs(
                  "div",
                  {
                    role: "button",
                    tabIndex: 0,
                    className:
                      "group flex items-center gap-4 px-4 py-3.5 bg-card hover:bg-muted/40 transition-colors cursor-pointer",
                    onClick: () => navigate(`/dashboard/developer/apps/${app.id}`),
                    onKeyDown: (e) => {
                      if (e.key === "Enter" || e.key === " ") {
                        e.preventDefault();
                        navigate(`/dashboard/developer/apps/${app.id}`);
                      }
                    },
                    children: [
                      _jsx(AppIcon, { icon: app.icon, iconUrl: app.icon_url, size: "h-9 w-9" }),
                      _jsxs("div", {
                        className: "flex-1 min-w-0",
                        children: [
                          _jsx("p", {
                            className: "text-sm font-semibold leading-tight",
                            children: app.name,
                          }),
                          app.slug
                            ? _jsx("p", {
                                className: "font-mono text-xs text-muted-foreground mt-0.5",
                                children: app.slug,
                              })
                            : null,
                        ],
                      }),
                      _jsx(ListingBadge, { listing: app.listing }),
                    ],
                  },
                  app.id,
                ),
              ),
            }),
      _jsx(Dialog, {
        open: dialogOpen,
        onOpenChange: handleDialogClose,
        children: _jsxs(DialogContent, {
          className: "sm:max-w-md",
          children: [
            _jsxs(DialogHeader, {
              children: [
                _jsx(DialogTitle, { children: "\u65B0\u5EFA\u5E94\u7528" }),
                _jsx(DialogDescription, {
                  children:
                    "\u8F93\u5165\u5E94\u7528\u540D\u79F0\u4EE5\u521B\u5EFA\u4E00\u4E2A\u65B0\u5E94\u7528\u3002",
                }),
              ],
            }),
            _jsxs("div", {
              className: "space-y-4 pt-2",
              children: [
                _jsx(Input, {
                  placeholder: "\u5E94\u7528\u540D\u79F0",
                  "aria-label": "\u5E94\u7528\u540D\u79F0",
                  value: newName,
                  onChange: (e) => setNewName(e.target.value),
                  onKeyDown: handleKeyDown,
                  disabled: creating,
                  autoFocus: true,
                }),
                _jsxs(Button, {
                  className: "w-full",
                  disabled: creating || !newName.trim(),
                  onClick: handleCreate,
                  children: [
                    creating ? _jsx(Loader2, { className: "h-4 w-4 animate-spin mr-2" }) : null,
                    "\u521B\u5EFA",
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
