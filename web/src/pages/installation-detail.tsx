import { useEffect, useState, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  ArrowLeft,
  Eye,
  EyeOff,
  Copy,
  Check,
  ArrowRight,
  Loader2,
  Trash2,
  RefreshCw,
} from "lucide-react";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Card, CardHeader, CardTitle, CardAction, CardContent } from "../components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "../components/ui/dialog";
import { Skeleton } from "../components/ui/skeleton";
import { Switch } from "../components/ui/switch";
import { Label } from "../components/ui/label";
import { api } from "../lib/api";
import { useToast } from "@/hooks/use-toast";
import { AppIcon } from "../components/app-icon";

// ==================== Page ====================

export function InstallationDetailPage() {
  const { id: botId, iid } = useParams<{ id: string; iid: string }>();
  const navigate = useNavigate();
  const { toast } = useToast();
  const [inst, setInst] = useState<any>(null);
  const [app, setApp] = useState<any>(null);
  const [botName, setBotName] = useState("");
  const [loading, setLoading] = useState(true);

  const loadData = useCallback(async () => {
    try {
      const installations = (await api.listBotApps(botId!)) || [];
      const found = installations.find((i: any) => i.id === iid);
      if (!found) throw new Error("未找到安装实例");
      setInst(found);

      const [appData, bots] = await Promise.all([
        api.getApp(found.app_id),
        api.listBots(),
      ]);
      setApp(appData);
      const bot = (bots || []).find((b: any) => b.id === botId);
      if (bot) setBotName(bot.name);
    } catch (e: any) {
      toast({ variant: "destructive", title: "加载失败", description: e.message });
    } finally {
      setLoading(false);
    }
  }, [botId, iid, toast]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  if (loading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-20 w-full rounded-3xl" />
        <Skeleton className="h-64 w-full rounded-3xl" />
        <Skeleton className="h-48 w-full rounded-3xl" />
      </div>
    );
  }

  if (!inst || !app) {
    return (
      <div className="py-20 text-center space-y-4">
        <p className="font-bold">未找到安装实例</p>
        <Button variant="link" onClick={() => navigate(`/dashboard/accounts/${botId}`)}>
          返回账号
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="space-y-4">
        <button
          onClick={() => navigate(`/dashboard/accounts/${botId}`)}
          className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
        >
          <ArrowLeft className="h-4 w-4" />
          {botName || "返回"}
        </button>

        <div className="flex items-start gap-4">
          <AppIcon
            icon={inst.app_icon || app.icon}
            iconUrl={inst.app_icon_url || app.icon_url}
            size="h-14 w-14"
          />
          <div className="flex-1 min-w-0 space-y-1">
            <div className="flex items-center gap-3 flex-wrap">
              <h1 className="text-2xl font-bold">{inst.app_name || app.name}</h1>
              {inst.handle && (
                <span className="text-sm text-muted-foreground font-mono">@{inst.handle}</span>
              )}
              <Badge
                variant={inst.enabled ? "default" : "secondary"}
                className="rounded-full font-bold"
              >
                {inst.enabled ? "运行中" : "已停用"}
              </Badge>
              {app.registry && app.registry !== "builtin" && (
                <Badge variant="outline" className="rounded-full font-bold">
                  来自应用市场
                </Badge>
              )}
              {app.registry === "builtin" && (
                <Badge variant="outline" className="rounded-full font-bold">
                  内置应用
                </Badge>
              )}
            </div>
            {app.description && (
              <p className="text-sm text-muted-foreground">{app.description}</p>
            )}
          </div>
        </div>
      </div>

      {/* Token & Usage */}
      <TokenSection app={app} inst={inst} />

      {/* Config */}
      <ConfigSection
        app={app}
        inst={inst}
        botId={botId!}
        onUpdate={loadData}
        onUninstall={() => navigate(`/dashboard/accounts/${botId}`)}
      />

      {/* Event Logs */}
      <EventLogsSection appId={inst.app_id} instId={inst.id} />

      {/* API Logs */}
      <ApiLogsSection appId={inst.app_id} instId={inst.id} />
    </div>
  );
}

// ==================== Token & Usage Section ====================

function TokenSection({ app, inst }: { app: any; inst: any }) {
  const [showToken, setShowToken] = useState(false);
  const [copied, setCopied] = useState(false);
  const { toast } = useToast();
  const token = inst.app_token || inst.token || "";
  const hubUrl = window.location.origin;

  function handleCopy(text: string) {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }).catch(() => {
      toast({ variant: "destructive", title: "复制失败", description: "请手动选中复制" });
    });
  }

  const maskedToken = token
    ? token.slice(0, 8) + "****" + token.slice(-4)
    : "---";

  // Render guide with variable replacement
  function renderGuide(): string | null {
    if (app.guide) {
      return app.guide
        .replace(/\{hub_url\}/g, hubUrl)
        .replace(/\{your_token\}/g, token || "<your_token>");
    }
    return null;
  }

  const guideText = renderGuide();
  const showGenericGuide = !guideText && app.registry === "builtin";
  const showUsageGuide = guideText || showGenericGuide;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Token & 使用方式</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Token display */}
        <div className="space-y-1.5">
          <Label className="text-muted-foreground">Token</Label>
          <div className="flex items-center gap-2 p-2 rounded-md border bg-background">
            <code className="text-xs font-mono flex-1 break-all select-all">
              {showToken ? token : maskedToken}
            </code>
            <button
              type="button"
              onClick={() => setShowToken(!showToken)}
              className="cursor-pointer text-muted-foreground hover:text-foreground shrink-0"
              aria-label={showToken ? "隐藏" : "显示"}
            >
              {showToken ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
            </button>
            <button
              type="button"
              onClick={() => handleCopy(token)}
              className="cursor-pointer text-muted-foreground hover:text-foreground shrink-0"
              aria-label="复制"
            >
              {copied ? (
                <Check className="w-3.5 h-3.5 text-primary" />
              ) : (
                <Copy className="w-3.5 h-3.5" />
              )}
            </button>
          </div>
        </div>

        {/* Guide content */}
        {guideText && (
          <div className="text-sm text-muted-foreground whitespace-pre-wrap font-mono leading-relaxed p-3 rounded-md bg-muted/30 border overflow-x-auto">
            {guideText}
          </div>
        )}

        {showGenericGuide && (
          <div className="space-y-3">
            <details className="group">
              <summary className="text-sm font-medium cursor-pointer flex items-center gap-2 select-none">
                <ArrowRight className="h-3.5 w-3.5 transition-transform group-open:rotate-90" />
                WebSocket 连接
              </summary>
              <pre className="mt-2 p-3 rounded-md bg-muted/30 border text-xs font-mono overflow-x-auto whitespace-pre-wrap">
                {`wss://${hubUrl.replace(/^https?:\/\//, "")}/bot/v1/ws?token=${token || "<your_token>"}`}
              </pre>
            </details>

            <details className="group">
              <summary className="text-sm font-medium cursor-pointer flex items-center gap-2 select-none">
                <ArrowRight className="h-3.5 w-3.5 transition-transform group-open:rotate-90" />
                HTTP 发消息
              </summary>
              <pre className="mt-2 p-3 rounded-md bg-muted/30 border text-xs font-mono overflow-x-auto whitespace-pre-wrap">
                {`curl -X POST ${hubUrl}/bot/v1/message/send \\\n  -H "Authorization: Bearer ${token || "<your_token>"}" \\\n  -H "Content-Type: application/json" \\\n  -d '{"content":"hello"}'`}
              </pre>
            </details>
          </div>
        )}

        {/* Non-integration apps with no guide: just show the token (already shown above) */}
        {!showUsageGuide && app.webhook_url && (
          <p className="text-xs text-muted-foreground">
            事件将推送到 <code className="font-mono">{app.webhook_url}</code>
          </p>
        )}
      </CardContent>
    </Card>
  );
}

