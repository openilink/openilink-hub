import { useEffect, useState, useMemo } from "react";
import { useNavigate } from "react-router-dom";
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
  Blocks,
  Download,
  Loader2,
  Search,
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
} from "@/components/ui/dialog";
import { useToast } from "@/hooks/use-toast";
import { AppIcon } from "../components/app-icon";

// ==================== Page ====================

export function AppsPage() {
  const navigate = useNavigate();
  const { toast } = useToast();
  const [listedApps, setListedApps] = useState<any[]>([]);
  const [registryApps, setRegistryApps] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [pendingApp, setPendingApp] = useState<any>(null);
  const [bots, setBots] = useState<any[]>([]);
  const [selectedBotId, setSelectedBotId] = useState("");
  const [syncing, setSyncing] = useState(false);

  useEffect(() => {
    setLoading(true);
    Promise.all([
      api.listApps({ listing: "listed" }).catch(() => []),
      api.getMarketplaceApps().catch(() => []),
      api.listBots().catch(() => []),
    ]).then(([listed, registry, botList]) => {
      setListedApps(listed || []);
      setRegistryApps(registry || []);
      setBots(botList || []);
      if (botList?.length) setSelectedBotId(botList[0].id);
    }).finally(() => setLoading(false));
  }, []);

  // Group registry apps by registry_name
  const registryGroups = useMemo(() => {
    const groups: Record<string, any[]> = {};
    for (const app of registryApps) {
      const name = app.registry_name || app.registry_url || "Registry";
      if (!groups[name]) groups[name] = [];
      groups[name].push(app);
    }
    return groups;
  }, [registryApps]);

  const registryNames = Object.keys(registryGroups);
  const showTabs = registryNames.length > 0;

  async function handleInstallConfirm() {
    if (!pendingApp || !selectedBotId) return;
    const appId = pendingApp.id || pendingApp.local_id;
    if (appId) {
      navigate(`/dashboard/accounts/${selectedBotId}/install/${appId}`);
      setPendingApp(null);
      return;
    }
    setSyncing(true);
    try {
      const synced = await api.syncMarketplaceApp(pendingApp.slug);
      navigate(`/dashboard/accounts/${selectedBotId}/install/${synced.id}`);
      setPendingApp(null);
    } catch (e: any) {
      toast({ variant: "destructive", title: "同步失败", description: e.message });
    } finally {
      setSyncing(false);
    }
  }

  function filterApps(apps: any[]) {
    if (!search) return apps;
    const q = search.toLowerCase();
    return apps.filter(a =>
      a.name?.toLowerCase().includes(q) || (a.slug || "").toLowerCase().includes(q)
    );
  }

  return (
    <div className="space-y-8">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div className="flex items-center gap-4">
          <div className="h-12 w-12 rounded-2xl bg-primary/10 flex items-center justify-center text-primary shadow-sm border border-primary/20">
            <Blocks className="h-6 w-6" />
          </div>
          <div>
            <h2 className="text-3xl font-bold tracking-tight">应用市场</h2>
            <p className="text-muted-foreground">浏览和安装应用。</p>
          </div>
        </div>
      </div>

      <div className="relative max-w-md">
        <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
        <Input placeholder="搜索应用..." value={search} onChange={e => setSearch(e.target.value)} className="pl-10 h-10 rounded-full bg-card shadow-sm border-border/50" aria-label="搜索应用" />
      </div>

      {loading ? (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map(i => <Card key={i} className="h-48 animate-pulse bg-muted/20 rounded-3xl" />)}
        </div>
      ) : showTabs ? (
        <Tabs defaultValue="local">
          <TabsList>
            <TabsTrigger value="local">本站</TabsTrigger>
            {registryNames.map(name => (
              <TabsTrigger key={name} value={name}>{name}</TabsTrigger>
            ))}
          </TabsList>
          <TabsContent value="local" className="mt-6">
            <AppGrid apps={filterApps(listedApps)} search={search} onInstall={setPendingApp} />
          </TabsContent>
          {registryNames.map(name => (
            <TabsContent key={name} value={name} className="mt-6">
              <AppGrid apps={filterApps(registryGroups[name])} search={search} onInstall={setPendingApp} />
            </TabsContent>
          ))}
        </Tabs>
      ) : (
        <AppGrid apps={filterApps(listedApps)} search={search} onInstall={setPendingApp} />
      )}

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
              <Button className="w-full" disabled={syncing} onClick={handleInstallConfirm}>{syncing && <Loader2 className="h-4 w-4 animate-spin mr-2" />}继续安装</Button>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ==================== App Grid ====================

function AppGrid({ apps, search, onInstall }: { apps: any[]; search: string; onInstall: (app: any) => void }) {
  if (apps.length === 0) {
    return (
      <div className="text-center py-16 space-y-3 border-2 border-dashed rounded-2xl">
        <Blocks className="w-10 h-10 mx-auto text-muted-foreground/40" />
        <p className="text-sm text-muted-foreground">{search ? "没有匹配的应用" : "暂无应用"}</p>
      </div>
    );
  }

  return (
    <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
      {apps.map((app) => (
        <Card key={`${app.registry || "local"}-${app.slug || app.id}`} className="group relative overflow-hidden rounded-2xl border-border/50 bg-card/50 transition-all hover:shadow-2xl hover:-translate-y-1">
          <CardHeader className="pb-4">
            <div className="flex items-start gap-4">
              <AppIcon icon={app.icon} iconUrl={app.icon_url} />
              <div className="min-w-0 space-y-1 pt-1">
                <CardTitle className="text-lg font-bold truncate group-hover:text-primary transition-colors">{app.name}</CardTitle>
                <div className="flex flex-wrap gap-1.5">
                  {(app.author || app.owner_name) && (
                    <span className="text-xs text-muted-foreground">{app.author || app.owner_name}</span>
                  )}
                  {app.version && (
                    <Badge variant="outline" className="h-4 font-bold tracking-tighter opacity-60">
                      v{app.version}
                    </Badge>
                  )}
                  {app.installed && (
                    <Badge variant="default" className="h-4 font-bold tracking-tighter">
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
            <span className="text-xs font-bold text-muted-foreground">{app.author || app.owner_name || app.slug}</span>
            {app.installed && app.update_available ? (
              <Button size="sm" variant="outline" onClick={() => onInstall(app)} className="h-8 rounded-full px-4 gap-1.5 font-bold text-xs">
                更新 <RefreshCw className="h-3 w-3" />
              </Button>
            ) : app.installed ? (
              <Badge variant="secondary" className="text-xs">已安装</Badge>
            ) : (
              <Button size="sm" onClick={() => onInstall(app)} className="h-8 rounded-full px-4 gap-1.5 font-bold text-xs shadow-lg shadow-primary/10">
                安装 <Download className="h-3 w-3" />
              </Button>
            )}
          </CardFooter>
        </Card>
      ))}
    </div>
  );
}
