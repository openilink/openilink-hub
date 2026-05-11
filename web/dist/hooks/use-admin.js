import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { invalidateAllAppQueries } from "@/hooks/use-apps";
// ── Queries ──────────────────────────────────────────────
export function useAdminStats() {
  return useQuery({
    queryKey: queryKeys.admin.stats(),
    queryFn: () => api.adminStats(),
    staleTime: 30000,
  });
}
export function useAdminUsers() {
  return useQuery({
    queryKey: queryKeys.admin.users(),
    queryFn: async () => (await api.listUsers()) || [],
    staleTime: 15000,
  });
}
export function useAdminApps() {
  return useQuery({
    queryKey: queryKeys.admin.apps(),
    queryFn: async () => (await api.adminListApps()) || [],
    staleTime: 15000,
  });
}
export function useAIConfig() {
  return useQuery({
    queryKey: queryKeys.admin.aiConfig(),
    queryFn: () => api.getAIConfig(),
    staleTime: 60000,
  });
}
export function useOAuthConfig() {
  return useQuery({
    queryKey: queryKeys.admin.oauthConfig(),
    queryFn: () => api.getOAuthConfig(),
    staleTime: 60000,
  });
}
export function useOIDCConfig() {
  return useQuery({
    queryKey: queryKeys.admin.oidcConfig(),
    queryFn: async () => (await api.getOIDCConfig()) || [],
    staleTime: 60000,
  });
}
export function useRegistries() {
  return useQuery({
    queryKey: queryKeys.admin.registries(),
    queryFn: async () => (await api.getRegistries()) || [],
    staleTime: 60000,
  });
}
export function useRegistryConfig() {
  return useQuery({
    queryKey: queryKeys.admin.registryConfig(),
    queryFn: () => api.getRegistryConfig(),
    staleTime: 60000,
  });
}
export function useRegistrationConfig() {
  return useQuery({
    queryKey: queryKeys.admin.registrationConfig(),
    queryFn: () => api.getRegistrationConfig(),
    staleTime: 60000,
  });
}
export function useAppReviewHistory(appId) {
  return useQuery({
    queryKey: queryKeys.apps.reviews(appId ?? ""),
    queryFn: () => api.listAppReviews(appId),
    enabled: !!appId,
  });
}
// ── Mutations ────────────────────────────────────────────
export function useUpdateUserStatus() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, status }) => api.updateUserStatus(id, status),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.users() }),
  });
}
export function useDeleteUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id) => api.deleteUser(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.users() }),
  });
}
export function useSetAppListing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, listing }) => api.setAppListing(id, listing),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.admin.apps() });
      invalidateAllAppQueries(qc);
    },
  });
}
export function useReviewListing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ appId, approve, reason }) => api.reviewListing(appId, approve, reason),
    onSuccess: (_data, { appId }) => {
      qc.invalidateQueries({ queryKey: queryKeys.admin.apps() });
      qc.invalidateQueries({ queryKey: queryKeys.apps.reviews(appId) });
      invalidateAllAppQueries(qc);
    },
  });
}
export function useDeleteAdminApp() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id) => api.deleteApp(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.admin.apps() });
      invalidateAllAppQueries(qc);
    },
  });
}
export function useSaveAIConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data) => api.setAIConfig(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.aiConfig() }),
  });
}
export function useSetRegistrationConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data) => api.setRegistrationConfig(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.registrationConfig() }),
  });
}
export function useSetRegistryConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data) => api.setRegistryConfig(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.registryConfig() }),
  });
}
export function useCreateRegistry() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data) => api.createRegistry(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.registries() }),
  });
}
export function useUpdateRegistry() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }) => api.updateRegistry(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.registries() }),
  });
}
export function useDeleteRegistry() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id) => api.deleteRegistry(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.registries() }),
  });
}
export function useSetOIDCConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ slug, data }) => api.setOIDCConfig(slug, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.oidcConfig() }),
  });
}
export function useDeleteOIDCConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (slug) => api.deleteOIDCConfig(slug),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.oidcConfig() }),
  });
}
