import { useEffect, useState } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import { ArrowLeft, Copy, Check, Cable, Bot, Webhook, Radio, RotateCw, Trash2, Puzzle } from "lucide-react";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Badge } from "../components/ui/badge";
import { api } from "../lib/api";

export function ChannelDetailPage() {
  const { id: botId, cid: channelId } = useParams<{ id: string; cid: string }>();
  const navigate = useNavigate();
  const [channel, setChannel] = useState<any>(null);
  const [bot, setBot] = useState<any>(null);
  const [tab, setTab] = useState<"overview" | "connect" | "webhook" | "ai" | "filter" | "live">("overview");

  async function load() {
    const bots = await api.listBots();
    const b = (bots || []).find((b: any) => b.id === botId);
    setBot(b);
    const chs = await api.listChannels(botId!);
    setChannel((chs || []).find((c: any) => c.id === channelId) || null);
  }

  useEffect(() => { load(); }, [botId, channelId]);

  async function handleDelete() {
    if (!confirm("删除此渠道？")) return;
    await api.deleteChannel(botId!, channelId!);
    navigate(`/bot/${botId}`);
  }

  async function handleToggle() {
    await api.updateChannel(botId!, channelId!, { enabled: !channel.enabled });
    load();
  }

  if (!channel || !bot) return <p className="text-sm text-muted-foreground p-8">加载中...</p>;

  const tabs = [
    { key: "overview", label: "概览" },
    { key: "connect", label: "接入" },
    { key: "webhook", label: "Webhook" },
    { key: "ai", label: "AI" },
    { key: "filter", label: "过滤" },
    { key: "live", label: "监控" },
  ] as const;

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center gap-3 pb-4 border-b">
        <Link to={`/bot/${botId}`} className="text-muted-foreground hover:text-foreground">
          <ArrowLeft className="w-4 h-4" />
        </Link>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <Cable className="w-4 h-4 text-muted-foreground" />
            <h2 className="font-semibold text-sm">{channel.name}</h2>
            {channel.handle && <span className="text-xs text-muted-foreground font-mono">@{channel.handle}</span>}
          </div>
          <p className="text-xs text-muted-foreground mt-0.5">{bot.name}</p>
        </div>
        <button onClick={handleToggle}>
          <Badge variant={channel.enabled ? "default" : "outline"}>
            {channel.enabled ? "启用" : "停用"}
          </Badge>
        </button>
        <Button variant="ghost" size="sm" onClick={handleDelete}>
          <Trash2 className="w-4 h-4 text-destructive" />
        </Button>
      </div>

      {/* Tabs */}
      <div className="flex border rounded-lg overflow-hidden w-fit">
        {tabs.map((t) => (
          <button key={t.key} className={`px-3 py-1 text-xs cursor-pointer ${tab === t.key ? "bg-secondary" : "text-muted-foreground"}`} onClick={() => setTab(t.key)}>
            {t.label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      {tab === "overview" && <OverviewTab channel={channel} botId={botId!} onRefresh={load} />}
      {tab === "connect" && <ConnectTab channel={channel} />}
      {tab === "webhook" && <WebhookTab channel={channel} botId={botId!} onRefresh={load} />}
      {tab === "ai" && <AITab channel={channel} botId={botId!} onRefresh={load} />}
      {tab === "filter" && <FilterTab channel={channel} botId={botId!} onRefresh={load} />}
      {tab === "live" && <LiveTab channel={channel} />}
    </div>
  );
}

// ==================== Overview ====================

function OverviewTab({ channel, botId, onRefresh }: { channel: any; botId: string; onRefresh: () => void }) {
  const [editingName, setEditingName] = useState(false);
  const [editingHandle, setEditingHandle] = useState(false);
  const [name, setName] = useState(channel.name);
  const [handle, setHandle] = useState(channel.handle || "");
  const [pluginInfo, setPluginInfo] = useState<any>(null);

  useEffect(() => {
    if (channel.webhook_config?.plugin_id) {
      api.getPlugin(channel.webhook_config.plugin_id).then(setPluginInfo).catch(() => {});
    }
  }, [channel]);

  async function saveName() {
    await api.updateChannel(botId, channel.id, { name });
    setEditingName(false);
    onRefresh();
  }
  async function saveHandle() {
    await api.updateChannel(botId, channel.id, { handle });
    setEditingHandle(false);
    onRefresh();
  }

  const cards = [
    { label: "名称", value: editingName ? null : channel.name, action: () => setEditingName(true) },
    { label: "@提及", value: channel.handle || "（全部消息）", action: () => setEditingHandle(true) },
  ];

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
        {cards.map((c) => (
          <div key={c.label} className="p-3 rounded-lg border bg-card cursor-pointer hover:border-primary/50" onClick={c.action}>
            <p className="text-[10px] text-muted-foreground">{c.label}</p>
            <p className="text-sm font-medium mt-0.5">{c.value}</p>
          </div>
        ))}
        <CopyCard label="API Key" value={channel.api_key} />
        {channel.ai_config?.enabled && (
          <div className="p-3 rounded-lg border bg-card">
            <p className="text-[10px] text-muted-foreground">AI 回复</p>
            <p className="text-sm font-medium mt-0.5 flex items-center gap-1"><Bot className="w-3 h-3" /> {channel.ai_config.source === "builtin" ? "内置" : "自定义"}</p>
          </div>
        )}
        {channel.webhook_config?.url && (
          <div className="p-3 rounded-lg border bg-card">
            <p className="text-[10px] text-muted-foreground">Webhook</p>
            <p className="text-sm font-mono mt-0.5 truncate">{channel.webhook_config.url}</p>
          </div>
        )}
        {pluginInfo && (
          <div className="p-3 rounded-lg border bg-card">
            <p className="text-[10px] text-muted-foreground">插件</p>
            <p className="text-sm font-medium mt-0.5">{pluginInfo.icon} {pluginInfo.name} v{pluginInfo.version}</p>
          </div>
        )}
      </div>

      {editingName && (
        <form onSubmit={(e) => { e.preventDefault(); saveName(); }} className="flex gap-2 items-center">
          <Input value={name} onChange={(e) => setName(e.target.value)} className="h-8 text-sm" autoFocus />
          <Button size="sm" type="submit">保存</Button>
          <Button size="sm" variant="ghost" onClick={() => setEditingName(false)}>取消</Button>
        </form>
      )}
      {editingHandle && (
        <form onSubmit={(e) => { e.preventDefault(); saveHandle(); }} className="flex gap-2 items-center">
          <span className="text-sm">@</span>
          <Input value={handle} onChange={(e) => setHandle(e.target.value)} placeholder="留空接收全部消息" className="h-8 text-sm w-40" autoFocus />
          <Button size="sm" type="submit">保存</Button>
          <Button size="sm" variant="ghost" onClick={() => setEditingHandle(false)}>取消</Button>
        </form>
      )}
    </div>
  );
}

function CopyCard({ label, value }: { label: string; value: string }) {
  const [copied, setCopied] = useState(false);
  function copy() {
    navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }
  return (
    <div className="p-3 rounded-lg border bg-card cursor-pointer hover:border-primary/50" onClick={copy}>
      <p className="text-[10px] text-muted-foreground">{label} {copied && <Check className="w-3 h-3 inline text-primary" />}</p>
      <p className="text-xs font-mono mt-0.5 truncate">{value}</p>
    </div>
  );
}

// ==================== Connect ====================

function ConnectTab({ channel }: { channel: any }) {
  const wsProto = location.protocol === "https:" ? "wss:" : "ws:";
  const wsUrl = `${wsProto}//${location.host}/api/v1/channels/connect?key=${channel.api_key}`;
  const httpBase = `${location.origin}/api/v1/channels`;

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <CopyField label="WebSocket" value={wsUrl} />
        <CopyField label="HTTP API" value={httpBase} />
        <CopyField label="API Key" value={channel.api_key} />
      </div>

      <div className="text-xs text-muted-foreground space-y-3 p-4 rounded-lg border bg-card">
        <p className="font-medium text-foreground">接入方式</p>

        <div>
          <p className="font-medium text-foreground mt-3 mb-1">WebSocket 连接</p>
          <pre className="bg-background p-2 rounded overflow-x-auto text-[10px]">{`ws://${location.host}/api/v1/channels/connect?key=API_KEY`}</pre>
          <p className="mt-1">连接后自动收到 init 消息，收到新消息时推送 message 事件。</p>
        </div>

        <div>
          <p className="font-medium text-foreground mt-3 mb-1">HTTP API</p>
          <pre className="bg-background p-2 rounded overflow-x-auto text-[10px]">{`# 拉取消息
GET /api/v1/channels/messages?key=KEY&limit=50

# 发送消息
POST /api/v1/channels/send?key=KEY
{"text": "内容"}

# 渠道状态
GET /api/v1/channels/status?key=KEY`}</pre>
        </div>
      </div>
    </div>
  );
}

function CopyField({ label, value }: { label: string; value: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <div className="flex items-center gap-2">
      <span className="text-xs text-muted-foreground w-16 shrink-0">{label}</span>
      <code className="flex-1 text-[10px] font-mono bg-card border rounded px-2 py-1 truncate select-all">{value}</code>
      <button onClick={() => { navigator.clipboard.writeText(value); setCopied(true); setTimeout(() => setCopied(false), 2000); }} className="cursor-pointer text-muted-foreground hover:text-foreground shrink-0">
        {copied ? <Check className="w-3 h-3 text-primary" /> : <Copy className="w-3 h-3" />}
      </button>
    </div>
  );
}

// ==================== Webhook ====================

function WebhookTab({ channel, botId, onRefresh }: { channel: any; botId: string; onRefresh: () => void }) {
  const cfg = channel.webhook_config || {};
  const [url, setUrl] = useState(cfg.url || "");
  const [authType, setAuthType] = useState(cfg.auth?.type || "");
  const [authToken, setAuthToken] = useState(cfg.auth?.token || "");
  const [authName, setAuthName] = useState(cfg.auth?.name || "");
  const [authValue, setAuthValue] = useState(cfg.auth?.value || cfg.auth?.secret || "");
  const [scriptMode, setScriptMode] = useState<"plugin" | "manual">(cfg.plugin_id ? "plugin" : "manual");
  const [script, setScript] = useState(cfg.script || "");
  const [pluginId, setPluginId] = useState(cfg.plugin_id || "");
  const [pluginInfo, setPluginInfo] = useState<any>(null);
  const [plugins, setPlugins] = useState<any[]>([]);
  const [showPicker, setShowPicker] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (pluginId) api.getPlugin(pluginId).then(setPluginInfo).catch(() => setPluginInfo(null));
  }, [pluginId]);
  useEffect(() => {
    if (showPicker) api.listPlugins().then((l) => setPlugins(l || [])).catch(() => {});
  }, [showPicker]);

  async function handleSave() {
    setSaving(true); setError("");
    try {
      let auth: any = null;
      if (authType === "bearer" && authToken) auth = { type: "bearer", token: authToken };
      else if (authType === "header" && authName) auth = { type: "header", name: authName, value: authValue };
      else if (authType === "hmac" && authValue) auth = { type: "hmac", secret: authValue };
      await api.updateChannel(botId, channel.id, {
        webhook_config: {
          url, auth,
          plugin_id: scriptMode === "plugin" ? pluginId : undefined,
          script: scriptMode === "manual" ? (script || undefined) : undefined,
        },
      });
      onRefresh();
    } catch (err: any) { setError(err.message); }
    setSaving(false);
  }

  async function installPlugin(id: string) {
    try {
      const r = await api.installPluginToChannel(id, botId, channel.id);
      setPluginId(r.plugin_id); setScriptMode("plugin"); setScript(""); setShowPicker(false);
      onRefresh();
    } catch (err: any) { setError(err.message); }
  }

  return (
    <div className="space-y-3">
      <Input placeholder="https://your-server.com/webhook" value={url} onChange={(e) => setUrl(e.target.value)} className="h-8 text-xs font-mono" />

      {/* Auth */}
      <div>
        <p className="text-[10px] text-muted-foreground mb-1">认证方式</p>
        <div className="flex gap-1">
          {["", "bearer", "header", "hmac"].map((t) => (
            <button key={t} onClick={() => setAuthType(t)} className={`px-2 py-0.5 text-[10px] rounded cursor-pointer ${authType === t ? "bg-primary text-primary-foreground" : "bg-secondary"}`}>
              {t || "无"}
            </button>
          ))}
        </div>
      </div>
      {authType === "bearer" && <Input placeholder="Token" value={authToken} onChange={(e) => setAuthToken(e.target.value)} className="h-7 text-[11px] font-mono" />}
      {authType === "header" && (
        <div className="flex gap-2">
          <Input placeholder="Header 名" value={authName} onChange={(e) => setAuthName(e.target.value)} className="h-7 text-[11px] font-mono" />
          <Input placeholder="Header 值" value={authValue} onChange={(e) => setAuthValue(e.target.value)} className="h-7 text-[11px] font-mono" />
        </div>
      )}
      {authType === "hmac" && <Input placeholder="HMAC Secret" value={authValue} onChange={(e) => setAuthValue(e.target.value)} className="h-7 text-[11px] font-mono" />}

      {/* Script source */}
      <div>
        <p className="text-[10px] text-muted-foreground mb-1">脚本来源</p>
        <div className="flex gap-1">
          <button onClick={() => setScriptMode("plugin")} className={`px-2 py-0.5 text-[10px] rounded cursor-pointer ${scriptMode === "plugin" ? "bg-primary text-primary-foreground" : "bg-secondary"}`}>插件市场</button>
          <button onClick={() => setScriptMode("manual")} className={`px-2 py-0.5 text-[10px] rounded cursor-pointer ${scriptMode === "manual" ? "bg-primary text-primary-foreground" : "bg-secondary"}`}>手动脚本</button>
        </div>
      </div>

      {scriptMode === "plugin" && (
        <div className="space-y-2">
          {pluginInfo ? (
            <div className="flex items-center justify-between p-2 rounded border bg-card">
              <div className="text-xs">
                <span>{pluginInfo.icon} <strong>{pluginInfo.name}</strong> v{pluginInfo.version}</span>
                <p className="text-[10px] text-muted-foreground">{pluginInfo.description}</p>
              </div>
              <div className="flex gap-1">
                <Button variant="ghost" size="sm" className="h-6 text-[10px]" onClick={() => setShowPicker(true)}>更换</Button>
                <Button variant="ghost" size="sm" className="h-6 text-[10px] text-destructive" onClick={() => { setPluginId(""); setPluginInfo(null); setScriptMode("manual"); }}>卸载</Button>
              </div>
            </div>
          ) : (
            <Button variant="outline" size="sm" className="w-full text-xs h-7" onClick={() => setShowPicker(true)}>
              <Puzzle className="w-3 h-3 mr-1" /> 选择插件
            </Button>
          )}
          {showPicker && (
            <div className="border rounded p-2 space-y-1 max-h-48 overflow-y-auto bg-card">
              {plugins.length === 0 && <p className="text-[10px] text-muted-foreground text-center py-2">暂无可用插件</p>}
              {plugins.map((p) => (
                <button key={p.id} onClick={() => installPlugin(p.id)} className="w-full text-left p-1.5 rounded hover:bg-secondary cursor-pointer text-xs flex items-center justify-between">
                  <span>{p.icon} {p.name} <span className="text-muted-foreground">v{p.version}</span></span>
                  <span className="text-[10px] text-muted-foreground">{p.install_count} 安装</span>
                </button>
              ))}
              <button onClick={() => setShowPicker(false)} className="w-full text-center text-[10px] text-muted-foreground hover:text-primary cursor-pointer py-1">取消</button>
            </div>
          )}
        </div>
      )}

      {scriptMode === "manual" && (
        <textarea
          placeholder={`JS 中间件（可选）\nfunction onRequest(ctx) { ... }`}
          value={script}
          onChange={(e) => setScript(e.target.value)}
          rows={6}
          className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-[11px] font-mono placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:border-ring resize-none"
        />
      )}

      <div className="flex items-center justify-between">
        {error && <span className="text-[10px] text-destructive">{error}</span>}
        <div className="ml-auto"><Button size="sm" onClick={handleSave} disabled={saving}>{saving ? "..." : "保存"}</Button></div>
      </div>
    </div>
  );
}

