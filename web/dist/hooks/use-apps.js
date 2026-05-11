import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
/** Invalidate all app-related queries after install/uninstall/create/delete/delist operations. */
export function invalidateAllAppQueries(qc, botId) {
  qc.invalidateQueries({ queryKey: ["apps"] });
  qc.invalidateQueries({ queryKey: queryKeys.marketplace.apps() });
  qc.invalidateQueries({ queryKey: queryKeys.marketplace.builtin() });
  qc.invalidateQueries({ queryKey: queryKeys.apps.all({ listing: "listed" }) });
  qc.invalidateQueries({ queryKey: queryKeys.bots.all() });
  if (botId) {
    qc.invalidateQueries({ queryKey: queryKeys.bots.apps(botId) });
  }
}
export function useApps(opts) {
  return useQuery({
    queryKey: queryKeys.apps.all(opts),
    queryFn: () => api.listApps(opts),
    staleTime: 60000,
  });
}
export function useApp(id) {
  return useQuery({
    queryKey: queryKeys.apps.detail(id),
    queryFn: () => api.getApp(id),
    enabled: !!id,
  });
}
export function useAppInstallations(appId) {
  return useQuery({
    queryKey: queryKeys.apps.installations(appId),
    queryFn: () => api.listInstallations(appId),
    enabled: !!appId,
  });
}
export function useAppInstallation(appId, iid) {
  return useQuery({
    queryKey: queryKeys.apps.installation(appId, iid),
    queryFn: () => api.getInstallation(appId, iid),
    enabled: !!appId && !!iid,
  });
}
export function useAppReviews(appId) {
  return useQuery({
    queryKey: queryKeys.apps.reviews(appId),
    queryFn: () => api.listAppReviews(appId),
    enabled: !!appId,
  });
}
export function useCreateApp() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data) => api.createApp(data),
    onSuccess: () => invalidateAllAppQueries(qc),
  });
}
export function useDeleteApp() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id) => api.deleteApp(id),
    onSuccess: () => invalidateAllAppQueries(qc),
  });
}
export function useInstallApp() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ appId, data }) => api.installApp(appId, data),
    onSuccess: (_data, { appId, data }) => {
      qc.invalidateQueries({ queryKey: queryKeys.apps.installations(appId) });
      invalidateAllAppQueries(qc, data?.bot_id);
    },
  });
}
export function useUninstallApp() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ appId, instId }) => api.deleteInstallation(appId, instId),
    onSuccess: (_data, { appId }) => {
      qc.invalidateQueries({ queryKey: queryKeys.apps.installations(appId) });
      invalidateAllAppQueries(qc);
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
    staleTime: 5 * 60000,
  });
}
