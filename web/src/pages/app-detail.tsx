import { useEffect, useState } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import {
  ArrowLeft, Plus, Trash2, ShieldCheck, Eye, EyeOff,
  Copy, Check, ExternalLink, Loader2, Settings, Download,
  Globe, Radio, Terminal, Shield, Zap,
} from "lucide-react";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Card } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { api } from "../lib/api";
import { useToast } from "@/hooks/use-toast";
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
    group: "Settings",
    items: [
      { key: "basic-info" as SectionKey, label: "Basic Information", icon: Settings },
      { key: "install-app" as SectionKey, label: "Install App", icon: Download },
      { key: "distribution" as SectionKey, label: "Manage Distribution", icon: Globe },
    ],
  },
  {
    group: "Features",
    items: [
      { key: "event-subscriptions" as SectionKey, label: "Event Subscriptions", icon: Radio },
      { key: "commands" as SectionKey, label: "Commands / Tools", icon: Terminal },
      { key: "oauth-permissions" as SectionKey, label: "OAuth & Permissions", icon: Shield },
    ],
  },
];

export function AppDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [app, setApp] = useState<any>(null);
  const [section, setSection] = useState<SectionKey>("basic-info");

  async function loadApp() {
    try { setApp(await api.getApp(id!)); }
    catch { navigate("/dashboard/apps"); }
  }

  useEffect(() => { loadApp(); }, [id]);

  if (!app) return null;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Link to="/dashboard/apps/my" className="text-muted-foreground hover:text-foreground">
          <ArrowLeft className="w-4 h-4" />
        </Link>
        <AppIcon icon={app.icon} iconUrl={app.icon_url} size="h-8 w-8" />
        <div className="flex-1 min-w-0">
          <h1 className="text-lg font-semibold">{app.name}</h1>
          <p className="text-xs text-muted-foreground font-mono">{app.slug}</p>
        </div>
        <Badge variant={app.status === "active" ? "default" : "outline"}>
          {app.status === "active" ? "Active" : "Draft"}
        </Badge>
      </div>

      {/* Mobile nav */}
      <div className="md:hidden">
        <select
          value={section}
          onChange={e => setSection(e.target.value as SectionKey)}
          className="w-full h-9 px-3 rounded-lg border bg-background text-sm"
        >
          {NAV_SECTIONS.flatMap(g => g.items).map(item => (
            <option key={item.key} value={item.key}>{item.label}</option>
          ))}
        </select>
      </div>

      {/* Desktop: Left nav + Right content */}
      <div className="flex gap-8">
        <nav className="hidden md:block w-52 shrink-0 space-y-6">
          {NAV_SECTIONS.map(group => (
            <div key={group.group} className="space-y-1">
              <p className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground px-2 mb-2">
                {group.group}
              </p>
              {group.items.map(item => (
                <button
                  key={item.key}
                  onClick={() => setSection(item.key)}
                  className={`w-full flex items-center gap-2 px-2 py-1.5 rounded-md text-sm cursor-pointer transition-colors ${
                    section === item.key
                      ? "bg-primary/10 text-primary font-medium"
                      : "text-muted-foreground hover:text-foreground hover:bg-muted/50"
                  }`}
                >
                  <item.icon className="h-4 w-4 shrink-0" />
                  {item.label}
                </button>
              ))}
            </div>
          ))}
        </nav>

        <div className="flex-1 min-w-0">
          {section === "basic-info" && <BasicInfoSection app={app} onUpdate={loadApp} />}
          {section === "install-app" && <InstallAppSection appId={id!} />}
          {section === "distribution" && <DistributionSection app={app} onUpdate={loadApp} />}
          {section === "event-subscriptions" && <EventSubscriptionsSection app={app} onUpdate={loadApp} />}
          {section === "commands" && <ToolsEditor app={app} onUpdate={loadApp} />}
          {section === "oauth-permissions" && <OAuthPermissionsSection app={app} onUpdate={loadApp} />}
        </div>
      </div>
    </div>
  );
}

