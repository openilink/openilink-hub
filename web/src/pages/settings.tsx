import { useEffect, useState } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { api } from "../lib/api";
import {
  Link2,
  Unlink,
  Trash2,
  Plus,
  Sun,
  Moon,
  Monitor,
  ShieldCheck,
  Github,
  Check,
  AlertCircle,
  Loader2,
  Smartphone,
  Fingerprint,
  Clock,
  Pencil,
} from "lucide-react";
import { useTheme, type Theme } from "../lib/theme";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "../components/ui/badge";

const THEME_OPTIONS = [
  { value: "light", label: "浅色", icon: Sun },
  { value: "dark", label: "深色", icon: Moon },
  { value: "system", label: "系统", icon: Monitor },
] as const;

const providerLabels: Record<string, { label: string; icon: any }> = {
  github: { label: "GitHub", icon: Github },
  linuxdo: { label: "LinuxDo", icon: ShieldCheck },
};

export function SettingsPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const [user, setUser] = useState<any>(null);
  const [oauthAccounts, setOauthAccounts] = useState<any[]>([]);
  const [oauthProviders, setOauthProviders] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const { theme, setTheme } = useTheme();

  const activeTab = location.pathname.split("/").pop() || "profile";

  async function load() {
    setLoading(true);
    try {
      const [u, accounts, providers] = await Promise.all([
        api.me(),
        api.oauthAccounts(),
        api.oauthProviders(),
      ]);
      setUser(u);
      setOauthAccounts(accounts || []);
      setOauthProviders(providers.providers || []);
    } finally {
      setLoading(false);
    }
  }

  const [oauthMsg, setOauthMsg] = useState("");

  useEffect(() => {
    load();
  }, []);
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const bound = params.get("oauth_bound");
    const error = params.get("oauth_error");
    if (bound) setOauthMsg(`${providerLabels[bound]?.label || bound} 绑定成功`);
    else if (error === "already_linked") setOauthMsg("该第三方账号已被其他用户绑定");
    if (bound || error) {
      window.history.replaceState({}, "", "/dashboard/settings/profile");
      load();
    }
  }, []);

  if (loading && !user)
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">账号设置</h1>
        <p className="text-sm text-muted-foreground mt-0.5">管理您的个人资料、安全选项和偏好。</p>
      </div>

      {oauthMsg ? (
        <div
          className={`flex items-center gap-3 p-4 rounded-xl border animate-in fade-in ${oauthMsg.includes("失败") ? "border-destructive/20 bg-destructive/5 text-destructive" : "border-primary/20 bg-primary/5 text-primary"}`}
        >
          <span className="text-sm font-medium flex-1">{oauthMsg}</span>
          <Button variant="ghost" size="xs" onClick={() => setOauthMsg("")}>
            关闭
          </Button>
        </div>
      ) : null}

      <Tabs
        value={activeTab}
        onValueChange={(v: string) => navigate(`/dashboard/settings/${v}`)}
        className="space-y-6"
      >
        <TabsList className="bg-muted/50 p-1">
          <TabsTrigger value="profile" className="px-6">
            个人资料
          </TabsTrigger>
          <TabsTrigger value="security" className="px-6">
            安全认证
          </TabsTrigger>
        </TabsList>

        <TabsContent value="profile" className="m-0 space-y-6">
          <Card className="border-border/50">
            <CardHeader>
              <CardTitle>基本信息</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <label className="text-sm font-medium">用户名</label>
                  <Input value={user.username} disabled className="bg-muted/30 font-mono" />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium">角色</label>
                  <div className="pt-2">
                    <Badge
                      variant="secondary"
                      className="uppercase text-[10px] tracking-wider font-bold"
                    >
                      {user.role}
                    </Badge>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
          {oauthProviders.length > 0 ? (
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>第三方绑定</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {oauthProviders.map((provider) => {
                  const account = oauthAccounts.find((a) => a.provider === provider);
                  const linked = !!account;
                  const Icon = providerLabels[provider]?.icon || ShieldCheck;
                  return (
                    <div
                      key={provider}
                      className="flex items-center justify-between p-4 rounded-xl border bg-muted/10"
                    >
                      {" "}
                      <div className="flex items-center gap-4">
                        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-background border shadow-sm">
                          <Icon className="h-5 w-5" />
                        </div>
                        <div>
                          <p className="text-sm font-bold uppercase">
                            {providerLabels[provider]?.label || provider}
                          </p>
                          <p className="text-xs text-muted-foreground">
                            {linked ? `已关联：${account.username}` : "未连接"}
                          </p>
                        </div>
                      </div>{" "}
                      {linked ? (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-destructive"
                          onClick={async () => {
                            if (!confirm(`解绑？`)) return;
                            try {
                              await api.unlinkOAuth(provider);
                              load();
                            } catch (e: any) {
                              alert(e.message);
                            }
                          }}
                        >
                          <Unlink className="h-3.5 w-3.5 mr-2" /> 解绑
                        </Button>
                      ) : (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() =>
                            (window.location.href = `/api/me/linked-accounts/${provider}/bind`)
                          }
                        >
                          <Link2 className="h-3.5 w-3.5 mr-2" /> 绑定
                        </Button>
                      )}
                    </div>
                  );
                })}
              </CardContent>
            </Card>
          ) : null}
          <Card className="border-border/50 max-w-2xl">
            <CardHeader>
              <CardTitle>界面外观</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-3 gap-4">
                {THEME_OPTIONS.map((item) => (
                  <Button
                    key={item.value}
                    variant="ghost"
                    onClick={() => setTheme(item.value as Theme)}
                    className={`flex flex-col items-center gap-3 p-4 h-auto rounded-xl border transition-all ${theme === item.value ? "border-primary bg-primary/[0.03] ring-1 ring-primary" : "bg-muted/20 border-border/50"}`}
                  >
                    <div
                      className={`h-10 w-10 flex items-center justify-center rounded-full ${theme === item.value ? "bg-primary text-primary-foreground shadow-md" : "bg-background text-muted-foreground border"}`}
                    >
                      <item.icon className="h-5 w-5" />
                    </div>
                    <p className="text-xs font-bold">{item.label}</p>
                  </Button>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="security" className="m-0 space-y-6">
          <PasskeySection />
          <ChangePasswordSection />
        </TabsContent>
      </Tabs>
    </div>
  );
}

// ... keep ChangePasswordSection and PasskeySection same ...
function ChangePasswordSection() {
  const [oldPwd, setOldPwd] = useState("");
  const [newPwd, setNewPwd] = useState("");
  const [confirmPwd, setConfirmPwd] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setSuccess("");
    if (newPwd.length < 8) return setError("新密码长度至少需要 8 位");
    if (newPwd !== confirmPwd) return setError("两次输入的密码不一致");
    setSaving(true);
    try {
      await api.changePassword({ old_password: oldPwd, new_password: newPwd });
      setOldPwd("");
      setNewPwd("");
      setConfirmPwd("");
      setSuccess("您的登录密码已成功更新。");
    } catch (err: any) {
      setError(err.message);
    }
    setSaving(false);
  }

  return (
    <Card className="border-border/50">
      <CardHeader>
        <CardTitle>修改登录密码</CardTitle>
        <CardDescription>建议定期更换密码以增强安全性。</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4 max-w-md">
          <div className="space-y-2">
            <label htmlFor="current-password" className="text-xs font-medium">
              当前密码
            </label>
            <Input
              id="current-password"
              name="current-password"
              type="password"
              autoComplete="current-password"
              value={oldPwd}
              onChange={(e) => setOldPwd(e.target.value)}
              placeholder="••••••••"
            />
          </div>
          <div className="space-y-2">
            <label htmlFor="new-password" className="text-xs font-medium">
              新密码
            </label>
            <Input
              id="new-password"
              name="new-password"
              type="password"
              autoComplete="new-password"
              value={newPwd}
              onChange={(e) => setNewPwd(e.target.value)}
              placeholder="至少 8 位"
            />
          </div>
          <div className="space-y-2">
            <label htmlFor="confirm-password" className="text-xs font-medium">
              确认新密码
            </label>
            <Input
              id="confirm-password"
              name="confirm-password"
              type="password"
              autoComplete="new-password"
              value={confirmPwd}
              onChange={(e) => setConfirmPwd(e.target.value)}
              placeholder="再次输入新密码"
            />
          </div>

          <div className="pt-2 flex flex-col gap-3">
            {error ? (
              <p className="text-xs text-destructive font-medium flex items-center gap-1.5">
                <AlertCircle className="h-3 w-3" /> {error}
              </p>
            ) : null}
            {success ? (
              <p className="text-xs text-green-600 font-medium flex items-center gap-1.5">
                <Check className="h-3 w-3" /> {success}
              </p>
            ) : null}
            <Button
              type="submit"
              className="w-full sm:w-fit"
              disabled={saving || !oldPwd || !newPwd}
            >
              {saving ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
              更新密码
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}

function PasskeySection() {
  const [passkeys, setPasskeys] = useState<any[]>([]);
  const [adding, setAdding] = useState(false);
  const [error, setError] = useState("");

  async function load() {
    try {
      setPasskeys((await api.listPasskeys()) || []);
    } catch {
      setPasskeys([]);
    }
  }
  useEffect(() => {
    load();
  }, []);

  async function handleAdd() {
    const name = window.prompt("为此通行密钥命名（如：工作电脑、iPhone）", "Passkey");
    if (name === null) return; // user cancelled
    setAdding(true);
    setError("");
    try {
      const options = await api.passkeyBindBegin();
      options.publicKey.challenge = base64urlToBuffer(options.publicKey.challenge);
      options.publicKey.user.id = base64urlToBuffer(options.publicKey.user.id);
      if (options.publicKey.excludeCredentials) {
        options.publicKey.excludeCredentials = options.publicKey.excludeCredentials.map(
          (c: any) => ({ ...c, id: base64urlToBuffer(c.id) }),
        );
      }
      const credential = (await navigator.credentials.create(options)) as PublicKeyCredential;
      if (!credential) throw new Error("cancelled");
      const response = credential.response as AuthenticatorAttestationResponse;
      await api.passkeyBindFinishRaw(
        JSON.stringify({
          id: credential.id,
          rawId: bufferToBase64url(credential.rawId),
          type: credential.type,
          response: {
            attestationObject: bufferToBase64url(response.attestationObject),
            clientDataJSON: bufferToBase64url(response.clientDataJSON),
          },
        }),
        name || "Passkey",
      );
      load();
    } catch (err: any) {
      if (err.name !== "NotAllowedError") setError(err.message || "Passkey 注册失败");
    }
    setAdding(false);
  }

  return (
    <Card className="border-border/50">
      <CardHeader className="flex flex-row items-start justify-between space-y-0">
        <div className="space-y-1.5">
          <CardTitle className="flex items-center gap-2">
            通行密钥{" "}
            <Badge className="bg-primary/10 text-primary border-none text-[9px]">推荐</Badge>
          </CardTitle>
          <CardDescription>
            使用生物识别（指纹、Face ID）或安全密钥进行登录，更安全、更快捷。
          </CardDescription>
        </div>
        <Button size="sm" onClick={handleAdd} disabled={adding} className="h-9">
          {adding ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Plus className="mr-2 h-4 w-4" />
          )}
          注册 Passkey
        </Button>
      </CardHeader>
      <CardContent className="space-y-4">
        {error ? (
          <div className="text-xs p-3 rounded-lg bg-destructive/5 text-destructive border border-destructive/10">
            {error}
          </div>
        ) : null}

        {passkeys.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-center border rounded-xl bg-muted/5 border-dashed">
            <Fingerprint className="h-10 w-10 text-muted-foreground opacity-20 mb-3" />
            <p className="text-sm text-muted-foreground">您尚未绑定任何 Passkey 设备</p>
          </div>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2">
            {passkeys.map((pk) => (
              <div
                key={pk.id}
                className="flex items-center justify-between p-4 rounded-xl border bg-background group hover:border-primary/50 transition-colors"
              >
                <div className="flex items-center gap-3">
                  <div className="h-9 w-9 flex items-center justify-center rounded-lg bg-primary/5 text-primary">
                    <Smartphone className="h-5 w-5" />
                  </div>
                  <div>
                    <p className="text-xs font-bold">{pk.name || pk.id.slice(0, 12) + "..."}</p>
                    <p className="text-[10px] text-muted-foreground flex items-center gap-1.5 uppercase font-medium">
                      <Clock className="h-2.5 w-2.5" />{" "}
                      {new Date(pk.created_at * 1000).toLocaleDateString()} 绑定
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-1">
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity"
                    onClick={async () => {
                      const newName = window.prompt("重命名通行密钥", pk.name || "");
                      if (!newName) return;
                      try {
                        await api.renamePasskey(pk.id, newName);
                        load();
                      } catch (e: any) {
                        setError(e.message || "重命名失败");
                      }
                    }}
                  >
                    <Pencil className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 text-destructive opacity-0 group-hover:opacity-100 transition-opacity"
                    onClick={async () => {
                      if (!confirm("确定要删除此 Passkey 吗？")) return;
                      try {
                        await api.deletePasskey(pk.id);
                        load();
                      } catch (e: any) {
                        setError(e.message || "删除失败");
                      }
                    }}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
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