// ==================== Config Section ====================

function ConfigSection({
  app,
  inst,
  botId,
  onUpdate,
  onUninstall,
}: {
  app: any;
  inst: any;
  botId: string;
  onUpdate: () => void;
  onUninstall: () => void;
}) {
  const { toast } = useToast();
  const [handle, setHandle] = useState(inst.handle || "");
  const [enabled, setEnabled] = useState(inst.enabled ?? true);
  const [saving, setSaving] = useState(false);
  const [showUninstallDialog, setShowUninstallDialog] = useState(false);
  const [uninstalling, setUninstalling] = useState(false);

  async function handleSave() {
    setSaving(true);
    try {
      await api.updateInstallation(inst.app_id, inst.id, {
        handle: handle.trim(),
        enabled,
      });
      toast({ title: "已保存" });
      onUpdate();
    } catch (e: any) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
    setSaving(false);
  }

  async function handleUninstall() {
    setUninstalling(true);
    try {
      await api.deleteInstallation(inst.app_id, inst.id);
      toast({ title: "已卸载" });
      onUninstall();
    } catch (e: any) {
      toast({ variant: "destructive", title: "卸载失败", description: e.message });
    }
    setUninstalling(false);
  }

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>配置</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="inst-handle" className="text-muted-foreground">
              Handle
            </Label>
            <div className="flex items-center gap-2">
              <Input
                id="inst-handle"
                value={handle}
                onChange={(e) => setHandle(e.target.value)}
                className="h-8 text-xs font-mono flex-1"
                placeholder="如 notify-prod"
              />
              <span className="text-xs text-muted-foreground font-mono shrink-0">
                @{handle || "handle"}
              </span>
            </div>
          </div>

          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label htmlFor="inst-enabled">启用状态</Label>
              <p className="text-xs text-muted-foreground">
                {enabled ? "应用正在接收事件和处理消息" : "应用已停用，不会接收任何事件"}
              </p>
            </div>
            <Switch id="inst-enabled" checked={enabled} onCheckedChange={setEnabled} />
          </div>

          <div className="flex items-center gap-2 pt-2 border-t">
            <Button size="sm" onClick={handleSave} disabled={saving}>
              {saving && <Loader2 className="h-3 w-3 animate-spin mr-1" />}
              保存
            </Button>
            <div className="flex-1" />
            <Button
              variant="destructive"
              size="sm"
              onClick={() => setShowUninstallDialog(true)}
            >
              <Trash2 className="h-3.5 w-3.5 mr-1" />
              卸载
            </Button>
          </div>
        </CardContent>
      </Card>

      <Dialog open={showUninstallDialog} onOpenChange={setShowUninstallDialog}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>确认卸载</DialogTitle>
            <DialogDescription>
              卸载后将删除此安装实例，Token 将失效，此操作不可撤销。
            </DialogDescription>
          </DialogHeader>
          <div className="flex justify-end gap-2 pt-4">
            <Button variant="ghost" onClick={() => setShowUninstallDialog(false)}>
              取消
            </Button>
            <Button variant="destructive" onClick={handleUninstall} disabled={uninstalling}>
              {uninstalling && <Loader2 className="h-3 w-3 animate-spin mr-1" />}
              确认卸载
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}

