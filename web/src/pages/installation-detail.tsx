import { useEffect, useState, useCallback } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
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
  Key,
  Settings,
  ScrollText,
  Terminal,
  Sliders,
  ChevronRight,
} from "lucide-react";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import {
  Card,
  CardHeader,
  CardTitle,
  CardAction,
  CardContent,
  CardDescription,
} from "../components/ui/card";
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
import { api, botDisplayName } from "../lib/api";
import { useToast } from "@/hooks/use-toast";
import { AppIcon } from "../components/app-icon";
import { ToolsDisplay, parseTools } from "../components/tools-display";

// ==================== Nav Definition ====================

type SectionKey = "token" | "config" | "app-config" | "tools" | "event-logs" | "api-logs";

function buildNavSections(app: any, inst?: any) {
  const items: { key: SectionKey; label: string; icon: any }[] = [
    { key: "token", label: "Token & 使用", icon: Key },
  ];
  if (parseTools(app?.tools).length > 0 || parseTools(inst?.tools).length > 0) {
    items.push({ key: "tools", label: "命令 / 工具", icon: Terminal });
  }
  if (app?.config_schema) {
    let parsed: any = {};
    try {
      parsed =
        typeof app.config_schema === "string"
          ? JSON.parse(app.config_schema || "{}")
          : app.config_schema || {};
    } catch {}
    if (Object.keys(parsed.properties || {}).length > 0) {
      items.push({ key: "app-config", label: "应用配置", icon: Sliders });
    }
  }
  items.push({ key: "config", label: "安装设置", icon: Settings });
  items.push({ key: "event-logs", label: "事件日志", icon: ScrollText });
  items.push({ key: "api-logs", label: "API 日志", icon: ScrollText });
  return items;
}

// ==================== Page ====================

