import { useEffect, useState, useCallback } from "react";
import { useParams, useNavigate, useLocation } from "react-router-dom";
import {
  ArrowUpRight,
  Cable,
  Plus,
  Trash2,
  Bot as BotIcon,
  Zap,
  Cpu,
  Unplug,
  MessageSquare,
  Activity,
} from "lucide-react";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { api } from "../lib/api";
import {
  Tabs,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import {
  Card,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { useToast } from "@/hooks/use-toast";
import { Skeleton } from "@/components/ui/skeleton";

import { BotAppsTab } from "./bot-apps-tab";
import { ChatPanel } from "./chat-panel";

export function BotDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const { toast } = useToast();
  const [bot, setBot] = useState<any>(null);
  const [channels, setChannels] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [chatOpen, setChatOpen] = useState(false);

  // Tabs synced with URL — only "channels" and "apps" are valid
  const rawTab = location.pathname.split("/").pop() || "channels";
  const activeTab = rawTab === "channels" || rawTab === "apps" ? rawTab : "channels";

  const load = useCallback(async () => {
    try {
      const bots = await api.listBots();
      const target = (bots || []).find((b: any) => b.id === id);
      if (!target) throw new Error("Instance not found");
      setBot(target);
      const chs = await api.listChannels(id!);
      setChannels(chs || []);
    } catch (e: any) {
      toast({ variant: "destructive", title: "加载失败", description: e.message });
    } finally {
      setLoading(false);
    }
  }, [id, toast]);

  useEffect(() => {
    load();
    // Poll bot status every 10s to keep can_send/status fresh
    const t = setInterval(async () => {
      try {
        const bots = await api.listBots();
        const target = (bots || []).find((b: any) => b.id === id);
        if (target) setBot(target);
      } catch {}
    }, 10000);
    return () => clearInterval(t);
  }, [load]);

  const handleAutoRenewalChange = async (checked: boolean) => {
    const hours = checked ? 23 : 0;
    try {
      await api.updateBot(bot.id, { reminder_hours: hours });
      toast({ title: "已保存" });
      load();
    } catch (e: any) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
  };

  if (loading) return <div className="space-y-6"><Skeleton className="h-20 w-full rounded-3xl" /><Skeleton className="h-96 w-full rounded-3xl" /></div>;
  if (!bot) return <div className="py-20 text-center space-y-4"><Unplug className="h-12 w-12 mx-auto opacity-20" /><p className="font-bold">未找到账号</p><Button variant="link" onClick={() => navigate("/dashboard/accounts")}>返回列表</Button></div>;

  return (
    <div className="flex flex-col gap-8 h-full">
      {/* Entity Banner */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-6">
        <div className="flex items-center gap-5">
          <div className="h-16 w-16 rounded-[1.5rem] bg-primary/10 flex items-center justify-center text-primary shadow-inner border border-primary/20">
            <BotIcon className="h-8 w-8" />
          </div>
          <div className="space-y-1">
            <div className="flex items-center gap-3">
              <h1 className="text-3xl font-black tracking-tighter">{bot.name}</h1>
              <Badge variant={bot.status === "connected" ? "default" : "destructive"} className="rounded-full px-3 py-0.5 text-[10px] font-black uppercase tracking-widest">
                {bot.status}
              </Badge>
              {bot.can_send === false && (
                <Badge variant="outline" className="rounded-full px-3 py-0.5 text-[10px] font-bold text-orange-600 border-orange-300">
                  不可发送
                </Badge>
              )}
            </div>
            <div className="flex items-center gap-2 text-xs font-bold text-muted-foreground uppercase tracking-widest">
               <Cpu className="h-3 w-3" /> {bot.provider}
               <Separator orientation="vertical" className="h-3 mx-1" />
               <span className="font-mono">{bot.id.slice(0, 12)}...</span>
            </div>
            {bot.send_disabled_reason && (
              <p className="text-xs text-orange-600 mt-1">{bot.send_disabled_reason}</p>
            )}
          </div>
        </div>
        <div className="flex items-center gap-3">
           {/* Inline auto-renewal checkbox */}
           <label className="flex items-center gap-2 text-xs font-bold text-muted-foreground cursor-pointer select-none">
             <input
               type="checkbox"
               checked={(bot.reminder_hours || 0) > 0}
               onChange={(e) => handleAutoRenewalChange(e.target.checked)}
               className="h-4 w-4 accent-primary"
             />
             自动续期
           </label>
           <Separator orientation="vertical" className="h-5" />
           <Button variant="outline" size="sm" className="rounded-full px-4 font-bold text-xs gap-1.5" onClick={() => setChatOpen(true)}>
             <MessageSquare className="h-3.5 w-3.5" />
             消息控制台
           </Button>
           <Button variant="outline" size="sm" className="rounded-full px-4 font-bold text-xs gap-1.5" onClick={() => navigate(`/dashboard/accounts/${id}/traces`)}>
             <Activity className="h-3.5 w-3.5" />
             消息追踪
           </Button>
           <Separator orientation="vertical" className="h-5" />
           <Button variant="outline" size="sm" className="rounded-full px-4 font-bold text-xs" onClick={() => navigate("/dashboard/accounts")}>
             返回列表
           </Button>
           <Button variant="destructive" size="sm" className="rounded-full h-9 w-9 p-0 shadow-lg shadow-destructive/10">
             <Trash2 className="h-4 w-4" />
           </Button>
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={(v: string) => navigate(`/dashboard/accounts/${id}/${v}`)} className="flex-1 flex flex-col space-y-6">
        <TabsList className="bg-muted/50 p-1 w-fit rounded-xl border border-border/50">
          <TabsTrigger value="channels" className="gap-2 px-6 rounded-lg font-bold text-xs uppercase tracking-widest"><Cable className="h-3.5 w-3.5" /> 转发规则</TabsTrigger>
          <TabsTrigger value="apps" className="gap-2 px-6 rounded-lg font-bold text-xs uppercase tracking-widest"><Zap className="h-3.5 w-3.5" /> 应用</TabsTrigger>
        </TabsList>

        <div className="flex-1">
          {activeTab === "channels" && <ChannelsTab botId={id!} channels={channels} onRefresh={load} />}
          {activeTab === "apps" && <BotAppsTab botId={id!} />}
        </div>
      </Tabs>

      <ChatPanel botId={id!} canSend={bot.can_send} sendDisabledReason={bot.send_disabled_reason} open={chatOpen} onOpenChange={setChatOpen} />
    </div>
  );
}

// --- ChannelsTab ---

function ChannelsTab({ botId, channels, onRefresh }: { botId: string; channels: any[]; onRefresh: () => void }) {
  const navigate = useNavigate();
  return (
    <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
       {channels.map((ch) => (
         <Card key={ch.id} className="group relative border-border/50 bg-card/50 rounded-3xl transition-all hover:shadow-xl hover:border-primary/20 cursor-pointer" onClick={() => navigate(`/dashboard/accounts/${botId}/channel/${ch.id}`)}>
            <CardHeader>
               <div className="flex justify-between items-start">
                  <div className="h-10 w-10 rounded-xl bg-muted flex items-center justify-center group-hover:bg-primary/10 transition-colors">
                    <Cable className="h-5 w-5 text-muted-foreground group-hover:text-primary transition-colors" />
                  </div>
                  <Badge variant={ch.enabled ? "default" : "secondary"} className="h-5 rounded-full text-[9px] font-black uppercase">{ch.enabled ? "运行中" : "已暂停"}</Badge>
               </div>
               <CardTitle className="text-lg font-bold mt-4">{ch.name}</CardTitle>
               <CardDescription className="font-mono text-[10px] uppercase">@{ch.handle || "默认"}</CardDescription>
            </CardHeader>
            <CardFooter className="bg-muted/30 pt-3 flex justify-between items-center px-6">
               <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest"></span>
               <ArrowUpRight className="h-4 w-4 text-muted-foreground group-hover:text-primary transition-all" />
            </CardFooter>
         </Card>
       ))}
       <Button variant="outline" className="h-auto border-dashed border-2 rounded-3xl py-10 flex-col gap-3 hover:bg-primary/5 hover:border-primary/20 transition-all" onClick={() => {/* Handle Create */}}>
          <div className="h-10 w-10 rounded-full bg-background border flex items-center justify-center shadow-sm"><Plus className="h-5 w-5" /></div>
          <span className="font-bold text-xs uppercase tracking-[0.2em] text-muted-foreground">添加转发规则</span>
       </Button>
    </div>
  );
}
