import { QueryClient } from "@tanstack/react-query";
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30000,
      gcTime: 5 * 60000,
      retry: 1,
      refetchOnWindowFocus: true,
    },
  },
});
