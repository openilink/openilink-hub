import { useMemo, useState } from "react";
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
  KeyRound,
  Loader2,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useToast } from "@/hooks/use-toast";
import { useConfirm } from "@/components/ui/confirm-dialog";
import {
  useAdminStats,
  useAIConfig,
  useSaveAIConfig,
  useRegistrationConfig,
  useSetRegistrationConfig,
  useOIDCConfig,
  useSetOIDCConfig,
  useDeleteOIDCConfig,
  useRegistryConfig,
  useSetRegistryConfig,
  useRegistries,
  useCreateRegistry,
  useUpdateRegistry,
  useDeleteRegistry,
} from "@/hooks/use-admin";

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
  const { data: stats, isLoading: loading } = useAdminStats();
  const { data: aiConfigData } = useAIConfig();
  const [aiConfig, setAIConfig] = useState<any>(null);
  const saveAIMutation = useSaveAIConfig();
  const { toast } = useToast();

  // Sync query data into local state for form editing
  const effectiveAIConfig = aiConfig ?? aiConfigData;

  const availableModelsText = useMemo(() => {
    try {
      return effectiveAIConfig?.available_models
        ? JSON.parse(effectiveAIConfig.available_models).join("\n")
        : "";
    } catch {
      return "";
    }
  }, [effectiveAIConfig?.available_models]);

  async function handleSaveAI() {
    if (!effectiveAIConfig) return;
    try {
      await saveAIMutation.mutateAsync(effectiveAIConfig);
      toast({ title: "全局 AI 配置已保存" });
    } catch (e: any) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
  }

  function updateAIConfig(patch: any) {
    setAIConfig((prev: any) => ({ ...(prev ?? aiConfigData), ...patch }));
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

      <OIDCConfigCard />

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
                value={effectiveAIConfig?.base_url || ""}
                onChange={(e) => updateAIConfig({ base_url: e.target.value })}
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs font-bold uppercase text-muted-foreground">默认模型</Label>
              <Input
                value={effectiveAIConfig?.model || ""}
                onChange={(e) => updateAIConfig({ model: e.target.value })}
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs font-bold uppercase text-muted-foreground">
                可用模型列表（每行一个）
              </Label>
              <Textarea
                value={availableModelsText}
                onChange={(e) => {
                  const models = e.target.value
                    .split("\n")
                    .map((s: string) => s.trim())
                    .filter(Boolean);
                  setAIConfig((prev: any) => ({
                    ...(prev ?? aiConfigData),
                    available_models: JSON.stringify(models),
                  }));
                }}
                rows={4}
                placeholder={"gpt-4o-mini\ngpt-4o\nclaude-3-5-sonnet-20241022"}
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs font-bold uppercase text-muted-foreground">API Key</Label>
              <Input
                type="password"
                value={effectiveAIConfig?.api_key || ""}
                onChange={(e) => updateAIConfig({ api_key: e.target.value })}
                placeholder="••••••••"
              />
            </div>
            <div className="flex items-center justify-between p-3 rounded-xl bg-muted/20 border border-border/50">
              <div>
                <p className="text-sm font-medium">隐藏思考过程</p>
                <p className="text-xs text-muted-foreground">启用后不会将模型的思考内容发送给用户</p>
              </div>
              <Switch
                checked={effectiveAIConfig?.hide_thinking === "true"}
                onCheckedChange={(checked) =>
                  updateAIConfig({ hide_thinking: checked ? "true" : "false" })
                }
              />
            </div>
            <div className="flex items-center justify-between p-3 rounded-xl bg-muted/20 border border-border/50">
              <div>
                <p className="text-sm font-medium">Markdown 转纯文本</p>
                <p className="text-xs text-muted-foreground">启用后将 AI 回复中的 Markdown 格式转为纯文本</p>
              </div>
              <Switch
                checked={effectiveAIConfig?.strip_markdown === "true"}
                onCheckedChange={(checked) =>
                  updateAIConfig({ strip_markdown: checked ? "true" : "false" })
                }
              />
            </div>
          </CardContent>
          <CardFooter className="flex justify-end">
            <Button onClick={handleSaveAI} disabled={saveAIMutation.isPending}>
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
  const { data: regConfig } = useRegistrationConfig();
  const setRegConfigMutation = useSetRegistrationConfig();
  const { toast } = useToast();

  async function handleToggle() {
    try {
      const newEnabled = regConfig?.enabled === "true" ? "false" : "true";
      await setRegConfigMutation.mutateAsync({ enabled: newEnabled });
      toast({ title: newEnabled === "true" ? "已开放注册" : "已关闭注册" });
    } catch (e: any) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
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
            disabled={setRegConfigMutation.isPending}
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
  const { data: registryConfig } = useRegistryConfig();
  const { data: registries = [] } = useRegistries();
  const setRegistryConfigMutation = useSetRegistryConfig();
  const createRegistryMutation = useCreateRegistry();
  const updateRegistryMutation = useUpdateRegistry();
  const deleteRegistryMutation = useDeleteRegistry();
  const [newName, setNewName] = useState("");
  const [newUrl, setNewUrl] = useState("");
  const { toast } = useToast();
  const { confirm, ConfirmDialog } = useConfirm();

  const adding = createRegistryMutation.isPending;

  async function handleToggleExpose() {
    try {
      const newEnabled = registryConfig?.enabled === "true" ? "false" : "true";
      await setRegistryConfigMutation.mutateAsync({ enabled: newEnabled });
      toast({ title: "Registry 配置已保存" });
    } catch (e: any) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
  }

  async function handleAddRegistry() {
    if (!newName.trim() || !newUrl.trim()) return;
    try {
      await createRegistryMutation.mutateAsync({ name: newName.trim(), url: newUrl.trim() });
      setNewName("");
      setNewUrl("");
      toast({ title: "Registry 已添加" });
    } catch (e: any) {
      toast({ variant: "destructive", title: "添加失败", description: e.message });
    }
  }

  async function handleImportDefault() {
    try {
      await createRegistryMutation.mutateAsync({ name: "OpeniLink Hub", url: "https://hub.openilink.com" });
      toast({ title: "已添加官方 Registry" });
    } catch (e: any) {
      toast({ variant: "destructive", title: "添加失败", description: e.message });
    }
  }

  async function handleToggleRegistry(reg: any) {
    try {
      await updateRegistryMutation.mutateAsync({ id: reg.id, data: { enabled: !reg.enabled } });
    } catch (e: any) {
      toast({ variant: "destructive", title: "操作失败", description: e.message });
    }
  }

  async function handleDeleteRegistry(reg: any) {
    const ok = await confirm({
      title: "删除确认",
      description: `确定删除 Registry "${reg.name}"？`,
      confirmText: "删除",
      variant: "destructive",
    });
    if (!ok) return;
    try {
      await deleteRegistryMutation.mutateAsync(reg.id);
      toast({ title: "已删除" });
    } catch (e: any) {
      toast({ variant: "destructive", title: "删除失败", description: e.message });
    }
  }

  return (
    <Card className="border-border/50 bg-card/50">
      {ConfirmDialog}
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
          <Switch
            checked={registryConfig?.enabled === "true"}
            onCheckedChange={handleToggleExpose}
            disabled={setRegistryConfigMutation.isPending}
          />
        </div>

        {/* Registry Sources */}
        <div className="space-y-2">
          <p className="text-xs font-bold uppercase tracking-widest text-muted-foreground">
            Registry 来源
          </p>
          {registries.length === 0 ? (
            <div className="flex items-center justify-between p-3 rounded-lg border border-dashed bg-muted/10">
              <p className="text-sm text-muted-foreground">暂无 Registry 来源</p>
              {window.location.origin !== "https://hub.openilink.com" && (
                <Button size="sm" onClick={handleImportDefault} disabled={adding}>
                  <Globe className="w-3.5 h-3.5 mr-1" /> 一键导入官方源
                </Button>
              )}
            </div>
          ) : (
            registries.map((reg: any) => (
              <div
                key={reg.id}
                className="flex items-center justify-between p-2.5 rounded-lg border bg-background"
              >
                <div className="min-w-0">
                  <p className="text-sm font-medium truncate">{reg.name}</p>
                  <p className="text-xs text-muted-foreground font-mono truncate">{reg.url}</p>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <Switch
                    checked={reg.enabled}
                    onCheckedChange={() => handleToggleRegistry(reg)}
                  />
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
              placeholder="https://hub.openilink.com"
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

function OIDCConfigCard() {
  const { data: providers = [] } = useOIDCConfig();
  const setOIDCMutation = useSetOIDCConfig();
  const deleteOIDCMutation = useDeleteOIDCConfig();
  const [slug, setSlug] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [issuerUrl, setIssuerUrl] = useState("");
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [scopes, setScopes] = useState("");
  const { toast } = useToast();
  const { confirm, ConfirmDialog } = useConfirm();

  const adding = setOIDCMutation.isPending;

  async function handleAdd() {
    const normalizedSlug = slug.trim();
    if (!normalizedSlug || !issuerUrl.trim() || !clientId.trim()) return;
    if (providers.some((p: any) => p.slug === normalizedSlug)) {
      toast({ variant: "destructive", title: "Slug 已存在", description: "请先删除再重新创建。" });
      return;
    }
    try {
      await setOIDCMutation.mutateAsync({
        slug: normalizedSlug,
        data: {
          display_name: displayName.trim() || normalizedSlug,
          issuer_url: issuerUrl.trim(),
          client_id: clientId.trim(),
          client_secret: clientSecret.trim(),
          scopes: scopes.trim(),
        },
      });
      setSlug("");
      setDisplayName("");
      setIssuerUrl("");
      setClientId("");
      setClientSecret("");
      setScopes("");
      toast({ title: "OIDC 提供商已添加" });
    } catch (e: any) {
      toast({ variant: "destructive", title: "添加失败", description: e.message });
    }
  }

  async function handleDelete(s: string, name: string) {
    const ok = await confirm({
      title: "删除确认",
      description: `确定删除 OIDC 提供商 "${name}"？`,
      confirmText: "删除",
      variant: "destructive",
    });
    if (!ok) return;
    try {
      await deleteOIDCMutation.mutateAsync(s);
      toast({ title: "已删除" });
    } catch (e: any) {
      toast({ variant: "destructive", title: "删除失败", description: e.message });
    }
  }

  return (
    <Card className="border-border/50 bg-card/50">
      {ConfirmDialog}
      <CardHeader>
        <CardTitle>OIDC 身份提供商</CardTitle>
        <CardDescription>
          添加自定义 OIDC 身份提供商（如 Pocket-ID、Keycloak、Authentik 等），用户可通过这些服务登录。
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Existing providers */}
        {providers.length > 0 && (
          <div className="space-y-2">
            {providers.map((p: any) => (
              <div
                key={p.slug}
                className="flex items-center justify-between p-3 rounded-lg border bg-background"
              >
                <div className="min-w-0 flex items-center gap-3">
                  <KeyRound className="h-4 w-4 text-muted-foreground shrink-0" />
                  <div className="min-w-0">
                    <p className="text-sm font-medium truncate">{p.display_name}</p>
                    <p className="text-xs text-muted-foreground font-mono truncate">{p.issuer_url}</p>
                  </div>
                </div>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 text-destructive hover:text-destructive shrink-0"
                      onClick={() => handleDelete(p.slug, p.display_name)}
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>删除</TooltipContent>
                </Tooltip>
              </div>
            ))}
          </div>
        )}

        {/* Add form */}
        <div className="space-y-3 pt-2 border-t">
          <p className="text-xs font-bold uppercase tracking-widest text-muted-foreground">
            添加 OIDC 提供商
          </p>
          <div className="grid gap-2 sm:grid-cols-2">
            <Input
              placeholder="标识 (slug, 如 pocket-id)"
              value={slug}
              onChange={(e) => setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))}
            />
            <Input
              placeholder="显示名称"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
            />
          </div>
          <Input
            placeholder="Issuer URL (如 https://auth.example.com)"
            value={issuerUrl}
            onChange={(e) => setIssuerUrl(e.target.value)}
          />
          <div className="grid gap-2 sm:grid-cols-2">
            <Input
              placeholder="Client ID"
              value={clientId}
              onChange={(e) => setClientId(e.target.value)}
            />
            <Input
              type="password"
              placeholder="Client Secret"
              value={clientSecret}
              onChange={(e) => setClientSecret(e.target.value)}
            />
          </div>
          <Input
            placeholder="Scopes (默认: openid profile email)"
            value={scopes}
            onChange={(e) => setScopes(e.target.value)}
          />
          <Button
            size="sm"
            onClick={handleAdd}
            disabled={adding || !slug.trim() || !issuerUrl.trim() || !clientId.trim()}
          >
            {adding ? <Loader2 className="w-3.5 h-3.5 mr-1 animate-spin" /> : <Plus className="w-3.5 h-3.5 mr-1" />}
            添加
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
