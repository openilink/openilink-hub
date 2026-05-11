import { create } from "zustand";
import { persist } from "zustand/middleware";
export const useUIStore = create()(
  persist(
    (set) => ({
      sidebarCollapsed: false,
      activeModal: null,
      toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
      openModal: (id) => set({ activeModal: id }),
      closeModal: () => set({ activeModal: null }),
    }),
    {
      name: "ui-store",
      partialize: (s) => ({ sidebarCollapsed: s.sidebarCollapsed }),
    },
  ),
);
