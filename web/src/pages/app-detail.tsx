import { useEffect, useState } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeft,
  Plus,
  Trash2,
  ShieldCheck,
  Eye,
  EyeOff,
  Copy,
  Check,
  ExternalLink,
  Loader2,
  Settings,
  Download,
  Globe,
  Radio,
  Terminal,
  Shield,
  Zap,
} from "lucide-react";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { api, botDisplayName } from "../lib/api";
import { useApp } from "@/hooks/use-apps";
import { queryKeys } from "@/lib/query-keys";
import { useToast } from "@/hooks/use-toast";
import { useConfirm } from "@/components/ui/confirm-dialog";
import { AppIcon } from "../components/app-icon";
import { EVENT_TYPES, SCOPES } from "../lib/constants";

type SectionKey =
  | "basic-info"
  | "install-app"
  | "distribution"
  | "event-subscriptions"
  | "commands"
  | "oauth-permissions";

const NAV_SECTIONS = [
  {
    group: "设置",
    items: [
      { key: "basic-info" as SectionKey, label: "基本信息", icon: Settings },
      { key: "install-app" as SectionKey, label: "安装管理", icon: Download },
      { key: "distribution" as SectionKey, label: "分发管理", icon: Globe },
    ],
  },
  {
    group: "功能",
    items: [
      { key: "event-subscriptions" as SectionKey, label: "事件订阅", icon: Radio },
      { key: "commands" as SectionKey, label: "命令 / 工具", icon: Terminal },
      { key: "oauth-permissions" as SectionKey, label: "OAuth 权限", icon: Shield },
    ],
  },
];

