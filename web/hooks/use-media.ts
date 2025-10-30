import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api/client";
import { MediaQueryParams, UpdateMediaRequest, ApiError } from "@/lib/types/api";
import { toast } from "sonner";

// Query key factory for hierarchical cache management
export const mediaKeys = {
  all: ["media"] as const,
  lists: () => [...mediaKeys.all, "list"] as const,
  list: (params?: MediaQueryParams) => [...mediaKeys.lists(), params ?? {}] as const,
  details: () => [...mediaKeys.all, "detail"] as const,
  detail: (id: string) => [...mediaKeys.details(), id] as const,
  scans: () => [...mediaKeys.all, "scan"] as const,
  scan: (scanId: string) => [...mediaKeys.scans(), scanId] as const,
};

// Query hooks

export function useMedia(params?: MediaQueryParams) {
  return useQuery({
    queryKey: mediaKeys.list(params),
    queryFn: () => apiClient.getMedia(params),
  });
}

// Fetch all media (for tree view/library browsing)
export function useMediaAll() {
  return useQuery({
    queryKey: mediaKeys.list({ limit: -1 }),
    queryFn: () => apiClient.getMedia({ limit: -1 }),
  });
}

export function useMediaItem(id: string) {
  return useQuery({
    queryKey: mediaKeys.detail(id),
    queryFn: () => apiClient.getMediaItem(id),
    enabled: !!id,
  });
}

export function useScanStatus(scanId: string) {
  return useQuery({
    queryKey: mediaKeys.scan(scanId),
    queryFn: () => apiClient.getScanStatus(scanId),
    enabled: !!scanId,
    refetchInterval: (query) => {
      const data = query.state.data;
      // Poll every 2 seconds while scan is running
      if (data?.status === "running") {
        return 2000;
      }
      // Stop polling when scan is complete, failed, or cancelled
      return false;
    },
  });
}

// Mutation hooks

export function useUpdateMedia(id: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: UpdateMediaRequest) => apiClient.updateMedia(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: mediaKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: mediaKeys.lists() });
      toast.success("Media updated successfully");
    },
    onError: (error: ApiError) => {
      toast.error(error.message || "Failed to update media");
    },
  });
}

export function useDeleteMedia() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => apiClient.deleteMedia(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: mediaKeys.lists() });
      toast.success("Media deleted successfully");
    },
    onError: (error: ApiError) => {
      toast.error(error.message || "Failed to delete media");
    },
  });
}

export function useScanMedia() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (path: string) => apiClient.scanMedia(path),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: mediaKeys.lists() });
      toast.success("Media scan started");
    },
    onError: (error: ApiError) => {
      toast.error(error.message || "Failed to start media scan");
    },
  });
}

