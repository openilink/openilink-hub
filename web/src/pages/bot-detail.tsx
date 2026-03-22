import { useEffect, useRef, useState, useCallback } from "react";
import { useParams, Link } from "react-router-dom";
import { ArrowLeft, Send, Cable, Copy, Check, Plus, Trash2, RotateCw, Radio, X, Bot, Settings2 } from "lucide-react";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Badge } from "../components/ui/badge";
import { api } from "../lib/api";

type Message = { id: number; direction: string; sender: string; recipient: string; msg_type: string; payload: any; created_at: number };

function getContent(m: Message): string {
  if (m.payload?.content) return m.payload.content;
  return `[${m.msg_type}]`;
}

export function BotDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [bot, setBot] = useState<any>(null);
  const [channels, setChannels] = useState<any[]>([]);
  const [tab, setTab] = useState<"chat" | "channels">("chat");
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const [sendError, setSendError] = useState("");
  const bottomRef = useRef<HTMLDivElement>(null);

  async function loadBot() {
    const bots = await api.listBots();
    setBot((bots || []).find((b: any) => b.id === id) || null);
  }

  async function loadChannels() {
    const all = await api.listChannels();
    setChannels((all || []).filter((c: any) => c.bot_id === id));
  }

  async function loadMessages() {
    if (!id) return;
    const data = await api.messages(id, 200);
    setMessages((data || []).reverse());
  }

  useEffect(() => { loadBot(); loadChannels(); loadMessages(); }, [id]);
  useEffect(() => { bottomRef.current?.scrollIntoView({ behavior: "smooth" }); }, [messages]);
  useEffect(() => {
    const t = setInterval(loadMessages, 5000);
    return () => clearInterval(t);
  }, [id]);

  async function handleSend(e: React.FormEvent) {
    e.preventDefault();
    if (!input.trim() || !id) return;
    setSending(true);
    setSendError("");
    try {
      const res = await fetch("/api/bots/" + id + "/send", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ text: input }),
      });
      const data = await res.json();
      if (!res.ok) {
        const msg = data.error || "";
        if (msg.includes("context token")) {
          setSendError("请先从微信给 Bot 发一条消息，建立会话后才能回复");
        } else if (msg.includes("not connected")) {
          setSendError("Bot 未连接，请返回重新连接");
        } else {
          setSendError(msg || "发送失败");
        }
      } else {
        setInput("");
        setTimeout(loadMessages, 500);
      }
    } catch {
      setSendError("网络错误");
    }
    setSending(false);
  }

  if (!bot) return <p className="text-sm text-muted-foreground p-8">加载中...</p>;

  return (
    <div className="flex flex-col h-[calc(100vh-64px)]">
      {/* Header */}
      <div className="flex items-center gap-3 pb-4 border-b shrink-0">
        <Link to="/" className="text-muted-foreground hover:text-foreground">
          <ArrowLeft className="w-4 h-4" />
        </Link>
        <div className="flex-1 min-w-0">
          <h2 className="font-semibold text-sm">{bot.name}</h2>
          <p className="text-xs text-muted-foreground font-mono truncate">{bot.extra?.bot_id}</p>
        </div>
        <Badge variant={bot.status === "connected" ? "default" : "outline"}>{bot.status}</Badge>
        <div className="flex border rounded-lg overflow-hidden">
          <button className={`px-3 py-1 text-xs cursor-pointer ${tab === "chat" ? "bg-secondary" : "text-muted-foreground"}`} onClick={() => setTab("chat")}>消息</button>
          <button className={`px-3 py-1 text-xs cursor-pointer ${tab === "channels" ? "bg-secondary" : "text-muted-foreground"}`} onClick={() => setTab("channels")}>通道</button>
        </div>
      </div>

      {tab === "chat" ? (
        <div className="flex-1 flex flex-col overflow-hidden mt-3 rounded-xl border">
          <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
            {messages.map((m) => {
              const isIn = m.direction === "inbound";
              return (
                <div key={m.id} className={`flex ${isIn ? "justify-start" : "justify-end"}`}>
                  <div className={`max-w-[75%] px-3 py-2 rounded-xl text-sm ${
                    isIn ? "bg-secondary rounded-bl-sm" : "bg-primary text-primary-foreground rounded-br-sm"
                  }`}>
                    {getContent(m)}
                    <div className={`text-[10px] mt-1 ${isIn ? "text-muted-foreground" : "opacity-50"}`}>
                      {new Date(m.created_at * 1000).toLocaleTimeString()}
                    </div>
                  </div>
                </div>
              );
            })}
            {messages.length === 0 && (
              <p className="text-center text-xs text-muted-foreground py-12">暂无消息</p>
            )}
            <div ref={bottomRef} />
          </div>

          {sendError && (
            <div className="px-4 py-2 text-xs text-destructive bg-secondary border-t">
              {sendError}
            </div>
          )}
          <form onSubmit={handleSend} className="flex gap-2 p-3 border-t shrink-0">
            <Input
              value={input}
              onChange={(e) => { setInput(e.target.value); setSendError(""); }}
              placeholder="输入消息..."
              className="h-9 text-sm flex-1"
            />
            <Button type="submit" size="sm" disabled={sending || !input.trim()}>
              <Send className="w-4 h-4" />
            </Button>
          </form>
        </div>
      ) : (
        <ChannelsTab botId={id!} channels={channels} onRefresh={loadChannels} />
      )}
    </div>
  );
}

