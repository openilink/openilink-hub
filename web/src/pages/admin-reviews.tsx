import { useEffect, useState, useCallback } from "react";
import { Check, X, Inbox, Globe, Terminal, Radio, Shield, History, ChevronDown } from "lucide-react";
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
import { Collapsible, CollapsibleTrigger, CollapsibleContent } from "@/components/ui/collapsible";
import { api } from "@/lib/api";
import { useToast } from "@/hooks/use-toast";
import { AppIcon } from "@/components/app-icon";
import { ListingBadge } from "@/components/listing-badge";

function timeAgo(ts: number) {
  if (!ts) return "—";
  const diff = Math.floor((Date.now() - ts * 1000) / 1000);
  if (diff < 0) return "刚刚";
  if (diff < 60) return `${diff}秒前`;
  if (diff < 3600) return `${Math.floor(diff / 60)}分钟前`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}小时前`;
  return `${Math.floor(diff / 86400)}天前`;
}

const ACTION_LABELS: Record<string, string> = {
  request: "申请上架",
  approve: "通过",
  reject: "拒绝",
  withdraw: "撤回",
  auto_revert: "自动撤回",
  admin_set: "管理员设置",
};

const ACTION_VARIANTS: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  request: "outline",
  approve: "default",
  reject: "destructive",
  withdraw: "secondary",
  auto_revert: "secondary",
  admin_set: "outline",
};

type TabFilter = "pending" | "listed" | "rejected" | "all";

const TABS: { key: TabFilter; label: string }[] = [
  { key: "pending", label: "待审核" },
  { key: "listed", label: "已通过" },
  { key: "rejected", label: "已拒绝" },
  { key: "all", label: "全部" },
];

export function AdminReviewsPage() {
  const [apps, setApps] = useState<any[]>([]);
  const [selected, setSelected] = useState<any>(null);
  const [reviews, setReviews] = useState<any[]>([]);
  const [rejectTarget, setRejectTarget] = useState<any>(null);
  const [rejectReason, setRejectReason] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<TabFilter>("pending");
  const { toast } = useToast();

  const loadReviews = useCallback((appId: string) => {
    api.listAppReviews(appId).then(setReviews).catch(() => setReviews([]));
  }, []);

  function loadApps() {
    setLoading(true);
    api.adminListApps()
      .then((data) => {
        setApps(data);
        setSelected((prev: any) =>
          prev ? (data.find((a: any) => a.id === prev.id) ?? null) : null
        );
      })
      .catch(() => {
        toast({ variant: "destructive", title: "加载失败", description: "无法获取应用列表，请刷新重试" });
      })
      .finally(() => setLoading(false));
  }

  useEffect(() => {
    loadApps();
  }, []);

  useEffect(() => {
    if (selected) {
      loadReviews(selected.id);
    } else {
      setReviews([]);
    }
  }, [selected?.id, loadReviews]);

  async function handleApprove(a: any) {
    setSubmitting(true);
    try {
      await api.reviewListing(a.id, true);
      toast({ title: `「${a.name}」已通过上架` });
      loadApps();
      loadReviews(a.id);
    } catch (e: any) {
      toast({ variant: "destructive", title: "操作失败", description: e.message });
    } finally {
      setSubmitting(false);
    }
  }

  async function handleRejectConfirm() {
    if (!rejectTarget || !rejectReason.trim()) return;
    const reason = rejectReason.trim();
    setSubmitting(true);
    try {
      await api.reviewListing(rejectTarget.id, false, reason);
      toast({ title: `「${rejectTarget.name}」已拒绝` });
      setRejectTarget(null);
      setRejectReason("");
      setSelected(null);
      loadApps();
    } catch (e: any) {
      toast({ variant: "destructive", title: "操作失败", description: e.message });
    } finally {
      setSubmitting(false);
    }
  }

  async function handleToggle(a: any) {
    const newListing = a.listing === "listed" ? "unlisted" : "listed";
    setSubmitting(true);
    try {
      await api.setAppListing(a.id, newListing);
      toast({ title: newListing === "listed" ? `「${a.name}」已上架` : `「${a.name}」已下架` });
      loadApps();
      loadReviews(a.id);
    } catch (e: any) {
      toast({ variant: "destructive", title: "操作失败", description: e.message });
    } finally {
      setSubmitting(false);
    }
  }

  // Only apps that have entered the review process
  const reviewed = apps.filter((a) => a.listing !== "unlisted");

  // Counts per tab
  const counts: Record<TabFilter, number> = {
    pending: reviewed.filter((a) => a.listing === "pending").length,
    listed: reviewed.filter((a) => a.listing === "listed").length,
    rejected: reviewed.filter((a) => a.listing === "rejected").length,
    all: reviewed.length,
  };

  // Filter by active tab
  const filtered = tab === "all" ? reviewed : reviewed.filter((a) => a.listing === tab);
  const sorted = [...filtered].sort((a, b) => (b.updated_at ?? 0) - (a.updated_at ?? 0));

  return (
    <div className="space-y-4">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight">审核中心</h1>
        <p className="text-sm text-muted-foreground mt-0.5">管理应用上架审核</p>
      </div>

      {/* Tabs */}
      <div role="tablist" className="inline-flex h-9 items-center rounded-lg bg-muted p-1 text-muted-foreground">
        {TABS.map(({ key, label }) => (
          <button
            key={key}
            role="tab"
            aria-selected={tab === key}
            onClick={() => { setTab(key); setSelected(null); }}
            className={`inline-flex items-center gap-1.5 rounded-md px-3 py-1 text-sm font-medium transition-all ${
              tab === key
                ? "bg-background text-foreground shadow-sm"
                : "hover:text-foreground/80"
            }`}
          >
            {label}
            {counts[key] > 0 && (
              <span className={`text-[10px] min-w-[1.25rem] text-center rounded-full px-1 py-px font-semibold ${
                tab === key
                  ? key === "pending" ? "bg-orange-500 text-white" : "bg-muted-foreground/20 text-foreground"
                  : key === "pending" ? "bg-orange-500/80 text-white" : "bg-muted-foreground/10"
              }`}>
                {counts[key]}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Main content */}
      <div className="flex flex-col md:flex-row gap-6 min-h-[calc(100vh-16rem)]">
        {/* Left: App Queue */}
        <div className="md:w-72 shrink-0 space-y-1 overflow-y-auto max-h-[50vh] md:max-h-none">
          {loading ? (
            <div className="space-y-1">
              {[1, 2, 3].map((i) => (
                <div key={i} className="flex items-center gap-3 p-3 rounded-lg">
                  <div className="h-9 w-9 rounded-lg bg-muted animate-pulse shrink-0" />
                  <div className="flex-1 space-y-1.5">
                    <div className="h-3.5 w-24 rounded bg-muted animate-pulse" />
                    <div className="h-2.5 w-16 rounded bg-muted animate-pulse" />
                  </div>
                </div>
              ))}
            </div>
          ) : sorted.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
              <Inbox className="h-10 w-10 mb-3 opacity-30" />
              <p className="text-sm">{tab === "pending" ? "无待审核应用" : "暂无记录"}</p>
            </div>
          ) : (
            sorted.map((a) => (
              <button
                key={a.id}
                onClick={() => setSelected(a)}
                aria-current={selected?.id === a.id ? "true" : undefined}
                className={`w-full flex items-center gap-3 p-3 rounded-lg text-left transition-colors ${
                  selected?.id === a.id
                    ? "bg-primary/10 border border-primary/20"
                    : "hover:bg-muted/50 border border-transparent"
                }`}
              >
                <AppIcon icon={a.icon} iconUrl={a.icon_url} size="h-9 w-9" />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-medium truncate">{a.name}</p>
                    {tab === "all" && <ListingBadge listing={a.listing} />}
                  </div>
                  <p className="text-xs text-muted-foreground truncate mt-0.5">
                    {a.version ? `v${a.version} · ` : ""}{timeAgo(a.updated_at)}
                  </p>
                </div>
              </button>
            ))
          )}
        </div>

        {/* Right: Detail Panel */}
        <div className="flex-1 min-w-0">
          {selected ? (
            <div className="rounded-xl border border-border/50 bg-card md:sticky md:top-6 md:max-h-[calc(100vh-12rem)] flex flex-col">
              {/* Scrollable content */}
              <div className="flex-1 overflow-y-auto p-6 space-y-5">
                {/* Header */}
                <div className="flex items-start gap-4">
                  <AppIcon icon={selected.icon} iconUrl={selected.icon_url} size="h-12 w-12" />
                  <div className="flex-1 min-w-0">
                    <h2 className="text-lg font-bold leading-tight">{selected.name}</h2>
                    <div className="flex items-center gap-2 mt-0.5">
                      <p className="text-xs text-muted-foreground font-mono">{selected.slug}</p>
                      {selected.version && (
                        <Badge variant="outline" className="text-[10px] font-mono">v{selected.version}</Badge>
                      )}
                    </div>
                  </div>
                  <ListingBadge listing={selected.listing} />
                </div>

                {/* Reject reason alert */}
                {selected.listing_reject_reason && (
                  <div className="rounded-lg bg-destructive/5 border border-destructive/20 p-3">
                    <p className="text-xs font-semibold text-destructive mb-1">拒绝原因</p>
                    <p className="text-sm">{selected.listing_reject_reason}</p>
                  </div>
                )}

                {/* Review Highlights — most important for reviewers */}
                {(() => {
                  const scopes = Array.isArray(selected.scopes) ? selected.scopes : [];
                  const tools = Array.isArray(selected.tools) ? selected.tools : [];
                  const events = Array.isArray(selected.events) ? selected.events : [];
                  return (
                    <div className="space-y-3">
                      <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">审核要点</p>

                      <ReviewField icon={<Shield className="h-3.5 w-3.5" />} label="权限">
                        {scopes.length > 0 ? (
                          <div className="flex flex-wrap gap-1">
                            {scopes.map((s: string, i: number) => (
                              <Badge key={i} variant="outline" className="text-[10px] font-mono">{s}</Badge>
                            ))}
                          </div>
                        ) : <span className="text-xs text-muted-foreground/50">无</span>}
                      </ReviewField>

                      <ReviewField icon={<Terminal className="h-3.5 w-3.5" />} label="工具">
                        {tools.length > 0 ? (
                          <div className="flex flex-wrap gap-1">
                            {tools.map((t: any, i: number) => (
                              <Badge key={i} variant="secondary" className="text-[10px] font-mono gap-1">
                                {t.command ? `/${t.command}` : t.name}
                              </Badge>
                            ))}
                          </div>
                        ) : <span className="text-xs text-muted-foreground/50">无</span>}
                      </ReviewField>

                      <ReviewField icon={<Radio className="h-3.5 w-3.5" />} label="事件">
                        {events.length > 0 ? (
                          <div className="flex flex-wrap gap-1">
                            {events.map((e: string, i: number) => (
                              <Badge key={i} variant="outline" className="text-[10px] font-mono">{e}</Badge>
                            ))}
                          </div>
                        ) : <span className="text-xs text-muted-foreground/50">无</span>}
                      </ReviewField>

                      <ReviewField icon={<Globe className="h-3.5 w-3.5" />} label="Webhook">
                        {selected.webhook_url ? (
                          <p className="font-mono text-xs truncate">{selected.webhook_url}</p>
                        ) : (
                          <span className="text-xs text-muted-foreground/50">未配置</span>
                        )}
                      </ReviewField>
                    </div>
                  );
                })()}

                {/* Review History */}
                {reviews.length > 0 && (
                  <>
                    <Separator />
                    <div>
                      <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-2 flex items-center gap-1.5">
                        <History className="h-3.5 w-3.5" /> 审核记录
                      </p>
                      <div className="space-y-2">
                        {reviews.map((review: any) => (
                          <div key={review.id} className="flex items-start gap-2 text-xs">
                            <span className="text-muted-foreground whitespace-nowrap mt-0.5 tabular-nums">
                              {new Date(review.created_at * 1000).toLocaleString("zh-CN", {
                                month: "2-digit", day: "2-digit",
                                hour: "2-digit", minute: "2-digit",
                              })}
                            </span>
                            <Badge variant={ACTION_VARIANTS[review.action] || "outline"} className="text-[10px] shrink-0">
                              {ACTION_LABELS[review.action] || review.action}
                            </Badge>
                            {review.version && (
                              <span className="text-muted-foreground font-mono">v{review.version}</span>
                            )}
                            {review.reason && (
                              <span className="text-muted-foreground truncate" title={review.reason}>{review.reason}</span>
                            )}
                          </div>
                        ))}
                      </div>
                    </div>
                  </>
                )}

                {/* Collapsible App Details */}
                <Separator />
                <Collapsible>
                  <CollapsibleTrigger className="flex items-center gap-1.5 text-xs font-semibold text-muted-foreground uppercase tracking-wide w-full group">
                    <ChevronDown className="h-3.5 w-3.5 transition-transform group-data-[state=open]:rotate-180" />
                    应用详情
                  </CollapsibleTrigger>
                  <CollapsibleContent className="mt-3 space-y-4">
                    {/* Basic info */}
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-2 text-sm">
                      <div>
                        <p className="text-xs text-muted-foreground">开发者</p>
                        <p className="font-medium">{selected.owner_name}</p>
                      </div>
                      <div>
                        <p className="text-xs text-muted-foreground">更新时间</p>
                        <p>{timeAgo(selected.updated_at)}</p>
                      </div>
                      {selected.homepage && (
                        <div className="col-span-2">
                          <p className="text-xs text-muted-foreground">主页</p>
                          <a
                            href={selected.homepage}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-primary hover:underline text-sm truncate block"
                          >
                            {selected.homepage.replace(/^https?:\/\//, "")}
                          </a>
                        </div>
                      )}
                    </div>

                    {selected.description && (
                      <div>
                        <p className="text-xs text-muted-foreground mb-1">描述</p>
                        <p className="text-sm leading-relaxed">{selected.description}</p>
                      </div>
                    )}

                    {selected.config_schema && (
                      <div>
                        <p className="text-xs text-muted-foreground mb-1">Config Schema</p>
                        <pre className="text-[10px] font-mono bg-muted/40 rounded p-2 max-h-32 overflow-auto whitespace-pre-wrap">
                          {typeof selected.config_schema === "string" ? selected.config_schema : JSON.stringify(selected.config_schema, null, 2)}
                        </pre>
                      </div>
                    )}

                    {selected.readme && (
                      <div>
                        <p className="text-xs text-muted-foreground mb-1">README</p>
                        <pre className="text-xs text-muted-foreground whitespace-pre-wrap font-mono bg-muted/40 rounded-lg p-3 max-h-60 overflow-y-auto">
                          {selected.readme}
                        </pre>
                      </div>
                    )}
                  </CollapsibleContent>
                </Collapsible>
              </div>

              {/* Sticky action buttons */}
              <div className="border-t p-4 shrink-0 bg-card rounded-b-xl">
                {selected.listing === "pending" ? (
                  <div className="flex gap-3">
                    <Button
                      className="flex-1 gap-1.5 bg-emerald-600 hover:bg-emerald-700"
                      onClick={() => handleApprove(selected)}
                      disabled={submitting}
                    >
                      <Check className="h-4 w-4" /> 通过上架
                    </Button>
                    <Button
                      variant="outline"
                      className="flex-1 gap-1.5 text-destructive border-destructive/30 hover:bg-destructive/5"
                      onClick={() => { setRejectTarget(selected); setRejectReason(""); }}
                      disabled={submitting}
                    >
                      <X className="h-4 w-4" /> 拒绝
                    </Button>
                  </div>
                ) : selected.listing === "listed" ? (
                  <Button
                    variant="outline"
                    className="w-full"
                    onClick={() => handleToggle(selected)}
                    disabled={submitting}
                  >
                    下架
                  </Button>
                ) : (
                  <Button
                    variant="outline"
                    className="w-full"
                    onClick={() => handleApprove(selected)}
                    disabled={submitting}
                  >
                    通过上架
                  </Button>
                )}
              </div>
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
              <Inbox className="h-12 w-12 mb-3 opacity-20" />
              <p className="text-sm">选择一个应用查看详情</p>
            </div>
          )}
        </div>
      </div>

      {/* Reject Dialog */}
      <Dialog
        open={!!rejectTarget}
        onOpenChange={(o) => { if (!o) { setRejectTarget(null); setRejectReason(""); } }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>拒绝「{rejectTarget?.name}」</DialogTitle>
            <DialogDescription>填写拒绝原因，开发者将收到此通知。</DialogDescription>
          </DialogHeader>
          <div className="space-y-3 py-2">
            <div className="space-y-1.5">
              <Label htmlFor="reject-reason">拒绝原因</Label>
              <Textarea
                id="reject-reason"
                placeholder="请说明拒绝原因，开发者将收到此消息…"
                rows={4}
                value={rejectReason}
                onChange={(e) => setRejectReason(e.target.value)}
                autoFocus
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setRejectTarget(null); setRejectReason(""); }}>
              取消
            </Button>
            <Button
              variant="destructive"
              onClick={handleRejectConfirm}
              disabled={!rejectReason.trim() || submitting}
            >
              确认拒绝
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function ReviewField({ icon, label, children }: { icon: React.ReactNode; label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-start gap-2 text-sm">
      <span className="mt-0.5 text-muted-foreground shrink-0">{icon}</span>
      <div className="min-w-0 flex-1">
        <p className="text-xs text-muted-foreground mb-1">{label}</p>
        {children}
      </div>
    </div>
  );
}
