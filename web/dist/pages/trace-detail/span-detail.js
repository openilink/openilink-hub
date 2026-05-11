import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useRef, useState } from "react";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";
import { kindColors, durationMs, formatDuration, StatusIcon } from "@/lib/trace-utils";
import { Clock, Tag, Zap, Coins, XCircle } from "lucide-react";
function MediaPreview({ mediaKey, attrs }) {
  const [error, setError] = useState(false);
  const src = `/api/v1/media/${mediaKey}`;
  const replyType = String(attrs["reply.type"] || "");
  const isImage = replyType === "image" || /\.(jpg|jpeg|png|gif|webp)$/i.test(mediaKey);
  if (error) {
    return _jsx("a", {
      href: src,
      target: "_blank",
      rel: "noopener noreferrer",
      className: "text-primary hover:underline",
      children: mediaKey,
    });
  }
  if (isImage) {
    return _jsx("img", {
      src: src,
      alt: mediaKey,
      className: "max-w-[240px] max-h-[240px] rounded-md border mt-1",
      loading: "lazy",
      onError: () => setError(true),
    });
  }
  return _jsx("a", {
    href: src,
    target: "_blank",
    rel: "noopener noreferrer",
    className: "text-primary hover:underline",
    children: mediaKey,
  });
}
function Section({ title, icon, children }) {
  return _jsxs("div", {
    className: "space-y-2",
    children: [
      _jsxs("div", {
        className:
          "flex items-center gap-1.5 text-xs font-semibold text-muted-foreground uppercase tracking-wider",
        children: [icon, title],
      }),
      children,
    ],
  });
}
export function SpanDetail({ span, open, onClose }) {
  // Keep last span so the Sheet close animation can still render content
  const lastSpanRef = useRef(null);
  if (span) lastSpanRef.current = span;
  const displaySpan = span ?? lastSpanRef.current;
  if (!displaySpan) return null;
  const dur = durationMs(displaySpan);
  const attrs = displaySpan.attributes ? Object.entries(displaySpan.attributes) : [];
  const events = displaySpan.events || [];
  return _jsx(Sheet, {
    open: open,
    onOpenChange: (v) => !v && onClose(),
    children: _jsxs(SheetContent, {
      side: "right",
      className: "w-[400px] sm:max-w-[400px] p-0",
      children: [
        _jsxs(SheetHeader, {
          className: "p-4 pb-3 border-b",
          children: [
            _jsxs("div", {
              className: "flex items-center gap-2 mb-1",
              children: [
                _jsx(StatusIcon, { code: displaySpan.status_code, size: "w-4 h-4" }),
                _jsx(Badge, {
                  variant: "outline",
                  className: `text-[9px] h-4 px-1.5 leading-none text-white ${kindColors[displaySpan.kind] || "bg-gray-400"}`,
                  children: displaySpan.kind,
                }),
                _jsx("span", {
                  className: "text-[10px] font-mono text-muted-foreground",
                  children: formatDuration(dur),
                }),
              ],
            }),
            _jsx(SheetTitle, {
              className: "font-mono text-sm truncate",
              children: displaySpan.name,
            }),
            displaySpan.status_message &&
              _jsx(SheetDescription, {
                className: "text-destructive text-xs line-clamp-2",
                children: displaySpan.status_message,
              }),
          ],
        }),
        _jsx(ScrollArea, {
          className: "h-[calc(100vh-120px)] px-4 py-3",
          children: _jsxs("div", {
            className: "space-y-5",
            children: [
              displaySpan.status_message &&
                _jsx(Section, {
                  title: "Error",
                  icon: _jsx(XCircle, { className: "w-3 h-3 text-destructive" }),
                  children: _jsx("div", {
                    className:
                      "rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-xs text-destructive whitespace-pre-wrap break-words font-mono",
                    children: displaySpan.status_message,
                  }),
                }),
              _jsx(Section, {
                title: "Timing",
                icon: _jsx(Clock, { className: "w-3 h-3" }),
                children: _jsxs("div", {
                  className: "grid grid-cols-2 gap-2 text-xs",
                  children: [
                    _jsxs("div", {
                      className: "space-y-0.5",
                      children: [
                        _jsx("div", { className: "text-muted-foreground", children: "Start" }),
                        _jsx("div", {
                          className: "font-mono",
                          children: new Date(displaySpan.start_time).toLocaleTimeString([], {
                            hour: "2-digit",
                            minute: "2-digit",
                            second: "2-digit",
                          }),
                        }),
                      ],
                    }),
                    _jsxs("div", {
                      className: "space-y-0.5",
                      children: [
                        _jsx("div", { className: "text-muted-foreground", children: "End" }),
                        _jsx("div", {
                          className: "font-mono",
                          children: displaySpan.end_time
                            ? new Date(displaySpan.end_time).toLocaleTimeString([], {
                                hour: "2-digit",
                                minute: "2-digit",
                                second: "2-digit",
                              })
                            : "—",
                        }),
                      ],
                    }),
                    _jsxs("div", {
                      className: "space-y-0.5",
                      children: [
                        _jsx("div", { className: "text-muted-foreground", children: "Duration" }),
                        _jsx("div", {
                          className: "font-mono font-medium",
                          children: formatDuration(dur),
                        }),
                      ],
                    }),
                    _jsxs("div", {
                      className: "space-y-0.5",
                      children: [
                        _jsx("div", { className: "text-muted-foreground", children: "Span ID" }),
                        _jsx("div", {
                          className: "font-mono text-[10px] truncate",
                          children: displaySpan.span_id,
                        }),
                      ],
                    }),
                  ],
                }),
              }),
              displaySpan.attributes?.["ai.tokens.total"] &&
                _jsxs(Section, {
                  title: "Token Usage",
                  icon: _jsx(Coins, { className: "w-3 h-3" }),
                  children: [
                    displaySpan.attributes["ai.model"] &&
                      _jsxs("div", {
                        className: "text-xs text-muted-foreground mb-2",
                        children: [
                          "\u6A21\u578B: ",
                          _jsx("span", {
                            className: "font-mono font-medium text-foreground",
                            children: displaySpan.attributes["ai.model"],
                          }),
                        ],
                      }),
                    _jsxs("div", {
                      className: "grid grid-cols-3 gap-2 text-xs",
                      children: [
                        _jsxs("div", {
                          className: "space-y-0.5",
                          children: [
                            _jsx("div", { className: "text-muted-foreground", children: "Prompt" }),
                            _jsx("div", {
                              className: "font-mono font-medium",
                              children: displaySpan.attributes["ai.tokens.prompt"] || "0",
                            }),
                          ],
                        }),
                        _jsxs("div", {
                          className: "space-y-0.5",
                          children: [
                            _jsx("div", {
                              className: "text-muted-foreground",
                              children: "Completion",
                            }),
                            _jsx("div", {
                              className: "font-mono font-medium",
                              children: displaySpan.attributes["ai.tokens.completion"] || "0",
                            }),
                          ],
                        }),
                        _jsxs("div", {
                          className: "space-y-0.5",
                          children: [
                            _jsx("div", { className: "text-muted-foreground", children: "Total" }),
                            _jsx("div", {
                              className: "font-mono font-medium",
                              children: displaySpan.attributes["ai.tokens.total"],
                            }),
                          ],
                        }),
                        displaySpan.attributes["ai.tokens.cached"] &&
                          _jsxs("div", {
                            className: "space-y-0.5",
                            children: [
                              _jsx("div", {
                                className: "text-muted-foreground",
                                children: "Cached",
                              }),
                              _jsx("div", {
                                className: "font-mono font-medium",
                                children: displaySpan.attributes["ai.tokens.cached"],
                              }),
                            ],
                          }),
                        displaySpan.attributes["ai.tokens.reasoning"] &&
                          _jsxs("div", {
                            className: "space-y-0.5",
                            children: [
                              _jsx("div", {
                                className: "text-muted-foreground",
                                children: "Reasoning",
                              }),
                              _jsx("div", {
                                className: "font-mono font-medium",
                                children: displaySpan.attributes["ai.tokens.reasoning"],
                              }),
                            ],
                          }),
                      ],
                    }),
                  ],
                }),
              attrs.length > 0 &&
                _jsx(Section, {
                  title: "Attributes",
                  icon: _jsx(Tag, { className: "w-3 h-3" }),
                  children: _jsx("div", {
                    className: "rounded-md border bg-muted/30 overflow-hidden",
                    children: attrs.map(([key, value], i) =>
                      _jsxs(
                        "div",
                        {
                          className: `flex gap-3 px-3 py-1.5 text-xs ${i < attrs.length - 1 ? "border-b border-border/50" : ""}`,
                          children: [
                            _jsx("span", {
                              className: "text-blue-500 font-semibold shrink-0 min-w-[100px]",
                              children: key,
                            }),
                            _jsx("span", {
                              className: "text-foreground/80 whitespace-pre-wrap break-words",
                              children:
                                key === "reply.media_key"
                                  ? _jsx(MediaPreview, {
                                      mediaKey: String(value),
                                      attrs: Object.fromEntries(attrs),
                                    })
                                  : String(value),
                            }),
                          ],
                        },
                        key,
                      ),
                    ),
                  }),
                }),
              events.length > 0 &&
                _jsx(Section, {
                  title: "Events",
                  icon: _jsx(Zap, { className: "w-3 h-3" }),
                  children: _jsx("div", {
                    className: "space-y-2",
                    children: events.map((evt, i) =>
                      _jsxs(
                        "div",
                        {
                          className: "rounded-md border bg-muted/30 px-3 py-2",
                          children: [
                            _jsxs("div", {
                              className: "flex items-center justify-between mb-1",
                              children: [
                                _jsx("span", {
                                  className: "text-xs font-semibold",
                                  children: evt.name,
                                }),
                                _jsx("span", {
                                  className: "text-[9px] font-mono text-muted-foreground",
                                  children: new Date(evt.timestamp).toLocaleTimeString([], {
                                    hour: "2-digit",
                                    minute: "2-digit",
                                    second: "2-digit",
                                  }),
                                }),
                              ],
                            }),
                            evt.attributes &&
                              Object.keys(evt.attributes).length > 0 &&
                              _jsx("div", {
                                className: "space-y-0.5 mt-1",
                                children: Object.entries(evt.attributes).map(([k, v]) =>
                                  _jsxs(
                                    "div",
                                    {
                                      className: "flex gap-2 text-[10px]",
                                      children: [
                                        _jsxs("span", {
                                          className: "text-blue-500 font-bold shrink-0",
                                          children: [k, ":"],
                                        }),
                                        _jsx("span", {
                                          className:
                                            "text-foreground/70 whitespace-pre-wrap break-words",
                                          children: String(v),
                                        }),
                                      ],
                                    },
                                    k,
                                  ),
                                ),
                              }),
                          ],
                        },
                        i,
                      ),
                    ),
                  }),
                }),
            ],
          }),
        }),
      ],
    }),
  });
}
