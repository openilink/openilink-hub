import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export function useMarketplaceApps() {
  return useQuery({
    queryKey: queryKeys.marketplace.apps(),
    queryFn: () => api.getMarketplaceApps(),
    staleTime: 60_000,
  });
}

export function useBuiltinApps() {
  return useQuery({
    queryKey: queryKeys.marketplace.builtin(),
    queryFn: () => api.getBuiltinApps(),
    staleTime: 60_000,
  });
}

export function useSyncMarketplaceApp() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (slug: string) => api.syncMarketplaceApp(slug),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.marketplace.apps() });
      qc.invalidateQueries({ queryKey: ["apps"] });
    },
  });
}
