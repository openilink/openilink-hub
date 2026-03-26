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
  Loader2,
  Zap,
  Eye,
  ExternalLink,
  ArrowRight,
  Copy,
  Check,
  RefreshCw,
} from "lucide-react";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { api } from "../lib/api";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useToast } from "@/hooks/use-toast";
import { Skeleton } from "@/components/ui/skeleton";
import { AppIcon } from "../components/app-icon";
import { APP_TEMPLATES, SCOPE_DESCRIPTIONS } from "../lib/constants";

function randomSuffix(): string {
  return Math.random().toString(36).slice(2, 8);
}

// ==================== Install Flow Types ====================

type InstallTarget =
  | { type: "template"; template: (typeof APP_TEMPLATES)[number] }
  | { type: "marketplace"; app: any };

type InstallResult = {
  appId: string;
  appName: string;
  token?: string;
  kind?: string;
  templateId?: string;
  guide?: string;
};

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
  const [marketplaceApps, setMarketplaceApps] = useState<any[]>([]);
  const [marketplaceLoading, setMarketplaceLoading] = useState(true);
  const [loading, setLoading] = useState(true);
  const [installTarget, setInstallTarget] = useState<InstallTarget | null>(null);
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
      setMarketplaceApps((await api.getMarketplace()) || []);
    } catch {
      setMarketplaceApps([]);
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

  const handleInstallClose = () => {
    setInstallTarget(null);
    loadInstallations();
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
              <Card key={inst.id} className="group cursor-pointer rounded-3xl border-border/50 bg-card/50 transition-all hover:border-primary/30 hover:shadow-xl" onClick={() => navigate(`/dashboard/apps/${inst.app_id}`)}>
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

        {/* Quick Create Templates */}
        <div className="space-y-3">
          <h4 className="text-xs font-medium text-muted-foreground">快速创建</h4>
          <div className="grid gap-4 md:grid-cols-3">
            {APP_TEMPLATES.map((tpl) => (
              <Card key={tpl.id} className="group relative overflow-hidden rounded-2xl border-border/50 bg-card/50 transition-all hover:shadow-xl hover:-translate-y-0.5">
                <CardHeader className="pb-3">
                  <div className="flex items-center gap-3">
                    <div className="h-10 w-10 rounded-xl bg-muted flex items-center justify-center text-xl border">
                      {tpl.emoji}
                    </div>
                    <CardTitle className="text-base font-bold group-hover:text-primary transition-colors">{tpl.name}</CardTitle>
                  </div>
                </CardHeader>
                <CardContent className="pb-4">
                  <p className="text-xs text-muted-foreground leading-relaxed">{tpl.description}</p>
                </CardContent>
                <CardFooter className="bg-muted/30 pt-3 flex justify-end px-6">
                  <Button size="sm" variant="outline" onClick={() => setInstallTarget({ type: "template", template: tpl })} className="h-8 rounded-full px-4 gap-1.5 font-bold text-xs">
                    安装 <Download className="h-3 w-3" />
                  </Button>
                </CardFooter>
              </Card>
            ))}
          </div>
        </div>

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
                      <Button size="sm" variant="outline" onClick={() => setInstallTarget({ type: "marketplace", app })} className="h-8 rounded-full px-4 gap-1.5 font-bold text-xs">
                        更新 <RefreshCw className="h-3 w-3" />
                      </Button>
                    ) : app.installed ? (
                      <Badge variant="secondary" className="text-xs">已安装</Badge>
                    ) : (
                      <Button size="sm" onClick={() => setInstallTarget({ type: "marketplace", app })} className="h-8 rounded-full px-4 gap-1.5 font-bold text-xs shadow-lg shadow-primary/10">
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

      {/* Install Dialog */}
      {installTarget && (
        <Dialog open={!!installTarget} onOpenChange={(o: boolean) => !o && handleInstallClose()}>
          <DialogContent className="sm:max-w-2xl rounded-[2rem]">
            <DialogHeader className="sr-only">
              <DialogTitle>安装应用</DialogTitle>
              <DialogDescription>选择账号并确认安装。</DialogDescription>
            </DialogHeader>
            <InstallFlowDialog target={installTarget} botId={id!} onClose={handleInstallClose} />
          </DialogContent>
        </Dialog>
      )}
      </>}
    </div>
  );
}

// ==================== Install Flow Dialog ====================

