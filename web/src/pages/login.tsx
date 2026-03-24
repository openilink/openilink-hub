import { useEffect, useState } from "react";
import { KeyRound } from "lucide-react";
import { useNavigate } from "react-router-dom";

import { Button } from "../components/ui/button";
import { HexagonBackground } from "../components/ui/hexagon-background";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldSeparator,
} from "../components/ui/field";
import { Input } from "../components/ui/input";
import { api } from "../lib/api";

const providerLabels: Record<string, string> = {
  github: "GitHub",
  linuxdo: "LinuxDo",
};

const loginCopy = {
  brand: "OpeniLink Hub",
  welcome: "欢迎回来",
  createAccount: "创建你的账号",
  loginIntro: "登录后即可管理 Bot、渠道和 Webhook 插件。",
  registerIntro: "注册后就可以开始配置微信 Bot 和消息路由。",
  loginTitle: "登录你的账号",
  registerTitle: "创建新账号",
  continueWithAccount: "或继续使用账号",
  username: "用户名",
  usernamePlaceholder: "请输入用户名",
  password: "密码",
  passwordHint: "至少 8 位",
  loginPasswordPlaceholder: "请输入密码",
  registerPasswordPlaceholder: "设置登录密码",
  passkeyLoginSeparator: "或使用 Passkey",
  passkeyRegisterSeparator: "或直接创建 Passkey",
  passkeyLogin: "使用 Passkey 登录",
  passkeyRegister: "使用 Passkey 注册（无需密码）",
  passkeyRegisterHint: "Passkey 注册会使用上方填写的用户名创建账号。",
  login: "登录",
  register: "注册",
  noAccount: "没有账号？",
  hasAccount: "已有账号？",
  supportText: "支持密码、Passkey 和 OAuth 登录方式。",
  oauthLogin: (provider: string) => `使用 ${provider} 登录`,
  enterUsernameFirst: "请先输入用户名",
  passkeyLoginFailed: "Passkey 登录失败",
  passkeyRegisterFailed: "Passkey 注册失败",
  cancelled: "已取消",
  usernameAndPasswordRequired: "请输入用户名和密码",
  usernameLength: "用户名长度需为 2 到 32 个字符",
  passwordTooShort: "密码至少需要 8 位",
  usernameTaken: "用户名已被占用",
  invalidCredentials: "用户名或密码错误",
  accountDisabled: "账号已被禁用",
  invalidRequest: "请求无效",
  usernameRequired: "请输入用户名",
  userNotFound: "用户不存在",
  noRegistrationSession: "注册会话不存在",
  registrationFailed: "注册失败",
  loginFailed: "登录失败",
  registerFailed: "注册失败",
} as const;

function localizeAuthError(rawMessage: unknown, fallback: string = loginCopy.loginFailed): string {
  const original = typeof rawMessage === "string" ? rawMessage.trim() : "";
  const key = original.toLowerCase();
  const translations: Record<string, string> = {
    cancelled: loginCopy.cancelled,
    "username and password required": loginCopy.usernameAndPasswordRequired,
    "username must be 2-32 characters": loginCopy.usernameLength,
    "password must be at least 8 characters": loginCopy.passwordTooShort,
    "username already taken": loginCopy.usernameTaken,
    "invalid credentials": loginCopy.invalidCredentials,
    "account disabled": loginCopy.accountDisabled,
    "invalid request": loginCopy.invalidRequest,
    "username required": loginCopy.usernameRequired,
    "user not found": loginCopy.userNotFound,
    "no registration session": loginCopy.noRegistrationSession,
    "login failed": loginCopy.loginFailed,
    "register failed": loginCopy.registerFailed,
  };

  if (!key) return fallback;

  if (key.startsWith("registration failed:")) {
    const reason = original.slice("registration failed:".length).trim();
    return reason ? `${loginCopy.registrationFailed}：${localizeAuthError(reason)}` : loginCopy.registrationFailed;
  }

  return translations[key] || original;
}

function base64urlToBuffer(b64: string): ArrayBuffer {
  const base64 = b64.replace(/-/g, "+").replace(/_/g, "/");
  const pad = base64.length % 4 === 0 ? "" : "=".repeat(4 - (base64.length % 4));
  const bin = atob(base64 + pad);
  const bytes = new Uint8Array(bin.length);

  for (let i = 0; i < bin.length; i++) {
    bytes[i] = bin.charCodeAt(i);
  }

  return bytes.buffer;
}

