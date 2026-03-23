import { useEffect, useState } from "react";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Card } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { api } from "../lib/api";
import { Save, Trash2, Check, X, Shield, Github } from "lucide-react";

const providerLabels: Record<string, string> = { github: "GitHub", linuxdo: "LinuxDo" };
const providerCallbackHelp: Record<string, string> = {
  github: "在 github.com/settings/developers → OAuth Apps 中创建应用",
  linuxdo: "在 connect.linux.do 中创建应用",
};

export function AdminPage() {
  const [tab, setTab] = useState<"dashboard" | "users" | "plugins" | "system" | "ai" | "oauth">("dashboard");
  const [pendingCount, setPendingCount] = useState(0);

  useEffect(() => {
    api.listPlugins("pending").then((l) => setPendingCount((l || []).length)).catch(() => {});
  }, [tab]);

  const tabs = [
    { key: "dashboard", label: "概览" },
    { key: "users", label: "用户" },
    { key: "plugins", label: `插件审核${pendingCount > 0 ? ` (${pendingCount})` : ""}` },
    { key: "system", label: "系统" },
    { key: "ai", label: "AI" },
    { key: "oauth", label: "OAuth" },
  ] as const;

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-lg font-semibold">系统管理</h1>
        <p className="text-xs text-muted-foreground mt-0.5">用户、插件审核、系统配置</p>
      </div>

      <div className="flex border rounded-lg overflow-hidden w-fit">
        {tabs.map((t) => (
          <button key={t.key} className={`px-3 py-1.5 text-xs cursor-pointer ${tab === t.key ? "bg-secondary font-medium" : "text-muted-foreground"}`}
            onClick={() => setTab(t.key)}>{t.label}</button>
        ))}
      </div>

      {tab === "dashboard" && <DashboardTab />}
      {tab === "users" && <UsersTab />}
      {tab === "plugins" && <PluginReviewTab />}
      {tab === "system" && <SystemTab />}
      {tab === "ai" && <AITab />}
      {tab === "oauth" && <OAuthTab />}
    </div>
  );
}

// ==================== Dashboard ====================

function DashboardTab() {
  const [stats, setStats] = useState<any>(null);
  useEffect(() => {
    api.adminStats().then(setStats).catch(() => {});
    const t = setInterval(() => api.adminStats().then(setStats).catch(() => {}), 10000);
    return () => clearInterval(t);
  }, []);
  if (!stats) return null;

  const items = [
    { label: "用户", value: stats.total_users, sub: `${stats.active_users} 活跃` },
    { label: "Bot", value: stats.total_bots, sub: `${stats.online_bots} 在线${stats.expired_bots > 0 ? ` / ${stats.expired_bots} 过期` : ""}` },
    { label: "渠道", value: stats.total_channels },
    { label: "WebSocket", value: stats.connected_ws, sub: "在线连接" },
    { label: "总消息", value: stats.total_messages.toLocaleString(), sub: `${stats.inbound_messages.toLocaleString()} 入 / ${stats.outbound_messages.toLocaleString()} 出` },
  ];

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
      {items.map((item) => (
        <div key={item.label} className="p-4 rounded-lg border bg-card text-center">
          <p className="text-2xl font-bold">{item.value}</p>
          <p className="text-xs text-muted-foreground">{item.label}</p>
          {item.sub && <p className="text-[10px] text-muted-foreground mt-0.5">{item.sub}</p>}
        </div>
      ))}
    </div>
  );
}

// ==================== Users ====================

