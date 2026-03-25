import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import QRCode from "qrcode";
import { Button } from "../components/ui/button";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { 
  Plus, 
  Trash2, 
  RefreshCw, 
  Bot as BotIcon, 
  MessageCircle, 
  Clock, 
  ExternalLink,
  Loader2,
  AlertCircle,
  MoreVertical,
  Activity,
  ArrowUpRight,
  ShieldCheck,
  Zap,
} from "lucide-react";
import { api } from "../lib/api";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useToast } from "@/hooks/use-toast";

const statusConfig: Record<string, { label: string; variant: any; color: string }> = {
  connected: { label: "运行中", variant: "default", color: "text-green-500" },
  disconnected: { label: "离线", variant: "outline", color: "text-muted-foreground" },
  error: { label: "故障", variant: "destructive", color: "text-destructive" },
  session_expired: { label: "授权过期", variant: "destructive", color: "text-destructive" },
};

export function BotsPage() {
  const [bots, setBots] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [binding, setBinding] = useState(false);
  const [qrUrl, setQrUrl] = useState("");
  const [bindStatus, setBindStatus] = useState("");
  const [enableAI, setEnableAI] = useState(true);
  const { toast } = useToast();

  async function load() {
    setLoading(true);
    try {
      const b = await api.listBots();
      setBots(b || []);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { load(); }, []);

  async function startBind() {
    setBinding(true);
    setBindStatus("正在初始化...");
    try {
      const { session_id, qr_url } = await api.bindStart();
      setQrUrl(qr_url);
      setBindStatus("请使用手机微信扫描上方二维码");
      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      const ws = new WebSocket(`${protocol}//${window.location.host}/api/bots/bind/status/${session_id}${enableAI ? "?enable_ai=true" : ""}`);
      ws.onmessage = (e) => {
        const data = JSON.parse(e.data);
        if (data.event === "status") {
          if (data.status === "scanned") setBindStatus("已扫码，请在手机上点击确认...");
          if (data.status === "refreshed") { setQrUrl(data.qr_url); setBindStatus("二维码已刷新"); }
          if (data.status === "connected") {
            toast({ title: "绑定成功", description: "微信账号已添加。" });
            ws.close();
            setBinding(false);
            load();
          }
        }
      };
      ws.onerror = () => { setBindStatus("同步中断，请重试"); ws.close(); };
      ws.onclose = () => {};
    } catch (err: any) {
      setBindStatus("初始化失败: " + err.message);
    }
  }

  return (
    <div className="space-y-8">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">账号管理</h2>
          <p className="text-muted-foreground">管理你的微信账号。</p>
        </div>
        <div className="flex items-center gap-3">
          <Button variant="outline" onClick={load} disabled={loading} className="h-10">
            <RefreshCw className={`h-4 w-4 mr-2 ${loading ? "animate-spin" : ""}`} /> 刷新
          </Button>
          <Dialog open={binding} onOpenChange={(o: boolean) => { setBinding(o); if(o) startBind(); else setQrUrl(""); }}>
            <DialogTrigger asChild>
              <Button className="h-10 px-6 shadow-lg shadow-primary/20">
                <Plus className="mr-2 h-4 w-4" /> 添加账号
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-md max-h-[90vh] overflow-y-auto">
              <DialogHeader className="text-left">
                <DialogTitle className="text-xl">扫码登录</DialogTitle>
                <DialogDescription>使用微信扫码登录。</DialogDescription>
              </DialogHeader>
              <div className="flex flex-col items-center justify-center gap-8 py-12">
                <div className="relative group">
                  <div className="absolute -inset-4 bg-primary/5 rounded-[2rem] blur-xl group-hover:bg-primary/10 transition-all" />
                  {qrUrl ? (
                    <div className="relative rounded-2xl border-4 border-background bg-white p-4 shadow-2xl">
                      <QrCanvas url={qrUrl} />
                    </div>
                  ) : (
                    <div className="relative flex h-[240px] w-[240px] items-center justify-center rounded-2xl border-2 border-dashed bg-muted/30">
                      <Loader2 className="h-10 w-10 animate-spin text-primary/40" />
                    </div>
                  )}
                </div>
                <div className="text-center space-y-2">
                  <p className="font-bold text-lg">{bindStatus}</p>
                  <p className="text-xs text-muted-foreground max-w-[240px] mx-auto leading-relaxed">
                    登录成功后即可使用。
                  </p>
                </div>
                <div className="w-full space-y-4 pt-4 border-t">
                   <div className="flex items-center justify-between p-3 rounded-xl border bg-muted/20">
                      <div className="flex items-center gap-2">
                        <Zap className="h-4 w-4 text-yellow-500" />
                        <span className="text-sm font-medium">启用 AI 自动回复</span>
                      </div>
                      <input type="checkbox" checked={enableAI} onChange={e => setEnableAI(e.target.checked)} className="h-4 w-4 accent-primary" />
                   </div>
                </div>
              </div>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {loading && bots.length === 0 ? (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map(i => <Card key={i} className="h-[220px] animate-pulse bg-muted/20" />)}
        </div>
      ) : (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {bots.map((bot) => (
            <BotInstanceCard key={bot.id} bot={bot} onRefresh={load} onRebind={() => setBinding(true)} />
          ))}
          
          {bots.length === 0 && (
            <div className="col-span-full py-24 border-2 border-dashed rounded-[2rem] flex flex-col items-center justify-center text-center bg-muted/5">
              <div className="h-20 w-20 rounded-3xl bg-background border shadow-sm flex items-center justify-center mb-6">
                <BotIcon className="h-10 w-10 text-primary/40" />
              </div>
              <h3 className="text-xl font-bold">还没有账号</h3>
              <p className="text-muted-foreground mt-2 max-w-sm">
                添加你的第一个微信账号。
              </p>
              <Button variant="outline" className="mt-8 h-11 px-8 rounded-full" onClick={() => { setBinding(true); startBind(); }}>
                添加账号
              </Button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function QrCanvas({ url }: { url: string }) {
  const ref = useRef<HTMLCanvasElement>(null);
  useEffect(() => {
    if (url && ref.current) QRCode.toCanvas(ref.current, url, { width: 224, margin: 0 });
  }, [url]);
  return <canvas ref={ref} className="block rounded-lg" />;
}

function BotInstanceCard({ bot, onRefresh, onRebind }: { bot: any; onRefresh: () => void; onRebind: () => void }) {
  const navigate = useNavigate();
  const { toast } = useToast();
  const status = statusConfig[bot.status] || statusConfig.disconnected;

  async function handleAction(action: string) {
    try {
      if (action === "delete") {
        if (!confirm("确定要删除此账号？相关转发规则将停止工作。")) return;
        await api.deleteBot(bot.id);
        toast({ title: "已删除账号" });
      } else if (action === "reconnect") {
        await api.reconnectBot(bot.id);
        toast({ title: "指令已发出", description: "正在尝试重新建立连接..." });
      }
      onRefresh();
    } catch (e: any) {
      toast({ variant: "destructive", title: "操作失败", description: e.message });
    }
  }

  return (
    <Card className="group relative overflow-hidden border-border/50 transition-all hover:shadow-2xl hover:border-primary/20 bg-card/50">
      <div className={`absolute top-0 left-0 w-1 h-full ${status.variant === "default" ? "bg-primary" : "bg-destructive"}`} />
      
      <CardHeader className="pb-4">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div className="h-10 w-10 rounded-xl bg-muted flex items-center justify-center group-hover:bg-primary/10 transition-colors">
              <BotIcon className="h-5 w-5 text-muted-foreground group-hover:text-primary transition-colors" />
            </div>
            <div className="space-y-0.5">
              <CardTitle className="text-lg font-bold">{bot.name}</CardTitle>
              <div className="flex items-center gap-2">
                <span className={`size-1.5 rounded-full animate-pulse ${status.color.replace('text', 'bg')}`} />
                <span className={`text-[10px] font-bold uppercase tracking-wider ${status.color}`}>{status.label}</span>
              </div>
            </div>
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-8 w-8 rounded-full opacity-0 group-hover:opacity-100 transition-opacity">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-40 rounded-xl">
              <DropdownMenuItem onClick={() => handleAction("reconnect")} className="gap-2">
                <RefreshCw className="h-3.5 w-3.5" /> 重新连接
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => handleAction("delete")} className="gap-2 text-destructive focus:bg-destructive/10 focus:text-destructive">
                <Trash2 className="h-3.5 w-3.5" /> 删除账号
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>

      <CardContent className="pb-6">
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-1">
            <p className="text-[10px] uppercase font-bold text-muted-foreground tracking-widest">消息数</p>
            <div className="flex items-center gap-1.5">
              <MessageCircle className="h-3.5 w-3.5 text-primary/60" />
              <span className="text-sm font-bold">{bot.msg_count || 0} <span className="text-[10px] font-normal opacity-60">MSGS</span></span>
            </div>
          </div>
          <div className="space-y-1">
            <p className="text-[10px] uppercase font-bold text-muted-foreground tracking-widest">自动续期</p>
            <div className="flex items-center gap-1.5">
              <Clock className="h-3.5 w-3.5 text-orange-500/60" />
              <span className="text-sm font-bold">{bot.reminder_hours ? `提前 ${24 - bot.reminder_hours}h` : "不提醒"}</span>
            </div>
          </div>
        </div>
        
        {bot.status === "session_expired" && (
          <div className="mt-4 flex items-center gap-2 rounded-xl bg-destructive/5 p-3 border border-destructive/10">
            <AlertCircle className="h-4 w-4 text-destructive shrink-0" />
            <p className="text-[11px] font-medium text-destructive leading-snug flex-1">
              登录已过期，请重新扫码。
            </p>
            <Button variant="destructive" size="sm" className="h-6 px-2 text-[10px] font-bold shrink-0" onClick={onRebind}>
              重新扫码
            </Button>
          </div>
        )}
      </CardContent>

      <CardFooter className="bg-muted/30 pt-4 flex gap-2">
        <Button 
          className="flex-1 h-9 rounded-lg gap-2 font-bold text-xs" 
          variant="secondary"
          onClick={() => navigate(`/dashboard/accounts/${bot.id}/chat`)}
        >
          查看详情 <ArrowUpRight className="h-3 w-3" />
        </Button>
      </CardFooter>
    </Card>
  );
}
