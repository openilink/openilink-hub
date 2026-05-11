import { CheckCircle2, XCircle, MinusCircle } from "lucide-react";
import { createElement } from "react";
export const kindColors = {
  internal: "bg-slate-500",
  client: "bg-blue-500",
  server: "bg-green-500",
};
export const kindTextColors = {
  internal: "text-slate-500",
  client: "text-blue-500",
  server: "text-green-500",
};
export const kindBorderColors = {
  internal: "border-slate-400",
  client: "border-blue-400",
  server: "border-green-400",
};
export const kindBgLight = {
  internal: "bg-slate-500/15",
  client: "bg-blue-500/15",
  server: "bg-green-500/15",
};
export function durationMs(span) {
  return span.end_time > span.start_time ? span.end_time - span.start_time : 0;
}
export function formatDuration(ms) {
  if (ms < 1) return "<1ms";
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}
export function buildTree(spans) {
  const children = new Map();
  for (const s of spans) {
    const parentKey = s.parent_span_id || "";
    if (!children.has(parentKey)) children.set(parentKey, []);
    children.get(parentKey).push(s);
  }
  return children;
}
export function StatusIcon({ code, size = "w-4 h-4" }) {
  if (code === "ok")
    return createElement(CheckCircle2, { className: `${size} text-green-500 shrink-0` });
  if (code === "error")
    return createElement(XCircle, { className: `${size} text-destructive shrink-0` });
  return createElement(MinusCircle, { className: `${size} text-muted-foreground shrink-0` });
}
/** Flatten tree into display order (DFS) with depth info */
export function flattenSpans(spans, tree, collapsed) {
  const roots = spans.filter((s) => !s.parent_span_id);
  const result = [];
  const spanMap = new Map();
  for (const s of spans) spanMap.set(s.span_id, s);
  function walk(spanId, depth) {
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
