import { create } from "zustand";
import { api } from "@/lib/api";
import { queryClient } from "@/lib/query-client";
export const useAuthStore = create((set) => ({
  user: null,
  setUser: (user) => set({ user }),
  clearUser: () => set({ user: null }),
  logout: async () => {
    try {
      await api.logout();
    } finally {
      set({ user: null });
      queryClient.clear();
    }
  },
}));
