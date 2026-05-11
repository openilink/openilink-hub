import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState, useMemo } from "react";
import { ChevronRight, ChevronDown } from "lucide-react";
import {
  kindColors,
  kindBgLight,
  durationMs,
  formatDuration,
  buildTree,
  flattenSpans,
  StatusIcon,
} from "@/lib/trace-utils";
const TICK_COUNT = 4;
export function TimelineView({ spans, selectedSpanId, onSelectSpan }) {
  const [collapsed, setCollapsed] = useState(new Set());
  const tree = useMemo(() => buildTree(spans), [spans]);
  const rows = useMemo(() => flattenSpans(spans, tree, collapsed), [spans, tree, collapsed]);
  const traceStart = useMemo(
    () => (spans.length > 0 ? Math.min(...spans.map((s) => s.start_time)) : 0),
    [spans],
  );
  const traceEnd = useMemo(
    () => (spans.length > 0 ? Math.max(...spans.map((s) => s.end_time || s.start_time)) : 0),
    [spans],
  );
  const traceDuration = traceEnd - traceStart || 1;
  function toggleCollapse(spanId) {
    setCollapsed((prev) => {
      const next = new Set(prev);
      if (next.has(spanId)) next.delete(spanId);
      else next.add(spanId);
      return next;
    });
  }
  return _jsxs("div", {
    className: "space-y-0",
    children: [
      _jsxs("div", {
        className: "flex border-b border-border/50 pb-1 mb-1",
        children: [
          _jsx("div", {
            className: "w-[220px] shrink-0 text-[10px] text-muted-foreground font-medium px-2",
            children: "Span",
          }),
          _jsx("div", {
            className: "flex-1 relative h-5",
            children: Array.from({ length: TICK_COUNT + 1 }).map((_, i) => {
              const ms = (traceDuration * i) / TICK_COUNT;
              return _jsx(
                "span",
                {
                  className: "absolute text-[9px] text-muted-foreground font-mono -translate-x-1/2",
                  style: { left: `${(i / TICK_COUNT) * 100}%` },
                  children: formatDuration(ms),
                },
                i,
              );
            }),
          }),
        ],
      }),
      _jsxs("div", {
        className: "flex",
        children: [
          _jsx("div", {
            className: "w-[220px] shrink-0",
            children: rows.map(({ span, depth }) => {
              const children = tree.get(span.span_id) || [];
              const hasChildren = children.length > 0;
              const isCollapsed = collapsed.has(span.span_id);
              const isSelected = selectedSpanId === span.span_id;
              return _jsxs(
                "div",
                {
                  role: "button",
                  tabIndex: 0,
                  className: `flex items-center h-8 cursor-pointer transition-colors ${isSelected ? "bg-primary/10" : "hover:bg-muted/50"}`,
                  style: { paddingLeft: `${8 + depth * 16}px` },
                  onClick: () => onSelectSpan(span.span_id),
                  onKeyDown: (e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      onSelectSpan(span.span_id);
                    }
                  },
                  children: [
                    hasChildren
                      ? _jsx("button", {
                          className: "w-4 h-4 flex items-center justify-center shrink-0",
                          "aria-label": isCollapsed ? "展开" : "折叠",
                          onClick: (e) => {
                            e.stopPropagation();
                            toggleCollapse(span.span_id);
                          },
                          children: isCollapsed
                            ? _jsx(ChevronRight, { className: "w-3 h-3 text-muted-foreground" })
                            : _jsx(ChevronDown, { className: "w-3 h-3 text-muted-foreground" }),
                        })
                      : _jsx("div", { className: "w-4 h-4 shrink-0" }),
                    _jsx(StatusIcon, { code: span.status_code, size: "w-3 h-3" }),
                    _jsx("span", {
                      className: "text-[11px] font-mono truncate ml-1",
                      title:
                        span.status_code === "error" && span.status_message
                          ? `❌ ${span.status_message}`
                          : span.name,
                      children: span.name,
                    }),
                  ],
                },
                span.span_id,
              );
            }),
          }),
          _jsxs("div", {
            className: "flex-1 relative",
            children: [
              Array.from({ length: TICK_COUNT + 1 }).map((_, i) =>
                _jsx(
                  "div",
                  {
                    className: "absolute top-0 bottom-0 border-l border-dashed border-border/40",
                    style: { left: `${(i / TICK_COUNT) * 100}%` },
                  },
                  i,
                ),
              ),
              rows.map(({ span }) => {
                const dur = durationMs(span);
                const barLeft = ((span.start_time - traceStart) / traceDuration) * 100;
                const barWidth = Math.max((dur / traceDuration) * 100, 0.5);
                const isSelected = selectedSpanId === span.span_id;
                return _jsx(
                  "div",
                  {
                    className: `relative h-8 flex items-center cursor-pointer transition-colors ${isSelected ? "bg-primary/10" : "hover:bg-muted/50"}`,
                    onClick: () => onSelectSpan(span.span_id),
                    children: _jsxs("div", {
                      className: `absolute h-5 rounded-sm flex items-center px-1.5 ${span.status_code === "error" ? "bg-destructive/15" : kindBgLight[span.kind] || "bg-gray-500/15"} border ${span.status_code === "error" ? "border-destructive/50" : isSelected ? "border-primary/50" : "border-transparent"}`,
                      style: {
                        left: `${barLeft}%`,
                        width: `${barWidth}%`,
                        minWidth: "4px",
                      },
                      children: [
                        _jsx("div", {
                          className: `h-full rounded-sm absolute inset-y-0 left-0 ${span.status_code === "error" ? "bg-destructive" : kindColors[span.kind] || "bg-gray-500"} opacity-30`,
                          style: { width: "100%" },
                        }),
                        barWidth > 5 &&
                          _jsx("span", {
                            className:
                              "relative text-[9px] font-mono font-medium text-foreground/80 whitespace-nowrap",
                            children: formatDuration(dur),
                          }),
                      ],
                    }),
                  },
                  span.span_id,
                );
              }),
            ],
          }),
        ],
      }),
    ],
  });
}