function UsersTab() {
  const [users, setUsers] = useState<any[]>([]);
  const [showCreate, setShowCreate] = useState(false);
  const [newUsername, setNewUsername] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [newRole, setNewRole] = useState("member");
  const [error, setError] = useState("");
  const [resetTarget, setResetTarget] = useState<string | null>(null);
  const [resetPwd, setResetPwd] = useState("");

  async function load() { try { setUsers(await api.listUsers() || []); } catch {} }
  useEffect(() => { load(); }, []);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault(); setError("");
    if (!newUsername.trim() || newPassword.length < 8) { setError("用户名必填，密码至少 8 位"); return; }
    try {
      await api.createUser({ username: newUsername.trim(), password: newPassword, role: newRole });
      setNewUsername(""); setNewPassword(""); setShowCreate(false); load();
    } catch (err: any) { setError(err.message); }
  }

  async function handleToggleRole(user: any) {
    const r = user.role === "admin" ? "member" : "admin";
    if (!confirm(`将 ${user.username} 改为 ${r === "admin" ? "管理员" : "成员"}？`)) return;
    try { await api.updateUserRole(user.id, r); load(); } catch (err: any) { setError(err.message); }
  }

  async function handleToggleStatus(user: any) {
    const s = user.status === "active" ? "disabled" : "active";
    if (!confirm(`${s === "disabled" ? "禁用" : "启用"} ${user.username}？`)) return;
    try { await api.updateUserStatus(user.id, s); load(); } catch (err: any) { setError(err.message); }
  }

  async function handleResetPassword() {
    if (!resetTarget || resetPwd.length < 8) { setError("密码至少 8 位"); return; }
    try { await api.resetUserPassword(resetTarget, resetPwd); setResetTarget(null); setResetPwd(""); setError(""); }
    catch (err: any) { setError(err.message); }
  }

  async function handleDelete(user: any) {
    if (!confirm(`永久删除 ${user.username}？不可撤销。`)) return;
    try { await api.deleteUser(user.id); load(); } catch (err: any) { setError(err.message); }
  }

  return (
    <div className="space-y-3">
      <div className="flex justify-end">
        <Button variant="outline" size="sm" className="text-xs h-7" onClick={() => setShowCreate(!showCreate)}>
          {showCreate ? "取消" : "创建用户"}
        </Button>
      </div>
      {error && <p className="text-[10px] text-destructive">{error}</p>}

      {showCreate && (
        <form onSubmit={handleCreate} className="p-3 rounded-lg border bg-card space-y-2">
          <div className="flex gap-2">
            <Input placeholder="用户名" value={newUsername} onChange={(e) => setNewUsername(e.target.value)} className="h-7 text-xs" />
            <Input type="password" placeholder="密码（至少 8 位）" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} className="h-7 text-xs" />
          </div>
          <div className="flex items-center justify-between">
            <div className="flex gap-1">
              {["member", "admin"].map((r) => (
                <button key={r} type="button" onClick={() => setNewRole(r)}
                  className={`px-2 py-0.5 text-[10px] rounded cursor-pointer ${newRole === r ? "bg-primary text-primary-foreground" : "bg-secondary"}`}>
                  {r === "admin" ? "管理员" : "成员"}
                </button>
              ))}
            </div>
            <Button type="submit" size="sm" className="h-7 text-xs">创建</Button>
          </div>
        </form>
      )}

      <div className="space-y-1">
        {users.map((u) => (
          <div key={u.id} className="flex items-center justify-between p-2.5 rounded-lg border bg-card">
            <div className="flex items-center gap-2">
              <div className="w-7 h-7 rounded-full bg-secondary flex items-center justify-center text-[10px] font-medium">
                {u.username.charAt(0).toUpperCase()}
              </div>
              <div>
                <div className="flex items-center gap-1.5">
                  <span className="text-xs font-medium">{u.username}</span>
                  <span className={`text-[10px] px-1 rounded ${u.role === "admin" ? "bg-primary/10 text-primary" : "bg-secondary text-muted-foreground"}`}>
                    {u.role === "admin" ? "管理员" : "成员"}
                  </span>
                  {u.status === "disabled" && <span className="text-[10px] px-1 rounded bg-destructive/10 text-destructive">已禁用</span>}
                </div>
                {u.email && <p className="text-[10px] text-muted-foreground">{u.email}</p>}
              </div>
            </div>
            <div className="flex items-center gap-1">
              <button onClick={() => handleToggleRole(u)} className="text-[10px] text-muted-foreground hover:text-foreground px-1.5 py-0.5 rounded hover:bg-secondary cursor-pointer">{u.role === "admin" ? "降级" : "升级"}</button>
              <button onClick={() => handleToggleStatus(u)} className="text-[10px] text-muted-foreground hover:text-foreground px-1.5 py-0.5 rounded hover:bg-secondary cursor-pointer">{u.status === "active" ? "禁用" : "启用"}</button>
              <button onClick={() => { setResetTarget(u.id); setResetPwd(""); }} className="text-[10px] text-muted-foreground hover:text-foreground px-1.5 py-0.5 rounded hover:bg-secondary cursor-pointer">重置密码</button>
              <button onClick={() => handleDelete(u)} className="text-[10px] text-destructive px-1.5 py-0.5 rounded hover:bg-destructive/10 cursor-pointer">删除</button>
            </div>
          </div>
        ))}
      </div>

      {resetTarget && (
        <div className="p-3 rounded-lg border bg-card space-y-2">
          <p className="text-xs font-medium">重置密码 — {users.find((u) => u.id === resetTarget)?.username}</p>
          <div className="flex gap-2">
            <Input type="password" placeholder="新密码（至少 8 位）" value={resetPwd} onChange={(e) => setResetPwd(e.target.value)} className="h-7 text-xs" autoFocus />
            <Button size="sm" className="h-7 text-xs" onClick={handleResetPassword}>确认</Button>
            <Button size="sm" variant="ghost" className="h-7 text-xs" onClick={() => setResetTarget(null)}>取消</Button>
          </div>
        </div>
      )}
    </div>
  );
}

