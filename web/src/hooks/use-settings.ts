import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export function usePasskeys() {
  return useQuery({
    queryKey: queryKeys.passkeys(),
    queryFn: async () => (await api.listPasskeys()) || [],
  });
}

export function useOAuthAccounts() {
  return useQuery({
    queryKey: queryKeys.oauthAccounts(),
    queryFn: async () => (await api.oauthAccounts()) || [],
  });
}

export function useOAuthProviders() {
  return useQuery({
    queryKey: queryKeys.oauthProviders(),
    queryFn: async () => {
      const data = await api.oauthProviders();
      return data.providers as Array<{ name: string; display_name: string; type: string; key?: string }>;
    },
    staleTime: 5 * 60_000,
  });
}

export function useDeletePasskey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deletePasskey(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.passkeys() }),
  });
}

export function useRenamePasskey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) => api.renamePasskey(id, name),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.passkeys() }),
  });
}

export function useUnlinkOAuth() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (provider: string) => api.unlinkOAuth(provider),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.oauthAccounts() });
      qc.invalidateQueries({ queryKey: queryKeys.user() });
    },
  });
}

export function useChangePassword() {
  return useMutation({
    mutationFn: (data: { old_password: string; new_password: string }) =>
      api.changePassword(data),
  });
}

export function useUpdateUsername() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (username: string) => api.updateUsername(username),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.user() }),
  });
}

export function useUpdateProfile() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { display_name?: string; email?: string }) =>
      api.updateProfile(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.user() }),
  });
}
