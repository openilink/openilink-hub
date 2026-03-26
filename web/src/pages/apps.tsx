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
  Copy,
  Check,
  RefreshCw,
} from "lucide-react";
import { api } from "../lib/api";
import { SCOPE_DESCRIPTIONS, APP_TEMPLATES } from "../lib/constants";
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

function randomSuffix(): string {
  return Math.random().toString(36).slice(2, 8);
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
  const [marketplaceApps, setMarketplaceApps] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [installTarget, setInstallTarget] = useState<{ type: "template"; template: typeof APP_TEMPLATES[number] } | { type: "marketplace"; app: any } | null>(null);
  const [search, setSearch] = useState("");

  useEffect(() => {
    setLoading(true);
    api.getMarketplace().then(l => setMarketplaceApps(l || [])).catch(() => setMarketplaceApps([])).finally(() => setLoading(false));
  }, []);

  const filteredApps = marketplaceApps.filter(a =>
    !search || a.name?.toLowerCase().includes(search.toLowerCase()) || (a.slug || "").toLowerCase().includes(search.toLowerCase())
  );

  const filteredTemplates = APP_TEMPLATES.filter(t =>
    !search || t.name.toLowerCase().includes(search.toLowerCase()) || t.description.includes(search)
  );

  return (
    <div className="space-y-8">
      <div className="relative max-w-md">
        <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
        <Input placeholder="搜索应用..." value={search} onChange={e => setSearch(e.target.value)} className="pl-10 h-10 rounded-full bg-card shadow-sm border-border/50" aria-label="搜索应用" />
      </div>

      {/* Quick Create Templates */}
      {filteredTemplates.length > 0 && (
        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-widest text-muted-foreground">快速创建</h3>
          <div className="grid gap-4 md:grid-cols-3">
            {filteredTemplates.map((tpl) => (
              <Card key={tpl.id} className="group relative overflow-hidden rounded-2xl border-border/50 bg-card/50 transition-all hover:shadow-xl hover:-translate-y-0.5">
                <CardHeader className="pb-3">
                  <div className="flex items-center gap-3">
                    <div className="h-10 w-10 rounded-xl bg-muted flex items-center justify-center text-xl border">
                      {tpl.emoji}
                    </div>
                    <CardTitle className="text-base font-bold group-hover:text-primary transition-colors">{tpl.name}</CardTitle>
                  </div>
                </CardHeader>
                <CardContent className="pb-4">
                  <p className="text-xs text-muted-foreground leading-relaxed">{tpl.description}</p>
                </CardContent>
                <CardFooter className="bg-muted/30 pt-3 flex justify-end px-6">
                  <Button size="sm" variant="outline" onClick={() => setInstallTarget({ type: "template", template: tpl })} className="h-8 rounded-full px-4 gap-1.5 font-bold text-xs">
                    安装 <Download className="h-3 w-3" />
                  </Button>
                </CardFooter>
              </Card>
            ))}
          </div>
        </div>
      )}

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
                    <Button size="sm" variant="outline" onClick={() => setInstallTarget({ type: "marketplace", app })} className="h-8 rounded-full px-4 gap-1.5 font-bold text-xs">
                      更新 <RefreshCw className="h-3 w-3" />
                    </Button>
                  ) : app.installed ? (
                    <Badge variant="secondary" className="text-xs">已安装</Badge>
                  ) : (
                    <Button size="sm" onClick={() => setInstallTarget({ type: "marketplace", app })} className="h-8 rounded-full px-4 gap-1.5 font-bold text-xs shadow-lg shadow-primary/10">
                      安装 <Download className="h-3 w-3" />
                    </Button>
                  )}
                </CardFooter>
              </Card>
            ))}
          </div>
        )}
      </div>

      {installTarget && (
        <Dialog open={!!installTarget} onOpenChange={(o: boolean) => !o && setInstallTarget(null)}>
          <DialogContent className="sm:max-w-2xl rounded-[2rem]">
            <DialogHeader className="sr-only">
              <DialogTitle>安装应用</DialogTitle>
              <DialogDescription>选择账号并确认安装。</DialogDescription>
            </DialogHeader>
            <InstallFlowDialog target={installTarget} onClose={() => setInstallTarget(null)} />
          </DialogContent>
        </Dialog>
      )}
    </div>
  );
}

// ==================== Install Flow Dialog ====================

type InstallTarget =
  | { type: "template"; template: typeof APP_TEMPLATES[number] }
  | { type: "marketplace"; app: any };

type InstallResult = {
  appId: string;
  appName: string;
  token?: string;
  registry?: string;
  templateId?: string;
  guide?: string;
};