// ==================== System ====================

function SystemTab() {
  const [info, setInfo] = useState<any>(null);
  useEffect(() => { api.info().then(setInfo).catch(() => {}); }, []);
  if (!info) return null;

  return (
    <div className="space-y-1.5">
      {[
        { label: "AI 服务", enabled: info.ai },
        { label: "对象存储 (MinIO)", enabled: info.storage },
      ].map((item) => (
        <div key={item.label} className="flex items-center justify-between text-sm p-3 rounded-lg border bg-card">
          <span>{item.label}</span>
          <span className={`text-xs px-2 py-0.5 rounded ${item.enabled ? "bg-primary/10 text-primary" : "bg-muted text-muted-foreground"}`}>
            {item.enabled ? "已启用" : "未配置"}
          </span>
        </div>
      ))}
    </div>
  );
}

// ==================== AI ====================

function AITab() {
  const [config, setConfig] = useState<any>(null);
  const [baseUrl, setBaseUrl] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [model, setModel] = useState("");
  const [systemPrompt, setSystemPrompt] = useState("");
  const [maxHistory, setMaxHistory] = useState(20);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function load() {
    try {
      const d = await api.getAIConfig(); setConfig(d);
      setBaseUrl(d.base_url || ""); setModel(d.model || "");
      setSystemPrompt(d.system_prompt || ""); setMaxHistory(parseInt(d.max_history) || 20); setApiKey("");
    } catch {}
  }
  useEffect(() => { load(); }, []);
  if (!config) return null;
  const configured = config.enabled === "true";

  async function handleSave() {
    setSaving(true); setError("");
    try {
      let url = baseUrl.replace(/\/+$/, "");
      if (url && !url.endsWith("/v1")) url += "/v1";
      setBaseUrl(url);
      await api.setAIConfig({ base_url: url, api_key: apiKey || undefined, model: model || undefined, system_prompt: systemPrompt, max_history: String(maxHistory || 20) });
      load();
    } catch (err: any) { setError(err.message); }
    setSaving(false);
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-xs text-muted-foreground">配置后渠道可选择「内置」模式，无需单独填写 API Key</p>
        {configured && <Button variant="ghost" size="sm" onClick={async () => { if (confirm("删除全局 AI 配置？")) { await api.deleteAIConfig(); load(); } }}><Trash2 className="w-3.5 h-3.5 text-destructive" /></Button>}
      </div>
      <Input placeholder="https://api.openai.com/v1" value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} className="h-8 text-xs font-mono" />
      <div className="flex gap-2">
        <Input type="password" placeholder={configured ? `已配置 (${config.api_key})，留空不变` : "API Key"} value={apiKey} onChange={(e) => setApiKey(e.target.value)} className="h-8 text-xs font-mono" />
        <Input placeholder="模型名称" value={model} onChange={(e) => setModel(e.target.value)} className="h-8 text-xs font-mono w-40" />
      </div>
      <textarea placeholder="默认 System Prompt" value={systemPrompt} onChange={(e) => setSystemPrompt(e.target.value)} rows={3}
        className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:border-ring resize-none" />
      <div className="flex items-center gap-2">
        <label className="text-[10px] text-muted-foreground">上下文消息数</label>
        <Input type="number" value={maxHistory} onChange={(e) => setMaxHistory(parseInt(e.target.value) || 20)} className="h-8 text-xs w-20" min={1} max={100} />
      </div>
      {error && <p className="text-[10px] text-destructive">{error}</p>}
      <div className="flex justify-end"><Button size="sm" onClick={handleSave} disabled={saving}>保存</Button></div>
    </div>
  );
}

