import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
export function useBots() {
  return useQuery({
    queryKey: queryKeys.bots.all(),
    queryFn: async () => (await api.listBots()) || [],
    staleTime: 15000,
  });
}
export function useBot(id) {
  return useQuery({
    queryKey: queryKeys.bots.all(),
    queryFn: async () => (await api.listBots()) || [],
    staleTime: 15000,
    select: (bots) => bots.find((b) => b.id === id) ?? null,
  });
}
export function useBotApps(botId) {
  return useQuery({
    queryKey: queryKeys.bots.apps(botId),
    queryFn: () => api.listBotApps(botId),
    enabled: !!botId,
  });
}
export function useBotChannels(botId) {
  return useQuery({
    queryKey: queryKeys.bots.channels(botId),
    queryFn: () => api.listChannels(botId),
    enabled: !!botId,
  });
}
export function useBotContacts(botId) {
  return useQuery({
    queryKey: queryKeys.bots.contacts(botId),
    queryFn: () => api.botContacts(botId),
    enabled: !!botId,
  });
}
export function useBotTraces(botId, limit = 50) {
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
    staleTime: 30000,
  });
}
export function useWebhookLogs(botId, channelId) {
  return useQuery({
    queryKey: queryKeys.bots.webhookLogs(botId, channelId),
    queryFn: () => api.webhookLogs(botId, channelId),
    enabled: !!botId,
  });
}
export function useDeleteBot() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id) => api.deleteBot(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.bots.all() }),
  });
}
export function useUpdateBot() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }) => api.updateBot(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.bots.all() }),
  });
}
export function useReconnectBot() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id) => api.reconnectBot(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.bots.all() }),
  });
}
export function useSetBotAI() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ botId, enabled }) => api.setBotAI(botId, enabled),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.bots.all() }),
  });
}
export function useSetBotAIModel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ botId, model }) => api.setBotAIModel(botId, model),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.bots.all() }),
  });
}
