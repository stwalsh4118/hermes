import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api/client";
import { CreateChannelRequest, UpdateChannelRequest, ApiError } from "@/lib/types/api";
import { toast } from "sonner";

// Query key factory for hierarchical cache management
export const channelKeys = {
  all: ["channels"] as const,
  lists: () => [...channelKeys.all, "list"] as const,
  list: (filters: string) => [...channelKeys.lists(), { filters }] as const,
  details: () => [...channelKeys.all, "detail"] as const,
  detail: (id: string) => [...channelKeys.details(), id] as const,
};

// Query hooks

export function useChannels() {
  return useQuery({
    queryKey: channelKeys.lists(),
    queryFn: () => apiClient.getChannels(),
  });
}

export function useChannel(id: string) {
  return useQuery({
    queryKey: channelKeys.detail(id),
    queryFn: () => apiClient.getChannel(id),
    enabled: !!id,
  });
}

// Mutation hooks

export function useCreateChannel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateChannelRequest) => apiClient.createChannel(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: channelKeys.lists() });
      toast.success("Channel created successfully");
    },
    onError: (error: ApiError) => {
      toast.error(error.message || "Failed to create channel");
    },
  });
}

export function useUpdateChannel(id: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: UpdateChannelRequest) => apiClient.updateChannel(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: channelKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: channelKeys.lists() });
      toast.success("Channel updated successfully");
    },
    onError: (error: ApiError) => {
      toast.error(error.message || "Failed to update channel");
    },
  });
}

export function useDeleteChannel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => apiClient.deleteChannel(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: channelKeys.lists() });
      toast.success("Channel deleted successfully");
    },
    onError: (error: ApiError) => {
      toast.error(error.message || "Failed to delete channel");
    },
  });
}