// ==================== Basic Information (merged Settings + Credentials) ====================

function BasicInfoSection({ app, onUpdate }: { app: any; onUpdate: () => void }) {
  const navigate = useNavigate();
  const [form, setForm] = useState({
    name: app.name || "",
    description: app.description || "",
    icon: app.icon || "",
    homepage: app.homepage || "",
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setError(""); setSuccess(""); setSaving(true);
    try {
      await api.updateApp(app.id, form);
      setSuccess("已保存");
      onUpdate();
    } catch (err: any) { setError(err.message); }
    setSaving(false);
  }

  async function handleDelete() {
    if (!confirm("确定删除此 App？所有安装也将被移除。")) return;
    try { await api.deleteApp(app.id); navigate("/dashboard/apps"); } catch {}
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-base font-semibold">Basic Information</h2>
        <p className="text-sm text-muted-foreground mt-1">应用的基本信息和凭证。</p>
      </div>

      {/* Display Information */}
      <Card className="space-y-3">
        <h3 className="text-sm font-medium">Display Information</h3>
        <form onSubmit={handleSave} className="space-y-2">
          <Input placeholder="名称" value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} className="h-8 text-xs" />
          <Input placeholder="描述" value={form.description} onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))} className="h-8 text-xs" />
          <Input placeholder="图标 (emoji 或 URL)" value={form.icon} onChange={(e) => setForm((f) => ({ ...f, icon: e.target.value }))} className="h-8 text-xs" />
          <Input placeholder="主页 URL" value={form.homepage} onChange={(e) => setForm((f) => ({ ...f, homepage: e.target.value }))} className="h-8 text-xs" />
          <div className="flex items-center justify-between">
            <div>
              {error && <span className="text-xs text-destructive">{error}</span>}
              {success && <span className="text-xs text-primary">{success}</span>}
            </div>
            <Button type="submit" size="sm" disabled={saving}>{saving ? "..." : "保存"}</Button>
          </div>
        </form>
      </Card>

      {/* App Credentials */}
      <Card className="space-y-4">
        <h3 className="text-sm font-medium">App Credentials</h3>
        <p className="text-xs text-muted-foreground">这些凭证用于你的 App 与 Hub 之间的安全通信。请妥善保管，不要泄露。</p>
        {app.client_secret && (
          <SecretField label="Client Secret" value={app.client_secret} description="用于 OAuth 流程中验证 App 身份" />
        )}
        {app.signing_secret && (
          <SecretField label="Signing Secret" value={app.signing_secret} description="Hub 使用此密钥对推送事件签名，App 用它验证请求来源" />
        )}
        {!app.client_secret && !app.signing_secret && (
          <p className="text-xs text-muted-foreground italic">凭证仅对 App 所有者可见。</p>
        )}
      </Card>

      {/* Delete App */}
      <Card className="space-y-3">
        <h3 className="text-sm font-medium text-destructive">Delete App</h3>
        <p className="text-xs text-muted-foreground">删除后所有安装也将被移除，此操作不可撤销。</p>
        <Button variant="destructive" size="sm" onClick={handleDelete}>
          <Trash2 className="w-3.5 h-3.5 mr-1" /> 删除 App
        </Button>
      </Card>
    </div>
  );
}

