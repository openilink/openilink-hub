import { useCallback, useEffect, useRef, useState, useMemo } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { ArrowLeft, Activity, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Skeleton } from "@/components/ui/skeleton";
import { api } from "@/lib/api";
import {
  TraceSpan,
  kindColors,
  durationMs,
  formatDuration,
  StatusIcon,
} from "@/lib/trace-utils";
import { TimelineView } from "./timeline-view";
import { FlowView } from "./flow-view";
import { SpanDetail } from "./span-detail";

export function TraceDetailPage() {
  const { id: botId, traceId } = useParams<{ id: string; traceId: string }>();
  const navigate = useNavigate();
  const [spans, setSpans] = useState<TraceSpan[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedSpanId, setSelectedSpanId] = useState<string | null>(null);
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
    () => (selectedSpanId ? spans.find((s) => s.span_id === selectedSpanId) ?? null : null),
    [spans, selectedSpanId],
  );
  const totalDuration = useMemo(() => {
    if (!rootSpan) return 0;
    return durationMs(rootSpan);
  }, [rootSpan]);

  if (!botId || !traceId) return null;

  return (
    <div className="flex flex-col gap-5">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Button
            variant="outline"
            size="sm"
            className="rounded-full px-4 font-bold text-xs"
            onClick={() => navigate(`/dashboard/accounts/${botId}/traces`)}
          >
            <ArrowLeft className="h-3.5 w-3.5 mr-1" />
            返回列表
          </Button>
          {rootSpan && (
            <div className="flex items-center gap-2">
              <StatusIcon code={rootSpan.status_code} size="w-4 h-4" />
              <Badge
                variant="outline"
                className={`text-[9px] h-4 px-1.5 leading-none text-white ${kindColors[rootSpan.kind] || "bg-gray-400"}`}
              >
                {rootSpan.kind}
              </Badge>
              <span className="text-sm font-mono font-medium">
                {rootSpan.attributes?.["message.sender"] || rootSpan.name}
              </span>
              <span className="text-xs text-muted-foreground font-mono">
                {formatDuration(totalDuration)}
              </span>
            </div>
          )}
        </div>
        <div className="flex items-center gap-2">
          <span className="text-[10px] font-mono text-muted-foreground hidden md:block">
            {traceId}
          </span>
          <Badge variant="secondary" className="text-[10px] h-5 font-mono">
            {spans.length} spans
          </Badge>
          <Button variant="outline" size="sm" onClick={load} disabled={loading} className="h-8">
            <RefreshCw className={`w-3.5 h-3.5 mr-1.5 ${loading ? "animate-spin" : ""}`} />
            刷新
          </Button>
        </div>
      </div>

      {/* Content */}
      {loading ? (
        <div className="space-y-3">
          <Skeleton className="h-10 w-64" />
          <Skeleton className="h-[400px] w-full" />
        </div>
      ) : spans.length === 0 ? (
        <div className="flex flex-col items-center justify-center h-64 text-muted-foreground">
          <Activity className="w-8 h-8 mb-2 opacity-50" />
          <p className="text-sm italic">未找到追踪数据</p>
        </div>
      ) : (
        <Tabs defaultValue="timeline" className="w-full">
          <TabsList>
            <TabsTrigger value="timeline" className="text-xs">
              Timeline
            </TabsTrigger>
            <TabsTrigger value="flow" className="text-xs">
              Flow
            </TabsTrigger>
          </TabsList>

          <TabsContent value="timeline">
            <div className="rounded-xl border bg-card/50 p-4 shadow-sm overflow-x-auto">
              <TimelineView
                spans={spans}
                selectedSpanId={selectedSpanId}
                onSelectSpan={setSelectedSpanId}
              />
            </div>
          </TabsContent>

          <TabsContent value="flow">
            <FlowView
              spans={spans}
              selectedSpanId={selectedSpanId}
              onSelectSpan={setSelectedSpanId}
            />
          </TabsContent>
        </Tabs>
      )}

      {/* Span Detail Drawer */}
      <SpanDetail
        span={selectedSpan}
        open={!!selectedSpan}
        onClose={() => setSelectedSpanId(null)}
      />
    </div>
  );
}
