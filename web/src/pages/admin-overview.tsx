import { useEffect, useState } from "react";
import {
  BarChart3,
  Users,
  Cpu,
  Globe,
  Blocks,
  Database,
  Settings,
  Trash2,
  Plus,
  UserPlus,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { api } from "@/lib/api";
import { useToast } from "@/hooks/use-toast";

const METRIC_CONFIG = [
  {
    label: "全站用户",
    key: "total_users",
    icon: Users,
    color: "text-blue-500",
    bg: "bg-blue-500/10",
  },
  {
    label: "微信账号",
    key: "total_bots",
    icon: Cpu,
    color: "text-emerald-500",
    bg: "bg-emerald-500/10",
  },
  {
    label: "已安装应用",
    key: "total_installations",
    icon: Globe,
    color: "text-violet-500",
    bg: "bg-violet-500/10",
  },
  {
    label: "活跃 App",
    key: "total_apps",
    icon: Blocks,
    color: "text-orange-500",
    bg: "bg-orange-500/10",
  },
];

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
    api
      .adminStats()
      .then(setStats)
      .finally(() => setLoading(false));
    api
      .getAIConfig()
      .then(setAIConfig)
      .catch(() => {});
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
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">系统概览</h1>
          <p className="text-sm text-muted-foreground mt-0.5">平台运行状态与配置。</p>
        </div>
      </div>

      {loading ? (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
          <SkeletonCard />
          <SkeletonCard />
          <SkeletonCard />
          <SkeletonCard />
        </div>
      ) : (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
          {METRIC_CONFIG.map((m) => (
            <Card
              key={m.label}
              className="border-border/50 bg-card/50 hover:bg-card transition-colors cursor-default"
            >
              <CardContent className="p-5">
                <div className="flex items-start justify-between mb-3">
                  <div
                    className={`h-8 w-8 rounded-lg ${m.bg} flex items-center justify-center ${m.color}`}
                  >
                    <m.icon className="h-4 w-4" />
                  </div>
                </div>
                <div className="text-2xl font-bold tabular-nums">{stats?.[m.key] || 0}</div>
                <p className="text-xs font-semibold text-foreground/80">{m.label}</p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Card className="border-border/50 bg-card/30">
        <CardHeader>
          <CardTitle>系统状态</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 md:grid-cols-3">
            <div className="p-4 rounded-2xl bg-muted/20 border border-border/50 flex items-center gap-4">
              <Database className="h-5 w-5 text-muted-foreground" />
              <div>
                <p className="text-xs font-bold uppercase text-muted-foreground">PostgreSQL</p>
                <p className="text-sm font-bold">已连接</p>
              </div>
            </div>
            <div className="p-4 rounded-2xl bg-muted/20 border border-border/50 flex items-center gap-4">
              <Globe className="h-5 w-5 text-muted-foreground" />
              <div>
                <p className="text-xs font-bold uppercase text-muted-foreground">WASM Runtime</p>
                <p className="text-sm font-bold">就绪</p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <RegistrationConfigCard />

      <div className="grid gap-8 md:grid-cols-2">
        <Card className="border-border/50 bg-card/50">
          <CardHeader>
            <CardTitle>AI 配置</CardTitle>
            <CardDescription>所有账号的默认 AI 设置。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-1.5">
              <Label className="text-xs font-bold uppercase text-muted-foreground">接口地址</Label>
              <Input
                value={aiConfig?.base_url || ""}
                onChange={(e) => setAIConfig({ ...aiConfig, base_url: e.target.value })}
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs font-bold uppercase text-muted-foreground">默认模型</Label>
              <Input
                value={aiConfig?.model || ""}
                onChange={(e) => setAIConfig({ ...aiConfig, model: e.target.value })}
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs font-bold uppercase text-muted-foreground">API Key</Label>
              <Input
                type="password"
                value={aiConfig?.api_key || ""}
                onChange={(e) => setAIConfig({ ...aiConfig, api_key: e.target.value })}
                placeholder="••••••••"
              />
            </div>
          </CardContent>
          <CardFooter className="flex justify-end">
            <Button onClick={handleSaveAI} disabled={saving}>
              保存
            </Button>
          </CardFooter>
        </Card>

        <RegistryConfigCard />
      </div>
    </div>
  );
}

// ==================== Registration Config ====================

function RegistrationConfigCard() {
  const [regConfig, setRegConfig] = useState<{ enabled: string } | null>(null);
  const [saving, setSaving] = useState(false);
  const { toast } = useToast();

  useEffect(() => {
    api
      .getRegistrationConfig()
      .then(setRegConfig)
      .catch(() => setRegConfig({ enabled: "true" }));
  }, []);

  async function handleToggle() {
    setSaving(true);
    try {
      const newEnabled = regConfig?.enabled === "true" ? "false" : "true";
      await api.setRegistrationConfig({ enabled: newEnabled });
      setRegConfig({ enabled: newEnabled });
      toast({ title: newEnabled === "true" ? "已开放注册" : "已关闭注册" });
    } catch (e: any) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
    setSaving(false);
  }

  return (
    <Card className="border-border/50 bg-card/30">
      <CardContent className="p-5">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className="h-10 w-10 rounded-xl bg-amber-500/10 flex items-center justify-center text-amber-500">
              <UserPlus className="h-5 w-5" />
            </div>
            <div>
              <p className="text-sm font-bold">开放注册</p>
              <p className="text-xs text-muted-foreground">
                关闭后，新用户无法通过密码注册、扫码登录或 OAuth 创建账号。管理员仍可手动创建用户。
              </p>
            </div>
          </div>
          <Button
            variant={regConfig?.enabled === "true" ? "default" : "outline"}
            size="sm"
            onClick={handleToggle}
            disabled={saving}
          >
            {regConfig?.enabled === "true" ? "已启用" : "已禁用"}
          </Button>
        </div>
      </CardContent>
    </Card>
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
    api
      .getRegistryConfig()
      .then(setRegistryConfig)
      .catch(() => setRegistryConfig({ enabled: "false" }));
    api
      .getRegistries()
      .then((r) => setRegistries(r || []))
      .catch(() => {});
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
    <Card className="border-border/50 bg-card/50">
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
          <p className="text-xs font-bold uppercase tracking-widest text-muted-foreground">
            Registry 来源
          </p>
          {registries.length === 0 ? (
            <p className="text-xs text-muted-foreground py-2">暂无 Registry 来源</p>
          ) : (
            registries.map((reg) => (
              <div
                key={reg.id}
                className="flex items-center justify-between p-2.5 rounded-lg border bg-background"
              >
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
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7 text-destructive hover:text-destructive"
                        onClick={() => handleDeleteRegistry(reg)}
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>删除</TooltipContent>
                  </Tooltip>
                </div>
              </div>
            ))
          )}
        </div>

        {/* Add Registry */}
        <div className="space-y-2 pt-2 border-t">
          <p className="text-xs font-bold uppercase tracking-widest text-muted-foreground">
            添加 Registry
          </p>
          <div className="flex gap-2">
            <Input
              placeholder="名称"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              className="flex-1"
            />
            <Input
              placeholder="URL"
              value={newUrl}
              onChange={(e) => setNewUrl(e.target.value)}
              className="flex-[2]"
            />
            <Button
              size="sm"
              onClick={handleAddRegistry}
              disabled={adding || !newName.trim() || !newUrl.trim()}
            >
              <Plus className="w-3.5 h-3.5 mr-1" /> 添加
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
