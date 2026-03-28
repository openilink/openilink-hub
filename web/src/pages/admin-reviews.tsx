import { useEffect, useState } from "react";
import { Check, X, ExternalLink, Inbox, Globe, Terminal, Radio, Shield } from "lucide-react";
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

export function AdminReviewsPage() {
  const [apps, setApps] = useState<any[]>([]);
  const [selected, setSelected] = useState<any>(null);
  const [rejectTarget, setRejectTarget] = useState<any>(null);
  const [rejectReason, setRejectReason] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [loading, setLoading] = useState(true);
  const { toast } = useToast();

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

  async function handleApprove(a: any) {
    setSubmitting(true);
    try {
      await api.reviewListing(a.id, true);
      toast({ title: `「${a.name}」已通过上架` });
      loadApps();
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
    } catch (e: any) {
      toast({ variant: "destructive", title: "操作失败", description: e.message });
    } finally {
      setSubmitting(false);
    }
  }

  // Pending apps first, then listed, then unlisted
  const sorted = [...apps].sort((a, b) => {
    const order: Record<string, number> = { pending: 0, listed: 1, unlisted: 2 };
    return (order[a.listing] ?? 3) - (order[b.listing] ?? 3)
      || (b.updated_at ?? 0) - (a.updated_at ?? 0);
  });

  const pendingCount = apps.filter((a) => a.listing === "pending").length;

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">审核中心</h1>
          <p className="text-sm text-muted-foreground mt-0.5">
            审核应用上架请求
            {pendingCount > 0 && (
              <Badge variant="destructive" className="ml-2 text-[10px] px-1.5 py-0">
                {pendingCount} 待审核
              </Badge>
            )}
          </p>
        </div>
      </div>

      <div className="flex flex-col md:flex-row gap-6 min-h-[calc(100vh-14rem)]">
        {/* Left: App List */}
        <div className="md:w-80 shrink-0 space-y-1 overflow-y-auto">
          {loading ? (
            <div className="space-y-1">
              {[1, 2, 3, 4].map((i) => (
                <div key={i} className="flex items-center gap-3 p-3 rounded-lg">
                  <div className="h-9 w-9 rounded-lg bg-muted animate-pulse shrink-0" />
                  <div className="flex-1 space-y-1.5">
                    <div className="h-3.5 w-28 rounded bg-muted animate-pulse" />
                    <div className="h-2.5 w-20 rounded bg-muted animate-pulse" />
                  </div>
                </div>
              ))}
            </div>
          ) : sorted.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
              <Inbox className="h-10 w-10 mb-3 opacity-30" />
              <p className="text-sm">暂无应用</p>
            </div>
          ) : (
            sorted.map((a) => (
              <button
                key={a.id}
                onClick={() => setSelected(a)}
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
                    <ListingBadge listing={a.listing} />
                  </div>
                  <p className="text-xs text-muted-foreground truncate mt-0.5">
                    {a.owner_username} · {timeAgo(a.updated_at)}
                  </p>
                </div>
              </button>
            ))
          )}
        </div>

        {/* Right: Detail Panel */}
        <div className="flex-1 min-w-0">
          {selected ? (
            <div className="rounded-xl border border-border/50 bg-card p-6 space-y-6 md:sticky md:top-6 md:max-h-[calc(100vh-10rem)] md:overflow-y-auto">
              {/* Header */}
              <div className="flex items-start gap-4">
                <AppIcon icon={selected.icon} iconUrl={selected.icon_url} size="h-12 w-12" />
                <div className="flex-1 min-w-0">
                  <h2 className="text-lg font-bold leading-tight">{selected.name}</h2>
                  <p className="text-xs text-muted-foreground font-mono mt-0.5">{selected.slug}</p>
                </div>
                <ListingBadge listing={selected.listing} />
              </div>

              {/* Info */}
              <div className="grid grid-cols-2 gap-x-6 gap-y-2 text-sm">
                <div>
                  <p className="text-xs text-muted-foreground">开发者</p>
                  <p className="font-medium">{selected.owner_username}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">版本</p>
                  <p className="font-mono">{selected.version || "—"}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">更新时间</p>
                  <p>{timeAgo(selected.updated_at)}</p>
                </div>
                {selected.homepage && (
                  <div>
                    <p className="text-xs text-muted-foreground">主页</p>
                    <a
                      href={selected.homepage}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-primary hover:underline flex items-center gap-1 text-sm truncate"
                    >
                      {selected.homepage.replace(/^https?:\/\//, "")}
                      <ExternalLink className="h-3 w-3 shrink-0" />
                    </a>
                  </div>
                )}
              </div>

              {selected.listing_reject_reason && (
                <div className="rounded-lg bg-destructive/5 border border-destructive/20 p-3">
                  <p className="text-xs font-semibold text-destructive mb-1">拒绝原因</p>
                  <p className="text-sm">{selected.listing_reject_reason}</p>
                </div>
              )}

              {/* Core fields for review */}
              <Separator />
              <div className="space-y-3">
                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">核心配置</p>

                {/* Webhook URL */}
                <div className="flex items-start gap-2 text-sm">
                  <Globe className="h-3.5 w-3.5 mt-0.5 text-muted-foreground shrink-0" />
                  <div className="min-w-0">
                    <p className="text-xs text-muted-foreground">Webhook URL</p>
                    {selected.webhook_url ? (
                      <p className="font-mono text-xs truncate">{selected.webhook_url}</p>
                    ) : (
                      <p className="text-xs text-muted-foreground/50">未配置</p>
                    )}
                  </div>
                </div>

                {/* Tools */}
                <div className="flex items-start gap-2 text-sm">
                  <Terminal className="h-3.5 w-3.5 mt-0.5 text-muted-foreground shrink-0" />
                  <div className="min-w-0 flex-1">
                    <p className="text-xs text-muted-foreground mb-1">Tools</p>
                    {(() => {
                      const tools = Array.isArray(selected.tools) ? selected.tools : [];
                      return tools.length > 0 ? (
                        <div className="flex flex-wrap gap-1">
                          {tools.map((t: any, i: number) => (
                            <Badge key={i} variant="secondary" className="text-[10px] font-mono gap-1">
                              {t.command ? `/${t.command}` : t.name}
                            </Badge>
                          ))}
                        </div>
                      ) : (
                        <p className="text-xs text-muted-foreground/50">无</p>
                      );
                    })()}
                  </div>
                </div>

                {/* Events */}
                <div className="flex items-start gap-2 text-sm">
                  <Radio className="h-3.5 w-3.5 mt-0.5 text-muted-foreground shrink-0" />
                  <div className="min-w-0 flex-1">
                    <p className="text-xs text-muted-foreground mb-1">Events</p>
                    {(() => {
                      const events = Array.isArray(selected.events) ? selected.events : [];
                      return events.length > 0 ? (
                        <div className="flex flex-wrap gap-1">
                          {events.map((e: string, i: number) => (
                            <Badge key={i} variant="outline" className="text-[10px] font-mono">
                              {e}
                            </Badge>
                          ))}
                        </div>
                      ) : (
                        <p className="text-xs text-muted-foreground/50">无</p>
                      );
                    })()}
                  </div>
                </div>

                {/* Scopes */}
                <div className="flex items-start gap-2 text-sm">
                  <Shield className="h-3.5 w-3.5 mt-0.5 text-muted-foreground shrink-0" />
                  <div className="min-w-0 flex-1">
                    <p className="text-xs text-muted-foreground mb-1">Scopes</p>
                    {(() => {
                      const scopes = Array.isArray(selected.scopes) ? selected.scopes : [];
                      return scopes.length > 0 ? (
                        <div className="flex flex-wrap gap-1">
                          {scopes.map((s: string, i: number) => (
                            <Badge key={i} variant="outline" className="text-[10px] font-mono">
                              {s}
                            </Badge>
                          ))}
                        </div>
                      ) : (
                        <p className="text-xs text-muted-foreground/50">无</p>
                      );
                    })()}
                  </div>
                </div>

                {/* Config Schema */}
                {selected.config_schema && (
                  <div className="flex items-start gap-2 text-sm">
                    <Terminal className="h-3.5 w-3.5 mt-0.5 text-muted-foreground shrink-0" />
                    <div className="min-w-0 flex-1">
                      <p className="text-xs text-muted-foreground mb-1">Config Schema</p>
                      <pre className="text-[10px] font-mono bg-muted/40 rounded p-2 max-h-32 overflow-auto whitespace-pre-wrap">
                        {typeof selected.config_schema === "string" ? selected.config_schema : JSON.stringify(selected.config_schema, null, 2)}
                      </pre>
                    </div>
                  </div>
                )}
              </div>

              {selected.description && (
                <>
                  <Separator />
                  <div>
                    <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-2">描述</p>
                    <p className="text-sm leading-relaxed">{selected.description}</p>
                  </div>
                </>
              )}

              {selected.readme && (
                <>
                  <Separator />
                  <div>
                    <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-2">README</p>
                    <pre className="text-xs text-muted-foreground whitespace-pre-wrap font-mono bg-muted/40 rounded-lg p-3 max-h-60 overflow-y-auto">
                      {selected.readme}
                    </pre>
                  </div>
                </>
              )}

              {/* Actions */}
              <Separator />
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
              ) : (
                <Button
                  variant="outline"
                  className="w-full"
                  onClick={() => handleToggle(selected)}
                  disabled={submitting}
                >
                  {selected.listing === "listed" ? "下架" : "上架"}
                </Button>
              )}
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