function ChannelsTab({ botId, channels, onRefresh }: { botId: string; channels: any[]; onRefresh: () => void }) {
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState("");
  const [handle, setHandle] = useState("");
  const [showDocs, setShowDocs] = useState(false);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!name) return;
    await api.createChannel(botId, name, handle);
    setName("");
    setHandle("");
    setCreating(false);
    onRefresh();
  }

  return (
    <div className="space-y-3 mt-4">
      {channels.map((ch) => <ChannelRow key={ch.id} channel={ch} onRefresh={onRefresh} />)}
      {creating ? (
        <form onSubmit={handleCreate} className="space-y-2">
          <div className="flex gap-2">
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="通道名称" className="h-8 text-sm" autoFocus />
            <Input value={handle} onChange={(e) => setHandle(e.target.value)} placeholder="@提及标识（可选）" className="h-8 text-sm w-40" />
          </div>
          <div className="flex gap-2">
            <Button type="submit" size="sm">创建</Button>
            <Button type="button" variant="ghost" size="sm" onClick={() => setCreating(false)}>取消</Button>
          </div>
          <p className="text-[10px] text-muted-foreground">设置提及标识后，用户发送 @标识 的消息将定向路由到此通道</p>
        </form>
      ) : (
        <Button variant="outline" size="sm" onClick={() => setCreating(true)} className="w-full">
          <Plus className="w-4 h-4 mr-1" /> 添加通道
        </Button>
      )}

      <button onClick={() => setShowDocs(!showDocs)} className="text-xs text-muted-foreground hover:text-primary cursor-pointer">
        {showDocs ? "收起" : "查看"} WebSocket 协议说明
      </button>
      {showDocs && <WsProtocolDocs />}
    </div>
  );
}

