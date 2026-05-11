export function botDisplayName(bot) {
  return bot.display_name || bot.name;
}
async function request(url, options) {
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
  let data;
  try {
    data = await res.json();
  } catch {
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    throw new Error("invalid response");
  }
  if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
  return data;
}
export const api = {
  // Auth
  register: (username, password) =>
    request("/api/auth/register", { method: "POST", body: JSON.stringify({ username, password }) }),
  login: (username, password) =>
    request("/api/auth/login", { method: "POST", body: JSON.stringify({ username, password }) }),
  logout: () => request("/api/auth/logout", { method: "POST" }),
  oauthProviders: () =>
    request("/api/auth/oauth/providers").then((data) => ({
      providers: (data.providers || []).map((p) =>
        typeof p === "string" ? { name: p, display_name: p, type: "oauth" } : p,
      ),
    })),
  me: () => request("/api/me"),
  info: () => request("/api/info"),
  // Passkeys
  listPasskeys: () => request("/api/me/passkeys"),
  passkeyBindBegin: () => request("/api/me/passkeys/register/begin", { method: "POST" }),
  passkeyBindFinishRaw: (body, name) =>
    fetch(`/api/me/passkeys/register/finish${name ? `?name=${encodeURIComponent(name)}` : ""}`, {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body,
    }).then(async (r) => {
      if (!r.ok) throw new Error((await r.json()).error);
    }),
  deletePasskey: (id) => request(`/api/me/passkeys/${id}`, { method: "DELETE" }),
  renamePasskey: (id, name) =>
    request(`/api/me/passkeys/${id}`, { method: "PATCH", body: JSON.stringify({ name }) }),
  // Profile
  updateProfile: (data) =>
    request("/api/me/profile", { method: "PUT", body: JSON.stringify(data) }),
  updateUsername: (username) =>
    request("/api/me/username", { method: "PUT", body: JSON.stringify({ username }) }),
  changePassword: (data) =>
    request("/api/me/password", { method: "PUT", body: JSON.stringify(data) }),
  // Bots
  listBots: () => request("/api/bots"),
  bindStart: () => request("/api/bots/bind/start", { method: "POST" }),
  reconnectBot: (id) => request(`/api/bots/${id}/reconnect`, { method: "POST" }),
  deleteBot: (id) => request(`/api/bots/${id}`, { method: "DELETE" }),
  listBotApps: (botId) => request(`/api/bots/${botId}/apps`),
  listTraces: (botId, limit = 50) => request(`/api/bots/${botId}/traces?limit=${limit}`),
  getTrace: (botId, traceId) => request(`/api/bots/${botId}/traces/${traceId}`),
  updateBot: (id, data) =>
    request(`/api/bots/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  setBotAI: (botId, enabled) =>
    request(`/api/bots/${botId}/ai`, {
      method: "PUT",
      body: JSON.stringify({ enabled }),
    }),
  setBotAIModel: (botId, model) =>
    request(`/api/bots/${botId}/ai_model`, {
      method: "PUT",
      body: JSON.stringify({ model }),
    }),
  botContacts: (id) => request(`/api/bots/${id}/contacts`),
  // Channels (under bots)
  listChannels: (botId) => request(`/api/bots/${botId}/channels`),
  createChannel: (botId, name, handle) =>
    request(`/api/bots/${botId}/channels`, {
      method: "POST",
      body: JSON.stringify({ name, handle: handle || "" }),
    }),
  updateChannel: (botId, id, data) =>
    request(`/api/bots/${botId}/channels/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteChannel: (botId, id) => request(`/api/bots/${botId}/channels/${id}`, { method: "DELETE" }),
  rotateKey: (botId, id) =>
    request(`/api/bots/${botId}/channels/${id}/rotate_key`, {
      method: "POST",
    }),
  // OAuth accounts
  oauthAccounts: () => request("/api/me/linked-accounts"),
  unlinkOAuth: (provider) => request(`/api/me/linked-accounts/${provider}`, { method: "DELETE" }),
  // Stats
  stats: () => request("/api/bots/stats"),
  // Messages (under bots)
  messages: (botId, limit = 30, cursor) =>
    request(`/api/bots/${botId}/messages?limit=${limit}${cursor ? "&cursor=" + cursor : ""}`),
  sendMessage: (botId, data) =>
    request(`/api/bots/${botId}/send`, { method: "POST", body: JSON.stringify(data) }),
  // Admin: system config
  getOAuthConfig: () => request("/api/admin/config/oauth"),
  setOAuthConfig: (provider, data) =>
    request(`/api/admin/config/oauth/${provider}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteOAuthConfig: (provider) =>
    request(`/api/admin/config/oauth/${provider}`, { method: "DELETE" }),
  // Admin: OIDC config
  getOIDCConfig: () => request("/api/admin/config/oidc"),
  setOIDCConfig: (slug, data) =>
    request(`/api/admin/config/oidc/${slug}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteOIDCConfig: (slug) => request(`/api/admin/config/oidc/${slug}`, { method: "DELETE" }),
  // Public: available models list (all authenticated users)
  getAvailableModels: () => request("/api/config/ai/available_models"),
  // Admin: AI config
  getAIConfig: () => request("/api/admin/config/ai"),
  setAIConfig: (data) =>
    request("/api/admin/config/ai", { method: "PUT", body: JSON.stringify(data) }),
  deleteAIConfig: () => request("/api/admin/config/ai", { method: "DELETE" }),
  // Apps
  importMCP: (data) =>
    request("/api/apps/import-mcp", { method: "POST", body: JSON.stringify(data) }),
  createApp: (data) => request("/api/apps", { method: "POST", body: JSON.stringify(data) }),
  listApps: (opts) => request(`/api/apps${opts?.listing ? `?listing=${opts.listing}` : ""}`),
  getApp: (id) => request(`/api/apps/${id}`),
  updateApp: (id, data) =>
    request(`/api/apps/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  verifyAppUrl: (appId) => request(`/api/apps/${appId}/verify-url`, { method: "POST" }),
  deleteApp: (id) => request(`/api/apps/${id}`, { method: "DELETE" }),
  // Admin: Apps
  adminListApps: () => request("/api/admin/apps"),
  setAppListing: (id, listing) =>
    request(`/api/admin/apps/${id}/listing`, { method: "PUT", body: JSON.stringify({ listing }) }),
  // App Installations
  installApp: (appId, data) =>
    request(`/api/apps/${appId}/install`, { method: "POST", body: JSON.stringify(data) }),
  listInstallations: (appId) => request(`/api/apps/${appId}/installations`),
  getInstallation: (appId, iid) => request(`/api/apps/${appId}/installations/${iid}`),
  updateInstallation: (appId, iid, data) =>
    request(`/api/apps/${appId}/installations/${iid}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),
  deleteInstallation: (appId, iid) =>
    request(`/api/apps/${appId}/installations/${iid}`, { method: "DELETE" }),
  regenerateToken: (appId, iid) =>
    request(`/api/apps/${appId}/installations/${iid}/regenerate-token`, { method: "POST" }),
  listEventLogs: (appId, iid, limit = 50) =>
    request(`/api/apps/${appId}/installations/${iid}/event-logs?limit=${limit}`),
  listApiLogs: (appId, iid, limit = 50) =>
    request(`/api/apps/${appId}/installations/${iid}/api-logs?limit=${limit}`),
  // Listing
  requestListing: (appId) => request(`/api/apps/${appId}/request-listing`, { method: "POST" }),
  reviewListing: (appId, approve, reason) =>
    request(`/api/admin/apps/${appId}/review-listing`, {
      method: "PUT",
      body: JSON.stringify({ approve, reason: reason || "" }),
    }),
  listAppReviews: (appId) => request(`/api/apps/${appId}/reviews`),
  // Webhook logs
  webhookLogs: (botId, channelId, limit = 50) =>
    request(
      `/api/bots/${botId}/webhook-logs?limit=${limit}${channelId ? "&channel_id=" + channelId : ""}`,
    ),
  // Marketplace
  getMarketplaceApps: () => request("/api/marketplace"),
  getBuiltinApps: () => request("/api/marketplace/builtin"),
  syncMarketplaceApp: (slug) => request(`/api/marketplace/sync/${slug}`, { method: "POST" }),
  // Registry admin
  getRegistries: () => request("/api/admin/registries"),
  createRegistry: (data) =>
    request("/api/admin/registries", { method: "POST", body: JSON.stringify(data) }),
  updateRegistry: (id, data) =>
    request(`/api/admin/registries/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteRegistry: (id) => request(`/api/admin/registries/${id}`, { method: "DELETE" }),
  // Registry config
  getRegistryConfig: () => request("/api/admin/config/registry"),
  setRegistryConfig: (data) =>
    request("/api/admin/config/registry", { method: "PUT", body: JSON.stringify(data) }),
  // Registration config
  getRegistrationConfig: () => request("/api/admin/config/registration"),
  setRegistrationConfig: (data) =>
    request("/api/admin/config/registration", { method: "PUT", body: JSON.stringify(data) }),
  // Admin: Dashboard
  adminStats: () => request("/api/admin/stats"),
  // Admin: Users
  listUsers: () => request("/api/admin/users"),
  createUser: (data) => request("/api/admin/users", { method: "POST", body: JSON.stringify(data) }),
  updateUserRole: (id, role) =>
    request(`/api/admin/users/${id}/role`, { method: "PUT", body: JSON.stringify({ role }) }),
  updateUserStatus: (id, status) =>
    request(`/api/admin/users/${id}/status`, { method: "PUT", body: JSON.stringify({ status }) }),
  resetUserPassword: (id) => request(`/api/admin/users/${id}/password`, { method: "PUT" }),
  deleteUser: (id) => request(`/api/admin/users/${id}`, { method: "DELETE" }),
};
