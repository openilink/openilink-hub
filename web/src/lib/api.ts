async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    credentials: "same-origin",
    headers: { "Content-Type": "application/json", ...options?.headers },
    ...options,
  });
  if (res.status === 401) {
    window.location.href = "/login";
    throw new Error("unauthorized");
  }
  const data = await res.json();
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
  me: () => request<{ id: string; username: string; display_name: string; role: string }>("/api/auth/me"),

  // Bots
  listBots: () => request<any[]>("/api/bots"),
  bindStart: () => request<{ session_id: string; qr_url: string }>("/api/bots/bind/start", { method: "POST" }),
  reconnectBot: (id: string) => request(`/api/bots/${id}/reconnect`, { method: "POST" }),
  deleteBot: (id: string) => request(`/api/bots/${id}`, { method: "DELETE" }),
  renameBot: (id: string, name: string) =>
    request(`/api/bots/${id}/name`, { method: "PUT", body: JSON.stringify({ name }) }),
  botContacts: (id: string) => request<any[]>(`/api/bots/${id}/contacts`),

  // Channels
  listChannels: () => request<any[]>("/api/channels"),
  createChannel: (bot_id: string, name: string, handle?: string) =>
    request("/api/channels", { method: "POST", body: JSON.stringify({ bot_id, name, handle: handle || "" }) }),
  updateChannel: (id: string, data: any) =>
    request(`/api/channels/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteChannel: (id: string) => request(`/api/channels/${id}`, { method: "DELETE" }),
  rotateKey: (id: string) => request<{ api_key: string }>(`/api/channels/${id}/rotate-key`, { method: "POST" }),

  // OAuth accounts
  oauthAccounts: () => request<any[]>("/api/auth/linked-accounts"),
  unlinkOAuth: (provider: string) =>
    request(`/api/auth/linked-accounts/${provider}`, { method: "DELETE" }),

  // Stats
  stats: () => request<any>("/api/stats"),

  // Messages
  messages: (bot_id: string, limit = 50) => request<any[]>(`/api/messages?bot_id=${bot_id}&limit=${limit}`),

  // Admin: system config
  getOAuthConfig: () => request<Record<string, any>>("/api/admin/config/oauth"),
  setOAuthConfig: (provider: string, data: { client_id: string; client_secret: string }) =>
    request(`/api/admin/config/oauth/${provider}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteOAuthConfig: (provider: string) =>
    request(`/api/admin/config/oauth/${provider}`, { method: "DELETE" }),

  // Admin: AI config
  getAIConfig: () => request<any>("/api/admin/config/ai"),
  setAIConfig: (data: { base_url?: string; api_key?: string; model?: string }) =>
    request("/api/admin/config/ai", { method: "PUT", body: JSON.stringify(data) }),
  deleteAIConfig: () => request("/api/admin/config/ai", { method: "DELETE" }),

  // Users (admin)
  listUsers: () => request<any[]>("/api/users"),
  createUser: (data: any) => request("/api/users", { method: "POST", body: JSON.stringify(data) }),
  deleteUser: (id: string) => request(`/api/users/${id}`, { method: "DELETE" }),
};
