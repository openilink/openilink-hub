import { useEffect, useState } from "react";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Card, CardContent } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { api } from "../lib/api";
import { useToast } from "@/hooks/use-toast";
import {
  Blocks, Plus, CheckCircle, Trash2, Loader2,
} from "lucide-react";
import { AppIcon } from "../components/app-icon";
import {
  Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog";

export function BotAppsTab({ botId }: { botId: string }) {
  const [installations, setInstallations] = useState<any[]>([]);
  const [showInstall, setShowInstall] = useState(false);
  const { toast } = useToast();

  async function load() {
    try { setInstallations((await api.listBotApps(botId)) || []); } catch {}
  }

  useEffect(() => { load(); }, [botId]);

  async function handleUninstall(appId: string, instId: string) {
    if (!confirm("确定要卸载？")) return;
    try {
      await api.deleteInstallation(appId, instId);
      toast({ title: "已卸载" });
      load();
    } catch (e: any) {
      toast({ variant: "destructive", title: "卸载失败", description: e.message });
    }
  }

  async function handleToggle(inst: any) {
    try {
      await api.updateInstallation(inst.app_id, inst.id, { enabled: !inst.enabled });
      load();
    } catch {}
  }

  return (
    <div className="space-y-4 mt-4">
      <div className="flex items-center justify-between">
        <p className="text-sm font-medium">已安装的应用</p>
        <Button variant="outline" size="sm" onClick={() => setShowInstall(true)}>
          <Plus className="w-3.5 h-3.5 mr-1" /> 安装应用
        </Button>
      </div>

      {installations.length === 0 && (
        <div className="text-center py-12 space-y-3">
          <Blocks className="w-10 h-10 mx-auto text-muted-foreground/50" />
          <p className="text-sm text-muted-foreground">暂无安装的应用</p>
          <Button variant="outline" size="sm" onClick={() => setShowInstall(true)}>
            浏览应用
          </Button>
        </div>
      )}

      <div className="space-y-2">
        {installations.map((inst) => (
          <Card key={inst.id}>
            <CardContent className="py-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3 min-w-0">
                  <AppIcon icon={inst.app_icon} iconUrl={inst.app_icon_url} size="h-8 w-8" />
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">{inst.app_name}</span>
                      {inst.handle && <Badge variant="outline" className="text-xs font-mono">@{inst.handle}</Badge>}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Button variant="outline" size="sm" className="h-7 text-xs" onClick={() => handleToggle(inst)}>
                    {inst.enabled ? "停用" : "启用"}
                  </Button>
                  <Button variant="ghost" size="sm" className="h-7 text-xs text-destructive" onClick={() => handleUninstall(inst.app_id, inst.id)}>
                    <Trash2 className="w-3 h-3" />
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <InstallDialog botId={botId} open={showInstall} onOpenChange={setShowInstall} onInstalled={load} />
    </div>
  );
}

// ==================== Unified Install Dialog ====================

function InstallDialog({ botId, open, onOpenChange, onInstalled }: {
  botId: string; open: boolean; onOpenChange: (o: boolean) => void; onInstalled: () => void;
}) {
  const [apps, setApps] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [confirmApp, setConfirmApp] = useState<any>(null);
  const [handle, setHandle] = useState("");
  const [installing, setInstalling] = useState(false);
  const [error, setError] = useState("");
  const { toast } = useToast();

  useEffect(() => {
    if (!open) { setConfirmApp(null); return; }
    setLoading(true);
    Promise.all([api.listApps(), api.listApps({ listed: true })]).then(([my, listed]) => {
      const seen = new Set<string>();
      const merged: any[] = [];
      for (const a of [...(my || []), ...(listed || [])]) {
        if (!seen.has(a.id)) { seen.add(a.id); merged.push(a); }
      }
      setApps(merged);
    }).finally(() => setLoading(false));
  }, [open]);

  async function doInstall() {
    if (!confirmApp) return;
    setInstalling(true);
    setError("");
    try {
      await api.installApp(confirmApp.id, { bot_id: botId, handle: handle.trim() || undefined });
      toast({ title: "安装成功", description: `已安装 ${confirmApp.name}。` });
      onOpenChange(false);
      onInstalled();
    } catch (e: any) {
      setError(e.message);
    }
    setInstalling(false);
  }

  // Step 2: Confirm permissions + handle
  if (confirmApp) {
    const tools = (confirmApp.tools || []) as any[];
    const events = (confirmApp.events || []) as string[];
    const scopes = (confirmApp.scopes || []) as string[];

    return (
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <div className="flex items-center gap-3">
              {confirmApp.icon && <span className="text-lg">{confirmApp.icon}</span>}
              <div>
                <DialogTitle>{confirmApp.name}</DialogTitle>
                <DialogDescription>{confirmApp.description}</DialogDescription>
              </div>
            </div>
          </DialogHeader>

          <div className="space-y-4 py-2">
            {tools.length > 0 && (
              <div>
                <p className="text-xs font-medium mb-1">命令</p>
                <div className="flex flex-wrap gap-1">
                  {tools.map((t: any, i: number) => (
                    <Badge key={i} variant="outline" className="text-[10px] font-mono">
                      {t.command ? `/${t.command}` : t.name}
                    </Badge>
                  ))}
                </div>
              </div>
            )}
            {events.length > 0 && (
              <div>
                <p className="text-xs font-medium mb-1">事件订阅</p>
                <div className="flex flex-wrap gap-1">
                  {events.map((e) => <Badge key={e} variant="secondary" className="text-[10px] font-mono">{e}</Badge>)}
                </div>
              </div>
            )}
            {scopes.length > 0 && (
              <div>
                <p className="text-xs font-medium mb-1">权限</p>
                <div className="space-y-0.5">
                  {scopes.map((s) => (
                    <div key={s} className="flex items-center gap-2 text-xs">
                      <CheckCircle className="w-3 h-3 text-primary shrink-0" />
                      <span className="font-mono">{s}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            <div className="space-y-1">
              <label className="text-xs font-medium">Handle（必填）</label>
              <Input value={handle} onChange={(e) => setHandle(e.target.value)} placeholder="如 notify-prod" className="h-8 text-xs font-mono" />
              <p className="text-[10px] text-muted-foreground">用户发送 @{handle || "handle"} 触发此 App</p>
            </div>

            {error && <p className="text-xs text-destructive">{error}</p>}
          </div>

          <DialogFooter>
            <Button variant="ghost" onClick={() => setConfirmApp(null)}>返回</Button>
            <Button onClick={doInstall} disabled={installing || !handle.trim()}>
              {installing && <Loader2 className="h-4 w-4 animate-spin mr-1" />}
              授权并安装
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  }

  // Step 1: Pick an app
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>安装应用</DialogTitle>
          <DialogDescription>选择要安装的应用。</DialogDescription>
        </DialogHeader>

        {loading ? (
          <div className="flex justify-center py-8"><Loader2 className="h-6 w-6 animate-spin text-muted-foreground" /></div>
        ) : apps.length === 0 ? (
          <div className="text-center py-8 space-y-2">
            <p className="text-xs text-muted-foreground">没有可用的应用</p>
          </div>
        ) : (
          <div className="space-y-1 max-h-64 overflow-y-auto">
            {apps.map((app) => (
              <div key={app.id} className="flex items-center justify-between p-2 rounded-lg border bg-background">
                <div className="flex items-center gap-2 min-w-0">
                  {app.icon && <span>{app.icon}</span>}
                  <div className="min-w-0">
                    <div className="flex items-center gap-1.5">
                      <span className="text-xs font-medium">{app.name}</span>
                      <span className="text-xs text-muted-foreground font-mono">{app.slug}</span>
                    </div>
                    {app.description && <p className="text-xs text-muted-foreground truncate">{app.description}</p>}
                  </div>
                </div>
                <Button size="sm" variant="outline" onClick={() => { setConfirmApp(app); setHandle(app.slug || ""); setError(""); }}>
                  安装
                </Button>
              </div>
            ))}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
