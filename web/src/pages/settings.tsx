import { useEffect, useState } from "react";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Card } from "../components/ui/card";
import { api } from "../lib/api";
import { Link2, Unlink, Save, Trash2 } from "lucide-react";

const providerLabels: Record<string, string> = {
  github: "GitHub",
  linuxdo: "LinuxDo",
};

const providerCallbackHelp: Record<string, string> = {
  github: "在 github.com/settings/developers → OAuth Apps 中创建应用",
  linuxdo: "在 connect.linux.do 中创建应用",
};

export function SettingsPage() {
  const [user, setUser] = useState<any>(null);
  const [oauthAccounts, setOauthAccounts] = useState<any[]>([]);
  const [oauthProviders, setOauthProviders] = useState<string[]>([]);

  async function load() {
    const [u, accounts, providers] = await Promise.all([
      api.me(),
      api.oauthAccounts(),
      api.oauthProviders(),
    ]);
    setUser(u);
    setOauthAccounts(accounts || []);
    setOauthProviders(providers.providers || []);
  }

  useEffect(() => { load(); }, []);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (params.get("oauth_bound") || params.get("oauth_error")) {
      window.history.replaceState({}, "", "/settings");
      load();
    }
  }, []);

  async function handleUnlink(provider: string) {
    if (!confirm(`确认解绑 ${providerLabels[provider] || provider}？`)) return;
    try {
      await api.unlinkOAuth(provider);
      load();
    } catch (err: any) {
      alert(err.message);
    }
  }

  function handleBind(provider: string) {
    window.location.href = `/api/me/linked-accounts/${provider}/bind`;
  }

  if (!user) return null;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-lg font-semibold">设置</h1>
        <p className="text-xs text-muted-foreground mt-0.5">账号信息、系统配置</p>
      </div>

      <Card className="space-y-3">
        <h3 className="text-sm font-medium">账号信息</h3>
        <div className="text-sm space-y-1">
          <p><span className="text-muted-foreground">用户名：</span>{user.username}</p>
          <p><span className="text-muted-foreground">显示名：</span>{user.display_name}</p>
          <p><span className="text-muted-foreground">角色：</span>{user.role}</p>
        </div>
      </Card>

      {oauthProviders.length > 0 && (
        <Card className="space-y-3">
          <h3 className="text-sm font-medium">第三方账号绑定</h3>
          <div className="space-y-2">
            {oauthProviders.map((provider) => {
              const account = oauthAccounts.find((a) => a.provider === provider);
              const linked = !!account;
              return (
                <div key={provider} className="flex items-center justify-between p-3 rounded-lg border bg-background">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-full bg-secondary flex items-center justify-center">
                      <span className="text-xs font-medium">
                        {(providerLabels[provider] || provider).charAt(0).toUpperCase()}
                      </span>
                    </div>
                    <div>
                      <p className="text-sm font-medium">{providerLabels[provider] || provider}</p>
                      <p className="text-xs text-muted-foreground">
                        {linked ? `已绑定：${account.username}` : "未绑定"}
                      </p>
                    </div>
                  </div>
                  {linked ? (
                    <Button variant="ghost" size="sm" onClick={() => handleUnlink(provider)}>
                      <Unlink className="w-3.5 h-3.5 mr-1" /> 解绑
                    </Button>
                  ) : (
                    <Button variant="outline" size="sm" onClick={() => handleBind(provider)}>
                      <Link2 className="w-3.5 h-3.5 mr-1" /> 绑定
                    </Button>
                  )}
                </div>
              );
            })}
          </div>
        </Card>
      )}

      {user.role === "admin" && <AdminDashboard />}
      {user.role === "admin" && <SystemStatusSection />}
      {user.role === "admin" && <AIConfigSection />}
      {user.role === "admin" && <OAuthConfigSection />}
    </div>
  );
}

