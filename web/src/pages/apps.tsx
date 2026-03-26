import { useEffect, useState } from "react";
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
  Search,
  RefreshCw,
} from "lucide-react";
import { api } from "../lib/api";
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

      <MarketplaceContent />
    </div>
  );
}

// ==================== Marketplace (Store) ====================

function MarketplaceContent() {
  const navigate = useNavigate();
  const [marketplaceApps, setMarketplaceApps] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [pendingApp, setPendingApp] = useState<any>(null);
  const [bots, setBots] = useState<any[]>([]);
  const [selectedBotId, setSelectedBotId] = useState("");
  const [syncing, setSyncing] = useState(false);
  const { toast } = useToast();

  useEffect(() => {
    setLoading(true);
    api.listApps({ listed: true }).then(l => setMarketplaceApps(l || [])).catch(() => setMarketplaceApps([])).finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    api.listBots().then(l => { setBots(l || []); if (l?.length) setSelectedBotId(l[0].id); });
  }, []);

  const filteredApps = marketplaceApps.filter(a =>
    !search || a.name?.toLowerCase().includes(search.toLowerCase()) || (a.slug || "").toLowerCase().includes(search.toLowerCase())
  );

  async function handleInstallConfirm() {
    if (!pendingApp || !selectedBotId) return;
    const appId = pendingApp.id || pendingApp.local_id;
    if (appId) {
      navigate(`/dashboard/accounts/${selectedBotId}/install/${appId}`);
      setPendingApp(null);
      return;
    }
    // Marketplace app without local record — sync first
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
              <Button className="w-full" disabled={syncing} onClick={handleInstallConfirm}>继续安装</Button>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