function InstallFlowDialog({ target, botId, onClose }: { target: InstallTarget; botId: string; onClose: () => void }) {
  const [handle, setHandle] = useState("");
  const [saving, setSaving] = useState(false);
  const [result, setResult] = useState<InstallResult | null>(null);
  const { toast } = useToast();

  const isTemplate = target.type === "template";
  const appName = isTemplate ? target.template.name : target.app.name;
  const appDescription = isTemplate ? target.template.description : target.app.description;
  const appEmoji = isTemplate ? target.template.emoji : undefined;
  const appIcon = isTemplate ? undefined : target.app.icon;
  const appIconUrl = isTemplate ? undefined : target.app.icon_url;
  const scopes = isTemplate ? target.template.scopes : (target.app.scopes || []);
  const events = isTemplate ? target.template.events : (target.app.events || []);
  const readScopes = scopes.filter((s: string) => s.includes("read"));
  const writeScopes = scopes.filter((s: string) => !s.includes("read"));

  useEffect(() => {
    if (isTemplate) {
      setHandle(target.template.id);
    } else {
      setHandle(target.app.slug || "");
    }
  }, [target, isTemplate]);

  async function handleInstall() {
    setSaving(true);
    try {
      if (isTemplate) {
        const slug = `${target.template.id}-${randomSuffix()}`;
        const created = await api.createApp({
          name: target.template.name,
          slug,
          description: target.template.description,
          icon: target.template.emoji,
          kind: "integration",
          scopes: target.template.scopes,
          events: target.template.events,
          readme: target.template.readme,
          guide: target.template.guide,
        });
        const installation = await api.installApp(created.id, {
          bot_id: botId,
          handle: handle.trim() || undefined,
          scopes: target.template.scopes,
        });
        setResult({
          appId: created.id,
          appName: target.template.name,
          token: installation.token,
          kind: "integration",
          templateId: target.template.id,
          guide: target.template.guide,
        });
      } else {
        const app = target.app;
        let appId = app.local_id || app.id;
        if (!appId) {
          const synced = await api.syncMarketplaceApp(app.slug);
          appId = synced.id;
        }
        const installation = await api.installApp(appId, {
          bot_id: botId,
          handle: handle.trim() || undefined,
          scopes: app.scopes,
        });
        setResult({
          appId: appId,
          appName: app.name,
          token: installation.token,
          kind: app.kind,
        });
      }
      toast({ title: "安装成功", description: `已安装 ${appName}。` });
    } catch (e: any) {
      toast({ variant: "destructive", title: "安装失败", description: e.message });
    }
    setSaving(false);
  }

  if (result) {
    return <InstallResultScreen result={result} onClose={onClose} />;
  }

  return (
    <div className="py-2">
      <div className="flex flex-col sm:flex-row gap-6">
        {/* Left: App identity */}
        <div className="sm:w-2/5 space-y-4 sm:border-r sm:pr-6">
          <div className="flex items-center gap-3">
            {appEmoji ? (
              <div className="h-14 w-14 rounded-xl bg-muted flex items-center justify-center text-2xl border">{appEmoji}</div>
            ) : (
              <AppIcon icon={appIcon} iconUrl={appIconUrl} size="h-14 w-14" />
            )}
            <div>
              <h3 className="text-lg font-bold">{appName}</h3>
              {!isTemplate && target.app.slug && (
                <p className="text-xs text-muted-foreground font-mono">{target.app.slug}</p>
              )}
            </div>
          </div>
          {appDescription && (
            <p className="text-sm text-muted-foreground leading-relaxed">{appDescription}</p>
          )}
          {!isTemplate && target.app.homepage && (
            <a href={target.app.homepage} target="_blank" rel="noopener noreferrer" className="text-xs text-primary hover:underline flex items-center gap-1">
              <ExternalLink className="h-3 w-3" /> 应用主页
            </a>
          )}
        </div>

        {/* Right: Permissions + config */}
        <div className="sm:w-3/5 space-y-5">
          <div className="space-y-3">
            <h4 className="text-xs font-bold uppercase tracking-wider text-muted-foreground">此应用将能够：</h4>

            {readScopes.length > 0 && (
              <div className="space-y-1.5">
                <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wide">查看</p>
                {readScopes.map((s: string) => (
                  <div key={s} className="flex items-start gap-2 text-sm">
                    <Eye className="h-3.5 w-3.5 mt-0.5 text-muted-foreground shrink-0" />
                    <span>{SCOPE_DESCRIPTIONS[s] || s}</span>
                  </div>
                ))}
              </div>
            )}

            {writeScopes.length > 0 && (
              <div className="space-y-1.5">
                <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wide">操作</p>
                {writeScopes.map((s: string) => (
                  <div key={s} className="flex items-start gap-2 text-sm">
                    <Zap className="h-3.5 w-3.5 mt-0.5 text-primary shrink-0" />
                    <span>{SCOPE_DESCRIPTIONS[s] || s}</span>
                  </div>
                ))}
              </div>
            )}

            {events.length > 0 && (
              <div className="space-y-1.5">
                <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wide">事件订阅</p>
                <div className="flex flex-wrap gap-1.5">
                  {events.map((e: string) => (
                    <Badge key={e} variant="outline" className="font-mono text-[10px]">{e}</Badge>
                  ))}
                </div>
              </div>
            )}

            {scopes.length === 0 && events.length === 0 && (
              <p className="text-sm text-muted-foreground">接收 @mention 消息并执行响应。</p>
            )}
          </div>

          <div className="space-y-3 pt-2 border-t">
            <div className="space-y-1.5">
              <label htmlFor="bd-install-handle" className="text-xs font-medium">Handle（可选）</label>
              <Input id="bd-install-handle" value={handle} onChange={e => setHandle(e.target.value)} className="h-9 font-mono" placeholder="如 notify-prod" />
              <p className="text-[10px] text-muted-foreground">用户发送 @{handle || "handle"} 触发此应用</p>
            </div>
          </div>
        </div>
      </div>

      <div className="flex justify-end gap-2 pt-4 mt-4 border-t">
        <Button variant="ghost" onClick={onClose}>取消</Button>
        <Button onClick={handleInstall} disabled={saving} className="px-6">
          {saving && <Loader2 className="h-4 w-4 animate-spin mr-1.5" />}
          允许并安装
        </Button>
      </div>
    </div>
  );
}