// ==================== OAuth ====================

function OAuthTab() {
  const [config, setConfig] = useState<Record<string, any> | null>(null);
  const [error, setError] = useState("");
  async function loadConfig() { try { setConfig(await api.getOAuthConfig()); } catch {} }
  useEffect(() => { loadConfig(); }, []);
  if (!config) return null;
  const callbackBase = window.location.origin + "/api/auth/oauth/";

  return (
    <div className="space-y-3">
      <p className="text-xs text-muted-foreground">DB 配置优先于环境变量，无需重启服务。</p>
      {error && <p className="text-xs text-destructive">{error}</p>}
      {Object.keys(providerLabels).map((name) => (
        <OAuthProviderForm key={name} name={name} label={providerLabels[name]} config={config[name]}
          callbackURL={callbackBase + name + "/callback"} help={providerCallbackHelp[name]}
          onSaved={loadConfig} onError={setError} />
      ))}
    </div>
  );
}

function OAuthProviderForm({ name, label, config, callbackURL, help, onSaved, onError }: {
  name: string; label: string; config: any; callbackURL: string; help: string; onSaved: () => void; onError: (msg: string) => void;
}) {
  const [clientId, setClientId] = useState(config?.client_id || "");
  const [clientSecret, setClientSecret] = useState("");
  const [saving, setSaving] = useState(false);
  useEffect(() => { setClientId(config?.client_id || ""); setClientSecret(""); }, [config]);

  async function handleSave() {
    if (!clientId.trim()) { onError("Client ID 不能为空"); return; }
    setSaving(true); onError("");
    try { await api.setOAuthConfig(name, { client_id: clientId.trim(), client_secret: clientSecret }); onSaved(); }
    catch (err: any) { onError(err.message); }
    setSaving(false);
  }

  const source = config?.source;
  const enabled = config?.enabled;

  return (
    <div className="space-y-2 p-3 rounded-lg border bg-card">
      <div className="flex items-center justify-between">
        <div>
          <span className="text-sm font-medium">{label}</span>
          {enabled && <span className={`ml-2 text-[10px] px-1.5 py-0.5 rounded ${source === "db" ? "bg-primary/10 text-primary" : "bg-muted text-muted-foreground"}`}>{source === "db" ? "数据库" : "环境变量"}</span>}
        </div>
        {source === "db" && (
          <Button variant="ghost" size="sm" onClick={async () => { if (confirm(`删除 ${label} OAuth？`)) { onError(""); try { await api.deleteOAuthConfig(name); onSaved(); } catch (e: any) { onError(e.message); } } }}>
            <Trash2 className="w-3.5 h-3.5 text-destructive" />
          </Button>
        )}
      </div>
      <Input placeholder="Client ID" value={clientId} onChange={(e) => setClientId(e.target.value)} className="h-8 text-xs font-mono" />
      <Input type="password" placeholder={enabled ? "Client Secret（留空不变）" : "Client Secret"} value={clientSecret} onChange={(e) => setClientSecret(e.target.value)} className="h-8 text-xs font-mono" />
      <div className="flex items-center justify-between">
        <div className="text-[10px] text-muted-foreground space-y-0.5">
          <p>回调：<code className="select-all">{callbackURL}</code></p>
          <p>{help}</p>
        </div>
        <Button size="sm" onClick={handleSave} disabled={saving}><Save className="w-3.5 h-3.5 mr-1" /> 保存</Button>
      </div>
    </div>
  );
}

// ==================== Plugin Review ====================