export function AppDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { data: app, isError } = useApp(id!);
  const [section, setSection] = useState<SectionKey>("basic-info");

  // Convenience: invalidate app detail cache after mutations
  const refreshApp = () => queryClient.invalidateQueries({ queryKey: queryKeys.apps.detail(id!) });

  if (isError) { navigate("/dashboard/apps"); return null; }
  if (!app) return null;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Link
          to="/dashboard/apps"
          className="text-muted-foreground hover:text-foreground"
          aria-label="返回我的应用"
        >
          <ArrowLeft className="w-4 h-4" />
        </Link>
        <AppIcon icon={app.icon} iconUrl={app.icon_url} size="h-8 w-8" />
        <div className="flex-1 min-w-0">
          <h1 className="text-2xl font-bold tracking-tight">{app.name}</h1>
          <p className="text-xs text-muted-foreground font-mono">{app.slug}</p>
        </div>
        <Badge variant={app.status === "active" ? "default" : "outline"}>
          {app.status === "active" ? "已启用" : "草稿"}
        </Badge>
        {app.registry && <Badge variant="outline">来自应用市场</Badge>}
        {app.listing === "listed" ? (
          <Badge variant="default">已上架</Badge>
        ) : app.listing === "pending" ? (
          <Badge variant="outline">审核中</Badge>
        ) : app.listing === "rejected" ? (
          <Badge variant="destructive">已拒绝</Badge>
        ) : null}
      </div>

      {/* Mobile nav */}
      <div className="md:hidden">
        <select
          value={section}
          onChange={(e) => setSection(e.target.value as SectionKey)}
          className="w-full h-9 px-3 rounded-md border bg-background text-sm"
          aria-label="选择设置页面"
        >
          {NAV_SECTIONS.flatMap((g) => g.items).map((item) => (
            <option key={item.key} value={item.key}>
              {item.label}
            </option>
          ))}
        </select>
      </div>

      {/* Desktop: Left nav + Right content */}
      <div className="flex gap-8">
        <nav className="hidden md:block w-52 shrink-0 space-y-6">
          {NAV_SECTIONS.map((group) => (
            <div key={group.group} className="space-y-1">
              <p className="text-xs font-bold uppercase tracking-widest text-muted-foreground px-2 mb-2">
                {group.group}
              </p>
              {group.items.map((item) => (
                <Button
                  key={item.key}
                  variant="ghost"
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
            </div>
          ))}
        </nav>

        <div className="flex-1 min-w-0">
          {section === "basic-info" && (
            <BasicInfoSection key={app.updated_at} app={app} onUpdate={refreshApp} />
          )}
          {section === "install-app" && <InstallAppSection appId={id!} />}
          {section === "distribution" && <DistributionSection app={app} onUpdate={refreshApp} />}
          {section === "event-subscriptions" && (
            <EventSubscriptionsSection app={app} onUpdate={refreshApp} />
          )}
          {section === "commands" && <ToolsEditor app={app} onUpdate={refreshApp} />}
          {section === "oauth-permissions" && (
            <OAuthPermissionsSection app={app} onUpdate={refreshApp} />
          )}
        </div>
      </div>
    </div>
  );
}

// ==================== Basic Information (merged Settings + Credentials) ====================

function BasicInfoSection({ app, onUpdate }: { app: any; onUpdate: () => void }) {
  const navigate = useNavigate();
  const { confirm, ConfirmDialog } = useConfirm();
  const [form, setForm] = useState({
    name: app.name || "",
    description: app.description || "",
    icon: app.icon || "",
    homepage: app.homepage || "",
    version: app.version || "",
    readme: app.readme || "",
    guide: app.guide || "",
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setSuccess("");
    setSaving(true);
    try {
      await api.updateApp(app.id, form);
      setSuccess("已保存");
      onUpdate();
    } catch (err: any) {
      setError(err.message);
    }
    setSaving(false);
  }

  async function handleDelete() {
    const ok = await confirm({
      title: "删除确认",
      description: "确定删除此 App？所有安装也将被移除。",
      confirmText: "删除",
      variant: "destructive",
    });
    if (!ok) return;
    try {
      await api.deleteApp(app.id);
      navigate("/dashboard/apps");
    } catch {}
  }

  return (
    <div className="space-y-6">
      {ConfirmDialog}
      <div>
        <h2 className="text-base font-semibold">基本信息</h2>
        <p className="text-sm text-muted-foreground mt-1">应用的基本信息和凭证。</p>
      </div>

      {/* Display Information */}
      <Card>
        <CardHeader>
          <CardTitle>展示信息</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSave} className="space-y-2">
            <Input
              placeholder="名称"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              className="h-8 text-xs"
              disabled={!!app.registry}
            />
            <Input
              placeholder="描述"
              value={form.description}
              onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
              className="h-8 text-xs"
              disabled={!!app.registry}
            />
            <Input
              placeholder="图标 (emoji 或 URL)"
              value={form.icon}
              onChange={(e) => setForm((f) => ({ ...f, icon: e.target.value }))}
              className="h-8 text-xs"
              disabled={!!app.registry}
            />
            <Input
              placeholder="主页 URL"
              value={form.homepage}
              onChange={(e) => setForm((f) => ({ ...f, homepage: e.target.value }))}
              className="h-8 text-xs"
              disabled={!!app.registry}
            />
            <Input
              placeholder="版本号 (如 1.0.0)"
              value={form.version}
              onChange={(e) => setForm((f) => ({ ...f, version: e.target.value }))}
              className="h-8 text-xs"
              disabled={!!app.registry}
            />
            <textarea
              placeholder="说明文档 (Readme)"
              value={form.readme}
              onChange={(e) => setForm((f) => ({ ...f, readme: e.target.value }))}
              className="w-full min-h-[80px] px-3 py-2 rounded-md border bg-background text-xs font-mono resize-y"
              disabled={!!app.registry}
            />
            <textarea
              placeholder="使用指南 (Guide)"
              value={form.guide}
              onChange={(e) => setForm((f) => ({ ...f, guide: e.target.value }))}
              className="w-full min-h-[80px] px-3 py-2 rounded-md border bg-background text-xs font-mono resize-y"
              disabled={!!app.registry}
            />
            {!app.registry ? (
              <div className="flex items-center justify-between">
                <div>
                  {error ? <span className="text-xs text-destructive">{error}</span> : null}
                  {success ? <span className="text-xs text-primary">{success}</span> : null}
                </div>
                <Button type="submit" size="sm" disabled={saving}>
                  {saving ? "..." : "保存"}
                </Button>
              </div>
            ) : null}
          </form>
        </CardContent>
      </Card>

      {/* Registry badge */}
      {app.registry ? (
        <Card>
          <CardContent>
            <div className="flex items-center gap-2">
              <Badge variant="outline">来自应用市场</Badge>
              <span className="text-xs text-muted-foreground">
                此应用来自应用市场 Registry，配置不可编辑。
              </span>
            </div>
          </CardContent>
        </Card>
      ) : null}

      {/* Integration Token Guide */}
      {app.registry === "builtin" ? <IntegrationTokenGuide app={app} /> : null}

      {/* Readme */}
      {app.readme ? (
        <Card>
          <CardHeader>
            <CardTitle>说明文档</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-sm text-muted-foreground whitespace-pre-wrap font-mono leading-relaxed">
              {app.readme
                .replace(/\{hub_url\}/g, window.location.origin)
                .replace(/\{your_token\}/g, "<your_token>")}
            </div>
          </CardContent>
        </Card>
      ) : null}

      {app.registry === "builtin" && !app.readme ? (
        <Card>
          <CardHeader>
            <CardTitle>使用说明</CardTitle>
            <CardDescription>
              此应用为 Integration 类型，使用 Token 进行 API 调用。请在安装管理中查看 Token。
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-2 text-xs font-mono text-muted-foreground">
            <p className="font-sans text-xs font-medium text-foreground">HTTP 发消息</p>
            <pre className="p-2 rounded-md bg-muted/30 border overflow-x-auto whitespace-pre-wrap">{`curl -X POST ${window.location.origin}/bot/v1/message/send \\
  -H "Authorization: Bearer <your_token>" \\
  -d '{"content":"hello"}'`}</pre>
            <p className="font-sans text-xs font-medium text-foreground">WebSocket 连接</p>
            <pre className="p-2 rounded-md bg-muted/30 border overflow-x-auto whitespace-pre-wrap">{`${window.location.protocol === "https:" ? "wss" : "ws"}://${window.location.host}/bot/v1/ws?token=<your_token>`}</pre>
          </CardContent>
        </Card>
      ) : null}

      {/* App Credentials */}
      <Card>
        <CardHeader>
          <CardTitle>应用凭证</CardTitle>
          <CardDescription>
            这些凭证用于你的 App 与 Hub 之间的安全通信。请妥善保管，不要泄露。
          </CardDescription>
        </CardHeader>
        <CardContent>
          {app.webhook_secret ? (
            <SecretField
              label="Webhook Secret"
              value={app.webhook_secret}
              description="Hub 使用此密钥对推送事件签名，App 用它验证请求来源"
            />
          ) : (
            <p className="text-xs text-muted-foreground italic">凭证仅对 App 所有者可见。</p>
          )}
        </CardContent>
      </Card>

      {/* Delete App */}
      <Card>
        <CardHeader>
          <CardTitle className="text-destructive">删除应用</CardTitle>
          <CardDescription>删除后所有安装也将被移除，此操作不可撤销。</CardDescription>
        </CardHeader>
        <CardContent>
          <Button variant="destructive" size="sm" onClick={handleDelete}>
            <Trash2 className="w-3.5 h-3.5 mr-1" /> 删除 App
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}

function IntegrationTokenGuide({ app }: { app: any }) {
  const hubUrl = window.location.origin;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Integration Token 使用指南</CardTitle>
        <CardDescription>
          此应用为 Integration 类型。安装实例的 Token 可在「安装管理」中查看。
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3 text-xs">
        <div className="space-y-1">
          <p className="font-medium text-foreground">HTTP 发消息</p>
          <pre className="p-2 rounded-md bg-muted/30 border font-mono overflow-x-auto whitespace-pre-wrap">{`curl -X POST ${hubUrl}/bot/v1/message/send \\
  -H "Authorization: Bearer {token}" \\
  -d '{"content":"hello"}'`}</pre>
        </div>
        <div className="space-y-1">
          <p className="font-medium text-foreground">WebSocket 连接</p>
          <pre className="p-2 rounded-md bg-muted/30 border font-mono overflow-x-auto whitespace-pre-wrap">{`${window.location.protocol === "https:" ? "wss" : "ws"}://${window.location.host}/bot/v1/ws?token={token}`}</pre>
        </div>
      </CardContent>
    </Card>
  );
}

function SecretField({
  label,
  value,
  description,
}: {
  label: string;
  value: string;
  description?: string;
}) {
  const [show, setShow] = useState(false);
  const [copied, setCopied] = useState(false);
  const { toast } = useToast();
  const masked = value ? value.slice(0, 8) + "..." + value.slice(-4) : "---";

  function handleCopy() {
    if (!navigator.clipboard?.writeText) {
      toast({ variant: "destructive", title: "复制失败", description: "当前浏览器不支持自动复制，请手动选中复制" });
      return;
    }
    navigator.clipboard
      .writeText(value)
      .then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      })
      .catch(() => {
        toast({ variant: "destructive", title: "复制失败", description: "请手动选中复制" });
      });
  }

  return (
    <div className="space-y-1">
      <p className="text-xs font-medium">{label}</p>
      {description && <p className="text-xs text-muted-foreground">{description}</p>}
      <div className="flex items-center gap-2 p-2 rounded-md border bg-background">
        <code className="text-xs font-mono flex-1 break-all">{show ? value : masked}</code>
        <Button
          variant="ghost"
          size="icon-xs"
          onClick={() => setShow(!show)}
          aria-label={show ? "隐藏" : "显示"}
        >
          {show ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
        </Button>
        <Button variant="ghost" size="icon-xs" onClick={handleCopy} aria-label="复制">
          {copied ? (
            <Check className="w-3.5 h-3.5 text-primary" />
          ) : (
            <Copy className="w-3.5 h-3.5" />
          )}
        </Button>
      </div>
    </div>
  );
}