function SecretField({ label, value, description }: { label: string; value: string; description?: string }) {
  const [show, setShow] = useState(false);
  const [copied, setCopied] = useState(false);
  const masked = value ? value.slice(0, 8) + "..." + value.slice(-4) : "---";

  function handleCopy() {
    navigator.clipboard.writeText(value).then(() => { setCopied(true); setTimeout(() => setCopied(false), 2000); });
  }

  return (
    <div className="space-y-1">
      <p className="text-xs font-medium">{label}</p>
      {description && <p className="text-[10px] text-muted-foreground">{description}</p>}
      <div className="flex items-center gap-2 p-2 rounded-lg border bg-background">
        <code className="text-xs font-mono flex-1 break-all">{show ? value : masked}</code>
        <button onClick={() => setShow(!show)} className="cursor-pointer text-muted-foreground hover:text-foreground">
          {show ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
        </button>
        <button onClick={handleCopy} className="cursor-pointer text-muted-foreground hover:text-foreground">
          {copied ? <Check className="w-3.5 h-3.5 text-primary" /> : <Copy className="w-3.5 h-3.5" />}
        </button>
      </div>
    </div>
  );
}

// ==================== Install App ====================

function InstallAppSection({ appId }: { appId: string }) {
  const [installations, setInstallations] = useState<any[]>([]);
  const { toast } = useToast();

  async function load() {
    try { setInstallations((await api.listInstallations(appId)) || []); } catch {}
  }

  useEffect(() => { load(); }, [appId]);

  async function handleDelete(instId: string) {
    if (!confirm("确定卸载此安装？")) return;
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
      <div>
        <h2 className="text-base font-semibold">Install App</h2>
        <p className="text-sm text-muted-foreground mt-1">所有安装了此 App 的账号。每个安装实例有独立的 app_token 和 handle。</p>
      </div>

      {installations.length === 0 && (
        <p className="text-center text-sm text-muted-foreground py-8">暂无安装</p>
      )}

      <div className="space-y-2">
        {installations.map((ins) => (
          <Card key={ins.id} className="flex items-center justify-between">
            <div className="space-y-0.5">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium">{ins.bot_name || ins.bot_id}</span>
                {ins.handle && <Badge variant="outline" className="text-xs font-mono">@{ins.handle}</Badge>}
              </div>
              <p className="text-[10px] text-muted-foreground font-mono">{ins.id}</p>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant={ins.enabled ? "default" : "outline"}>
                {ins.enabled ? "启用" : "禁用"}
              </Badge>
              <Button variant="ghost" size="sm" className="h-7 text-xs text-destructive" onClick={() => handleDelete(ins.id)}>
                <Trash2 className="w-3 h-3" />
              </Button>
            </div>
          </Card>
        ))}
      </div>
    </div>
  );
}

// ==================== Manage Distribution ====================

function DistributionSection({ app, onUpdate }: { app: any; onUpdate: () => void }) {
  const [loading, setLoading] = useState(false);

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-base font-semibold">Manage Distribution</h2>
        <p className="text-sm text-muted-foreground mt-1">管理应用的上架状态，上架后其他用户可以搜索并安装。</p>
      </div>

      <Card className="space-y-4">
        <h3 className="text-sm font-medium">App Marketplace</h3>

        {app.listed ? (
          <div className="flex items-center gap-2">
            <Badge variant="default">已上架</Badge>
            <span className="text-xs text-muted-foreground">你的应用已在应用市场中展示。</span>
          </div>
        ) : app.listing_status === "pending" ? (
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <Badge variant="outline">审核中</Badge>
              <span className="text-xs text-muted-foreground">上架申请已提交，等待管理员审核。</span>
            </div>
          </div>
        ) : app.listing_status === "rejected" ? (
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <Badge variant="destructive">已拒绝</Badge>
              {app.listing_reject_reason && (
                <span className="text-xs text-destructive">原因：{app.listing_reject_reason}</span>
              )}
            </div>
            <Button size="sm" variant="outline" disabled={loading} onClick={async () => {
              setLoading(true);
              try { await api.requestListing(app.id); onUpdate(); } catch {}
              setLoading(false);
            }}>{loading ? "..." : "重新申请"}</Button>
          </div>
        ) : (
          <div className="space-y-3">
            <p className="text-xs text-muted-foreground">你的应用尚未上架。上架后其他用户可以搜索并安装。</p>
            <Button size="sm" variant="outline" disabled={loading} onClick={async () => {
              setLoading(true);
              try { await api.requestListing(app.id); onUpdate(); } catch {}
              setLoading(false);
            }}>{loading ? "..." : "申请上架"}</Button>
          </div>
        )}
      </Card>
    </div>
  );
}

