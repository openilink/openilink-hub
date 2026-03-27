import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Blocks, Loader2, Plus } from "lucide-react";
import { api } from "@/lib/api";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useToast } from "@/hooks/use-toast";
import { AppIcon } from "@/components/app-icon";
import { ListingBadge } from "@/components/listing-badge";

// ==================== Page ====================

export function DeveloperAppsPage() {
  const navigate = useNavigate();
  const { toast } = useToast();
  const [apps, setApps] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [newName, setNewName] = useState("");
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    setLoading(true);
    api.listApps()
      .then((list) => setApps(list || []))
      .catch(() => {
        toast({ variant: "destructive", title: "加载失败", description: "无法获取应用列表" });
      })
      .finally(() => setLoading(false));
  }, []);

  function openDialog() {
    setNewName("");
    setDialogOpen(true);
  }

  function handleDialogClose(open: boolean) {
    setDialogOpen(open);
    if (!open) setNewName("");
  }

  async function handleCreate() {
    const name = newName.trim();
    if (!name) return;
    setCreating(true);
    try {
      const app = await api.createApp({ name });
      setDialogOpen(false);
      setNewName("");
      navigate(`/dashboard/apps/${app.id}`);
    } catch (e: any) {
      toast({
        variant: "destructive",
        title: "创建失败",
        description: e.message,
      });
    } finally {
      setCreating(false);
    }
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (creating) return;
    if (e.key === "Enter") {
      handleCreate();
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">我的应用</h1>
          <p className="text-sm text-muted-foreground mt-0.5">管理你创建的应用。</p>
        </div>
        <Button onClick={openDialog} className="gap-1.5 shrink-0">
          <Plus className="h-4 w-4" />
          新建应用
        </Button>
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
              <div className="h-5 w-14 rounded-full bg-muted shrink-0" />
            </div>
          ))}
        </div>
      ) : apps.length === 0 ? (
        <div className="text-center py-16 space-y-3 border-2 border-dashed rounded-xl">
          <Blocks className="w-8 h-8 mx-auto text-muted-foreground/40" />
          <p className="text-sm text-muted-foreground">还没有创建过应用</p>
          <Button variant="outline" onClick={openDialog} className="gap-1.5">
            <Plus className="h-4 w-4" />
            新建第一个应用
          </Button>
        </div>
      ) : (
        <div className="divide-y divide-border/50 rounded-xl border border-border/50 overflow-hidden">
          {apps.map((app) => (
            <div
              key={app.id}
              role="button"
              tabIndex={0}
              className="group flex items-center gap-4 px-4 py-3.5 bg-card hover:bg-muted/40 transition-colors cursor-pointer"
              onClick={() => navigate(`/dashboard/apps/${app.id}`)}
              onKeyDown={(e) => (e.key === "Enter" || e.key === " ") && navigate(`/dashboard/apps/${app.id}`)}
            >
              <AppIcon icon={app.icon} iconUrl={app.icon_url} size="h-9 w-9" />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-semibold leading-tight">{app.name}</p>
                {app.slug ? (
                  <p className="font-mono text-xs text-muted-foreground mt-0.5">{app.slug}</p>
                ) : null}
              </div>
              <ListingBadge listing={app.listing} />
            </div>
          ))}
        </div>
      )}

      <Dialog open={dialogOpen} onOpenChange={handleDialogClose}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>新建应用</DialogTitle>
            <DialogDescription>输入应用名称以创建一个新应用。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 pt-2">
            <Input
              placeholder="应用名称"
              aria-label="应用名称"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              onKeyDown={handleKeyDown}
              disabled={creating}
              autoFocus
            />
            <Button
              className="w-full"
              disabled={creating || !newName.trim()}
              onClick={handleCreate}
            >
              {creating ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
              创建
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
