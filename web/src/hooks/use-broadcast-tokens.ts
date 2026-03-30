import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export function useBroadcastTokens() {
  return useQuery({
    queryKey: queryKeys.broadcastTokens(),
    queryFn: async () => (await api.listBroadcastTokens()) || [],
    staleTime: 15_000,
  });
}

export function useCreateBroadcastToken() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { name: string; bot_ids: string[] }) => api.createBroadcastToken(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.broadcastTokens() }),
  });
}

export function useUpdateBroadcastToken() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: { name: string; bot_ids: string[] } }) =>
      api.updateBroadcastToken(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.broadcastTokens() }),
  });
}

export function useDeleteBroadcastToken() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteBroadcastToken(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.broadcastTokens() }),
  });
}

export function useRegenerateBroadcastToken() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.regenerateBroadcastToken(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.broadcastTokens() }),
  });
}
