import { useNavigate } from "react-router-dom";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { useBotPush } from "../lib/ws";
import { useBotTraces } from "../hooks/use-bots";
import { RefreshCw, Activity, ChevronRight } from "lucide-react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { durationMs, StatusIcon } from "@/lib/trace-utils";

export function BotTracesTab({ botId }: { botId: string }) {
  const navigate = useNavigate();
  const { data: rootSpans = [], isLoading: loading, refetch } = useBotTraces(botId, 100);

  // Subscribe to push events — cache is auto-invalidated by PushProvider.
  useBotPush(botId);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Activity className="w-4 h-4 text-primary" />
          <h3 className="text-sm font-semibold">消息日志</h3>
        </div>
        <Button variant="outline" size="sm" onClick={() => refetch()} disabled={loading} className="h-8">
          <RefreshCw className={`w-3.5 h-3.5 mr-1.5 ${loading ? "animate-spin" : ""}`} /> 刷新
        </Button>
      </div>

      <div className="rounded-xl border bg-card/50 overflow-hidden shadow-sm">
        <Table>
          <TableHeader className="bg-muted/30">
            <TableRow>
              <TableHead className="w-[100px]">状态</TableHead>
              <TableHead>发送者</TableHead>
              <TableHead className="hidden md:table-cell">核心事件</TableHead>
              <TableHead className="text-right">Tokens</TableHead>
              <TableHead className="text-right">耗时</TableHead>
              <TableHead className="text-right">时间</TableHead>
              <TableHead className="w-8" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-full" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-10 ml-auto" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-12 ml-auto" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-20 ml-auto" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-4" /></TableCell>
                </TableRow>
              ))
            ) : rootSpans.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} className="h-32 text-center text-muted-foreground italic">
                  暂无记录
                </TableCell>
              </TableRow>
            ) : (
              rootSpans.map((root) => {
                const dur = durationMs(root);
                const sender = root.attributes?.["message.sender"] || "System";
                const content = root.attributes?.["message.content"] || root.name;

                return (
                  <TableRow
                    key={root.id}
                    className="cursor-pointer focus-visible:bg-muted/50"
                    tabIndex={0}
                    onClick={() => navigate(`/dashboard/accounts/${botId}/traces/${root.trace_id}`)}
                    onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); navigate(`/dashboard/accounts/${botId}/traces/${root.trace_id}`); } }}
                  >
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <StatusIcon code={root.status_code} size="w-3.5 h-3.5" />
                        <Badge variant="secondary" className="text-[9px] h-4 leading-none uppercase">
                          {root.attributes?.["message.type"] || "执行"}
                        </Badge>
                      </div>
                    </TableCell>
                    <TableCell className="font-mono text-xs max-w-[120px] truncate">
                      {sender}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground truncate max-w-[200px]">
                      {content}
                    </TableCell>
                    <TableCell className="text-right font-mono text-[10px] text-muted-foreground">
                      {root.attributes?.["ai.tokens.total"] || "\u2014"}
                    </TableCell>
                    <TableCell className="text-right font-mono text-[10px] text-muted-foreground">
                      {dur > 0 ? `${dur}ms` : "<1ms"}
                    </TableCell>
                    <TableCell className="text-right text-[10px] text-muted-foreground">
                      {new Date(root.start_time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}
                    </TableCell>
                    <TableCell className="w-8 px-2">
                      <ChevronRight className="h-4 w-4 text-muted-foreground" />
                    </TableCell>
                  </TableRow>
                );
              })
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
