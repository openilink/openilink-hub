import { useEffect, useRef, useState } from "react";
import { Send, Terminal, Ban } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";
import { api } from "@/lib/api";

type MessageItem = { type: string; text?: string; file_name?: string };
type Message = {
  id: number;
  bot_id?: string;
  direction: string;
  item_list: MessageItem[];
  media_status?: string;
  media_keys?: Record<string, string>;
  created_at: number;
  _sending?: boolean;
  _error?: string;
};

function MessageContent({ m }: { m: Message }) {
  const items = m.item_list || [];
  if (items.length === 0)
    return <span className="text-muted-foreground italic">[空消息]</span>;
  return (
    <div className="space-y-2">
      {items.map((item, i) => (
        <p key={i} className="leading-relaxed whitespace-pre-wrap">
          {item.text}
        </p>
      ))}
    </div>
  );
}

interface ChatPanelProps {
  botId: string;
  canSend?: boolean;
  sendDisabledReason?: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ChatPanel({ botId, canSend = true, sendDisabledReason, open, onOpenChange }: ChatPanelProps) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [sendError, setSendError] = useState("");
  const scrollRef = useRef<HTMLDivElement>(null);

  const fetchMessages = async () => {
    try {
      const res = await api.messages(botId, 30);
      setMessages((res.messages || []).reverse());
    } catch {}
  };

  useEffect(() => {
    if (!open) return;
    fetchMessages();
    const t = setInterval(fetchMessages, 5000);
    return () => clearInterval(t);
  }, [botId, open]);

  useEffect(() => {
    if (scrollRef.current)
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
  }, [messages]);

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || !canSend) return;
    setSendError("");
    const text = input;
    setInput("");
    try {
      await api.sendMessage(botId, { text });
      fetchMessages();
    } catch (err: any) {
      setSendError(err?.message || "发送失败");
      setInput(text); // restore input on error
    }
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="flex flex-col sm:max-w-md w-full p-0 gap-0"
      >
        <SheetHeader className="px-6 py-4 border-b bg-muted/20">
          <div className="flex items-center gap-2">
            <Terminal className="h-4 w-4 text-primary" />
            <SheetTitle className="text-xs font-bold uppercase tracking-widest">
              实时控制台
            </SheetTitle>
          </div>
          <SheetDescription className="sr-only">
            该账号的实时消息流
          </SheetDescription>
          <Badge
            variant="outline"
            className="bg-background text-[10px] font-bold w-fit"
          >
            实时推送
          </Badge>
        </SheetHeader>

        <div ref={scrollRef} className="flex-1 overflow-y-auto p-6 space-y-4">
          {messages.map((m) => (
            <div
              key={m.id}
              className={`flex ${m.direction === "inbound" ? "justify-start" : "justify-end"}`}
            >
              <div
                className={`max-w-[80%] px-4 py-3 rounded-2xl text-sm font-medium ${m.direction === "inbound" ? "bg-background border border-border/50 text-foreground rounded-bl-none shadow-sm" : "bg-primary text-primary-foreground rounded-br-none shadow-lg shadow-primary/10"}`}
              >
                <MessageContent m={m} />
                <p
                  className={`text-[9px] mt-1.5 font-bold uppercase opacity-40 ${m.direction === "inbound" ? "text-left" : "text-right"}`}
                >
                  {new Date(m.created_at * 1000).toLocaleTimeString()}
                </p>
              </div>
            </div>
          ))}
        </div>

        <div className="p-4 bg-muted/20 border-t space-y-2">
          {!canSend && (
            <div className="flex items-center gap-2 text-xs text-muted-foreground bg-muted/50 rounded-lg px-3 py-2">
              <Ban className="h-3.5 w-3.5 shrink-0" />
              <span>{sendDisabledReason || "当前无法发送消息"}</span>
            </div>
          )}
          {sendError && (
            <div className="text-xs text-destructive bg-destructive/10 rounded-lg px-3 py-2">
              {sendError}
            </div>
          )}
          <form className="flex gap-2" onSubmit={handleSend}>
            <Input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder={canSend ? "输入消息..." : "无法发送"}
              disabled={!canSend}
              className="h-11 rounded-xl bg-background border-none shadow-inner"
            />
            <Button
              type="submit"
              disabled={!canSend}
              className="h-11 rounded-xl px-6 gap-2 font-bold shadow-lg shadow-primary/20"
            >
              发送 <Send className="h-4 w-4" />
            </Button>
          </form>
        </div>
      </SheetContent>
    </Sheet>
  );
}
