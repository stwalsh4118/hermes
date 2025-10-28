import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api/client";
import { AddToPlaylistRequest, ReorderPlaylistRequest, ApiError } from "@/lib/types/api";
import { toast } from "sonner";
import { channelKeys } from "./use-channels";

// Query key factory for hierarchical cache management
export const playlistKeys = {
  all: ["playlists"] as const,
  lists: () => [...playlistKeys.all, "list"] as const,
  list: (channelId: string) => [...playlistKeys.lists(), channelId] as const,
};

// Query hooks

export function usePlaylist(channelId: string) {
  return useQuery({
    queryKey: playlistKeys.list(channelId),
    queryFn: () => apiClient.getPlaylist(channelId),
    enabled: !!channelId,
  });
}

// Mutation hooks

export function useAddToPlaylist() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ channelId, data }: { channelId: string; data: AddToPlaylistRequest }) =>
      apiClient.addToPlaylist(channelId, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: playlistKeys.list(variables.channelId) });
      queryClient.invalidateQueries({ queryKey: channelKeys.detail(variables.channelId) });
      toast.success("Media added to playlist");
    },
    onError: (error: ApiError) => {
      toast.error(`Failed to add to playlist: ${error.message}`);
    },
  });
}

export function useRemoveFromPlaylist() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ channelId, itemId }: { channelId: string; itemId: string }) =>
      apiClient.removeFromPlaylist(channelId, itemId),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: playlistKeys.list(variables.channelId) });
      queryClient.invalidateQueries({ queryKey: channelKeys.detail(variables.channelId) });
      toast.success("Item removed from playlist");
    },
    onError: (error: ApiError) => {
      toast.error(`Failed to remove from playlist: ${error.message}`);
    },
  });
}

export function useReorderPlaylist() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ channelId, data }: { channelId: string; data: ReorderPlaylistRequest }) =>
      apiClient.reorderPlaylist(channelId, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: playlistKeys.list(variables.channelId) });
      toast.success("Playlist reordered");
    },
    onError: (error: ApiError) => {
      toast.error(`Failed to reorder playlist: ${error.message}`);
    },
  });
}

