import { useEffect, useRef, useState, useCallback } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import { ArrowLeft, Send, Cable, Copy, Check, Plus, Trash2, RotateCw, Radio, X, Bot, Webhook, Paperclip, QrCode, Puzzle } from "lucide-react";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Badge } from "../components/ui/badge";
import { api } from "../lib/api";

type MessageItem = { type: string; text?: string; file_name?: string };
type Message = {
  id: number; bot_id?: string; direction: string;
  from_user_id: string; to_user_id: string;
  message_type: number; item_list: MessageItem[];
  media_status?: string; media_keys?: Record<string, string>;
  created_at: number;
  // Optimistic UI fields (client-only)
  _sending?: boolean; _error?: string; _preview_url?: string;
};

function getMediaUrl(m: Message, index: number): string | null {
  const key = m.media_keys?.[String(index)];
  if (key) return `/api/v1/media/${key}`;
  return null;
}

function getSilkUrl(m: Message, index: number): string | null {
  const key = m.media_keys?.[`${index}_silk`];
  if (key) return `/api/v1/media/${key}`;
  return null;
}

function getItemMediaType(item: MessageItem): string {
  return item.type || "text";
}

function ItemContent({ item, m, index }: { item: MessageItem; m: Message; index: number }) {
  const mediaType = getItemMediaType(item);
  const url = getMediaUrl(m, index);

  if (mediaType === "image" && url) {
    return <img src={url} alt="image" className="max-w-full rounded-lg max-h-48 cursor-pointer" onClick={() => window.open(url)} />;
  }
  if (mediaType === "video" && url) {
    return <video src={url} controls className="max-w-full rounded-lg max-h-48" />;
  }
  if (mediaType === "voice" && url) {
    const silkUrl = getSilkUrl(m, index);
    return (
      <div className="space-y-1">
        <audio src={url} controls className="h-8" />
        {item.text && item.text !== "[voice]" && <p className="text-xs opacity-70">{item.text}</p>}
        <div className="flex gap-2 text-[10px]">
          <a href={url} download className="text-muted-foreground hover:text-primary">WAV</a>
          {silkUrl && <a href={silkUrl} download className="text-muted-foreground hover:text-primary">SILK</a>}
        </div>
      </div>
    );
  }
  if (mediaType === "file" && url) {
    return (
      <a href={url} target="_blank" rel="noopener" className="flex items-center gap-2 underline text-xs">
        📎 {item.file_name || item.text || "下载文件"}
      </a>
    );
  }
  if (item.text) return <>{item.text}</>;
  if (mediaType !== "text") return <span className="text-muted-foreground">[{mediaType}]</span>;
  return null;
}

function MessageContent({ m }: { m: Message }) {
  // Media downloading/failed status (applies to entire message)
  if (m.media_status === "downloading") {
    const firstMedia = (m.item_list || []).find((i) => i.type !== "text");
    const t = firstMedia?.type || "file";
    return (
      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        <span className="animate-pulse">⏳</span>
        {t === "image" ? "图片下载中..." : t === "video" ? "视频下载中..." : t === "voice" ? "语音下载中..." : "文件下载中..."}
      </div>
    );
  }
  if (m.media_status === "failed") {
    return (
      <div className="space-y-1">
        <div className="flex items-center gap-2 text-xs">
          <span className="text-destructive">下载失败</span>
          <button
            className="text-primary hover:underline cursor-pointer"
            onClick={async (e) => {
              e.stopPropagation();
              try {
                await fetch(`/api/bots/${m.bot_id}/messages/${m.id}/retry_media`, {
                  method: "POST", credentials: "same-origin",
                });
              } catch {}
            }}
          >重试</button>
        </div>
        <p className="text-[10px] text-muted-foreground">CDN 链接可能已过期，需要对方重新发送</p>
      </div>
    );
  }

  // Optimistic preview for sending media
  if (m._preview_url && m._sending) {
    const t = (m.item_list || [])[0]?.type || "file";
    if (t === "image") return <img src={m._preview_url} alt="preview" className="max-w-full rounded-lg max-h-48 opacity-60" />;
    if (t === "video") return <video src={m._preview_url} className="max-w-full rounded-lg max-h-48 opacity-60" />;
  }

  const items = m.item_list || [];
  if (items.length === 0) return <span className="text-muted-foreground">[空消息]</span>;

  return (
    <div className="space-y-1">
      {items.map((item, i) => (
        <ItemContent key={i} item={item} m={m} index={i} />
      ))}
    </div>
  );
}

