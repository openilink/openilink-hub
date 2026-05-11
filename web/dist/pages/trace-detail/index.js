import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useCallback, useEffect, useRef, useState, useMemo } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { ArrowLeft, Activity, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Skeleton } from "@/components/ui/skeleton";
import { api } from "@/lib/api";
import { kindColors, durationMs, formatDuration, StatusIcon } from "@/lib/trace-utils";
import { TimelineView } from "./timeline-view";
import { FlowView } from "./flow-view";
import { SpanDetail } from "./span-detail";
export function TraceDetailPage() {
  const { id: botId, traceId } = useParams();
  const navigate = useNavigate();
  const [spans, setSpans] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedSpanId, setSelectedSpanId] = useState(null);
  const fetchIdRef = useRef(0);
  const load = useCallback(async () => {
    if (!botId || !traceId) return;
    const id = ++fetchIdRef.current;
    setLoading(true);
    try {
      const data = await api.getTrace(botId, traceId);
      if (fetchIdRef.current !== id) return;
      setSpans(data || []);
    } catch (e) {
      console.error("Failed to load trace:", e);
      if (fetchIdRef.current !== id) return;
      setSpans([]);
    } finally {
      if (fetchIdRef.current === id) setLoading(false);
    }
  }, [botId, traceId]);
  useEffect(() => {
    load();
  }, [load]);
  const rootSpan = useMemo(() => spans.find((s) => !s.parent_span_id), [spans]);
  const selectedSpan = useMemo(
    () => (selectedSpanId ? (spans.find((s) => s.span_id === selectedSpanId) ?? null) : null),
    [spans, selectedSpanId],
  );
  const totalDuration = useMemo(() => {
    if (!rootSpan) return 0;
    return durationMs(rootSpan);
  }, [rootSpan]);
  if (!botId || !traceId) return null;
  return _jsxs("div", {
    className: "flex flex-col gap-5",
    children: [
      _jsxs("div", {
        className: "flex items-center justify-between",
        children: [
          _jsxs("div", {
            className: "flex items-center gap-3",
            children: [
              _jsxs(Button, {
                variant: "outline",
                size: "sm",
                className: "rounded-full px-4 font-bold text-xs",
                onClick: () => navigate(`/dashboard/accounts/${botId}/traces`),
                children: [
                  _jsx(ArrowLeft, { className: "h-3.5 w-3.5 mr-1" }),
                  "\u8FD4\u56DE\u5217\u8868",
                ],
              }),
              rootSpan &&
                _jsxs("div", {
                  className: "flex items-center gap-2",
                  children: [
                    _jsx(StatusIcon, { code: rootSpan.status_code, size: "w-4 h-4" }),
                    _jsx(Badge, {
                      variant: "outline",
                      className: `text-[9px] h-4 px-1.5 leading-none text-white ${kindColors[rootSpan.kind] || "bg-gray-400"}`,
                      children: rootSpan.kind,
                    }),
                    _jsx("span", {
                      className: "text-sm font-mono font-medium",
                      children: rootSpan.attributes?.["message.sender"] || rootSpan.name,
                    }),
                    _jsx("span", {
                      className: "text-xs text-muted-foreground font-mono",
                      children: formatDuration(totalDuration),
                    }),
                    rootSpan.attributes?.["ai.tokens.total"] &&
                      _jsxs(Badge, {
                        variant: "outline",
                        className: "text-[10px] h-4 px-1.5 leading-none font-mono",
                        children: [rootSpan.attributes["ai.tokens.total"], " tokens"],
                      }),
                  ],
                }),
            ],
          }),
          _jsxs("div", {
            className: "flex items-center gap-2",
            children: [
              _jsx("span", {
                className: "text-[10px] font-mono text-muted-foreground hidden md:block",
                children: traceId,
              }),
              _jsxs(Badge, {
                variant: "secondary",
                className: "text-[10px] h-5 font-mono",
                children: [spans.length, " spans"],
              }),
              _jsxs(Button, {
                variant: "outline",
                size: "sm",
                onClick: load,
                disabled: loading,
                className: "h-8",
                children: [
                  _jsx(RefreshCw, {
                    className: `w-3.5 h-3.5 mr-1.5 ${loading ? "animate-spin" : ""}`,
                  }),
                  "\u5237\u65B0",
                ],
              }),
            ],
          }),
        ],
      }),
      loading
        ? _jsxs("div", {
            className: "space-y-3",
            children: [
              _jsx(Skeleton, { className: "h-10 w-64" }),
              _jsx(Skeleton, { className: "h-[400px] w-full" }),
            ],
          })
        : spans.length === 0
          ? _jsxs("div", {
              className: "flex flex-col items-center justify-center h-64 text-muted-foreground",
              children: [
                _jsx(Activity, { className: "w-8 h-8 mb-2 opacity-50" }),
                _jsx("p", {
                  className: "text-sm italic",
                  children: "\u672A\u627E\u5230\u8FFD\u8E2A\u6570\u636E",
                }),
              ],
            })
          : _jsxs(Tabs, {
              defaultValue: "timeline",
              className: "w-full",
              children: [
                _jsxs(TabsList, {
                  children: [
                    _jsx(TabsTrigger, {
                      value: "timeline",
                      className: "text-xs",
                      children: "Timeline",
                    }),
                    _jsx(TabsTrigger, { value: "flow", className: "text-xs", children: "Flow" }),
                  ],
                }),
                _jsx(TabsContent, {
                  value: "timeline",
                  children: _jsx("div", {
                    className: "rounded-xl border bg-card/50 p-4 shadow-sm overflow-x-auto",
                    children: _jsx(TimelineView, {
                      spans: spans,
                      selectedSpanId: selectedSpanId,
                      onSelectSpan: setSelectedSpanId,
                    }),
                  }),
                }),
                _jsx(TabsContent, {
                  value: "flow",
                  children: _jsx(FlowView, {
                    spans: spans,
                    selectedSpanId: selectedSpanId,
                    onSelectSpan: setSelectedSpanId,
                  }),
                }),
              ],
            }),
      _jsx(SpanDetail, {
        span: selectedSpan,
        open: !!selectedSpan,
        onClose: () => setSelectedSpanId(null),
      }),
    ],
  });
}
