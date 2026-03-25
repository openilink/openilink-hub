async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    credentials: "same-origin",
    headers: { "Content-Type": "application/json", ...options?.headers },
    ...options,
  });
  if (res.status === 401) {
    const path = window.location.pathname;
    const isPublic = path === "/" || path.startsWith("/webhook-plugins");
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
  info: () => request<{ ai: boolean }>("/api/info"),

  // My plugins
  myPlugins: () => request<any[]>("/api/me/plugins"),

  // Passkeys
  listPasskeys: () => request<any[]>("/api/me/passkeys"),
  passkeyBindBegin: () => request<any>("/api/me/passkeys/register/begin", { method: "POST" }),
  passkeyBindFinishRaw: (body: string) =>
    fetch("/api/me/passkeys/register/finish", {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body,
    }).then(async (r) => {
      if (!r.ok) throw new Error((await r.json()).error);
    }),
  deletePasskey: (id: string) => request(`/api/me/passkeys/${id}`, { method: "DELETE" }),

  // Profile
  updateProfile: (data: { display_name?: string; email?: string }) =>
    request("/api/me/profile", { method: "PUT", body: JSON.stringify(data) }),
  changePassword: (data: { old_password: string; new_password: string }) =>
    request("/api/me/password", { method: "PUT", body: JSON.stringify(data) }),

  // Bots
  listBots: () => request<any[]>("/api/bots"),
  bindStart: () =>
    request<{ session_id: string; qr_url: string }>("/api/bots/bind/start", { method: "POST" }),
  reconnectBot: (id: string) => request(`/api/bots/${id}/reconnect`, { method: "POST" }),
  deleteBot: (id: string) => request(`/api/bots/${id}`, { method: "DELETE" }),
  listBotApps: (botId: string) => request<any[]>(`/api/bots/${botId}/apps`),
  listTraces: (botId: string, limit = 50) => request<any[]>(`/api/bots/${botId}/traces?limit=${limit}`),
  getTrace: (botId: string, traceId: string) => request<any[]>(`/api/bots/${botId}/traces/${traceId}`),
  updateBot: (id: string, data: { name?: string; reminder_hours?: number }) =>
    request(`/api/bots/${id}`, { method: "PUT", body: JSON.stringify(data) }),
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

  // Admin: AI config
  getAIConfig: () => request<any>("/api/admin/config/ai"),
  setAIConfig: (data: {
    base_url?: string;
    api_key?: string;
    model?: string;
    system_prompt?: string;
    max_history?: string;
  }) => request("/api/admin/config/ai", { method: "PUT", body: JSON.stringify(data) }),
  deleteAIConfig: () => request("/api/admin/config/ai", { method: "DELETE" }),

  // Plugins
  listPlugins: (status?: string) =>
    request<any[]>(`/api/webhook-plugins${status ? `?status=${status}` : ""}`),
  getPlugin: (id: string) => request<any>(`/api/webhook-plugins/${id}`),
  submitPlugin: (data: { github_url?: string; script?: string }) =>
    request<any>("/api/webhook-plugins/submit", { method: "POST", body: JSON.stringify(data) }),
  installPlugin: (id: string) =>
    request<any>(`/api/webhook-plugins/${id}/install`, { method: "POST" }),
  pluginVersions: (id: string) => request<any[]>(`/api/webhook-plugins/${id}/versions`),
  cancelVersion: (pluginId: string, versionId: string) =>
    request(`/api/webhook-plugins/${pluginId}/versions/${versionId}/cancel`, { method: "POST" }),
  debugRequest: (data: { script: string; webhook_url?: string; mock_message?: any }) =>
    request<any>("/api/webhook-plugins/debug/request", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  debugResponse: (data: { script: string; mock_message?: any; response: any }) =>
    request<any>("/api/webhook-plugins/debug/response", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  installPluginToChannel: (id: string, botId: string, channelId: string) =>
    request<any>(`/api/webhook-plugins/${id}/install-to-channel`, {
      method: "POST",
      body: JSON.stringify({ bot_id: botId, channel_id: channelId }),
    }),
  reviewPlugin: (id: string, status: string, reason?: string) =>
    request(`/api/admin/webhook-plugins/${id}/review`, {
      method: "PUT",
      body: JSON.stringify({ status, reason: reason || "" }),
    }),
  deletePlugin: (id: string) => request(`/api/admin/webhook-plugins/${id}`, { method: "DELETE" }),

  // Apps
  createApp: (data: any) =>
    request<any>("/api/apps", { method: "POST", body: JSON.stringify(data) }),
  listApps: (opts?: { listed?: boolean }) =>
    request<any[]>(`/api/apps${opts?.listed ? "?listed=true" : ""}`),
  getApp: (id: string) => request<any>(`/api/apps/${id}`),
  updateApp: (id: string, data: any) =>
    request<any>(`/api/apps/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  verifyAppUrl: (appId: string) =>
    request<any>(`/api/apps/${appId}/verify-url`, { method: "POST" }),
  deleteApp: (id: string) => request(`/api/apps/${id}`, { method: "DELETE" }),

  // Admin: Apps
  adminListApps: () => request<any[]>("/api/admin/apps"),
  setAppListed: (id: string, listed: boolean) =>
    request(`/api/admin/apps/${id}/listed`, { method: "PUT", body: JSON.stringify({ listed }) }),

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

  // Webhook logs
  webhookLogs: (botId: string, channelId?: string, limit = 50) =>
    request<any[]>(
      `/api/bots/${botId}/webhook-logs?limit=${limit}${channelId ? "&channel_id=" + channelId : ""}`,
    ),

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