// ==================== Event Subscriptions ====================

function EventSubscriptionsSection({ app, onUpdate }: { app: any; onUpdate: () => void }) {
  const [requestUrl, setRequestUrl] = useState(app.request_url || "");
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
      await api.updateApp(app.id, { request_url: requestUrl, events });
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
      if (requestUrl !== (app.request_url || "")) {
        await api.updateApp(app.id, { request_url: requestUrl });
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
        <h2 className="text-base font-semibold">Event Subscriptions</h2>
        <p className="text-sm text-muted-foreground mt-1">配置事件推送 URL 和订阅的事件类型。</p>
      </div>

      <Card className="space-y-4">
        <h3 className="text-sm font-medium">Request URL</h3>
        <p className="text-xs text-muted-foreground">Hub 会将事件推送到此 URL，payload 中包含 installation_id 和 bot_id 以区分来源。</p>
        <div className="flex gap-2">
          <Input
            placeholder="https://your-app.example.com/webhook"
            value={requestUrl}
            onChange={(e) => setRequestUrl(e.target.value)}
            className="h-8 text-xs font-mono flex-1"
          />
          <Button size="sm" variant="outline" onClick={handleVerify} disabled={verifying || !requestUrl.trim()} className="h-8">
            {verifying ? <Loader2 className="h-3 w-3 animate-spin" /> : <ExternalLink className="h-3 w-3 mr-1" />}
            验证
          </Button>
        </div>
        {app.url_verified && (
          <div className="flex items-center gap-1 text-xs text-primary">
            <ShieldCheck className="w-3 h-3" /> URL 已验证
          </div>
        )}
      </Card>

      <Card className="space-y-4">
        <h3 className="text-sm font-medium">Subscribe to Events</h3>
        <div className="grid grid-cols-2 gap-2">
          {EVENT_TYPES.map((et) => (
            <label key={et.key} className="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" checked={events.includes(et.key)} onChange={() => toggleEvent(et.key)} className="w-3.5 h-3.5 accent-primary" />
              <span className="text-xs">{et.label}</span>
              <span className="text-xs text-muted-foreground font-mono">{et.key}</span>
            </label>
          ))}
        </div>
        <Button size="sm" onClick={handleSave} disabled={saving}>{saving ? "..." : "保存"}</Button>
      </Card>
    </div>
  );
}

// ==================== Commands / Tools ====================

