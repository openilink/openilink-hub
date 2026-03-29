import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

// ── Queries ──────────────────────────────────────────────

export function useAdminStats() {
  return useQuery({
    queryKey: queryKeys.admin.stats(),
    queryFn: () => api.adminStats(),
    staleTime: 30_000,
  });
}

export function useAdminUsers() {
  return useQuery({
    queryKey: queryKeys.admin.users(),
    queryFn: async () => (await api.listUsers()) || [],
    staleTime: 15_000,
  });
}

export function useAdminApps() {
  return useQuery({
    queryKey: queryKeys.admin.apps(),
    queryFn: async () => (await api.adminListApps()) || [],
    staleTime: 15_000,
  });
}

export function useAIConfig() {
  return useQuery({
    queryKey: queryKeys.admin.aiConfig(),
    queryFn: () => api.getAIConfig(),
    staleTime: 60_000,
  });
}

export function useOAuthConfig() {
  return useQuery({
    queryKey: queryKeys.admin.oauthConfig(),
    queryFn: () => api.getOAuthConfig(),
    staleTime: 60_000,
  });
}

export function useOIDCConfig() {
  return useQuery({
    queryKey: queryKeys.admin.oidcConfig(),
    queryFn: async () => (await api.getOIDCConfig()) || [],
    staleTime: 60_000,
  });
}

export function useRegistries() {
  return useQuery({
    queryKey: queryKeys.admin.registries(),
    queryFn: async () => (await api.getRegistries()) || [],
    staleTime: 60_000,
  });
}

export function useRegistryConfig() {
  return useQuery({
    queryKey: queryKeys.admin.registryConfig(),
    queryFn: () => api.getRegistryConfig(),
    staleTime: 60_000,
  });
}

export function useRegistrationConfig() {
  return useQuery({
    queryKey: queryKeys.admin.registrationConfig(),
    queryFn: () => api.getRegistrationConfig(),
    staleTime: 60_000,
  });
}

export function useAppReviewHistory(appId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.apps.reviews(appId ?? ""),
    queryFn: () => api.listAppReviews(appId!),
    enabled: !!appId,
  });
}

// ── Mutations ────────────────────────────────────────────

export function useUpdateUserStatus() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      api.updateUserStatus(id, status),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.users() }),
  });
}

export function useDeleteUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteUser(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.users() }),
  });
}

export function useSetAppListing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, listing }: { id: string; listing: string }) =>
      api.setAppListing(id, listing),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.apps() }),
  });
}

export function useReviewListing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ appId, approve, reason }: { appId: string; approve: boolean; reason?: string }) =>
      api.reviewListing(appId, approve, reason),
    onSuccess: (_data, { appId }) => {
      qc.invalidateQueries({ queryKey: queryKeys.admin.apps() });
      qc.invalidateQueries({ queryKey: queryKeys.apps.reviews(appId) });
    },
  });
}

export function useDeleteAdminApp() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteApp(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.apps() }),
  });
}

export function useSaveAIConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: any) => api.setAIConfig(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.aiConfig() }),
  });
}

export function useSetRegistrationConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { enabled: string }) => api.setRegistrationConfig(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.registrationConfig() }),
  });
}

export function useSetRegistryConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { enabled: string }) => api.setRegistryConfig(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.registryConfig() }),
  });
}

export function useCreateRegistry() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { name: string; url: string }) => api.createRegistry(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.registries() }),
  });
}

export function useUpdateRegistry() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: { enabled: boolean } }) =>
      api.updateRegistry(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.registries() }),
  });
}

export function useDeleteRegistry() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteRegistry(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.registries() }),
  });
}

export function useSetOIDCConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ slug, data }: { slug: string; data: { display_name: string; issuer_url: string; client_id: string; client_secret: string; scopes?: string } }) =>
      api.setOIDCConfig(slug, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.oidcConfig() }),
  });
}

export function useDeleteOIDCConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (slug: string) => api.deleteOIDCConfig(slug),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.oidcConfig() }),
  });
}
