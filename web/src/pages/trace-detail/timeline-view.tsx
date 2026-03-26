import { useState, useMemo } from "react";
import { ChevronRight, ChevronDown } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  TraceSpan,
  kindColors,
  kindBgLight,
  durationMs,
  formatDuration,
  buildTree,
  flattenSpans,
  StatusIcon,
} from "@/lib/trace-utils";

interface TimelineViewProps {
  spans: TraceSpan[];
  selectedSpanId: string | null;
  onSelectSpan: (spanId: string) => void;
}

function TimelineTicks({ count }: { count: number }) {
  return (
    <div className="absolute inset-0 flex">
      {Array.from({ length: count + 1 }).map((_, i) => (
        <div
          key={i}
          className="absolute top-0 bottom-0 border-l border-dashed border-border/40"
          style={{ left: `${(i / count) * 100}%` }}
        />
      ))}
    </div>
  );
}

export function TimelineView({ spans, selectedSpanId, onSelectSpan }: TimelineViewProps) {
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set());

  const tree = useMemo(() => buildTree(spans), [spans]);
  const rows = useMemo(() => flattenSpans(spans, tree, collapsed), [spans, tree, collapsed]);

  const traceStart = useMemo(
    () => Math.min(...spans.map((s) => s.start_time)),
    [spans],
  );
  const traceEnd = useMemo(
    () => Math.max(...spans.map((s) => s.end_time || s.start_time)),
    [spans],
  );
  const traceDuration = traceEnd - traceStart || 1;
  const tickCount = 4;

  function toggleCollapse(spanId: string) {
    setCollapsed((prev) => {
      const next = new Set(prev);
      if (next.has(spanId)) next.delete(spanId);
      else next.add(spanId);
      return next;
    });
  }

  return (
    <div className="space-y-0">
      {/* Time axis header */}
      <div className="flex border-b border-border/50 pb-1 mb-1">
        <div className="w-[220px] shrink-0 text-[10px] text-muted-foreground font-medium px-2">
          Span
        </div>
        <div className="flex-1 relative h-5">
          {Array.from({ length: tickCount + 1 }).map((_, i) => {
            const ms = (traceDuration * i) / tickCount;
            return (
              <span
                key={i}
                className="absolute text-[9px] text-muted-foreground font-mono -translate-x-1/2"
                style={{ left: `${(i / tickCount) * 100}%` }}
              >
                {formatDuration(ms)}
              </span>
            );
          })}
        </div>
      </div>

      {/* Span rows */}
      <div className="space-y-0">
        {rows.map(({ span, depth }) => {
          const children = tree.get(span.span_id) || [];
          const hasChildren = children.length > 0;
          const isCollapsed = collapsed.has(span.span_id);
          const dur = durationMs(span);
          const barLeft = ((span.start_time - traceStart) / traceDuration) * 100;
          const barWidth = Math.max((dur / traceDuration) * 100, 0.5);
          const isSelected = selectedSpanId === span.span_id;

          return (
            <div
              key={span.span_id}
              className={`flex items-center h-8 cursor-pointer transition-colors group ${
                isSelected
                  ? "bg-primary/10 hover:bg-primary/15"
                  : "hover:bg-muted/50"
              }`}
              onClick={() => onSelectSpan(span.span_id)}
            >
              {/* Label column */}
              <div
                className="w-[220px] shrink-0 flex items-center gap-1 px-2 overflow-hidden"
                style={{ paddingLeft: `${8 + depth * 16}px` }}
              >
                <button
                  className="w-4 h-4 flex items-center justify-center shrink-0"
                  onClick={(e) => {
                    e.stopPropagation();
                    if (hasChildren) toggleCollapse(span.span_id);
                  }}
                >
                  {hasChildren &&
                    (isCollapsed ? (
                      <ChevronRight className="w-3 h-3 text-muted-foreground" />
                    ) : (
                      <ChevronDown className="w-3 h-3 text-muted-foreground" />
                    ))}
                </button>
                <StatusIcon code={span.status_code} size="w-3 h-3" />
                <span className="text-[11px] font-mono truncate">{span.name}</span>
              </div>

              {/* Bar column */}
              <div className="flex-1 relative h-full flex items-center">
                <TimelineTicks count={tickCount} />
                <div
                  className={`absolute h-5 rounded-sm flex items-center px-1.5 ${
                    kindBgLight[span.kind] || "bg-gray-500/15"
                  } border ${
                    isSelected
                      ? "border-primary/50"
                      : `border-transparent`
                  }`}
                  style={{
                    left: `${barLeft}%`,
                    width: `${barWidth}%`,
                    minWidth: "4px",
                  }}
                >
                  <div
                    className={`h-full rounded-sm absolute inset-y-0 left-0 ${
                      kindColors[span.kind] || "bg-gray-500"
                    } opacity-30`}
                    style={{ width: "100%" }}
                  />
                  {barWidth > 5 && (
                    <span className="relative text-[9px] font-mono font-medium text-foreground/80 whitespace-nowrap">
                      {formatDuration(dur)}
                    </span>
                  )}
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
