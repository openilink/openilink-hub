import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { useState, useRef } from "react";
import { Check, X, Inbox, Globe, Terminal, Radio, Shield, History } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { useToast } from "@/hooks/use-toast";
import { AppIcon } from "@/components/app-icon";
import { ListingBadge } from "@/components/listing-badge";
import {
  useAdminApps,
  useSetAppListing,
  useReviewListing,
  useAppReviewHistory,
} from "@/hooks/use-admin";
function timeAgo(ts) {
  if (!ts) return "—";
  const diff = Math.floor((Date.now() - ts * 1000) / 1000);
  if (diff < 0) return "刚刚";
  if (diff < 60) return `${diff}秒前`;
  if (diff < 3600) return `${Math.floor(diff / 60)}分钟前`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}小时前`;
  return `${Math.floor(diff / 86400)}天前`;
}
const ACTION_LABELS = {
  request: "申请上架",
  approve: "通过",
  reject: "拒绝",
  withdraw: "撤回",
  auto_revert: "自动撤回",
  admin_set: "管理员设置",
};
const ACTION_VARIANTS = {
  request: "outline",
  approve: "default",
  reject: "destructive",
  withdraw: "secondary",
  auto_revert: "secondary",
  admin_set: "outline",
};
const TABS = [
  { key: "pending", label: "待审核" },
  { key: "listed", label: "已通过" },
  { key: "rejected", label: "已拒绝" },
  { key: "all", label: "全部" },
];
export function AdminReviewsPage() {
  const { data: apps = [], isLoading: loading } = useAdminApps();
  const [selectedId, setSelectedId] = useState(null);
  const selected = selectedId ? (apps.find((a) => a.id === selectedId) ?? null) : null;
  const [rejectTarget, setRejectTarget] = useState(null);
  const [rejectReason, setRejectReason] = useState("");
  const [tab, setTab] = useState("pending");
  const tabListRef = useRef(null);
  const { toast } = useToast();
  const { data: reviews = [] } = useAppReviewHistory(selected?.id);
  const reviewMutation = useReviewListing();
  const setListingMutation = useSetAppListing();
  const submitting = reviewMutation.isPending || setListingMutation.isPending;
  async function handleApprove(a) {
    try {
      await reviewMutation.mutateAsync({ appId: a.id, approve: true });
      toast({ title: `「${a.name}」已通过上架` });
    } catch (e) {
      toast({ variant: "destructive", title: "操作失败", description: e.message });
    }
  }
  async function handleRejectConfirm() {
    if (!rejectTarget || !rejectReason.trim()) return;
    const reason = rejectReason.trim();
    try {
      await reviewMutation.mutateAsync({ appId: rejectTarget.id, approve: false, reason });
      toast({ title: `「${rejectTarget.name}」已拒绝` });
      setRejectTarget(null);
      setRejectReason("");
      setSelectedId(null);
    } catch (e) {
      toast({ variant: "destructive", title: "操作失败", description: e.message });
    }
  }
  async function handleToggle(a) {
    const newListing = a.listing === "listed" ? "unlisted" : "listed";
    try {
      await setListingMutation.mutateAsync({ id: a.id, listing: newListing });
      toast({ title: newListing === "listed" ? `「${a.name}」已上架` : `「${a.name}」已下架` });
    } catch (e) {
      toast({ variant: "destructive", title: "操作失败", description: e.message });
    }
  }
  // Only apps that have entered the review process
  const reviewed = apps.filter((a) => a.listing !== "unlisted");
  // Counts per tab
  const counts = {
    pending: reviewed.filter((a) => a.listing === "pending").length,
    listed: reviewed.filter((a) => a.listing === "listed").length,
    rejected: reviewed.filter((a) => a.listing === "rejected").length,
    all: reviewed.length,
  };
  // Filter by active tab
  const filtered = tab === "all" ? reviewed : reviewed.filter((a) => a.listing === tab);
  const sorted = [...filtered].sort((a, b) => (b.updated_at ?? 0) - (a.updated_at ?? 0));
  // Computed arrays for review highlights (avoids IIFE in JSX)
  const selScopes = Array.isArray(selected?.scopes) ? selected.scopes : [];
  const selTools = Array.isArray(selected?.tools) ? selected.tools : [];
  const selEvents = Array.isArray(selected?.events) ? selected.events : [];
  return _jsxs("div", {
    className: "space-y-4",
    children: [
      _jsxs("div", {
        children: [
          _jsx("h1", {
            className: "text-2xl font-bold tracking-tight",
            children: "\u5BA1\u6838\u4E2D\u5FC3",
          }),
          _jsx("p", {
            className: "text-sm text-muted-foreground mt-0.5",
            children: "\u7BA1\u7406\u5E94\u7528\u4E0A\u67B6\u5BA1\u6838",
          }),
        ],
      }),
      _jsx("div", {
        role: "tablist",
        ref: tabListRef,
        className: "inline-flex h-9 items-center rounded-lg bg-muted p-1 text-muted-foreground",
        onKeyDown: (e) => {
          const keys = TABS.map((t) => t.key);
          const idx = keys.indexOf(tab);
          let next = -1;
          if (e.key === "ArrowRight") next = (idx + 1) % keys.length;
          else if (e.key === "ArrowLeft") next = (idx - 1 + keys.length) % keys.length;
          if (next >= 0) {
            e.preventDefault();
            setTab(keys[next]);
            setSelectedId(null);
            tabListRef.current?.querySelectorAll("[role=tab]")[next]?.focus();
          }
        },
        children: TABS.map(({ key, label }) =>
          _jsxs(
            "button",
            {
              role: "tab",
              id: `tab-${key}`,
              "aria-selected": tab === key,
              "aria-controls": "review-tabpanel",
              tabIndex: tab === key ? 0 : -1,
              onClick: () => {
                setTab(key);
                setSelectedId(null);
              },
              className: `inline-flex items-center gap-1.5 rounded-md px-3 py-1 text-sm font-medium transition-all ${
                tab === key ? "bg-background text-foreground shadow-sm" : "hover:text-foreground/80"
              }`,
              children: [
                label,
                counts[key] > 0 &&
                  _jsx("span", {
                    className: `text-[10px] min-w-[1.25rem] text-center rounded-full px-1 py-px font-semibold ${
                      tab === key
                        ? key === "pending"
                          ? "bg-orange-500 text-white"
                          : "bg-muted-foreground/20 text-foreground"
                        : key === "pending"
                          ? "bg-orange-500/80 text-white"
                          : "bg-muted-foreground/10"
                    }`,
                    children: counts[key],
                  }),
              ],
            },
            key,
          ),
        ),
      }),
      _jsxs("div", {
        id: "review-tabpanel",
        role: "tabpanel",
        "aria-labelledby": `tab-${tab}`,
        className: "flex flex-col md:flex-row gap-4",
        children: [
          _jsx("div", {
            className:
              "md:w-64 shrink-0 space-y-0.5 overflow-y-auto max-h-[50vh] md:max-h-[calc(100vh-14rem)]",
            children: loading
              ? _jsx("div", {
                  className: "space-y-1",
                  children: [1, 2, 3].map((i) =>
                    _jsxs(
                      "div",
                      {
                        className: "flex items-center gap-3 p-3 rounded-lg",
                        children: [
                          _jsx("div", {
                            className: "h-9 w-9 rounded-lg bg-muted animate-pulse shrink-0",
                          }),
                          _jsxs("div", {
                            className: "flex-1 space-y-1.5",
                            children: [
                              _jsx("div", {
                                className: "h-3.5 w-24 rounded bg-muted animate-pulse",
                              }),
                              _jsx("div", {
                                className: "h-2.5 w-16 rounded bg-muted animate-pulse",
                              }),
                            ],
                          }),
                        ],
                      },
                      i,
                    ),
                  ),
                })
              : sorted.length === 0
                ? _jsxs("div", {
                    className:
                      "flex flex-col items-center justify-center py-12 text-muted-foreground",
                    children: [
                      _jsx(Inbox, { className: "h-8 w-8 mb-2 opacity-30" }),
                      _jsx("p", {
                        className: "text-sm",
                        children: tab === "pending" ? "无待审核应用" : "暂无记录",
                      }),
                    ],
                  })
                : sorted.map((a) =>
                    _jsxs(
                      "button",
                      {
                        onClick: () => setSelectedId(a.id),
                        "aria-current": selected?.id === a.id ? "true" : undefined,
                        className: `w-full flex items-center gap-3 p-3 rounded-lg text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${
                          selected?.id === a.id
                            ? "bg-primary/10 border border-primary/20"
                            : "hover:bg-muted/50 border border-transparent"
                        }`,
                        children: [
                          _jsx(AppIcon, { icon: a.icon, iconUrl: a.icon_url, size: "h-9 w-9" }),
                          _jsxs("div", {
                            className: "flex-1 min-w-0",
                            children: [
                              _jsxs("div", {
                                className: "flex items-center gap-2",
                                children: [
                                  _jsx("p", {
                                    className: "text-sm font-medium truncate",
                                    children: a.name,
                                  }),
                                  tab === "all" && _jsx(ListingBadge, { listing: a.listing }),
                                ],
                              }),
                              _jsxs("p", {
                                className: "text-xs text-muted-foreground truncate mt-0.5",
                                children: [
                                  a.version ? `v${a.version} · ` : "",
                                  timeAgo(a.updated_at),
                                ],
                              }),
                            ],
                          }),
                        ],
                      },
                      a.id,
                    ),
                  ),
          }),
          _jsx("div", {
            className: "flex-1 min-w-0",
            children: selected
              ? _jsxs("div", {
                  className:
                    "rounded-xl border border-border/50 bg-card md:sticky md:top-6 md:max-h-[calc(100vh-14rem)] flex flex-col",
                  children: [
                    _jsxs("div", {
                      className: "flex-1 overflow-y-auto p-5 space-y-4",
                      children: [
                        _jsxs("div", {
                          className: "flex items-start gap-3",
                          children: [
                            _jsx(AppIcon, {
                              icon: selected.icon,
                              iconUrl: selected.icon_url,
                              size: "h-10 w-10",
                            }),
                            _jsxs("div", {
                              className: "flex-1 min-w-0",
                              children: [
                                _jsx("h2", {
                                  className: "text-base font-bold leading-tight",
                                  children: selected.name,
                                }),
                                _jsxs("div", {
                                  className: "flex items-center gap-2 mt-0.5",
                                  children: [
                                    _jsx("p", {
                                      className: "text-xs text-muted-foreground font-mono",
                                      children: selected.slug,
                                    }),
                                    selected.version &&
                                      _jsxs(Badge, {
                                        variant: "outline",
                                        className: "text-[10px] font-mono",
                                        children: ["v", selected.version],
                                      }),
                                  ],
                                }),
                              ],
                            }),
                            _jsx(ListingBadge, { listing: selected.listing }),
                          ],
                        }),
                        selected.listing_reject_reason &&
                          _jsxs("div", {
                            className:
                              "rounded-lg bg-destructive/5 border border-destructive/20 p-3",
                            children: [
                              _jsx("p", {
                                className: "text-xs font-semibold text-destructive mb-1",
                                children: "\u62D2\u7EDD\u539F\u56E0",
                              }),
                              _jsx("p", {
                                className: "text-sm",
                                children: selected.listing_reject_reason,
                              }),
                            ],
                          }),
                        _jsxs("div", {
                          className: "space-y-3",
                          children: [
                            _jsx("p", {
                              className:
                                "text-xs font-semibold text-muted-foreground uppercase tracking-wide",
                              children: "\u5BA1\u6838\u8981\u70B9",
                            }),
                            _jsx(ReviewField, {
                              icon: _jsx(Shield, { className: "h-3.5 w-3.5" }),
                              label: "\u6743\u9650",
                              children:
                                selScopes.length > 0
                                  ? _jsx("div", {
                                      className: "flex flex-wrap gap-1",
                                      children: selScopes.map((s, i) =>
                                        _jsx(
                                          Badge,
                                          {
                                            variant: "outline",
                                            className: "text-[10px] font-mono",
                                            children: s,
                                          },
                                          i,
                                        ),
                                      ),
                                    })
                                  : _jsx("span", {
                                      className: "text-xs text-muted-foreground/50",
                                      children: "\u65E0",
                                    }),
                            }),
                            _jsx(ReviewField, {
                              icon: _jsx(Terminal, { className: "h-3.5 w-3.5" }),
                              label: "\u5DE5\u5177",
                              children:
                                selTools.length > 0
                                  ? _jsx("div", {
                                      className: "flex flex-wrap gap-1",
                                      children: selTools.map((t, i) =>
                                        _jsx(
                                          Badge,
                                          {
                                            variant: "secondary",
                                            className: "text-[10px] font-mono gap-1",
                                            children: t.command ? `/${t.command}` : t.name,
                                          },
                                          i,
                                        ),
                                      ),
                                    })
                                  : _jsx("span", {
                                      className: "text-xs text-muted-foreground/50",
                                      children: "\u65E0",
                                    }),
                            }),
                            _jsx(ReviewField, {
                              icon: _jsx(Radio, { className: "h-3.5 w-3.5" }),
                              label: "\u4E8B\u4EF6",
                              children:
                                selEvents.length > 0
                                  ? _jsx("div", {
                                      className: "flex flex-wrap gap-1",
                                      children: selEvents.map((e, i) =>
                                        _jsx(
                                          Badge,
                                          {
                                            variant: "outline",
                                            className: "text-[10px] font-mono",
                                            children: e,
                                          },
                                          i,
                                        ),
                                      ),
                                    })
                                  : _jsx("span", {
                                      className: "text-xs text-muted-foreground/50",
                                      children: "\u65E0",
                                    }),
                            }),
                            _jsx(ReviewField, {
                              icon: _jsx(Globe, { className: "h-3.5 w-3.5" }),
                              label: "Webhook",
                              children: selected.webhook_url
                                ? _jsx("p", {
                                    className: "font-mono text-xs truncate",
                                    children: selected.webhook_url,
                                  })
                                : _jsx("span", {
                                    className: "text-xs text-muted-foreground/50",
                                    children: "\u672A\u914D\u7F6E",
                                  }),
                            }),
                          ],
                        }),
                        reviews.length > 0 &&
                          _jsxs(_Fragment, {
                            children: [
                              _jsx(Separator, {}),
                              _jsxs("div", {
                                children: [
                                  _jsxs("p", {
                                    className:
                                      "text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-2 flex items-center gap-1.5",
                                    children: [
                                      _jsx(History, { className: "h-3.5 w-3.5" }),
                                      " \u5BA1\u6838\u8BB0\u5F55",
                                    ],
                                  }),
                                  _jsx("div", {
                                    className: "space-y-2",
                                    children: reviews.map((review) =>
                                      _jsxs(
                                        "div",
                                        {
                                          className: "flex items-start gap-2 text-xs",
                                          children: [
                                            _jsx("span", {
                                              className:
                                                "text-muted-foreground whitespace-nowrap mt-0.5 tabular-nums",
                                              children: new Date(
                                                review.created_at * 1000,
                                              ).toLocaleString("zh-CN", {
                                                month: "2-digit",
                                                day: "2-digit",
                                                hour: "2-digit",
                                                minute: "2-digit",
                                              }),
                                            }),
                                            _jsx(Badge, {
                                              variant: ACTION_VARIANTS[review.action] || "outline",
                                              className: "text-[10px] shrink-0",
                                              children:
                                                ACTION_LABELS[review.action] || review.action,
                                            }),
                                            review.version &&
                                              _jsxs("span", {
                                                className: "text-muted-foreground font-mono",
                                                children: ["v", review.version],
                                              }),
                                            review.reason &&
                                              _jsx("span", {
                                                className: "text-muted-foreground truncate",
                                                title: review.reason,
                                                children: review.reason,
                                              }),
                                          ],
                                        },
                                        review.id,
                                      ),
                                    ),
                                  }),
                                ],
                              }),
                            ],
                          }),
                        _jsx(Separator, {}),
                        _jsxs("div", {
                          className: "space-y-4",
                          children: [
                            _jsx("p", {
                              className:
                                "text-xs font-semibold text-muted-foreground uppercase tracking-wide",
                              children: "\u5E94\u7528\u8BE6\u60C5",
                            }),
                            _jsxs("div", {
                              className: "grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-2 text-sm",
                              children: [
                                _jsxs("div", {
                                  children: [
                                    _jsx("p", {
                                      className: "text-xs text-muted-foreground",
                                      children: "\u5F00\u53D1\u8005",
                                    }),
                                    _jsx("p", {
                                      className: "font-medium",
                                      children: selected.owner_name,
                                    }),
                                  ],
                                }),
                                _jsxs("div", {
                                  children: [
                                    _jsx("p", {
                                      className: "text-xs text-muted-foreground",
                                      children: "\u66F4\u65B0\u65F6\u95F4",
                                    }),
                                    _jsx("p", { children: timeAgo(selected.updated_at) }),
                                  ],
                                }),
                                selected.homepage &&
                                  _jsxs("div", {
                                    className: "sm:col-span-2",
                                    children: [
                                      _jsx("p", {
                                        className: "text-xs text-muted-foreground",
                                        children: "\u4E3B\u9875",
                                      }),
                                      _jsx("a", {
                                        href: selected.homepage,
                                        target: "_blank",
                                        rel: "noopener noreferrer",
                                        className:
                                          "text-primary hover:underline text-sm truncate block",
                                        children: selected.homepage.replace(/^https?:\/\//, ""),
                                      }),
                                    ],
                                  }),
                              ],
                            }),
                            selected.description &&
                              _jsxs("div", {
                                children: [
                                  _jsx("p", {
                                    className: "text-xs text-muted-foreground mb-1",
                                    children: "\u63CF\u8FF0",
                                  }),
                                  _jsx("p", {
                                    className: "text-sm leading-relaxed",
                                    children: selected.description,
                                  }),
                                ],
                              }),
                            selected.config_schema &&
                              _jsxs("div", {
                                children: [
                                  _jsx("p", {
                                    className: "text-xs text-muted-foreground mb-1",
                                    children: "Config Schema",
                                  }),
                                  _jsx("pre", {
                                    className:
                                      "text-[10px] font-mono bg-muted/40 rounded p-2 max-h-32 overflow-auto whitespace-pre-wrap",
                                    children:
                                      typeof selected.config_schema === "string"
                                        ? selected.config_schema
                                        : JSON.stringify(selected.config_schema, null, 2),
                                  }),
                                ],
                              }),
                            selected.readme &&
                              _jsxs("div", {
                                children: [
                                  _jsx("p", {
                                    className: "text-xs text-muted-foreground mb-1",
                                    children: "README",
                                  }),
                                  _jsx("pre", {
                                    className:
                                      "text-xs text-muted-foreground whitespace-pre-wrap font-mono bg-muted/40 rounded-lg p-3 max-h-60 overflow-y-auto",
                                    children: selected.readme,
                                  }),
                                ],
                              }),
                          ],
                        }),
                      ],
                    }),
                    _jsx("div", {
                      className: "border-t px-5 py-3 shrink-0 bg-card rounded-b-xl",
                      children:
                        selected.listing === "pending"
                          ? _jsxs("div", {
                              className: "flex gap-3",
                              children: [
                                _jsxs(Button, {
                                  className: "flex-1 gap-1.5 bg-emerald-600 hover:bg-emerald-700",
                                  onClick: () => handleApprove(selected),
                                  disabled: submitting,
                                  children: [
                                    _jsx(Check, { className: "h-4 w-4" }),
                                    " \u901A\u8FC7\u4E0A\u67B6",
                                  ],
                                }),
                                _jsxs(Button, {
                                  variant: "outline",
                                  className:
                                    "flex-1 gap-1.5 text-destructive border-destructive/30 hover:bg-destructive/5",
                                  onClick: () => {
                                    setRejectTarget(selected);
                                    setRejectReason("");
                                  },
                                  disabled: submitting,
                                  children: [_jsx(X, { className: "h-4 w-4" }), " \u62D2\u7EDD"],
                                }),
                              ],
                            })
                          : selected.listing === "listed"
                            ? _jsx(Button, {
                                variant: "outline",
                                className: "w-full",
                                onClick: () => handleToggle(selected),
                                disabled: submitting,
                                children: "\u4E0B\u67B6",
                              })
                            : _jsx(Button, {
                                variant: "outline",
                                className: "w-full",
                                onClick: () => handleApprove(selected),
                                disabled: submitting,
                                children: "\u901A\u8FC7\u4E0A\u67B6",
                              }),
                    }),
                  ],
                })
              : _jsxs("div", {
                  className:
                    "flex flex-col items-center justify-center py-20 text-muted-foreground",
                  children: [
                    _jsx(Inbox, { className: "h-10 w-10 mb-3 opacity-20" }),
                    _jsx("p", {
                      className: "text-sm",
                      children: "\u9009\u62E9\u4E00\u4E2A\u5E94\u7528\u67E5\u770B\u8BE6\u60C5",
                    }),
                  ],
                }),
          }),
        ],
      }),
      _jsx(Dialog, {
        open: !!rejectTarget,
        onOpenChange: (o) => {
          if (!o) {
            setRejectTarget(null);
            setRejectReason("");
          }
        },
        children: _jsxs(DialogContent, {
          className: "sm:max-w-md",
          children: [
            _jsxs(DialogHeader, {
              children: [
                _jsxs(DialogTitle, {
                  children: ["\u62D2\u7EDD\u300C", rejectTarget?.name, "\u300D"],
                }),
                _jsx(DialogDescription, {
                  children:
                    "\u586B\u5199\u62D2\u7EDD\u539F\u56E0\uFF0C\u5F00\u53D1\u8005\u5C06\u6536\u5230\u6B64\u901A\u77E5\u3002",
                }),
              ],
            }),
            _jsx("div", {
              className: "space-y-3 py-2",
              children: _jsxs("div", {
                className: "space-y-1.5",
                children: [
                  _jsx(Label, { htmlFor: "reject-reason", children: "\u62D2\u7EDD\u539F\u56E0" }),
                  _jsx(Textarea, {
                    id: "reject-reason",
                    placeholder:
                      "\u8BF7\u8BF4\u660E\u62D2\u7EDD\u539F\u56E0\uFF0C\u5F00\u53D1\u8005\u5C06\u6536\u5230\u6B64\u6D88\u606F\u2026",
                    rows: 4,
                    value: rejectReason,
                    onChange: (e) => setRejectReason(e.target.value),
                    autoFocus: true,
                  }),
                ],
              }),
            }),
            _jsxs(DialogFooter, {
              children: [
                _jsx(Button, {
                  variant: "outline",
                  onClick: () => {
                    setRejectTarget(null);
                    setRejectReason("");
                  },
                  children: "\u53D6\u6D88",
                }),
                _jsx(Button, {
                  variant: "destructive",
                  onClick: handleRejectConfirm,
                  disabled: !rejectReason.trim() || submitting,
                  children: "\u786E\u8BA4\u62D2\u7EDD",
                }),
              ],
            }),
          ],
        }),
      }),
    ],
  });
}
function ReviewField({ icon, label, children }) {
  return _jsxs("div", {
    className: "flex items-start gap-2 text-sm",
    children: [
      _jsx("span", { className: "mt-0.5 text-muted-foreground shrink-0", children: icon }),
      _jsxs("div", {
        className: "min-w-0 flex-1",
        children: [
          _jsx("p", { className: "text-xs text-muted-foreground mb-1", children: label }),
          children,
        ],
      }),
    ],
  });
}