// ==================== Install Result Screen ====================

function InstallResultScreen({ result, onClose }: { result: InstallResult; onClose: () => void }) {
  const navigate = useNavigate();
  const [copied, setCopied] = useState(false);
  const hubUrl = window.location.origin;

  function handleCopy(text: string) {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  const isIntegration = result.kind === "integration";

  return (
    <div className="py-2 space-y-6">
      <div className="text-center space-y-2">
        <div className="h-16 w-16 rounded-full bg-primary/10 flex items-center justify-center mx-auto">
          <Check className="h-8 w-8 text-primary" />
        </div>
        <h3 className="text-xl font-bold">安装成功</h3>
        <p className="text-sm text-muted-foreground">{result.appName} 已安装。</p>
      </div>

      {isIntegration && result.token && (
        <div className="space-y-4">
          <div className="space-y-2">
            <label className="text-xs font-bold uppercase tracking-widest text-muted-foreground">Token</label>
            <div className="flex items-center gap-2 p-3 rounded-lg border bg-muted/30">
              <code className="text-sm font-mono flex-1 break-all">{result.token}</code>
              <button onClick={() => handleCopy(result.token!)} className="cursor-pointer text-muted-foreground hover:text-foreground shrink-0" aria-label="复制">
                {copied ? <Check className="w-4 h-4 text-primary" /> : <Copy className="w-4 h-4" />}
              </button>
            </div>
            <p className="text-[10px] text-destructive font-medium">请妥善保管此 Token，关闭后将无法再次查看。</p>
          </div>

          <div className="space-y-3">
            <details className="group">
              <summary className="text-sm font-medium cursor-pointer flex items-center gap-2 select-none">
                <ArrowRight className="h-3.5 w-3.5 transition-transform group-open:rotate-90" />
                HTTP 发消息
              </summary>
              <pre className="mt-2 p-3 rounded-lg bg-muted/30 border text-xs font-mono overflow-x-auto whitespace-pre-wrap">{`curl -X POST ${hubUrl}/bot/v1/message/send \\
  -H "Authorization: Bearer ${result.token}" \\
  -d '{"to":"wxid_xxx","content":"hello"}'`}</pre>
            </details>

            {(result.templateId === "websocket-app" || result.templateId === "openclaw-channel") && (
              <details className="group">
                <summary className="text-sm font-medium cursor-pointer flex items-center gap-2 select-none">
                  <ArrowRight className="h-3.5 w-3.5 transition-transform group-open:rotate-90" />
                  WebSocket 连接
                </summary>
                <pre className="mt-2 p-3 rounded-lg bg-muted/30 border text-xs font-mono overflow-x-auto whitespace-pre-wrap">{`wss://${hubUrl.replace(/^https?:\/\//, "")}/bot/v1/ws?token=${result.token}`}</pre>
              </details>
            )}
          </div>
        </div>
      )}

      {!isIntegration && (
        <div className="text-center">
          <p className="text-sm text-muted-foreground">应用已安装到你的账号。</p>
        </div>
      )}

      <div className="flex justify-end gap-2 pt-2 border-t">
        <Button variant="outline" onClick={() => { onClose(); navigate(`/dashboard/apps/${result.appId}`); }}>
          查看应用详情
        </Button>
        <Button onClick={onClose}>完成</Button>
      </div>
    </div>
  );
}