export function InstallationDetailPage() {
  const { id: botId, iid } = useParams<{ id: string; iid: string }>();
  const navigate = useNavigate();
  const { toast } = useToast();
  const [inst, setInst] = useState<any>(null);
  const [app, setApp] = useState<any>(null);
  const [botName, setBotName] = useState("");
  const [loading, setLoading] = useState(true);
  const [section, setSection] = useState<SectionKey>("token");

  const loadData = useCallback(async () => {
    try {
      const installations = (await api.listBotApps(botId!)) || [];
      const found = installations.find((i: any) => i.id === iid);
      if (!found) throw new Error("未找到安装实例");
      setInst(found);

      const [appData, bots] = await Promise.all([api.getApp(found.app_id), api.listBots()]);
      setApp(appData);
      const bot = (bots || []).find((b: any) => b.id === botId);
      if (bot) setBotName(botDisplayName(bot));
    } catch (e: any) {
      toast({ variant: "destructive", title: "加载失败", description: e.message });
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [botId, iid]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  if (loading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-20 w-full rounded-3xl" />
        <Skeleton className="h-64 w-full rounded-3xl" />
      </div>
    );
  }

  if (!inst || !app) {
    return (
      <div className="py-20 text-center space-y-4">
        <p className="font-bold">未找到安装实例</p>
        <Button variant="link" asChild>
          <Link to={`/dashboard/accounts/${botId}`}>返回账号</Link>
        </Button>
      </div>
    );
  }

  const navItems = buildNavSections(app, inst);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="space-y-4">
        <Button
          variant="ghost"
          size="sm"
          asChild
          className="gap-1.5 text-muted-foreground hover:text-foreground"
        >
          <Link to={`/dashboard/accounts/${botId}`}>
            <ArrowLeft className="h-4 w-4" />
            {botName || "返回"}
          </Link>
        </Button>

        <div className="flex items-start gap-4">
          <AppIcon
            icon={inst.app_icon || app.icon}
            iconUrl={inst.app_icon_url || app.icon_url}
            size="h-14 w-14"
          />
          <div className="flex-1 min-w-0 space-y-1">
            <div className="flex items-center gap-3 flex-wrap">
              <h1 className="text-2xl font-bold tracking-tight">{inst.app_name || app.name}</h1>
              {inst.handle ? (
                <span className="text-sm text-muted-foreground font-mono">@{inst.handle}</span>
              ) : null}
              <Badge
                variant={inst.enabled ? "default" : "secondary"}
                className="rounded-full font-bold"
              >
                {inst.enabled ? "运行中" : "已停用"}
              </Badge>
              {app.registry && app.registry !== "builtin" ? (
                <Badge variant="outline" className="rounded-full font-bold">
                  来自应用市场
                </Badge>
              ) : null}
              {app.registry === "builtin" ? (
                <Badge variant="outline" className="rounded-full font-bold">
                  内置应用
                </Badge>
              ) : null}
            </div>
            {app.description ? (
              <p className="text-sm text-muted-foreground">{app.description}</p>
            ) : null}
          </div>
        </div>
      </div>

      {/* Mobile nav */}
      <div className="md:hidden">
        <select
          value={section}
          onChange={(e) => setSection(e.target.value as SectionKey)}
          className="w-full h-9 px-3 rounded-md border bg-background text-sm"
          aria-label="选择页面"
        >
          {navItems.map((item) => (
            <option key={item.key} value={item.key}>
              {item.label}
            </option>
          ))}
        </select>
      </div>

      {/* Desktop: Left nav + Right content */}
      <div className="flex gap-8">
        <nav className="hidden md:block w-48 shrink-0 space-y-1">
          {navItems.map((item) => (
            <Button
              key={item.key}
              variant="ghost"
              size="sm"
              onClick={() => setSection(item.key)}
              className={`w-full justify-start gap-2 ${
                section === item.key
                  ? "bg-primary/10 text-primary font-medium hover:bg-primary/10 hover:text-primary"
                  : "text-muted-foreground"
              }`}
            >
              <item.icon className="h-4 w-4 shrink-0" />
              {item.label}
            </Button>
          ))}
        </nav>

        <div className="flex-1 min-w-0">
          {section === "token" && <TokenSection app={app} inst={inst} />}
          {section === "tools" &&
            (() => {
              const appTools = parseTools(app?.tools);
              const instTools = parseTools(inst?.tools);
              return (
                <div className="space-y-6">
                  <div>
                    <h2 className="text-base font-semibold">命令 / 工具</h2>
                    <p className="text-sm text-muted-foreground mt-1">此安装可用的命令和工具。</p>
                  </div>
                  {instTools.length > 0 ? (
                    <Card className="p-5 space-y-3">
                      <h3 className="text-xs font-bold uppercase tracking-wider text-muted-foreground">
                        自定义命令
                      </h3>
                      <ToolsDisplay tools={instTools} />
                    </Card>
                  ) : null}
                  {appTools.length > 0 ? (
                    <Card className="p-5 space-y-3">
                      {instTools.length > 0 ? (
                        <h3 className="text-xs font-bold uppercase tracking-wider text-muted-foreground">
                          应用命令
                        </h3>
                      ) : null}
                      <ToolsDisplay tools={appTools} />
                    </Card>
                  ) : null}
                </div>
              );
            })()}
          {section === "app-config" && <AppConfigForm app={app} inst={inst} onUpdate={loadData} />}
          {section === "config" && (
            <ConfigSection
              app={app}
              inst={inst}
              botId={botId!}
              onUpdate={loadData}
              onUninstall={() => navigate(`/dashboard/accounts/${botId}`)}
            />
          )}
          {section === "event-logs" && <EventLogsSection appId={inst.app_id} instId={inst.id} botId={botId!} />}
          {section === "api-logs" && <ApiLogsSection appId={inst.app_id} instId={inst.id} />}
        </div>
      </div>
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
    navigator.clipboard
      .writeText(text)
      .then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      })
      .catch(() => {
        toast({ variant: "destructive", title: "复制失败", description: "请手动选中复制" });
      });
  }

  const maskedToken = token ? token.slice(0, 8) + "****" + token.slice(-4) : "---";

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
    <div className="space-y-6">
      <div>
        <h2 className="text-base font-semibold">Token & 使用方式</h2>
        <p className="text-sm text-muted-foreground mt-1">应用的 Token 和接入指南。</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Token</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-2 p-2 rounded-md border bg-background">
            <code className="text-xs font-mono flex-1 break-all select-all">
              {showToken ? token : maskedToken}
            </code>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              onClick={() => setShowToken(!showToken)}
              aria-label={showToken ? "隐藏" : "显示"}
            >
              {showToken ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              onClick={() => handleCopy(token)}
              aria-label="复制"
            >
              {copied ? (
                <Check className="w-3.5 h-3.5 text-primary" />
              ) : (
                <Copy className="w-3.5 h-3.5" />
              )}
            </Button>
          </div>
        </CardContent>
      </Card>

      {guideText ? (
        <Card>
          <CardHeader>
            <CardTitle>使用指南</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-sm text-muted-foreground whitespace-pre-wrap font-mono leading-relaxed p-3 rounded-md bg-muted/30 border overflow-x-auto">
              {guideText}
            </div>
          </CardContent>
        </Card>
      ) : null}

      {showGenericGuide ? (
        <Card>
          <CardHeader>
            <CardTitle>接入方式</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
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
          </CardContent>
        </Card>
      ) : null}

      {!showUsageGuide && app.webhook_url ? (
        <Card>
          <CardContent>
            <p className="text-xs text-muted-foreground">
              事件将推送到 <code className="font-mono">{app.webhook_url}</code>
            </p>
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}

// ==================== App Config Form (config_schema) ====================

function AppConfigForm({ app, inst, onUpdate }: { app: any; inst: any; onUpdate: () => void }) {
  const { toast } = useToast();
  const [saving, setSaving] = useState(false);

  let parsed: any = {};
  try {
    parsed =
      typeof app.config_schema === "string"
        ? JSON.parse(app.config_schema || "{}")
        : app.config_schema || {};
  } catch {
    return null;
  }
  const properties = parsed.properties || {};
  if (Object.keys(properties).length === 0) return null;

  let currentConfig: Record<string, string> = {};
  try {
    currentConfig =
      typeof inst.config === "string" ? JSON.parse(inst.config || "{}") : inst.config || {};
  } catch {}

  const [form, setForm] = useState<Record<string, string>>(currentConfig);

  async function handleSave() {
    setSaving(true);
    try {
      await api.updateInstallation(inst.app_id, inst.id, {
        config: JSON.stringify(form),
      });
      toast({ title: "配置已保存" });
      onUpdate();
    } catch (e: any) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
    setSaving(false);
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-base font-semibold">应用配置</h2>
        <p className="text-sm text-muted-foreground mt-1">配置此应用的运行参数。</p>
      </div>

      <Card>
        <CardContent className="space-y-4 pt-6">
          {Object.entries(properties).map(([key, prop]: [string, any]) => (
            <div key={key} className="space-y-1.5">
              <Label className="text-muted-foreground">{prop.title || key}</Label>
              <Input
                value={form[key] || ""}
                onChange={(e) => setForm({ ...form, [key]: e.target.value })}
                className="h-8 text-xs font-mono"
                placeholder={prop.description || ""}
              />
              {prop.description ? (
                <p className="text-xs text-muted-foreground">{prop.description}</p>
              ) : null}
            </div>
          ))}
          <div className="flex items-center gap-2 pt-2 border-t">
            <Button size="sm" onClick={handleSave} disabled={saving}>
              {saving && <Loader2 className="h-3 w-3 animate-spin mr-1" />}
              保存配置
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
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
    } finally {
      setSaving(false);
    }
  }

  async function handleUninstall() {
    setUninstalling(true);
    try {
      await api.deleteInstallation(inst.app_id, inst.id);
      toast({ title: "已卸载" });
      onUninstall();
    } catch (e: any) {
      toast({ variant: "destructive", title: "卸载失败", description: e.message });
    } finally {
      setUninstalling(false);
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-base font-semibold">安装设置</h2>
        <p className="text-sm text-muted-foreground mt-1">
          管理此安装实例的 Handle、状态和生命周期。
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Handle</CardTitle>
          <CardDescription>用户通过 @handle 触发此应用</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-2">
            <Input
              value={handle}
              onChange={(e) => setHandle(e.target.value)}
              className="h-8 text-xs font-mono flex-1"
              placeholder="如 notify-prod"
            />
            <span className="text-xs text-muted-foreground font-mono shrink-0">
              @{handle || "handle"}
            </span>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>启用状态</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              {enabled ? "应用正在接收事件和处理消息" : "应用已停用，不会接收任何事件"}
            </p>
            <Switch checked={enabled} onCheckedChange={setEnabled} />
          </div>
        </CardContent>
      </Card>

      <div className="flex items-center gap-2">
        <Button size="sm" onClick={handleSave} disabled={saving}>
          {saving && <Loader2 className="h-3 w-3 animate-spin mr-1" />}
          保存
        </Button>
        <div className="flex-1" />
        <Button variant="destructive" size="sm" onClick={() => setShowUninstallDialog(true)}>
          <Trash2 className="h-3.5 w-3.5 mr-1" />
          卸载应用
        </Button>
      </div>

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
    </div>
  );
}

// ==================== Event Logs Section ====================

function EventLogsSection({ appId, instId, botId }: { appId: string; instId: string; botId: string }) {
  const navigate = useNavigate();
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
    let timer: ReturnType<typeof setTimeout>;
    let cancelled = false;
    async function poll() {
      await loadLogs();
      if (!cancelled) timer = setTimeout(poll, 10000);
    }
    poll();
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [loadLogs]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-base font-semibold">事件投递日志</h2>
          <p className="text-sm text-muted-foreground mt-1">Hub 推送到此应用的事件记录。</p>
        </div>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 text-xs gap-1"
          onClick={() => {
            setLoading(true);
            loadLogs();
          }}
        >
          <RefreshCw className="h-3 w-3" />
          刷新
        </Button>
      </div>

      <Card className="overflow-hidden">
        {loading ? (
          <div className="p-6 space-y-2">
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-8 w-full" />
            ))}
          </div>
        ) : logs.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-8 px-4">暂无事件日志</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>时间</TableHead>
                <TableHead>事件类型</TableHead>
                <TableHead>Trace ID</TableHead>
                <TableHead>状态码</TableHead>
                <TableHead>耗时</TableHead>
                <TableHead>错误</TableHead>
                <TableHead className="w-8" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {logs.map((log) => (
                <TableRow
                  key={log.id || log.trace_id + log.created_at}
                  className={log.trace_id ? "cursor-pointer hover:bg-primary/5" : ""}
                  onClick={() => log.trace_id && navigate(`/dashboard/accounts/${botId}/traces/${log.trace_id}`)}
                >
                  <TableCell className="font-mono whitespace-nowrap">
                    {formatTime(log.created_at)}
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline" className="font-mono">
                      {log.event_type}
                    </Badge>
                  </TableCell>
                  <TableCell className="font-mono text-muted-foreground">
                    {log.trace_id ? log.trace_id.slice(0, 12) + "…" : "-"}
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
                  <TableCell className="w-8 px-2">
                    {log.trace_id && <ChevronRight className="h-4 w-4 text-muted-foreground" />}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>
    </div>
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
    let timer: ReturnType<typeof setTimeout>;
    let cancelled = false;
    async function poll() {
      await loadLogs();
      if (!cancelled) timer = setTimeout(poll, 10000);
    }
    poll();
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [loadLogs]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-base font-semibold">API 调用日志</h2>
          <p className="text-sm text-muted-foreground mt-1">此应用通过 Bot API 发起的调用记录。</p>
        </div>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 text-xs gap-1"
          onClick={() => {
            setLoading(true);
            loadLogs();
          }}
        >
          <RefreshCw className="h-3 w-3" />
          刷新
        </Button>
      </div>

      <Card className="overflow-hidden">
        {loading ? (
          <div className="p-6 space-y-2">
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-8 w-full" />
            ))}
          </div>
        ) : logs.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-8 px-4">暂无 API 日志</p>
        ) : (
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
        )}
      </Card>
    </div>
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
