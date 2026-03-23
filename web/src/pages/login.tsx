import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Card } from "../components/ui/card";
import { KeyRound } from "lucide-react";
import { api } from "../lib/api";

const providerLabels: Record<string, string> = {
  github: "GitHub",
  linuxdo: "LinuxDo",
};

// WebAuthn helpers
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

export function LoginPage() {
  const navigate = useNavigate();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [oauthProviders, setOauthProviders] = useState<string[]>([]);

  useEffect(() => {
    api.oauthProviders().then((data) => setOauthProviders(data.providers || [])).catch(() => {});
  }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(""); setLoading(true);
    try {
      if (mode === "register") {
        await api.register(username, password);
      } else {
        await api.login(username, password);
      }
      navigate("/dashboard");
    } catch (err: any) { setError(err.message); }
    setLoading(false);
  }

  async function handlePasskeyLogin() {
    setError(""); setLoading(true);
    try {
      const options = await fetch("/api/auth/passkey/login/begin", {
        method: "POST", credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      }).then(r => r.json());

      options.publicKey.challenge = base64urlToBuffer(options.publicKey.challenge);
      if (options.publicKey.allowCredentials) {
        options.publicKey.allowCredentials = options.publicKey.allowCredentials.map((c: any) => ({
          ...c, id: base64urlToBuffer(c.id),
        }));
      }

      const cred = await navigator.credentials.get(options) as PublicKeyCredential;
      if (!cred) throw new Error("cancelled");

      const response = cred.response as AuthenticatorAssertionResponse;
      const body = JSON.stringify({
        id: cred.id,
        rawId: bufferToBase64url(cred.rawId),
        type: cred.type,
        response: {
          authenticatorData: bufferToBase64url(response.authenticatorData),
          clientDataJSON: bufferToBase64url(response.clientDataJSON),
          signature: bufferToBase64url(response.signature),
          userHandle: response.userHandle ? bufferToBase64url(response.userHandle) : "",
        },
      });

      const res = await fetch("/api/auth/passkey/login/finish", {
        method: "POST", credentials: "same-origin",
        headers: { "Content-Type": "application/json" }, body,
      });
      if (!res.ok) {
        const data = await res.json();
        throw new Error(data.error || "login failed");
      }
      navigate("/dashboard");
    } catch (err: any) {
      if (err.name !== "NotAllowedError") setError(err.message || "Passkey 登录失败");
    }
    setLoading(false);
  }

  async function handlePasskeyRegister() {
    if (!username.trim()) { setError("请先输入用户名"); return; }
    setError(""); setLoading(true);
    try {
      const options = await fetch("/api/auth/passkey/register/begin", {
        method: "POST", credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username: username.trim() }),
      }).then(async r => { if (!r.ok) throw new Error((await r.json()).error); return r.json(); });

      options.publicKey.challenge = base64urlToBuffer(options.publicKey.challenge);
      options.publicKey.user.id = base64urlToBuffer(options.publicKey.user.id);
      if (options.publicKey.excludeCredentials) {
        options.publicKey.excludeCredentials = options.publicKey.excludeCredentials.map((c: any) => ({
          ...c, id: base64urlToBuffer(c.id),
        }));
      }

      const cred = await navigator.credentials.create(options) as PublicKeyCredential;
      if (!cred) throw new Error("cancelled");

      const response = cred.response as AuthenticatorAttestationResponse;
      const body = JSON.stringify({
        id: cred.id,
        rawId: bufferToBase64url(cred.rawId),
        type: cred.type,
        response: {
          attestationObject: bufferToBase64url(response.attestationObject),
          clientDataJSON: bufferToBase64url(response.clientDataJSON),
        },
      });

      const res = await fetch(`/api/auth/passkey/register/finish?username=${encodeURIComponent(username.trim())}`, {
        method: "POST", credentials: "same-origin",
        headers: { "Content-Type": "application/json" }, body,
      });
      if (!res.ok) {
        const data = await res.json();
        throw new Error(data.error || "register failed");
      }
      navigate("/dashboard");
    } catch (err: any) {
      if (err.name !== "NotAllowedError") setError(err.message || "Passkey 注册失败");
    }
    setLoading(false);
  }

  function handleOAuth(provider: string) {
    window.location.href = `/api/auth/oauth/${provider}`;
  }

  const supportsPasskey = !!window.PublicKeyCredential;

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <Card className="w-full max-w-sm space-y-6">
        <div className="text-center">
          <h1 className="text-xl font-semibold">OpenILink Hub</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {mode === "login" ? "登录你的账号" : "创建新账号"}
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <Input placeholder="用户名" value={username} onChange={(e) => setUsername(e.target.value)} autoFocus />
          <Input type="password" placeholder="密码 (至少8位)" value={password} onChange={(e) => setPassword(e.target.value)} />
          {error && <p className="text-sm text-destructive">{error}</p>}
          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? "..." : mode === "login" ? "登录" : "注册"}
          </Button>
        </form>

        {/* Passkey */}
        {supportsPasskey && (
          <>
            <div className="relative">
              <div className="absolute inset-0 flex items-center"><span className="w-full border-t" /></div>
              <div className="relative flex justify-center text-xs uppercase">
                <span className="bg-card px-2 text-muted-foreground">或</span>
              </div>
            </div>
            {mode === "login" ? (
              <Button type="button" variant="outline" className="w-full" onClick={handlePasskeyLogin} disabled={loading}>
                <KeyRound className="w-4 h-4 mr-2" /> 使用 Passkey 登录
              </Button>
            ) : (
              <Button type="button" variant="outline" className="w-full" onClick={handlePasskeyRegister} disabled={loading || !username.trim()}>
                <KeyRound className="w-4 h-4 mr-2" /> 使用 Passkey 注册（无需密码）
              </Button>
            )}
          </>
        )}

        {/* OAuth */}
        {oauthProviders.length > 0 && (
          <div className="space-y-2">
            {!supportsPasskey && (
              <div className="relative">
                <div className="absolute inset-0 flex items-center"><span className="w-full border-t" /></div>
                <div className="relative flex justify-center text-xs uppercase">
                  <span className="bg-card px-2 text-muted-foreground">或</span>
                </div>
              </div>
            )}
            {oauthProviders.map((provider) => (
              <Button key={provider} type="button" variant="outline" className="w-full" onClick={() => handleOAuth(provider)}>
                使用 {providerLabels[provider] || provider} 登录
              </Button>
            ))}
          </div>
        )}

        <p className="text-center text-sm text-muted-foreground">
          {mode === "login" ? "没有账号？" : "已有账号？"}
          <button type="button" className="text-primary ml-1 hover:underline cursor-pointer"
            onClick={() => { setMode(mode === "login" ? "register" : "login"); setError(""); }}>
            {mode === "login" ? "注册" : "登录"}
          </button>
        </p>
      </Card>
    </div>
  );
}
