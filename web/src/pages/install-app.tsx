import { useEffect, useState, useCallback } from "react";
import { useParams, useNavigate, useSearchParams } from "react-router-dom";
import { ArrowLeft, Eye, Zap, Loader2, ExternalLink } from "lucide-react";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Card, CardHeader, CardTitle, CardContent } from "../components/ui/card";
import { Skeleton } from "../components/ui/skeleton";
import { Label } from "../components/ui/label";
import { api } from "../lib/api";
import { useToast } from "@/hooks/use-toast";
import { AppIcon } from "../components/app-icon";
import { SCOPE_DESCRIPTIONS } from "../lib/constants";
import { ToolsDisplay, parseTools } from "../components/tools-display";

export function InstallAppPage() {
  const { id: botId, appId } = useParams<{ id: string; appId: string }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { toast } = useToast();
  const [app, setApp] = useState<any>(null);
  const [botName, setBotName] = useState("");
  const [loading, setLoading] = useState(true);
  const [handle, setHandle] = useState(searchParams.get("handle") || "");
  const [configForm, setConfigForm] = useState<Record<string, string>>(() => {
    // Prefill config from URL search params (e.g. ?config.forward_url=https://...)
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

  const loadData = useCallback(async () => {
    try {
      const [appData, bots] = await Promise.all([
        api.getApp(appId!),
        api.listBots(),
      ]);
      setApp(appData);
      const bot = (bots || []).find((b: any) => b.id === botId);
      if (bot) setBotName(bot.display_name || bot.name);
    } catch (e: any) {
      toast({ variant: "destructive", title: "加载失败", description: e.message });
    } finally {
      setLoading(false);
    }
  }, [botId, appId, toast]);

  useEffect(() => {
    loadData();
  }, [loadData]);

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

    // Fallback: poll every 3s in case postMessage doesn't work
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

      // If app requires OAuth setup, open popup for OAuth flow
      if (result?.needs_oauth && result?.oauth_redirect) {
        setWaitingForOAuth(true);
        const popup = window.open(result.oauth_redirect, "oauth_popup", "width=600,height=700,scrollbars=yes");
        setOAuthPopup(popup);
        setInstalling(false);
        return;
      }

      const installationId = result?.id || result?.installation_id;

      // If config_schema exists and form was filled, save config
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
  const events: string[] = app.events || [];

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
  const hasPermissions = readScopes.length > 0 || writeScopes.length > 0 || events.length > 0;

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="space-y-4">
        <button
          onClick={() => navigate(`/dashboard/accounts/${botId}`)}
          className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
        >
          <ArrowLeft className="h-4 w-4" />
          {botName || "返回"}
        </button>
        <h1 className="text-2xl font-bold">安装应用</h1>
      </div>

      <div className="grid gap-8 md:grid-cols-2">
        {/* Left section: App identity */}
        <Card>
          <CardHeader>
            <CardTitle>应用信息</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-start gap-4">
              <AppIcon icon={app.icon} iconUrl={app.icon_url} size="h-14 w-14" />
              <div className="flex-1 min-w-0 space-y-1">
                <h2 className="text-lg font-bold">{app.name}</h2>
                {app.slug && (
                  <p className="text-sm text-muted-foreground font-mono">{app.slug}</p>
                )}
              </div>
            </div>
            {app.description && (
              <p className="text-sm text-muted-foreground">{app.description}</p>
            )}
            {app.homepage_url && (
              <a
                href={app.homepage_url}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1.5 text-sm text-primary hover:underline"
              >
                <ExternalLink className="h-3.5 w-3.5" />
                主页
              </a>
            )}
          </CardContent>
        </Card>

        {/* Right section: Permissions + config */}
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>权限与配置</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Permissions */}
              {hasPermissions ? (
                <div className="space-y-3">
                  <p className="text-sm font-medium">此应用将能够：</p>

                  {readScopes.length > 0 && (
                    <div className="space-y-1.5">
                      {readScopes.map((scope: string) => (
                        <div key={scope} className="flex items-center gap-2 text-sm text-muted-foreground">
                          <Eye className="h-3.5 w-3.5 shrink-0" />
                          <span>{SCOPE_DESCRIPTIONS[scope] || scope}</span>
                        </div>
                      ))}
                    </div>
                  )}

                  {writeScopes.length > 0 && (
                    <div className="space-y-1.5">
                      {writeScopes.map((scope: string) => (
                        <div key={scope} className="flex items-center gap-2 text-sm text-muted-foreground">
                          <Zap className="h-3.5 w-3.5 shrink-0" />
                          <span>{SCOPE_DESCRIPTIONS[scope] || scope}</span>
                        </div>
                      ))}
                    </div>
                  )}

                  {events.length > 0 && (
                    <div className="space-y-1.5">
                      <p className="text-xs text-muted-foreground">订阅事件：</p>
                      <div className="flex flex-wrap gap-1.5">
                        {events.map((event: string) => (
                          <Badge key={event} variant="outline" className="font-mono text-xs">
                            {event}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">
                  接收 @mention 消息并执行响应。
                </p>
              )}

              {/* Tools (commands) */}
              {tools.length > 0 && (
                <div className="pt-2 border-t">
                  <ToolsDisplay tools={tools} />
                </div>
              )}

              {/* Handle */}
              <div className="space-y-1.5 pt-2 border-t">
                <Label htmlFor="install-handle" className="text-muted-foreground">
                  Handle
                </Label>
                <div className="flex items-center gap-2">
                  <Input
                    id="install-handle"
                    value={handle}
                    onChange={(e) => setHandle(e.target.value)}
                    className="h-8 text-xs font-mono flex-1"
                    placeholder="如 notify-prod"
                  />
                  <span className="text-xs text-muted-foreground font-mono shrink-0">
                    @{handle || "handle"}
                  </span>
                </div>
              </div>

              {/* Config schema form */}
              {Object.keys(schemaProperties).length > 0 && (
                <div className="space-y-3 pt-2 border-t">
                  <p className="text-sm font-medium">应用配置</p>
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
                </div>
              )}
            </CardContent>
          </Card>

          {/* Install button */}
          <Button className="w-full" onClick={handleInstall} disabled={installing}>
            {installing && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
            允许并安装
          </Button>
        </div>
      </div>
    </div>
  );
}