function InstallFlowDialog({ target, onClose }: { target: InstallTarget; onClose: () => void }) {
  const [bots, setBots] = useState<any[]>([]);
  const [botId, setBotId] = useState("");
  const [handle, setHandle] = useState("");
  const [saving, setSaving] = useState(false);
  const [result, setResult] = useState<InstallResult | null>(null);
  const { toast } = useToast();

  const isTemplate = target.type === "template";
  const appName = isTemplate ? target.template.name : target.app.name;
  const appDescription = isTemplate ? target.template.description : target.app.description;
  const appEmoji = isTemplate ? target.template.emoji : undefined;
  const appIcon = isTemplate ? undefined : target.app.icon;
  const appIconUrl = isTemplate ? undefined : target.app.icon_url;
  const scopes = isTemplate ? target.template.scopes : (target.app.scopes || []);
  const events = isTemplate ? target.template.events : (target.app.events || []);
  const readScopes = scopes.filter((s: string) => s.includes("read"));
  const writeScopes = scopes.filter((s: string) => !s.includes("read"));

  useEffect(() => {
    api.listBots().then(l => {
      const items = l || []; setBots(items);
      if (items.length) setBotId(items[0].id);
    });
  }, []);

  useEffect(() => {
    if (isTemplate) {
      setHandle(target.template.id);
    } else {
      setHandle(target.app.slug || "");
    }
  }, [target]);

  async function handleInstall() {
    if (!botId) return;
    setSaving(true);
    try {
      if (isTemplate) {
        const installation = await api.unifiedInstall(botId, {
          template_slug: target.template.id,
          name: target.template.name,
          description: target.template.description,
          icon: target.template.emoji,
          scopes: target.template.scopes,
          events: target.template.events,
          readme: target.template.readme,
          guide: target.template.guide,
          handle: handle.trim() || undefined,
        });
        setResult({
          appId: installation.app_id,
          appName: target.template.name,
          token: installation.app_token,
          registry: "builtin",
          templateId: target.template.id,
          guide: target.template.guide,
        });
      } else {
        const app = target.app;
        const installation = await api.unifiedInstall(botId, {
          marketplace_slug: app.slug,
          handle: handle.trim() || undefined,
          scopes: app.scopes,
        });
        setResult({
          appId: installation.app_id,
          appName: app.name,
          token: installation.app_token,
          registry: app.registry,
        });
      }
      toast({ title: "安装成功", description: `已安装 ${appName}。` });
    } catch (e: any) {
      toast({ variant: "destructive", title: "安装失败", description: e.message });
    }
    setSaving(false);
  }

  // ---- Result screen ----
  if (result) {
    return <InstallResultScreen result={result} onClose={onClose} />;
  }

  // ---- Install form ----
  return (
    <div className="py-2">
      <div className="flex flex-col sm:flex-row gap-6">
        {/* Left: App identity */}
        <div className="sm:w-2/5 space-y-4 sm:border-r sm:pr-6">
          <div className="flex items-center gap-3">
            {appEmoji ? (
              <div className="h-14 w-14 rounded-xl bg-muted flex items-center justify-center text-2xl border">{appEmoji}</div>
            ) : (
              <AppIcon icon={appIcon} iconUrl={appIconUrl} size="h-14 w-14" />
            )}
            <div>
              <h3 className="text-lg font-bold">{appName}</h3>
              {!isTemplate && target.app.slug && (
                <p className="text-xs text-muted-foreground font-mono">{target.app.slug}</p>
              )}
            </div>
          </div>
          {appDescription && (
            <p className="text-sm text-muted-foreground leading-relaxed">{appDescription}</p>
          )}
          {!isTemplate && target.app.homepage && (
            <a href={target.app.homepage} target="_blank" rel="noopener noreferrer" className="text-xs text-primary hover:underline flex items-center gap-1">
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
                {readScopes.map((s: string) => (
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
                {writeScopes.map((s: string) => (
                  <div key={s} className="flex items-start gap-2 text-sm">
                    <Zap className="h-3.5 w-3.5 mt-0.5 text-primary shrink-0" />
                    <span>{SCOPE_DESCRIPTIONS[s] || s}</span>
                  </div>
                ))}
              </div>
            )}

            {events.length > 0 && (
              <div className="space-y-1.5">
                <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wide">事件订阅</p>
                <div className="flex flex-wrap gap-1.5">
                  {events.map((e: string) => (
                    <Badge key={e} variant="outline" className="font-mono text-[10px]">{e}</Badge>
                  ))}
                </div>
              </div>
            )}

            {scopes.length === 0 && events.length === 0 && (
              <p className="text-sm text-muted-foreground">接收 @mention 消息并执行响应。</p>
            )}
          </div>

          <div className="space-y-3 pt-2 border-t">
            {bots.length === 0 ? (
              <p className="text-sm text-muted-foreground py-2">请先创建一个账号，然后再安装应用。</p>
            ) : (
              <>
                <div className="space-y-1.5">
                  <label htmlFor="mp-install-bot" className="text-xs font-medium">安装到账号</label>
                  <select id="mp-install-bot" value={botId} onChange={e => setBotId(e.target.value)}
                    className="w-full h-9 px-3 rounded-lg border bg-background text-sm outline-none focus:ring-2 focus:ring-primary/20">
                    {bots.map(b => <option key={b.id} value={b.id}>{b.name}</option>)}
                  </select>
                </div>
                <div className="space-y-1.5">
                  <label htmlFor="mp-install-handle" className="text-xs font-medium">Handle（可选）</label>
                  <Input id="mp-install-handle" value={handle} onChange={e => setHandle(e.target.value)} className="h-9 font-mono" placeholder="如 notify-prod" />
                  <p className="text-[10px] text-muted-foreground">用户发送 @{handle || "handle"} 触发此应用</p>
                </div>
              </>
            )}
          </div>
        </div>
      </div>

      <div className="flex justify-end gap-2 pt-4 mt-4 border-t">
        <Button variant="ghost" onClick={onClose}>取消</Button>
        <Button onClick={handleInstall} disabled={saving || !botId} className="px-6">
          {saving && <Loader2 className="h-4 w-4 animate-spin mr-1.5" />}
          允许并安装
        </Button>
      </div>
    </div>
  );
}

// ==================== Install Result Screen ====================

function InstallResultScreen({ result, onClose }: { result: InstallResult; onClose: () => void }) {
  const navigate = useNavigate();
  const [copied, setCopied] = useState(false);
  const hubUrl = window.location.origin;

  function handleCopy(text: string) {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  const isIntegration = result.registry === "builtin";

  return (
    <div className="py-2 space-y-6">
      <div className="text-center space-y-2">
        <div className="h-16 w-16 rounded-full bg-primary/10 flex items-center justify-center mx-auto">
          <Check className="h-8 w-8 text-primary" />
        </div>
        <h3 className="text-xl font-bold">安装成功</h3>
        <p className="text-sm text-muted-foreground">{result.appName} 已安装。</p>
      </div>

      {isIntegration && result.token && (
        <div className="space-y-4">
          <div className="space-y-2">
            <label className="text-xs font-bold uppercase tracking-widest text-muted-foreground">Token</label>
            <div className="flex items-center gap-2 p-3 rounded-lg border bg-muted/30">
              <code className="text-sm font-mono flex-1 break-all">{result.token}</code>
              <button onClick={() => handleCopy(result.token!)} className="cursor-pointer text-muted-foreground hover:text-foreground shrink-0" aria-label="复制">
                {copied ? <Check className="w-4 h-4 text-primary" /> : <Copy className="w-4 h-4" />}
              </button>
            </div>
            <p className="text-[10px] text-destructive font-medium">请妙善保管此 Token，关闭后将无法再次查看。</p>
          </div>

          <div className="space-y-3">
            <details className="group">
              <summary className="text-sm font-medium cursor-pointer flex items-center gap-2 select-none">
                <ArrowRight className="h-3.5 w-3.5 transition-transform group-open:rotate-90" />
                HTTP 发消息
              </summary>
              <pre className="mt-2 p-3 rounded-lg bg-muted/30 border text-xs font-mono overflow-x-auto whitespace-pre-wrap">{`curl -X POST ${hubUrl}/bot/v1/message/send \\
  -H "Authorization: Bearer ${result.token}" \\
  -d '{"to":"wxid_xxx","content":"hello"}'`}</pre>
            </details>

            {(result.templateId === "websocket-app" || result.templateId === "openclaw-channel") && (
              <details className="group">
                <summary className="text-sm font-medium cursor-pointer flex items-center gap-2 select-none">
                  <ArrowRight className="h-3.5 w-3.5 transition-transform group-open:rotate-90" />
                  WebSocket 连接
                </summary>
                <pre className="mt-2 p-3 rounded-lg bg-muted/30 border text-xs font-mono overflow-x-auto whitespace-pre-wrap">{`wss://${hubUrl.replace(/^https?:\/\//, "")}/bot/v1/ws?token=${result.token}`}</pre>
              </details>
            )}
          </div>
        </div>
      )}

      {!isIntegration && (
        <div className="text-center">
          <p className="text-sm text-muted-foreground">应用已安装到你的账号。</p>
        </div>
      )}

      <div className="flex justify-end gap-2 pt-2 border-t">
        <Button variant="outline" onClick={() => { onClose(); navigate(`/dashboard/apps/${result.appId}`); }}>
          查看应用详情
        </Button>
        <Button onClick={onClose}>完成</Button>
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
