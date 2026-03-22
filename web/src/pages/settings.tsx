import { useEffect, useState } from "react";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Card } from "../components/ui/card";
import { api } from "../lib/api";
import { Link2, Unlink, Trash2, KeyRound, Plus, Sun, Moon, Monitor } from "lucide-react";
import { useTheme, type Theme } from "../lib/theme";

const providerLabels: Record<string, string> = { github: "GitHub", linuxdo: "LinuxDo" };

export function SettingsPage() {
  const [user, setUser] = useState<any>(null);
  const [oauthAccounts, setOauthAccounts] = useState<any[]>([]);
  const [oauthProviders, setOauthProviders] = useState<string[]>([]);
  const { theme, setTheme } = useTheme();

  async function load() {
    const [u, accounts, providers] = await Promise.all([api.me(), api.oauthAccounts(), api.oauthProviders()]);
    setUser(u); setOauthAccounts(accounts || []); setOauthProviders(providers.providers || []);
  }

  const [oauthMsg, setOauthMsg] = useState("");

  useEffect(() => { load(); }, []);
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const bound = params.get("oauth_bound");
    const error = params.get("oauth_error");
    if (bound) {
      setOauthMsg(`${providerLabels[bound] || bound} 绑定成功`);
    } else if (error === "already_linked") {
      setOauthMsg("绑定失败：该第三方账号已被其他用户绑定，请联系管理员处理");
    } else if (error === "bind_failed") {
      setOauthMsg("绑定失败，请重试");
    } else if (error) {
      setOauthMsg("OAuth 错误：" + error);
    }
    if (bound || error) {
      window.history.replaceState({}, "", "/dashboard/settings");
      load();
    }
  }, []);

  if (!user) return null;

  const themeOptions: { value: Theme; label: string; icon: React.ReactNode }[] = [
    { value: "light", label: "浅色", icon: <Sun className="w-3.5 h-3.5" /> },
    { value: "dark", label: "深色", icon: <Moon className="w-3.5 h-3.5" /> },
    { value: "system", label: "跟随系统", icon: <Monitor className="w-3.5 h-3.5" /> },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-lg font-semibold">账号设置</h1>
        <p className="text-xs text-muted-foreground mt-0.5">密码、Passkey、第三方绑定</p>
      </div>

      {oauthMsg && (
        <div className={`text-xs p-3 rounded-lg border ${oauthMsg.includes("失败") || oauthMsg.includes("错误") ? "border-destructive/30 bg-destructive/5 text-destructive" : "border-primary/30 bg-primary/5 text-primary"}`}>
          {oauthMsg}
          <button onClick={() => setOauthMsg("")} className="ml-2 underline cursor-pointer">关闭</button>
        </div>
      )}

      {/* Account info */}
      <Card className="space-y-3">
        <h3 className="text-sm font-medium">外观</h3>
        <div className="flex gap-2">
          {themeOptions.map(({ value, label, icon }) => (
            <Button
              key={value}
              variant={theme === value ? "default" : "outline"}
              size="sm"
              onClick={() => setTheme(value)}
            >
              {icon}
              {label}
            </Button>
          ))}
        </div>
      </Card>

      <Card className="space-y-3">
        <h3 className="text-sm font-medium">账号信息</h3>
        <div className="text-sm space-y-1">
          <p><span className="text-muted-foreground">用户名：</span>{user.username}</p>
          <p><span className="text-muted-foreground">显示名：</span>{user.display_name}</p>
          <p><span className="text-muted-foreground">角色：</span>{user.role === "superadmin" ? "超级管理员" : user.role === "admin" ? "管理员" : "成员"}</p>
        </div>
      </Card>

      <ChangePasswordSection />
      <PasskeySection />

      {/* OAuth binding */}
      {oauthProviders.length > 0 && (
        <Card className="space-y-3">
          <h3 className="text-sm font-medium">第三方账号绑定</h3>
          <div className="space-y-2">
            {oauthProviders.map((provider) => {
              const account = oauthAccounts.find((a) => a.provider === provider);
              const linked = !!account;
              return (
                <div key={provider} className="flex items-center justify-between p-3 rounded-lg border bg-background">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-full bg-secondary flex items-center justify-center">
                      <span className="text-xs font-medium">{(providerLabels[provider] || provider).charAt(0).toUpperCase()}</span>
                    </div>
                    <div>
                      <p className="text-sm font-medium">{providerLabels[provider] || provider}</p>
                      <p className="text-xs text-muted-foreground">{linked ? `已绑定：${account.username}` : "未绑定"}</p>
                    </div>
                  </div>
                  {linked ? (
                    <Button variant="ghost" size="sm" onClick={async () => {
                      if (!confirm(`解绑 ${providerLabels[provider]}？`)) return;
                      try { await api.unlinkOAuth(provider); load(); } catch (e: any) { alert(e.message); }
                    }}><Unlink className="w-3.5 h-3.5 mr-1" /> 解绑</Button>
                  ) : (
                    <Button variant="outline" size="sm" onClick={() => { window.location.href = `/api/me/linked-accounts/${provider}/bind`; }}>
                      <Link2 className="w-3.5 h-3.5 mr-1" /> 绑定
                    </Button>
                  )}
                </div>
              );
            })}
          </div>
        </Card>
      )}
    </div>
  );
}