// ==================== AI ====================

function AITab({ channel, botId, onRefresh }: { channel: any; botId: string; onRefresh: () => void }) {
  const cfg = channel.ai_config || {};
  const [enabled, setEnabled] = useState(cfg.enabled || false);
  const [source, setSource] = useState(cfg.source || "builtin");
  const [baseUrl, setBaseUrl] = useState(cfg.base_url || "");
  const [apiKey, setApiKey] = useState("");
  const [model, setModel] = useState(cfg.model || "");
  const [systemPrompt, setSystemPrompt] = useState(cfg.system_prompt || "");
  const [maxHistory, setMaxHistory] = useState(cfg.max_history || 20);
  const [saving, setSaving] = useState(false);

  async function handleSave() {
    setSaving(true);
    try {
      await api.updateChannel(botId, channel.id, {
        ai_config: { enabled, source, base_url: baseUrl, api_key: apiKey || undefined, model, system_prompt: systemPrompt, max_history: maxHistory },
      });
      onRefresh();
    } catch {}
    setSaving(false);
  }

  return (
    <div className="space-y-3">
      <label className="flex items-center gap-2 cursor-pointer">
        <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} className="w-3.5 h-3.5 accent-primary" />
        <span className="text-sm">启用 AI 自动回复</span>
      </label>

      {enabled && (
        <>
          <div className="flex gap-1">
            <button onClick={() => setSource("builtin")} className={`px-2 py-0.5 text-[10px] rounded cursor-pointer ${source === "builtin" ? "bg-primary text-primary-foreground" : "bg-secondary"}`}>内置（全局配置）</button>
            <button onClick={() => setSource("custom")} className={`px-2 py-0.5 text-[10px] rounded cursor-pointer ${source === "custom" ? "bg-primary text-primary-foreground" : "bg-secondary"}`}>自定义</button>
          </div>
          {source === "custom" && (
            <div className="space-y-2">
              <Input placeholder="https://api.openai.com/v1" value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} className="h-7 text-[11px] font-mono" />
              <div className="flex gap-2">
                <Input type="password" placeholder="API Key" value={apiKey} onChange={(e) => setApiKey(e.target.value)} className="h-7 text-[11px] font-mono" />
                <Input placeholder="模型名称" value={model} onChange={(e) => setModel(e.target.value)} className="h-7 text-[11px] font-mono w-40" />
              </div>
            </div>
          )}
          <textarea
            placeholder="系统提示词（System Prompt）"
            value={systemPrompt}
            onChange={(e) => setSystemPrompt(e.target.value)}
            rows={3}
            className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-[11px] placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:border-ring resize-none"
          />
          <div className="flex items-center gap-2">
            <label className="text-[10px] text-muted-foreground">上下文消息数</label>
            <Input type="number" value={maxHistory} onChange={(e) => setMaxHistory(parseInt(e.target.value) || 20)} className="h-7 text-xs w-20" min={1} max={100} />
          </div>
        </>
      )}

      <div className="flex justify-end">
        <Button size="sm" onClick={handleSave} disabled={saving}>{saving ? "..." : "保存"}</Button>
      </div>
    </div>
  );
}

