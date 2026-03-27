import { useEffect, useState } from "react";
import { Check, X, ExternalLink } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
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
          prev ? (data.find((a: any) => a.id === prev.id) ?? prev) : null
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

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">审核中心</h1>
          <p className="text-sm text-muted-foreground mt-0.5">审核应用上架请求。</p>
        </div>
      </div>

      <div className="rounded-xl border border-border/50 overflow-hidden">
        <Table className="table-fixed">
          <TableHeader className="bg-muted/30">
            <TableRow>
              <TableHead className="w-[280px]">应用</TableHead>
              <TableHead>开发者</TableHead>
              <TableHead>更新时间</TableHead>
              <TableHead>状态</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <>
                {[1, 2, 3].map((i) => (
                  <TableRow key={i}>
                    <TableCell>
                      <div className="flex items-center gap-2.5">
                        <div className="h-7 w-7 rounded-lg bg-muted animate-pulse shrink-0" />
                        <div className="space-y-1.5 flex-1">
                          <div className="h-3 w-28 rounded bg-muted animate-pulse" />
                          <div className="h-2.5 w-20 rounded bg-muted animate-pulse" />
                        </div>
                      </div>
                    </TableCell>
                    <TableCell><div className="h-3 w-16 rounded bg-muted animate-pulse" /></TableCell>
                    <TableCell><div className="h-3 w-12 rounded bg-muted animate-pulse" /></TableCell>
                    <TableCell><div className="h-5 w-14 rounded bg-muted animate-pulse" /></TableCell>
                    <TableCell />
                  </TableRow>
                ))}
              </>
            ) : (
              <>
                {apps.map((a) => (
                  <TableRow
                    key={a.id}
                    className="cursor-pointer"
                    onClick={() => setSelected(a)}
                  >
                    <TableCell>
                      <div className="flex items-center gap-2.5">
                        <AppIcon icon={a.icon} iconUrl={a.icon_url} size="h-7 w-7" />
                        <div className="min-w-0">
                          <p className="text-sm font-medium leading-tight truncate">{a.name}</p>
                          <p className="text-xs text-muted-foreground font-mono truncate">{a.slug}</p>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell className="text-sm">{a.owner_username}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {timeAgo(a.updated_at)}
                    </TableCell>
                    <TableCell><ListingBadge listing={a.listing} /></TableCell>
                    <TableCell className="text-right" onClick={(e) => e.stopPropagation()}>
                      <div className="flex items-center justify-end gap-1">
                        {a.listing === "pending" ? (
                          <>
                            <Button
                              size="xs"
                              variant="outline"
                              className="gap-1 text-emerald-600 border-emerald-300 hover:bg-emerald-50 dark:hover:bg-emerald-950/30"
                              onClick={() => handleApprove(a)}
                              disabled={submitting}
                            >
                              <Check className="h-3 w-3" /> 通过
                            </Button>
                            <Button
                              size="xs"
                              variant="outline"
                              className="gap-1 text-destructive border-destructive/30 hover:bg-destructive/5"
                              onClick={() => { setRejectTarget(a); setRejectReason(""); }}
                              disabled={submitting}
                            >
                              <X className="h-3 w-3" /> 拒绝
                            </Button>
                          </>
                        ) : (
                          <Button size="xs" variant="outline" onClick={() => handleToggle(a)} disabled={submitting}>
                            {a.listing === "listed" ? "下架" : "上架"}
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {apps.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center text-sm text-muted-foreground py-10">
                      暂无应用
                    </TableCell>
                  </TableRow>
                )}
              </>
            )}
          </TableBody>
        </Table>
      </div>

      {/* 详情 Drawer */}
      <Sheet open={!!selected} onOpenChange={(o) => !o && setSelected(null)}>
        <SheetContent className="w-[480px] sm:max-w-[480px] overflow-y-auto">
          {selected && (
            <>
              <SheetHeader className="pb-4">
                <div className="flex items-center gap-3">
                  <AppIcon icon={selected.icon} iconUrl={selected.icon_url} size="h-10 w-10" />
                  <div className="flex-1 min-w-0">
                    <SheetTitle className="text-left leading-tight">{selected.name}</SheetTitle>
                    <p className="text-xs text-muted-foreground font-mono mt-0.5">{selected.slug}</p>
                  </div>
                  <ListingBadge listing={selected.listing} />
                </div>
              </SheetHeader>

              <div className="space-y-5">
                {/* 基本信息 */}
                <div className="space-y-2">
                  <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">基本信息</p>
                  <div className="space-y-1 text-sm">
                    <div className="flex gap-2">
                      <span className="text-muted-foreground w-16 shrink-0">开发者</span>
                      <span>{selected.owner_username}</span>
                    </div>
                    <div className="flex gap-2">
                      <span className="text-muted-foreground w-16 shrink-0">版本</span>
                      <span className="font-mono">{selected.version || "—"}</span>
                    </div>
                    {selected.homepage && (
                      <div className="flex gap-2">
                        <span className="text-muted-foreground w-16 shrink-0">主页</span>
                        <a
                          href={selected.homepage}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-primary hover:underline flex items-center gap-1 truncate"
                        >
                          {selected.homepage}
                          <ExternalLink className="h-3 w-3 shrink-0" />
                        </a>
                      </div>
                    )}
                    {selected.listing_reject_reason && (
                      <div className="flex gap-2">
                        <span className="text-muted-foreground w-16 shrink-0">拒绝原因</span>
                        <span className="text-destructive">{selected.listing_reject_reason}</span>
                      </div>
                    )}
                  </div>
                </div>

                {selected.description && (
                  <>
                    <Separator />
                    <div className="space-y-1.5">
                      <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">描述</p>
                      <p className="text-sm leading-relaxed">{selected.description}</p>
                    </div>
                  </>
                )}

                {selected.readme && (
                  <>
                    <Separator />
                    <div className="space-y-1.5">
                      <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">README</p>
                      <pre className="text-xs text-muted-foreground whitespace-pre-wrap font-mono bg-muted/40 rounded-lg p-3 max-h-48 overflow-y-auto">
                        {selected.readme}
                      </pre>
                    </div>
                  </>
                )}

                {/* 操作按钮 */}
                {selected.listing === "pending" && (
                  <>
                    <Separator />
                    <div className="flex gap-2">
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
                  </>
                )}
                {selected.listing !== "pending" && (
                  <>
                    <Separator />
                    <Button
                      variant="outline"
                      className="w-full"
                      onClick={() => handleToggle(selected)}
                      disabled={submitting}
                    >
                      {selected.listing === "listed" ? "下架" : "上架"}
                    </Button>
                  </>
                )}
              </div>
            </>
          )}
        </SheetContent>
      </Sheet>

      {/* 拒绝 Dialog */}
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