// ==================== Change Password ====================

function ChangePasswordSection() {
  const [oldPwd, setOldPwd] = useState("");
  const [newPwd, setNewPwd] = useState("");
  const [confirmPwd, setConfirmPwd] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault(); setError(""); setSuccess("");
    if (newPwd.length < 8) { setError("新密码至少 8 位"); return; }
    if (newPwd !== confirmPwd) { setError("两次输入不一致"); return; }
    setSaving(true);
    try {
      await api.changePassword({ old_password: oldPwd, new_password: newPwd });
      setOldPwd(""); setNewPwd(""); setConfirmPwd(""); setSuccess("密码已修改");
    } catch (err: any) { setError(err.message); }
    setSaving(false);
  }

  return (
    <Card className="space-y-3">
      <h3 className="text-sm font-medium">修改密码</h3>
      <form onSubmit={handleSubmit} className="space-y-2">
        <Input type="password" placeholder="当前密码" value={oldPwd} onChange={(e) => setOldPwd(e.target.value)} className="h-8 text-xs" />
        <Input type="password" placeholder="新密码（至少 8 位）" value={newPwd} onChange={(e) => setNewPwd(e.target.value)} className="h-8 text-xs" />
        <Input type="password" placeholder="确认新密码" value={confirmPwd} onChange={(e) => setConfirmPwd(e.target.value)} className="h-8 text-xs" />
        <div className="flex items-center justify-between">
          <div>
            {error && <span className="text-[10px] text-destructive">{error}</span>}
            {success && <span className="text-[10px] text-primary">{success}</span>}
          </div>
          <Button type="submit" size="sm" disabled={saving}>{saving ? "..." : "修改密码"}</Button>
        </div>
      </form>
    </Card>
  );
}

// ==================== Passkey ====================

function PasskeySection() {
  const [passkeys, setPasskeys] = useState<any[]>([]);
  const [adding, setAdding] = useState(false);
  const [error, setError] = useState("");

  async function load() { try { setPasskeys(await api.listPasskeys() || []); } catch {} }
  useEffect(() => { load(); }, []);

  async function handleAdd() {
    setAdding(true); setError("");
    try {
      const options = await api.passkeyBindBegin();
      options.publicKey.challenge = base64urlToBuffer(options.publicKey.challenge);
      options.publicKey.user.id = base64urlToBuffer(options.publicKey.user.id);
      if (options.publicKey.excludeCredentials) {
        options.publicKey.excludeCredentials = options.publicKey.excludeCredentials.map((c: any) => ({ ...c, id: base64urlToBuffer(c.id) }));
      }
      const credential = await navigator.credentials.create(options) as PublicKeyCredential;
      if (!credential) throw new Error("cancelled");
      const response = credential.response as AuthenticatorAttestationResponse;
      await api.passkeyBindFinishRaw(JSON.stringify({
        id: credential.id, rawId: bufferToBase64url(credential.rawId), type: credential.type,
        response: { attestationObject: bufferToBase64url(response.attestationObject), clientDataJSON: bufferToBase64url(response.clientDataJSON) },
      }));
      load();
    } catch (err: any) {
      if (err.name !== "NotAllowedError") setError(err.message || "注册失败");
    }
    setAdding(false);
  }

  return (
    <Card className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium">Passkey</h3>
        <Button variant="outline" size="sm" className="text-xs h-7" onClick={handleAdd} disabled={adding}>
          <Plus className="w-3 h-3 mr-1" /> {adding ? "注册中..." : "添加 Passkey"}
        </Button>
      </div>
      <p className="text-xs text-muted-foreground">使用指纹、Face ID 或安全密钥登录，无需密码。</p>
      {error && <p className="text-[10px] text-destructive">{error}</p>}
      {passkeys.length === 0 ? (
        <p className="text-[10px] text-muted-foreground">暂未绑定任何 Passkey</p>
      ) : (
        <div className="space-y-1">
          {passkeys.map((pk) => (
            <div key={pk.id} className="flex items-center justify-between p-2 rounded-lg border bg-background">
              <div className="flex items-center gap-2">
                <KeyRound className="w-4 h-4 text-muted-foreground" />
                <div>
                  <p className="text-xs font-mono">{pk.id.slice(0, 16)}...</p>
                  <p className="text-[10px] text-muted-foreground">{new Date(pk.created_at * 1000).toLocaleDateString()}</p>
                </div>
              </div>
              <Button variant="ghost" size="sm" className="h-6" onClick={async () => {
                if (!confirm("删除此 Passkey？")) return;
                try { await api.deletePasskey(pk.id); load(); } catch {}
              }}><Trash2 className="w-3 h-3 text-destructive" /></Button>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}

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
