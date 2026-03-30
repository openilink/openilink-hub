import { useEffect, useRef, useState } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { api } from "../lib/api";
import { useConfirm } from "@/components/ui/confirm-dialog";
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
  Copy,
  Pencil,
  RefreshCw,
  Radio,
} from "lucide-react";
import { useTheme, type Theme } from "../lib/theme";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "../components/ui/badge";
import { useUser } from "@/hooks/use-auth";
import { useOAuthAccounts, useOAuthProviders, usePasskeys, useDeletePasskey, useRenamePasskey, useUnlinkOAuth } from "@/hooks/use-settings";
import { useBots } from "@/hooks/use-bots";
import { botDisplayName } from "@/lib/api";
import {
  useBroadcastTokens,
  useCreateBroadcastToken,
  useUpdateBroadcastToken,
  useDeleteBroadcastToken,
  useRegenerateBroadcastToken,
} from "@/hooks/use-broadcast-tokens";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { toast } from "@/hooks/use-toast";

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
  const { data: user, isLoading: userLoading } = useUser();
  const { data: oauthAccounts = [], isError: oauthAccountsError } = useOAuthAccounts();
  const { data: oauthProviders = [], isError: oauthProvidersError } = useOAuthProviders();
  const unlinkOAuth = useUnlinkOAuth();
  const { theme, setTheme } = useTheme();
  const { confirm, ConfirmDialog } = useConfirm();

  const activeTab = location.pathname.split("/").pop() || "profile";

  const [oauthMsg, setOauthMsg] = useState("");

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const bound = params.get("oauth_bound");
    const error = params.get("oauth_error");
    if (bound) setOauthMsg(`${providerLabels[bound]?.label || bound} 绑定成功`);
    else if (error === "already_linked") setOauthMsg("该第三方账号已被其他用户绑定");
    if (bound || error) {
      window.history.replaceState({}, "", "/dashboard/settings/profile");
    }
  }, []);

  if (userLoading && !user)
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );

  return (
    <div className="space-y-6">
      {ConfirmDialog}
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
          <TabsTrigger value="broadcast" className="px-6">
            广播
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
                  <Input value={user?.username} disabled className="bg-muted/30 font-mono" />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium">角色</label>
                  <div className="pt-2">
                    <Badge
                      variant="secondary"
                      className="uppercase text-[10px] tracking-wider font-bold"
                    >
                      {user?.role}
                    </Badge>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
          {oauthProvidersError || oauthAccountsError ? (
            <Card className="border-destructive/20">
              <CardContent className="p-4 text-sm text-destructive">第三方账号信息加载失败</CardContent>
            </Card>
          ) : oauthProviders.length > 0 ? (
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>第三方绑定</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {oauthProviders.map((provider) => {
                  const providerKey = provider.key || provider.name;
                  const account = oauthAccounts.find((a) => a.provider === providerKey);
                  const linked = !!account;
                  const Icon = providerLabels[provider.name]?.icon || ShieldCheck;
                  const label = providerLabels[provider.name]?.label || provider.display_name || provider.name;
                  return (
                    <div
                      key={provider.name}
                      className="flex items-center justify-between p-4 rounded-xl border bg-muted/10"
                    >
                      {" "}
                      <div className="flex items-center gap-4">
                        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-background border shadow-sm">
                          <Icon className="h-5 w-5" />
                        </div>
                        <div>
                          <p className="text-sm font-bold uppercase">
                            {label}
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
                            const ok = await confirm({
                              title: "解绑确认",
                              description: `确定要解绑 ${label}？`,
                              confirmText: "解绑",
                              variant: "destructive",
                            });
                            if (!ok) return;
                            unlinkOAuth.mutate(providerKey, {
                              onError: (e: any) => setOauthMsg(e.message),
                            });
                          }}
                        >
                          <Unlink className="h-3.5 w-3.5 mr-2" /> 解绑
                        </Button>
                      ) : (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() =>
                            (window.location.href = provider.type === "oidc" ? `/api/me/oidc/${provider.name}/bind` : `/api/me/linked-accounts/${provider.name}/bind`)
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
          <ChangePasswordSection hasPassword={user?.has_password} />
        </TabsContent>

        <TabsContent value="broadcast" className="m-0 space-y-6">
          <BroadcastTokenSection />
        </TabsContent>
      </Tabs>
    </div>
  );
}

// ... keep ChangePasswordSection and PasskeySection same ...
function ChangePasswordSection({ hasPassword }: { hasPassword?: boolean }) {
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
      setSuccess(hasPassword ? "您的登录密码已成功更新。" : "登录密码已设置成功。");
    } catch (err: any) {
      setError(err.message);
    }
    setSaving(false);
  }

  return (
    <Card className="border-border/50">
      <CardHeader>
        <CardTitle>{hasPassword ? "修改登录密码" : "设置登录密码"}</CardTitle>
        <CardDescription>
          {hasPassword
            ? "建议定期更换密码以增强安全性。"
            : "您还未设置密码。设置后可使用密码登录。"}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4 max-w-md">
          {hasPassword && (
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
          )}
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
              disabled={saving || (hasPassword && !oldPwd) || !newPwd}
            >
              {saving ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
              {hasPassword ? "更新密码" : "设置密码"}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}

const isXiaomiDevice = () => /xiaomi|redmi|miui|hyperos/i.test(navigator.userAgent);

function PasskeyNameEditor({ passkey, onError }: {
  passkey: { id: string; name: string };
  onError: (msg: string) => void;
}) {
  const renamePasskey = useRenamePasskey();
  const [editing, setEditing] = useState(!passkey.name);
  const [value, setValue] = useState(passkey.name || "Passkey");
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (editing) inputRef.current?.select();
  }, [editing]);

  async function save() {
    const trimmed = value.trim();
    if (!trimmed || trimmed === passkey.name) {
      setValue(passkey.name || "Passkey");
      setEditing(false);
      return;
    }
    renamePasskey.mutate({ id: passkey.id, name: trimmed }, {
      onSuccess: () => setEditing(false),
      onError: (e: any) => onError(e.message || "重命名失败"),
    });
  }

  if (editing) {
    return (
      <Input
        ref={inputRef}
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onBlur={save}
        onKeyDown={(e) => {
          if (e.key === "Enter") save();
          if (e.key === "Escape") { setValue(passkey.name || "Passkey"); setEditing(false); }
        }}
        className="h-6 text-xs font-bold px-1.5 py-0 w-32 bg-muted/30"
        maxLength={50}
        autoFocus
      />
    );
  }

  return (
    <button
      className="text-xs font-bold hover:underline decoration-dashed underline-offset-2 cursor-pointer text-left"
      onClick={() => setEditing(true)}
      title="点击修改名称"
    >
      {passkey.name || passkey.id.slice(0, 12) + "..."}
    </button>
  );
}

function PasskeySection() {
  const { data: passkeys = [], refetch: refetchPasskeys } = usePasskeys();
  const deletePasskeyMut = useDeletePasskey();
  const [adding, setAdding] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [showXiaomiGuide, setShowXiaomiGuide] = useState(false);
  const { confirm, ConfirmDialog } = useConfirm();

  async function handleAdd() {
    if (isXiaomiDevice() && !showXiaomiGuide) {
      setShowXiaomiGuide(true);
      return;
    }
    setAdding(true);
    setError("");
    setSuccess("");
    setShowXiaomiGuide(false);
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
      );
      await refetchPasskeys();
      setSuccess("通行密钥注册成功！点击名称可修改。建议退出后尝试使用通行密钥登录以确认可用。");
    } catch (err: any) {
      if (err.name !== "NotAllowedError") setError(err.message || "Passkey 注册失败");
    }
    setAdding(false);
  }

  return (
    <Card className="border-border/50">
      {ConfirmDialog}
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

        {success ? (
          <div className="text-xs p-3 rounded-lg bg-green-500/5 text-green-600 border border-green-500/10">
            {success}
          </div>
        ) : null}

        {showXiaomiGuide ? (
          <div className="text-xs p-4 rounded-lg bg-amber-500/5 text-amber-700 dark:text-amber-400 border border-amber-500/15 space-y-2.5">
            <p className="font-bold flex items-center gap-1.5">
              <AlertCircle className="h-3.5 w-3.5" />
              小米 / 红米设备请先确认以下设置
            </p>
            <ol className="list-decimal ml-4 space-y-1.5 leading-relaxed">
              <li>打开 <b>设置 &gt; 指纹、面部与密码 &gt; 智能密码管理</b>，<b>关闭</b>"自动填充密码与通行密钥"</li>
              <li>打开 <b>设置 &gt; 更多设置 &gt; 语言与输入法 &gt; 密码与账号</b>，将"首选服务"设为 <b>Google</b> 或 <b>小米智能密码管理</b></li>
              <li>确保 Google Play 服务已更新到最新版本</li>
            </ol>
            <p className="text-[10px] text-muted-foreground">设置完成后，点击下方按钮继续注册。如果注册后无法登录，请检查密码管理器中是否有保存的通行密钥。</p>
            <div className="flex gap-2 pt-1">
              <Button size="sm" className="h-7 text-xs" onClick={handleAdd}>
                已确认，继续注册
              </Button>
              <Button size="sm" variant="ghost" className="h-7 text-xs" onClick={() => setShowXiaomiGuide(false)}>
                取消
              </Button>
            </div>
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
                    <PasskeyNameEditor
                      passkey={pk}
                      onError={(msg) => setError(msg)}
                    />
                    <p className="text-[10px] text-muted-foreground flex items-center gap-1.5 uppercase font-medium">
                      <Clock className="h-2.5 w-2.5" />{" "}
                      {new Date(pk.created_at * 1000).toLocaleDateString()} 绑定
                    </p>
                  </div>
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8 text-destructive opacity-0 group-hover:opacity-100 transition-opacity"
                  onClick={async () => {
                    const ok = await confirm({
                      title: "删除确认",
                      description: "确定要删除此 Passkey 吗？",
                      confirmText: "删除",
                      variant: "destructive",
                    });
                    if (!ok) return;
                    setSuccess("");
                    deletePasskeyMut.mutate(pk.id, {
                      onError: (e: any) => setError(e.message || "删除失败"),
                    });
                  }}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function maskToken(token: string): string {
  if (token.length <= 11) return token;
  return token.slice(0, 7) + "..." + token.slice(-4);
}

function BroadcastTokenSection() {
  const { data: tokens = [] } = useBroadcastTokens();
  const { data: bots = [] } = useBots();
  const createMut = useCreateBroadcastToken();
  const updateMut = useUpdateBroadcastToken();
  const deleteMut = useDeleteBroadcastToken();
  const regenerateMut = useRegenerateBroadcastToken();
  const { confirm, ConfirmDialog } = useConfirm();

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingToken, setEditingToken] = useState<{ id: string; name: string; bot_ids: string[] } | null>(null);
  const [formName, setFormName] = useState("");
  const [formBotIds, setFormBotIds] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);

  // Token reveal dialog (after create)
  const [revealOpen, setRevealOpen] = useState(false);
  const [revealedToken, setRevealedToken] = useState("");
  const [copied, setCopied] = useState(false);

  function openCreate() {
    setEditingToken(null);
    setFormName("");
    setFormBotIds([]);
    setDialogOpen(true);
  }

  function openEdit(token: { id: string; name: string; bot_ids: string[] }) {
    setEditingToken(token);
    setFormName(token.name);
    setFormBotIds(token.bot_ids || []);
    setDialogOpen(true);
  }

  async function handleSave() {
    if (!formName.trim()) return;
    setSaving(true);
    try {
      if (editingToken) {
        await updateMut.mutateAsync({ id: editingToken.id, data: { name: formName.trim(), bot_ids: formBotIds } });
        toast({ title: "广播令牌已更新" });
      } else {
        const result = await createMut.mutateAsync({ name: formName.trim(), bot_ids: formBotIds });
        setRevealedToken(result.token);
        setCopied(false);
        setRevealOpen(true);
      }
      setDialogOpen(false);
    } catch (err: any) {
      toast({ title: "操作失败", description: err.message, variant: "destructive" });
    }
    setSaving(false);
  }

  async function handleRegenerate(token: any) {
    const ok = await confirm({
      title: "重新生成令牌",
      description: `确定要重新生成「${token.name}」的令牌吗？当前令牌将立即失效。`,
      confirmText: "重新生成",
      variant: "destructive",
    });
    if (!ok) return;
    try {
      const result = await regenerateMut.mutateAsync(token.id);
      setRevealedToken(result.token);
      setCopied(false);
      setRevealOpen(true);
    } catch (err: any) {
      toast({ title: "重新生成失败", description: err.message, variant: "destructive" });
    }
  }

  async function handleDelete(token: any) {
    const ok = await confirm({
      title: "删除确认",
      description: `确定要删除广播令牌「${token.name}」吗？此操作不可撤销。`,
      confirmText: "删除",
      variant: "destructive",
    });
    if (!ok) return;
    try {
      await deleteMut.mutateAsync(token.id);
      toast({ title: "广播令牌已删除" });
    } catch (err: any) {
      toast({ title: "删除失败", description: err.message, variant: "destructive" });
    }
  }

  function toggleBot(botId: string) {
    setFormBotIds((prev) =>
      prev.includes(botId) ? prev.filter((id) => id !== botId) : [...prev, botId]
    );
  }

  async function copyToClipboard(text: string) {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast({ title: "复制失败", variant: "destructive" });
    }
  }

  return (
    <Card className="border-border/50">
      {ConfirmDialog}
      <CardHeader className="flex flex-row items-start justify-between space-y-0">
        <div className="space-y-1.5">
          <CardTitle className="flex items-center gap-2">广播令牌</CardTitle>
          <CardDescription>
            广播令牌可以同时向多个 Bot 发送消息。
          </CardDescription>
        </div>
        <Button size="sm" onClick={openCreate} className="h-9">
          <Plus className="mr-2 h-4 w-4" />
          创建广播令牌
        </Button>
      </CardHeader>
      <CardContent className="space-y-4">
        {tokens.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-center border rounded-xl bg-muted/5 border-dashed">
            <Radio className="h-10 w-10 text-muted-foreground opacity-20 mb-3" />
            <p className="text-sm text-muted-foreground">您尚未创建任何广播令牌</p>
          </div>
        ) : (
          <div className="grid gap-3">
            {tokens.map((tk: { id: string; name: string; token: string; bot_ids: string[]; created_at: number }) => (
              <div
                key={tk.id}
                className="flex items-center justify-between p-4 rounded-xl border bg-background group hover:border-primary/50 transition-colors"
              >
                <div className="flex items-center gap-3 min-w-0 flex-1">
                  <div className="h-9 w-9 flex-shrink-0 flex items-center justify-center rounded-lg bg-primary/5 text-primary">
                    <Radio className="h-5 w-5" />
                  </div>
                  <div className="min-w-0">
                    <p className="text-xs font-bold truncate">{tk.name}</p>
                    <div className="flex items-center gap-3 text-[10px] text-muted-foreground uppercase font-medium mt-0.5">
                      <span className="font-mono">{maskToken(tk.token)}</span>
                      <button
                        className="hover:text-foreground transition-colors"
                        onClick={() => { navigator.clipboard.writeText(tk.token).then(() => toast({ title: "令牌已复制" })).catch(() => toast({ title: "复制失败", variant: "destructive" })); }}
                        title="复制令牌"
                      >
                        <Copy className="h-2.5 w-2.5" />
                      </button>
                      <span>{(tk.bot_ids || []).length} 个 Bot</span>
                      <span className="flex items-center gap-1">
                        <Clock className="h-2.5 w-2.5" />
                        {new Date(tk.created_at * 1000).toLocaleDateString()}
                      </span>
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0">
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    onClick={() => openEdit(tk)}
                    title="编辑"
                  >
                    <Pencil className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    onClick={() => handleRegenerate(tk)}
                    title="重新生成令牌"
                  >
                    <RefreshCw className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 text-destructive"
                    onClick={() => handleDelete(tk)}
                    title="删除"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>

      {/* Create / Edit Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingToken ? "编辑广播令牌" : "创建广播令牌"}</DialogTitle>
            <DialogDescription>
              {editingToken ? "修改广播令牌名称和关联的 Bot。" : "创建一个新的广播令牌，选择要关联的 Bot。"}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label>名称</Label>
              <Input
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                placeholder="输入令牌名称"
                autoFocus
              />
            </div>
            <div className="space-y-2">
              <Label>关联 Bot</Label>
              {bots.length === 0 ? (
                <p className="text-xs text-muted-foreground">暂无可用 Bot</p>
              ) : (
                <div className="max-h-48 overflow-y-auto border rounded-lg p-2 space-y-1">
                  {bots.map((bot: any) => (
                    <label
                      key={bot.id}
                      className="flex items-center gap-2 p-2 rounded-lg hover:bg-muted/50 cursor-pointer text-sm"
                    >
                      <input
                        type="checkbox"
                        checked={formBotIds.includes(bot.id)}
                        onChange={() => toggleBot(bot.id)}
                        className="rounded"
                      />
                      <span>{botDisplayName(bot)}</span>
                    </label>
                  ))}
                </div>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={handleSave} disabled={saving || !formName.trim()}>
              {saving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {editingToken ? "保存" : "创建"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Token Reveal Dialog */}
      <Dialog open={revealOpen} onOpenChange={setRevealOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>令牌已生成</DialogTitle>
            <DialogDescription>
              请立即复制此令牌，关闭后将无法再次查看完整令牌。
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3 py-2">
            <div className="flex items-center gap-2">
              <Input
                value={revealedToken}
                readOnly
                className="font-mono text-xs bg-muted/30"
              />
              <Button
                variant="outline"
                size="icon"
                className="flex-shrink-0"
                onClick={() => copyToClipboard(revealedToken)}
              >
                {copied ? <Check className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}
              </Button>
            </div>
            <div className="text-xs p-3 rounded-lg bg-amber-500/5 text-amber-700 dark:text-amber-400 border border-amber-500/15">
              <AlertCircle className="h-3.5 w-3.5 inline mr-1.5" />
              此令牌仅显示一次，请妥善保管。
            </div>
          </div>
          <DialogFooter>
            <Button onClick={() => setRevealOpen(false)}>确定</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
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
