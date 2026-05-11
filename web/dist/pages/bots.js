import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect, useRef, useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import QRCode from "qrcode";
import { Button } from "../components/ui/button";
import { Card, CardContent } from "../components/ui/card";
import {
  Plus,
  Trash2,
  RefreshCw,
  Bot as BotIcon,
  MessageCircle,
  Clock,
  Loader2,
  AlertCircle,
  MoreVertical,
  ArrowUpRight,
  Cpu,
  Wifi,
  WifiOff,
} from "lucide-react";
import { api, botDisplayName } from "../lib/api";
import { useBots, useDeleteBot, useReconnectBot } from "@/hooks/use-bots";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useToast } from "@/hooks/use-toast";
import { useConfirm } from "@/components/ui/confirm-dialog";
const statusConfig = {
  connected: { label: "运行中", variant: "default", dot: "bg-green-500" },
  disconnected: { label: "离线", variant: "outline", dot: "bg-muted-foreground" },
  error: { label: "故障", variant: "destructive", dot: "bg-destructive" },
  session_expired: { label: "授权过期", variant: "destructive", dot: "bg-destructive" },
};
export function BotsPage() {
  const { data: bots = [], isLoading, isFetching, refetch } = useBots();
  const loading = isLoading || isFetching;
  const [binding, setBinding] = useState(false);
  const [qrUrl, setQrUrl] = useState("");
  const [bindStatus, setBindStatus] = useState("");
  const navigate = useNavigate();
  const bindWsRef = useRef(null);
  const bindTimerRef = useRef(null);
  // Cleanup WS/timer when dialog closes or component unmounts
  useEffect(() => {
    return () => {
      if (bindTimerRef.current) clearTimeout(bindTimerRef.current);
      if (bindWsRef.current) bindWsRef.current.close();
    };
  }, []);
  async function startBind() {
    setBinding(true);
    setBindStatus("正在初始化...");
    try {
      const { session_id, qr_url } = await api.bindStart();
      setQrUrl(qr_url);
      setBindStatus("请使用手机微信扫描上方二维码");
      connectBindWS(session_id);
    } catch (err) {
      setBindStatus("初始化失败: " + err.message);
    }
  }
  function connectBindWS(sessionID, retries = 0) {
    const MAX_RETRIES = 5;
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const ws = new WebSocket(
      `${protocol}//${window.location.host}/api/bots/bind/status/${sessionID}`,
    );
    bindWsRef.current = ws;
    let settled = false;
    ws.onmessage = (e) => {
      const data = JSON.parse(e.data);
      if (data.event === "status") {
        if (data.status === "scanned") setBindStatus("已扫码，请在手机上点击确认...");
        if (data.status === "refreshed") {
          setQrUrl(data.qr_url);
          setBindStatus("二维码已刷新");
        }
        if (data.status === "connected") {
          settled = true;
          ws.close();
          setBinding(false);
          navigate(
            data.is_new && data.bot_id
              ? `/dashboard/onboarding?bot_id=${data.bot_id}`
              : "/dashboard/accounts",
          );
        }
      }
    };
    ws.onerror = () => {
      ws.close();
    };
    ws.onclose = () => {
      if (settled) return;
      if (retries < MAX_RETRIES) {
        const delay = Math.min(1000 * 2 ** retries, 8000);
        setBindStatus("连接中断，正在重连...");
        bindTimerRef.current = setTimeout(() => connectBindWS(sessionID, retries + 1), delay);
      } else {
        setBindStatus("连接中断，请重试");
        setBinding(false);
      }
    };
  }
  return _jsxs("div", {
    className: "space-y-6",
    children: [
      _jsxs("div", {
        className: "flex items-start justify-between gap-4",
        children: [
          _jsxs("div", {
            children: [
              _jsx("h1", {
                className: "text-2xl font-bold tracking-tight",
                children: "\u8D26\u53F7\u7BA1\u7406",
              }),
              _jsx("p", {
                className: "text-sm text-muted-foreground mt-0.5",
                children: "\u7BA1\u7406\u4F60\u7684\u5FAE\u4FE1\u8D26\u53F7\u3002",
              }),
            ],
          }),
          _jsxs("div", {
            className: "flex items-center gap-2 shrink-0",
            children: [
              _jsxs(Button, {
                variant: "outline",
                onClick: () => refetch(),
                disabled: loading,
                children: [
                  _jsx(RefreshCw, { className: `h-4 w-4 mr-2 ${loading ? "animate-spin" : ""}` }),
                  " \u5237\u65B0",
                ],
              }),
              _jsxs(Dialog, {
                open: binding,
                onOpenChange: (o) => {
                  setBinding(o);
                  if (o) startBind();
                  else setQrUrl("");
                },
                children: [
                  _jsx(DialogTrigger, {
                    asChild: true,
                    children: _jsxs(Button, {
                      className: "px-6 shadow-lg shadow-primary/20",
                      children: [
                        _jsx(Plus, { className: "mr-2 h-4 w-4" }),
                        " \u6DFB\u52A0\u8D26\u53F7",
                      ],
                    }),
                  }),
                  _jsxs(DialogContent, {
                    className: "sm:max-w-md max-h-[90vh] overflow-y-auto",
                    children: [
                      _jsxs(DialogHeader, {
                        className: "text-left",
                        children: [
                          _jsx(DialogTitle, {
                            className: "text-xl",
                            children: "\u626B\u7801\u767B\u5F55",
                          }),
                          _jsx(DialogDescription, {
                            children: "\u4F7F\u7528\u5FAE\u4FE1\u626B\u7801\u767B\u5F55\u3002",
                          }),
                        ],
                      }),
                      _jsxs("div", {
                        className: "flex flex-col items-center justify-center gap-8 py-12",
                        children: [
                          _jsxs("div", {
                            className: "relative group",
                            children: [
                              _jsx("div", {
                                className:
                                  "absolute -inset-4 bg-primary/5 rounded-[2rem] blur-xl group-hover:bg-primary/10 transition-all",
                              }),
                              qrUrl
                                ? _jsx("div", {
                                    className:
                                      "relative rounded-2xl border-4 border-background bg-white p-4 shadow-2xl",
                                    children: _jsx(QrCanvas, { url: qrUrl }),
                                  })
                                : _jsx("div", {
                                    className:
                                      "relative flex h-[240px] w-[240px] items-center justify-center rounded-2xl border-2 border-dashed bg-muted/30",
                                    children: _jsx(Loader2, {
                                      className: "h-10 w-10 animate-spin text-primary/40",
                                    }),
                                  }),
                            ],
                          }),
                          _jsxs("div", {
                            className: "text-center space-y-2",
                            children: [
                              _jsx("p", { className: "font-bold text-lg", children: bindStatus }),
                              _jsx("p", {
                                className:
                                  "text-xs text-muted-foreground max-w-[240px] mx-auto leading-relaxed",
                                children:
                                  "\u767B\u5F55\u6210\u529F\u540E\u5373\u53EF\u4F7F\u7528\u3002",
                              }),
                            ],
                          }),
                        ],
                      }),
                    ],
                  }),
                ],
              }),
            ],
          }),
        ],
      }),
      loading && bots.length === 0
        ? _jsx("div", {
            className: "grid gap-6 md:grid-cols-2 lg:grid-cols-3",
            children: [1, 2, 3].map((i) =>
              _jsx(Card, { className: "h-[220px] animate-pulse bg-muted/20" }, i),
            ),
          })
        : _jsxs("div", {
            className: "grid gap-6 md:grid-cols-2 lg:grid-cols-3",
            children: [
              bots.map((bot) =>
                _jsx(BotInstanceCard, { bot: bot, onRebind: () => setBinding(true) }, bot.id),
              ),
              bots.length === 0
                ? _jsxs("div", {
                    className:
                      "col-span-full py-24 border-2 border-dashed rounded-[2rem] flex flex-col items-center justify-center text-center bg-muted/5",
                    children: [
                      _jsx("div", {
                        className:
                          "h-20 w-20 rounded-3xl bg-background border shadow-sm flex items-center justify-center mb-6",
                        children: _jsx(BotIcon, { className: "h-10 w-10 text-primary/40" }),
                      }),
                      _jsx("h3", {
                        className: "text-xl font-bold",
                        children: "\u8FD8\u6CA1\u6709\u8D26\u53F7",
                      }),
                      _jsx("p", {
                        className: "text-muted-foreground mt-2 max-w-sm",
                        children:
                          "\u6DFB\u52A0\u4F60\u7684\u7B2C\u4E00\u4E2A\u5FAE\u4FE1\u8D26\u53F7\u3002",
                      }),
                      _jsx(Button, {
                        variant: "outline",
                        className: "mt-8 h-11 px-8 rounded-full",
                        onClick: () => {
                          setBinding(true);
                          startBind();
                        },
                        children: "\u6DFB\u52A0\u8D26\u53F7",
                      }),
                    ],
                  })
                : null,
            ],
          }),
    ],
  });
}
function QrCanvas({ url }) {
  const ref = useRef(null);
  useEffect(() => {
    if (url && ref.current) QRCode.toCanvas(ref.current, url, { width: 224, margin: 0 });
  }, [url]);
  return _jsx("canvas", { ref: ref, className: "block rounded-lg" });
}
function BotInstanceCard({ bot, onRebind }) {
  const { toast } = useToast();
  const { confirm, ConfirmDialog } = useConfirm();
  const deleteMutation = useDeleteBot();
  const reconnectMutation = useReconnectBot();
  const status = statusConfig[bot.status] || statusConfig.disconnected;
  const isOnline = bot.status === "connected";
  async function handleAction(action) {
    if (action === "delete") {
      const ok = await confirm({
        title: "删除确认",
        description: "确定要删除此账号？相关转发规则将停止工作。",
        confirmText: "删除",
        variant: "destructive",
      });
      if (!ok) return;
      deleteMutation.mutate(bot.id, {
        onSuccess: () => toast({ title: "已删除账号" }),
        onError: (e) =>
          toast({ variant: "destructive", title: "操作失败", description: e.message }),
      });
    } else if (action === "reconnect") {
      reconnectMutation.mutate(bot.id, {
        onSuccess: () => toast({ title: "指令已发出", description: "正在尝试重新建立连接..." }),
        onError: (e) =>
          toast({ variant: "destructive", title: "操作失败", description: e.message }),
      });
    }
  }
  return _jsxs(Card, {
    className:
      "group flex flex-col border-border/50 hover:border-primary/20 hover:shadow-lg transition-all duration-200",
    children: [
      ConfirmDialog,
      _jsxs(CardContent, {
        className: "p-5 flex-1 space-y-4",
        children: [
          _jsxs("div", {
            className: "flex items-start justify-between gap-2",
            children: [
              _jsxs("div", {
                className: "flex items-center gap-3 min-w-0",
                children: [
                  _jsx("div", {
                    className: `h-10 w-10 rounded-xl flex items-center justify-center shrink-0 transition-colors ${isOnline ? "bg-primary/10 text-primary" : "bg-muted text-muted-foreground"}`,
                    children: isOnline
                      ? _jsx(Wifi, { className: "h-5 w-5" })
                      : _jsx(WifiOff, { className: "h-5 w-5" }),
                  }),
                  _jsxs("div", {
                    className: "min-w-0",
                    children: [
                      _jsx("p", {
                        className: "font-semibold leading-tight truncate",
                        children: botDisplayName(bot),
                      }),
                      _jsxs("div", {
                        className: "flex items-center gap-1.5 mt-0.5",
                        children: [
                          _jsx("span", {
                            className: `size-1.5 rounded-full shrink-0 ${status.dot} ${isOnline ? "animate-pulse" : ""}`,
                          }),
                          _jsx("span", {
                            className: "text-xs text-muted-foreground",
                            children: status.label,
                          }),
                        ],
                      }),
                    ],
                  }),
                ],
              }),
              _jsxs(DropdownMenu, {
                children: [
                  _jsx(DropdownMenuTrigger, {
                    asChild: true,
                    children: _jsx(Button, {
                      variant: "ghost",
                      size: "icon-sm",
                      className:
                        "shrink-0 opacity-0 group-hover:opacity-100 transition-opacity mt-0.5",
                      children: _jsx(MoreVertical, { className: "h-4 w-4" }),
                    }),
                  }),
                  _jsxs(DropdownMenuContent, {
                    align: "end",
                    className: "w-40",
                    children: [
                      bot.status !== "session_expired"
                        ? _jsxs(DropdownMenuItem, {
                            onClick: () => handleAction("reconnect"),
                            className: "gap-2",
                            children: [
                              _jsx(RefreshCw, { className: "h-3.5 w-3.5" }),
                              " \u91CD\u65B0\u8FDE\u63A5",
                            ],
                          })
                        : null,
                      _jsxs(DropdownMenuItem, {
                        onSelect: (e) => {
                          e.preventDefault();
                          void handleAction("delete");
                        },
                        className:
                          "gap-2 text-destructive focus:bg-destructive/10 focus:text-destructive",
                        children: [
                          _jsx(Trash2, { className: "h-3.5 w-3.5" }),
                          " \u5220\u9664\u8D26\u53F7",
                        ],
                      }),
                    ],
                  }),
                ],
              }),
            ],
          }),
          _jsxs("div", {
            className: "flex items-center gap-4",
            children: [
              _jsxs("div", {
                className: "flex items-center gap-1.5 text-xs text-muted-foreground",
                children: [
                  _jsx(MessageCircle, { className: "h-3.5 w-3.5" }),
                  _jsxs("span", { children: [bot.msg_count ?? 0, " \u6D88\u606F"] }),
                ],
              }),
              bot.reminder_hours
                ? _jsxs("div", {
                    className: "flex items-center gap-1.5 text-xs text-muted-foreground",
                    children: [
                      _jsx(Clock, { className: "h-3.5 w-3.5 text-orange-500" }),
                      _jsxs("span", {
                        children: ["\u63D0\u524D ", 24 - bot.reminder_hours, "h \u63D0\u9192"],
                      }),
                    ],
                  })
                : null,
            ],
          }),
          bot.status === "session_expired"
            ? _jsx("div", {
                className: "rounded-lg bg-destructive/5 border border-destructive/10 p-3",
                children: _jsxs("div", {
                  className: "flex items-start gap-2",
                  children: [
                    _jsx(AlertCircle, {
                      className: "h-3.5 w-3.5 mt-0.5 text-destructive shrink-0",
                    }),
                    _jsxs("div", {
                      className: "space-y-1",
                      children: [
                        _jsx("p", {
                          className: "text-xs text-destructive leading-snug",
                          children:
                            "\u4F1A\u8BDD\u5DF2\u8FC7\u671F\uFF0C\u8BF7\u5728\u5FAE\u4FE1\u4E2D\u7ED9\u8BE5\u8D26\u53F7\u53D1\u4E00\u6761\u6D88\u606F\u4EE5\u6062\u590D\u8FDE\u63A5\u3002",
                        }),
                        _jsx(Button, {
                          variant: "link",
                          size: "xs",
                          className: "h-auto p-0 text-destructive text-xs",
                          onClick: onRebind,
                          children: "\u6216\u91CD\u65B0\u626B\u7801\u7ED1\u5B9A",
                        }),
                      ],
                    }),
                  ],
                }),
              })
            : null,
        ],
      }),
      _jsxs("div", {
        className: "mx-5 mb-5 pt-3 border-t border-border/40 flex items-center justify-between",
        children: [
          _jsxs("div", {
            className: "flex items-center gap-1.5 text-xs text-muted-foreground/60",
            children: [
              _jsx(Cpu, { className: "h-3 w-3 shrink-0" }),
              _jsx("span", { className: "capitalize", children: bot.provider || "未知" }),
              _jsx("span", { className: "mx-0.5 opacity-40", children: "\u00B7" }),
              _jsx("span", { className: "font-mono", children: bot.id.slice(0, 8) }),
            ],
          }),
          _jsxs(Link, {
            to: `/dashboard/accounts/${bot.id}`,
            className:
              "flex items-center gap-1 text-xs text-muted-foreground hover:text-primary transition-colors group/link",
            children: [
              _jsx("span", { children: "\u67E5\u770B\u8BE6\u60C5" }),
              _jsx(ArrowUpRight, {
                className:
                  "h-3 w-3 group-hover/link:translate-x-0.5 group-hover/link:-translate-y-0.5 transition-transform duration-200",
              }),
            ],
          }),
        ],
      }),
    ],
  });
}
