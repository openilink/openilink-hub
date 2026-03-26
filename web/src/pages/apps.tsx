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
import { SCOPE_DESCRIPTIONS } from "../lib/constants";
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
  return name.toLowerCase().replace(/[^a-z0-9\u4e00-\u9fff]+/g, "-").replace(/^-|-$/g, "").slice(0, 32);
}

function randomSuffix(): string {
  return Math.random().toString(36).slice(2, 8);
}

// ==================== Templates ====================

const TEMPLATES = [
  {
    id: "websocket-app",
    emoji: "\u{1F4E1}",
    name: "WebSocket App",
    description: "\u901A\u8FC7 WebSocket \u5B9E\u65F6\u6536\u53D1 Bot \u6D88\u606F",
    scopes: ["message:write", "message:read", "contact:read", "bot:read"],
    events: ["message"],
    readme: `## WebSocket App

\u8FDE\u63A5 WebSocket \u5B9E\u65F6\u6536\u53D1\u6D88\u606F\u3002

### \u8FDE\u63A5\u65B9\u5F0F

\`\`\`
wss://{hub_url}/bot/v1/ws?token={your_token}
\`\`\`

### \u53D1\u9001\u6D88\u606F

\u901A\u8FC7 WebSocket \u53D1\u9001\uFF1A
\`\`\`json
{"type":"send","to":"wxid_xxx","content":"hello"}
\`\`\`

\u6216\u901A\u8FC7 HTTP\uFF1A
\`\`\`bash
curl -X POST {hub_url}/bot/v1/message/send \\
  -H "Authorization: Bearer {your_token}" \\
  -d '{"to":"wxid_xxx","content":"hello"}'
\`\`\``,
  },
  {
    id: "webhook-app",
    emoji: "\u{1F517}",
    name: "Webhook App",
    description: "\u901A\u8FC7 HTTP API \u5411 Bot \u53D1\u9001\u6D88\u606F",
    scopes: ["message:write"],
    events: [],
    readme: `## Webhook App

\u901A\u8FC7 HTTP API \u53D1\u9001\u6D88\u606F\u3002

### \u53D1\u9001\u6D88\u606F

\`\`\`bash
curl -X POST {hub_url}/bot/v1/message/send \\
  -H "Authorization: Bearer {your_token}" \\
  -H "Content-Type: application/json" \\
  -d '{"to":"wxid_xxx","content":"hello"}'
\`\`\`

### \u53D1\u9001\u56FE\u7247

\`\`\`bash
curl -X POST {hub_url}/bot/v1/message/send \\
  -H "Authorization: Bearer {your_token}" \\
  -d '{"to":"wxid_xxx","type":"image","url":"https://example.com/img.png"}'
\`\`\``,
  },
  {
    id: "openclaw-channel",
    emoji: "\u{1F99E}",
    name: "OpenClaw Channel",
    description: "\u901A\u8FC7 OpenClaw \u534F\u8BAE\u63A5\u5165 Bot",
    scopes: ["message:write", "message:read", "contact:read", "bot:read"],
    events: ["message"],
    readme: `## OpenClaw Channel

\u901A\u8FC7 OpenClaw Channel Plugin \u63A5\u5165 Bot\u3002

### \u5B89\u88C5 Plugin

\u8BF7\u53C2\u8003 [OpenClaw Channel Plugin \u6587\u6863](https://github.com/nicepkg/openclaw) \u5B89\u88C5\u548C\u914D\u7F6E\u3002

### \u914D\u7F6E

\u5728 OpenClaw \u914D\u7F6E\u4E2D\u586B\u5165\u4EE5\u4E0B\u4FE1\u606F\uFF1A

- **Hub URL**: \`{hub_url}\`
- **Token**: \`{your_token}\``,
  },
];

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
            <h2 className="text-3xl font-bold tracking-tight">\u5E94\u7528</h2>
            <p className="text-muted-foreground">\u7BA1\u7406\u548C\u5B89\u88C5\u5E94\u7528\u3002</p>
          </div>
        </div>
      </div>

      <Tabs value={tab} onValueChange={(v) => navigate(`/dashboard/apps/${v}`)}>
        <TabsList>
          <TabsTrigger value="my">\u6211\u7684\u5E94\u7528</TabsTrigger>
          <TabsTrigger value="marketplace">\u5E94\u7528\u5E02\u573A</TabsTrigger>
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
  const [installTarget, setInstallTarget] = useState<{ type: "template"; template: typeof TEMPLATES[number] } | { type: "marketplace"; app: any } | null>(null);
  const [search, setSearch] = useState("");

  useEffect(() => {
    setLoading(true);
    api.getMarketplace().then(l => setMarketplaceApps(l || [])).catch(() => setMarketplaceApps([])).finally(() => setLoading(false));
  }, []);

  const filteredApps = marketplaceApps.filter(a =>
    !search || a.name?.toLowerCase().includes(search.toLowerCase()) || (a.slug || "").toLowerCase().includes(search.toLowerCase())
  );

  const filteredTemplates = TEMPLATES.filter(t =>
    !search || t.name.toLowerCase().includes(search.toLowerCase()) || t.description.includes(search)
  );

  return (
    <div className="space-y-8">
      <div className="relative max-w-md">
        <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
        <Input placeholder="\u641C\u7D22\u5E94\u7528..." value={search} onChange={e => setSearch(e.target.value)} className="pl-10 h-10 rounded-full bg-card shadow-sm border-border/50" aria-label="\u641C\u7D22\u5E94\u7528" />
      </div>

      {/* Quick Create Templates */}
      {filteredTemplates.length > 0 && (
        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-widest text-muted-foreground">\u5FEB\u901F\u521B\u5EFA</h3>
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
                    \u5B89\u88C5 <Download className="h-3 w-3" />
                  </Button>
                </CardFooter>
              </Card>
            ))}
          </div>
        </div>
      )}

      {/* Marketplace Apps */}
      <div className="space-y-4">
        <h3 className="text-sm font-bold uppercase tracking-widest text-muted-foreground">\u5E94\u7528\u5E02\u573A</h3>
        {loading ? (
          <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {[1, 2, 3].map(i => <Card key={i} className="h-48 animate-pulse bg-muted/20 rounded-3xl" />)}
          </div>
        ) : filteredApps.length === 0 ? (
          <div className="text-center py-16 space-y-3 border-2 border-dashed rounded-2xl">
            <Blocks className="w-10 h-10 mx-auto text-muted-foreground/40" />
            <p className="text-sm text-muted-foreground">{search ? "\u6CA1\u6709\u5339\u914D\u7684\u5E94\u7528" : "\u5E02\u573A\u6682\u65E0\u5E94\u7528"}</p>
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
                            \u5DF2\u5B89\u88C5
                          </Badge>
                        )}
                      </div>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="pb-6">
                  <p className="text-xs text-muted-foreground leading-relaxed line-clamp-2 min-h-[2.5rem]">
                    {app.description || "\u6682\u65E0\u63CF\u8FF0"}
                  </p>
                </CardContent>
                <CardFooter className="bg-muted/30 pt-4 flex justify-between items-center px-6">
                  <span className="text-[10px] font-bold text-muted-foreground">{app.author || app.slug}</span>
                  {app.installed && app.update_available ? (
                    <Button size="sm" variant="outline" onClick={() => setInstallTarget({ type: "marketplace", app })} className="h-8 rounded-full px-4 gap-1.5 font-bold text-xs">
                      \u66F4\u65B0 <RefreshCw className="h-3 w-3" />
                    </Button>
                  ) : app.installed ? (
                    <Badge variant="secondary" className="text-xs">\u5DF2\u5B89\u88C5</Badge>
                  ) : (
                    <Button size="sm" onClick={() => setInstallTarget({ type: "marketplace", app })} className="h-8 rounded-full px-4 gap-1.5 font-bold text-xs shadow-lg shadow-primary/10">
                      \u5B89\u88C5 <Download className="h-3 w-3" />
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
              <DialogTitle>\u5B89\u88C5\u5E94\u7528</DialogTitle>
              <DialogDescription>\u9009\u62E9\u8D26\u53F7\u5E76\u786E\u8BA4\u5B89\u88C5\u3002</DialogDescription>
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
  | { type: "template"; template: typeof TEMPLATES[number] }
  | { type: "marketplace"; app: any };

type InstallResult = {
  appId: string;
  appName: string;
  token?: string;
  kind?: string;
  templateId?: string;
  readme?: string;
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
        // Step 1: Create a new integration app
        const slug = `${target.template.id}-${randomSuffix()}`;
        const created = await api.createApp({
          name: target.template.name,
          slug,
          description: target.template.description,
          icon: target.template.emoji,
          kind: "integration",
          scopes: target.template.scopes,
          events: target.template.events,
          readme: target.template.readme,
        });
        // Step 2: Install to bot
        const installation = await api.installApp(created.id, {
          bot_id: botId,
          handle: handle.trim() || undefined,
          scopes: target.template.scopes,
        });
        setResult({
          appId: created.id,
          appName: target.template.name,
          token: installation.token,
          kind: "integration",
          templateId: target.template.id,
          readme: target.template.readme,
        });
      } else {
        // Marketplace app
        const app = target.app;
        let appId = app.local_id || app.id;
        if (!appId) {
          // Sync from marketplace first
          const synced = await api.syncMarketplaceApp(app.slug);
          appId = synced.id;
        }
        const installation = await api.installApp(appId, {
          bot_id: botId,
          handle: handle.trim() || undefined,
          scopes: app.scopes,
        });
        setResult({
          appId: appId,
          appName: app.name,
          token: installation.token,
          kind: app.kind,
        });
      }
      toast({ title: "\u5B89\u88C5\u6210\u529F", description: `\u5DF2\u5B89\u88C5 ${appName}\u3002` });
    } catch (e: any) {
      toast({ variant: "destructive", title: "\u5B89\u88C5\u5931\u8D25", description: e.message });
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
              <ExternalLink className="h-3 w-3" /> \u5E94\u7528\u4E3B\u9875
            </a>
          )}
        </div>

        {/* Right: Permissions + config */}
        <div className="sm:w-3/5 space-y-5">
          <div className="space-y-3">
            <h4 className="text-xs font-bold uppercase tracking-wider text-muted-foreground">\u6B64\u5E94\u7528\u5C06\u80FD\u591F\uFF1A</h4>

            {readScopes.length > 0 && (
              <div className="space-y-1.5">
                <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wide">\u67E5\u770B</p>
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
                <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wide">\u64CD\u4F5C</p>
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
                <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wide">\u4E8B\u4EF6\u8BA2\u9605</p>
                <div className="flex flex-wrap gap-1.5">
                  {events.map((e: string) => (
                    <Badge key={e} variant="outline" className="font-mono text-[10px]">{e}</Badge>
                  ))}
                </div>
              </div>
            )}

            {scopes.length === 0 && events.length === 0 && (
              <p className="text-sm text-muted-foreground">\u63A5\u6536 @mention \u6D88\u606F\u5E76\u6267\u884C\u54CD\u5E94\u3002</p>
            )}
          </div>

          <div className="space-y-3 pt-2 border-t">
            {bots.length === 0 ? (
              <p className="text-sm text-muted-foreground py-2">\u8BF7\u5148\u521B\u5EFA\u4E00\u4E2A\u8D26\u53F7\uFF0C\u7136\u540E\u518D\u5B89\u88C5\u5E94\u7528\u3002</p>
            ) : (
              <>
                <div className="space-y-1.5">
                  <label htmlFor="mp-install-bot" className="text-xs font-medium">\u5B89\u88C5\u5230\u8D26\u53F7</label>
                  <select id="mp-install-bot" value={botId} onChange={e => setBotId(e.target.value)}
                    className="w-full h-9 px-3 rounded-lg border bg-background text-sm outline-none focus:ring-2 focus:ring-primary/20">
                    {bots.map(b => <option key={b.id} value={b.id}>{b.name}</option>)}
                  </select>
                </div>
                <div className="space-y-1.5">
                  <label htmlFor="mp-install-handle" className="text-xs font-medium">Handle\uFF08\u53EF\u9009\uFF09</label>
                  <Input id="mp-install-handle" value={handle} onChange={e => setHandle(e.target.value)} className="h-9 font-mono" placeholder="\u5982 notify-prod" />
                  <p className="text-[10px] text-muted-foreground">\u7528\u6237\u53D1\u9001 @{handle || "handle"} \u89E6\u53D1\u6B64\u5E94\u7528</p>
                </div>
              </>
            )}
          </div>
        </div>
      </div>

      <div className="flex justify-end gap-2 pt-4 mt-4 border-t">
        <Button variant="ghost" onClick={onClose}>\u53D6\u6D88</Button>
        <Button onClick={handleInstall} disabled={saving || !botId} className="px-6">
          {saving && <Loader2 className="h-4 w-4 animate-spin mr-1.5" />}
          \u5141\u8BB8\u5E76\u5B89\u88C5
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

  const isIntegration = result.kind === "integration";

  return (
    <div className="py-2 space-y-6">
      <div className="text-center space-y-2">
        <div className="h-16 w-16 rounded-full bg-primary/10 flex items-center justify-center mx-auto">
          <Check className="h-8 w-8 text-primary" />
        </div>
        <h3 className="text-xl font-bold">\u5B89\u88C5\u6210\u529F</h3>
        <p className="text-sm text-muted-foreground">{result.appName} \u5DF2\u5B89\u88C5\u3002</p>
      </div>

      {isIntegration && result.token && (
        <div className="space-y-4">
          <div className="space-y-2">
            <label className="text-xs font-bold uppercase tracking-widest text-muted-foreground">Token</label>
            <div className="flex items-center gap-2 p-3 rounded-lg border bg-muted/30">
              <code className="text-sm font-mono flex-1 break-all">{result.token}</code>
              <button onClick={() => handleCopy(result.token!)} className="cursor-pointer text-muted-foreground hover:text-foreground shrink-0" aria-label="\u590D\u5236">
                {copied ? <Check className="w-4 h-4 text-primary" /> : <Copy className="w-4 h-4" />}
              </button>
            </div>
            <p className="text-[10px] text-destructive font-medium">\u8BF7\u5999\u5584\u4FDD\u7BA1\u6B64 Token\uFF0C\u5173\u95ED\u540E\u5C06\u65E0\u6CD5\u518D\u6B21\u67E5\u770B\u3002</p>
          </div>

          <div className="space-y-3">
            <details className="group">
              <summary className="text-sm font-medium cursor-pointer flex items-center gap-2 select-none">
                <ArrowRight className="h-3.5 w-3.5 transition-transform group-open:rotate-90" />
                HTTP \u53D1\u6D88\u606F
              </summary>
              <pre className="mt-2 p-3 rounded-lg bg-muted/30 border text-xs font-mono overflow-x-auto whitespace-pre-wrap">{`curl -X POST ${hubUrl}/bot/v1/message/send \\
  -H "Authorization: Bearer ${result.token}" \\
  -d '{"to":"wxid_xxx","content":"hello"}'`}</pre>
            </details>

            {(result.templateId === "websocket-app" || result.templateId === "openclaw-channel") && (
              <details className="group">
                <summary className="text-sm font-medium cursor-pointer flex items-center gap-2 select-none">
                  <ArrowRight className="h-3.5 w-3.5 transition-transform group-open:rotate-90" />
                  WebSocket \u8FDE\u63A5
                </summary>
                <pre className="mt-2 p-3 rounded-lg bg-muted/30 border text-xs font-mono overflow-x-auto whitespace-pre-wrap">{`wss://${hubUrl.replace(/^https?:\/\//, "")}/bot/v1/ws?token=${result.token}`}</pre>
              </details>
            )}
          </div>
        </div>
      )}

      {!isIntegration && (
        <div className="text-center">
          <p className="text-sm text-muted-foreground">\u5E94\u7528\u5DF2\u5B89\u88C5\u5230\u4F60\u7684\u8D26\u53F7\u3002</p>
        </div>
      )}

      <div className="flex justify-end gap-2 pt-2 border-t">
        <Button variant="outline" onClick={() => { onClose(); navigate(`/dashboard/apps/${result.appId}`); }}>
          \u67E5\u770B\u5E94\u7528\u8BE6\u60C5
        </Button>
        <Button onClick={onClose}>\u5B8C\u6210</Button>
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
      toast({ title: "\u521B\u5EFA\u6210\u529F", description: "\u5E94\u7528\u5DF2\u521B\u5EFA\u3002" });
      setIsCreating(false);
      load();
    } catch (e: any) {
      toast({ variant: "destructive", title: "\u521B\u5EFA\u5931\u8D25", description: e.message });
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
              <Plus className="h-4 w-4" /> \u521B\u5EFA\u5E94\u7528
            </Button>
          </DialogTrigger>
          <DialogContent className="rounded-[2rem]">
            <DialogHeader><DialogTitle className="text-2xl font-bold">\u521B\u5EFA\u5E94\u7528</DialogTitle><DialogDescription>\u586B\u5199\u57FA\u672C\u4FE1\u606F\u3002</DialogDescription></DialogHeader>
            <form onSubmit={handleCreate} className="space-y-5 pt-4">
               <div className="space-y-2"><label className="text-xs font-bold uppercase text-muted-foreground">\u540D\u79F0</label><Input placeholder="\u4F8B\u5982: \u901A\u77E5\u52A9\u624B" value={form.name} onChange={e => { const n = e.target.value; setForm({...form, name: n, slug: slugify(n)}); }} /></div>
               <div className="space-y-2"><label className="text-xs font-bold uppercase text-muted-foreground">\u552F\u4E00\u6807\u8BC6</label><Input value={form.slug} onChange={e => setForm({...form, slug: e.target.value})} className="font-mono" /></div>
               <div className="space-y-2"><label className="text-xs font-bold uppercase text-muted-foreground">\u63CF\u8FF0</label><Input placeholder="\u8FD9\u4E2A\u5E94\u7528\u662F\u7528\u6765..." value={form.description} onChange={e => setForm({...form, description: e.target.value})} /></div>
               <DialogFooter className="pt-4"><Button type="submit" className="w-full rounded-full h-11">\u521B\u5EFA</Button></DialogFooter>
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
              <Badge variant={app.status === "active" ? "default" : "secondary"} className="h-5 rounded-full text-[9px] px-2 font-bold">{app.status === "active" ? "\u5DF2\u53D1\u5E03" : "\u8349\u7A3F"}</Badge>
            </CardHeader>
            <CardFooter className="bg-muted/30 pt-3 flex justify-between items-center px-6">
               <span className="text-[10px] font-bold text-muted-foreground flex items-center gap-1.5"><Rocket className="h-3 w-3" /> {app.tools?.length || 0} \u4E2A\u5DE5\u5177\u5DF2\u914D\u7F6E</span>
               <ArrowRight className="h-4 w-4 text-muted-foreground group-hover:text-primary group-hover:translate-x-1 transition-all" />
            </CardFooter>
          </Card>
        ))}

        {apps.length === 0 && (
          <div className="col-span-full py-24 border-2 border-dashed rounded-[2rem] flex flex-col items-center justify-center text-center bg-muted/5">
            <div className="h-20 w-20 rounded-3xl bg-background border shadow-sm flex items-center justify-center mb-6">
              <Blocks className="h-10 w-10 text-primary/40" />
            </div>
            <h3 className="text-xl font-bold">\u8FD8\u6CA1\u6709\u5E94\u7528</h3>
            <p className="text-muted-foreground mt-2 max-w-sm">\u521B\u5EFA\u4F60\u7684\u7B2C\u4E00\u4E2A\u5E94\u7528\u3002</p>
            <Button variant="outline" className="mt-8 h-11 px-8 rounded-full" onClick={() => setIsCreating(true)}>\u521B\u5EFA\u5E94\u7528</Button>
          </div>
        )}
      </div>
    </div>
  );
}
