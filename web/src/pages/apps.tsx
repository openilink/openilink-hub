import { useEffect, useState } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import {
  Plus,
  Blocks,
  Download,
  ArrowRight,
  Loader2,
  Zap,
  Search,
  Eye,
  ExternalLink,
  Rocket,
} from "lucide-react";
import { api } from "../lib/api";
import { SCOPE_DESCRIPTIONS } from "../lib/constants";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogFooter,
} from "@/components/ui/dialog";
import { useToast } from "@/hooks/use-toast";
import { AppIcon } from "../components/app-icon";

function slugify(name: string): string {
  return name.toLowerCase().replace(/[^a-z0-9\u4e00-\u9fff]+/g, "-").replace(/^-|-$/g, "").slice(0, 32);
}

export function AppsPage() {
  const navigate = useNavigate();
  const location = useLocation();
  
  const activeTab = location.pathname.split("/").pop() || "my";
  const tab = activeTab === "marketplace" ? "marketplace" : "my";

  return (
    <div className="space-y-8">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div className="flex items-center gap-4">
          <div className="h-12 w-12 rounded-2xl bg-primary/10 flex items-center justify-center text-primary shadow-sm border border-primary/20">
            <Blocks className="h-6 w-6" />
          </div>
          <div>
            <h2 className="text-3xl font-bold tracking-tight">应用</h2>
            <p className="text-muted-foreground">管理和安装应用。</p>
          </div>
        </div>
      </div>

      <Tabs value={tab} onValueChange={(v) => navigate(`/dashboard/apps/${v}`)}>
        <TabsList>
          <TabsTrigger value="my">我的应用</TabsTrigger>
          <TabsTrigger value="marketplace">应用市场</TabsTrigger>
        </TabsList>
        <TabsContent value="my" className="flex flex-col gap-6 mt-6">
          <MyAppsTab />
        </TabsContent>
        <TabsContent value="marketplace" className="flex flex-col gap-6 mt-6">
          <MarketplaceTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}

// ==================== Marketplace (Store) ====================

function MarketplaceTab() {
  const [apps, setApps] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [installApp, setInstallApp] = useState<any>(null);

  useEffect(() => {
    setLoading(true);
    api.listApps({ listed: true }).then(l => setApps(l || [])).finally(() => setLoading(false));
  }, []);

  if (loading) return (
    <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
      {[1, 2, 3].map(i => <Card key={i} className="h-48 animate-pulse bg-muted/20 rounded-3xl" />)}
    </div>
  );

  return (
    <div className="space-y-6">
      <div className="relative max-w-md">
        <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
        <Input placeholder="搜索应用..." className="pl-10 h-10 rounded-full bg-card shadow-sm border-border/50" />
      </div>

      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        {apps.map((app) => (
          <Card key={app.id} className="group relative overflow-hidden rounded-[2rem] border-border/50 bg-card/50 transition-all hover:shadow-2xl hover:-translate-y-1">
            <CardHeader className="pb-4">
              <div className="flex items-start gap-4">
                <AppIcon icon={app.icon} iconUrl={app.icon_url} />
                <div className="min-w-0 space-y-1 pt-1">
                  <CardTitle className="text-lg font-bold truncate group-hover:text-primary transition-colors">{app.name}</CardTitle>
                  <div className="flex flex-wrap gap-1.5">
                    <Badge variant="secondary" className="text-[9px] h-4 font-bold uppercase tracking-tighter bg-primary/5 text-primary border-none">
                      {app.tools?.length || 0} 个工具
                    </Badge>
                    <Badge variant="outline" className="text-[9px] h-4 font-bold uppercase tracking-tighter opacity-60">
                      {app.status}
                    </Badge>
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
              <div className="flex items-center gap-3">
                 <div className="flex -space-x-2">
                    {[1, 2].map(i => <div key={i} className="h-6 w-6 rounded-full border-2 border-background bg-muted flex items-center justify-center"><Zap className="h-3 w-3 text-yellow-500" /></div>)}
                 </div>
                 <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">Verified</span>
              </div>
              <Button size="sm" onClick={() => setInstallApp(app)} className="h-8 rounded-full px-4 gap-1.5 font-bold text-xs shadow-lg shadow-primary/10">
                安装 <Download className="h-3 w-3" />
              </Button>
            </CardFooter>
          </Card>
        ))}
      </div>

      {installApp && (
        <Dialog open={!!installApp} onOpenChange={(o: boolean) => !o && setInstallApp(null)}>
          <DialogContent className="sm:max-w-2xl rounded-[2rem]">
            <InstallFlowDialog app={installApp} onClose={() => setInstallApp(null)} />
          </DialogContent>
        </Dialog>
      )}

    </div>
  );
}

function InstallFlowDialog({ app, onClose }: { app: any; onClose: () => void }) {
  const [bots, setBots] = useState<any[]>([]);
  const [botId, setBotId] = useState("");
  const [handle, setHandle] = useState(app.slug || "");
  const [saving, setSaving] = useState(false);
  const { toast } = useToast();

  useEffect(() => {
    api.listBots().then(l => {
      const items = l || []; setBots(items);
      if (items.length) setBotId(items[0].id);
    });
  }, []);

  async function handleInstall() {
    if (!botId) return;
    setSaving(true);
    try {
      await api.installApp(app.id, { bot_id: botId, handle: handle.trim() || undefined });
      toast({ title: "安装成功", description: `已安装 ${app.name}。` });
      onClose();
    } catch (e: any) {
      toast({ variant: "destructive", title: "安装失败", description: e.message });
    }
    setSaving(false);
  }

  const tools = (app.tools || []) as any[];
  const events = (app.events || []) as string[];
  const scopes = (app.scopes || []) as string[];
  const readScopes = scopes.filter(s => s.includes("read"));
  const writeScopes = scopes.filter(s => !s.includes("read"));

  return (
    <div className="py-2">
      <div className="flex flex-col sm:flex-row gap-6">
        {/* Left: App identity */}
        <div className="sm:w-2/5 space-y-4 sm:border-r sm:pr-6">
          <div className="flex items-center gap-3">
            <AppIcon icon={app.icon} iconUrl={app.icon_url} size="h-14 w-14" />
            <div>
              <h3 className="text-lg font-bold">{app.name}</h3>
              <p className="text-xs text-muted-foreground font-mono">{app.slug}</p>
            </div>
          </div>
          {app.description && (
            <p className="text-sm text-muted-foreground leading-relaxed">{app.description}</p>
          )}
          {app.homepage && (
            <a href={app.homepage} target="_blank" rel="noopener noreferrer" className="text-xs text-primary hover:underline flex items-center gap-1">
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
                {readScopes.map(s => (
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
                {writeScopes.map(s => (
                  <div key={s} className="flex items-start gap-2 text-sm">
                    <Zap className="h-3.5 w-3.5 mt-0.5 text-primary shrink-0" />
                    <span>{SCOPE_DESCRIPTIONS[s] || s}</span>
                  </div>
                ))}
              </div>
            )}

            {tools.length > 0 && (
              <div className="space-y-1.5">
                <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wide">命令</p>
                <div className="flex flex-wrap gap-1.5">
                  {tools.map((t: any) => (
                    <Badge key={t.name} variant="secondary" className="font-mono text-xs">/{t.command || t.name}</Badge>
                  ))}
                </div>
              </div>
            )}

            {events.length > 0 && (
              <div className="space-y-1.5">
                <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wide">事件订阅</p>
                <div className="flex flex-wrap gap-1.5">
                  {events.map(e => (
                    <Badge key={e} variant="outline" className="font-mono text-[10px]">{e}</Badge>
                  ))}
                </div>
              </div>
            )}
          </div>

          <div className="space-y-3 pt-2 border-t">
            <div className="space-y-1.5">
              <label className="text-xs font-medium">安装到账号</label>
              <select value={botId} onChange={e => setBotId(e.target.value)}
                className="w-full h-9 px-3 rounded-lg border bg-background text-sm outline-none focus:ring-2 focus:ring-primary/20">
                {bots.map(b => <option key={b.id} value={b.id}>{b.name}</option>)}
              </select>
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium">Handle</label>
              <Input value={handle} onChange={e => setHandle(e.target.value)} className="h-9 font-mono" placeholder="如 notify-prod" />
              <p className="text-[10px] text-muted-foreground">用户发送 @{handle || "handle"} 触发此应用</p>
            </div>
          </div>
        </div>
      </div>

      <div className="flex justify-end gap-2 pt-4 mt-4 border-t">
        <Button variant="ghost" onClick={onClose}>取消</Button>
        <Button onClick={handleInstall} disabled={saving || !botId || !handle.trim()} className="px-6">
          {saving && <Loader2 className="h-4 w-4 animate-spin mr-1.5" />}
          允许并安装
        </Button>
      </div>
    </div>
  );
}

// ==================== Studio (Development) ====================

function MyAppsTab() {
  const navigate = useNavigate();
  const [apps, setApps] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [isCreating, setIsCreating] = useState(false);
  const [form, setForm] = useState({ name: "", slug: "", description: "", icon: "" });
  const { toast } = useToast();

  async function load() {
    setLoading(true);
    try { setApps((await api.listApps()) || []); } finally { setLoading(false); }
  }
  useEffect(() => { load(); }, []);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!form.name.trim()) return;
    try {
      await api.createApp(form);
      toast({ title: "创建成功", description: "应用已创建。" });
      setIsCreating(false);
      load();
    } catch (e: any) {
      toast({ variant: "destructive", title: "创建失败", description: e.message });
    }
  }

  if (loading && apps.length === 0) return (
    <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
      {[1, 2, 3].map(i => <Card key={i} className="h-40 animate-pulse bg-muted/20 rounded-3xl" />)}
    </div>
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div />
        <Dialog open={isCreating} onOpenChange={setIsCreating}>
          <DialogTrigger asChild>
            <Button className="rounded-full h-10 px-6 gap-2 shadow-lg shadow-primary/20">
              <Plus className="h-4 w-4" /> 创建应用
            </Button>
          </DialogTrigger>
          <DialogContent className="rounded-[2rem]">
            <DialogHeader><DialogTitle className="text-2xl font-bold">创建应用</DialogTitle><DialogDescription>填写基本信息。</DialogDescription></DialogHeader>
            <form onSubmit={handleCreate} className="space-y-5 pt-4">
               <div className="space-y-2"><label className="text-xs font-bold uppercase text-muted-foreground">名称</label><Input placeholder="例如: 通知助手" value={form.name} onChange={e => { const n = e.target.value; setForm({...form, name: n, slug: slugify(n)}); }} /></div>
               <div className="space-y-2"><label className="text-xs font-bold uppercase text-muted-foreground">唯一标识</label><Input value={form.slug} onChange={e => setForm({...form, slug: e.target.value})} className="font-mono" /></div>
               <div className="space-y-2"><label className="text-xs font-bold uppercase text-muted-foreground">描述</label><Input placeholder="这个应用是用来..." value={form.description} onChange={e => setForm({...form, description: e.target.value})} /></div>
               <DialogFooter className="pt-4"><Button type="submit" className="w-full rounded-full h-11">创建</Button></DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        {apps.map((app) => (
          <Card key={app.id} className="group cursor-pointer rounded-3xl border-border/50 bg-card/50 transition-all hover:border-primary/30 hover:shadow-xl" onClick={() => navigate(`/dashboard/apps/${app.id}`)}>
            <CardHeader className="pb-4 flex flex-row items-center justify-between space-y-0">
              <div className="flex items-center gap-4">
                <AppIcon icon={app.icon} iconUrl={app.icon_url} size="h-10 w-10" />
                <div className="space-y-0.5">
                  <CardTitle className="text-base font-bold">{app.name}</CardTitle>
                  <p className="text-[10px] font-mono uppercase tracking-widest text-muted-foreground">{app.slug}</p>
                </div>
              </div>
              <Badge variant={app.status === "active" ? "default" : "secondary"} className="h-5 rounded-full text-[9px] px-2 font-bold">{app.status === "active" ? "已发布" : "草稿"}</Badge>
            </CardHeader>
            <CardFooter className="bg-muted/30 pt-3 flex justify-between items-center px-6">
               <span className="text-[10px] font-bold text-muted-foreground flex items-center gap-1.5"><Rocket className="h-3 w-3" /> {app.tools?.length || 0} 个工具已配置</span>
               <ArrowRight className="h-4 w-4 text-muted-foreground group-hover:text-primary group-hover:translate-x-1 transition-all" />
            </CardFooter>
          </Card>
        ))}
        
        {apps.length === 0 && (
          <div className="col-span-full py-24 border-2 border-dashed rounded-[2rem] flex flex-col items-center justify-center text-center bg-muted/5">
            <div className="h-20 w-20 rounded-3xl bg-background border shadow-sm flex items-center justify-center mb-6">
              <Blocks className="h-10 w-10 text-primary/40" />
            </div>
            <h3 className="text-xl font-bold">还没有应用</h3>
            <p className="text-muted-foreground mt-2 max-w-sm">创建你的第一个应用。</p>
            <Button variant="outline" className="mt-8 h-11 px-8 rounded-full" onClick={() => setIsCreating(true)}>创建应用</Button>
          </div>
        )}
      </div>
    </div>
  );
}