function WsProtocolDocs() {
  return (
    <div className="text-xs text-muted-foreground space-y-3 p-4 rounded-lg border bg-background">
      <p className="font-medium text-foreground">WebSocket 协议说明</p>
      <p>所有消息均为 JSON 格式，包含 <code className="text-primary">type</code> 字段标识消息类型。</p>

      <div>
        <p className="font-medium text-foreground mt-3 mb-1">连接后自动收到：init</p>
        <pre className="bg-card p-2 rounded overflow-x-auto">{`{
  "type": "init",
  "data": {
    "channel_id": "uuid",
    "channel_name": "通道名",
    "bot_id": "uuid",
    "bot_status": "connected"
  }
}`}</pre>
      </div>

      <div>
        <p className="font-medium text-foreground mt-3 mb-1">收到消息：message</p>
        <pre className="bg-card p-2 rounded overflow-x-auto">{`{
  "type": "message",
  "data": {
    "seq_id": 123,
    "sender": "xxx@im.wechat",
    "timestamp": 1711100000000,
    "items": [
      { "type": "text", "text": "你好" },
      { "type": "image" },
      { "type": "voice", "text": "语音转文字" },
      { "type": "file", "file_name": "doc.pdf" },
      { "type": "video" }
    ]
  }
}`}</pre>
      </div>

      <div>
        <p className="font-medium text-foreground mt-3 mb-1">Bot 状态变化：bot_status</p>
        <pre className="bg-card p-2 rounded overflow-x-auto">{`{
  "type": "bot_status",
  "data": { "bot_id": "uuid", "status": "disconnected" }
}`}</pre>
        <p className="mt-1">status: connected / disconnected / error / session_expired</p>
      </div>

      <div>
        <p className="font-medium text-foreground mt-3 mb-1">发送消息（客户端 → 服务端）</p>
        <pre className="bg-card p-2 rounded overflow-x-auto">{`{
  "type": "send_text",
  "req_id": "自定义请求ID",
  "data": {
    "recipient": "xxx@im.wechat",
    "text": "回复内容"
  }
}`}</pre>
      </div>

      <div>
        <p className="font-medium text-foreground mt-3 mb-1">发送确认：send_ack</p>
        <pre className="bg-card p-2 rounded overflow-x-auto">{`{
  "type": "send_ack",
  "data": {
    "req_id": "自定义请求ID",
    "success": true,
    "client_id": "sdk-xxx",
    "error": ""
  }
}`}</pre>
      </div>

      <div>
        <p className="font-medium text-foreground mt-3 mb-1">心跳</p>
        <p>发送 <code className="text-primary">{`{"type":"ping"}`}</code>，收到 <code className="text-primary">{`{"type":"pong"}`}</code></p>
      </div>

      <div>
        <p className="font-medium text-foreground mt-3 mb-1">测试命令</p>
        <pre className="bg-card p-2 rounded overflow-x-auto">node example/ws-test.mjs "ws://host:port/api/ws?key=API_KEY"</pre>
      </div>
    </div>
  );
}

function ChannelRow({ channel, onRefresh }: { channel: any; onRefresh: () => void }) {
  const [copiedKey, setCopiedKey] = useState(false);
  const [copiedWs, setCopiedWs] = useState(false);
  const [showLive, setShowLive] = useState(false);
  const [showAI, setShowAI] = useState(false);
  const [editingHandle, setEditingHandle] = useState(false);
  const [handleVal, setHandleVal] = useState(channel.handle || "");

  const wsProto = location.protocol === "https:" ? "wss:" : "ws:";
  const wsUrl = `${wsProto}//${location.host}/api/ws?key=${channel.api_key}`;
  const aiEnabled = channel.ai_config?.enabled;

  async function saveHandle() {
    await api.updateChannel(channel.id, { handle: handleVal });
    setEditingHandle(false);
    onRefresh();
  }

  function copyKey() {
    navigator.clipboard.writeText(channel.api_key);
    setCopiedKey(true);
    setTimeout(() => setCopiedKey(false), 2000);
  }
  function copyWs() {
    navigator.clipboard.writeText(wsUrl);
    setCopiedWs(true);
    setTimeout(() => setCopiedWs(false), 2000);
  }

  return (
    <div className="p-3 rounded-lg border bg-card space-y-2">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Cable className="w-3.5 h-3.5 text-muted-foreground" />
          <span className="text-sm font-medium">{channel.name}</span>
          {editingHandle ? (
            <form onSubmit={(e) => { e.preventDefault(); saveHandle(); }} className="flex items-center gap-1">
              <Input
                value={handleVal}
                onChange={(e) => setHandleVal(e.target.value)}
                placeholder="handle"
                className="h-5 text-[10px] w-24 px-1.5 font-mono"
                autoFocus
                onBlur={saveHandle}
              />
            </form>
          ) : (
            <button
              onClick={() => setEditingHandle(true)}
              className="text-[10px] text-muted-foreground bg-secondary px-1.5 py-0.5 rounded cursor-pointer hover:bg-secondary/80"
            >
              {channel.handle ? `@${channel.handle}` : "+ handle"}
            </button>
          )}
          {aiEnabled && (
            <span className="text-[10px] text-primary bg-primary/10 px-1.5 py-0.5 rounded flex items-center gap-0.5">
              <Bot className="w-2.5 h-2.5" /> AI
            </span>
          )}
        </div>
        <div className="flex gap-1 shrink-0">
          <Button variant={showAI ? "default" : "ghost"} size="sm" onClick={() => setShowAI(!showAI)} title="AI 配置">
            <Bot className="w-3.5 h-3.5" />
          </Button>
          <Button variant={showLive ? "default" : "ghost"} size="sm" onClick={() => setShowLive(!showLive)} title="实时监听">
            <Radio className="w-3.5 h-3.5" />
          </Button>
          <Button variant="ghost" size="sm" onClick={async () => { if (confirm("重新生成 Key？")) { await api.rotateKey(channel.id); onRefresh(); } }}>
            <RotateCw className="w-3.5 h-3.5" />
          </Button>
          <Button variant="ghost" size="sm" onClick={async () => { if (confirm("删除？")) { await api.deleteChannel(channel.id); onRefresh(); } }}>
            <Trash2 className="w-3.5 h-3.5 text-destructive" />
          </Button>
        </div>
      </div>

      <CopyRow label="API Key" value={channel.api_key} copied={copiedKey} onCopy={copyKey} />
      <CopyRow label="WebSocket" value={wsUrl} copied={copiedWs} onCopy={copyWs} />

      {showAI && <AIConfigPanel channelId={channel.id} config={channel.ai_config} onSaved={onRefresh} />}
      {showLive && <LivePanel wsUrl={wsUrl} onClose={() => setShowLive(false)} />}
    </div>
  );
}