export function BotDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [bot, setBot] = useState<any>(null);
  const [channels, setChannels] = useState<any[]>([]);
  const [tab, setTab] = useState<"chat" | "channels" | "settings">("chat");
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const [sendError, setSendError] = useState("");
  const [pendingFile, setPendingFile] = useState<File | null>(null);
  const [pendingPreview, setPendingPreview] = useState<string | null>(null);
  const [showRebind, setShowRebind] = useState(false);
  const [nextCursor, setNextCursor] = useState<string>("");
  const [hasMore, setHasMore] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);
  const scrollRef = useRef<HTMLDivElement>(null);

  async function loadBot() {
    const bots = await api.listBots();
    setBot((bots || []).find((b: any) => b.id === id) || null);
  }

  async function loadChannels() {
    if (!id) return;
    const all = await api.listChannels(id);
    setChannels(all || []);
  }

  async function loadMessages() {
    if (!id) return;
    const res = await api.messages(id, 30);
    setMessages((res.messages || []).reverse());
    setNextCursor(res.next_cursor || "");
    setHasMore(res.has_more);
  }

  async function loadOlder() {
    if (!id || !nextCursor || loadingMore) return;
    setLoadingMore(true);
    const scrollEl = scrollRef.current;
    const prevHeight = scrollEl?.scrollHeight || 0;
    try {
      const res = await api.messages(id, 30, nextCursor);
      const older = (res.messages || []).reverse();
      setMessages((prev) => [...older, ...prev]);
      setNextCursor(res.next_cursor || "");
      setHasMore(res.has_more);
      // Restore scroll position after prepending
      requestAnimationFrame(() => {
        if (scrollEl) scrollEl.scrollTop = scrollEl.scrollHeight - prevHeight;
      });
    } finally {
      setLoadingMore(false);
    }
  }

  useEffect(() => { loadBot(); loadChannels(); loadMessages(); }, [id]);
  useEffect(() => {
    requestAnimationFrame(() => bottomRef.current?.scrollIntoView({ behavior: "smooth" }));
  }, [messages.length]);
  useEffect(() => {
    // Poll for new messages only (don't reload all)
    const t = setInterval(async () => {
      if (!id) return;
      const res = await api.messages(id, 30);
      const fresh = (res.messages || []).reverse();
      setMessages((prev) => {
        // Merge: keep optimistic (negative ids, still sending), replace rest with fresh
        const optimistic = prev.filter((m) => m.id < 0);
        return [...fresh, ...optimistic];
      });
    }, 5000);
    return () => clearInterval(t);
  }, [id]);

  async function handleSend(e: React.FormEvent) {
    e.preventDefault();
    if (pendingFile) {
      await confirmFileSend();
      return;
    }
    if (!input.trim() || !id) return;
    const optId = -Date.now();
    setMessages((prev) => [...prev, {
      id: optId, direction: "outbound", from_user_id: "", to_user_id: "",
      message_type: 2, item_list: [{ type: "text", text: input }],
      created_at: Date.now() / 1000, _sending: true,
    }]);
    const text = input;
    setInput("");
    const err = await doSend({ text });
    setMessages((prev) => prev.map((m) => m.id === optId
      ? { ...m, _sending: false, _error: err || undefined }
      : m));
  }

  function handleFileSelect(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    e.target.value = "";
    setPendingFile(file);
    if (file.type.startsWith("image/") || file.type.startsWith("video/")) {
      setPendingPreview(URL.createObjectURL(file));
    } else {
      setPendingPreview(null);
    }
  }

  function cancelFile() {
    if (pendingPreview) URL.revokeObjectURL(pendingPreview);
    setPendingFile(null);
    setPendingPreview(null);
  }

  async function confirmFileSend() {
    if (!pendingFile || !id) return;
    const file = pendingFile;
    const preview = pendingPreview;
    const caption = input.trim();

    const optId = -Date.now();
    const mediaType = file.type.startsWith("image/") ? "image" : file.type.startsWith("video/") ? "video" : "file";
    setMessages((prev) => [...prev, {
      id: optId, direction: "outbound", from_user_id: "", to_user_id: "",
      message_type: 2,
      item_list: [{ type: mediaType, text: caption || file.name, file_name: file.name }],
      created_at: Date.now() / 1000,
      _sending: true, _preview_url: preview || undefined,
    }]);

    const form = new FormData();
    form.append("file", file);
    if (caption) form.append("text", caption);
    setInput("");
    setPendingFile(null);
    setPendingPreview(null);
    const err = await doSend(form);
    setMessages((prev) => prev.map((m) => m.id === optId
      ? { ...m, _sending: false, _error: err || undefined }
      : m));
  }

  async function retrySend(m: Message) {
    const optId = m.id;
    setMessages((prev) => prev.map((msg) => msg.id === optId
      ? { ...msg, _sending: true, _error: undefined }
      : msg));
    const textItem = (m.item_list || []).find((i) => i.text);
    const err = await doSend({ text: textItem?.text || "" });
    setMessages((prev) => prev.map((msg) => msg.id === optId
      ? { ...msg, _sending: false, _error: err || undefined }
      : msg));
  }

  async function doSend(body: any): Promise<string | null> {
    setSending(true);
    setSendError("");
    try {
      const isForm = body instanceof FormData;
      const res = await fetch("/api/bots/" + id + "/send", {
        method: "POST",
        credentials: "same-origin",
        ...(isForm ? { body } : { headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }),
      });
      const data = await res.json();
      if (!res.ok) {
        const msg = data.error || "";
        if (msg.includes("context token")) return "请先从微信给 Bot 发一条消息";
        if (msg.includes("session expired")) {
          setShowRebind(true);
          return "会话已过期";
        }
        if (msg.includes("not connected")) return "Bot 未连接，请尝试重连";
        return msg || "发送失败";
      }
      setTimeout(loadMessages, 500);
      return null;
    } catch (e: any) {
      return "网络错误: " + (e?.message || "请求失败");
    } finally {
      setSending(false);
    }
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
          <h1 className="font-semibold text-base">{bot.name}</h1>
          <p className="text-xs text-muted-foreground font-mono truncate">{bot.extra?.bot_id}</p>
        </div>
        <Badge variant={bot.status === "connected" ? "default" : bot.status === "session_expired" ? "destructive" : "outline"}>
          {bot.status === "session_expired" ? "已过期" : bot.status}
        </Badge>
        {bot.status === "session_expired" && (
          <Button variant="outline" size="sm" className="text-xs h-7" onClick={() => navigate("/")}>
            <QrCode className="w-3.5 h-3.5 mr-1" /> 重新绑定
          </Button>
        )}
      </div>
      <div className="flex border rounded-lg overflow-hidden w-fit mt-3">
        <button className={`px-3 py-1.5 text-xs cursor-pointer ${tab === "chat" ? "bg-secondary font-medium" : "text-muted-foreground"}`} onClick={() => setTab("chat")}>消息</button>
        <button className={`px-3 py-1.5 text-xs cursor-pointer ${tab === "channels" ? "bg-secondary font-medium" : "text-muted-foreground"}`} onClick={() => setTab("channels")}>通道</button>
        <button className={`px-3 py-1.5 text-xs cursor-pointer ${tab === "settings" ? "bg-secondary font-medium" : "text-muted-foreground"}`} onClick={() => setTab("settings")}>设置</button>
      </div>

      {/* Session expired rebind dialog */}
      {showRebind && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowRebind(false)}>
          <div className="bg-background border rounded-xl p-6 max-w-sm mx-4 space-y-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-2 text-destructive">
              <QrCode className="w-5 h-5" />
              <h3 className="font-semibold text-sm">会话已过期</h3>
            </div>
            <p className="text-sm text-muted-foreground">Bot 的微信登录会话已过期，需要重新扫码绑定。重新绑定后，现有通道和配置将自动保留。</p>
            <div className="flex gap-2 justify-end">
              <Button variant="ghost" size="sm" onClick={() => setShowRebind(false)}>稍后</Button>
              <Button size="sm" onClick={() => navigate("/")}>去重新绑定</Button>
            </div>
          </div>
        </div>
      )}

      {tab === "chat" ? (
        <div className="flex-1 flex flex-col overflow-hidden mt-3 rounded-xl border">
          <div ref={scrollRef} className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
            {hasMore && (
              <div className="text-center py-2">
                <button
                  onClick={loadOlder}
                  disabled={loadingMore}
                  className="text-xs text-muted-foreground hover:text-primary cursor-pointer"
                >{loadingMore ? "加载中..." : "加载更早消息"}</button>
              </div>
            )}
            {messages.map((m) => {
              const isIn = m.direction === "inbound";
              return (
                <div key={m.id} className={`flex ${isIn ? "justify-start" : "justify-end"}`}>
                  <div className={`max-w-[75%] px-3 py-2 rounded-xl text-sm ${
                    isIn ? "bg-secondary rounded-bl-sm" : "bg-primary text-primary-foreground rounded-br-sm"
                  } ${m._sending ? "opacity-60" : ""}`}>
                    <MessageContent m={m} />
                    <div className={`text-[10px] mt-1 ${isIn ? "text-muted-foreground" : "opacity-50"}`}>
                      {m._sending ? "发送中..." : m._error ? (
                        <span className="text-destructive">
                          {m._error}
                          <button className="ml-2 underline cursor-pointer" onClick={(e) => { e.stopPropagation(); retrySend(m); }}>重试</button>
                        </span>
                      ) : new Date(m.created_at * 1000).toLocaleTimeString()}
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

          {pendingFile && (
            <div className="px-4 py-2 border-t bg-secondary/50 flex items-center gap-3">
              {pendingPreview && pendingFile?.type.startsWith("image/") ? (
                <img src={pendingPreview} alt="preview" className="h-16 rounded" />
              ) : pendingPreview && pendingFile?.type.startsWith("video/") ? (
                <video src={pendingPreview} className="h-16 rounded" />
              ) : (
                <div className="h-16 w-16 rounded bg-muted flex items-center justify-center text-xs text-muted-foreground">
                  {pendingFile.name.split('.').pop()?.toUpperCase()}
                </div>
              )}
              <div className="flex-1 min-w-0">
                <p className="text-xs truncate">{pendingFile.name}</p>
                <p className="text-[10px] text-muted-foreground">{(pendingFile.size / 1024).toFixed(1)} KB</p>
              </div>
              <Button size="sm" className="h-7" onClick={confirmFileSend} disabled={sending}>发送</Button>
              <Button size="sm" variant="ghost" className="h-7" onClick={cancelFile}>取消</Button>
            </div>
          )}
          <form onSubmit={handleSend} className="flex gap-2 p-3 border-t shrink-0">
            <label className="cursor-pointer text-muted-foreground hover:text-foreground flex items-center">
              <Paperclip className="w-4 h-4" />
              <input type="file" className="hidden" onChange={handleFileSelect} disabled={sending} />
            </label>
            <Input
              value={input}
              onChange={(e) => { setInput(e.target.value); setSendError(""); }}
              placeholder={pendingFile ? "添加说明（可选）..." : "输入消息..."}
              className="h-9 text-sm flex-1"
            />
            <Button type="submit" size="sm" disabled={sending || (!input.trim() && !pendingFile)}>
              <Send className="w-4 h-4" />
            </Button>
          </form>
        </div>
      ) : tab === "channels" ? (
        <ChannelsTab botId={id!} channels={channels} onRefresh={loadChannels} />
      ) : (
        <BotSettingsTab bot={bot} onUpdate={loadBot} />
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
      {channels.map((ch) => <ChannelCard key={ch.id} botId={botId} channel={ch} onRefresh={onRefresh} />)}
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
        <p className="font-medium text-foreground mt-3 mb-1">HTTP API</p>
        <p>所有请求通过 <code className="text-primary">?key=API_KEY</code> 或 <code className="text-primary">X-API-Key</code> 头认证。</p>
        <pre className="bg-card p-2 rounded overflow-x-auto mt-1">{`# 拉取消息（cursor 分页）
GET /api/v1/channels/messages?key=KEY&cursor=&limit=50

# 发送消息
POST /api/v1/channels/send?key=KEY
{"text": "内容"}

# 输入状态
POST /api/v1/channels/typing?key=KEY
{"ticket": "xxx", "status": "typing"}

# 获取配置（typing ticket）
POST /api/v1/channels/config?key=KEY
{"context_token": "xxx"}

# 渠道状态
GET /api/v1/channels/status?key=KEY`}</pre>
      </div>

      <div>
        <p className="font-medium text-foreground mt-3 mb-1">测试命令</p>
        <pre className="bg-card p-2 rounded overflow-x-auto">node example/ws-test.mjs "ws://host:port/api/v1/channels/connect?key=API_KEY"</pre>
      </div>
    </div>
  );
}

function ChannelCard({ botId, channel, onRefresh }: { botId: string; channel: any; onRefresh: () => void }) {
  const nav = useNavigate();
  const aiEnabled = channel.ai_config?.enabled;
  const hasWebhook = !!channel.webhook_config?.url;
  const hasPlugin = !!channel.webhook_config?.plugin_id;
  const filterCount = (channel.filter_rule?.user_ids?.length || 0) + (channel.filter_rule?.keywords?.length || 0) + (channel.filter_rule?.message_types?.length || 0);

  return (
    <div
      className="p-3 rounded-lg border bg-card cursor-pointer hover:border-primary/50 transition-colors"
      onClick={() => nav(`/bot/${botId}/channel/${channel.id}`)}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Cable className="w-3.5 h-3.5 text-muted-foreground" />
          <span className="text-sm font-medium">{channel.name}</span>
          {channel.handle ? (
            <span className="text-[10px] font-mono text-muted-foreground">@{channel.handle}</span>
          ) : (
            <span className="text-[10px] text-muted-foreground">全部消息</span>
          )}
          {!channel.enabled && <Badge variant="outline" className="text-[10px]">停用</Badge>}
        </div>
        <Button variant="ghost" size="sm" onClick={(e) => {
          e.stopPropagation();
          if (confirm("删除此渠道？")) { api.deleteChannel(botId, channel.id).then(onRefresh); }
        }}>
          <Trash2 className="w-3.5 h-3.5 text-destructive" />
        </Button>
      </div>
      <div className="flex items-center gap-2 mt-1.5 text-[10px] text-muted-foreground">
        {hasWebhook && <span className="bg-primary/10 text-primary px-1.5 py-0.5 rounded flex items-center gap-0.5"><Webhook className="w-2.5 h-2.5" /> Webhook</span>}
        {hasPlugin && <span className="bg-primary/10 text-primary px-1.5 py-0.5 rounded flex items-center gap-0.5"><Puzzle className="w-2.5 h-2.5" /> 插件</span>}
        {aiEnabled && <span className="bg-primary/10 text-primary px-1.5 py-0.5 rounded flex items-center gap-0.5"><Bot className="w-2.5 h-2.5" /> AI</span>}
        {filterCount > 0 && <span>{filterCount} 条过滤规则</span>}
      </div>
    </div>
  );
}

// Legacy: kept for backward compat but no longer used in channel list
function ChannelRow({ botId, channel, onRefresh }: { botId: string; channel: any; onRefresh: () => void }) {
  const [copiedKey, setCopiedKey] = useState(false);
  const [copiedWs, setCopiedWs] = useState(false);
  const [copiedHttp, setCopiedHttp] = useState(false);
  const [showLive, setShowLive] = useState(false);
  const [showAI, setShowAI] = useState(false);
  const [showWebhook, setShowWebhook] = useState(false);
  const [editingHandle, setEditingHandle] = useState(false);
  const [handleVal, setHandleVal] = useState(channel.handle || "");

  const wsProto = location.protocol === "https:" ? "wss:" : "ws:";
  const wsUrl = `${wsProto}//${location.host}/api/v1/channels/connect?key=${channel.api_key}`;
  const httpBase = `${location.origin}/api/v1/channels`;
  const aiEnabled = channel.ai_config?.enabled;

  async function saveHandle() {
    await api.updateChannel(botId, channel.id, { handle: handleVal });
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
  function copyHttp() {
    navigator.clipboard.writeText(httpBase);
    setCopiedHttp(true);
    setTimeout(() => setCopiedHttp(false), 2000);
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
          {channel.webhook_config?.url && (
            <span className="text-[10px] text-primary bg-primary/10 px-1.5 py-0.5 rounded flex items-center gap-0.5">
              <Webhook className="w-2.5 h-2.5" /> Webhook
            </span>
          )}
          {channel.webhook_config?.plugin_id && (
            <span className="text-[10px] text-primary bg-primary/10 px-1.5 py-0.5 rounded flex items-center gap-0.5">
              <Puzzle className="w-2.5 h-2.5" /> 插件
            </span>
          )}
        </div>
        <div className="flex gap-1 shrink-0">
          <Button variant={showAI ? "default" : "ghost"} size="sm" onClick={() => setShowAI(!showAI)} title="AI 配置">
            <Bot className="w-3.5 h-3.5" />
          </Button>
          <Button variant={showWebhook ? "default" : "ghost"} size="sm" onClick={() => setShowWebhook(!showWebhook)} title="Webhook">
            <Webhook className="w-3.5 h-3.5" />
          </Button>
          <Button variant={showLive ? "default" : "ghost"} size="sm" onClick={() => setShowLive(!showLive)} title="实时监听">
            <Radio className="w-3.5 h-3.5" />
          </Button>
          <Button variant="ghost" size="sm" onClick={async () => { if (confirm("重新生成 Key？")) { await api.rotateKey(botId, channel.id); onRefresh(); } }}>
            <RotateCw className="w-3.5 h-3.5" />
          </Button>
          <Button variant="ghost" size="sm" onClick={async () => { if (confirm("删除？")) { await api.deleteChannel(botId, channel.id); onRefresh(); } }}>
            <Trash2 className="w-3.5 h-3.5 text-destructive" />
          </Button>
        </div>
      </div>

      <CopyRow label="API Key" value={channel.api_key} copied={copiedKey} onCopy={copyKey} />
      <CopyRow label="WebSocket" value={wsUrl} copied={copiedWs} onCopy={copyWs} />
      <CopyRow label="HTTP API" value={httpBase} copied={copiedHttp} onCopy={copyHttp} />

      {showAI && <AIConfigPanel botId={botId} channelId={channel.id} config={channel.ai_config} onSaved={onRefresh} />}
      {showWebhook && <WebhookPanel botId={botId} channelId={channel.id} config={channel.webhook_config} onSaved={onRefresh} />}
      {showLive && <LivePanel wsUrl={wsUrl} onClose={() => setShowLive(false)} />}
    </div>
  );
}

function AIConfigPanel({ botId, channelId, config, onSaved }: { botId: string; channelId: string; config: any; onSaved: () => void }) {
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
      await api.updateChannel(botId, channelId, { ai_config: cfg });
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

function WebhookPanel({ botId, channelId, config, onSaved }: {
  botId: string; channelId: string; config: any; onSaved: () => void;
}) {
  const [url, setUrl] = useState(config?.url || "");
  const [authType, setAuthType] = useState(config?.auth?.type || "");
  const [authToken, setAuthToken] = useState(config?.auth?.token || "");
  const [authName, setAuthName] = useState(config?.auth?.name || "");
  const [authValue, setAuthValue] = useState(config?.auth?.value || config?.auth?.secret || "");
  const [scriptMode, setScriptMode] = useState<"plugin" | "manual">(config?.plugin_id ? "plugin" : "manual");
  const [script, setScript] = useState(config?.script || "");
  const [pluginId, setPluginId] = useState(config?.plugin_id || "");
  const [pluginInfo, setPluginInfo] = useState<any>(null);
  const [plugins, setPlugins] = useState<any[]>([]);
  const [showPluginPicker, setShowPluginPicker] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (pluginId) {
      api.getPlugin(pluginId).then(setPluginInfo).catch(() => setPluginInfo(null));
    }
  }, [pluginId]);

  useEffect(() => {
    if (showPluginPicker) {
      api.listPlugins().then((list) => setPlugins(list || [])).catch(() => {});
    }
  }, [showPluginPicker]);

  async function handleSave() {
    setSaving(true);
    setError("");
    try {
      let auth: any = null;
      if (authType === "bearer" && authToken) auth = { type: "bearer", token: authToken };
      else if (authType === "header" && authName) auth = { type: "header", name: authName, value: authValue };
      else if (authType === "hmac" && authValue) auth = { type: "hmac", secret: authValue };
      await api.updateChannel(botId, channelId, {
        webhook_config: {
          url,
          auth,
          plugin_id: scriptMode === "plugin" ? pluginId : undefined,
          script: scriptMode === "manual" ? (script || undefined) : undefined,
        },
      });
      onSaved();
    } catch (err: any) { setError(err.message); }
    setSaving(false);
  }

  async function handleInstallPlugin(id: string) {
    try {
      const result = await api.installPluginToChannel(id, botId, channelId);
      setPluginId(result.plugin_id);
      setScriptMode("plugin");
      setScript("");
      setShowPluginPicker(false);
      onSaved();
    } catch (err: any) { setError(err.message); }
  }

  function handleUninstallPlugin() {
    setPluginId("");
    setPluginInfo(null);
    setScriptMode("manual");
  }

  return (
    <div className="border rounded-lg bg-background p-3 space-y-3">
      <span className="text-xs font-medium flex items-center gap-1.5">
        <Webhook className="w-3.5 h-3.5" /> Webhook 推送
      </span>
      <div className="space-y-2">
        <Input placeholder="https://your-server.com/webhook" value={url} onChange={(e) => setUrl(e.target.value)} className="h-7 text-[11px] font-mono" />

        {/* Auth */}
        <div className="flex gap-1">
          {["", "bearer", "header", "hmac"].map((t) => (
            <button key={t} onClick={() => setAuthType(t)} className={`px-2 py-0.5 text-[10px] rounded cursor-pointer transition-colors ${authType === t ? "bg-primary text-primary-foreground" : "bg-secondary text-secondary-foreground hover:bg-secondary/80"}`}>
              {t || "无认证"}
            </button>
          ))}
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
        <div className="flex gap-1">
          <button onClick={() => setScriptMode("plugin")} className={`px-2 py-0.5 text-[10px] rounded cursor-pointer transition-colors ${scriptMode === "plugin" ? "bg-primary text-primary-foreground" : "bg-secondary text-secondary-foreground hover:bg-secondary/80"}`}>
            插件市场
          </button>
          <button onClick={() => setScriptMode("manual")} className={`px-2 py-0.5 text-[10px] rounded cursor-pointer transition-colors ${scriptMode === "manual" ? "bg-primary text-primary-foreground" : "bg-secondary text-secondary-foreground hover:bg-secondary/80"}`}>
            手动脚本
          </button>
        </div>

        {scriptMode === "plugin" && (
          <div className="space-y-2">
            {pluginInfo ? (
              <div className="flex items-center justify-between p-2 rounded border bg-card">
                <div className="text-xs">
                  <span>{pluginInfo.icon} </span>
                  <span className="font-medium">{pluginInfo.name}</span>
                  <span className="text-muted-foreground ml-1">v{pluginInfo.version}</span>
                  <p className="text-[10px] text-muted-foreground mt-0.5">{pluginInfo.description}</p>
                </div>
                <div className="flex gap-1">
                  <Button variant="ghost" size="sm" className="h-6 text-[10px]" onClick={() => setShowPluginPicker(true)}>更换</Button>
                  <Button variant="ghost" size="sm" className="h-6 text-[10px] text-destructive" onClick={handleUninstallPlugin}>卸载</Button>
                </div>
              </div>
            ) : (
              <Button variant="outline" size="sm" className="w-full text-xs h-7" onClick={() => setShowPluginPicker(true)}>
                <Puzzle className="w-3 h-3 mr-1" /> 选择插件
              </Button>
            )}

            {showPluginPicker && (
              <div className="border rounded p-2 space-y-1 max-h-40 overflow-y-auto bg-card">
                {plugins.length === 0 && <p className="text-[10px] text-muted-foreground text-center py-2">暂无可用插件</p>}
                {plugins.map((p) => (
                  <button key={p.id} onClick={() => handleInstallPlugin(p.id)} className="w-full text-left p-1.5 rounded hover:bg-secondary cursor-pointer text-xs flex items-center justify-between">
                    <span>{p.icon} {p.name} <span className="text-muted-foreground">v{p.version}</span></span>
                    <span className="text-[10px] text-muted-foreground">{p.install_count} 安装</span>
                  </button>
                ))}
                <button onClick={() => setShowPluginPicker(false)} className="w-full text-center text-[10px] text-muted-foreground hover:text-primary cursor-pointer py-1">取消</button>
              </div>
            )}
          </div>
        )}

        {scriptMode === "manual" && (
          <textarea
            placeholder={`JS 中间件（可选）\n\nfunction onRequest(ctx) {\n  ctx.req.body = JSON.stringify({text: ctx.msg.content});\n}`}
            value={script}
            onChange={(e) => setScript(e.target.value)}
            rows={5}
            className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-[11px] font-mono placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:border-ring resize-none"
          />
        )}
      </div>
      <div className="flex items-center justify-between">
        <p className="text-[10px] text-muted-foreground">收到消息时 POST 到此 URL。</p>
        <div className="flex items-center gap-2">
          {error && <span className="text-[10px] text-destructive">{error}</span>}
          <Button size="sm" className="h-7" onClick={handleSave} disabled={saving}>{saving ? "..." : "保存"}</Button>
        </div>
      </div>
    </div>
  );
}

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

function BotSettingsTab({ bot, onUpdate }: { bot: any; onUpdate: () => void }) {
  const [reminderEnabled, setReminderEnabled] = useState(bot.reminder_hours > 0);
  const [reminderHours, setReminderHours] = useState(bot.reminder_hours || 23);
  const [saving, setSaving] = useState(false);

  async function handleSave() {
    setSaving(true);
    try {
      await api.updateBot(bot.id, { reminder_hours: reminderEnabled ? reminderHours : 0 } as any);
      onUpdate();
    } catch {}
    setSaving(false);
  }

  return (
    <div className="space-y-4 mt-4">
      <div className="p-4 rounded-lg border space-y-3">
        <h3 className="text-sm font-medium">会话保活提醒</h3>
        <p className="text-xs text-muted-foreground">
          微信会话 24 小时未活动将过期。开启后，当 Bot 在设定时间内没有收发任何消息时，系统会自动通过 Bot 发送一条提醒消息给你的微信，同时起到保活会话的作用。
        </p>
        <div className="flex items-center gap-3">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={reminderEnabled}
              onChange={(e) => setReminderEnabled(e.target.checked)}
              className="w-3.5 h-3.5 accent-primary"
            />
            <span className="text-sm">启用提醒</span>
          </label>
        </div>
        {reminderEnabled && (
          <>
            <div className="flex items-center gap-2">
              <span className="text-xs text-muted-foreground">无消息超过</span>
              <input
                type="number"
                value={reminderHours}
                onChange={(e) => setReminderHours(Math.max(1, Math.min(23, parseInt(e.target.value) || 23)))}
                className="w-16 h-7 rounded border px-2 text-xs text-center"
                min={1}
                max={23}
              />
              <span className="text-xs text-muted-foreground">小时后提醒</span>
            </div>
            <p className="text-[10px] text-muted-foreground">
              设为 {reminderHours} 小时：Bot 静默 {reminderHours} 小时后发送提醒，距 24 小时过期还剩约 {Math.max(1, 24 - reminderHours)} 小时。建议设为 23 小时（提前 1 小时提醒）。
            </p>
          </>
        )}
        <div className="flex justify-end">
          <Button size="sm" onClick={handleSave} disabled={saving}>
            {saving ? "保存中..." : "保存"}
          </Button>
        </div>
      </div>
    </div>
  );
}
