import { useEffect, useRef, useState, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  ArrowLeft,
  Send,
  Terminal,
  Ban,
  Paperclip,
  X,
  Image as ImageIcon,
  Film,
  FileText,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { api } from "@/lib/api";
import { MessageItem, type MessageItemData } from "./message-items";

type Message = {
  id: number;
  bot_id?: string;
  direction: string;
  item_list: MessageItemData[];
  media_status?: string;
  media_keys?: Record<string, string>;
  created_at: number;
  _sending?: boolean;
  _error?: string;
};

export function ConsolePage() {
  const { id: botId } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [sendError, setSendError] = useState("");
  const [loadError, setLoadError] = useState("");
  const [canSend, setCanSend] = useState(true);
  const [sendDisabledReason, setSendDisabledReason] = useState<string>();
  const [stagedFile, setStagedFile] = useState<File | null>(null);
  const [stagedPreview, setStagedPreview] = useState<string | null>(null);
  const [dragOver, setDragOver] = useState(false);
  const [sending, setSending] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const stickToBottomRef = useRef(true);
  const dragDepthRef = useRef(0);

  const fetchData = useCallback(async () => {
    if (!botId) return;
    try {
      const res = await api.messages(botId, 50);
      setLoadError("");
      setMessages((res.messages || []).reverse());
      if (res.can_send !== undefined) {
        setCanSend(res.can_send);
        setSendDisabledReason(res.send_disabled_reason);
        if (res.can_send) setSendError("");
      }
    } catch (err: any) {
      setLoadError(err?.message || "消息加载失败");
    }
  }, [botId]);

  useEffect(() => {
    fetchData();
    const t = setInterval(fetchData, 5000);
    return () => clearInterval(t);
  }, [fetchData]);

  // Auto-scroll only when user is near bottom
  useEffect(() => {
    const el = scrollRef.current;
    if (el && stickToBottomRef.current) {
      el.scrollTop = el.scrollHeight;
    }
  }, [messages]);

  const handleScroll = useCallback(() => {
    const el = scrollRef.current;
    if (!el) return;
    const threshold = 80;
    stickToBottomRef.current =
      el.scrollHeight - (el.scrollTop + el.clientHeight) <= threshold;
  }, []);

  // Stage file + generate preview (revoke old blob URL)
  const stageFile = useCallback((file: File) => {
    setStagedFile(file);
    setStagedPreview((prev) => {
      if (prev) URL.revokeObjectURL(prev);
      return file.type.startsWith("image/") || file.type.startsWith("video/")
        ? URL.createObjectURL(file)
        : null;
    });
  }, []);

  const clearStaged = useCallback(() => {
    setStagedPreview((prev) => {
      if (prev) URL.revokeObjectURL(prev);
      return null;
    });
    setStagedFile(null);
  }, []);

  // Cleanup blob URL on unmount
  useEffect(() => {
    return () => {
      if (stagedPreview) URL.revokeObjectURL(stagedPreview);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Drag and drop handlers (track depth to avoid flicker on child elements)
  const onDragEnter = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    dragDepthRef.current++;
    if (dragDepthRef.current === 1) setDragOver(true);
  }, []);
  const onDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
  }, []);
  const onDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    dragDepthRef.current--;
    if (dragDepthRef.current === 0) setDragOver(false);
  }, []);
  const onDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      dragDepthRef.current = 0;
      setDragOver(false);
      const file = e.dataTransfer.files?.[0];
      if (file) stageFile(file);
    },
    [stageFile],
  );

  // Send message (text or file)
  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault();
    if (sending) return;
    const hasText = input.trim().length > 0;
    const hasFile = !!stagedFile;
    if (!hasText && !hasFile) return;
    if (!canSend) return;

    setSendError("");
    setSending(true);
    const text = input;

    try {
      if (hasFile && stagedFile) {
        const formData = new FormData();
        formData.append("file", stagedFile);
        if (hasText) formData.append("text", text);
        const r = await fetch(`/api/bots/${botId}/send`, {
          method: "POST",
          credentials: "same-origin",
          body: formData,
        });
        if (r.status === 401) {
          window.location.href = "/login";
          throw new Error("unauthorized");
        }
        if (!r.ok) {
          const data = await r.json().catch(() => ({}));
          throw new Error(data.error || `HTTP ${r.status}`);
        }
        clearStaged();
        setInput("");
      } else {
        await api.sendMessage(botId!, { text });
        setInput("");
      }
      fetchData();
    } catch (err: any) {
      setSendError(err?.message || "发送失败");
      setInput(text); // restore draft on error
    } finally {
      setSending(false);
    }
  };

  const fileTypeIcon = (file: File) => {
    if (file.type.startsWith("image/")) return <ImageIcon className="h-4 w-4" />;
    if (file.type.startsWith("video/")) return <Film className="h-4 w-4" />;
    return <FileText className="h-4 w-4" />;
  };

  if (!botId) return null;

  return (
    <div
      className="relative flex flex-col h-[calc(100dvh-4rem)] -m-6 lg:-m-8"
      onDragEnter={onDragEnter}
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      onDrop={onDrop}
    >
      {/* Header */}
      <div className="flex items-center gap-3 px-6 py-3 border-b bg-background/80 backdrop-blur-sm shrink-0">
        <Button
          variant="ghost"
          size="sm"
          className="rounded-full h-8 w-8 p-0"
          onClick={() => navigate(`/dashboard/accounts/${botId}`)}
          aria-label="返回账号详情"
        >
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <Terminal className="h-4 w-4 text-primary" />
        <h1 className="text-sm font-bold uppercase tracking-widest">
          实时控制台
        </h1>
        <Badge
          variant="outline"
          className="bg-background text-[10px] font-bold"
        >
          实时推送
        </Badge>
      </div>

      {/* Drag overlay */}
      {dragOver && (
        <div className="absolute inset-0 z-40 flex items-center justify-center bg-primary/5 border-2 border-dashed border-primary/30 rounded-lg pointer-events-none">
          <div className="text-center">
            <Paperclip className="h-10 w-10 mx-auto text-primary/50 mb-2" />
            <p className="text-sm font-bold text-primary/70">
              拖放文件到此处
            </p>
          </div>
        </div>
      )}

      {/* Messages */}
      <div
        ref={scrollRef}
        onScroll={handleScroll}
        className="flex-1 overflow-y-auto px-6 py-4 space-y-4"
      >
        <div className="max-w-3xl mx-auto space-y-4">
          {loadError && (
            <div className="text-center py-4">
              <p className="text-sm text-destructive">{loadError}</p>
            </div>
          )}
          {!loadError && messages.length === 0 && (
            <div className="text-center py-20 text-muted-foreground">
              <Terminal className="h-10 w-10 mx-auto mb-3 opacity-20" />
              <p className="text-sm font-medium">暂无消息</p>
            </div>
          )}
          {messages.map((m) => (
            <div
              key={m.id}
              className={`flex ${m.direction === "inbound" ? "justify-start" : "justify-end"}`}
            >
              <div
                className={`max-w-[75%] px-4 py-3 rounded-2xl text-sm font-medium ${
                  m.direction === "inbound"
                    ? "bg-background border border-border/50 text-foreground rounded-bl-none shadow-sm"
                    : "bg-primary text-primary-foreground rounded-br-none shadow-lg shadow-primary/10"
                }`}
              >
                <MessageContent m={m} />
                <p
                  className={`text-[9px] mt-1.5 font-bold uppercase opacity-40 ${
                    m.direction === "inbound" ? "text-left" : "text-right"
                  }`}
                >
                  {new Date(m.created_at * 1000).toLocaleTimeString()}
                </p>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Input area */}
      <div className="border-t bg-background/80 backdrop-blur-sm shrink-0">
        <div className="max-w-3xl mx-auto px-6 py-3 space-y-2">
          {!canSend && (
            <div className="flex items-center gap-2 text-xs text-muted-foreground bg-muted/50 rounded-lg px-3 py-2">
              <Ban className="h-3.5 w-3.5 shrink-0" />
              <span>{sendDisabledReason || "当前无法发送消息"}</span>
            </div>
          )}
          {sendError && (
            <div className="text-xs text-destructive bg-destructive/10 rounded-lg px-3 py-2">
              {sendError}
            </div>
          )}

          {/* Staged file preview */}
          {stagedFile && (
            <div className="flex items-center gap-3 bg-muted/50 rounded-xl px-3 py-2">
              {stagedPreview && stagedFile.type.startsWith("image/") ? (
                <img
                  src={stagedPreview}
                  alt="preview"
                  className="h-12 w-12 rounded-lg object-cover"
                />
              ) : stagedPreview && stagedFile.type.startsWith("video/") ? (
                <video
                  src={stagedPreview}
                  className="h-12 w-12 rounded-lg object-cover"
                />
              ) : (
                <div className="h-12 w-12 rounded-lg bg-muted flex items-center justify-center">
                  {fileTypeIcon(stagedFile)}
                </div>
              )}
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium truncate">
                  {stagedFile.name}
                </p>
                <p className="text-xs text-muted-foreground">
                  {(stagedFile.size / 1024).toFixed(0)} KB
                </p>
              </div>
              <button
                type="button"
                onClick={clearStaged}
                disabled={sending}
                className="p-1 rounded-full hover:bg-muted disabled:opacity-50"
                aria-label="移除附件"
              >
                <X className="h-4 w-4 text-muted-foreground" />
              </button>
            </div>
          )}

          <form className="flex items-center gap-2" onSubmit={handleSend}>
            <input
              ref={fileInputRef}
              type="file"
              className="hidden"
              onChange={(e) => {
                const file = e.target.files?.[0];
                if (file) stageFile(file);
                e.target.value = "";
              }}
            />
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-10 w-10 p-0 rounded-xl shrink-0"
              disabled={!canSend || sending}
              onClick={() => fileInputRef.current?.click()}
              aria-label="添加附件"
            >
              <Paperclip className="h-4 w-4" />
            </Button>
            <label className="sr-only" htmlFor="console-msg-input">消息内容</label>
            <input
              id="console-msg-input"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder={canSend ? "输入消息..." : "无法发送"}
              disabled={!canSend || sending}
              className="flex-1 h-10 rounded-xl bg-muted/50 border-none px-4 text-sm outline-none focus:ring-2 focus:ring-primary/20 transition-shadow"
            />
            <Button
              type="submit"
              disabled={!canSend || sending || (!input.trim() && !stagedFile)}
              className="h-10 rounded-xl px-5 gap-2 font-bold shadow-lg shadow-primary/20 shrink-0"
              aria-label="发送消息"
            >
              发送 <Send className="h-4 w-4" />
            </Button>
          </form>
        </div>
      </div>
    </div>
  );
}

function MessageContent({ m }: { m: Message }) {
  const items = m.item_list || [];
  if (items.length === 0)
    return <span className="text-muted-foreground italic">[空消息]</span>;
  return (
    <div className="space-y-2">
      {items.map((item, i) => (
        <MessageItem
          key={`${item.type}-${i}`}
          item={item}
          index={i}
          mediaKeys={m.media_keys}
          mediaStatus={m.media_status}
          direction={m.direction}
        />
      ))}
    </div>
  );
}
