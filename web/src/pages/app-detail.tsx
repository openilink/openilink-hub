import { useEffect, useState } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import {
  ArrowLeft, Blocks, Plus, Trash2, ShieldCheck, Eye, EyeOff,
  Copy, Check, RefreshCw, ExternalLink, Loader2,
} from "lucide-react";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Card } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { api } from "../lib/api";
import { useToast } from "@/hooks/use-toast";

const EVENT_TYPES = [
  { key: "message.text", label: "文本消息" },
  { key: "message.image", label: "图片消息" },
  { key: "message.video", label: "视频消息" },
  { key: "message.voice", label: "语音消息" },
  { key: "message.file", label: "文件消息" },
  { key: "message.location", label: "位置消息" },
  { key: "contact.added", label: "新增联系人" },
  { key: "group.join", label: "入群" },
  { key: "group.leave", label: "退群" },
];

const SCOPES = [
  { key: "messages.send", label: "发送消息" },
  { key: "messages.read", label: "读取消息" },
  { key: "contacts.read", label: "读取联系人" },
  { key: "bot.read", label: "读取 Bot 信息" },
];

type TabKey = "settings" | "credentials" | "features" | "installations";

export function AppDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [app, setApp] = useState<any>(null);
  const [tab, setTab] = useState<TabKey>("settings");

  async function loadApp() {
    try {
      setApp(await api.getApp(id!));
    } catch {
      navigate("/dashboard/apps");
    }
  }

  useEffect(() => { loadApp(); }, [id]);

  if (!app) return null;

  const tabs: { key: TabKey; label: string }[] = [
    { key: "settings", label: "基本信息" },
    { key: "credentials", label: "凭证" },
    { key: "features", label: "功能" },
    { key: "installations", label: "安装管理" },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Link to="/dashboard/apps/my" className="text-muted-foreground hover:text-foreground">
          <ArrowLeft className="w-4 h-4" />
        </Link>
        {app.icon_url ? (
          <img src={app.icon_url} alt="" className="w-8 h-8 rounded-lg object-cover" />
        ) : app.icon ? (
          <div className="w-8 h-8 rounded-lg bg-secondary flex items-center justify-center text-lg">{app.icon}</div>
        ) : (
          <div className="w-8 h-8 rounded-lg bg-secondary flex items-center justify-center">
            <Blocks className="w-4 h-4 text-muted-foreground" />
          </div>
        )}
        <div className="flex-1 min-w-0">
          <h1 className="text-lg font-semibold">{app.name}</h1>
          <p className="text-xs text-muted-foreground font-mono">{app.slug}</p>
        </div>
        <Badge variant={app.status === "active" ? "default" : "outline"}>
          {app.status === "active" ? "启用" : app.status || "草稿"}
        </Badge>
        {app.listed ? (
          <Badge variant="default">已上架</Badge>
        ) : app.listing_status === "pending" ? (
          <Badge variant="outline">审核中</Badge>
        ) : app.listing_status === "rejected" ? (
          <Badge variant="destructive">已拒绝</Badge>
        ) : null}
      </div>

      {/* Listing status */}
      {!app.listed && app.listing_status !== "pending" && (
        <ListingSection app={app} onUpdate={loadApp} />
      )}
      {app.listing_status === "pending" && (
        <Card className="bg-primary/5 border-primary/20">
          <p className="text-xs">上架申请已提交，等待管理员审核。</p>
        </Card>
      )}
      {app.listing_status === "rejected" && app.listing_reject_reason && (
        <Card className="bg-destructive/5 border-destructive/20 space-y-2">
          <p className="text-xs text-destructive">上架被拒绝：{app.listing_reject_reason}</p>
          <Button size="sm" variant="outline" onClick={async () => { await api.requestListing(app.id); loadApp(); }}>重新申请</Button>
        </Card>
      )}

      {/* Tabs */}
      <div className="flex rounded-lg border overflow-hidden w-fit">
        {tabs.map((t) => (
          <button
            key={t.key}
            className={`px-3 py-1.5 text-xs cursor-pointer ${tab === t.key ? "bg-secondary font-medium" : "text-muted-foreground"}`}
            onClick={() => setTab(t.key)}
          >
            {t.label}
          </button>
        ))}
      </div>

      {tab === "settings" && <SettingsTab app={app} onUpdate={loadApp} />}
      {tab === "credentials" && <CredentialsTab app={app} />}
      {tab === "features" && <FeaturesTab app={app} onUpdate={loadApp} />}
      {tab === "installations" && <InstallationsTab appId={id!} />}
    </div>
  );
}

// ==================== Settings Tab ====================

function SettingsTab({ app, onUpdate }: { app: any; onUpdate: () => void }) {
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
    <div className="space-y-4">
      <Card className="space-y-3">
        <h3 className="text-sm font-medium">基本信息</h3>
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

      <Card className="space-y-3">
        <h3 className="text-sm font-medium text-destructive">危险区域</h3>
        <Button variant="destructive" size="sm" onClick={handleDelete}>
          <Trash2 className="w-3.5 h-3.5 mr-1" /> 删除 App
        </Button>
      </Card>
    </div>
  );
}

// ==================== Credentials Tab ====================

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

