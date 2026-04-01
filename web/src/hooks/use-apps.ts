import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export function useApps(opts?: { listing?: string }) {
  return useQuery({
    queryKey: queryKeys.apps.all(opts),
    queryFn: () => api.listApps(opts),
    staleTime: 60_000,
  });
}

export function useApp(id: string) {
  return useQuery({
    queryKey: queryKeys.apps.detail(id),
    queryFn: () => api.getApp(id),
    enabled: !!id,
  });
}

export function useAppInstallations(appId: string) {
  return useQuery({
    queryKey: queryKeys.apps.installations(appId),
    queryFn: () => api.listInstallations(appId),
    enabled: !!appId,
  });
}

export function useAppInstallation(appId: string, iid: string) {
  return useQuery({
    queryKey: queryKeys.apps.installation(appId, iid),
    queryFn: () => api.getInstallation(appId, iid),
    enabled: !!appId && !!iid,
  });
}

export function useAppReviews(appId: string) {
  return useQuery({
    queryKey: queryKeys.apps.reviews(appId),
    queryFn: () => api.listAppReviews(appId),
    enabled: !!appId,
  });
}

export function useCreateApp() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: any) => api.createApp(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["apps"] }),
  });
}

export function useDeleteApp() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteApp(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["apps"] }),
  });
}

export function useInstallApp() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ appId, data }: { appId: string; data: any }) => api.installApp(appId, data),
    onSuccess: (_data, { appId, data }) => {
      qc.invalidateQueries({ queryKey: queryKeys.apps.installations(appId) });
      qc.invalidateQueries({ queryKey: queryKeys.bots.all() });
      if (data?.bot_id) {
        qc.invalidateQueries({ queryKey: queryKeys.bots.apps(data.bot_id) });
      }
      qc.invalidateQueries({ queryKey: queryKeys.marketplace.apps() });
      qc.invalidateQueries({ queryKey: queryKeys.marketplace.builtin() });
      qc.invalidateQueries({ queryKey: queryKeys.apps.all({ listing: "listed" }) });
    },
  });
}

export function useUninstallApp() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ appId, instId }: { appId: string; instId: string }) =>
      api.deleteInstallation(appId, instId),
    onSuccess: (_data, { appId }) => {
      qc.invalidateQueries({ queryKey: queryKeys.apps.installations(appId) });
      qc.invalidateQueries({ queryKey: queryKeys.bots.all() });
      qc.invalidateQueries({ queryKey: queryKeys.marketplace.apps() });
      qc.invalidateQueries({ queryKey: queryKeys.marketplace.builtin() });
      qc.invalidateQueries({ queryKey: queryKeys.apps.all({ listing: "listed" }) });
    },
  });
}

export function useAvailableModels() {
  return useQuery({
    queryKey: queryKeys.config.availableModels(),
    queryFn: async () => {
      const models = await api.getAvailableModels();
      return Array.isArray(models) ? models : [];
    },
    staleTime: 5 * 60_000,
  });
}