// ==================== Filter ====================

function FilterTab({ channel, botId, onRefresh }: { channel: any; botId: string; onRefresh: () => void }) {
  const rule = channel.filter_rule || {};
  const [userIds, setUserIds] = useState((rule.user_ids || []).join(", "));
  const [keywords, setKeywords] = useState((rule.keywords || []).join(", "));
  const [msgTypes, setMsgTypes] = useState((rule.message_types || []).join(", "));
  const [saving, setSaving] = useState(false);

  async function handleSave() {
    setSaving(true);
    const parse = (s: string) => s.split(",").map((v) => v.trim()).filter(Boolean);
    try {
      await api.updateChannel(botId, channel.id, {
        filter_rule: {
          user_ids: parse(userIds),
          keywords: parse(keywords),
          message_types: parse(msgTypes),
        },
      });
      onRefresh();
    } catch {}
    setSaving(false);
  }

  return (
    <div className="space-y-3">
      <p className="text-xs text-muted-foreground">留空表示不过滤（接收所有消息）。多个值用逗号分隔。</p>
      <div className="space-y-2">
        <div>
          <label className="text-[10px] text-muted-foreground">用户 ID 白名单</label>
          <Input value={userIds} onChange={(e) => setUserIds(e.target.value)} placeholder="user1@wx, user2@wx" className="h-7 text-[11px] font-mono" />
        </div>
        <div>
          <label className="text-[10px] text-muted-foreground">关键词匹配</label>
          <Input value={keywords} onChange={(e) => setKeywords(e.target.value)} placeholder="help, urgent" className="h-7 text-[11px] font-mono" />
        </div>
        <div>
          <label className="text-[10px] text-muted-foreground">消息类型</label>
          <Input value={msgTypes} onChange={(e) => setMsgTypes(e.target.value)} placeholder="text, image, voice, video, file" className="h-7 text-[11px] font-mono" />
        </div>
      </div>
      <div className="flex justify-end">
        <Button size="sm" onClick={handleSave} disabled={saving}>{saving ? "..." : "保存"}</Button>
      </div>
    </div>
  );
}

