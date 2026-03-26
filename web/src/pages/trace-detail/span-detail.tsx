import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";
import {
  TraceSpan,
  kindColors,
  durationMs,
  formatDuration,
  StatusIcon,
} from "@/lib/trace-utils";
import { Clock, Tag, Zap } from "lucide-react";

interface SpanDetailProps {
  span: TraceSpan | null;
  open: boolean;
  onClose: () => void;
}

function Section({ title, icon, children }: { title: string; icon: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
        {icon}
        {title}
      </div>
      {children}
    </div>
  );
}

export function SpanDetail({ span, open, onClose }: SpanDetailProps) {
  if (!span) return null;

  const dur = durationMs(span);
  const attrs = span.attributes ? Object.entries(span.attributes) : [];
  const events = span.events || [];

  return (
    <Sheet open={open} onOpenChange={(v) => !v && onClose()}>
      <SheetContent side="right" className="w-[400px] sm:max-w-[400px] p-0">
        <SheetHeader className="p-4 pb-3 border-b">
          <div className="flex items-center gap-2 mb-1">
            <StatusIcon code={span.status_code} size="w-4 h-4" />
            <Badge
              variant="outline"
              className={`text-[9px] h-4 px-1.5 leading-none text-white ${kindColors[span.kind] || "bg-gray-400"}`}
            >
              {span.kind}
            </Badge>
            <span className="text-[10px] font-mono text-muted-foreground">{formatDuration(dur)}</span>
          </div>
          <SheetTitle className="font-mono text-sm truncate">{span.name}</SheetTitle>
          {span.status_message && (
            <SheetDescription className="text-destructive text-xs">{span.status_message}</SheetDescription>
          )}
        </SheetHeader>

        <ScrollArea className="h-[calc(100vh-120px)] px-4 py-3">
          <div className="space-y-5">
            {/* Timing */}
            <Section title="Timing" icon={<Clock className="w-3 h-3" />}>
              <div className="grid grid-cols-2 gap-2 text-xs">
                <div className="space-y-0.5">
                  <div className="text-muted-foreground">Start</div>
                  <div className="font-mono">
                    {new Date(span.start_time).toLocaleTimeString([], {
                      hour: "2-digit",
                      minute: "2-digit",
                      second: "2-digit",
                    } as Intl.DateTimeFormatOptions)}
                  </div>
                </div>
                <div className="space-y-0.5">
                  <div className="text-muted-foreground">End</div>
                  <div className="font-mono">
                    {span.end_time
                      ? new Date(span.end_time).toLocaleTimeString([], {
                          hour: "2-digit",
                          minute: "2-digit",
                          second: "2-digit",
                        } as Intl.DateTimeFormatOptions)
                      : "—"}
                  </div>
                </div>
                <div className="space-y-0.5">
                  <div className="text-muted-foreground">Duration</div>
                  <div className="font-mono font-medium">{formatDuration(dur)}</div>
                </div>
                <div className="space-y-0.5">
                  <div className="text-muted-foreground">Span ID</div>
                  <div className="font-mono text-[10px] truncate">{span.span_id}</div>
                </div>
              </div>
            </Section>

            {/* Attributes */}
            {attrs.length > 0 && (
              <Section title="Attributes" icon={<Tag className="w-3 h-3" />}>
                <div className="rounded-md border bg-muted/30 overflow-hidden">
                  {attrs.map(([key, value], i) => (
                    <div
                      key={key}
                      className={`flex gap-3 px-3 py-1.5 text-xs ${
                        i < attrs.length - 1 ? "border-b border-border/50" : ""
                      }`}
                    >
                      <span className="text-blue-500 font-semibold shrink-0 min-w-[100px]">{key}</span>
                      <span className="text-foreground/80 break-all">{String(value)}</span>
                    </div>
                  ))}
                </div>
              </Section>
            )}

            {/* Events */}
            {events.length > 0 && (
              <Section title="Events" icon={<Zap className="w-3 h-3" />}>
                <div className="space-y-2">
                  {events.map((evt, i) => (
                    <div key={i} className="rounded-md border bg-muted/30 px-3 py-2">
                      <div className="flex items-center justify-between mb-1">
                        <span className="text-xs font-semibold">{evt.name}</span>
                        <span className="text-[9px] font-mono text-muted-foreground">
                          {new Date(evt.timestamp).toLocaleTimeString([], {
                            hour: "2-digit",
                            minute: "2-digit",
                            second: "2-digit",
                          })}
                        </span>
                      </div>
                      {evt.attributes && Object.keys(evt.attributes).length > 0 && (
                        <div className="space-y-0.5 mt-1">
                          {Object.entries(evt.attributes).map(([k, v]) => (
                            <div key={k} className="flex gap-2 text-[10px]">
                              <span className="text-blue-500 font-bold shrink-0">{k}:</span>
                              <span className="text-foreground/70 break-all">{String(v)}</span>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </Section>
            )}
          </div>
        </ScrollArea>
      </SheetContent>
    </Sheet>
  );
}
