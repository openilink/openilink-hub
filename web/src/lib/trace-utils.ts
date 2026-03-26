import { CheckCircle2, XCircle, MinusCircle } from "lucide-react";
import { createElement } from "react";

export interface TraceSpan {
  id: number;
  trace_id: string;
  span_id: string;
  parent_span_id: string;
  name: string;
  kind: string;
  status_code: string;
  status_message: string;
  start_time: number;
  end_time: number;
  attributes: Record<string, any> | null;
  events: {
    name: string;
    timestamp: number;
    attributes?: Record<string, any>;
  }[] | null;
  created_at: number;
}

export const kindColors: Record<string, string> = {
  internal: "bg-slate-500",
  client: "bg-blue-500",
  server: "bg-green-500",
};

export const kindTextColors: Record<string, string> = {
  internal: "text-slate-500",
  client: "text-blue-500",
  server: "text-green-500",
};

export const kindBorderColors: Record<string, string> = {
  internal: "border-slate-400",
  client: "border-blue-400",
  server: "border-green-400",
};

export const kindBgLight: Record<string, string> = {
  internal: "bg-slate-500/15",
  client: "bg-blue-500/15",
  server: "bg-green-500/15",
};

export function durationMs(span: TraceSpan): number {
  return span.end_time > span.start_time ? span.end_time - span.start_time : 0;
}

export function formatDuration(ms: number): string {
  if (ms < 1) return "<1ms";
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

export function buildTree(spans: TraceSpan[]): Map<string, TraceSpan[]> {
  const children = new Map<string, TraceSpan[]>();
  for (const s of spans) {
    const parentKey = s.parent_span_id || "";
    if (!children.has(parentKey)) children.set(parentKey, []);
    children.get(parentKey)!.push(s);
  }
  return children;
}

export function StatusIcon({ code, size = "w-4 h-4" }: { code: string; size?: string }) {
  if (code === "ok")
    return createElement(CheckCircle2, { className: `${size} text-green-500 shrink-0` });
  if (code === "error")
    return createElement(XCircle, { className: `${size} text-destructive shrink-0` });
  return createElement(MinusCircle, { className: `${size} text-muted-foreground shrink-0` });
}

/** Flatten tree into display order (DFS) with depth info */
export function flattenSpans(
  spans: TraceSpan[],
  tree: Map<string, TraceSpan[]>,
  collapsed: Set<string>,
): { span: TraceSpan; depth: number }[] {
  const roots = spans.filter((s) => !s.parent_span_id);
  const result: { span: TraceSpan; depth: number }[] = [];
  const spanMap = new Map<string, TraceSpan>();
  for (const s of spans) spanMap.set(s.span_id, s);

  function walk(spanId: string, depth: number) {
    const span = spanMap.get(spanId);
    if (!span) return;
    result.push({ span, depth });
    if (collapsed.has(spanId)) return;
    const children = tree.get(spanId) || [];
    for (const child of children) {
      walk(child.span_id, depth + 1);
    }
  }

  for (const root of roots) {
    walk(root.span_id, 0);
  }
  return result;
}
