import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export function useCronJobs(botId: string) {
  return useQuery({
    queryKey: queryKeys.bots.cronJobs(botId),
    queryFn: () => api.listCronJobs(botId),
    enabled: !!botId,
  });
}

export function useCreateCronJob(botId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { name: string; cron_expr: string; message: string; recipient?: string }) =>
      api.createCronJob(botId, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.bots.cronJobs(botId) }),
  });
}

export function useUpdateCronJob(botId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      jobId,
      data,
    }: {
      jobId: string;
      data: Partial<{
        name: string;
        cron_expr: string;
        message: string;
        recipient: string;
        enabled: boolean;
      }>;
    }) => api.updateCronJob(botId, jobId, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.bots.cronJobs(botId) }),
  });
}

export function useDeleteCronJob(botId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (jobId: string) => api.deleteCronJob(botId, jobId),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.bots.cronJobs(botId) }),
  });
}
