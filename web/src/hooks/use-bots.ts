import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export function useBots() {
  return useQuery({
    queryKey: queryKeys.bots.all(),
    queryFn: async () => (await api.listBots()) || [],
    staleTime: 15_000,
  });
}

export function useBot(id: string) {
  return useQuery({
    queryKey: queryKeys.bots.all(),
    queryFn: async () => (await api.listBots()) || [],
    staleTime: 15_000,
    select: (bots: any[]) => bots.find((b) => b.id === id) ?? null,
  });
}

export function useBotApps(botId: string) {
  return useQuery({
    queryKey: queryKeys.bots.apps(botId),
    queryFn: () => api.listBotApps(botId),
    enabled: !!botId,
  });
}

export function useBotChannels(botId: string) {
  return useQuery({
    queryKey: queryKeys.bots.channels(botId),
    queryFn: () => api.listChannels(botId),
    enabled: !!botId,
  });
}

export function useBotContacts(botId: string) {
  return useQuery({
    queryKey: queryKeys.bots.contacts(botId),
    queryFn: () => api.botContacts(botId),
    enabled: !!botId,
  });
}

export function useBotTraces(botId: string, limit = 50) {
  return useQuery({
    queryKey: queryKeys.bots.traces(botId, limit),
    queryFn: () => api.listTraces(botId, limit),
    enabled: !!botId,
  });
}

export function useBotStats() {
  return useQuery({
    queryKey: queryKeys.bots.stats(),
    queryFn: () => api.stats(),
    staleTime: 30_000,
  });
}

export function useWebhookLogs(botId: string, channelId?: string) {
  return useQuery({
    queryKey: queryKeys.bots.webhookLogs(botId, channelId),
    queryFn: () => api.webhookLogs(botId, channelId),
    enabled: !!botId,
  });
}

export function useDeleteBot() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteBot(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.bots.all() }),
  });
}

export function useUpdateBot() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: any }) => api.updateBot(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.bots.all() }),
  });
}

export function useReconnectBot() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.reconnectBot(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.bots.all() }),
  });
}

export function useSetBotAI() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ botId, enabled }: { botId: string; enabled: boolean }) =>
      api.setBotAI(botId, enabled),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.bots.all() }),
  });
}

export function useSetBotAIModel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ botId, model }: { botId: string; model: string }) =>
      api.setBotAIModel(botId, model),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.bots.all() }),
  });
}