function ToolsEditor({ app, onUpdate }: { app: any; onUpdate: () => void }) {
  const [tools, setTools] = useState<{ name: string; description: string; command: string; parameters: string }[]>(
    (app.tools || []).map((t: any) => ({ ...t, parameters: t.parameters ? JSON.stringify(t.parameters, null, 2) : "" })),
  );
  const [saving, setSaving] = useState(false);

  function addTool() { setTools([...tools, { name: "", description: "", command: "", parameters: "" }]); }
  function removeTool(index: number) { setTools(tools.filter((_, i) => i !== index)); }
  function updateTool(index: number, field: string, value: string) { setTools(tools.map((t, i) => (i === index ? { ...t, [field]: value } : t))); }

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
          <h2 className="text-base font-semibold">Commands / Tools</h2>
          <p className="text-sm text-muted-foreground mt-1">定义 App 的工具和命令，用户通过 /command 触发。</p>
        </div>
        <Button variant="outline" size="sm" className="h-7 text-xs" onClick={addTool}>
          <Plus className="w-3 h-3 mr-1" /> 添加
        </Button>
      </div>

      {tools.length === 0 && (
        <p className="text-xs text-muted-foreground">暂无工具。点击右上角添加。</p>
      )}

      {tools.map((tool, i) => (
        <Card key={i} className="space-y-2">
          <div className="flex items-start gap-2">
            <div className="flex-1 space-y-1">
              <div className="flex gap-1">
                <Input placeholder="工具名（如 list_prs）" value={tool.name} onChange={(e) => updateTool(i, "name", e.target.value)} className="h-7 text-xs font-mono flex-1" />
                <Input placeholder="命令触发（如 pr）" value={tool.command} onChange={(e) => updateTool(i, "command", e.target.value)} className="h-7 text-xs font-mono w-36" />
              </div>
              <Input placeholder="描述" value={tool.description} onChange={(e) => updateTool(i, "description", e.target.value)} className="h-7 text-xs" />
              <textarea
                placeholder='参数 JSON Schema（可选）'
                value={tool.parameters}
                onChange={(e) => updateTool(i, "parameters", e.target.value)}
                rows={2}
                className="w-full rounded-md border border-input bg-transparent px-2 py-1 text-[11px] font-mono placeholder:text-muted-foreground/40 focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 resize-none"
              />
            </div>
            <button onClick={() => removeTool(i)} className="cursor-pointer mt-1"><Trash2 className="w-3.5 h-3.5 text-destructive" /></button>
          </div>
        </Card>
      ))}

      {tools.length > 0 && (
        <Button size="sm" onClick={handleSave} disabled={saving}>{saving ? "..." : "保存工具"}</Button>
      )}
    </div>
  );
}

// ==================== OAuth & Permissions ====================

function OAuthPermissionsSection({ app, onUpdate }: { app: any; onUpdate: () => void }) {
  const [scopes, setScopes] = useState<string[]>(app.scopes || []);
  const [saving, setSaving] = useState(false);

  const readScopes = SCOPES.filter(s => s.category === "read");
  const writeScopes = SCOPES.filter(s => s.category === "write");

  function toggleScope(key: string) {
    setScopes(prev => prev.includes(key) ? prev.filter(s => s !== key) : [...prev, key]);
  }

  async function handleSave() {
    setSaving(true);
    try { await api.updateApp(app.id, { scopes }); onUpdate(); } catch {}
    setSaving(false);
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-base font-semibold">OAuth & Permissions</h2>
        <p className="text-sm text-muted-foreground mt-1">管理应用通过 Bot API 调用时所需的权限范围。</p>
      </div>

      <Card className="space-y-4">
        <h3 className="text-sm font-medium">Scopes</h3>
        <p className="text-xs text-muted-foreground">定义应用能够访问和执行的操作。安装时用户将看到这些权限描述。</p>

        <div className="space-y-2">
          <p className="text-xs font-medium flex items-center gap-1.5">
            <Eye className="h-3.5 w-3.5 text-muted-foreground" /> 查看信息
          </p>
          {readScopes.map(s => (
            <label key={s.key} className="flex items-start gap-3 p-2 rounded-lg border bg-background cursor-pointer hover:bg-muted/30 transition-colors">
              <input type="checkbox" checked={scopes.includes(s.key)} onChange={() => toggleScope(s.key)} className="mt-0.5 accent-primary" />
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
          {writeScopes.map(s => (
            <label key={s.key} className="flex items-start gap-3 p-2 rounded-lg border bg-background cursor-pointer hover:bg-muted/30 transition-colors">
              <input type="checkbox" checked={scopes.includes(s.key)} onChange={() => toggleScope(s.key)} className="mt-0.5 accent-primary" />
              <div>
                <span className="text-sm font-medium">{s.label}</span>
                <span className="text-xs text-muted-foreground font-mono ml-2">{s.key}</span>
                <p className="text-xs text-muted-foreground mt-0.5">{s.description}</p>
              </div>
            </label>
          ))}
        </div>

        <Button size="sm" onClick={handleSave} disabled={saving}>{saving ? "..." : "保存更改"}</Button>
      </Card>
    </div>
  );
}
