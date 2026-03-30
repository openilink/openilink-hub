import { useEffect, useLayoutEffect, useRef, useState, useCallback } from "react";
import { useParams, Link } from "react-router-dom";
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
  ChevronDown,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { api } from "@/lib/api";
import { useBotPush, usePushListener } from "@/lib/ws";
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
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [sendError, setSendError] = useState("");
  const [loadError, setLoadError] = useState("");
  const [loading, setLoading] = useState(true);
  const [canSend, setCanSend] = useState(true);
  const [sendDisabledReason, setSendDisabledReason] = useState<string>();
  const [stagedFile, setStagedFile] = useState<File | null>(null);
  const [stagedPreview, setStagedPreview] = useState<string | null>(null);
  const [dragOver, setDragOver] = useState(false);
  const [sending, setSending] = useState(false);
  const [isAtBottom, setIsAtBottom] = useState(true);
  const scrollRef = useRef<HTMLDivElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const stickToBottomRef = useRef(true);
  const isFirstLoadRef = useRef(true);
  const dragDepthRef = useRef(0);
  const stagedPreviewRef = useRef<string | null>(null);

  // Lock parent <main> scroll so console manages its own height
  useLayoutEffect(() => {
    const main = document.querySelector("main");
    if (!main) return;
    const prev = main.style.overflow;
    main.style.overflow = "hidden";
    return () => {
      main.style.overflow = prev;
    };
  }, []);

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
    } finally {
      setLoading(false);
    }
  }, [botId]);

  // Subscribe to push events for real-time updates.
  useBotPush(botId);
  usePushListener(useCallback((env) => {
    if (env.type === "message_new" && env.data?.bot_id === botId) {
      fetchData();
    }
  }, [botId, fetchData]));

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // Auto-scroll: instant on first load, smooth for new messages
  useEffect(() => {
    const el = scrollRef.current;
    if (!el || !stickToBottomRef.current) return;
    if (isFirstLoadRef.current) {
      el.scrollTop = el.scrollHeight;
      isFirstLoadRef.current = false;
    } else {
      el.scrollTo({ top: el.scrollHeight, behavior: "smooth" });
    }
  }, [messages]);

  const scrollToBottom = useCallback(() => {
    const el = scrollRef.current;
    if (!el) return;
    el.scrollTo({ top: el.scrollHeight, behavior: "smooth" });
    stickToBottomRef.current = true;
    setIsAtBottom(true);
  }, []);

  const handleScroll = useCallback(() => {
    const el = scrollRef.current;
    if (!el) return;
    const threshold = 80;
    const atBottom = el.scrollHeight - (el.scrollTop + el.clientHeight) <= threshold;
    stickToBottomRef.current = atBottom;
    setIsAtBottom(atBottom);
  }, []);

  // Stage file + generate preview (revoke old blob URL)
  const stageFile = useCallback((file: File) => {
    setStagedFile(file);
    setStagedPreview((prev) => {
      if (prev) URL.revokeObjectURL(prev);
      const next =
        file.type.startsWith("image/") || file.type.startsWith("video/")
          ? URL.createObjectURL(file)
          : null;
      stagedPreviewRef.current = next;
      return next;
    });
  }, []);

  const clearStaged = useCallback(() => {
    setStagedPreview((prev) => {
      if (prev) URL.revokeObjectURL(prev);
      stagedPreviewRef.current = null;
      return null;
    });
    setStagedFile(null);
  }, []);

  // Cleanup blob URL on unmount via ref (avoids stale closure)
  useEffect(() => {
    return () => {
      if (stagedPreviewRef.current) URL.revokeObjectURL(stagedPreviewRef.current);
    };
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
      className="relative flex flex-col h-full -m-6 lg:-m-8"
      onDragEnter={onDragEnter}
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      onDrop={onDrop}
    >
      {/* Header */}
      <div className="flex items-center gap-3 px-4 h-12 border-b bg-background/80 backdrop-blur-sm shrink-0">
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="icon-sm" asChild>
              <Link to={`/dashboard/accounts/${botId}`}>
                <ArrowLeft className="h-4 w-4" />
              </Link>
            </Button>
          </TooltipTrigger>
          <TooltipContent>返回账号详情</TooltipContent>
        </Tooltip>
        <Terminal className="h-4 w-4 text-muted-foreground" />
        <h1 className="text-sm font-semibold">消息控制台</h1>
        <Badge variant="outline" className="text-[10px]">
          实时推送
        </Badge>
      </div>

      {/* Drag overlay */}
      {dragOver ? (
        <div className="absolute inset-0 z-40 flex items-center justify-center bg-primary/5 border-2 border-dashed border-primary/30 rounded-lg pointer-events-none">
          <div className="text-center space-y-2">
            <Paperclip className="h-8 w-8 mx-auto text-primary/50" />
            <p className="text-sm font-medium text-primary/70">拖放文件到此处</p>
          </div>
        </div>
      ) : null}

      {/* Messages */}
      <div
        ref={scrollRef}
        onScroll={handleScroll}
        className="flex-1 overflow-y-auto px-6 py-4 bg-muted/20"
      >
        <div className="max-w-3xl mx-auto space-y-4">
          {loading ? (
            <div className="space-y-4 py-4">
              {["70%", "45%", "60%", "35%"].map((w, i) => (
                <div key={i} className={`flex ${i % 2 === 0 ? "justify-start" : "justify-end"}`}>
                  <Skeleton className={`h-12 rounded-2xl`} style={{ width: w }} />
                </div>
              ))}
            </div>
          ) : loadError ? (
            <div className="text-center py-4">
              <p className="text-sm text-destructive">{loadError}</p>
            </div>
          ) : messages.length === 0 ? (
            <div className="text-center py-20 text-muted-foreground">
              <Terminal className="h-10 w-10 mx-auto mb-3 opacity-20" />
              <p className="text-sm font-medium">暂无消息</p>
            </div>
          ) : null}
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

      {/* Scroll-to-bottom FAB */}
      {!isAtBottom ? (
        <div className="absolute bottom-20 right-8 z-10">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                type="button"
                size="icon"
                variant="secondary"
                className="rounded-full shadow-lg"
                onClick={scrollToBottom}
              >
                <ChevronDown className="h-4 w-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="left">回到底部</TooltipContent>
          </Tooltip>
        </div>
      ) : null}

      {/* Input area */}
      <div className="border-t bg-background shrink-0">
        <div className="max-w-3xl mx-auto px-4 py-3 space-y-2">
          {/* Status banners */}
          {!canSend ? (
            <div className="flex items-center gap-2 text-xs text-muted-foreground bg-muted/50 rounded-lg px-3 py-2">
              <Ban className="h-3.5 w-3.5 shrink-0" />
              <span>{sendDisabledReason || "当前无法发送消息"}</span>
            </div>
          ) : null}
          {sendError ? (
            <div className="flex items-center gap-2 text-xs text-destructive bg-destructive/5 border border-destructive/10 rounded-lg px-3 py-2">
              <Ban className="h-3.5 w-3.5 shrink-0" />
              <span>{sendError}</span>
            </div>
          ) : null}

          {/* Composer */}
          <form
            onSubmit={handleSend}
            className="flex flex-col rounded-2xl border border-border bg-background shadow-sm focus-within:ring-2 focus-within:ring-ring/30 transition-shadow"
          >
            {/* Staged file preview */}
            {stagedFile ? (
              <div className="flex items-center gap-3 px-3 pt-3 pb-2 border-b border-border/50">
                {stagedPreview && stagedFile.type.startsWith("image/") ? (
                  <img
                    src={stagedPreview}
                    alt="preview"
                    className="h-10 w-10 rounded-lg object-cover shrink-0"
                  />
                ) : stagedPreview && stagedFile.type.startsWith("video/") ? (
                  <video
                    src={stagedPreview}
                    className="h-10 w-10 rounded-lg object-cover shrink-0"
                  />
                ) : (
                  <div className="h-10 w-10 rounded-lg bg-muted flex items-center justify-center shrink-0">
                    {fileTypeIcon(stagedFile)}
                  </div>
                )}
                <div className="flex-1 min-w-0">
                  <p className="text-xs font-medium truncate">{stagedFile.name}</p>
                  <p className="text-[10px] text-muted-foreground">
                    {(stagedFile.size / 1024).toFixed(0)} KB
                  </p>
                </div>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 shrink-0"
                      onClick={clearStaged}
                      disabled={sending}
                    >
                      <X className="h-3.5 w-3.5" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>移除附件</TooltipContent>
                </Tooltip>
              </div>
            ) : null}

            {/* Input row */}
            <div className="flex items-center gap-1 px-2 py-1.5">
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
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 shrink-0 text-muted-foreground hover:text-foreground"
                    disabled={!canSend || sending}
                    onClick={() => fileInputRef.current?.click()}
                  >
                    <Paperclip className="h-4 w-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>添加附件</TooltipContent>
              </Tooltip>

              <label className="sr-only" htmlFor="console-msg-input">
                消息内容
              </label>
              <input
                id="console-msg-input"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && !e.shiftKey) {
                    e.preventDefault();
                    if (canSend && !sending && (input.trim() || stagedFile)) {
                      handleSend(e as any);
                    }
                  }
                }}
                placeholder={canSend ? "输入消息，Enter 发送..." : "无法发送"}
                disabled={!canSend || sending}
                className="flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50 py-1.5 px-1"
              />

              <Button
                type="submit"
                size="icon"
                className="h-8 w-8 shrink-0 rounded-xl"
                disabled={!canSend || sending || (!input.trim() && !stagedFile)}
              >
                <Send className="h-3.5 w-3.5" />
              </Button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}

function MessageContent({ m }: { m: Message }) {
  const items = m.item_list || [];
  if (items.length === 0) return <span className="text-muted-foreground italic">[空消息]</span>;
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
