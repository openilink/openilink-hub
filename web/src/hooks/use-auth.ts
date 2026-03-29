import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useAuthStore } from "@/stores/auth-store";

export function useUser() {
  const setUser = useAuthStore((s) => s.setUser);
  return useQuery({
    queryKey: queryKeys.user(),
    queryFn: async () => {
      const user = await api.me();
      setUser(user);
      return user;
    },
    staleTime: 5 * 60_000,
    retry: false,
  });
}

export function useInfo() {
  return useQuery({
    queryKey: queryKeys.info(),
    queryFn: () => api.info(),
    staleTime: 10 * 60_000,
  });
}
