export const queryKeys = {
  user: () => ["user"] as const,
  info: () => ["info"] as const,

  bots: {
    all: () => ["bots"] as const,
    detail: (id: string) => ["bots", id] as const,
    apps: (id: string) => ["bots", id, "apps"] as const,
    traces: (id: string, limit?: number) => ["bots", id, "traces", { limit }] as const,
    trace: (botId: string, traceId: string) => ["bots", botId, "traces", traceId] as const,
    channels: (id: string) => ["bots", id, "channels"] as const,
    contacts: (id: string) => ["bots", id, "contacts"] as const,
    messages: (id: string, limit = 30, cursor?: string) => ["bots", id, "messages", { limit, cursor }] as const,
    stats: () => ["bots", "stats"] as const,
    webhookLogs: (botId: string, channelId?: string, limit = 50) => ["bots", botId, "webhook-logs", { channelId, limit }] as const,
  },

  apps: {
    all: (opts?: { listing?: string }) => ["apps", opts] as const,
    detail: (id: string) => ["apps", id] as const,
    installations: (appId: string) => ["apps", appId, "installations"] as const,
    installation: (appId: string, iid: string) => ["apps", appId, "installations", iid] as const,
    reviews: (appId: string) => ["apps", appId, "reviews"] as const,
    eventLogs: (appId: string, iid: string, limit = 50) => ["apps", appId, "installations", iid, "event-logs", { limit }] as const,
    apiLogs: (appId: string, iid: string, limit = 50) => ["apps", appId, "installations", iid, "api-logs", { limit }] as const,
  },

  marketplace: {
    apps: () => ["marketplace"] as const,
    builtin: () => ["marketplace", "builtin"] as const,
  },

  admin: {
    stats: () => ["admin", "stats"] as const,
    users: () => ["admin", "users"] as const,
    apps: () => ["admin", "apps"] as const,
    aiConfig: () => ["admin", "ai-config"] as const,
    oauthConfig: () => ["admin", "oauth-config"] as const,
    oidcConfig: () => ["admin", "oidc-config"] as const,
    registryConfig: () => ["admin", "registry-config"] as const,
    registries: () => ["admin", "registries"] as const,
    registrationConfig: () => ["admin", "registration-config"] as const,
  },

  config: {
    availableModels: () => ["config", "available-models"] as const,
  },

  passkeys: () => ["passkeys"] as const,
  oauthAccounts: () => ["oauth-accounts"] as const,
  oauthProviders: () => ["oauth-providers"] as const,
} as const;