function CredentialsTab({ app }: { app: any }) {
  return (
    <div className="space-y-4 max-w-2xl">
      <Card className="space-y-4">
        <h3 className="text-sm font-medium">App 凭证</h3>
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
    </div>
  );
}

// ==================== Features Tab ====================

function FeaturesTab({ app, onUpdate }: { app: any; onUpdate: () => void }) {
  return (
    <div className="space-y-6">
      <EventSubscriptionsSection app={app} onUpdate={onUpdate} />
      <ToolsEditor app={app} onUpdate={onUpdate} />
      <ScopesSection app={app} onUpdate={onUpdate} />
    </div>
  );
}

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
      await api.verifyAppUrl(app.id);
      toast({ title: "URL 验证成功" });
      onUpdate();
    } catch (e: any) {
      toast({ variant: "destructive", title: "验证失败", description: e.message });
    }
    setVerifying(false);
  }

  return (
    <Card className="space-y-4">
      <h3 className="text-sm font-medium">Event Subscriptions</h3>
      <p className="text-xs text-muted-foreground">Hub 会将事件推送到此 URL，payload 中包含 installation_id 和 bot_id 以区分来源。</p>

      <div className="space-y-2">
        <label className="text-xs font-medium">Request URL</label>
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
      </div>

      <div className="space-y-2">
        <label className="text-xs font-medium">订阅事件</label>
        <div className="grid grid-cols-2 gap-2">
          {EVENT_TYPES.map((et) => (
            <label key={et.key} className="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" checked={events.includes(et.key)} onChange={() => toggleEvent(et.key)} className="w-3.5 h-3.5 accent-primary" />
              <span className="text-xs">{et.label}</span>
              <span className="text-xs text-muted-foreground font-mono">{et.key}</span>
            </label>
          ))}
        </div>
      </div>

      <Button size="sm" onClick={handleSave} disabled={saving}>{saving ? "..." : "保存"}</Button>
    </Card>
  );
}

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
    <Card className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-medium">Commands / Tools</h3>
          <p className="text-xs text-muted-foreground mt-0.5">定义 App 的工具和命令，用户通过 /command 触发。</p>
        </div>
        <Button variant="outline" size="sm" className="h-7 text-xs" onClick={addTool}>
          <Plus className="w-3 h-3 mr-1" /> 添加
        </Button>
      </div>
      {tools.length === 0 && (
        <p className="text-xs text-muted-foreground">暂无工具。</p>
      )}
      {tools.map((tool, i) => (
        <div key={i} className="flex items-start gap-2 p-2 rounded-lg border bg-background">
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
      ))}
      {tools.length > 0 && (
        <Button size="sm" onClick={handleSave} disabled={saving}>{saving ? "..." : "保存工具"}</Button>
      )}
    </Card>
  );
}

function ScopesSection({ app, onUpdate }: { app: any; onUpdate: () => void }) {
  const [scopes, setScopes] = useState<string[]>(app.scopes || []);
  const [saving, setSaving] = useState(false);

  function toggleScope(key: string) {
    setScopes((prev) => (prev.includes(key) ? prev.filter((s) => s !== key) : [...prev, key]));
  }

  async function handleSave() {
    setSaving(true);
    try { await api.updateApp(app.id, { scopes }); onUpdate(); } catch {}
    setSaving(false);
  }

  return (
    <Card className="space-y-3">
      <h3 className="text-sm font-medium">OAuth Permissions</h3>
      <p className="text-xs text-muted-foreground">App 通过 Bot API 调用时需要的权限范围。</p>
      <div className="grid grid-cols-2 gap-2">
        {SCOPES.map((s) => (
          <label key={s.key} className="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" checked={scopes.includes(s.key)} onChange={() => toggleScope(s.key)} className="w-3.5 h-3.5 accent-primary" />
            <span className="text-xs">{s.label}</span>
            <span className="text-xs text-muted-foreground font-mono">{s.key}</span>
          </label>
        ))}
      </div>
      <Button size="sm" onClick={handleSave} disabled={saving}>{saving ? "..." : "保存"}</Button>
    </Card>
  );
}

// ==================== Installations Tab ====================

function InstallationsTab({ appId }: { appId: string }) {
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
    <div className="space-y-4">
      <div>
        <h3 className="text-sm font-medium">安装实例</h3>
        <p className="text-xs text-muted-foreground mt-0.5">所有安装了此 App 的 Bot。每个安装实例有独立的 app_token 和 handle。</p>
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

// ==================== Listing Section ====================

function ListingSection({ app, onUpdate }: { app: any; onUpdate: () => void }) {
  const [loading, setLoading] = useState(false);
  return (
    <Card className="flex items-center justify-between">
      <div>
        <p className="text-xs font-medium">上架到 App 市场</p>
        <p className="text-[10px] text-muted-foreground">上架后其他用户可以搜索并安装你的 App</p>
      </div>
      <Button size="sm" variant="outline" disabled={loading} onClick={async () => {
        setLoading(true);
        try { await api.requestListing(app.id); onUpdate(); } catch {}
        setLoading(false);
      }}>{loading ? "..." : "申请上架"}</Button>
    </Card>
  );
}