function AdminDashboard() {
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
    <Card className="space-y-3">
      <h3 className="text-sm font-medium">管理面板</h3>
      <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
        {items.map((item) => (
          <div key={item.label} className="p-3 rounded-lg border bg-background text-center">
            <p className="text-2xl font-bold">{item.value}</p>
            <p className="text-xs text-muted-foreground">{item.label}</p>
            {item.sub && <p className="text-[10px] text-muted-foreground mt-0.5">{item.sub}</p>}
          </div>
        ))}
      </div>
    </Card>
  );
}

function SystemStatusSection() {
  const [info, setInfo] = useState<any>(null);
  useEffect(() => { api.info().then(setInfo).catch(() => {}); }, []);
  if (!info) return null;

  const items = [
    { label: "AI 服务", enabled: info.ai },
    { label: "对象存储 (MinIO)", enabled: info.storage },
  ];

  return (
    <Card className="space-y-3">
      <h3 className="text-sm font-medium">系统状态</h3>
      <div className="space-y-1.5">
        {items.map((item) => (
          <div key={item.label} className="flex items-center justify-between text-sm">
            <span>{item.label}</span>
            <span className={`text-xs px-2 py-0.5 rounded ${item.enabled ? "bg-primary/10 text-primary" : "bg-muted text-muted-foreground"}`}>
              {item.enabled ? "已启用" : "未配置"}
            </span>
          </div>
        ))}
      </div>
    </Card>
  );
}

function AIConfigSection() {
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
      const data = await api.getAIConfig();
      setConfig(data);
      setBaseUrl(data.base_url || "");
      setModel(data.model || "");
      setSystemPrompt(data.system_prompt || "");
      setMaxHistory(parseInt(data.max_history) || 20);
      setApiKey("");
    } catch { /* not admin */ }
  }

  useEffect(() => { load(); }, []);
  if (!config) return null;

  const configured = config.enabled === "true";

  async function handleSave() {
    setSaving(true);
    setError("");
    try {
      let url = baseUrl.replace(/\/+$/, "");
      if (url && !url.endsWith("/v1")) url += "/v1";
      setBaseUrl(url);
      await api.setAIConfig({
        base_url: url,
        api_key: apiKey || undefined,
        model: model || undefined,
        system_prompt: systemPrompt,
        max_history: String(maxHistory || 20),
      });
      load();
    } catch (err: any) { setError(err.message); }
    setSaving(false);
  }

  async function handleDelete() {
    if (!confirm("删除全局 AI 配置？")) return;
    await api.deleteAIConfig();
    load();
  }

  return (
    <Card className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-medium">全局 AI 配置</h3>
          <p className="text-[10px] text-muted-foreground mt-0.5">
            配置后渠道可选择「内置」模式，无需单独填写 API Key
          </p>
        </div>
        {configured && (
          <Button variant="ghost" size="sm" onClick={handleDelete}>
            <Trash2 className="w-3.5 h-3.5 text-destructive" />
          </Button>
        )}
      </div>
      <div className="space-y-2">
        <Input placeholder="https://api.openai.com/v1" value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} className="h-8 text-xs font-mono" />
        <div className="flex gap-2">
          <Input
            type="password"
            placeholder={configured ? `已配置 (${config.api_key})，留空保持不变` : "API Key"}
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            className="h-8 text-xs font-mono"
          />
          <Input placeholder="模型名称" value={model} onChange={(e) => setModel(e.target.value)} className="h-8 text-xs font-mono w-40" />
        </div>
        <textarea
          placeholder="默认系统提示词（System Prompt），渠道未设置时使用"
          value={systemPrompt}
          onChange={(e) => setSystemPrompt(e.target.value)}
          rows={3}
          className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:border-ring resize-none"
        />
        <div className="flex items-center gap-2">
          <label className="text-[10px] text-muted-foreground shrink-0">默认上下文消息数</label>
          <Input type="number" value={maxHistory} onChange={(e) => setMaxHistory(parseInt(e.target.value) || 20)} className="h-8 text-xs w-20" min={1} max={100} />
        </div>
      </div>
      {error && <p className="text-[10px] text-destructive">{error}</p>}
      <div className="flex justify-end">
        <Button size="sm" onClick={handleSave} disabled={saving}>保存</Button>
      </div>
    </Card>
  );
}