function bufferToBase64url(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf);
  let bin = "";

  for (let i = 0; i < bytes.length; i++) {
    bin += String.fromCharCode(bytes[i]);
  }

  return btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=/g, "");
}

export function LoginPage() {
  const navigate = useNavigate();
  const copy = loginCopy;
  const [mode, setMode] = useState<"login" | "register">("login");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [oauthProviders, setOauthProviders] = useState<string[]>([]);

  useEffect(() => {
    api
      .oauthProviders()
      .then((data) => setOauthProviders(data.providers || []))
      .catch(() => {});
  }, []);

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
      setError(localizeAuthError(err?.message ?? err, mode === "register" ? copy.registerFailed : copy.loginFailed));
    }

    setLoading(false);
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
          (credential: any) => ({
            ...credential,
            id: base64urlToBuffer(credential.id),
          }),
        );
      }

      const credential = (await navigator.credentials.get(options)) as PublicKeyCredential;

      if (!credential) {
        throw new Error("cancelled");
      }

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
        throw new Error(data.error || "login failed");
      }

      navigate("/dashboard");
    } catch (err: any) {
      if (err.name !== "NotAllowedError") {
        setError(localizeAuthError(err?.message ?? err, copy.passkeyLoginFailed));
      }
    }

    setLoading(false);
  }

  async function handlePasskeyRegister() {
    if (!username.trim()) {
      setError(copy.enterUsernameFirst);
      return;
    }

    setError("");
    setLoading(true);

    try {
      const options = await fetch("/api/auth/passkey/register/begin", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username: username.trim() }),
      }).then(async (r) => {
        if (!r.ok) {
          throw new Error((await r.json()).error);
        }

        return r.json();
      });

      options.publicKey.challenge = base64urlToBuffer(options.publicKey.challenge);
      options.publicKey.user.id = base64urlToBuffer(options.publicKey.user.id);

      if (options.publicKey.excludeCredentials) {
        options.publicKey.excludeCredentials = options.publicKey.excludeCredentials.map(
          (credential: any) => ({
            ...credential,
            id: base64urlToBuffer(credential.id),
          }),
        );
      }

      const credential = (await navigator.credentials.create(options)) as PublicKeyCredential;

      if (!credential) {
        throw new Error("cancelled");
      }

      const response = credential.response as AuthenticatorAttestationResponse;
      const body = JSON.stringify({
        id: credential.id,
        rawId: bufferToBase64url(credential.rawId),
        type: credential.type,
        response: {
          attestationObject: bufferToBase64url(response.attestationObject),
          clientDataJSON: bufferToBase64url(response.clientDataJSON),
        },
      });

      const res = await fetch(
        `/api/auth/passkey/register/finish?username=${encodeURIComponent(username.trim())}`,
        {
          method: "POST",
          credentials: "same-origin",
          headers: { "Content-Type": "application/json" },
          body,
        },
      );

      if (!res.ok) {
        const data = await res.json();
        throw new Error(data.error || "register failed");
      }

      navigate("/dashboard");
    } catch (err: any) {
      if (err.name !== "NotAllowedError") {
        setError(localizeAuthError(err?.message ?? err, copy.passkeyRegisterFailed));
      }
    }

    setLoading(false);
  }

  function handleOAuth(provider: string) {
    window.location.href = `/api/auth/oauth/${provider}`;
  }

  function toggleMode() {
    setMode(mode === "login" ? "register" : "login");
    setError("");
  }

  const supportsPasskey = typeof window !== "undefined" && "PublicKeyCredential" in window;

  return (
    <div className="relative isolate flex min-h-screen items-center justify-center overflow-x-hidden bg-background px-6 py-12 sm:px-8 sm:py-16">
      <HexagonBackground className="opacity-55" hexagonSize={78} hexagonMargin={5} />
      <div className="absolute inset-x-0 top-0 h-[28rem] bg-[radial-gradient(circle_at_top,rgba(255,255,255,0.14),transparent_42%)]" />

      <div className="relative z-10 w-full max-w-md">
        <div className="flex flex-col gap-10">
          <div className="space-y-4 text-center">
            <p className="text-sm font-medium tracking-[0.18em] text-muted-foreground uppercase">
              {copy.brand}
            </p>
            <div className="space-y-3">
              <h1 className="text-3xl font-semibold tracking-tight sm:text-4xl">
                {mode === "login" ? copy.welcome : copy.createAccount}
              </h1>
              <p className="text-base leading-7 text-muted-foreground">
                {mode === "login" ? copy.loginIntro : copy.registerIntro}
              </p>
            </div>
          </div>

          <Card className="rounded-[1.75rem] border-white/8 bg-card/82 backdrop-blur-sm">
            <CardHeader className="px-6 pt-8 pb-4 text-center sm:px-8">
              <CardTitle className="text-2xl">{copy.brand}</CardTitle>
              <CardDescription>{mode === "login" ? copy.loginTitle : copy.registerTitle}</CardDescription>
            </CardHeader>
            <CardContent className="px-6 pb-8 sm:px-8">
              <form onSubmit={handleSubmit}>
                <FieldGroup className="gap-6">
                  {oauthProviders.length > 0 && (
                    <>
                      <Field>
                        {oauthProviders.map((provider) => (
                          <Button
                            key={provider}
                            type="button"
                            variant="outline"
                            className="h-10 w-full text-sm"
                            onClick={() => handleOAuth(provider)}
                            disabled={loading}
                          >
                            {copy.oauthLogin(providerLabels[provider] || provider)}
                          </Button>
                        ))}
                      </Field>
                      <FieldSeparator className="*:data-[slot=field-separator-content]:bg-card">
                        {copy.continueWithAccount}
                      </FieldSeparator>
                    </>
                  )}

                  <Field>
                    <FieldLabel htmlFor="username">{copy.username}</FieldLabel>
                    <Input
                      id="username"
                      placeholder={copy.usernamePlaceholder}
                      className="h-10 text-base"
                      value={username}
                      onChange={(e) => setUsername(e.target.value)}
                      autoFocus
                      disabled={loading}
                      required
                    />
                  </Field>

                  <Field>
                    <div className="flex items-center">
                      <FieldLabel htmlFor="password">{copy.password}</FieldLabel>
                      {mode === "register" && (
                        <span className="ml-auto text-sm text-muted-foreground">{copy.passwordHint}</span>
                      )}
                    </div>
                    <Input
                      id="password"
                      type="password"
                      placeholder={
                        mode === "login" ? copy.loginPasswordPlaceholder : copy.registerPasswordPlaceholder
                      }
                      className="h-10 text-base"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      disabled={loading}
                      required
                    />
                  </Field>

                  <FieldError>{error}</FieldError>

                  {supportsPasskey && (
                    <>
                      <FieldSeparator className="*:data-[slot=field-separator-content]:bg-card">
                        {mode === "login" ? copy.passkeyLoginSeparator : copy.passkeyRegisterSeparator}
                      </FieldSeparator>
                      <Field>
                        <Button
                          type="button"
                          variant="outline"
                          className="h-10 w-full text-sm"
                          onClick={mode === "login" ? handlePasskeyLogin : handlePasskeyRegister}
                          disabled={loading || (mode === "register" && !username.trim())}
                        >
                          <KeyRound className="mr-2 h-4 w-4" />
                          {mode === "login" ? copy.passkeyLogin : copy.passkeyRegister}
                        </Button>
                        {mode === "register" && (
                          <FieldDescription className="text-center">
                            {copy.passkeyRegisterHint}
                          </FieldDescription>
                        )}
                      </Field>
                    </>
                  )}

                  <Field>
                    <Button type="submit" className="w-full" disabled={loading}>
                      {loading ? "..." : mode === "login" ? copy.login : copy.register}
                    </Button>
                    <FieldDescription className="text-center">
                      {mode === "login" ? copy.noAccount : copy.hasAccount}{" "}
                      <button
                        type="button"
                        className="font-medium text-foreground underline underline-offset-4"
                        onClick={toggleMode}
                      >
                        {mode === "login" ? copy.register : copy.login}
                      </button>
                    </FieldDescription>
                  </Field>
                </FieldGroup>
              </form>
            </CardContent>
          </Card>

          <FieldDescription className="px-6 pt-1 text-center text-sm leading-6">
            {copy.supportText}
          </FieldDescription>
        </div>
      </div>
    </div>
  );
}
