import { useEffect, useState } from "react";
import { useParams, useNavigate, useSearchParams } from "react-router-dom";
import { ArrowLeft, Eye, Zap, Loader2, ExternalLink, ShieldCheck, Terminal, Sliders } from "lucide-react";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Card, CardContent } from "../components/ui/card";
import { Skeleton } from "../components/ui/skeleton";
import { Label } from "../components/ui/label";
import { api, botDisplayName } from "../lib/api";
import { useApp } from "@/hooks/use-apps";
import { useBots } from "@/hooks/use-bots";
import { useToast } from "@/hooks/use-toast";
import { AppIcon } from "../components/app-icon";
import { SCOPE_DESCRIPTIONS, EVENT_TYPES } from "../lib/constants";
import { ToolsDisplay, parseTools } from "../components/tools-display";

type TabKey = "permissions" | "tools" | "config";

export function InstallAppPage() {
  const { id: botId, appId } = useParams<{ id: string; appId: string }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { toast } = useToast();
  const { data: app, isLoading: appLoading } = useApp(appId!);
  const { data: allBots = [] } = useBots();
  const bot = allBots.find((b: any) => b.id === botId);
  const botName = bot ? botDisplayName(bot) : "";
  const loading = appLoading;
  const [handle, setHandle] = useState(searchParams.get("handle") || "");
  const [configForm, setConfigForm] = useState<Record<string, string>>(() => {
    const prefill: Record<string, string> = {};
    searchParams.forEach((value, key) => {
      if (key.startsWith("config.")) {
        prefill[key.slice(7)] = value;
      }
    });
    return prefill;
  });
  const [installing, setInstalling] = useState(false);
  const [waitingForOAuth, setWaitingForOAuth] = useState(false);
  const [oauthPopup, setOAuthPopup] = useState<Window | null>(null);
  const [tab, setTab] = useState<TabKey>("permissions");

  // Listen for OAuth completion from popup
  useEffect(() => {
    if (!waitingForOAuth) return;

    const handleMessage = (event: MessageEvent) => {
      if (event.data?.type === "oauth_complete") {
        setWaitingForOAuth(false);
        if (oauthPopup && !oauthPopup.closed) oauthPopup.close();
        toast({ title: "安装成功" });
        navigate(`/dashboard/accounts/${botId}`);
      }
    };
    window.addEventListener("message", handleMessage);

    const interval = setInterval(async () => {
      try {
        const installations = await api.listBotApps(botId!);
        if (installations?.find((i: any) => i.app_id === appId)) {
          clearInterval(interval);
          setWaitingForOAuth(false);
          if (oauthPopup && !oauthPopup.closed) oauthPopup.close();
          toast({ title: "安装成功" });
          navigate(`/dashboard/accounts/${botId}`);
        }
      } catch {}
    }, 3000);

    return () => {
      window.removeEventListener("message", handleMessage);
      clearInterval(interval);
    };
  }, [waitingForOAuth, oauthPopup, botId, appId, navigate, toast]);

  async function handleInstall() {
    setInstalling(true);
    try {
      const result = await api.installApp(appId!, {
        bot_id: botId,
        handle: handle.trim(),
        scopes: app.scopes || [],
      });

      if (result?.needs_oauth && result?.oauth_redirect) {
        setWaitingForOAuth(true);
        const popup = window.open(result.oauth_redirect, "oauth_popup", "width=600,height=700,scrollbars=yes");
        setOAuthPopup(popup);
        setInstalling(false);
        return;
      }

      const installationId = result?.id || result?.installation_id;

      if (app.config_schema && installationId) {
        const hasConfig = Object.values(configForm).some((v) => v !== "");
        if (hasConfig) {
          try {
            await api.updateInstallation(appId!, installationId, {
              config: JSON.stringify(configForm),
            });
          } catch {
            toast({ title: "配置保存失败", description: "应用已安装，但配置未保存。" });
          }
        }
      }

      toast({ title: "安装成功" });
      navigate(`/dashboard/accounts/${botId}/apps/${installationId}`);
    } catch (e: any) {
      toast({ variant: "destructive", title: "安装失败", description: e.message });
    } finally {
      setInstalling(false);
    }
  }

  if (waitingForOAuth) {
    return (
      <div className="max-w-xl mx-auto py-20 text-center space-y-4">
        <Loader2 className="h-8 w-8 animate-spin mx-auto text-primary" />
        <h2 className="text-lg font-bold">等待授权完成</h2>
        <p className="text-sm text-muted-foreground">请在弹出窗口中完成应用授权。完成后此页面将自动更新。</p>
        <Button variant="outline" size="sm" onClick={() => {
          setWaitingForOAuth(false);
          if (oauthPopup && !oauthPopup.closed) oauthPopup.close();
        }}>
          取消
        </Button>
      </div>
    );
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-20 w-full rounded-3xl" />
        <Skeleton className="h-64 w-full rounded-3xl" />
      </div>
    );
  }

  if (!app) {
    return (
      <div className="py-20 text-center space-y-4">
        <p className="font-bold">未找到应用</p>
        <Button variant="link" onClick={() => navigate(`/dashboard/accounts/${botId}`)}>
          返回账号
        </Button>
      </div>
    );
  }

  // Parse scopes
  const scopes: string[] = app.scopes || [];
  const readScopes = scopes.filter((s: string) => s.endsWith(":read"));
  const writeScopes = scopes.filter((s: string) => s.endsWith(":write"));
  const otherScopes = scopes.filter((s: string) => !s.endsWith(":read") && !s.endsWith(":write"));
  const events: string[] = app.events || [];
  const hasPermissions = readScopes.length > 0 || writeScopes.length > 0 || otherScopes.length > 0 || events.length > 0;

  // Parse config_schema
  let schemaProperties: Record<string, any> = {};
  if (app.config_schema) {
    try {
      const parsed = typeof app.config_schema === "string"
        ? JSON.parse(app.config_schema)
        : app.config_schema;
      schemaProperties = parsed.properties || {};
    } catch {
      // ignore
    }
  }

  const tools = parseTools(app.tools);

  // Build tabs — permissions only shown when app has scopes or events
  const tabs: { key: TabKey; label: string; icon: any }[] = [];
  if (hasPermissions) {
    tabs.push({ key: "permissions", label: "权限", icon: ShieldCheck });
  }
  if (tools.length > 0) {
    tabs.push({ key: "tools", label: "命令 / 工具", icon: Terminal });
  }
  if (Object.keys(schemaProperties).length > 0) {
    tabs.push({ key: "config", label: "配置", icon: Sliders });
  }
  const activeTab = tabs.find((t) => t.key === tab) ? tab : tabs[0]?.key;
  const hasTabs = tabs.length > 0;

  return (
    <div className="space-y-6">
      {/* Back */}
      <button
        onClick={() => navigate(`/dashboard/accounts/${botId}`)}
        className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
      >
        <ArrowLeft className="h-4 w-4" />
        {botName || "返回"}
      </button>

      {/* Header: icon + info + handle + install */}
      <div className="flex items-start gap-4">
        <AppIcon icon={app.icon} iconUrl={app.icon_url} size="h-14 w-14" />
        <div className="flex-1 min-w-0 space-y-1">
          <div className="flex items-center gap-3 flex-wrap">
            <h1 className="text-2xl font-bold tracking-tight">{app.name}</h1>
            {app.slug && (
              <span className="text-sm text-muted-foreground font-mono">{app.slug}</span>
            )}
            {app.homepage && (
              <a
                href={app.homepage}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1.5 text-sm text-primary hover:underline"
              >
                <ExternalLink className="h-3.5 w-3.5" />
                主页
              </a>
            )}
          </div>
          {app.description && (
            <p className="text-sm text-muted-foreground">{app.description}</p>
          )}
          <div className="flex items-center gap-3 pt-2">
            <div className="flex items-center gap-2">
              <Input
                id="install-handle"
                value={handle}
                onChange={(e) => setHandle(e.target.value)}
                className="h-8 text-xs font-mono w-40"
                placeholder="如 notify-prod"
              />
              <span className="text-xs text-muted-foreground font-mono shrink-0">
                @{handle || "handle"}
              </span>
            </div>
            <Button size="sm" onClick={handleInstall} disabled={installing}>
              {installing && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
              允许并安装
            </Button>
          </div>
        </div>
      </div>

      {/* Mobile nav */}
      {hasTabs && tabs.length > 1 && (
        <div className="md:hidden">
          <select
            value={activeTab}
            onChange={(e) => setTab(e.target.value as TabKey)}
            className="w-full h-9 px-3 rounded-md border bg-background text-sm"
          >
            {tabs.map((t) => (
              <option key={t.key} value={t.key}>{t.label}</option>
            ))}
          </select>
        </div>
      )}

      {/* Desktop: Left nav + Right content */}
      {hasTabs && (
        <div className="flex gap-8">
          {tabs.length > 1 && (
            <nav className="hidden md:block w-48 shrink-0 space-y-1">
              {tabs.map((t) => (
                <Button
                  key={t.key}
                  variant="ghost"
                  size="sm"
                  onClick={() => setTab(t.key)}
                  className={`w-full justify-start gap-2 ${
                    activeTab === t.key
                      ? "bg-primary/10 text-primary font-medium hover:bg-primary/10 hover:text-primary"
                      : "text-muted-foreground"
                  }`}
                >
                  <t.icon className="h-4 w-4 shrink-0" />
                  {t.label}
                </Button>
              ))}
            </nav>
          )}

          <div className="flex-1 min-w-0">
            {activeTab === "permissions" && (
              <div className="space-y-6">
                <div>
                  <h2 className="text-base font-semibold">权限</h2>
                  <p className="text-sm text-muted-foreground mt-1">安装后此应用将获得以下权限。</p>
                </div>
                <Card>
                  <CardContent className="pt-6 space-y-4">
                    {readScopes.length > 0 && (
                      <div className="space-y-2">
                        <p className="text-xs font-bold uppercase tracking-wider text-muted-foreground">读取权限</p>
                        <div className="space-y-1.5">
                          {readScopes.map((scope: string) => (
                            <div key={scope} className="flex items-center gap-2 text-sm text-muted-foreground">
                              <Eye className="h-3.5 w-3.5 shrink-0" />
                              <span>{SCOPE_DESCRIPTIONS[scope] || scope}</span>
                              <span className="font-mono text-xs ml-auto text-muted-foreground/60">{scope}</span>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                    {writeScopes.length > 0 && (
                      <div className="space-y-2">
                        <p className="text-xs font-bold uppercase tracking-wider text-muted-foreground">写入权限</p>
                        <div className="space-y-1.5">
                          {writeScopes.map((scope: string) => (
                            <div key={scope} className="flex items-center gap-2 text-sm text-muted-foreground">
                              <Zap className="h-3.5 w-3.5 shrink-0" />
                              <span>{SCOPE_DESCRIPTIONS[scope] || scope}</span>
                              <span className="font-mono text-xs ml-auto text-muted-foreground/60">{scope}</span>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                    {otherScopes.length > 0 && (
                      <div className="space-y-2">
                        <p className="text-xs font-bold uppercase tracking-wider text-muted-foreground">其他权限</p>
                        <div className="space-y-1.5">
                          {otherScopes.map((scope: string) => (
                            <div key={scope} className="flex items-center gap-2 text-sm text-muted-foreground">
                              <ShieldCheck className="h-3.5 w-3.5 shrink-0" />
                              <span>{SCOPE_DESCRIPTIONS[scope] || scope}</span>
                              <span className="font-mono text-xs ml-auto text-muted-foreground/60">{scope}</span>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                    {events.length > 0 && (
                      <div className="space-y-2">
                        <p className="text-xs font-bold uppercase tracking-wider text-muted-foreground">订阅事件</p>
                        <div className="flex flex-wrap gap-1.5">
                          {events.map((event: string) => {
                            const found = EVENT_TYPES.find((e) => e.key === event);
                            return (
                              <Badge key={event} variant="outline" className="text-xs">
                                {found ? found.label : event}
                                {found && <span className="font-mono text-muted-foreground/60 ml-1">· {event}</span>}
                              </Badge>
                            );
                          })}
                        </div>
                      </div>
                    )}
                  </CardContent>
                </Card>
              </div>
            )}

            {activeTab === "tools" && tools.length > 0 && (
              <div className="space-y-6">
                <div>
                  <h2 className="text-base font-semibold">命令 / 工具</h2>
                  <p className="text-sm text-muted-foreground mt-1">此应用提供的命令和工具。</p>
                </div>
                <Card>
                  <CardContent className="pt-6">
                    <ToolsDisplay tools={tools} />
                  </CardContent>
                </Card>
              </div>
            )}

            {activeTab === "config" && Object.keys(schemaProperties).length > 0 && (
              <div className="space-y-6">
                <div>
                  <h2 className="text-base font-semibold">应用配置</h2>
                  <p className="text-sm text-muted-foreground mt-1">安装前填写此应用所需的配置。</p>
                </div>
                <Card>
                  <CardContent className="space-y-4 pt-6">
                    {Object.entries(schemaProperties).map(([key, prop]: [string, any]) => (
                      <div key={key} className="space-y-1.5">
                        <Label className="text-muted-foreground">{prop.title || key}</Label>
                        <Input
                          value={configForm[key] || ""}
                          onChange={(e) => setConfigForm({ ...configForm, [key]: e.target.value })}
                          className="h-8 text-xs font-mono"
                          placeholder={prop.description || ""}
                        />
                        {prop.description && (
                          <p className="text-xs text-muted-foreground">{prop.description}</p>
                        )}
                      </div>
                    ))}
                  </CardContent>
                </Card>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
