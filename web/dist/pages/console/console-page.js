import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect, useRef, useState, useCallback } from "react";
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
import { MessageItem } from "./message-items";
export function ConsolePage() {
  const { id: botId } = useParams();
  const [messages, setMessages] = useState([]);
  const [input, setInput] = useState("");
  const [sendError, setSendError] = useState("");
  const [loadError, setLoadError] = useState("");
  const [loading, setLoading] = useState(true);
  const [canSend, setCanSend] = useState(true);
  const [sendDisabledReason, setSendDisabledReason] = useState();
  const [stagedFile, setStagedFile] = useState(null);
  const [stagedPreview, setStagedPreview] = useState(null);
  const [dragOver, setDragOver] = useState(false);
  const [sending, setSending] = useState(false);
  const [isAtBottom, setIsAtBottom] = useState(true);
  const scrollRef = useRef(null);
  const fileInputRef = useRef(null);
  const stickToBottomRef = useRef(true);
  const isFirstLoadRef = useRef(true);
  const dragDepthRef = useRef(0);
  const stagedPreviewRef = useRef(null);
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
    } catch (err) {
      setLoadError(err?.message || "消息加载失败");
    } finally {
      setLoading(false);
    }
  }, [botId]);
  // Subscribe to push events for real-time updates.
  useBotPush(botId);
  usePushListener(
    useCallback(
      (env) => {
        if (env.type === "message_new" && env.data?.bot_id === botId) {
          fetchData();
        }
      },
      [botId, fetchData],
    ),
  );
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
  const stageFile = useCallback((file) => {
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
  const onDragEnter = useCallback((e) => {
    e.preventDefault();
    dragDepthRef.current++;
    if (dragDepthRef.current === 1) setDragOver(true);
  }, []);
  const onDragOver = useCallback((e) => {
    e.preventDefault();
  }, []);
  const onDragLeave = useCallback((e) => {
    e.preventDefault();
    dragDepthRef.current--;
    if (dragDepthRef.current === 0) setDragOver(false);
  }, []);
  const onDrop = useCallback(
    (e) => {
      e.preventDefault();
      dragDepthRef.current = 0;
      setDragOver(false);
      const file = e.dataTransfer.files?.[0];
      if (file) stageFile(file);
    },
    [stageFile],
  );
  // Send message (text or file)
  const handleSend = async (e) => {
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
        await api.sendMessage(botId, { text });
        setInput("");
      }
      fetchData();
    } catch (err) {
      setSendError(err?.message || "发送失败");
      setInput(text); // restore draft on error
    } finally {
      setSending(false);
    }
  };
  const fileTypeIcon = (file) => {
    if (file.type.startsWith("image/")) return _jsx(ImageIcon, { className: "h-4 w-4" });
    if (file.type.startsWith("video/")) return _jsx(Film, { className: "h-4 w-4" });
    return _jsx(FileText, { className: "h-4 w-4" });
  };
  if (!botId) return null;
  return _jsxs("div", {
    "data-full-page": true,
    className: "relative flex flex-col h-full",
    onDragEnter: onDragEnter,
    onDragOver: onDragOver,
    onDragLeave: onDragLeave,
    onDrop: onDrop,
    children: [
      _jsxs("div", {
        className:
          "flex items-center gap-3 px-4 h-12 border-b bg-background/80 backdrop-blur-sm shrink-0",
        children: [
          _jsxs(Tooltip, {
            children: [
              _jsx(TooltipTrigger, {
                asChild: true,
                children: _jsx(Button, {
                  variant: "ghost",
                  size: "icon-sm",
                  asChild: true,
                  children: _jsx(Link, {
                    to: `/dashboard/accounts/${botId}`,
                    children: _jsx(ArrowLeft, { className: "h-4 w-4" }),
                  }),
                }),
              }),
              _jsx(TooltipContent, { children: "\u8FD4\u56DE\u8D26\u53F7\u8BE6\u60C5" }),
            ],
          }),
          _jsx(Terminal, { className: "h-4 w-4 text-muted-foreground" }),
          _jsx("h1", {
            className: "text-sm font-semibold",
            children: "\u6D88\u606F\u63A7\u5236\u53F0",
          }),
          _jsx(Badge, {
            variant: "outline",
            className: "text-[10px]",
            children: "\u5B9E\u65F6\u63A8\u9001",
          }),
        ],
      }),
      dragOver
        ? _jsx("div", {
            className:
              "absolute inset-0 z-40 flex items-center justify-center bg-primary/5 border-2 border-dashed border-primary/30 rounded-lg pointer-events-none",
            children: _jsxs("div", {
              className: "text-center space-y-2",
              children: [
                _jsx(Paperclip, { className: "h-8 w-8 mx-auto text-primary/50" }),
                _jsx("p", {
                  className: "text-sm font-medium text-primary/70",
                  children: "\u62D6\u653E\u6587\u4EF6\u5230\u6B64\u5904",
                }),
              ],
            }),
          })
        : null,
      _jsx("div", {
        ref: scrollRef,
        onScroll: handleScroll,
        className: "flex-1 overflow-y-auto px-6 py-4 bg-muted/20",
        children: _jsxs("div", {
          className: "max-w-3xl mx-auto space-y-4",
          children: [
            loading
              ? _jsx("div", {
                  className: "space-y-4 py-4",
                  children: ["70%", "45%", "60%", "35%"].map((w, i) =>
                    _jsx(
                      "div",
                      {
                        className: `flex ${i % 2 === 0 ? "justify-start" : "justify-end"}`,
                        children: _jsx(Skeleton, {
                          className: `h-12 rounded-2xl`,
                          style: { width: w },
                        }),
                      },
                      i,
                    ),
                  ),
                })
              : loadError
                ? _jsx("div", {
                    className: "text-center py-4",
                    children: _jsx("p", {
                      className: "text-sm text-destructive",
                      children: loadError,
                    }),
                  })
                : messages.length === 0
                  ? _jsxs("div", {
                      className: "text-center py-20 text-muted-foreground",
                      children: [
                        _jsx(Terminal, { className: "h-10 w-10 mx-auto mb-3 opacity-20" }),
                        _jsx("p", {
                          className: "text-sm font-medium",
                          children: "\u6682\u65E0\u6D88\u606F",
                        }),
                      ],
                    })
                  : null,
            messages.map((m) =>
              _jsx(
                "div",
                {
                  className: `flex ${m.direction === "inbound" ? "justify-start" : "justify-end"}`,
                  children: _jsxs("div", {
                    className: `max-w-[75%] px-4 py-3 rounded-2xl text-sm font-medium ${
                      m.direction === "inbound"
                        ? "bg-background border border-border/50 text-foreground rounded-bl-none shadow-sm"
                        : "bg-primary text-primary-foreground rounded-br-none shadow-lg shadow-primary/10"
                    }`,
                    children: [
                      _jsx(MessageContent, { m: m }),
                      _jsx("p", {
                        className: `text-[9px] mt-1.5 font-bold uppercase opacity-40 ${m.direction === "inbound" ? "text-left" : "text-right"}`,
                        children: new Date(m.created_at * 1000).toLocaleTimeString(),
                      }),
                    ],
                  }),
                },
                m.id,
              ),
            ),
          ],
        }),
      }),
      !isAtBottom
        ? _jsx("div", {
            className: "absolute bottom-20 right-8 z-10",
            children: _jsxs(Tooltip, {
              children: [
                _jsx(TooltipTrigger, {
                  asChild: true,
                  children: _jsx(Button, {
                    type: "button",
                    size: "icon",
                    variant: "secondary",
                    className: "rounded-full shadow-lg",
                    onClick: scrollToBottom,
                    children: _jsx(ChevronDown, { className: "h-4 w-4" }),
                  }),
                }),
                _jsx(TooltipContent, { side: "left", children: "\u56DE\u5230\u5E95\u90E8" }),
              ],
            }),
          })
        : null,
      _jsx("div", {
        className: "border-t bg-background shrink-0",
        children: _jsxs("div", {
          className: "max-w-3xl mx-auto px-4 py-3 space-y-2",
          children: [
            !canSend
              ? _jsxs("div", {
                  className:
                    "flex items-center gap-2 text-xs text-muted-foreground bg-muted/50 rounded-lg px-3 py-2",
                  children: [
                    _jsx(Ban, { className: "h-3.5 w-3.5 shrink-0" }),
                    _jsx("span", { children: sendDisabledReason || "当前无法发送消息" }),
                  ],
                })
              : null,
            sendError
              ? _jsxs("div", {
                  className:
                    "flex items-center gap-2 text-xs text-destructive bg-destructive/5 border border-destructive/10 rounded-lg px-3 py-2",
                  children: [
                    _jsx(Ban, { className: "h-3.5 w-3.5 shrink-0" }),
                    _jsx("span", { children: sendError }),
                  ],
                })
              : null,
            _jsxs("form", {
              onSubmit: handleSend,
              className:
                "flex flex-col rounded-2xl border border-border bg-background shadow-sm focus-within:ring-2 focus-within:ring-ring/30 transition-shadow",
              children: [
                stagedFile
                  ? _jsxs("div", {
                      className: "flex items-center gap-3 px-3 pt-3 pb-2 border-b border-border/50",
                      children: [
                        stagedPreview && stagedFile.type.startsWith("image/")
                          ? _jsx("img", {
                              src: stagedPreview,
                              alt: "preview",
                              className: "h-10 w-10 rounded-lg object-cover shrink-0",
                            })
                          : stagedPreview && stagedFile.type.startsWith("video/")
                            ? _jsx("video", {
                                src: stagedPreview,
                                className: "h-10 w-10 rounded-lg object-cover shrink-0",
                              })
                            : _jsx("div", {
                                className:
                                  "h-10 w-10 rounded-lg bg-muted flex items-center justify-center shrink-0",
                                children: fileTypeIcon(stagedFile),
                              }),
                        _jsxs("div", {
                          className: "flex-1 min-w-0",
                          children: [
                            _jsx("p", {
                              className: "text-xs font-medium truncate",
                              children: stagedFile.name,
                            }),
                            _jsxs("p", {
                              className: "text-[10px] text-muted-foreground",
                              children: [(stagedFile.size / 1024).toFixed(0), " KB"],
                            }),
                          ],
                        }),
                        _jsxs(Tooltip, {
                          children: [
                            _jsx(TooltipTrigger, {
                              asChild: true,
                              children: _jsx(Button, {
                                type: "button",
                                variant: "ghost",
                                size: "icon",
                                className: "h-7 w-7 shrink-0",
                                onClick: clearStaged,
                                disabled: sending,
                                children: _jsx(X, { className: "h-3.5 w-3.5" }),
                              }),
                            }),
                            _jsx(TooltipContent, { children: "\u79FB\u9664\u9644\u4EF6" }),
                          ],
                        }),
                      ],
                    })
                  : null,
                _jsxs("div", {
                  className: "flex items-center gap-1 px-2 py-1.5",
                  children: [
                    _jsx("input", {
                      ref: fileInputRef,
                      type: "file",
                      className: "hidden",
                      onChange: (e) => {
                        const file = e.target.files?.[0];
                        if (file) stageFile(file);
                        e.target.value = "";
                      },
                    }),
                    _jsxs(Tooltip, {
                      children: [
                        _jsx(TooltipTrigger, {
                          asChild: true,
                          children: _jsx(Button, {
                            type: "button",
                            variant: "ghost",
                            size: "icon",
                            className:
                              "h-8 w-8 shrink-0 text-muted-foreground hover:text-foreground",
                            disabled: !canSend || sending,
                            onClick: () => fileInputRef.current?.click(),
                            children: _jsx(Paperclip, { className: "h-4 w-4" }),
                          }),
                        }),
                        _jsx(TooltipContent, { children: "\u6DFB\u52A0\u9644\u4EF6" }),
                      ],
                    }),
                    _jsx("label", {
                      className: "sr-only",
                      htmlFor: "console-msg-input",
                      children: "\u6D88\u606F\u5185\u5BB9",
                    }),
                    _jsx("input", {
                      id: "console-msg-input",
                      value: input,
                      onChange: (e) => setInput(e.target.value),
                      onKeyDown: (e) => {
                        if (e.key === "Enter" && !e.shiftKey) {
                          e.preventDefault();
                          if (canSend && !sending && (input.trim() || stagedFile)) {
                            handleSend(e);
                          }
                        }
                      },
                      placeholder: canSend ? "输入消息，Enter 发送..." : "无法发送",
                      disabled: !canSend || sending,
                      className:
                        "flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50 py-1.5 px-1",
                    }),
                    _jsx(Button, {
                      type: "submit",
                      size: "icon",
                      className: "h-8 w-8 shrink-0 rounded-xl",
                      disabled: !canSend || sending || (!input.trim() && !stagedFile),
                      children: _jsx(Send, { className: "h-3.5 w-3.5" }),
                    }),
                  ],
                }),
              ],
            }),
          ],
        }),
      }),
    ],
  });
}
function MessageContent({ m }) {
  const items = m.item_list || [];
  if (items.length === 0)
    return _jsx("span", {
      className: "text-muted-foreground italic",
      children: "[\u7A7A\u6D88\u606F]",
    });
  return _jsx("div", {
    className: "space-y-2",
    children: items.map((item, i) =>
      _jsx(
        MessageItem,
        {
          item: item,
          index: i,
          mediaKeys: m.media_keys,
          mediaStatus: m.media_status,
          direction: m.direction,
        },
        `${item.type}-${i}`,
      ),
    ),
  });
}