// ==================== Event Logs Section ====================

function EventLogsSection({ appId, instId }: { appId: string; instId: string }) {
  const [logs, setLogs] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const loadLogs = useCallback(async () => {
    try {
      const data = (await api.listEventLogs(appId, instId, 50)) || [];
      setLogs(data);
    } catch {
      setLogs([]);
    } finally {
      setLoading(false);
    }
  }, [appId, instId]);

  useEffect(() => {
    loadLogs();
    const t = setInterval(loadLogs, 10000);
    return () => clearInterval(t);
  }, [loadLogs]);

  return (
    <Card>
      <CardHeader>
        <CardTitle>事件投递日志</CardTitle>
        <CardAction>
          <Button
            variant="ghost"
            size="sm"
            className="h-7 text-xs gap-1"
            onClick={() => { setLoading(true); loadLogs(); }}
          >
            <RefreshCw className="h-3 w-3" />
            刷新
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="space-y-2">
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-8 w-full" />
            ))}
          </div>
        ) : logs.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-8">暂无事件日志</p>
        ) : (
          <div className="rounded-md border overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>时间</TableHead>
                  <TableHead>事件类型</TableHead>
                  <TableHead>Trace ID</TableHead>
                  <TableHead>状态码</TableHead>
                  <TableHead>耗时</TableHead>
                  <TableHead>错误</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {logs.map((log) => (
                  <TableRow key={log.id || log.trace_id + log.created_at}>
                    <TableCell className="font-mono whitespace-nowrap">
                      {formatTime(log.created_at)}
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline" className="font-mono">
                        {log.event_type}
                      </Badge>
                    </TableCell>
                    <TableCell className="font-mono text-muted-foreground">
                      {log.trace_id ? log.trace_id.slice(0, 12) + "..." : "-"}
                    </TableCell>
                    <TableCell>
                      <StatusBadge status={log.status_code || log.status} />
                    </TableCell>
                    <TableCell className="font-mono">
                      {log.duration_ms != null ? `${log.duration_ms}ms` : "-"}
                    </TableCell>
                    <TableCell className="text-destructive max-w-48 truncate">
                      {log.error || "-"}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// ==================== API Logs Section ====================

function ApiLogsSection({ appId, instId }: { appId: string; instId: string }) {
  const [logs, setLogs] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  const loadLogs = useCallback(async () => {
    try {
      const data = (await api.listApiLogs(appId, instId, 50)) || [];
      setLogs(data);
    } catch {
      setLogs([]);
    } finally {
      setLoading(false);
    }
  }, [appId, instId]);

  useEffect(() => {
    loadLogs();
    const t = setInterval(loadLogs, 10000);
    return () => clearInterval(t);
  }, [loadLogs]);

  return (
    <Card>
      <CardHeader>
        <CardTitle>API 调用日志</CardTitle>
        <CardAction>
          <Button
            variant="ghost"
            size="sm"
            className="h-7 text-xs gap-1"
            onClick={() => { setLoading(true); loadLogs(); }}
          >
            <RefreshCw className="h-3 w-3" />
            刷新
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="space-y-2">
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-8 w-full" />
            ))}
          </div>
        ) : logs.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-8">暂无 API 日志</p>
        ) : (
          <div className="rounded-md border overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>时间</TableHead>
                  <TableHead>方法</TableHead>
                  <TableHead>路径</TableHead>
                  <TableHead>状态码</TableHead>
                  <TableHead>耗时</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {logs.map((log, idx) => (
                  <TableRow key={log.id || idx}>
                    <TableCell className="font-mono whitespace-nowrap">
                      {formatTime(log.created_at)}
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline" className="font-mono font-bold">
                        {log.method}
                      </Badge>
                    </TableCell>
                    <TableCell className="font-mono text-muted-foreground max-w-64 truncate">
                      {log.path}
                    </TableCell>
                    <TableCell>
                      <StatusBadge status={log.status_code || log.status} />
                    </TableCell>
                    <TableCell className="font-mono">
                      {log.duration_ms != null ? `${log.duration_ms}ms` : "-"}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// ==================== Helpers ====================

function StatusBadge({ status }: { status: number | string | undefined }) {
  if (status == null) return <span className="text-xs text-muted-foreground">-</span>;
  const n = typeof status === "string" ? parseInt(status, 10) : status;
  if (isNaN(n)) return <span className="text-xs text-muted-foreground">{status}</span>;

  const variant = n >= 200 && n < 300 ? "default" : n >= 400 ? "destructive" : "outline";
  return (
    <Badge variant={variant} className="font-mono">
      {n}
    </Badge>
  );
}

function formatTime(ts: string | undefined): string {
  if (!ts) return "-";
  try {
    const d = new Date(ts);
    return d.toLocaleString("zh-CN", {
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  } catch {
    return ts;
  }
}
