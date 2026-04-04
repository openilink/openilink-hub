import { useMemo, useCallback, useEffect } from "react";
import {
  ReactFlow,
  Background,
  Controls,
  type Node,
  type Edge,
  type NodeTypes,
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
  TraceSpan,
  kindColors,
  kindBorderColors,
  durationMs,
  formatDuration,
  StatusIcon,
} from "@/lib/trace-utils";

interface FlowViewProps {
  spans: TraceSpan[];
  selectedSpanId: string | null;
  onSelectSpan: (spanId: string) => void;
}

const NODE_WIDTH = 220;
const NODE_HEIGHT = 60;

function layoutGraph(nodes: Node[], edges: Edge[]): Node[] {
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

function SpanNodeComponent({ data }: { data: any }) {
  const span = data.span as TraceSpan;
  const isSelected = data.isSelected as boolean;
  const dur = durationMs(span);

  return (
    <div
      className={`rounded-lg border-2 px-3 py-2 bg-card shadow-sm transition-colors cursor-pointer w-[220px] ${
        isSelected
          ? "border-primary ring-2 ring-primary/20"
          : kindBorderColors[span.kind] || "border-border"
      }`}
    >
      <Handle type="target" position={Position.Top} className="!bg-muted-foreground !w-2 !h-2" />
      <div className="flex items-center gap-1.5 mb-1">
        <StatusIcon code={span.status_code} size="w-3 h-3" />
        <Badge
          variant="outline"
          className={`text-[8px] h-3.5 px-1 leading-none text-white ${kindColors[span.kind] || "bg-gray-400"}`}
        >
          {span.kind}
        </Badge>
      </div>
      <div className="text-[11px] font-mono font-medium truncate" title={span.name}>{span.name}</div>
      {span.status_code === "error" && span.status_message ? (
        <div className="text-[9px] text-destructive font-mono mt-0.5 truncate" title={span.status_message}>
          {span.status_message}
        </div>
      ) : (
        <div className="text-[9px] text-muted-foreground font-mono mt-0.5">
          {formatDuration(dur)}
        </div>
      )}
      <Handle type="source" position={Position.Bottom} className="!bg-muted-foreground !w-2 !h-2" />
    </div>
  );
}

const nodeTypes: NodeTypes = {
  spanNode: SpanNodeComponent,
};

export function FlowView({ spans, selectedSpanId, onSelectSpan }: FlowViewProps) {
  // Expensive dagre layout — only recompute when spans change
  const { baseNodes, layoutEdges } = useMemo(() => {
    const nodes: Node[] = spans.map((span) => ({
      id: span.span_id,
      type: "spanNode",
      position: { x: 0, y: 0 },
      data: { span, isSelected: false },
    }));

    const edges: Edge[] = spans
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
    (_: any, node: Node) => {
      onSelectSpan(node.id);
    },
    [onSelectSpan],
  );

  return (
    <div className="h-[500px] rounded-lg border bg-card/30 overflow-hidden">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onNodeClick={onNodeClick}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        minZoom={0.3}
        maxZoom={1.5}
      >
        <Background variant={BackgroundVariant.Dots} gap={16} size={1} className="!bg-transparent" />
        <Controls className="!bg-card !border-border !shadow-sm" />
      </ReactFlow>
    </div>
  );
}
