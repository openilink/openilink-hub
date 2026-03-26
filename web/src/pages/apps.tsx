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
  Search,
  Rocket,
  RefreshCw,
} from "lucide-react";
import { api } from "../lib/api";
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
  return name.toLowerCase().replace(/[^a-z0-9一-鿿]+/g, "-").replace(/^-|-$/g, "").slice(0, 32);
}

// ==================== Page ====================

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
  const navigate = useNavigate();
  const [marketplaceApps, setMarketplaceApps] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [pendingApp, setPendingApp] = useState<any>(null);
  const [bots, setBots] = useState<any[]>([]);
  const [selectedBotId, setSelectedBotId] = useState("");
  const { toast } = useToast();

  useEffect(() => {
    setLoading(true);
    api.getMarketplaceApps().then(l => setMarketplaceApps(l || [])).catch(() => setMarketplaceApps([])).finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    api.listBots().then(l => { setBots(l || []); if (l?.length) setSelectedBotId(l[0].id); });
  }, []);

  const filteredApps = marketplaceApps.filter(a =>
    !search || a.name?.toLowerCase().includes(search.toLowerCase()) || (a.slug || "").toLowerCase().includes(search.toLowerCase())
  );

  async function handleInstallConfirm() {
    if (!pendingApp || !selectedBotId) return;
    try {
      if (pendingApp.local_id) {
        navigate(`/dashboard/accounts/${selectedBotId}/install/${pendingApp.local_id}`);
      } else {
        const synced = await api.syncMarketplaceApp(pendingApp.slug);
        navigate(`/dashboard/accounts/${selectedBotId}/install/${synced.id}`);
      }
      setPendingApp(null);
    } catch (e: any) {
      toast({ variant: "destructive", title: "操作失败", description: e.message });
    }
  }

  return (
    <div className="space-y-8">
      <div className="relative max-w-md">
        <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
        <Input placeholder="搜索应用..." value={search} onChange={e => setSearch(e.target.value)} className="pl-10 h-10 rounded-full bg-card shadow-sm border-border/50" aria-label="搜索应用" />
      </div>

      {/* Marketplace Apps */}
      <div className="space-y-4">
        <h3 className="text-sm font-bold uppercase tracking-widest text-muted-foreground">应用市场</h3>
        {loading ? (
          <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {[1, 2, 3].map(i => <Card key={i} className="h-48 animate-pulse bg-muted/20 rounded-3xl" />)}
          </div>
        ) : filteredApps.length === 0 ? (
          <div className="text-center py-16 space-y-3 border-2 border-dashed rounded-2xl">
            <Blocks className="w-10 h-10 mx-auto text-muted-foreground/40" />
            <p className="text-sm text-muted-foreground">{search ? "没有匹配的应用" : "市场暂无应用"}</p>
          </div>
        ) : (
          <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {filteredApps.map((app) => (
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
                    <Button size="sm" variant="outline" onClick={() => setPendingApp(app)} className="h-8 rounded-full px-4 gap-1.5 font-bold text-xs">
                      更新 <RefreshCw className="h-3 w-3" />
                    </Button>
                  ) : app.installed ? (
                    <Badge variant="secondary" className="text-xs">已安装</Badge>
                  ) : (
                    <Button size="sm" onClick={() => setPendingApp(app)} className="h-8 rounded-full px-4 gap-1.5 font-bold text-xs shadow-lg shadow-primary/10">
                      安装 <Download className="h-3 w-3" />
                    </Button>
                  )}
                </CardFooter>
              </Card>
            ))}
          </div>
        )}
      </div>

      <Dialog open={!!pendingApp} onOpenChange={(o) => !o && setPendingApp(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>选择账号</DialogTitle>
            <DialogDescription>选择要安装「{pendingApp?.name}」的账号。</DialogDescription>
          </DialogHeader>
          {bots.length === 0 ? (
            <p className="text-sm text-muted-foreground py-4">请先创建一个账号。</p>
          ) : (
            <div className="space-y-4 pt-2">
              <select value={selectedBotId} onChange={e => setSelectedBotId(e.target.value)}
                className="w-full h-9 px-3 rounded-md border bg-background text-sm">
                {bots.map(b => <option key={b.id} value={b.id}>{b.name}</option>)}
              </select>
              <Button className="w-full" onClick={handleInstallConfirm}>继续安装</Button>
            </div>
          )}
        </DialogContent>
      </Dialog>
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
