import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useMemo, useCallback, useEffect } from "react";
import {
  ReactFlow,
  Background,
  Controls,
  useNodesState,
  useEdgesState,
  Handle,
  Position,
  BackgroundVariant,
} from "@xyflow/react";
import dagre from "@dagrejs/dagre";
import "@xyflow/react/dist/style.css";
import { Badge } from "@/components/ui/badge";
import {
  kindColors,
  kindBorderColors,
  durationMs,
  formatDuration,
  StatusIcon,
} from "@/lib/trace-utils";
const NODE_WIDTH = 220;
const NODE_HEIGHT = 60;
function layoutGraph(nodes, edges) {
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: "TB", nodesep: 60, ranksep: 80 });
  for (const node of nodes) {
    g.setNode(node.id, { width: NODE_WIDTH, height: NODE_HEIGHT });
  }
  for (const edge of edges) {
    g.setEdge(edge.source, edge.target);
  }
  dagre.layout(g);
  return nodes.map((node) => {
    const pos = g.node(node.id);
    return {
      ...node,
      position: {
        x: pos.x - NODE_WIDTH / 2,
        y: pos.y - NODE_HEIGHT / 2,
      },
    };
  });
}
function SpanNodeComponent({ data }) {
  const span = data.span;
  const isSelected = data.isSelected;
  const dur = durationMs(span);
  return _jsxs("div", {
    className: `rounded-lg border-2 px-3 py-2 bg-card shadow-sm transition-colors cursor-pointer w-[220px] ${
      isSelected
        ? "border-primary ring-2 ring-primary/20"
        : span.status_code === "error"
          ? "border-destructive ring-1 ring-destructive/20"
          : kindBorderColors[span.kind] || "border-border"
    }`,
    children: [
      _jsx(Handle, {
        type: "target",
        position: Position.Top,
        className: "!bg-muted-foreground !w-2 !h-2",
      }),
      _jsxs("div", {
        className: "flex items-center gap-1.5 mb-1",
        children: [
          _jsx(StatusIcon, { code: span.status_code, size: "w-3 h-3" }),
          _jsx(Badge, {
            variant: "outline",
            className: `text-[8px] h-3.5 px-1 leading-none text-white ${kindColors[span.kind] || "bg-gray-400"}`,
            children: span.kind,
          }),
        ],
      }),
      _jsx("div", {
        className: "text-[11px] font-mono font-medium truncate",
        title: span.name,
        children: span.name,
      }),
      span.status_code === "error" && span.status_message
        ? _jsx("div", {
            className: "text-[9px] text-destructive font-mono mt-0.5 truncate",
            title: span.status_message,
            children: span.status_message,
          })
        : _jsx("div", {
            className: "text-[9px] text-muted-foreground font-mono mt-0.5",
            children: formatDuration(dur),
          }),
      _jsx(Handle, {
        type: "source",
        position: Position.Bottom,
        className: "!bg-muted-foreground !w-2 !h-2",
      }),
    ],
  });
}
const nodeTypes = {
  spanNode: SpanNodeComponent,
};
export function FlowView({ spans, selectedSpanId, onSelectSpan }) {
  // Expensive dagre layout — only recompute when spans change
  const { baseNodes, layoutEdges } = useMemo(() => {
    const nodes = spans.map((span) => ({
      id: span.span_id,
      type: "spanNode",
      position: { x: 0, y: 0 },
      data: { span, isSelected: false },
    }));
    const edges = spans
      .filter((s) => s.parent_span_id)
      .map((s) => ({
        id: `${s.parent_span_id}-${s.span_id}`,
        source: s.parent_span_id,
        target: s.span_id,
        animated: s.status_code === "error",
        style: { stroke: s.status_code === "error" ? "var(--destructive)" : "var(--border)" },
      }));
    const laid = layoutGraph(nodes, edges);
    return { baseNodes: laid, layoutEdges: edges };
  }, [spans]);
  // Cheap selection update — no dagre relayout
  const layoutNodes = useMemo(
    () =>
      baseNodes.map((node) => ({
        ...node,
        data: { ...node.data, isSelected: node.id === selectedSpanId },
      })),
    [baseNodes, selectedSpanId],
  );
  const [nodes, setNodes, onNodesChange] = useNodesState(layoutNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(layoutEdges);
  // Sync nodes when selection or layout changes
  useEffect(() => {
    setNodes(layoutNodes);
  }, [layoutNodes, setNodes]);
  // Sync edges when spans change
  useEffect(() => {
    setEdges(layoutEdges);
  }, [layoutEdges, setEdges]);
  const onNodeClick = useCallback(
    (_, node) => {
      onSelectSpan(node.id);
    },
    [onSelectSpan],
  );
  return _jsx("div", {
    className: "h-[500px] rounded-lg border bg-card/30 overflow-hidden",
    children: _jsxs(ReactFlow, {
      nodes: nodes,
      edges: edges,
      onNodesChange: onNodesChange,
      onEdgesChange: onEdgesChange,
      onNodeClick: onNodeClick,
      nodeTypes: nodeTypes,
      fitView: true,
      fitViewOptions: { padding: 0.2 },
      minZoom: 0.3,
      maxZoom: 1.5,
      children: [
        _jsx(Background, {
          variant: BackgroundVariant.Dots,
          gap: 16,
          size: 1,
          className: "!bg-transparent",
        }),
        _jsx(Controls, { className: "!bg-card !border-border !shadow-sm" }),
      ],
    }),
  });
}