function OAuthConfigSection() {
  const [config, setConfig] = useState<Record<string, any> | null>(null);
  const [error, setError] = useState("");

  async function loadConfig() {
    try {
      const data = await api.getOAuthConfig();
      setConfig(data);
    } catch { /* not admin */ }
  }

  useEffect(() => { loadConfig(); }, []);
  if (!config) return null;

  const callbackBase = window.location.origin + "/api/auth/oauth/";

  return (
    <Card className="space-y-4">
      <div>
        <h3 className="text-sm font-medium">OAuth 配置</h3>
        <p className="text-xs text-muted-foreground mt-1">
          管理员可在此配置第三方登录，无需重启服务。DB 配置优先于环境变量。
        </p>
      </div>
      {error && <p className="text-xs text-destructive">{error}</p>}
      {Object.keys(providerLabels).map((name) => (
        <OAuthProviderForm
          key={name}
          name={name}
          label={providerLabels[name]}
          config={config[name]}
          callbackURL={callbackBase + name + "/callback"}
          help={providerCallbackHelp[name]}
          onSaved={loadConfig}
          onError={setError}
        />
      ))}
    </Card>
  );
}

function OAuthProviderForm({
  name, label, config, callbackURL, help, onSaved, onError,
}: {
  name: string; label: string; config: any; callbackURL: string;
  help: string; onSaved: () => void; onError: (msg: string) => void;
}) {
  const [clientId, setClientId] = useState(config?.client_id || "");
  const [clientSecret, setClientSecret] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setClientId(config?.client_id || "");
    setClientSecret("");
  }, [config]);

  async function handleSave() {
    if (!clientId.trim()) { onError("Client ID 不能为空"); return; }
    setSaving(true);
    onError("");
    try {
      await api.setOAuthConfig(name, { client_id: clientId.trim(), client_secret: clientSecret });
      onSaved();
    } catch (err: any) { onError(err.message); }
    setSaving(false);
  }

  async function handleDelete() {
    if (!confirm(`删除 ${label} 的 OAuth 配置？将回退到环境变量配置。`)) return;
    onError("");
    try { await api.deleteOAuthConfig(name); onSaved(); } catch (err: any) { onError(err.message); }
  }

  const source = config?.source;
  const enabled = config?.enabled;

  return (
    <div className="space-y-2 p-3 rounded-lg border bg-background">
      <div className="flex items-center justify-between">
        <div>
          <span className="text-sm font-medium">{label}</span>
          {enabled && (
            <span className={`ml-2 text-[10px] px-1.5 py-0.5 rounded ${
              source === "db" ? "bg-primary/10 text-primary" : "bg-muted text-muted-foreground"
            }`}>
              {source === "db" ? "数据库" : "环境变量"}
            </span>
          )}
        </div>
        {source === "db" && (
          <Button variant="ghost" size="sm" onClick={handleDelete}>
            <Trash2 className="w-3.5 h-3.5 text-destructive" />
          </Button>
        )}
      </div>

      <div className="space-y-1.5">
        <Input placeholder="Client ID" value={clientId} onChange={(e) => setClientId(e.target.value)} className="h-8 text-xs font-mono" />
        <Input type="password" placeholder={enabled ? "Client Secret（留空保持不变）" : "Client Secret"} value={clientSecret} onChange={(e) => setClientSecret(e.target.value)} className="h-8 text-xs font-mono" />
      </div>

      <div className="flex items-center justify-between">
        <div className="text-[10px] text-muted-foreground space-y-0.5">
          <p>回调地址：<code className="select-all">{callbackURL}</code></p>
          <p>{help}</p>
        </div>
        <Button size="sm" onClick={handleSave} disabled={saving}>
          <Save className="w-3.5 h-3.5 mr-1" /> 保存
        </Button>
      </div>
    </div>
  );
}
