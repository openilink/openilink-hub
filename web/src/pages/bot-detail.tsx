import { useEffect, useState, useCallback, useRef } from "react";
import { useParams, useNavigate, useLocation } from "react-router-dom";
import {
  ArrowUpRight,
  Cable,
  Trash2,
  Bot as BotIcon,
  Cpu,
  Unplug,
  MessageSquare,
  Activity,
  AlertTriangle,
  Blocks,
  Download,
  RefreshCw,
} from "lucide-react";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { api } from "../lib/api";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { useToast } from "@/hooks/use-toast";
import { Skeleton } from "@/components/ui/skeleton";
import { AppIcon } from "../components/app-icon";

// ==================== Page ====================

export function BotDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const { toast } = useToast();
  const showChannels = location.pathname.endsWith("/channels");
  const [bot, setBot] = useState<any>(null);
  const [channels, setChannels] = useState<any[]>([]);
  const [installations, setInstallations] = useState<any[]>([]);
  const [builtinApps, setBuiltinApps] = useState<any[]>([]);
  const [marketplaceApps, setMarketplaceApps] = useState<any[]>([]);
  const [marketplaceLoading, setMarketplaceLoading] = useState(true);
  const [syncing, setSyncing] = useState(false);
  const [loading, setLoading] = useState(true);
  const marketplaceRef = useRef<HTMLDivElement>(null);

  const loadBot = useCallback(async () => {
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

  const loadInstallations = useCallback(async () => {
    try {
      setInstallations((await api.listBotApps(id!)) || []);
    } catch {}
  }, [id]);

  const loadMarketplace = useCallback(async () => {
    setMarketplaceLoading(true);
    try {
      const [builtin, marketplace] = await Promise.all([
        api.getBuiltinApps().catch(() => []),
        api.getMarketplaceApps().catch(() => []),
      ]);
      setBuiltinApps(builtin || []);
      setMarketplaceApps(marketplace || []);
    } finally {
      setMarketplaceLoading(false);
    }
  }, []);

  useEffect(() => {
    loadBot();
    loadInstallations();
    loadMarketplace();
    const t = setInterval(async () => {
      try {
        const bots = await api.listBots();
        const target = (bots || []).find((b: any) => b.id === id);
        if (target) setBot(target);
      } catch {}
    }, 10000);
    return () => clearInterval(t);
  }, [loadBot, loadInstallations, loadMarketplace]);

  const handleAutoRenewalChange = async (hours: number) => {
    try {
      await api.updateBot(bot.id, { reminder_hours: hours });
      toast({ title: "已保存" });
      loadBot();
    } catch (e: any) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
  };

  const handleInstallApp = async (app: any) => {
    setSyncing(true);
    try {
      if (app.local_id) {
        navigate(`/dashboard/accounts/${id}/install/${app.local_id}`);
      } else {
        const synced = await api.syncMarketplaceApp(app.slug);
        navigate(`/dashboard/accounts/${id}/install/${synced.id}`);
      }
    } catch (e: any) {
      toast({ variant: "destructive", title: "同步失败", description: e.message });
    } finally {
      setSyncing(false);
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
        <div className="flex items-center gap-3 flex-wrap">
           <Button variant="outline" size="sm" className="rounded-full px-4 font-bold text-xs gap-1.5" onClick={() => navigate(`/dashboard/accounts/${id}/channels`)}>
             <Cable className="h-3.5 w-3.5" />
             转发规则
           </Button>
           <Separator orientation="vertical" className="h-5" />
           <Button variant="outline" size="sm" className="rounded-full px-4 font-bold text-xs gap-1.5" onClick={() => navigate(`/dashboard/accounts/${id}/console`)}>
             <MessageSquare className="h-3.5 w-3.5" />
             消息控制台
           </Button>
           <Button variant="outline" size="sm" className="rounded-full px-4 font-bold text-xs gap-1.5" onClick={() => navigate(`/dashboard/accounts/${id}/traces`)}>
             <Activity className="h-3.5 w-3.5" />
             消息追踪
           </Button>
           <Separator orientation="vertical" className="h-5" />
           <label className="flex items-center gap-2 text-xs font-bold text-muted-foreground select-none">
             自动续期
             <select
               value={bot.reminder_hours || 0}
               onChange={(e) => handleAutoRenewalChange(Number(e.target.value))}
               className="h-7 rounded-md border border-input bg-background px-2 text-xs font-bold cursor-pointer focus:outline-none focus:ring-1 focus:ring-ring"
             >
               <option value={0}>不提醒</option>
               <option value={23}>提前 1 小时</option>
               <option value={22}>提前 2 小时</option>
             </select>
           </label>
           <Separator orientation="vertical" className="h-5" />
           <Button variant="outline" size="sm" className="rounded-full px-4 font-bold text-xs" onClick={() => navigate("/dashboard/accounts")}>
             返回列表
           </Button>
           <Button variant="destructive" size="sm" className="rounded-full h-9 w-9 p-0 shadow-lg shadow-destructive/10">
             <Trash2 className="h-4 w-4" />
           </Button>
        </div>
      </div>

      {/* Migration Banner */}
      {channels.length > 0 && (
        <div className="flex items-center gap-3 p-4 bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-800 rounded-2xl">
          <AlertTriangle className="h-5 w-5 text-amber-600 shrink-0" />
          <div className="flex-1">
            <p className="text-sm font-medium text-amber-800 dark:text-amber-200">
              你有 {channels.length} 个转发规则尚未迁移为 App
            </p>
            <p className="text-xs text-amber-600 dark:text-amber-400 mt-0.5">
              推荐创建「WebSocket App」或「Webhook App」替代，获得更灵活的权限控制和事件订阅。
            </p>
          </div>
          <Button variant="outline" size="sm" className="shrink-0 border-amber-300 text-amber-700 hover:bg-amber-100 dark:border-amber-700 dark:text-amber-300" onClick={() => navigate(`/dashboard/accounts/${id}/channels`)}>
            查看转发规则
          </Button>
        </div>
      )}

      {/* Channels View (migration) */}
      {showChannels && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-bold uppercase tracking-widest text-muted-foreground">转发规则</h3>
            <Button variant="outline" size="sm" className="rounded-full px-4 font-bold text-xs" onClick={() => navigate(`/dashboard/accounts/${id}`)}>
              返回概览
            </Button>
          </div>
          {channels.length === 0 ? (
            <div className="text-center py-16 space-y-3 border-2 border-dashed rounded-2xl">
              <Cable className="w-10 h-10 mx-auto text-muted-foreground/40" />
              <p className="text-sm text-muted-foreground">暂无转发规则</p>
            </div>
          ) : (
            <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
              {channels.map((ch) => (
                <Card key={ch.id} className="group relative border-border/50 bg-card/50 rounded-3xl transition-all hover:shadow-xl hover:border-primary/20 cursor-pointer" onClick={() => navigate(`/dashboard/accounts/${id}/channel/${ch.id}`)}>
                  <CardHeader>
                    <div className="flex justify-between items-start">
                      <div className="h-10 w-10 rounded-xl bg-muted flex items-center justify-center group-hover:bg-primary/10 transition-colors">
                        <Cable className="h-5 w-5 text-muted-foreground group-hover:text-primary transition-colors" />
                      </div>
                      <Badge variant={ch.enabled ? "default" : "secondary"} className="h-5 rounded-full text-[9px] font-black uppercase">{ch.enabled ? "运行中" : "已暂停"}</Badge>
                    </div>
                    <CardTitle className="text-lg font-bold mt-4">{ch.name}</CardTitle>
                    <p className="text-[10px] font-mono text-muted-foreground uppercase">@{ch.handle || "默认"}</p>
                  </CardHeader>
                  <CardFooter className="bg-muted/30 pt-3 flex justify-between items-center px-6">
                    <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest"></span>
                    <ArrowUpRight className="h-4 w-4 text-muted-foreground group-hover:text-primary transition-all" />
                  </CardFooter>
                </Card>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Installed Apps + Marketplace (default view) */}
      {!showChannels && <>
      {/* Installed Apps Section */}
      <div className="space-y-4">
        <h3 className="text-sm font-bold uppercase tracking-widest text-muted-foreground">已安装的应用</h3>
        {installations.length === 0 ? (
          <div className="text-center py-16 space-y-3 border-2 border-dashed rounded-2xl">
            <Blocks className="w-10 h-10 mx-auto text-muted-foreground/40" />
            <p className="text-sm text-muted-foreground">暂无安装的应用</p>
            <Button variant="outline" size="sm" onClick={() => marketplaceRef.current?.scrollIntoView({ behavior: "smooth" })}>
              去应用市场看看
            </Button>
          </div>
        ) : (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {installations.map((inst) => (
              <Card key={inst.id} className="group cursor-pointer rounded-3xl border-border/50 bg-card/50 transition-all hover:border-primary/30 hover:shadow-xl" onClick={() => navigate(`/dashboard/accounts/${id}/apps/${inst.id}`)}>
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3">
                      <AppIcon icon={inst.app_icon} iconUrl={inst.app_icon_url} size="h-10 w-10" />
                      <div className="space-y-0.5">
                        <CardTitle className="text-base font-bold group-hover:text-primary transition-colors">{inst.app_name}</CardTitle>
                        {inst.handle && (
                          <p className="text-[10px] font-mono text-muted-foreground">@{inst.handle}</p>
                        )}
                      </div>
                    </div>
                    <Badge variant={inst.enabled ? "default" : "outline"} className="h-5 rounded-full text-[9px] font-bold px-2">
                      {inst.enabled ? "运行中" : "已停用"}
                    </Badge>
                  </div>
                </CardHeader>
                <CardFooter className="bg-muted/30 pt-3 flex justify-between items-center px-6">
                  <span className="text-[10px] font-bold text-muted-foreground font-mono">{inst.app_slug}</span>
                  <ArrowUpRight className="h-4 w-4 text-muted-foreground group-hover:text-primary transition-all" />
                </CardFooter>
              </Card>
            ))}
          </div>
        )}
      </div>

      {/* App Marketplace Section */}
      <div ref={marketplaceRef} className="space-y-6">
        <h3 className="text-sm font-bold uppercase tracking-widest text-muted-foreground">应用市场</h3>

        {/* Builtin Apps */}
        {!marketplaceLoading && builtinApps.length > 0 && (
          <div className="space-y-3">
            <h4 className="text-xs font-medium text-muted-foreground">内置应用</h4>
            <div className="grid gap-4 md:grid-cols-3">
              {builtinApps.map((app: any) => (
                <Card key={app.slug || app.id} className="group relative overflow-hidden rounded-2xl border-border/50 bg-card/50 transition-all hover:shadow-xl hover:-translate-y-0.5">
                  <CardHeader className="pb-3">
                    <div className="flex items-center gap-3">
                      <AppIcon icon={app.icon} iconUrl={app.icon_url} size="h-10 w-10" />
                      <CardTitle className="text-base font-bold group-hover:text-primary transition-colors">{app.name}</CardTitle>
                    </div>
                  </CardHeader>
                  <CardContent className="pb-4">
                    <p className="text-xs text-muted-foreground leading-relaxed">{app.description}</p>
                  </CardContent>
                  <CardFooter className="bg-muted/30 pt-3 flex justify-end px-6">
                    <Button size="sm" onClick={() => navigate(`/dashboard/accounts/${id}/install/${app.id}`)} className="h-8 rounded-full px-4 gap-1.5 font-bold text-xs shadow-lg shadow-primary/10">
                      安装 <Download className="h-3 w-3" />
                    </Button>
                  </CardFooter>
                </Card>
              ))}
            </div>
          </div>
        )}

        {/* Marketplace Apps */}
        <div className="space-y-3">
          <h4 className="text-xs font-medium text-muted-foreground">应用市场</h4>
          {marketplaceLoading ? (
            <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
              {[1, 2, 3].map(i => <Card key={i} className="h-48 animate-pulse bg-muted/20 rounded-3xl" />)}
            </div>
          ) : marketplaceApps.length === 0 ? (
            <div className="text-center py-12 space-y-3 border-2 border-dashed rounded-2xl">
              <Blocks className="w-10 h-10 mx-auto text-muted-foreground/40" />
              <p className="text-sm text-muted-foreground">市场暂无应用</p>
            </div>
          ) : (
            <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
              {marketplaceApps.map((app) => (
                <Card key={app.slug || app.id} className="group relative overflow-hidden rounded-[2rem] border-border/50 bg-card/50 transition-all hover:shadow-2xl hover:-translate-y-1">
                  <CardHeader className="pb-4">
                    <div className="flex items-start gap-4">
                      <AppIcon icon={app.icon} iconUrl={app.icon_url} />
                      <div className="min-w-0 space-y-1 pt-1">
                        <CardTitle className="text-lg font-bold truncate group-hover:text-primary transition-colors">{app.name}</CardTitle>
                        <div className="flex flex-wrap gap-1.5">
                          {app.author && (
                            <span className="text-[10px] text-muted-foreground">{app.author}</span>
                          )}
                          {app.version && (
                            <Badge variant="outline" className="text-[9px] h-4 font-bold tracking-tighter opacity-60">
                              v{app.version}
                            </Badge>
                          )}
                          {app.installed && (
                            <Badge variant="default" className="text-[9px] h-4 font-bold tracking-tighter">
                              已安装
                            </Badge>
                          )}
                        </div>
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent className="pb-6">
                    <p className="text-xs text-muted-foreground leading-relaxed line-clamp-2 min-h-[2.5rem]">
                      {app.description || "暂无描述"}
                    </p>
                  </CardContent>
                  <CardFooter className="bg-muted/30 pt-4 flex justify-between items-center px-6">
                    <span className="text-[10px] font-bold text-muted-foreground">{app.author || app.slug}</span>
                    {app.installed && app.update_available ? (
                      <Button size="sm" variant="outline" disabled={syncing} onClick={() => handleInstallApp(app)} className="h-8 rounded-full px-4 gap-1.5 font-bold text-xs">
                        更新 <RefreshCw className="h-3 w-3" />
                      </Button>
                    ) : app.installed ? (
                      <Badge variant="secondary" className="text-xs">已安装</Badge>
                    ) : (
                      <Button size="sm" disabled={syncing} onClick={() => handleInstallApp(app)} className="h-8 rounded-full px-4 gap-1.5 font-bold text-xs shadow-lg shadow-primary/10">
                        安装 <Download className="h-3 w-3" />
                      </Button>
                    )}
                  </CardFooter>
                </Card>
              ))}
            </div>
          )}
        </div>
      </div>

      </>}
    </div>
  );
}

