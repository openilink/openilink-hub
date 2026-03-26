import { useEffect, useState } from "react";
import { BarChart3, Users, Cpu, Globe, Blocks, Database, Settings, Trash2, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { api } from "@/lib/api";
import { useToast } from "@/hooks/use-toast";

function SkeletonCard() {
  return <Card className="h-24 animate-pulse bg-muted/20 border-none" />;
}

export function AdminOverviewPage() {
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [aiConfig, setAIConfig] = useState<any>(null);
  const [saving, setSaving] = useState(false);
  const { toast } = useToast();

  useEffect(() => {
    api.adminStats().then(setStats).finally(() => setLoading(false));
    api.getAIConfig().then(setAIConfig).catch(() => {});
  }, []);

  async function handleSaveAI() {
    setSaving(true);
    try {
      await api.setAIConfig(aiConfig);
      toast({ title: "全局 AI 配置已保存" });
    } catch (e: any) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
    setSaving(false);
  }

  return (
    <div className="space-y-8">
      <div className="flex items-center gap-4">
        <div className="h-12 w-12 rounded-2xl bg-primary/10 flex items-center justify-center text-primary shadow-sm border border-primary/20">
          <BarChart3 className="h-6 w-6" />
        </div>
        <div>
          <h2 className="text-3xl font-bold tracking-tight">系统概览</h2>
          <p className="text-muted-foreground">平台运行状态与配置。</p>
        </div>
      </div>

      {loading ? (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
          <SkeletonCard /><SkeletonCard /><SkeletonCard /><SkeletonCard />
        </div>
      ) : (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
          {[
            { label: "全站用户", value: stats?.total_users || 0, icon: Users, color: "text-blue-500" },
            { label: "微信账号", value: stats?.total_bots || 0, icon: Cpu, color: "text-green-500" },
            { label: "转发规则", value: stats?.total_channels || 0, icon: Globe, color: "text-purple-500" },
            { label: "活跃 App", value: stats?.total_apps || 0, icon: Blocks, color: "text-orange-500" },
          ].map((m, i) => (
            <Card key={i} className="border-border/50 bg-card/50">
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <CardTitle className="text-[10px] font-black uppercase tracking-widest text-muted-foreground">{m.label}</CardTitle>
                <m.icon className={`h-4 w-4 ${m.color}`} />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-black">{m.value}</div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Card className="border-border/50 bg-card/30 rounded-[2rem]">
        <CardHeader><CardTitle>系统状态</CardTitle><CardDescription></CardDescription></CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 md:grid-cols-3">
            <div className="p-4 rounded-2xl bg-muted/20 border border-border/50 flex items-center gap-4">
              <Database className="h-5 w-5 text-muted-foreground" />
              <div><p className="text-xs font-bold uppercase text-muted-foreground">PostgreSQL</p><p className="text-sm font-bold">已连接</p></div>
            </div>
            <div className="p-4 rounded-2xl bg-muted/20 border border-border/50 flex items-center gap-4">
              <Globe className="h-5 w-5 text-muted-foreground" />
              <div><p className="text-xs font-bold uppercase text-muted-foreground">WASM Runtime</p><p className="text-sm font-bold">就绪</p></div>
            </div>
          </div>
        </CardContent>
      </Card>

      <div className="grid gap-8 md:grid-cols-2">
        <Card className="border-border/50 bg-card/50 rounded-[2rem]">
          <CardHeader>
            <CardTitle>AI 配置</CardTitle>
            <CardDescription>所有账号的默认 AI 设置。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-1.5"><label className="text-xs font-bold uppercase text-muted-foreground">接口地址</label><Input value={aiConfig?.base_url || ""} onChange={e => setAIConfig({...aiConfig, base_url: e.target.value})} className="rounded-xl h-10" /></div>
            <div className="space-y-1.5"><label className="text-xs font-bold uppercase text-muted-foreground">默认模型</label><Input value={aiConfig?.model || ""} onChange={e => setAIConfig({...aiConfig, model: e.target.value})} className="rounded-xl h-10" /></div>
            <div className="space-y-1.5"><label className="text-xs font-bold uppercase text-muted-foreground">API Key</label><Input type="password" value={aiConfig?.api_key || ""} onChange={e => setAIConfig({...aiConfig, api_key: e.target.value})} className="rounded-xl h-10" placeholder="••••••••" /></div>
          </CardContent>
          <CardFooter className="bg-muted/30 pt-4 flex justify-end"><Button onClick={handleSaveAI} disabled={saving} className="rounded-full">保存</Button></CardFooter>
        </Card>

        <RegistryConfigCard />
      </div>
    </div>
  );
}

// ==================== Registry Config ====================

function RegistryConfigCard() {
  const [registryConfig, setRegistryConfig] = useState<any>(null);
  const [registries, setRegistries] = useState<any[]>([]);
  const [saving, setSaving] = useState(false);
  const [newName, setNewName] = useState("");
  const [newUrl, setNewUrl] = useState("");
  const [adding, setAdding] = useState(false);
  const { toast } = useToast();

  useEffect(() => {
    api.getRegistryConfig().then(setRegistryConfig).catch(() => setRegistryConfig({ enabled: "false" }));
    api.getRegistries().then(r => setRegistries(r || [])).catch(() => {});
  }, []);

  async function handleToggleExpose() {
    setSaving(true);
    try {
      const newEnabled = registryConfig?.enabled === "true" ? "false" : "true";
      await api.setRegistryConfig({ enabled: newEnabled });
      setRegistryConfig({ ...registryConfig, enabled: newEnabled });
      toast({ title: "Registry 配置已保存" });
    } catch (e: any) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
    setSaving(false);
  }

  async function handleAddRegistry() {
    if (!newName.trim() || !newUrl.trim()) return;
    setAdding(true);
    try {
      await api.createRegistry({ name: newName.trim(), url: newUrl.trim() });
      setNewName("");
      setNewUrl("");
      const r = await api.getRegistries();
      setRegistries(r || []);
      toast({ title: "Registry 已添加" });
    } catch (e: any) {
      toast({ variant: "destructive", title: "添加失败", description: e.message });
    }
    setAdding(false);
  }

  async function handleToggleRegistry(reg: any) {
    try {
      await api.updateRegistry(reg.id, { enabled: !reg.enabled });
      const r = await api.getRegistries();
      setRegistries(r || []);
    } catch (e: any) {
      toast({ variant: "destructive", title: "操作失败", description: e.message });
    }
  }

  async function handleDeleteRegistry(reg: any) {
    if (!confirm(`确定删除 Registry "${reg.name}"？`)) return;
    try {
      await api.deleteRegistry(reg.id);
      const r = await api.getRegistries();
      setRegistries(r || []);
      toast({ title: "已删除" });
    } catch (e: any) {
      toast({ variant: "destructive", title: "删除失败", description: e.message });
    }
  }

  return (
    <Card className="border-border/50 bg-card/50 rounded-[2rem]">
      <CardHeader>
        <CardTitle>Registry 配置</CardTitle>
        <CardDescription>管理应用市场 Registry 来源。</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Expose toggle */}
        <div className="flex items-center justify-between p-3 rounded-xl bg-muted/20 border border-border/50">
          <div>
            <p className="text-sm font-medium">对外暴露 Registry</p>
            <p className="text-xs text-muted-foreground">允许其他 Hub 从此实例拉取应用</p>
          </div>
          <Button
            variant={registryConfig?.enabled === "true" ? "default" : "outline"}
            size="sm"
            onClick={handleToggleExpose}
            disabled={saving}
          >
            {registryConfig?.enabled === "true" ? "已启用" : "已禁用"}
          </Button>
        </div>

        {/* Registry Sources */}
        <div className="space-y-2">
          <p className="text-xs font-bold uppercase tracking-widest text-muted-foreground">Registry 来源</p>
          {registries.length === 0 ? (
            <p className="text-xs text-muted-foreground py-2">暂无 Registry 来源</p>
          ) : (
            registries.map((reg) => (
              <div key={reg.id} className="flex items-center justify-between p-2.5 rounded-lg border bg-background">
                <div className="min-w-0">
                  <p className="text-sm font-medium truncate">{reg.name}</p>
                  <p className="text-xs text-muted-foreground font-mono truncate">{reg.url}</p>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <Button
                    variant={reg.enabled ? "default" : "outline"}
                    size="sm"
                    className="h-7 text-xs"
                    onClick={() => handleToggleRegistry(reg)}
                  >
                    {reg.enabled ? "启用" : "禁用"}
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-7 text-xs text-destructive"
                    onClick={() => handleDeleteRegistry(reg)}
                    aria-label="删除"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </Button>
                </div>
              </div>
            ))
          )}
        </div>

        {/* Add Registry */}
        <div className="space-y-2 pt-2 border-t">
          <p className="text-xs font-bold uppercase tracking-widest text-muted-foreground">添加 Registry</p>
          <div className="flex gap-2">
            <Input
              placeholder="名称"
              value={newName}
              onChange={e => setNewName(e.target.value)}
              className="rounded-xl h-9 flex-1"
            />
            <Input
              placeholder="URL"
              value={newUrl}
              onChange={e => setNewUrl(e.target.value)}
              className="rounded-xl h-9 flex-[2]"
            />
            <Button
              size="sm"
              onClick={handleAddRegistry}
              disabled={adding || !newName.trim() || !newUrl.trim()}
              className="h-9 rounded-xl"
            >
              <Plus className="w-3.5 h-3.5 mr-1" /> 添加
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