function AIConfigPanel({ channelId, config, onSaved }: { channelId: string; config: any; onSaved: () => void }) {
  const [enabled, setEnabled] = useState(config?.enabled || false);
  const [source, setSource] = useState(config?.source || "builtin");
  const [baseUrl, setBaseUrl] = useState(config?.base_url || "");
  const [apiKey, setApiKey] = useState(config?.api_key || "");
  const [model, setModel] = useState(config?.model || "");
  const [systemPrompt, setSystemPrompt] = useState(config?.system_prompt || "");
  const [maxHistory, setMaxHistory] = useState(config?.max_history || 20);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  function normalizeBaseUrl(url: string): string {
    if (!url) return "";
    let u = url.replace(/\/+$/, "");
    if (u && !u.endsWith("/v1")) u += "/v1";
    return u;
  }

  async function handleSave() {
    setSaving(true);
    setError("");
    try {
      const cfg: any = {
        enabled,
        source,
        system_prompt: systemPrompt,
        max_history: maxHistory || 20,
      };
      if (source === "custom") {
        cfg.base_url = normalizeBaseUrl(baseUrl);
        cfg.api_key = apiKey;
        cfg.model = model || "gpt-4o-mini";
        setBaseUrl(cfg.base_url);
      }
      await api.updateChannel(channelId, { ai_config: cfg });
      onSaved();
    } catch (err: any) {
      setError(err.message);
    }
    setSaving(false);
  }

  const canSave = source === "builtin" || apiKey;

  return (
    <div className="border rounded-lg bg-background p-3 space-y-3">
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium flex items-center gap-1.5">
          <Bot className="w-3.5 h-3.5" /> AI 自动回复
        </span>
        <label className="flex items-center gap-1.5 cursor-pointer">
          <span className="text-[10px] text-muted-foreground">{enabled ? "已开启" : "已关闭"}</span>
          <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} className="w-3.5 h-3.5 accent-primary" />
        </label>
      </div>

      {enabled && (
        <div className="space-y-2">
          {/* Source selector */}
          <div className="flex gap-1">
            <button
              onClick={() => setSource("builtin")}
              className={`px-2.5 py-1 text-[11px] rounded cursor-pointer transition-colors ${
                source === "builtin" ? "bg-primary text-primary-foreground" : "bg-secondary text-secondary-foreground hover:bg-secondary/80"
              }`}
            >内置</button>
            <button
              onClick={() => setSource("custom")}
              className={`px-2.5 py-1 text-[11px] rounded cursor-pointer transition-colors ${
                source === "custom" ? "bg-primary text-primary-foreground" : "bg-secondary text-secondary-foreground hover:bg-secondary/80"
              }`}
            >自定义</button>
          </div>

          {source === "builtin" && (
            <p className="text-[10px] text-muted-foreground">使用管理员在设置中配置的全局 AI 服务</p>
          )}

          {source === "custom" && (
            <div className="grid grid-cols-2 gap-2">
              <Input
                placeholder="https://api.openai.com/v1"
                value={baseUrl}
                onChange={(e) => setBaseUrl(e.target.value)}
                onBlur={() => setBaseUrl(normalizeBaseUrl(baseUrl))}
                className="h-7 text-[11px] font-mono col-span-2"
              />
              <Input type="password" placeholder="API Key" value={apiKey} onChange={(e) => setApiKey(e.target.value)} className="h-7 text-[11px] font-mono" />
              <Input placeholder="模型（默认 gpt-4o-mini）" value={model} onChange={(e) => setModel(e.target.value)} className="h-7 text-[11px] font-mono" />
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
            <label className="text-[10px] text-muted-foreground shrink-0">上下文消息数</label>
            <Input type="number" value={maxHistory} onChange={(e) => setMaxHistory(parseInt(e.target.value) || 20)} className="h-7 text-[11px] w-20" min={1} max={100} />
            <div className="flex-1" />
            {error && <span className="text-[10px] text-destructive">{error}</span>}
            <Button size="sm" className="h-7" onClick={handleSave} disabled={saving || !canSave}>{saving ? "..." : "保存"}</Button>
          </div>
        </div>
      )}
    </div>
  );
}

type WsLogEntry = {
  id: number;
  dir: "in" | "out" | "sys";
  type: string;
  data: any;
  time: string;
};

function LivePanel({ wsUrl, onClose }: { wsUrl: string; onClose: () => void }) {
  const [status, setStatus] = useState<"connecting" | "connected" | "disconnected">("connecting");
  const [logs, setLogs] = useState<WsLogEntry[]>([]);
  const [input, setInput] = useState("");
  const wsRef = useRef<WebSocket | null>(null);
  const logEndRef = useRef<HTMLDivElement>(null);
  const seqRef = useRef(0);

  const addLog = useCallback((dir: WsLogEntry["dir"], type: string, data: any) => {
    setLogs((prev) => {
      const entry: WsLogEntry = {
        id: ++seqRef.current,
        dir,
        type,
        data,
        time: new Date().toLocaleTimeString(),
      };
      const next = [...prev, entry];
      return next.length > 200 ? next.slice(-200) : next;
    });
  }, []);

  useEffect(() => {
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      setStatus("connected");
      addLog("sys", "connected", null);
    };

    ws.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data);
        addLog("in", msg.type, msg.data);
      } catch {
        addLog("in", "raw", e.data);
      }
    };

    ws.onclose = () => {
      setStatus("disconnected");
      addLog("sys", "disconnected", null);
    };

    ws.onerror = () => {
      addLog("sys", "error", null);
    };

    // Ping keepalive
    const pingInterval = setInterval(() => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "ping" }));
      }
    }, 30000);

    return () => {
      clearInterval(pingInterval);
      ws.close();
    };
  }, [wsUrl, addLog]);

  useEffect(() => {
    logEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [logs]);

  function handleSend(e: React.FormEvent) {
    e.preventDefault();
    if (!input.trim() || !wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;

    const reqId = `req-${Date.now()}`;
    const msg = {
      type: "send_text",
      req_id: reqId,
      data: { text: input },
    };
    wsRef.current.send(JSON.stringify(msg));
    addLog("out", "send_text", msg.data);
    setInput("");
  }

  const statusColor = {
    connecting: "text-yellow-500",
    connected: "text-green-500",
    disconnected: "text-destructive",
  }[status];

  return (
    <div className="border rounded-lg bg-background overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-1.5 border-b bg-secondary/30">
        <div className="flex items-center gap-2">
          <span className={`text-[10px] font-medium ${statusColor}`}>
            {status === "connected" ? "LIVE" : status === "connecting" ? "CONNECTING" : "DISCONNECTED"}
          </span>
        </div>
        <button onClick={onClose} className="text-muted-foreground hover:text-foreground cursor-pointer">
          <X className="w-3.5 h-3.5" />
        </button>
      </div>

      {/* Log */}
      <div className="h-64 overflow-y-auto text-[11px] p-2 space-y-1">
        {logs.map((log) => <LogEntry key={log.id} log={log} />)}
        {logs.length === 0 && (
          <p className="text-muted-foreground text-center py-8">等待消息...</p>
        )}
        <div ref={logEndRef} />
      </div>

      {/* Send */}
      <form onSubmit={handleSend} className="flex gap-1.5 p-2 border-t">
        <Input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="发送内容..."
          className="h-7 text-[11px] flex-1"
        />
        <Button type="submit" size="sm" className="h-7 px-2" disabled={status !== "connected" || !input.trim()}>
          <Send className="w-3 h-3" />
        </Button>
      </form>
    </div>
  );
}

function LogEntry({ log }: { log: WsLogEntry }) {
  const [expanded, setExpanded] = useState(false);

  if (log.dir === "sys") {
    return (
      <div className="text-muted-foreground text-center text-[10px] py-0.5">
        — {log.type} {log.time} —
      </div>
    );
  }

  const isIn = log.dir === "in";
  const dirColor = isIn ? "text-green-500" : "text-blue-500";
  const dirIcon = isIn ? "◀" : "▶";

  // Format summary based on message type
  let summary = "";
  const d = log.data;
  if (d) {
    switch (log.type) {
      case "message": {
        const sender = d.sender || "";
        const items = d.items || [];
        const texts = items.map((it: any) => {
          if (it.type === "text") return it.text;
          if (it.type === "voice" && it.text) return `[语音] ${it.text}`;
          if (it.type === "file") return `[文件] ${it.file_name || ""}`;
          return `[${it.type}]`;
        }).join(" ");
        summary = sender ? `${sender}: ${texts}` : texts;
        break;
      }
      case "init":
        summary = `channel=${d.channel_name || d.channel_id} bot_status=${d.bot_status}`;
        break;
      case "bot_status":
        summary = d.status || "";
        break;
      case "send_ack":
        summary = d.success ? "ok" : `err: ${d.error}`;
        break;
      case "send_text":
        summary = d.text || "";
        break;
      case "pong":
        summary = "";
        break;
      default:
        summary = typeof d === "string" ? d : JSON.stringify(d);
    }
  }

  const hasDetail = d != null && log.type !== "pong";

  return (
    <div className="group">
      <div
        className={`flex items-start gap-1.5 ${hasDetail ? "cursor-pointer" : ""} hover:bg-secondary/30 rounded px-1 -mx-1`}
        onClick={() => hasDetail && setExpanded(!expanded)}
      >
        <span className="text-muted-foreground shrink-0 w-14 text-[10px] pt-px">{log.time}</span>
        <span className={`shrink-0 ${dirColor}`}>{dirIcon}</span>
        <span className="text-primary shrink-0 font-medium">{log.type}</span>
        <span className="text-foreground truncate">{summary}</span>
      </div>
      {expanded && d != null && (
        <pre className="ml-[70px] text-[10px] text-muted-foreground bg-secondary/30 rounded p-2 mt-0.5 mb-1 overflow-x-auto whitespace-pre-wrap break-all">
          {JSON.stringify(d, null, 2)}
        </pre>
      )}
    </div>
  );
}

function CopyRow({ label, value, copied, onCopy }: { label: string; value: string; copied: boolean; onCopy: () => void }) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-[10px] text-muted-foreground w-16 shrink-0">{label}</span>
      <code className="flex-1 text-[10px] text-muted-foreground font-mono bg-background border rounded px-2 py-1 truncate select-all">
        {value}
      </code>
      <button onClick={onCopy} className="cursor-pointer text-muted-foreground hover:text-foreground shrink-0">
        {copied ? <Check className="w-3 h-3" /> : <Copy className="w-3 h-3" />}
      </button>
    </div>
  );
}
