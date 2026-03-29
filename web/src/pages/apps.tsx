import { useState, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Badge } from "../components/ui/badge";
import { Blocks, Download, Loader2, Search, RefreshCw } from "lucide-react";
import { api, botDisplayName } from "../lib/api";
import { useApps } from "@/hooks/use-apps";
import { useBots } from "@/hooks/use-bots";
import { useMarketplaceApps, useSyncMarketplaceApp } from "@/hooks/use-marketplace";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useToast } from "@/hooks/use-toast";
import { AppIcon } from "../components/app-icon";
import { parseTools } from "../components/tools-display";

// ==================== Page ====================

export function AppsPage() {
  const navigate = useNavigate();
  const { toast } = useToast();
  const { data: listedApps = [], isLoading: listedLoading } = useApps({ listing: "listed" });
  const { data: registryApps = [], isLoading: registryLoading } = useMarketplaceApps();
  const { data: bots = [] } = useBots();
  const syncAppMutation = useSyncMarketplaceApp();
  const loading = listedLoading || registryLoading;
  const [search, setSearch] = useState("");
  const [pendingApp, setPendingApp] = useState<any>(null);
  const [selectedBotId, setSelectedBotId] = useState("");

  // Auto-select first bot when bots load
  if (bots.length > 0 && !selectedBotId) setSelectedBotId(bots[0].id);

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

  const syncing = syncAppMutation.isPending;

  function handleInstallConfirm() {
    if (!pendingApp || !selectedBotId) return;
    const appId = pendingApp.id || pendingApp.local_id;
    if (appId) {
      navigate(`/dashboard/accounts/${selectedBotId}/install/${appId}`);
      setPendingApp(null);
      return;
    }
    syncAppMutation.mutate(pendingApp.slug, {
      onSuccess: (synced: any) => {
        navigate(`/dashboard/accounts/${selectedBotId}/install/${synced.id}`);
        setPendingApp(null);
      },
      onError: (e) => toast({ variant: "destructive", title: "同步失败", description: e.message }),
    });
  }

  function filterApps(apps: any[]) {
    if (!search) return apps;
    const q = search.toLowerCase();
    return apps.filter(
      (a) => a.name?.toLowerCase().includes(q) || (a.slug || "").toLowerCase().includes(q),
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">应用市场</h1>
          <p className="text-sm text-muted-foreground mt-0.5">浏览和安装应用。</p>
        </div>
      </div>

      <div className="relative max-w-sm">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <Input
          placeholder="搜索应用..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="pl-9"
          aria-label="搜索应用"
        />
      </div>

      {loading ? (
        <div className="divide-y divide-border/50 rounded-xl border border-border/50 overflow-hidden">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="flex items-center gap-4 px-4 py-3.5 animate-pulse">
              <div className="h-9 w-9 rounded-lg bg-muted shrink-0" />
              <div className="flex-1 space-y-1.5">
                <div className="h-3.5 w-32 rounded bg-muted" />
                <div className="h-3 w-48 rounded bg-muted" />
              </div>
              <div className="h-7 w-14 rounded-md bg-muted shrink-0" />
            </div>
          ))}
        </div>
      ) : showTabs ? (
        <Tabs defaultValue="local">
          <TabsList>
            <TabsTrigger value="local">本站</TabsTrigger>
            {registryNames.map((name) => (
              <TabsTrigger key={name} value={name}>
                {name}
              </TabsTrigger>
            ))}
          </TabsList>
          <TabsContent value="local" className="mt-6">
            <AppGrid apps={filterApps(listedApps)} search={search} onInstall={setPendingApp} />
          </TabsContent>
          {registryNames.map((name) => (
            <TabsContent key={name} value={name} className="mt-6">
              <AppGrid
                apps={filterApps(registryGroups[name])}
                search={search}
                onInstall={setPendingApp}
              />
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
              <Select value={selectedBotId} onValueChange={setSelectedBotId}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="选择账号" />
                </SelectTrigger>
                <SelectContent>
                  {bots.map((b) => (
                    <SelectItem key={b.id} value={b.id}>
                      {botDisplayName(b)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Button className="w-full" disabled={syncing} onClick={handleInstallConfirm}>
                {syncing ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}继续安装
              </Button>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ==================== App Grid ====================

function AppGrid({
  apps,
  search,
  onInstall,
}: {
  apps: any[];
  search: string;
  onInstall: (app: any) => void;
}) {
  if (apps.length === 0) {
    return (
      <div className="text-center py-16 space-y-3 border-2 border-dashed rounded-xl">
        <Blocks className="w-8 h-8 mx-auto text-muted-foreground/40" />
        <p className="text-sm text-muted-foreground">{search ? "没有匹配的应用" : "暂无应用"}</p>
      </div>
    );
  }

  return (
    <div className="divide-y divide-border/50 rounded-xl border border-border/50 overflow-hidden">
      {apps.map((app) => (
        <div
          key={`${app.registry || "local"}-${app.slug || app.id}`}
          className="group flex items-center gap-4 px-4 py-3.5 bg-card hover:bg-muted/40 transition-colors"
        >
          <AppIcon icon={app.icon} iconUrl={app.icon_url} size="h-9 w-9" />
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <p className="text-sm font-semibold leading-tight">{app.name}</p>
              {app.version ? (
                <Badge variant="outline" className="text-[10px] font-mono shrink-0">
                  v{app.version}
                </Badge>
              ) : null}
              {app.installed ? (
                <Badge variant="secondary" className="text-[10px] shrink-0">
                  已安装
                </Badge>
              ) : null}
            </div>
            <p className="text-xs text-muted-foreground mt-0.5 line-clamp-1">
              {app.description || "暂无描述"}
            </p>
          </div>
          {app.author || app.owner_name ? (
            <span className="text-[11px] text-muted-foreground/50 shrink-0 hidden sm:block">
              {app.author || app.owner_name}
            </span>
          ) : null}
          {parseTools(app.tools).length > 0 ? (
            <span className="text-[11px] text-muted-foreground/50 shrink-0 hidden md:block">
              {parseTools(app.tools).length} 个命令
            </span>
          ) : null}
          {app.installed && app.update_available ? (
            <Button
              size="sm"
              variant="outline"
              className="shrink-0 gap-1.5"
              onClick={() => onInstall(app)}
            >
              <RefreshCw className="h-3.5 w-3.5" />
              更新
            </Button>
          ) : app.installed ? (
            <span className="text-[11px] text-muted-foreground/50 shrink-0">已安装</span>
          ) : (
            <Button
              size="sm"
              variant="outline"
              className="shrink-0 gap-1.5"
              onClick={() => onInstall(app)}
            >
              <Download className="h-3.5 w-3.5" />
              安装
            </Button>
          )}
        </div>
      ))}
    </div>
  );
}
