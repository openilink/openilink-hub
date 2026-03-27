import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import {
  Bot as BotIcon, Sparkles, ArrowRight, Check, Download, Loader2, Blocks, ChevronRight,
} from "lucide-react";
import { api } from "../lib/api";
import { useToast } from "@/hooks/use-toast";
import { AppIcon } from "../components/app-icon";

export function OnboardingPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const botId = searchParams.get("bot_id");
  const { toast } = useToast();

  const [step, setStep] = useState(1);
  const [bot, setBotInfo] = useState<any>(null);
  const [enableAI, setEnableAI] = useState(true);
  const [saving, setSaving] = useState(false);
  const [loadingConfig, setLoadingConfig] = useState(true);

  // Step 2
  const [apps, setApps] = useState<any[]>([]);
  const [loadingApps, setLoadingApps] = useState(false);
  const [installedIds, setInstalledIds] = useState<Set<string>>(new Set());
  const [installingId, setInstallingId] = useState<string | null>(null);

  useEffect(() => {
    if (!botId) { navigate("/dashboard/accounts", { replace: true }); return; }
    api.listBots().then(bots => {
      const found = (bots || []).find((b: any) => b.id === botId);
      if (found) {
        setBotInfo(found);
        setEnableAI(found.ai_enabled ?? false);
      }
      setLoadingConfig(false);
    }).catch(() => setLoadingConfig(false));
  }, [botId]);

  useEffect(() => {
    if (step !== 2 || !botId) return;
    setLoadingApps(true);
    Promise.all([
      api.listApps({ listing: "listed" }),
      api.listBotApps(botId),
    ]).then(([marketplace, installed]) => {
      setApps(marketplace || []);
      setInstalledIds(new Set((installed || []).map((i: any) => i.app_id)));
    }).finally(() => setLoadingApps(false));
  }, [step, botId]);

  async function handleNextFromStep1() {
    setSaving(true);
    try {
      await api.setBotAI(botId!, enableAI);
      setStep(2);
    } catch (e: any) {
      toast({ variant: "destructive", title: "保存失败", description: e.message || "请稍后重试" });
    }
    setSaving(false);
  }

  async function handleInstall(app: any) {
    setInstallingId(app.id);
    try {
      await api.installApp(app.id, { bot_id: botId!, handle: app.slug || app.name });
      setInstalledIds(prev => new Set(prev).add(app.id));
      toast({ title: "安装成功", description: `已安装 ${app.name}` });
    } catch (e: any) {
      toast({ variant: "destructive", title: "安装失败", description: e.message });
    }
    setInstallingId(null);
  }

  function handleFinish() {
    navigate(`/dashboard/accounts/${botId}`);
  }

  if (!botId) return null;

  return (
    <div className="max-w-2xl mx-auto py-12 px-4">
      {/* Progress indicator */}
      <div className="flex items-center justify-center gap-3 mb-10">
        <div className={`flex items-center gap-2 px-4 py-2 rounded-full text-sm font-bold transition-colors ${step === 1 ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"}`}>
          <span className="h-5 w-5 rounded-full bg-background/20 flex items-center justify-center text-xs">{step > 1 ? <Check className="h-3 w-3" /> : "1"}</span>
          基础设置
        </div>
        <ChevronRight className="h-4 w-4 text-muted-foreground" />
        <div className={`flex items-center gap-2 px-4 py-2 rounded-full text-sm font-bold transition-colors ${step === 2 ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"}`}>
          <span className="h-5 w-5 rounded-full bg-background/20 flex items-center justify-center text-xs">2</span>
          安装应用
        </div>
      </div>

      {step === 1 && (
        <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4">
          <div className="text-center space-y-2">
            <div className="h-16 w-16 rounded-2xl bg-primary/10 flex items-center justify-center mx-auto mb-4">
              <BotIcon className="h-8 w-8 text-primary" />
            </div>
            <h1 className="text-2xl font-bold">账号添加成功！</h1>
            <p className="text-muted-foreground">
              {(bot?.display_name || bot?.name) ? `"${bot.display_name || bot.name}" 已就绪` : "你的账号已就绪"}，接下来做一些基础设置。
            </p>
          </div>

          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between p-4 rounded-xl border bg-muted/20">
                <div className="flex items-center gap-3">
                  <Sparkles className="h-5 w-5 text-primary" />
                  <div>
                    <p className="font-bold text-sm">AI 自动回复</p>
                    <p className="text-xs text-muted-foreground">开启后，Bot 会自动回复收到的消息</p>
                  </div>
                </div>
                <input
                  type="checkbox"
                  checked={enableAI}
                  onChange={e => setEnableAI(e.target.checked)}
                  className="h-5 w-5 accent-primary cursor-pointer"
                />
              </div>
            </CardContent>
          </Card>

          <div className="flex justify-end gap-3">
            <Button variant="ghost" onClick={handleFinish}>跳过</Button>
            <Button onClick={handleNextFromStep1} disabled={saving} className="px-8">
              {saving && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
              下一步 <ArrowRight className="h-4 w-4 ml-2" />
            </Button>
          </div>
        </div>
      )}

      {step === 2 && (
        <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4">
          <div className="text-center space-y-2">
            <div className="h-16 w-16 rounded-2xl bg-primary/10 flex items-center justify-center mx-auto mb-4">
              <Blocks className="h-8 w-8 text-primary" />
            </div>
            <h1 className="text-2xl font-bold">安装应用</h1>
            <p className="text-muted-foreground">
              为你的账号安装应用，扩展 Bot 的能力。
            </p>
          </div>

          {loadingApps ? (
            <div className="flex justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : apps.length === 0 ? (
            <Card>
              <CardContent className="py-12 text-center">
                <Blocks className="h-10 w-10 mx-auto text-muted-foreground/40 mb-3" />
                <p className="text-sm text-muted-foreground">暂无可用的应用</p>
              </CardContent>
            </Card>
          ) : (
            <div className="grid gap-4 md:grid-cols-2">
              {apps.map(app => {
                const installed = installedIds.has(app.id);
                const installing = installingId === app.id;
                return (
                  <Card key={app.id} className="overflow-hidden border-border/50">
                    <CardHeader className="pb-3">
                      <div className="flex items-start gap-3">
                        <AppIcon icon={app.icon} iconUrl={app.icon_url} />
                        <div className="min-w-0 space-y-0.5">
                          <CardTitle className="text-sm font-bold truncate">{app.name}</CardTitle>
                          <p className="text-xs text-muted-foreground line-clamp-1">{app.description || app.slug}</p>
                        </div>
                      </div>
                    </CardHeader>
                    <CardFooter className="bg-muted/30 pt-3 flex justify-end">
                      {installed ? (
                        <Badge variant="secondary" className="gap-1">
                          <Check className="h-3 w-3" /> 已安装
                        </Badge>
                      ) : (
                        <Button size="sm" variant="outline" onClick={() => handleInstall(app)} disabled={installing}>
                          {installing ? <Loader2 className="h-3 w-3 animate-spin mr-1" /> : <Download className="h-3 w-3 mr-1" />}
                          安装
                        </Button>
                      )}
                    </CardFooter>
                  </Card>
                );
              })}
            </div>
          )}

          <div className="flex justify-end gap-3">
            <Button variant="ghost" onClick={handleFinish}>跳过</Button>
            <Button onClick={handleFinish} className="px-8">
              完成 <Check className="h-4 w-4 ml-2" />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
