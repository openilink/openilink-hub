import { useEffect, useState } from "react";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { api } from "../lib/api";
import { RefreshCw } from "lucide-react";

const spanColors: Record<string, string> = {
  receive: "bg-blue-500",
  store: "bg-slate-400",
  media: "bg-slate-400",
  match_channel: "bg-purple-500",
  match_handle: "bg-indigo-500",
  match_command: "bg-indigo-500",
  match_event: "bg-indigo-500",
  deliver_app: "bg-green-500",
  deliver_webhook: "bg-yellow-500",
  reply: "bg-pink-500",
  ai: "bg-orange-500",
};

const statusColors: Record<string, string> = {
  ok: "text-primary",
  error: "text-destructive",
  skip: "text-muted-foreground",
  timeout: "text-yellow-600",
  pending: "text-muted-foreground",
};

export function BotTracesTab({ botId }: { botId: string }) {
  const [traces, setTraces] = useState<any[]>([]);
  const [expanded, setExpanded] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);

  async function load() {
    setLoading(true);
    try { setTraces((await api.listTraces(botId, 100)) || []); } catch {}
    setLoading(false);
  }

  useEffect(() => { load(); }, [botId]);

  function formatTime(ts: number) {
    if (ts > 1e12) return new Date(ts).toLocaleTimeString(); // millis
    return new Date(ts * 1000).toLocaleTimeString(); // seconds
  }

  function formatDate(ts: number) {
    return new Date(ts * 1000).toLocaleString();
  }

  return (
    <div className="space-y-3 mt-4">
      <div className="flex items-center justify-between">
        <p className="text-sm font-medium">消息链路追踪</p>
        <Button variant="outline" size="sm" onClick={load} disabled={loading}>
          <RefreshCw className={`w-3.5 h-3.5 mr-1 ${loading ? "animate-spin" : ""}`} /> 刷新
        </Button>
      </div>

      {traces.length === 0 && !loading && (
        <p className="text-center text-sm text-muted-foreground py-8">暂无追踪记录</p>
      )}

      <div className="space-y-1">
        {traces.map((trace) => {
          const isOpen = expanded === trace.id;
          const spans = (trace.spans || []) as any[];
          const hasError = spans.some((s: any) => s.status === "error" || s.status === "timeout");
          const appSpans = spans.filter((s: any) => s.type === "deliver_app" || s.type === "deliver_webhook");
          const totalDuration = appSpans.reduce((sum: number, s: any) => sum + (s.duration_ms || 0), 0);

          return (
            <div key={trace.id} className="rounded-lg border bg-card overflow-hidden">
              <div
                className="flex items-center gap-3 p-2.5 cursor-pointer hover:bg-secondary/50"
                onClick={() => setExpanded(isOpen ? null : trace.id)}
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <Badge variant={hasError ? "destructive" : "default"} className="text-[10px] shrink-0">
                      {trace.msg_type || "text"}
                    </Badge>
                    <span className="text-xs font-mono truncate">{trace.sender}</span>
                    <span className="text-xs text-muted-foreground truncate">{trace.content}</span>
                  </div>
                </div>
                <div className="flex items-center gap-2 shrink-0 text-[10px] text-muted-foreground">
                  <span>{spans.length} steps</span>
                  {totalDuration > 0 && <span>{totalDuration}ms</span>}
                  <span>{formatDate(trace.created_at)}</span>
                </div>
              </div>

              {isOpen && (
                <div className="border-t p-3 space-y-1 bg-background">
                  <div className="text-[10px] text-muted-foreground mb-2 font-mono">
                    trace: {trace.trace_id} · message #{trace.message_id}
                  </div>
                  {spans.map((span: any, i: number) => (
                    <div key={i} className="flex items-start gap-2 text-xs">
                      <div className="flex items-center gap-1.5 shrink-0 w-28">
                        <div className={`w-2 h-2 rounded-full shrink-0 ${spanColors[span.type] || "bg-gray-400"}`} />
                        <span className="text-muted-foreground font-mono text-[10px]">{formatTime(span.timestamp)}</span>
                      </div>
                      <Badge variant="outline" className="text-[10px] font-mono shrink-0">{span.type}</Badge>
                      <span className={`${statusColors[span.status] || ""} shrink-0`}>
                        {span.status === "ok" ? "✓" : span.status === "error" ? "✕" : span.status === "skip" ? "⊘" : "…"}
                      </span>
                      <span className="truncate">{span.name}</span>
                      {span.duration_ms > 0 && (
                        <span className="text-muted-foreground shrink-0">{span.duration_ms}ms</span>
                      )}
                    </div>
                  ))}
                  {/* Detail rows */}
                  {spans.filter((s: any) => s.detail).map((span: any, i: number) => (
                    <div key={`d-${i}`} className="ml-[7.5rem] text-[10px] text-muted-foreground font-mono break-all">
                      {span.type}: {span.detail}
                    </div>
                  ))}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
