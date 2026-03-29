export interface Bot {
  id: string;
  name: string;
  display_name: string;
  provider: string;
  status: string;
  can_send: boolean;
  send_disabled_reason?: string;
  ai_enabled: boolean;
  msg_count: number;
  reminder_hours: number;
  last_msg_at?: number;
  last_reminded_at?: number;
  created_at: number;
  extra?: Record<string, any>;
}

export function botDisplayName(bot: Pick<Bot, "display_name" | "name">): string {
  return bot.display_name || bot.name;
}

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    credentials: "same-origin",
    headers: { "Content-Type": "application/json", ...options?.headers },
    ...options,
  });
  if (res.status === 401) {
    const path = window.location.pathname;
    const isPublic = path === "/";
    if (!isPublic) {
      window.location.href = "/login";
    }
    throw new Error("unauthorized");
  }
  let data: any;
  try {
    data = await res.json();
  } catch {
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    throw new Error("invalid response");
  }
  if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
  return data as T;
}

export const api = {
  // Auth
  register: (username: string, password: string) =>
    request("/api/auth/register", { method: "POST", body: JSON.stringify({ username, password }) }),
  login: (username: string, password: string) =>
    request("/api/auth/login", { method: "POST", body: JSON.stringify({ username, password }) }),
  logout: () => request("/api/auth/logout", { method: "POST" }),
  oauthProviders: () => request<{ providers: string[] }>("/api/auth/oauth/providers"),
  me: () =>
    request<{ id: string; username: string; display_name: string; role: string }>("/api/me"),
  info: () => request<{ ai: boolean; registration_enabled: boolean; version: string }>("/api/info"),

  // Passkeys
  listPasskeys: () => request<any[]>("/api/me/passkeys"),
  passkeyBindBegin: () => request<any>("/api/me/passkeys/register/begin", { method: "POST" }),
  passkeyBindFinishRaw: (body: string, name?: string) =>
    fetch(`/api/me/passkeys/register/finish${name ? `?name=${encodeURIComponent(name)}` : ''}`, {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body,
    }).then(async (r) => {
      if (!r.ok) throw new Error((await r.json()).error);
    }),
  deletePasskey: (id: string) => request(`/api/me/passkeys/${id}`, { method: "DELETE" }),
  renamePasskey: (id: string, name: string) =>
    request(`/api/me/passkeys/${id}`, { method: "PATCH", body: JSON.stringify({ name }) }),

  // Profile
  updateProfile: (data: { display_name?: string; email?: string }) =>
    request("/api/me/profile", { method: "PUT", body: JSON.stringify(data) }),
  changePassword: (data: { old_password: string; new_password: string }) =>
    request("/api/me/password", { method: "PUT", body: JSON.stringify(data) }),

  // Bots
  listBots: () => request<Bot[]>("/api/bots"),
  bindStart: () =>
    request<{ session_id: string; qr_url: string }>("/api/bots/bind/start", { method: "POST" }),
  reconnectBot: (id: string) => request(`/api/bots/${id}/reconnect`, { method: "POST" }),
  deleteBot: (id: string) => request(`/api/bots/${id}`, { method: "DELETE" }),
  listBotApps: (botId: string) => request<any[]>(`/api/bots/${botId}/apps`),
  listTraces: (botId: string, limit = 50) =>
    request<import("./trace-utils").TraceSpan[]>(`/api/bots/${botId}/traces?limit=${limit}`),
  getTrace: (botId: string, traceId: string) =>
    request<import("./trace-utils").TraceSpan[]>(`/api/bots/${botId}/traces/${traceId}`),
  updateBot: (id: string, data: { name?: string; display_name?: string; reminder_hours?: number }) =>
    request(`/api/bots/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  setBotAI: (botId: string, enabled: boolean) =>
    request(`/api/bots/${botId}/ai`, {
      method: "PUT",
      body: JSON.stringify({ enabled }),
    }),
  setBotAIModel: (botId: string, model: string) =>
    request(`/api/bots/${botId}/ai_model`, {
      method: "PUT",
      body: JSON.stringify({ model }),
    }),
  botContacts: (id: string) => request<any[]>(`/api/bots/${id}/contacts`),

  // Channels (under bots)
  listChannels: (botId: string) => request<any[]>(`/api/bots/${botId}/channels`),
  createChannel: (botId: string, name: string, handle?: string) =>
    request(`/api/bots/${botId}/channels`, {
      method: "POST",
      body: JSON.stringify({ name, handle: handle || "" }),
    }),
  updateChannel: (botId: string, id: string, data: any) =>
    request(`/api/bots/${botId}/channels/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteChannel: (botId: string, id: string) =>
    request(`/api/bots/${botId}/channels/${id}`, { method: "DELETE" }),
  rotateKey: (botId: string, id: string) =>
    request<{ api_key: string }>(`/api/bots/${botId}/channels/${id}/rotate_key`, {
      method: "POST",
    }),

  // OAuth accounts
  oauthAccounts: () => request<any[]>("/api/me/linked-accounts"),
  unlinkOAuth: (provider: string) =>
    request(`/api/me/linked-accounts/${provider}`, { method: "DELETE" }),

  // Stats
  stats: () => request<any>("/api/bots/stats"),

  // Messages (under bots)
  messages: (botId: string, limit = 30, cursor?: string) =>
    request<{ messages: any[]; next_cursor: string; has_more: boolean; can_send?: boolean; send_disabled_reason?: string }>(
      `/api/bots/${botId}/messages?limit=${limit}${cursor ? "&cursor=" + cursor : ""}`,
    ),
  sendMessage: (botId: string, data: any) =>
    request(`/api/bots/${botId}/send`, { method: "POST", body: JSON.stringify(data) }),

  // Admin: system config
  getOAuthConfig: () => request<Record<string, any>>("/api/admin/config/oauth"),
  setOAuthConfig: (provider: string, data: { client_id: string; client_secret: string }) =>
    request(`/api/admin/config/oauth/${provider}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteOAuthConfig: (provider: string) =>
    request(`/api/admin/config/oauth/${provider}`, { method: "DELETE" }),

  // Public: available models list (all authenticated users)
  getAvailableModels: () => request<string[]>("/api/config/ai/available_models"),

  // Admin: AI config
  getAIConfig: () => request<any>("/api/admin/config/ai"),
  setAIConfig: (data: {
    base_url?: string;
    api_key?: string;
    model?: string;
    system_prompt?: string;
    max_history?: string;
    hide_thinking?: string;
    strip_markdown?: string;
    available_models?: string;
  }) => request("/api/admin/config/ai", { method: "PUT", body: JSON.stringify(data) }),
  deleteAIConfig: () => request("/api/admin/config/ai", { method: "DELETE" }),

  // Apps
  createApp: (data: any) =>
    request<any>("/api/apps", { method: "POST", body: JSON.stringify(data) }),
  listApps: (opts?: { listing?: string }) =>
    request<any[]>(`/api/apps${opts?.listing ? `?listing=${opts.listing}` : ""}`),
  getApp: (id: string) => request<any>(`/api/apps/${id}`),
  updateApp: (id: string, data: any) =>
    request<any>(`/api/apps/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  verifyAppUrl: (appId: string) =>
    request<any>(`/api/apps/${appId}/verify-url`, { method: "POST" }),
  deleteApp: (id: string) => request(`/api/apps/${id}`, { method: "DELETE" }),

  // Admin: Apps
  adminListApps: () => request<any[]>("/api/admin/apps"),
  setAppListing: (id: string, listing: string) =>
    request(`/api/admin/apps/${id}/listing`, { method: "PUT", body: JSON.stringify({ listing }) }),

  // App Installations
  installApp: (appId: string, data: any) =>
    request<any>(`/api/apps/${appId}/install`, { method: "POST", body: JSON.stringify(data) }),
  listInstallations: (appId: string) => request<any[]>(`/api/apps/${appId}/installations`),
  getInstallation: (appId: string, iid: string) =>
    request<any>(`/api/apps/${appId}/installations/${iid}`),
  updateInstallation: (appId: string, iid: string, data: any) =>
    request<any>(`/api/apps/${appId}/installations/${iid}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),
  deleteInstallation: (appId: string, iid: string) =>
    request(`/api/apps/${appId}/installations/${iid}`, { method: "DELETE" }),
  regenerateToken: (appId: string, iid: string) =>
    request<any>(`/api/apps/${appId}/installations/${iid}/regenerate-token`, { method: "POST" }),
  listEventLogs: (appId: string, iid: string, limit = 50) =>
    request<any[]>(`/api/apps/${appId}/installations/${iid}/event-logs?limit=${limit}`),
  listApiLogs: (appId: string, iid: string, limit = 50) =>
    request<any[]>(`/api/apps/${appId}/installations/${iid}/api-logs?limit=${limit}`),

  // Listing
  requestListing: (appId: string) =>
    request(`/api/apps/${appId}/request-listing`, { method: "POST" }),
  reviewListing: (appId: string, approve: boolean, reason?: string) =>
    request(`/api/admin/apps/${appId}/review-listing`, {
      method: "PUT",
      body: JSON.stringify({ approve, reason: reason || "" }),
    }),
  listAppReviews: (appId: string) =>
    request<any[]>(`/api/apps/${appId}/reviews`),

  // Webhook logs
  webhookLogs: (botId: string, channelId?: string, limit = 50) =>
    request<any[]>(
      `/api/bots/${botId}/webhook-logs?limit=${limit}${channelId ? "&channel_id=" + channelId : ""}`,
    ),

  // Marketplace
  getMarketplaceApps: () => request<any[]>("/api/marketplace"),
  getBuiltinApps: () => request<any[]>("/api/marketplace/builtin"),
  syncMarketplaceApp: (slug: string) => request<any>(`/api/marketplace/sync/${slug}`, { method: "POST" }),

  // Registry admin
  getRegistries: () => request<any[]>("/api/admin/registries"),
  createRegistry: (data: { name: string; url: string }) => request<any>("/api/admin/registries", { method: "POST", body: JSON.stringify(data) }),
  updateRegistry: (id: string, data: { enabled: boolean }) => request<any>(`/api/admin/registries/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteRegistry: (id: string) => request<any>(`/api/admin/registries/${id}`, { method: "DELETE" }),

  // Registry config
  getRegistryConfig: () => request<any>("/api/admin/config/registry"),
  setRegistryConfig: (data: { enabled: string }) => request<any>("/api/admin/config/registry", { method: "PUT", body: JSON.stringify(data) }),

  // Registration config
  getRegistrationConfig: () => request<{ enabled: string }>("/api/admin/config/registration"),
  setRegistrationConfig: (data: { enabled: string }) => request<any>("/api/admin/config/registration", { method: "PUT", body: JSON.stringify(data) }),

  // Admin: Dashboard
  adminStats: () => request<any>("/api/admin/stats"),

  // Admin: Users
  listUsers: () => request<any[]>("/api/admin/users"),
  createUser: (data: { username: string; password: string; role?: string }) =>
    request("/api/admin/users", { method: "POST", body: JSON.stringify(data) }),
  updateUserRole: (id: string, role: string) =>
    request(`/api/admin/users/${id}/role`, { method: "PUT", body: JSON.stringify({ role }) }),
  updateUserStatus: (id: string, status: string) =>
    request(`/api/admin/users/${id}/status`, { method: "PUT", body: JSON.stringify({ status }) }),
  resetUserPassword: (id: string) =>
    request<{ password: string }>(`/api/admin/users/${id}/password`, { method: "PUT" }),
  deleteUser: (id: string) => request(`/api/admin/users/${id}`, { method: "DELETE" }),
};
