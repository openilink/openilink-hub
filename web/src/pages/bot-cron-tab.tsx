import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { useToast } from "@/hooks/use-toast";
import { useConfirm } from "@/components/ui/confirm-dialog";
import {
  useCronJobs,
  useCreateCronJob,
  useUpdateCronJob,
  useDeleteCronJob,
} from "@/hooks/use-cron";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Clock, Plus, Trash2, Pencil } from "lucide-react";
import { Textarea } from "@/components/ui/textarea";

const PRESETS = [
  { label: "每分钟", value: "* * * * *" },
  { label: "每 5 分钟", value: "*/5 * * * *" },
  { label: "每 30 分钟", value: "*/30 * * * *" },
  { label: "每小时", value: "0 * * * *" },
  { label: "每天 8:00", value: "0 8 * * *" },
  { label: "每天 9:00", value: "0 9 * * *" },
  { label: "每天 12:00", value: "0 12 * * *" },
  { label: "每天 18:00", value: "0 18 * * *" },
  { label: "工作日 9:00", value: "0 9 * * 1-5" },
  { label: "每周一 9:00", value: "0 9 * * 1" },
  { label: "每月 1 号 9:00", value: "0 9 1 * *" },
];

function formatTime(ts?: number) {
  if (!ts) return "—";
  return new Date(ts * 1000).toLocaleString();
}

function cronDescription(expr: string): string {
  const preset = PRESETS.find((p) => p.value === expr);
  if (preset) return preset.label;
  return expr;
}

