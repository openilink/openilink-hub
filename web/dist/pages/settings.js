import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
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
} from "lucide-react";
import { useTheme } from "../lib/theme";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "../components/ui/badge";
import { useUser } from "@/hooks/use-auth";
import {
  useOAuthAccounts,
  useOAuthProviders,
  usePasskeys,
  useDeletePasskey,
  useRenamePasskey,
  useUnlinkOAuth,
  useUpdateUsername,
} from "@/hooks/use-settings";
const THEME_OPTIONS = [
  { value: "light", label: "浅色", icon: Sun },
  { value: "dark", label: "深色", icon: Moon },
  { value: "system", label: "系统", icon: Monitor },
];
const providerLabels = {
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
    return _jsx("div", {
      className: "flex h-64 items-center justify-center",
      children: _jsx(Loader2, { className: "h-8 w-8 animate-spin text-muted-foreground" }),
    });
  return _jsxs("div", {
    className: "space-y-6",
    children: [
      ConfirmDialog,
      _jsxs("div", {
        children: [
          _jsx("h1", {
            className: "text-2xl font-bold tracking-tight",
            children: "\u8D26\u53F7\u8BBE\u7F6E",
          }),
          _jsx("p", {
            className: "text-sm text-muted-foreground mt-0.5",
            children:
              "\u7BA1\u7406\u60A8\u7684\u4E2A\u4EBA\u8D44\u6599\u3001\u5B89\u5168\u9009\u9879\u548C\u504F\u597D\u3002",
          }),
        ],
      }),
      oauthMsg
        ? _jsxs("div", {
            className: `flex items-center gap-3 p-4 rounded-xl border animate-in fade-in ${oauthMsg.includes("失败") ? "border-destructive/20 bg-destructive/5 text-destructive" : "border-primary/20 bg-primary/5 text-primary"}`,
            children: [
              _jsx("span", { className: "text-sm font-medium flex-1", children: oauthMsg }),
              _jsx(Button, {
                variant: "ghost",
                size: "xs",
                onClick: () => setOauthMsg(""),
                children: "\u5173\u95ED",
              }),
            ],
          })
        : null,
      _jsxs(Tabs, {
        value: activeTab,
        onValueChange: (v) => navigate(`/dashboard/settings/${v}`),
        className: "space-y-6",
        children: [
          _jsxs(TabsList, {
            className: "bg-muted/50 p-1",
            children: [
              _jsx(TabsTrigger, {
                value: "profile",
                className: "px-6",
                children: "\u4E2A\u4EBA\u8D44\u6599",
              }),
              _jsx(TabsTrigger, {
                value: "security",
                className: "px-6",
                children: "\u5B89\u5168\u8BA4\u8BC1",
              }),
            ],
          }),
          _jsxs(TabsContent, {
            value: "profile",
            className: "m-0 space-y-6",
            children: [
              _jsxs(Card, {
                className: "border-border/50",
                children: [
                  _jsx(CardHeader, {
                    children: _jsx(CardTitle, { children: "\u57FA\u672C\u4FE1\u606F" }),
                  }),
                  _jsx(CardContent, {
                    className: "space-y-4",
                    children: _jsxs("div", {
                      className: "grid gap-4 md:grid-cols-2",
                      children: [
                        _jsx(UsernameEditor, { username: user?.username }),
                        _jsxs("div", {
                          className: "space-y-2",
                          children: [
                            _jsx("label", {
                              className: "text-sm font-medium",
                              children: "\u89D2\u8272",
                            }),
                            _jsx("div", {
                              className: "pt-2",
                              children: _jsx(Badge, {
                                variant: "secondary",
                                className: "uppercase text-[10px] tracking-wider font-bold",
                                children: user?.role,
                              }),
                            }),
                          ],
                        }),
                      ],
                    }),
                  }),
                ],
              }),
              oauthProvidersError || oauthAccountsError
                ? _jsx(Card, {
                    className: "border-destructive/20",
                    children: _jsx(CardContent, {
                      className: "p-4 text-sm text-destructive",
                      children:
                        "\u7B2C\u4E09\u65B9\u8D26\u53F7\u4FE1\u606F\u52A0\u8F7D\u5931\u8D25",
                    }),
                  })
                : oauthProviders.length > 0
                  ? _jsxs(Card, {
                      className: "border-border/50",
                      children: [
                        _jsx(CardHeader, {
                          children: _jsx(CardTitle, { children: "\u7B2C\u4E09\u65B9\u7ED1\u5B9A" }),
                        }),
                        _jsx(CardContent, {
                          className: "space-y-3",
                          children: oauthProviders.map((provider) => {
                            const providerKey = provider.key || provider.name;
                            const account = oauthAccounts.find((a) => a.provider === providerKey);
                            const linked = !!account;
                            const Icon = providerLabels[provider.name]?.icon || ShieldCheck;
                            const label =
                              providerLabels[provider.name]?.label ||
                              provider.display_name ||
                              provider.name;
                            return _jsxs(
                              "div",
                              {
                                className:
                                  "flex items-center justify-between p-4 rounded-xl border bg-muted/10",
                                children: [
                                  " ",
                                  _jsxs("div", {
                                    className: "flex items-center gap-4",
                                    children: [
                                      _jsx("div", {
                                        className:
                                          "flex h-10 w-10 items-center justify-center rounded-full bg-background border shadow-sm",
                                        children: _jsx(Icon, { className: "h-5 w-5" }),
                                      }),
                                      _jsxs("div", {
                                        children: [
                                          _jsx("p", {
                                            className: "text-sm font-bold uppercase",
                                            children: label,
                                          }),
                                          _jsx("p", {
                                            className: "text-xs text-muted-foreground",
                                            children: linked
                                              ? `已关联：${account.username}`
                                              : "未连接",
                                          }),
                                        ],
                                      }),
                                    ],
                                  }),
                                  " ",
                                  linked
                                    ? _jsxs(Button, {
                                        variant: "ghost",
                                        size: "sm",
                                        className: "text-destructive",
                                        onClick: async () => {
                                          const ok = await confirm({
                                            title: "解绑确认",
                                            description: `确定要解绑 ${label}？`,
                                            confirmText: "解绑",
                                            variant: "destructive",
                                          });
                                          if (!ok) return;
                                          unlinkOAuth.mutate(providerKey, {
                                            onError: (e) => setOauthMsg(e.message),
                                          });
                                        },
                                        children: [
                                          _jsx(Unlink, { className: "h-3.5 w-3.5 mr-2" }),
                                          " \u89E3\u7ED1",
                                        ],
                                      })
                                    : _jsxs(Button, {
                                        variant: "outline",
                                        size: "sm",
                                        onClick: () =>
                                          (window.location.href =
                                            provider.type === "oidc"
                                              ? `/api/me/oidc/${provider.name}/bind`
                                              : `/api/me/linked-accounts/${provider.name}/bind`),
                                        children: [
                                          _jsx(Link2, { className: "h-3.5 w-3.5 mr-2" }),
                                          " \u7ED1\u5B9A",
                                        ],
                                      }),
                                ],
                              },
                              provider.name,
                            );
                          }),
                        }),
                      ],
                    })
                  : null,
              _jsxs(Card, {
                className: "border-border/50 max-w-2xl",
                children: [
                  _jsx(CardHeader, {
                    children: _jsx(CardTitle, { children: "\u754C\u9762\u5916\u89C2" }),
                  }),
                  _jsx(CardContent, {
                    children: _jsx("div", {
                      className: "grid grid-cols-3 gap-4",
                      children: THEME_OPTIONS.map((item) =>
                        _jsxs(
                          Button,
                          {
                            variant: "ghost",
                            onClick: () => setTheme(item.value),
                            className: `flex flex-col items-center gap-3 p-4 h-auto rounded-xl border transition-all ${theme === item.value ? "border-primary bg-primary/[0.03] ring-1 ring-primary" : "bg-muted/20 border-border/50"}`,
                            children: [
                              _jsx("div", {
                                className: `h-10 w-10 flex items-center justify-center rounded-full ${theme === item.value ? "bg-primary text-primary-foreground shadow-md" : "bg-background text-muted-foreground border"}`,
                                children: _jsx(item.icon, { className: "h-5 w-5" }),
                              }),
                              _jsx("p", { className: "text-xs font-bold", children: item.label }),
                            ],
                          },
                          item.value,
                        ),
                      ),
                    }),
                  }),
                ],
              }),
            ],
          }),
          _jsxs(TabsContent, {
            value: "security",
            className: "m-0 space-y-6",
            children: [
              _jsx(PasskeySection, {}),
              _jsx(ChangePasswordSection, { hasPassword: user?.has_password }),
            ],
          }),
        ],
      }),
    ],
  });
}
const usernameRegex = /^[a-z0-9][a-z0-9_-]*[a-z0-9]$/;
const reservedUsernames = new Set([
  "admin",
  "administrator",
  "superadmin",
  "root",
  "system",
  "api",
  "support",
]);
function UsernameEditor({ username }) {
  const [value, setValue] = useState(username || "");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const updateUsername = useUpdateUsername();
  useEffect(() => {
    if (username) setValue(username);
  }, [username]);
  const changed = value !== username;
  function validate(v) {
    if (v.length < 2 || v.length > 32) return "用户名长度需要 2-32 个字符";
    if (!usernameRegex.test(v))
      return "只能包含小写字母、数字、下划线和连字符，且不能以 _ 或 - 开头结尾";
    if (reservedUsernames.has(v)) return "该用户名为系统保留名称";
    return null;
  }
  async function handleSave() {
    setError("");
    setSuccess("");
    const err = validate(value);
    if (err) return setError(err);
    updateUsername.mutate(value, {
      onSuccess: () => setSuccess("用户名已更新。如使用密码登录，请使用新用户名。"),
      onError: (e) => setError(e.message || "修改失败"),
    });
  }
  return _jsxs("div", {
    className: "space-y-2",
    children: [
      _jsx("label", { className: "text-sm font-medium", children: "\u7528\u6237\u540D" }),
      _jsxs("div", {
        className: "flex gap-2",
        children: [
          _jsx(Input, {
            value: value,
            onChange: (e) => {
              setValue(e.target.value.toLowerCase());
              setError("");
              setSuccess("");
            },
            className: "font-mono",
            maxLength: 32,
            placeholder: "your-username",
          }),
          changed &&
            _jsx(Button, {
              size: "sm",
              className: "shrink-0",
              onClick: handleSave,
              disabled: updateUsername.isPending,
              children: updateUsername.isPending
                ? _jsx(Loader2, { className: "h-4 w-4 animate-spin" })
                : "保存",
            }),
        ],
      }),
      error
        ? _jsxs("p", {
            className: "text-xs text-destructive flex items-center gap-1",
            children: [_jsx(AlertCircle, { className: "h-3 w-3" }), " ", error],
          })
        : null,
      success
        ? _jsxs("p", {
            className: "text-xs text-green-600 flex items-center gap-1",
            children: [_jsx(Check, { className: "h-3 w-3" }), " ", success],
          })
        : null,
    ],
  });
}
function ChangePasswordSection({ hasPassword }) {
  const [oldPwd, setOldPwd] = useState("");
  const [newPwd, setNewPwd] = useState("");
  const [confirmPwd, setConfirmPwd] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  async function handleSubmit(e) {
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
    } catch (err) {
      setError(err.message);
    }
    setSaving(false);
  }
  return _jsxs(Card, {
    className: "border-border/50",
    children: [
      _jsxs(CardHeader, {
        children: [
          _jsx(CardTitle, { children: hasPassword ? "修改登录密码" : "设置登录密码" }),
          _jsx(CardDescription, {
            children: hasPassword
              ? "建议定期更换密码以增强安全性。"
              : "您还未设置密码。设置后可使用密码登录。",
          }),
        ],
      }),
      _jsx(CardContent, {
        children: _jsxs("form", {
          onSubmit: handleSubmit,
          className: "space-y-4 max-w-md",
          children: [
            hasPassword &&
              _jsxs("div", {
                className: "space-y-2",
                children: [
                  _jsx("label", {
                    htmlFor: "current-password",
                    className: "text-xs font-medium",
                    children: "\u5F53\u524D\u5BC6\u7801",
                  }),
                  _jsx(Input, {
                    id: "current-password",
                    name: "current-password",
                    type: "password",
                    autoComplete: "current-password",
                    value: oldPwd,
                    onChange: (e) => setOldPwd(e.target.value),
                    placeholder: "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022",
                  }),
                ],
              }),
            _jsxs("div", {
              className: "space-y-2",
              children: [
                _jsx("label", {
                  htmlFor: "new-password",
                  className: "text-xs font-medium",
                  children: "\u65B0\u5BC6\u7801",
                }),
                _jsx(Input, {
                  id: "new-password",
                  name: "new-password",
                  type: "password",
                  autoComplete: "new-password",
                  value: newPwd,
                  onChange: (e) => setNewPwd(e.target.value),
                  placeholder: "\u81F3\u5C11 8 \u4F4D",
                }),
              ],
            }),
            _jsxs("div", {
              className: "space-y-2",
              children: [
                _jsx("label", {
                  htmlFor: "confirm-password",
                  className: "text-xs font-medium",
                  children: "\u786E\u8BA4\u65B0\u5BC6\u7801",
                }),
                _jsx(Input, {
                  id: "confirm-password",
                  name: "confirm-password",
                  type: "password",
                  autoComplete: "new-password",
                  value: confirmPwd,
                  onChange: (e) => setConfirmPwd(e.target.value),
                  placeholder: "\u518D\u6B21\u8F93\u5165\u65B0\u5BC6\u7801",
                }),
              ],
            }),
            _jsxs("div", {
              className: "pt-2 flex flex-col gap-3",
              children: [
                error
                  ? _jsxs("p", {
                      className: "text-xs text-destructive font-medium flex items-center gap-1.5",
                      children: [_jsx(AlertCircle, { className: "h-3 w-3" }), " ", error],
                    })
                  : null,
                success
                  ? _jsxs("p", {
                      className: "text-xs text-green-600 font-medium flex items-center gap-1.5",
                      children: [_jsx(Check, { className: "h-3 w-3" }), " ", success],
                    })
                  : null,
                _jsxs(Button, {
                  type: "submit",
                  className: "w-full sm:w-fit",
                  disabled: saving || (hasPassword && !oldPwd) || !newPwd,
                  children: [
                    saving ? _jsx(Loader2, { className: "mr-2 h-4 w-4 animate-spin" }) : null,
                    hasPassword ? "更新密码" : "设置密码",
                  ],
                }),
              ],
            }),
          ],
        }),
      }),
    ],
  });
}
const isXiaomiDevice = () => /xiaomi|redmi|miui|hyperos/i.test(navigator.userAgent);
function PasskeyNameEditor({ passkey, onError }) {
  const renamePasskey = useRenamePasskey();
  const [editing, setEditing] = useState(!passkey.name);
  const [value, setValue] = useState(passkey.name || "Passkey");
  const inputRef = useRef(null);
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
    renamePasskey.mutate(
      { id: passkey.id, name: trimmed },
      {
        onSuccess: () => setEditing(false),
        onError: (e) => onError(e.message || "重命名失败"),
      },
    );
  }
  if (editing) {
    return _jsx(Input, {
      ref: inputRef,
      value: value,
      onChange: (e) => setValue(e.target.value),
      onBlur: save,
      onKeyDown: (e) => {
        if (e.key === "Enter") save();
        if (e.key === "Escape") {
          setValue(passkey.name || "Passkey");
          setEditing(false);
        }
      },
      className: "h-6 text-xs font-bold px-1.5 py-0 w-32 bg-muted/30",
      maxLength: 50,
      autoFocus: true,
    });
  }
  return _jsx("button", {
    className:
      "text-xs font-bold hover:underline decoration-dashed underline-offset-2 cursor-pointer text-left",
    onClick: () => setEditing(true),
    title: "\u70B9\u51FB\u4FEE\u6539\u540D\u79F0",
    children: passkey.name || passkey.id.slice(0, 12) + "...",
  });
}
function PasskeySection() {
  const { data: passkeys = [], refetch: refetchPasskeys } = usePasskeys();
  const deletePasskeyMut = useDeletePasskey();
  const [adding, setAdding] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [showXiaomiGuide, setShowXiaomiGuide] = useState(false);
  const { confirm, ConfirmDialog } = useConfirm();
  const supportsPasskey = typeof window !== "undefined" && "PublicKeyCredential" in window;
  async function handleAdd() {
    if (!supportsPasskey) return;
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
        options.publicKey.excludeCredentials = options.publicKey.excludeCredentials.map((c) => ({
          ...c,
          id: base64urlToBuffer(c.id),
        }));
      }
      const credential = await navigator.credentials.create(options);
      if (!credential) throw new Error("cancelled");
      const response = credential.response;
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
    } catch (err) {
      if (err.name !== "NotAllowedError") setError(err.message || "Passkey 注册失败");
    }
    setAdding(false);
  }
  return _jsxs(Card, {
    className: "border-border/50",
    children: [
      ConfirmDialog,
      _jsxs(CardHeader, {
        className: "flex flex-row items-start justify-between space-y-0",
        children: [
          _jsxs("div", {
            className: "space-y-1.5",
            children: [
              _jsxs(CardTitle, {
                className: "flex items-center gap-2",
                children: [
                  "\u901A\u884C\u5BC6\u94A5",
                  " ",
                  _jsx(Badge, {
                    className: "bg-primary/10 text-primary border-none text-[9px]",
                    children: "\u63A8\u8350",
                  }),
                ],
              }),
              _jsx(CardDescription, {
                children:
                  "\u4F7F\u7528\u751F\u7269\u8BC6\u522B\uFF08\u6307\u7EB9\u3001Face ID\uFF09\u6216\u5B89\u5168\u5BC6\u94A5\u8FDB\u884C\u767B\u5F55\uFF0C\u66F4\u5B89\u5168\u3001\u66F4\u5FEB\u6377\u3002",
              }),
            ],
          }),
          _jsxs(Button, {
            size: "sm",
            onClick: handleAdd,
            disabled: adding || !supportsPasskey,
            className: "h-9",
            title: supportsPasskey ? undefined : "需要 HTTPS 安全连接才能使用通行密钥",
            children: [
              adding
                ? _jsx(Loader2, { className: "mr-2 h-4 w-4 animate-spin" })
                : _jsx(Plus, { className: "mr-2 h-4 w-4" }),
              "\u6CE8\u518C Passkey",
            ],
          }),
        ],
      }),
      _jsxs(CardContent, {
        className: "space-y-4",
        children: [
          error
            ? _jsx("div", {
                className:
                  "text-xs p-3 rounded-lg bg-destructive/5 text-destructive border border-destructive/10",
                children: error,
              })
            : null,
          success
            ? _jsx("div", {
                className:
                  "text-xs p-3 rounded-lg bg-green-500/5 text-green-600 border border-green-500/10",
                children: success,
              })
            : null,
          !supportsPasskey
            ? _jsx("div", {
                className:
                  "text-xs p-3 rounded-lg bg-amber-500/5 text-amber-700 dark:text-amber-400 border border-amber-500/15",
                children:
                  "\u5F53\u524D\u73AF\u5883\u4E0D\u652F\u6301\u901A\u884C\u5BC6\u94A5\u3002\u8BF7\u901A\u8FC7 HTTPS \u5B89\u5168\u8FDE\u63A5\u8BBF\u95EE\u540E\u518D\u8BD5\u3002",
              })
            : null,
          showXiaomiGuide
            ? _jsxs("div", {
                className:
                  "text-xs p-4 rounded-lg bg-amber-500/5 text-amber-700 dark:text-amber-400 border border-amber-500/15 space-y-2.5",
                children: [
                  _jsxs("p", {
                    className: "font-bold flex items-center gap-1.5",
                    children: [
                      _jsx(AlertCircle, { className: "h-3.5 w-3.5" }),
                      "\u5C0F\u7C73 / \u7EA2\u7C73\u8BBE\u5907\u8BF7\u5148\u786E\u8BA4\u4EE5\u4E0B\u8BBE\u7F6E",
                    ],
                  }),
                  _jsxs("ol", {
                    className: "list-decimal ml-4 space-y-1.5 leading-relaxed",
                    children: [
                      _jsxs("li", {
                        children: [
                          "\u6253\u5F00 ",
                          _jsx("b", {
                            children:
                              "\u8BBE\u7F6E > \u6307\u7EB9\u3001\u9762\u90E8\u4E0E\u5BC6\u7801 > \u667A\u80FD\u5BC6\u7801\u7BA1\u7406",
                          }),
                          "\uFF0C",
                          _jsx("b", { children: "\u5173\u95ED" }),
                          '"\u81EA\u52A8\u586B\u5145\u5BC6\u7801\u4E0E\u901A\u884C\u5BC6\u94A5"',
                        ],
                      }),
                      _jsxs("li", {
                        children: [
                          "\u6253\u5F00 ",
                          _jsx("b", {
                            children:
                              "\u8BBE\u7F6E > \u66F4\u591A\u8BBE\u7F6E > \u8BED\u8A00\u4E0E\u8F93\u5165\u6CD5 > \u5BC6\u7801\u4E0E\u8D26\u53F7",
                          }),
                          '\uFF0C\u5C06"\u9996\u9009\u670D\u52A1"\u8BBE\u4E3A ',
                          _jsx("b", { children: "Google" }),
                          " \u6216 ",
                          _jsx("b", {
                            children: "\u5C0F\u7C73\u667A\u80FD\u5BC6\u7801\u7BA1\u7406",
                          }),
                        ],
                      }),
                      _jsx("li", {
                        children:
                          "\u786E\u4FDD Google Play \u670D\u52A1\u5DF2\u66F4\u65B0\u5230\u6700\u65B0\u7248\u672C",
                      }),
                    ],
                  }),
                  _jsx("p", {
                    className: "text-[10px] text-muted-foreground",
                    children:
                      "\u8BBE\u7F6E\u5B8C\u6210\u540E\uFF0C\u70B9\u51FB\u4E0B\u65B9\u6309\u94AE\u7EE7\u7EED\u6CE8\u518C\u3002\u5982\u679C\u6CE8\u518C\u540E\u65E0\u6CD5\u767B\u5F55\uFF0C\u8BF7\u68C0\u67E5\u5BC6\u7801\u7BA1\u7406\u5668\u4E2D\u662F\u5426\u6709\u4FDD\u5B58\u7684\u901A\u884C\u5BC6\u94A5\u3002",
                  }),
                  _jsxs("div", {
                    className: "flex gap-2 pt-1",
                    children: [
                      _jsx(Button, {
                        size: "sm",
                        className: "h-7 text-xs",
                        onClick: handleAdd,
                        children: "\u5DF2\u786E\u8BA4\uFF0C\u7EE7\u7EED\u6CE8\u518C",
                      }),
                      _jsx(Button, {
                        size: "sm",
                        variant: "ghost",
                        className: "h-7 text-xs",
                        onClick: () => setShowXiaomiGuide(false),
                        children: "\u53D6\u6D88",
                      }),
                    ],
                  }),
                ],
              })
            : null,
          passkeys.length === 0
            ? _jsxs("div", {
                className:
                  "flex flex-col items-center justify-center py-8 text-center border rounded-xl bg-muted/5 border-dashed",
                children: [
                  _jsx(Fingerprint, {
                    className: "h-10 w-10 text-muted-foreground opacity-20 mb-3",
                  }),
                  _jsx("p", {
                    className: "text-sm text-muted-foreground",
                    children: "\u60A8\u5C1A\u672A\u7ED1\u5B9A\u4EFB\u4F55 Passkey \u8BBE\u5907",
                  }),
                ],
              })
            : _jsx("div", {
                className: "grid gap-3 sm:grid-cols-2",
                children: passkeys.map((pk) =>
                  _jsxs(
                    "div",
                    {
                      className:
                        "flex items-center justify-between p-4 rounded-xl border bg-background group hover:border-primary/50 transition-colors",
                      children: [
                        _jsxs("div", {
                          className: "flex items-center gap-3",
                          children: [
                            _jsx("div", {
                              className:
                                "h-9 w-9 flex items-center justify-center rounded-lg bg-primary/5 text-primary",
                              children: _jsx(Smartphone, { className: "h-5 w-5" }),
                            }),
                            _jsxs("div", {
                              children: [
                                _jsx(PasskeyNameEditor, {
                                  passkey: pk,
                                  onError: (msg) => setError(msg),
                                }),
                                _jsxs("p", {
                                  className:
                                    "text-[10px] text-muted-foreground flex items-center gap-1.5 uppercase font-medium",
                                  children: [
                                    _jsx(Clock, { className: "h-2.5 w-2.5" }),
                                    " ",
                                    new Date(pk.created_at * 1000).toLocaleDateString(),
                                    " \u7ED1\u5B9A",
                                  ],
                                }),
                              ],
                            }),
                          ],
                        }),
                        _jsx(Button, {
                          variant: "ghost",
                          size: "icon",
                          className:
                            "h-8 w-8 text-destructive opacity-0 group-hover:opacity-100 transition-opacity",
                          onClick: async () => {
                            const ok = await confirm({
                              title: "删除确认",
                              description: "确定要删除此 Passkey 吗？",
                              confirmText: "删除",
                              variant: "destructive",
                            });
                            if (!ok) return;
                            setSuccess("");
                            deletePasskeyMut.mutate(pk.id, {
                              onError: (e) => setError(e.message || "删除失败"),
                            });
                          },
                          children: _jsx(Trash2, { className: "h-4 w-4" }),
                        }),
                      ],
                    },
                    pk.id,
                  ),
                ),
              }),
        ],
      }),
    ],
  });
}
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