function PluginReviewTab() {
  const [plugins, setPlugins] = useState<any[]>([]);
  const [selected, setSelected] = useState<any>(null);
  const [detail, setDetail] = useState<any>(null);
  const [rejectReason, setRejectReason] = useState("");
  const [showReject, setShowReject] = useState(false);
  const [filter, setFilter] = useState<"pending" | "approved" | "rejected">("pending");

  async function load() {
    try { setPlugins(await api.listPlugins(filter) || []); } catch {}
  }
  useEffect(() => { load(); }, [filter]);

  async function openDetail(plugin: any) {
    setSelected(plugin);
    setShowReject(false);
    setRejectReason("");
    try { setDetail(await api.getPlugin(plugin.id)); } catch { setDetail(plugin); }
  }

  async function handleApprove() {
    if (!selected) return;
    await api.reviewPlugin(selected.id, "approved");
    setSelected(null); setDetail(null); load();
  }

  async function handleReject() {
    if (!selected || !rejectReason.trim()) return;
    await api.reviewPlugin(selected.id, "rejected", rejectReason.trim());
    setSelected(null); setDetail(null); load();
  }

  async function handleDelete(id: string) {
    if (!confirm("永久删除？")) return;
    await api.deletePlugin(id);
    if (selected?.id === id) { setSelected(null); setDetail(null); }
    load();
  }

  // Security analysis for detail modal
  function analyzeRisks(plugin: any) {
    const grants = (plugin.grant_perms || "").split(",").filter(Boolean);
    const connect = plugin.connect_domains || "*";
    const match = plugin.match_types || "*";
    const script = plugin.script || "";
    const risks: { level: "ok" | "warn" | "danger"; text: string }[] = [];

    if (grants.includes("none")) risks.push({ level: "ok", text: "@grant none — 无副作用" });
    else if (grants.length === 0) risks.push({ level: "warn", text: "未声明 @grant — 默认全部 API 可用" });
    if (grants.includes("reply")) risks.push({ level: "warn", text: "reply() — 可向用户发消息" });
    if (grants.includes("skip")) risks.push({ level: "ok", text: "skip() — 可跳过推送" });
    if (connect === "*") risks.push({ level: "danger", text: "@connect * — 可重定向到任意域名" });
    else risks.push({ level: "ok", text: `@connect ${connect}` });
    if (match === "*") risks.push({ level: "ok", text: "@match * — 全部消息类型" });
    else risks.push({ level: "ok", text: `@match ${match}` });
    if (script.includes("while(true)") || script.includes("for(;;)")) risks.push({ level: "danger", text: "疑似死循环" });
    if (script.includes("__proto__") || script.includes("prototype")) risks.push({ level: "warn", text: "原型链操作" });

    return risks;
  }

  const riskColors: Record<string, string> = { ok: "text-primary", warn: "text-yellow-500", danger: "text-destructive" };
  const riskIcons: Record<string, string> = { ok: "✓", warn: "⚠", danger: "✕" };

  return (
    <div className="space-y-3">
      {/* Filter tabs */}
      <div className="flex gap-1">
        {(["pending", "approved", "rejected"] as const).map((f) => (
          <button key={f} onClick={() => setFilter(f)}
            className={`px-2 py-0.5 text-[10px] rounded cursor-pointer ${filter === f ? "bg-primary text-primary-foreground" : "bg-secondary"}`}>
            {f === "pending" ? "待审核" : f === "approved" ? "已通过" : "已拒绝"}
          </button>
        ))}
      </div>

      {/* Plugin list */}
      {plugins.length === 0 && (
        <p className="text-sm text-muted-foreground text-center py-8">
          {filter === "pending" ? "没有待审核的插件" : "暂无插件"}
        </p>
      )}
      <div className="space-y-1">
        {plugins.map((p) => (
          <div key={p.id} className="flex items-center justify-between p-2.5 rounded-lg border bg-card cursor-pointer hover:border-primary/50"
            onClick={() => openDetail(p)}>
            <div className="flex items-center gap-2 min-w-0">
              {p.icon && <span className="text-base">{p.icon}</span>}
              <div className="min-w-0">
                <div className="flex items-center gap-1.5">
                  <span className="text-xs font-medium">{p.name}</span>
                  <span className="text-[10px] text-muted-foreground">v{p.version}</span>
                  {p.submitter_name && <span className="text-[10px] text-muted-foreground">by {p.submitter_name}</span>}
                </div>
                <p className="text-[10px] text-muted-foreground truncate">{p.description}</p>
              </div>
            </div>
            <div className="flex items-center gap-1.5 shrink-0">
              <span className="text-[10px] text-muted-foreground">{p.install_count} 安装</span>
              <Button variant="ghost" size="sm" className="h-6" onClick={(e) => { e.stopPropagation(); handleDelete(p.id); }}>
                <Trash2 className="w-3 h-3 text-destructive" />
              </Button>
            </div>
          </div>
        ))}
      </div>

      {/* Detail modal */}
      {selected && detail && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => { setSelected(null); setDetail(null); }}>
          <div className="bg-background border rounded-xl max-w-2xl w-full mx-4 max-h-[85vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
            {/* Header */}
            <div className="p-4 border-b flex items-start justify-between">
              <div>
                <div className="flex items-center gap-2">
                  {detail.icon && <span className="text-lg">{detail.icon}</span>}
                  <span className="font-semibold">{detail.name}</span>
                  <Badge variant="outline" className="text-[10px]">v{detail.version}</Badge>
                  {detail.license && <span className="text-[10px] text-muted-foreground">{detail.license}</span>}
                </div>
                <p className="text-xs text-muted-foreground mt-0.5">{detail.description}</p>
                <div className="flex items-center gap-3 mt-1 text-[10px] text-muted-foreground">
                  <span>by {detail.author || "anonymous"}</span>
                  <span>拥有者: {detail.submitter_name}</span>
                  <span>{new Date(detail.created_at * 1000).toLocaleDateString()}</span>
                  {detail.namespace && <span className="font-mono">{detail.namespace}</span>}
                  {detail.github_url && (
                    <a href={detail.github_url} target="_blank" rel="noopener" className="text-primary hover:underline flex items-center gap-0.5">
                      <Github className="w-3 h-3" /> GitHub
                    </a>
                  )}
                  {detail.commit_hash && <span className="font-mono">{detail.commit_hash.slice(0, 7)}</span>}
                </div>
              </div>
              <button onClick={() => { setSelected(null); setDetail(null); }} className="text-muted-foreground hover:text-foreground cursor-pointer">
                <X className="w-4 h-4" />
              </button>
            </div>

            {/* Security analysis */}
            <div className="p-4 border-b space-y-1.5">
              <p className="text-xs font-medium flex items-center gap-1"><Shield className="w-3.5 h-3.5" /> 安全分析</p>
              {analyzeRisks(detail).map((r, i) => (
                <div key={i} className={`text-[11px] flex items-start gap-1.5 ${riskColors[r.level]}`}>
                  <span className="shrink-0">{riskIcons[r.level]}</span>
                  <span>{r.text}</span>
                </div>
              ))}
            </div>

            {/* Config schema */}
            {(detail.config_schema || []).length > 0 && (
              <div className="p-4 border-b">
                <p className="text-xs font-medium mb-1">配置参数</p>
                {(detail.config_schema || []).map((c: any, i: number) => (
                  <div key={i} className="text-[11px] flex items-center gap-2">
                    <code className="font-mono bg-secondary px-1 rounded">{c.name}</code>
                    <span className="text-muted-foreground">{c.type}</span>
                    {c.description && <span className="text-muted-foreground">— {c.description}</span>}
                  </div>
                ))}
              </div>
            )}

            {/* Source code */}
            <div className="border-b">
              <div className="px-4 py-2 flex items-center justify-between">
                <p className="text-xs font-medium">源码</p>
                <span className="text-[10px] text-muted-foreground">{(detail.script || "").split("\n").length} 行</span>
              </div>
              <pre className="px-4 pb-4 text-[10px] font-mono overflow-x-auto max-h-64 overflow-y-auto whitespace-pre-wrap">
                {detail.script || "加载中..."}
              </pre>
            </div>

            {/* Reject reason (if rejected) */}
            {detail.reject_reason && (
              <div className="p-4 border-b">
                <p className="text-xs text-destructive">拒绝原因：{detail.reject_reason}</p>
                {detail.reviewer_name && <p className="text-[10px] text-muted-foreground">审核人：{detail.reviewer_name}</p>}
              </div>
            )}

            {/* Actions */}
            {detail.status === "pending" && (
              <div className="p-4">
                {!showReject ? (
                  <div className="flex gap-2">
                    <Button size="sm" onClick={handleApprove} className="flex-1">
                      <Check className="w-3.5 h-3.5 mr-1" /> 通过
                    </Button>
                    <Button size="sm" variant="outline" onClick={() => setShowReject(true)} className="flex-1">
                      <X className="w-3.5 h-3.5 mr-1" /> 拒绝
                    </Button>
                  </div>
                ) : (
                  <div className="space-y-2">
                    <Input value={rejectReason} onChange={(e) => setRejectReason(e.target.value)}
                      placeholder="请输入拒绝原因..." className="h-8 text-xs" autoFocus />
                    <div className="flex gap-2">
                      <Button size="sm" variant="destructive" onClick={handleReject} disabled={!rejectReason.trim()} className="flex-1">确认拒绝</Button>
                      <Button size="sm" variant="ghost" onClick={() => { setShowReject(false); setRejectReason(""); }}>取消</Button>
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
