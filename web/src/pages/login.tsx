import { useEffect, useRef, useState } from "react";
import { KeyRound, Shield, User, Lock, ArrowRight, Loader2, Github, X, QrCode, ChevronDown } from "lucide-react";
import { useNavigate, Link } from "react-router-dom";
import QRCode from "qrcode";

import { Button } from "../components/ui/button";
import { HexagonBackground } from "../components/ui/hexagon-background";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  CardFooter
} from "../components/ui/card";
import { Input } from "../components/ui/input";
import { api } from "../lib/api";
import { Separator } from "../components/ui/separator";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";

const providerLabels: Record<string, { label: string, icon: any }> = {
  github: { label: "GitHub", icon: Github },
  linuxdo: { label: "LinuxDo", icon: Shield },
};

function QrCanvas({ url }: { url: string }) {
  const ref = useRef<HTMLCanvasElement>(null);
  useEffect(() => {
    if (url && ref.current) QRCode.toCanvas(ref.current, url, { width: 224, margin: 0 });
  }, [url]);
  return <canvas ref={ref} className="block rounded-lg" />;
}

export function LoginPage() {
  const navigate = useNavigate();

  // Scan login state
  const [qrUrl, setQrUrl] = useState("");
  const [scanStatus, setScanStatus] = useState<"idle" | "loading" | "wait" | "scanned" | "error">("idle");
  const [scanMessage, setScanMessage] = useState("");

  // Password login state
  const [mode, setMode] = useState<"login" | "register">("login");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  // OAuth
  const [oauthProviders, setOauthProviders] = useState<string[]>([]);

  useEffect(() => {
    api.oauthProviders().then((data) => setOauthProviders(data.providers || [])).catch(() => {});
  }, []);

  // Auto-start scan login on mount
  useEffect(() => {
    startScanLogin();
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

      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      const ws = new WebSocket(`${protocol}//${window.location.host}/api/auth/scan/status/${data.session_id}`);
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
            if (d.session_token) {
              document.cookie = `session=${d.session_token}; path=/; max-age=${7*24*3600}; samesite=lax`;
            }
            ws.close();
            navigate(d.bot_id ? `/dashboard/onboarding?bot_id=${d.bot_id}` : "/dashboard");
          }
        } else if (d.event === "error") {
          setScanMessage(d.message || "扫码登录失败");
          setScanStatus("error");
          ws.close();
        }
      };
      ws.onerror = () => {
        setScanStatus("error");
        setScanMessage("连接中断，请刷新重试");
        ws.close();
      };
      ws.onclose = () => {};
    } catch (err: any) {
      setScanStatus("error");
      setScanMessage(err.message || "初始化失败");
    }
  }

  // Password login
  async function handleSubmit(e: React.FormEvent) {
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
    } catch (err: any) {
      setError(err.message);
    }
    setLoading(false);
  }

  // Passkey login
  function base64urlToBuffer(b64: string): ArrayBuffer {
    const base64 = b64.replace(/-/g, "+").replace(/_/g, "/");
    const pad = base64.length % 4 === 0 ? "" : "=".repeat(4 - (base64.length % 4));
    const bin = atob(base64 + pad);
    const bytes = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
    return bytes.buffer;
  }

  function bufferToBase64url(buf: ArrayBuffer): string {
    const bytes = new Uint8Array(buf);
    let bin = "";
    for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i]);
    return btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=/g, "");
  }

  async function handlePasskeyLogin() {
    setError(""); setLoading(true);
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
          (credential: any) => ({ ...credential, id: base64urlToBuffer(credential.id) }),
        );
      }
      const credential = (await navigator.credentials.get(options)) as PublicKeyCredential;
      if (!credential) throw new Error("cancelled");
      const response = credential.response as AuthenticatorAssertionResponse;
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
    } catch (err: any) {
      if (err.name !== "NotAllowedError") setError(err.message || "Passkey 登录失败");
    }
    setLoading(false);
  }

  const supportsPasskey = typeof window !== "undefined" && "PublicKeyCredential" in window;

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-background px-4 py-12">
      <HexagonBackground className="opacity-20" hexagonSize={60} hexagonMargin={4} />
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,transparent_0%,hsl(var(--background))_100%)]" />

      <div className="relative z-10 w-full max-w-[420px] animate-in fade-in zoom-in-95 duration-500">
        <div className="mb-8 text-center space-y-2">
          <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-2xl bg-primary text-primary-foreground shadow-lg shadow-primary/20 mb-4">
            <Shield className="h-6 w-6" />
          </div>
          <h1 className="text-3xl font-extrabold tracking-tight">OpeniLink Hub</h1>
          <p className="text-sm text-muted-foreground font-medium">
            微信扫码，一键登录
          </p>
        </div>

        <Card className="border-border/50 shadow-2xl backdrop-blur-md bg-card/80">
          <CardContent className="pt-8 pb-6">
            {/* Primary: QR Code Scan Login */}
            <div className="flex flex-col items-center gap-6">
              <div className="relative group">
                <div className="absolute -inset-4 bg-primary/5 rounded-[2rem] blur-xl group-hover:bg-primary/10 transition-all" />
                {qrUrl ? (
                  <div className="relative rounded-2xl border-4 border-background bg-white p-4 shadow-2xl">
                    <QrCanvas url={qrUrl} />
                  </div>
                ) : (
                  <div className="relative flex h-[224px] w-[224px] items-center justify-center rounded-2xl border-2 border-dashed bg-muted/30">
                    {scanStatus === "error" ? (
                      <div className="text-center space-y-3 px-4">
                        <X className="h-8 w-8 text-destructive mx-auto" />
                        <p className="text-xs text-muted-foreground">{scanMessage}</p>
                        <Button size="sm" variant="outline" onClick={startScanLogin}>重新获取</Button>
                      </div>
                    ) : (
                      <Loader2 className="h-10 w-10 animate-spin text-primary/40" />
                    )}
                  </div>
                )}
              </div>

              <div className="text-center space-y-1.5">
                <div className="flex items-center justify-center gap-2">
                  <QrCode className="h-4 w-4 text-primary" />
                  <p className="font-bold text-sm">{scanMessage || "正在加载..."}</p>
                </div>
                <p className="text-xs text-muted-foreground max-w-[260px] mx-auto leading-relaxed">
                  {scanStatus === "scanned"
                    ? "请在手机上确认登录"
                    : "打开微信，扫描二维码即可登录。首次使用会自动创建账号并绑定 Bot。"}
                </p>
              </div>

            </div>

            {/* Divider */}
            <div className="relative my-6">
              <div className="absolute inset-0 flex items-center"><Separator /></div>
              <div className="relative flex justify-center text-[10px] uppercase font-bold tracking-widest text-muted-foreground">
                <span className="bg-card px-3">其他登录方式</span>
              </div>
            </div>

            {/* Secondary: OAuth providers */}
            {oauthProviders.length > 0 && (
              <div className="space-y-2 mb-4">
                {oauthProviders.map((provider) => {
                  const config = providerLabels[provider] || { label: provider, icon: Shield };
                  return (
                    <Button
                      key={provider}
                      variant="outline"
                      className="w-full h-9 gap-2 font-medium text-sm"
                      onClick={() => (window.location.href = `/api/auth/oauth/${provider}`)}
                    >
                      <config.icon className="h-4 w-4" />
                      使用 {config.label} 登录
                    </Button>
                  );
                })}
              </div>
            )}

            {/* Passkey */}
            {supportsPasskey && (
              <Button
                type="button"
                variant="outline"
                className="w-full h-9 gap-2 font-medium text-sm mb-4"
                onClick={handlePasskeyLogin}
                disabled={loading}
              >
                <KeyRound className="h-4 w-4 text-primary" />
                使用通行密钥登录
              </Button>
            )}

            {/* Collapsible: Username/Password */}
            <Collapsible open={showPassword} onOpenChange={setShowPassword}>
              <CollapsibleTrigger asChild>
                <button className="w-full flex items-center justify-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors py-1">
                  <span>{mode === "login" ? "账号密码登录" : "注册新账号"}</span>
                  <ChevronDown className={`h-3 w-3 transition-transform duration-200 ${showPassword ? "rotate-180" : ""}`} />
                </button>
              </CollapsibleTrigger>
              <CollapsibleContent className="pt-4 space-y-4 animate-in fade-in slide-in-from-top-2">
                <form onSubmit={handleSubmit} className="space-y-3">
                  <div className="relative">
                    <User className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                    <Input
                      placeholder="用户名"
                      className="pl-10 h-9 bg-muted/20"
                      value={username}
                      onChange={(e) => setUsername(e.target.value)}
                      required
                      disabled={loading}
                    />
                  </div>
                  <div className="relative">
                    <Lock className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                    <Input
                      type="password"
                      placeholder="登录密码"
                      className="pl-10 h-9 bg-muted/20"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      required
                      disabled={loading}
                    />
                  </div>

                  {error && (
                    <div className="flex items-center gap-2 p-2.5 rounded-lg bg-destructive/10 text-destructive text-xs font-medium border border-destructive/20">
                      <X className="h-3.5 w-3.5 shrink-0" />
                      {error}
                    </div>
                  )}

                  <Button type="submit" className="w-full h-9 font-bold text-sm" disabled={loading}>
                    {loading ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
                    {mode === "login" ? "登录" : "注册"}
                    {!loading && <ArrowRight className="ml-2 h-3.5 w-3.5" />}
                  </Button>
                </form>

                <div className="text-center text-xs text-muted-foreground">
                  {mode === "login" ? "还没有账号？" : "已经有账号了？"}
                  <button
                    type="button"
                    className="ml-1 font-bold text-primary hover:underline"
                    onClick={() => { setMode(mode === "login" ? "register" : "login"); setError(""); }}
                  >
                    {mode === "login" ? "立即注册" : "点击登录"}
                  </button>
                </div>
              </CollapsibleContent>
            </Collapsible>
          </CardContent>
          <CardFooter className="border-t bg-muted/30 pt-4 pb-4 rounded-b-xl justify-center">
            <p className="text-[10px] text-center text-muted-foreground/60 leading-relaxed px-6">
              登录即代表您同意我们的 <Link to="#" className="underline">服务条款</Link> 和 <Link to="#" className="underline">隐私政策</Link>。
            </p>
          </CardFooter>
        </Card>

        <footer className="mt-8 text-center text-[11px] text-muted-foreground/50 font-medium">
          &copy; 2026 OpeniLink Hub 项目保留所有权利。
        </footer>
      </div>
    </div>
  );
}