export function BotCronTab({ botId }: { botId: string }) {
  const { toast } = useToast();
  const { confirm, ConfirmDialog } = useConfirm();
  const { data: jobs = [], isLoading, isError, error } = useCronJobs(botId);
  const createMutation = useCreateCronJob(botId);
  const updateMutation = useUpdateCronJob(botId);
  const deleteMutation = useDeleteCronJob(botId);

  const [showDialog, setShowDialog] = useState(false);
  const [editingJob, setEditingJob] = useState<any>(null);
  const [form, setForm] = useState({ name: "", cron_expr: "", message: "", recipient: "" });

  function openCreate() {
    setEditingJob(null);
    setForm({ name: "", cron_expr: "0 9 * * *", message: "", recipient: "" });
    setShowDialog(true);
  }

  function openEdit(job: any) {
    setEditingJob(job);
    setForm({
      name: job.name,
      cron_expr: job.cron_expr,
      message: job.message,
      recipient: job.recipient || "",
    });
    setShowDialog(true);
  }

  function handleSubmit() {
    if (!form.cron_expr || !form.message) {
      toast({ variant: "destructive", title: "请填写 cron 表达式和消息内容" });
      return;
    }
    if (editingJob) {
      updateMutation.mutate(
        { jobId: editingJob.id, data: form },
        {
          onSuccess: () => {
            toast({ title: "已更新" });
            setShowDialog(false);
          },
          onError: (e) =>
            toast({ variant: "destructive", title: "更新失败", description: e.message }),
        },
      );
    } else {
      createMutation.mutate(form, {
        onSuccess: () => {
          toast({ title: "已创建" });
          setShowDialog(false);
        },
        onError: (e) =>
          toast({ variant: "destructive", title: "创建失败", description: e.message }),
      });
    }
  }

  async function handleDelete(job: any) {
    const ok = await confirm({
      title: "删除确认",
      description: `确定要删除定时任务「${job.name || job.cron_expr}」？`,
      confirmText: "删除",
      variant: "destructive",
    });
    if (!ok) return;
    deleteMutation.mutate(job.id, {
      onSuccess: () => toast({ title: "已删除" }),
      onError: (e) => toast({ variant: "destructive", title: "删除失败", description: e.message }),
    });
  }

  function handleToggle(job: any, enabled: boolean) {
    updateMutation.mutate(
      { jobId: job.id, data: { enabled } },
      {
        onSuccess: () => toast({ title: enabled ? "已启用" : "已停用" }),
        onError: (e) =>
          toast({ variant: "destructive", title: "操作失败", description: e.message }),
      },
    );
  }

  if (isLoading) {
    return (
      <div className="space-y-3">
        {[1, 2].map((i) => (
          <div key={i} className="h-16 rounded-xl bg-muted animate-pulse" />
        ))}
      </div>
    );
  }

  if (isError) {
    return (
      <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-4 text-sm text-destructive">
        加载定时任务失败{error instanceof Error ? `：${error.message}` : ""}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {ConfirmDialog}

      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-muted-foreground">定时任务</h2>
        <Button size="sm" variant="outline" className="gap-1.5" onClick={openCreate}>
          <Plus className="h-3.5 w-3.5" />
          新建
        </Button>
      </div>

      {jobs.length === 0 ? (
        <div className="text-center py-16 space-y-3 border-2 border-dashed rounded-xl">
          <Clock className="w-8 h-8 mx-auto text-muted-foreground/40" />
          <p className="text-sm text-muted-foreground">暂无定时任务</p>
          <p className="text-xs text-muted-foreground/60">
            创建定时任务，Bot 将按设定的时间自动发送消息。
          </p>
        </div>
      ) : (
        <div className="divide-y divide-border/50 rounded-xl border border-border/50 overflow-hidden">
          {jobs.map((job: any) => (
            <div
              key={job.id}
              className="flex items-center gap-4 px-4 py-3.5 bg-card hover:bg-muted/40 transition-colors"
            >
              <div className="h-9 w-9 rounded-xl bg-muted flex items-center justify-center border shrink-0">
                <Clock className="h-4 w-4 text-muted-foreground/60" />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 flex-wrap">
                  <p className="text-sm font-semibold leading-tight truncate">
                    {job.name || "未命名任务"}
                  </p>
                  <Badge variant="outline" className="text-[10px] font-mono shrink-0">
                    {cronDescription(job.cron_expr)}
                  </Badge>
                  <Badge
                    variant={job.enabled ? "default" : "secondary"}
                    className="text-[10px] shrink-0"
                  >
                    {job.enabled ? "运行中" : "已停用"}
                  </Badge>
                </div>
                <p className="text-xs text-muted-foreground mt-0.5 line-clamp-1">{job.message}</p>
                <div className="flex items-center gap-3 mt-1 text-[10px] text-muted-foreground/50">
                  {job.recipient && <span>发送给: {job.recipient}</span>}
                  {job.last_run_at ? <span>上次: {formatTime(job.last_run_at)}</span> : null}
                  {job.next_run_at ? <span>下次: {formatTime(job.next_run_at)}</span> : null}
                </div>
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <Switch
                  aria-label={`${job.enabled ? "停用" : "启用"}定时任务 ${job.name || job.cron_expr}`}
                  checked={job.enabled}
                  onCheckedChange={(v) => handleToggle(job, v)}
                />
                <Button
                  aria-label={`编辑定时任务 ${job.name || job.cron_expr}`}
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => openEdit(job)}
                >
                  <Pencil className="h-3.5 w-3.5" />
                </Button>
                <Button
                  aria-label={`删除定时任务 ${job.name || job.cron_expr}`}
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => void handleDelete(job)}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <Dialog open={showDialog} onOpenChange={setShowDialog}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{editingJob ? "编辑定时任务" : "新建定时任务"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 pt-2">
            <div className="space-y-2">
              <Label htmlFor="cron-name">任务名称</Label>
              <Input
                id="cron-name"
                placeholder="如：每日早报"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="cron-expr">执行时间</Label>
              <Select
                value={
                  PRESETS.find((p) => p.value === form.cron_expr) ? form.cron_expr : "__custom__"
                }
                onValueChange={(v) => {
                  if (v !== "__custom__") setForm({ ...form, cron_expr: v });
                }}
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择预设时间" />
                </SelectTrigger>
                <SelectContent>
                  {PRESETS.map((p) => (
                    <SelectItem key={p.value} value={p.value}>
                      {p.label} ({p.value})
                    </SelectItem>
                  ))}
                  <SelectItem value="__custom__">自定义</SelectItem>
                </SelectContent>
              </Select>
              <Input
                id="cron-expr"
                placeholder="cron 表达式，如 0 9 * * *"
                value={form.cron_expr}
                onChange={(e) => setForm({ ...form, cron_expr: e.target.value })}
                className="font-mono text-sm"
              />
              <p className="text-[11px] text-muted-foreground/60">
                格式: 分 时 日 月 星期 (0-59 0-23 1-31 1-12 0-6)
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="cron-message">消息内容</Label>
              <Textarea
                id="cron-message"
                placeholder="Bot 将发送的消息内容"
                value={form.message}
                onChange={(e) => setForm({ ...form, message: e.target.value })}
                rows={3}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="cron-recipient">接收人 (可选)</Label>
              <Input
                id="cron-recipient"
                placeholder="留空则发送到默认对话"
                value={form.recipient}
                onChange={(e) => setForm({ ...form, recipient: e.target.value })}
              />
            </div>
            <Button
              className="w-full"
              disabled={createMutation.isPending || updateMutation.isPending}
              onClick={handleSubmit}
            >
              {editingJob ? "保存" : "创建"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