// ==================== Live ====================

function LiveTab({ channel }: { channel: any }) {
  const wsProto = location.protocol === "https:" ? "wss:" : "ws:";
  const wsUrl = `${wsProto}//${location.host}/api/v1/channels/connect?key=${channel.api_key}`;
  const [status, setStatus] = useState<"connecting" | "connected" | "disconnected">("connecting");
  const [logs, setLogs] = useState<{ time: string; type: string; data: string }[]>([]);

  useEffect(() => {
    const ws = new WebSocket(wsUrl);
    ws.onopen = () => setStatus("connected");
    ws.onclose = () => setStatus("disconnected");
    ws.onmessage = (e) => {
      const now = new Date().toLocaleTimeString();
      try {
        const msg = JSON.parse(e.data);
        setLogs((prev) => [...prev.slice(-99), { time: now, type: msg.type, data: JSON.stringify(msg.data, null, 2) }]);
      } catch {
        setLogs((prev) => [...prev.slice(-99), { time: now, type: "raw", data: e.data }]);
      }
    };
    return () => ws.close();
  }, [wsUrl]);

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <Radio className={`w-3 h-3 ${status === "connected" ? "text-primary" : "text-muted-foreground"}`} />
        <span className="text-xs">{status === "connected" ? "已连接" : status === "connecting" ? "连接中..." : "已断开"}</span>
        {logs.length > 0 && (
          <button onClick={() => setLogs([])} className="text-[10px] text-muted-foreground hover:text-primary cursor-pointer ml-auto">清空</button>
        )}
      </div>
      <div className="border rounded-lg bg-card p-2 max-h-96 overflow-y-auto space-y-1">
        {logs.length === 0 && <p className="text-[10px] text-muted-foreground text-center py-4">等待消息...</p>}
        {logs.map((log, i) => (
          <div key={i} className="text-[10px] font-mono">
            <span className="text-muted-foreground">{log.time}</span>{" "}
            <span className="text-primary">{log.type}</span>
            <pre className="text-muted-foreground whitespace-pre-wrap ml-4">{log.data}</pre>
          </div>
        ))}
      </div>
    </div>
  );
}