// ==================== Install App ====================

function InstallAppSection({ appId }: { appId: string }) {
  const [installations, setInstallations] = useState<any[]>([]);
  const [bots, setBots] = useState<any[]>([]);
  const [botId, setBotId] = useState("");
  const [handle, setHandle] = useState("");
  const [installing, setInstalling] = useState(false);
  const { toast } = useToast();
  const { confirm, ConfirmDialog } = useConfirm();

  async function load() {
    try {
      setInstallations((await api.listInstallations(appId)) || []);
    } catch {}
  }

  useEffect(() => {
    load();
    api.listBots().then((l) => {
      const items = l || [];
      setBots(items);
      if (items.length) setBotId(items[0].id);
    });
  }, [appId]);

  async function handleInstall() {
    if (!botId || !handle.trim()) return;
    setInstalling(true);
    try {
      await api.installApp(appId!, { bot_id: botId, handle: handle.trim() });
      toast({ title: "安装成功" });
      setHandle("");
      load();
    } catch (e: any) {
      toast({ variant: "destructive", title: "安装失败", description: e.message });
    }
    setInstalling(false);
  }

  async function handleDelete(instId: string) {
    const ok = await confirm({
      title: "卸载确认",
      description: "确定卸载此安装？",
      confirmText: "卸载",
      variant: "destructive",
    });
    if (!ok) return;
    try {
      await api.deleteInstallation(appId, instId);
      toast({ title: "已卸载" });
      load();
    } catch (e: any) {
      toast({ variant: "destructive", title: "卸载失败", description: e.message });
    }
  }

  return (
    <div className="space-y-6">
      {ConfirmDialog}
      <div>
        <h2 className="text-base font-semibold">安装管理</h2>
        <p className="text-sm text-muted-foreground mt-1">
          所有安装了此应用的账号。每个安装实例有独立的 app_token 和 handle。
        </p>
      </div>

      {/* Install to Bot */}
      <Card>
        <CardHeader>
          <CardTitle>安装到账号</CardTitle>
        </CardHeader>
        <CardContent>
          {bots.length === 0 ? (
            <p className="text-sm text-muted-foreground">请先创建一个账号，然后再安装应用。</p>
          ) : (
            <div className="flex gap-2 items-end">
              <div className="flex-1 space-y-1">
                <label htmlFor="install-bot-select" className="text-xs text-muted-foreground">
                  账号
                </label>
                <select
                  id="install-bot-select"
                  value={botId}
                  onChange={(e) => setBotId(e.target.value)}
                  className="w-full h-8 px-2 rounded-md border bg-background text-xs outline-none"
                >
                  {bots.map((b) => (
                    <option key={b.id} value={b.id}>
                      {botDisplayName(b)}
                    </option>
                  ))}
                </select>
              </div>
              <div className="flex-1 space-y-1">
                <label htmlFor="install-handle-input" className="text-xs text-muted-foreground">
                  Handle
                </label>
                <Input
                  id="install-handle-input"
                  value={handle}
                  onChange={(e) => setHandle(e.target.value)}
                  placeholder="如 notify"
                  className="h-8 text-xs font-mono"
                />
              </div>
              <Button
                size="sm"
                onClick={handleInstall}
                disabled={installing || !botId || !handle.trim()}
                className="h-8"
              >
                {installing && <Loader2 className="h-3 w-3 animate-spin mr-1" />}
                安装
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Existing installations */}
      {installations.length === 0 ? (
        <p className="text-center text-sm text-muted-foreground py-8">暂无安装</p>
      ) : (
        <div className="space-y-2">
          {installations.map((ins) => (
            <Card key={ins.id}>
              <CardContent className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium">{ins.bot_name || ins.bot_id}</span>
                    {ins.handle ? (
                      <Badge variant="outline" className="text-xs font-mono">
                        @{ins.handle}
                      </Badge>
                    ) : null}
                  </div>
                  <p className="text-xs text-muted-foreground font-mono">{ins.id}</p>
                </div>
                <div className="flex items-center gap-2">
                  <Badge variant={ins.enabled ? "default" : "outline"}>
                    {ins.enabled ? "启用" : "禁用"}
                  </Badge>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-7 text-xs text-destructive"
                    aria-label="卸载"
                    onClick={() => handleDelete(ins.id)}
                  >
                    <Trash2 className="w-3 h-3" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

// ==================== Manage Distribution ====================

const ACTION_LABELS: Record<string, string> = {
  request: "申请上架",
  approve: "通过",
  reject: "拒绝",
  withdraw: "撤回",
  auto_revert: "自动回退",
  admin_set: "管理员操作",
};

const ACTION_VARIANTS: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  request: "outline",
  approve: "default",
  reject: "destructive",
  withdraw: "secondary",
  auto_revert: "secondary",
  admin_set: "outline",
};

function DistributionSection({ app, onUpdate }: { app: any; onUpdate: () => void }) {
  const [loading, setLoading] = useState(false);
  const [reviews, setReviews] = useState<any[]>([]);

  useEffect(() => {
    api.listAppReviews(app.id).then(setReviews).catch(() => {});
  }, [app.id]);

  async function handleRequestListing() {
    setLoading(true);
    try {
      await api.requestListing(app.id);
      onUpdate();
      api.listAppReviews(app.id).then(setReviews).catch(() => {});
    } catch {}
    setLoading(false);
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-base font-semibold">分发管理</h2>
        <p className="text-sm text-muted-foreground mt-1">
          管理应用的上架状态，上架后其他用户可以搜索并安装。
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>应用市场</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {app.listing === "listed" ? (
            <div className="flex items-center gap-2">
              <Badge variant="default">已上架</Badge>
              <span className="text-xs text-muted-foreground">你的应用已在应用市场中展示。</span>
            </div>
          ) : app.listing === "pending" ? (
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <Badge variant="outline">审核中</Badge>
                <span className="text-xs text-muted-foreground">
                  上架申请已提交，等待管理员审核。
                </span>
              </div>
            </div>
          ) : app.listing === "rejected" ? (
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <Badge variant="destructive">已拒绝</Badge>
                {app.listing_reject_reason ? (
                  <span className="text-xs text-destructive">
                    原因：{app.listing_reject_reason}
                  </span>
                ) : null}
              </div>
              <Button size="sm" variant="outline" disabled={loading} onClick={handleRequestListing}>
                {loading ? "..." : "重新申请"}
              </Button>
            </div>
          ) : (
            <div className="space-y-3">
              <p className="text-xs text-muted-foreground">
                你的应用尚未上架。上架后其他用户可以搜索并安装。
              </p>
              <Button size="sm" variant="outline" disabled={loading} onClick={handleRequestListing}>
                {loading ? "..." : "申请上架"}
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {reviews.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>审核记录</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {reviews.map((review: any) => (
                <div key={review.id} className="flex items-start gap-3 text-sm">
                  <span className="text-xs text-muted-foreground whitespace-nowrap mt-0.5">
                    {new Date(review.created_at * 1000).toLocaleString()}
                  </span>
                  <Badge variant={ACTION_VARIANTS[review.action] || "outline"} className="shrink-0">
                    {ACTION_LABELS[review.action] || review.action}
                  </Badge>
                  {review.version && (
                    <span className="text-xs text-muted-foreground">v{review.version}</span>
                  )}
                  {review.reason && (
                    <span className="text-xs text-muted-foreground truncate">{review.reason}</span>
                  )}
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

// ==================== Event Subscriptions ====================

function EventSubscriptionsSection({ app, onUpdate }: { app: any; onUpdate: () => void }) {
  const [webhookUrl, setWebhookUrl] = useState(app.webhook_url || "");
  const [events, setEvents] = useState<string[]>(app.events || []);
  const [saving, setSaving] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const { toast } = useToast();

  function toggleEvent(key: string) {
    setEvents((prev) => (prev.includes(key) ? prev.filter((e) => e !== key) : [...prev, key]));
  }

  async function handleSave() {
    setSaving(true);
    try {
      await api.updateApp(app.id, { webhook_url: webhookUrl, events });
      toast({ title: "已保存" });
      onUpdate();
    } catch (e: any) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
    setSaving(false);
  }

  async function handleVerify() {
    setVerifying(true);
    try {
      if (webhookUrl !== (app.webhook_url || "")) {
        await api.updateApp(app.id, { webhook_url: webhookUrl });
      }
      await api.verifyAppUrl(app.id);
      toast({ title: "URL 验证成功" });
      onUpdate();
    } catch (e: any) {
      toast({ variant: "destructive", title: "验证失败", description: e.message });
    }
    setVerifying(false);
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-base font-semibold">事件订阅</h2>
        <p className="text-sm text-muted-foreground mt-1">配置事件推送 URL 和订阅的事件类型。</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>转发地址</CardTitle>
          <CardDescription>Bot 收到的消息将 POST 到此地址</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex gap-2">
            <Input
              placeholder="https://your-app.example.com/webhook"
              value={webhookUrl}
              onChange={(e) => setWebhookUrl(e.target.value)}
              className="h-8 text-xs font-mono flex-1"
            />
            <Button
              size="sm"
              variant="outline"
              onClick={handleVerify}
              disabled={verifying || !webhookUrl.trim()}
              className="h-8"
            >
              {verifying ? (
                <Loader2 className="h-3 w-3 animate-spin" />
              ) : (
                <ExternalLink className="h-3 w-3 mr-1" />
              )}
              验证
            </Button>
          </div>
          {app.url_verified ? (
            <div className="flex items-center gap-1 text-xs text-primary">
              <ShieldCheck className="w-3 h-3" /> URL 已验证
            </div>
          ) : null}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>订阅事件</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="grid grid-cols-2 gap-2">
            {EVENT_TYPES.map((et) => (
              <label key={et.key} className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={events.includes(et.key)}
                  onChange={() => toggleEvent(et.key)}
                  className="w-3.5 h-3.5 accent-primary"
                />
                <span className="text-xs">{et.label}</span>
                <span className="text-xs text-muted-foreground font-mono">{et.key}</span>
              </label>
            ))}
          </div>
          <Button size="sm" onClick={handleSave} disabled={saving}>
            {saving ? "..." : "保存"}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}

// ==================== Commands / Tools ====================

function ToolsEditor({ app, onUpdate }: { app: any; onUpdate: () => void }) {
  const [tools, setTools] = useState<
    { name: string; description: string; command: string; parameters: string }[]
  >(
    (app.tools || []).map((t: any) => ({
      ...t,
      parameters: t.parameters ? JSON.stringify(t.parameters, null, 2) : "",
    })),
  );
  const [saving, setSaving] = useState(false);

  function addTool() {
    setTools([...tools, { name: "", description: "", command: "", parameters: "" }]);
  }
  function removeTool(index: number) {
    setTools(tools.filter((_, i) => i !== index));
  }
  function updateTool(index: number, field: string, value: string) {
    setTools(tools.map((t, i) => (i === index ? { ...t, [field]: value } : t)));
  }

  async function handleSave() {
    setSaving(true);
    try {
      const payload = tools.map((t) => {
        const tool: any = { name: t.name, description: t.description };
        if (t.command) tool.command = t.command.replace(/^\//, "");
        if (t.parameters?.trim()) tool.parameters = JSON.parse(t.parameters);
        return tool;
      });
      await api.updateApp(app.id, { tools: payload });
      onUpdate();
    } catch {}
    setSaving(false);
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-base font-semibold">命令 / 工具</h2>
          <p className="text-sm text-muted-foreground mt-1">
            定义应用的工具和命令，用户通过 /command 触发。
          </p>
        </div>
        <Button variant="outline" size="sm" className="h-7 text-xs" onClick={addTool}>
          <Plus className="w-3 h-3 mr-1" /> 添加
        </Button>
      </div>

      {tools.length === 0 ? (
        <p className="text-xs text-muted-foreground">暂无工具。点击右上角添加。</p>
      ) : null}

      {tools.map((tool, i) => (
        <Card key={i}>
          <CardContent>
            <div className="flex items-start gap-2">
              <div className="flex-1 space-y-1">
                <div className="flex gap-1">
                  <Input
                    placeholder="工具名（如 list_prs）"
                    value={tool.name}
                    onChange={(e) => updateTool(i, "name", e.target.value)}
                    className="h-7 text-xs font-mono flex-1"
                  />
                  <Input
                    placeholder="命令触发（如 pr）"
                    value={tool.command}
                    onChange={(e) => updateTool(i, "command", e.target.value)}
                    className="h-7 text-xs font-mono w-36"
                  />
                </div>
                <Input
                  placeholder="描述"
                  value={tool.description}
                  onChange={(e) => updateTool(i, "description", e.target.value)}
                  className="h-7 text-xs"
                />
                <textarea
                  placeholder="参数 JSON Schema（可选）"
                  value={tool.parameters}
                  onChange={(e) => updateTool(i, "parameters", e.target.value)}
                  rows={2}
                  className="w-full rounded-md border border-input bg-transparent px-2 py-1 text-xs font-mono placeholder:text-muted-foreground/40 focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 resize-none"
                />
              </div>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                onClick={() => removeTool(i)}
                className="mt-1 text-destructive hover:text-destructive"
                aria-label="删除工具"
              >
                <Trash2 className="w-3.5 h-3.5" />
              </Button>
            </div>
          </CardContent>
        </Card>
      ))}

      {tools.length > 0 ? (
        <Button size="sm" onClick={handleSave} disabled={saving}>
          {saving ? "..." : "保存工具"}
        </Button>
      ) : null}
    </div>
  );
}

// ==================== OAuth & Permissions ====================

function OAuthPermissionsSection({ app, onUpdate }: { app: any; onUpdate: () => void }) {
  const [scopes, setScopes] = useState<string[]>(app.scopes || []);
  const [saving, setSaving] = useState(false);

  const readScopes = SCOPES.filter((s) => s.category === "read");
  const writeScopes = SCOPES.filter((s) => s.category === "write");

  function toggleScope(key: string) {
    setScopes((prev) => (prev.includes(key) ? prev.filter((s) => s !== key) : [...prev, key]));
  }

  async function handleSave() {
    setSaving(true);
    try {
      await api.updateApp(app.id, { scopes });
      onUpdate();
    } catch {}
    setSaving(false);
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-base font-semibold">OAuth 权限</h2>
        <p className="text-sm text-muted-foreground mt-1">
          管理应用通过 Bot API 调用时所需的权限范围。
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>权限范围</CardTitle>
          <CardDescription>
            定义应用能够访问和执行的操作。安装时用户将看到这些权限描述。
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <p className="text-xs font-medium flex items-center gap-1.5">
              <Eye className="h-3.5 w-3.5 text-muted-foreground" /> 查看信息
            </p>
            {readScopes.map((s) => (
              <label
                key={s.key}
                className="flex items-start gap-3 p-2 rounded-md border bg-background cursor-pointer hover:bg-muted/30 transition-colors"
              >
                <input
                  type="checkbox"
                  checked={scopes.includes(s.key)}
                  onChange={() => toggleScope(s.key)}
                  className="mt-0.5 accent-primary"
                />
                <div>
                  <span className="text-sm font-medium">{s.label}</span>
                  <span className="text-xs text-muted-foreground font-mono ml-2">{s.key}</span>
                  <p className="text-xs text-muted-foreground mt-0.5">{s.description}</p>
                </div>
              </label>
            ))}
          </div>

          <div className="space-y-2">
            <p className="text-xs font-medium flex items-center gap-1.5">
              <Zap className="h-3.5 w-3.5 text-primary" /> 执行操作
            </p>
            {writeScopes.map((s) => (
              <label
                key={s.key}
                className="flex items-start gap-3 p-2 rounded-md border bg-background cursor-pointer hover:bg-muted/30 transition-colors"
              >
                <input
                  type="checkbox"
                  checked={scopes.includes(s.key)}
                  onChange={() => toggleScope(s.key)}
                  className="mt-0.5 accent-primary"
                />
                <div>
                  <span className="text-sm font-medium">{s.label}</span>
                  <span className="text-xs text-muted-foreground font-mono ml-2">{s.key}</span>
                  <p className="text-xs text-muted-foreground mt-0.5">{s.description}</p>
                </div>
              </label>
            ))}
          </div>

          <Button size="sm" onClick={handleSave} disabled={saving}>
            {saving ? "..." : "保存更改"}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
