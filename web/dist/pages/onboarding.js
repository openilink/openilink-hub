import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import {
  Bot as BotIcon,
  Sparkles,
  ArrowRight,
  Check,
  Download,
  Loader2,
  Blocks,
  ChevronRight,
} from "lucide-react";
import { api, botDisplayName } from "../lib/api";
import { useToast } from "@/hooks/use-toast";
import { AppIcon } from "../components/app-icon";
import { useBot, useSetBotAI, useBotApps } from "@/hooks/use-bots";
import { useApps } from "@/hooks/use-apps";
export function OnboardingPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const botId = searchParams.get("bot_id");
  const { toast } = useToast();
  const [step, setStep] = useState(1);
  const [enableAI, setEnableAI] = useState(true);
  const [installedIds, setInstalledIds] = useState(new Set());
  const [installingId, setInstallingId] = useState(null);
  const { data: bot, isLoading: loadingConfig, isError: botError } = useBot(botId || "");
  const setBotAI = useSetBotAI();
  // Step 2: marketplace apps + installed apps for this bot
  const {
    data: apps = [],
    isLoading: loadingApps,
    isError: appsError,
  } = useApps({ listing: "listed" });
  const { data: installedApps } = useBotApps(botId || "");
  useEffect(() => {
    if (!botId) {
      navigate("/dashboard/accounts", { replace: true });
    }
  }, [botId, navigate]);
  useEffect(() => {
    if (bot) setEnableAI(bot.ai_enabled ?? false);
  }, [bot]);
  useEffect(() => {
    if (installedApps) {
      setInstalledIds(new Set((installedApps || []).map((i) => i.app_id)));
    }
  }, [installedApps]);
  async function handleNextFromStep1() {
    setBotAI.mutate(
      { botId: botId, enabled: enableAI },
      {
        onSuccess: () => setStep(2),
        onError: (e) =>
          toast({
            variant: "destructive",
            title: "保存失败",
            description: e.message || "请稍后重试",
          }),
      },
    );
  }
  async function handleInstall(app) {
    setInstallingId(app.id);
    try {
      await api.installApp(app.id, { bot_id: botId, handle: app.slug || app.name });
      setInstalledIds((prev) => new Set(prev).add(app.id));
      toast({ title: "安装成功", description: `已安装 ${app.name}` });
    } catch (e) {
      toast({ variant: "destructive", title: "安装失败", description: e.message });
    }
    setInstallingId(null);
  }
  function handleFinish() {
    navigate(`/dashboard/accounts/${botId}`);
  }
  if (!botId) return null;
  if (botError || appsError)
    return _jsx("div", {
      className: "flex items-center justify-center py-20 text-sm text-destructive",
      children: "\u52A0\u8F7D\u5931\u8D25\uFF0C\u8BF7\u5237\u65B0\u91CD\u8BD5",
    });
  return _jsxs("div", {
    className: "max-w-2xl mx-auto py-12 px-4",
    children: [
      _jsxs("div", {
        className: "flex items-center justify-center gap-3 mb-10",
        children: [
          _jsxs("div", {
            className: `flex items-center gap-2 px-4 py-2 rounded-full text-sm font-bold transition-colors ${step === 1 ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"}`,
            children: [
              _jsx("span", {
                className:
                  "h-5 w-5 rounded-full bg-background/20 flex items-center justify-center text-xs",
                children: step > 1 ? _jsx(Check, { className: "h-3 w-3" }) : "1",
              }),
              "\u57FA\u7840\u8BBE\u7F6E",
            ],
          }),
          _jsx(ChevronRight, { className: "h-4 w-4 text-muted-foreground" }),
          _jsxs("div", {
            className: `flex items-center gap-2 px-4 py-2 rounded-full text-sm font-bold transition-colors ${step === 2 ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"}`,
            children: [
              _jsx("span", {
                className:
                  "h-5 w-5 rounded-full bg-background/20 flex items-center justify-center text-xs",
                children: "2",
              }),
              "\u5B89\u88C5\u5E94\u7528",
            ],
          }),
        ],
      }),
      step === 1 &&
        _jsxs("div", {
          className: "space-y-8 animate-in fade-in slide-in-from-bottom-4",
          children: [
            _jsxs("div", {
              className: "text-center space-y-2",
              children: [
                _jsx("div", {
                  className:
                    "h-16 w-16 rounded-2xl bg-primary/10 flex items-center justify-center mx-auto mb-4",
                  children: _jsx(BotIcon, { className: "h-8 w-8 text-primary" }),
                }),
                _jsx("h1", {
                  className: "text-2xl font-bold",
                  children: "\u8D26\u53F7\u6DFB\u52A0\u6210\u529F\uFF01",
                }),
                _jsxs("p", {
                  className: "text-muted-foreground",
                  children: [
                    bot ? `"${botDisplayName(bot)}" 已就绪` : "你的账号已就绪",
                    "\uFF0C\u63A5\u4E0B\u6765\u505A\u4E00\u4E9B\u57FA\u7840\u8BBE\u7F6E\u3002",
                  ],
                }),
              ],
            }),
            _jsx(Card, {
              children: _jsx(CardContent, {
                className: "pt-6",
                children: _jsxs("div", {
                  className: "flex items-center justify-between p-4 rounded-xl border bg-muted/20",
                  children: [
                    _jsxs("div", {
                      className: "flex items-center gap-3",
                      children: [
                        _jsx(Sparkles, { className: "h-5 w-5 text-primary" }),
                        _jsxs("div", {
                          children: [
                            _jsx("p", {
                              className: "font-bold text-sm",
                              children: "AI \u81EA\u52A8\u56DE\u590D",
                            }),
                            _jsx("p", {
                              className: "text-xs text-muted-foreground",
                              children:
                                "\u5F00\u542F\u540E\uFF0CBot \u4F1A\u81EA\u52A8\u56DE\u590D\u6536\u5230\u7684\u6D88\u606F",
                            }),
                          ],
                        }),
                      ],
                    }),
                    _jsx("input", {
                      type: "checkbox",
                      checked: enableAI,
                      onChange: (e) => setEnableAI(e.target.checked),
                      className: "h-5 w-5 accent-primary cursor-pointer",
                    }),
                  ],
                }),
              }),
            }),
            _jsxs("div", {
              className: "flex justify-end gap-3",
              children: [
                _jsx(Button, { variant: "ghost", onClick: handleFinish, children: "\u8DF3\u8FC7" }),
                _jsxs(Button, {
                  onClick: handleNextFromStep1,
                  disabled: setBotAI.isPending,
                  className: "px-8",
                  children: [
                    setBotAI.isPending && _jsx(Loader2, { className: "h-4 w-4 animate-spin mr-2" }),
                    "\u4E0B\u4E00\u6B65 ",
                    _jsx(ArrowRight, { className: "h-4 w-4 ml-2" }),
                  ],
                }),
              ],
            }),
          ],
        }),
      step === 2 &&
        _jsxs("div", {
          className: "space-y-8 animate-in fade-in slide-in-from-bottom-4",
          children: [
            _jsxs("div", {
              className: "text-center space-y-2",
              children: [
                _jsx("div", {
                  className:
                    "h-16 w-16 rounded-2xl bg-primary/10 flex items-center justify-center mx-auto mb-4",
                  children: _jsx(Blocks, { className: "h-8 w-8 text-primary" }),
                }),
                _jsx("h1", {
                  className: "text-2xl font-bold",
                  children: "\u5B89\u88C5\u5E94\u7528",
                }),
                _jsx("p", {
                  className: "text-muted-foreground",
                  children:
                    "\u4E3A\u4F60\u7684\u8D26\u53F7\u5B89\u88C5\u5E94\u7528\uFF0C\u6269\u5C55 Bot \u7684\u80FD\u529B\u3002",
                }),
              ],
            }),
            loadingApps
              ? _jsx("div", {
                  className: "flex justify-center py-12",
                  children: _jsx(Loader2, {
                    className: "h-8 w-8 animate-spin text-muted-foreground",
                  }),
                })
              : apps.length === 0
                ? _jsx(Card, {
                    children: _jsxs(CardContent, {
                      className: "py-12 text-center",
                      children: [
                        _jsx(Blocks, {
                          className: "h-10 w-10 mx-auto text-muted-foreground/40 mb-3",
                        }),
                        _jsx("p", {
                          className: "text-sm text-muted-foreground",
                          children: "\u6682\u65E0\u53EF\u7528\u7684\u5E94\u7528",
                        }),
                      ],
                    }),
                  })
                : _jsx("div", {
                    className: "grid gap-4 md:grid-cols-2",
                    children: apps.map((app) => {
                      const installed = installedIds.has(app.id);
                      const installing = installingId === app.id;
                      return _jsxs(
                        Card,
                        {
                          className: "overflow-hidden border-border/50",
                          children: [
                            _jsx(CardHeader, {
                              className: "pb-3",
                              children: _jsxs("div", {
                                className: "flex items-start gap-3",
                                children: [
                                  _jsx(AppIcon, { icon: app.icon, iconUrl: app.icon_url }),
                                  _jsxs("div", {
                                    className: "min-w-0 space-y-0.5",
                                    children: [
                                      _jsx(CardTitle, {
                                        className: "text-sm font-bold truncate",
                                        children: app.name,
                                      }),
                                      _jsx("p", {
                                        className: "text-xs text-muted-foreground line-clamp-1",
                                        children: app.description || app.slug,
                                      }),
                                    ],
                                  }),
                                ],
                              }),
                            }),
                            _jsx(CardFooter, {
                              className: "bg-muted/30 pt-3 flex justify-end",
                              children: installed
                                ? _jsxs(Badge, {
                                    variant: "secondary",
                                    className: "gap-1",
                                    children: [
                                      _jsx(Check, { className: "h-3 w-3" }),
                                      " \u5DF2\u5B89\u88C5",
                                    ],
                                  })
                                : _jsxs(Button, {
                                    size: "sm",
                                    variant: "outline",
                                    onClick: () => handleInstall(app),
                                    disabled: installing,
                                    children: [
                                      installing
                                        ? _jsx(Loader2, { className: "h-3 w-3 animate-spin mr-1" })
                                        : _jsx(Download, { className: "h-3 w-3 mr-1" }),
                                      "\u5B89\u88C5",
                                    ],
                                  }),
                            }),
                          ],
                        },
                        app.id,
                      );
                    }),
                  }),
            _jsxs("div", {
              className: "flex justify-end gap-3",
              children: [
                _jsx(Button, { variant: "ghost", onClick: handleFinish, children: "\u8DF3\u8FC7" }),
                _jsxs(Button, {
                  onClick: handleFinish,
                  className: "px-8",
                  children: ["\u5B8C\u6210 ", _jsx(Check, { className: "h-4 w-4 ml-2" })],
                }),
              ],
            }),
          ],
        }),
    ],
  });
}
