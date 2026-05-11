import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect, useRef, useState } from "react";
import {
  KeyRound,
  Shield,
  User,
  Lock,
  ArrowRight,
  Loader2,
  Github,
  X,
  QrCode,
  ChevronDown,
} from "lucide-react";
import { useNavigate, Link } from "react-router-dom";
import QRCode from "qrcode";
import { Button } from "../components/ui/button";
import { HexagonBackground } from "../components/ui/hexagon-background";
import { Card, CardContent, CardFooter } from "../components/ui/card";
import { Input } from "../components/ui/input";
import { api } from "../lib/api";
import { Separator } from "../components/ui/separator";
import { useOAuthProviders } from "@/hooks/use-settings";
import { useInfo } from "@/hooks/use-auth";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
const providerLabels = {
  github: { label: "GitHub", icon: Github },
  linuxdo: { label: "LinuxDo", icon: Shield },
};
function QrCanvas({ url }) {
  const ref = useRef(null);
  useEffect(() => {
    if (url && ref.current) QRCode.toCanvas(ref.current, url, { width: 224, margin: 0 });
  }, [url]);
  return _jsx("canvas", { ref: ref, className: "block rounded-lg" });
}
export function LoginPage() {
  const navigate = useNavigate();
  // Scan login state
  const [qrUrl, setQrUrl] = useState("");
  const [scanStatus, setScanStatus] = useState("idle");
  const [scanMessage, setScanMessage] = useState("");
  const scanWsRef = useRef(null);
  const scanTimerRef = useRef(null);
  // Password login state
  const [mode, setMode] = useState("login");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  // OAuth
  const { data: oauthProviders = [] } = useOAuthProviders();
  // Registration enabled flag (from /api/info)
  const { data: infoData } = useInfo();
  const registrationEnabled = infoData?.registration_enabled !== false;
  useEffect(() => {
    if (infoData?.registration_enabled === false) {
      setMode("login");
    }
  }, [infoData]);
  // Auto-start scan login on mount; cleanup WS/timer on unmount
  useEffect(() => {
    startScanLogin();
    return () => {
      if (scanTimerRef.current) clearTimeout(scanTimerRef.current);
      if (scanWsRef.current) scanWsRef.current.close();
    };
  }, []);
  async function startScanLogin() {
    setScanStatus("loading");
    setScanMessage("正在初始化...");
    setQrUrl("");
    try {
      const res = await fetch("/api/auth/scan/start", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "初始化失败");
      setQrUrl(data.qr_url);
      setScanStatus("wait");
      setScanMessage("请使用微信扫描二维码");
      connectScanWS(data.session_id);
    } catch (err) {
      setScanStatus("error");
      setScanMessage(err.message || "初始化失败");
    }
  }
  function connectScanWS(sessionID, retries = 0) {
    const MAX_RETRIES = 5;
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const ws = new WebSocket(
      `${protocol}//${window.location.host}/api/auth/scan/status/${sessionID}`,
    );
    scanWsRef.current = ws;
    let settled = false;
    ws.onmessage = (e) => {
      const d = JSON.parse(e.data);
      if (d.event === "status") {
        if (d.status === "wait") {
          // keep waiting
        } else if (d.status === "scanned") {
          setScanStatus("scanned");
          setScanMessage("已扫码，请在手机上确认...");
        } else if (d.status === "refreshed") {
          setQrUrl(d.qr_url);
          setScanStatus("wait");
          setScanMessage("二维码已刷新，请重新扫描");
        } else if (d.status === "connected") {
          settled = true;
          if (d.session_token) {
            document.cookie = `session=${d.session_token}; path=/; max-age=${7 * 24 * 3600}; samesite=lax`;
          }
          ws.close();
          navigate(
            d.is_new && d.bot_id ? `/dashboard/onboarding?bot_id=${d.bot_id}` : "/dashboard",
          );
        }
      } else if (d.event === "error") {
        settled = true;
        setScanMessage(d.message || "扫码登录失败");
        setScanStatus("error");
        ws.close();
      }
    };
    ws.onerror = () => {
      ws.close();
    };
    ws.onclose = () => {
      if (settled) return;
      if (retries < MAX_RETRIES) {
        const delay = Math.min(1000 * 2 ** retries, 8000);
        setScanMessage("连接中断，正在重连...");
        scanTimerRef.current = setTimeout(() => connectScanWS(sessionID, retries + 1), delay);
      } else {
        setScanStatus("error");
        setScanMessage("连接中断，请刷新重试");
      }
    };
  }
  // Password login
  async function handleSubmit(e) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      if (mode === "register") {
        await api.register(username, password);
      } else {
        await api.login(username, password);
      }
      navigate("/dashboard");
    } catch (err) {
      setError(err.message);
    }
    setLoading(false);
  }
  // Passkey login
  function base64urlToBuffer(b64) {
    const base64 = b64.replace(/-/g, "+").replace(/_/g, "/");
    const pad = base64.length % 4 === 0 ? "" : "=".repeat(4 - (base64.length % 4));
    const bin = atob(base64 + pad);
    const bytes = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
    return bytes.buffer;
  }
  function bufferToBase64url(buf) {
    const bytes = new Uint8Array(buf);
    let bin = "";
    for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i]);
    return btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=/g, "");
  }
  async function handlePasskeyLogin() {
    setError("");
    setLoading(true);
    try {
      const options = await fetch("/api/auth/passkey/login/begin", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      }).then((r) => r.json());
      options.publicKey.challenge = base64urlToBuffer(options.publicKey.challenge);
      if (options.publicKey.allowCredentials) {
        options.publicKey.allowCredentials = options.publicKey.allowCredentials.map(
          (credential) => ({ ...credential, id: base64urlToBuffer(credential.id) }),
        );
      }
      const credential = await navigator.credentials.get(options);
      if (!credential) throw new Error("cancelled");
      const response = credential.response;
      const body = JSON.stringify({
        id: credential.id,
        rawId: bufferToBase64url(credential.rawId),
        type: credential.type,
        response: {
          authenticatorData: bufferToBase64url(response.authenticatorData),
          clientDataJSON: bufferToBase64url(response.clientDataJSON),
          signature: bufferToBase64url(response.signature),
          userHandle: response.userHandle ? bufferToBase64url(response.userHandle) : "",
        },
      });
      const res = await fetch("/api/auth/passkey/login/finish", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body,
      });
      if (!res.ok) {
        const data = await res.json();
        throw new Error(data.error || "登录失败");
      }
      navigate("/dashboard");
    } catch (err) {
      if (err.name !== "NotAllowedError") setError(err.message || "Passkey 登录失败");
    }
    setLoading(false);
  }
  const supportsPasskey = typeof window !== "undefined" && "PublicKeyCredential" in window;
  return _jsxs("div", {
    className:
      "relative flex min-h-screen items-center justify-center overflow-hidden bg-background px-4 py-12",
    children: [
      _jsx(HexagonBackground, { className: "opacity-20", hexagonSize: 60, hexagonMargin: 4 }),
      _jsx("div", {
        className:
          "absolute inset-0 bg-[radial-gradient(circle_at_center,transparent_0%,hsl(var(--background))_100%)]",
      }),
      _jsxs("div", {
        className: "relative z-10 w-full max-w-[420px] animate-in fade-in zoom-in-95 duration-500",
        children: [
          _jsxs("div", {
            className: "mb-8 text-center space-y-2",
            children: [
              _jsx("div", {
                className:
                  "mx-auto flex h-12 w-12 items-center justify-center rounded-2xl bg-primary text-primary-foreground shadow-lg shadow-primary/20 mb-4",
                children: _jsx(Shield, { className: "h-6 w-6" }),
              }),
              _jsx("h1", {
                className: "text-3xl font-extrabold tracking-tight",
                children: "OpeniLink Hub",
              }),
              _jsx("p", {
                className: "text-sm text-muted-foreground font-medium",
                children: "\u5FAE\u4FE1\u626B\u7801\uFF0C\u4E00\u952E\u767B\u5F55",
              }),
            ],
          }),
          _jsxs(Card, {
            className: "border-border/50 shadow-2xl backdrop-blur-md bg-card/80",
            children: [
              _jsxs(CardContent, {
                className: "pt-8 pb-6",
                children: [
                  _jsxs("div", {
                    className: "flex flex-col items-center gap-6",
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
                                  "relative flex h-[224px] w-[224px] items-center justify-center rounded-2xl border-2 border-dashed bg-muted/30",
                                children:
                                  scanStatus === "error"
                                    ? _jsxs("div", {
                                        className: "text-center space-y-3 px-4",
                                        children: [
                                          _jsx(X, {
                                            className: "h-8 w-8 text-destructive mx-auto",
                                          }),
                                          _jsx("p", {
                                            className: "text-xs text-muted-foreground",
                                            children: scanMessage,
                                          }),
                                          _jsx(Button, {
                                            size: "sm",
                                            variant: "outline",
                                            onClick: startScanLogin,
                                            children: "\u91CD\u65B0\u83B7\u53D6",
                                          }),
                                        ],
                                      })
                                    : _jsx(Loader2, {
                                        className: "h-10 w-10 animate-spin text-primary/40",
                                      }),
                              }),
                        ],
                      }),
                      _jsxs("div", {
                        className: "text-center space-y-1.5",
                        children: [
                          _jsxs("div", {
                            className: "flex items-center justify-center gap-2",
                            children: [
                              _jsx(QrCode, { className: "h-4 w-4 text-primary" }),
                              _jsx("p", {
                                className: "font-bold text-sm",
                                children: scanMessage || "正在加载...",
                              }),
                            ],
                          }),
                          _jsx("p", {
                            className:
                              "text-xs text-muted-foreground max-w-[260px] mx-auto leading-relaxed",
                            children:
                              scanStatus === "scanned"
                                ? "请在手机上确认登录"
                                : !infoData
                                  ? "打开微信，扫描二维码即可登录。"
                                  : registrationEnabled
                                    ? "打开微信，扫描二维码即可登录。首次使用会自动创建账号并绑定 Bot。"
                                    : "打开微信，扫描二维码即可登录。仅限已绑定的微信账号。",
                          }),
                        ],
                      }),
                    ],
                  }),
                  _jsxs("div", {
                    className: "relative my-6",
                    children: [
                      _jsx("div", {
                        className: "absolute inset-0 flex items-center",
                        children: _jsx(Separator, {}),
                      }),
                      _jsx("div", {
                        className:
                          "relative flex justify-center text-[10px] uppercase font-bold tracking-widest text-muted-foreground",
                        children: _jsx("span", {
                          className: "bg-card px-3",
                          children: "\u5176\u4ED6\u767B\u5F55\u65B9\u5F0F",
                        }),
                      }),
                    ],
                  }),
                  oauthProviders.length > 0 &&
                    _jsx("div", {
                      className: "space-y-2 mb-4",
                      children: oauthProviders.map((provider) => {
                        const config = providerLabels[provider.name] || {
                          label: provider.display_name || provider.name,
                          icon: Shield,
                        };
                        return _jsxs(
                          Button,
                          {
                            variant: "outline",
                            className: "w-full h-9 gap-2 font-medium text-sm",
                            onClick: () =>
                              (window.location.href =
                                provider.type === "oidc"
                                  ? `/api/auth/oidc/${provider.name}`
                                  : `/api/auth/oauth/${provider.name}`),
                            children: [
                              _jsx(config.icon, { className: "h-4 w-4" }),
                              "\u4F7F\u7528 ",
                              config.label,
                              " \u767B\u5F55",
                            ],
                          },
                          provider.name,
                        );
                      }),
                    }),
                  supportsPasskey &&
                    _jsxs(Button, {
                      type: "button",
                      variant: "outline",
                      className: "w-full h-9 gap-2 font-medium text-sm mb-4",
                      onClick: handlePasskeyLogin,
                      disabled: loading,
                      children: [
                        _jsx(KeyRound, { className: "h-4 w-4 text-primary" }),
                        "\u4F7F\u7528\u901A\u884C\u5BC6\u94A5\u767B\u5F55",
                      ],
                    }),
                  _jsxs(Collapsible, {
                    open: showPassword,
                    onOpenChange: setShowPassword,
                    children: [
                      _jsx(CollapsibleTrigger, {
                        asChild: true,
                        children: _jsxs("button", {
                          className:
                            "w-full flex items-center justify-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors py-1",
                          children: [
                            _jsx("span", {
                              children:
                                mode === "login"
                                  ? "账号密码登录"
                                  : registrationEnabled
                                    ? "注册新账号"
                                    : "账号密码登录",
                            }),
                            _jsx(ChevronDown, {
                              className: `h-3 w-3 transition-transform duration-200 ${showPassword ? "rotate-180" : ""}`,
                            }),
                          ],
                        }),
                      }),
                      _jsxs(CollapsibleContent, {
                        className: "pt-4 space-y-4 animate-in fade-in slide-in-from-top-2",
                        children: [
                          _jsxs("form", {
                            onSubmit: handleSubmit,
                            className: "space-y-3",
                            children: [
                              _jsxs("div", {
                                className: "relative",
                                children: [
                                  _jsx(User, {
                                    className:
                                      "absolute left-3 top-2.5 h-4 w-4 text-muted-foreground",
                                  }),
                                  _jsx(Input, {
                                    placeholder: "\u7528\u6237\u540D",
                                    className: "pl-10 h-9 bg-muted/20",
                                    value: username,
                                    onChange: (e) => setUsername(e.target.value),
                                    required: true,
                                    disabled: loading,
                                  }),
                                ],
                              }),
                              _jsxs("div", {
                                className: "relative",
                                children: [
                                  _jsx(Lock, {
                                    className:
                                      "absolute left-3 top-2.5 h-4 w-4 text-muted-foreground",
                                  }),
                                  _jsx(Input, {
                                    type: "password",
                                    placeholder: "\u767B\u5F55\u5BC6\u7801",
                                    className: "pl-10 h-9 bg-muted/20",
                                    value: password,
                                    onChange: (e) => setPassword(e.target.value),
                                    required: true,
                                    disabled: loading,
                                  }),
                                ],
                              }),
                              error &&
                                _jsxs("div", {
                                  className:
                                    "flex items-center gap-2 p-2.5 rounded-lg bg-destructive/10 text-destructive text-xs font-medium border border-destructive/20",
                                  children: [_jsx(X, { className: "h-3.5 w-3.5 shrink-0" }), error],
                                }),
                              _jsxs(Button, {
                                type: "submit",
                                className: "w-full h-9 font-bold text-sm",
                                disabled: loading,
                                children: [
                                  loading
                                    ? _jsx(Loader2, { className: "h-4 w-4 animate-spin mr-2" })
                                    : null,
                                  mode === "login" ? "登录" : "注册",
                                  !loading && _jsx(ArrowRight, { className: "ml-2 h-3.5 w-3.5" }),
                                ],
                              }),
                            ],
                          }),
                          registrationEnabled &&
                            _jsxs("div", {
                              className: "text-center text-xs text-muted-foreground",
                              children: [
                                mode === "login" ? "还没有账号？" : "已经有账号了？",
                                _jsx("button", {
                                  type: "button",
                                  className: "ml-1 font-bold text-primary hover:underline",
                                  onClick: () => {
                                    setMode(mode === "login" ? "register" : "login");
                                    setError("");
                                  },
                                  children: mode === "login" ? "立即注册" : "点击登录",
                                }),
                              ],
                            }),
                        ],
                      }),
                    ],
                  }),
                ],
              }),
              _jsx(CardFooter, {
                className: "border-t bg-muted/30 pt-4 pb-4 rounded-b-xl justify-center",
                children: _jsxs("p", {
                  className:
                    "text-[10px] text-center text-muted-foreground/60 leading-relaxed px-6",
                  children: [
                    "\u767B\u5F55\u5373\u4EE3\u8868\u60A8\u540C\u610F\u6211\u4EEC\u7684 ",
                    _jsx(Link, {
                      to: "#",
                      className: "underline",
                      children: "\u670D\u52A1\u6761\u6B3E",
                    }),
                    " \u548C ",
                    _jsx(Link, {
                      to: "#",
                      className: "underline",
                      children: "\u9690\u79C1\u653F\u7B56",
                    }),
                    "\u3002",
                  ],
                }),
              }),
            ],
          }),
          _jsx("footer", {
            className: "mt-8 text-center text-[11px] text-muted-foreground/50 font-medium",
            children:
              "\u00A9 2026 OpeniLink Hub \u9879\u76EE\u4FDD\u7559\u6240\u6709\u6743\u5229\u3002",
          }),
        ],
      }),
    ],
  });
}
