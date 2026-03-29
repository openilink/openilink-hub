import { useEffect, useRef, useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import QRCode from "qrcode";
import { Button } from "../components/ui/button";
import { Card, CardContent } from "../components/ui/card";
import {
  Plus,
  Trash2,
  RefreshCw,
  Bot as BotIcon,
  MessageCircle,
  Clock,
  Loader2,
  AlertCircle,
  MoreVertical,
  ArrowUpRight,
  Cpu,
  Wifi,
  WifiOff,
} from "lucide-react";
import { api, botDisplayName } from "../lib/api";
import { useBots, useDeleteBot, useReconnectBot } from "@/hooks/use-bots";
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
import { useConfirm } from "@/components/ui/confirm-dialog";

const statusConfig: Record<
  string,
  { label: string; variant: "default" | "destructive" | "outline"; dot: string }
> = {
  connected: { label: "运行中", variant: "default", dot: "bg-green-500" },
  disconnected: { label: "离线", variant: "outline", dot: "bg-muted-foreground" },
  error: { label: "故障", variant: "destructive", dot: "bg-destructive" },
  session_expired: { label: "授权过期", variant: "destructive", dot: "bg-destructive" },
};

export function BotsPage() {
  const { data: bots = [], isLoading, isFetching, refetch } = useBots();
  const loading = isLoading || isFetching;
  const [binding, setBinding] = useState(false);
  const [qrUrl, setQrUrl] = useState("");
  const [bindStatus, setBindStatus] = useState("");
  const navigate = useNavigate();

  async function startBind() {
    setBinding(true);
    setBindStatus("正在初始化...");
    try {
      const { session_id, qr_url } = await api.bindStart();
      setQrUrl(qr_url);
      setBindStatus("请使用手机微信扫描上方二维码");
      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      const ws = new WebSocket(
        `${protocol}//${window.location.host}/api/bots/bind/status/${session_id}`,
      );
      ws.onmessage = (e) => {
        const data = JSON.parse(e.data);
        if (data.event === "status") {
          if (data.status === "scanned") setBindStatus("已扫码，请在手机上点击确认...");
          if (data.status === "refreshed") {
            setQrUrl(data.qr_url);
            setBindStatus("二维码已刷新");
          }
          if (data.status === "connected") {
            ws.close();
            setBinding(false);
            navigate(
              data.is_new && data.bot_id
                ? `/dashboard/onboarding?bot_id=${data.bot_id}`
                : "/dashboard/accounts",
            );
          }
        }
      };
      ws.onerror = () => {
        setBindStatus("同步中断，请重试");
        ws.close();
      };
      ws.onclose = () => {};
    } catch (err: any) {
      setBindStatus("初始化失败: " + err.message);
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">账号管理</h1>
          <p className="text-sm text-muted-foreground mt-0.5">管理你的微信账号。</p>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <Button variant="outline" onClick={() => refetch()} disabled={loading}>
            <RefreshCw className={`h-4 w-4 mr-2 ${loading ? "animate-spin" : ""}`} /> 刷新
          </Button>
          <Dialog
            open={binding}
            onOpenChange={(o: boolean) => {
              setBinding(o);
              if (o) startBind();
              else setQrUrl("");
            }}
          >
            <DialogTrigger asChild>
              <Button className="px-6 shadow-lg shadow-primary/20">
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
              </div>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {loading && bots.length === 0 ? (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Card key={i} className="h-[220px] animate-pulse bg-muted/20" />
          ))}
        </div>
      ) : (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {bots.map((bot) => (
            <BotInstanceCard
              key={bot.id}
              bot={bot}
              onRebind={() => setBinding(true)}
            />
          ))}

          {bots.length === 0 ? (
            <div className="col-span-full py-24 border-2 border-dashed rounded-[2rem] flex flex-col items-center justify-center text-center bg-muted/5">
              <div className="h-20 w-20 rounded-3xl bg-background border shadow-sm flex items-center justify-center mb-6">
                <BotIcon className="h-10 w-10 text-primary/40" />
              </div>
              <h3 className="text-xl font-bold">还没有账号</h3>
              <p className="text-muted-foreground mt-2 max-w-sm">添加你的第一个微信账号。</p>
              <Button
                variant="outline"
                className="mt-8 h-11 px-8 rounded-full"
                onClick={() => {
                  setBinding(true);
                  startBind();
                }}
              >
                添加账号
              </Button>
            </div>
          ) : null}
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

function BotInstanceCard({
  bot,
  onRebind,
}: {
  bot: any;
  onRebind: () => void;
}) {
  const { toast } = useToast();
  const { confirm, ConfirmDialog } = useConfirm();
  const deleteMutation = useDeleteBot();
  const reconnectMutation = useReconnectBot();
  const status = statusConfig[bot.status] || statusConfig.disconnected;
  const isOnline = bot.status === "connected";

  async function handleAction(action: string) {
    if (action === "delete") {
      const ok = await confirm({
        title: "删除确认",
        description: "确定要删除此账号？相关转发规则将停止工作。",
        confirmText: "删除",
        variant: "destructive",
      });
      if (!ok) return;
      deleteMutation.mutate(bot.id, {
        onSuccess: () => toast({ title: "已删除账号" }),
        onError: (e) => toast({ variant: "destructive", title: "操作失败", description: e.message }),
      });
    } else if (action === "reconnect") {
      reconnectMutation.mutate(bot.id, {
        onSuccess: () => toast({ title: "指令已发出", description: "正在尝试重新建立连接..." }),
        onError: (e) => toast({ variant: "destructive", title: "操作失败", description: e.message }),
      });
    }
  }

  return (
    <Card className="group flex flex-col border-border/50 hover:border-primary/20 hover:shadow-lg transition-all duration-200">
      {ConfirmDialog}
      <CardContent className="p-5 flex-1 space-y-4">
        {/* Header row */}
        <div className="flex items-start justify-between gap-2">
          <div className="flex items-center gap-3 min-w-0">
            <div
              className={`h-10 w-10 rounded-xl flex items-center justify-center shrink-0 transition-colors ${
                isOnline ? "bg-primary/10 text-primary" : "bg-muted text-muted-foreground"
              }`}
            >
              {isOnline ? <Wifi className="h-5 w-5" /> : <WifiOff className="h-5 w-5" />}
            </div>
            <div className="min-w-0">
              <p className="font-semibold leading-tight truncate">{botDisplayName(bot)}</p>
              <div className="flex items-center gap-1.5 mt-0.5">
                <span
                  className={`size-1.5 rounded-full shrink-0 ${status.dot} ${isOnline ? "animate-pulse" : ""}`}
                />
                <span className="text-xs text-muted-foreground">{status.label}</span>
              </div>
            </div>
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="icon-sm"
                className="shrink-0 opacity-0 group-hover:opacity-100 transition-opacity mt-0.5"
              >
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-40">
              {bot.status !== "session_expired" ? (
                <DropdownMenuItem onClick={() => handleAction("reconnect")} className="gap-2">
                  <RefreshCw className="h-3.5 w-3.5" /> 重新连接
                </DropdownMenuItem>
              ) : null}
              <DropdownMenuItem
                onClick={() => handleAction("delete")}
                className="gap-2 text-destructive focus:bg-destructive/10 focus:text-destructive"
              >
                <Trash2 className="h-3.5 w-3.5" /> 删除账号
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        {/* Stats row */}
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <MessageCircle className="h-3.5 w-3.5" />
            <span>{bot.msg_count ?? 0} 消息</span>
          </div>
          {bot.reminder_hours ? (
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <Clock className="h-3.5 w-3.5 text-orange-500" />
              <span>提前 {24 - bot.reminder_hours}h 提醒</span>
            </div>
          ) : null}
        </div>

        {/* Session expired warning */}
        {bot.status === "session_expired" ? (
          <div className="rounded-lg bg-destructive/5 border border-destructive/10 p-3">
            <div className="flex items-start gap-2">
              <AlertCircle className="h-3.5 w-3.5 mt-0.5 text-destructive shrink-0" />
              <div className="space-y-1">
                <p className="text-xs text-destructive leading-snug">
                  会话已过期，请在微信中给该账号发一条消息以恢复连接。
                </p>
                <Button
                  variant="link"
                  size="xs"
                  className="h-auto p-0 text-destructive text-xs"
                  onClick={onRebind}
                >
                  或重新扫码绑定
                </Button>
              </div>
            </div>
          </div>
        ) : null}
      </CardContent>

      {/* Slim footer: meta info + navigate link */}
      <div className="mx-5 mb-5 pt-3 border-t border-border/40 flex items-center justify-between">
        <div className="flex items-center gap-1.5 text-xs text-muted-foreground/60">
          <Cpu className="h-3 w-3 shrink-0" />
          <span className="capitalize">{bot.provider || "未知"}</span>
          <span className="mx-0.5 opacity-40">·</span>
          <span className="font-mono">{bot.id.slice(0, 8)}</span>
        </div>
        <Link
          to={`/dashboard/accounts/${bot.id}`}
          className="flex items-center gap-1 text-xs text-muted-foreground hover:text-primary transition-colors group/link"
        >
          <span>查看详情</span>
          <ArrowUpRight className="h-3 w-3 group-hover/link:translate-x-0.5 group-hover/link:-translate-y-0.5 transition-transform duration-200" />
        </Link>
      </div>
    </Card>
  );
}
