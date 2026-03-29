import { create } from "zustand";
import { api } from "@/lib/api";
import { queryClient } from "@/lib/query-client";

interface User {
  id: string;
  username: string;
  display_name: string;
  role: string;
  email?: string;
  has_password?: boolean;
  has_passkey?: boolean;
  has_oauth?: boolean;
}

interface AuthState {
  user: User | null;
  setUser: (user: User) => void;
  clearUser: () => void;
  logout: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
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
