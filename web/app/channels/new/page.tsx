"use client";

import { useCallback } from "react";
import { useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { RetroHeaderLayout } from "@/components/layout/retro-header-layout";
import { ChannelForm } from "@/components/channel/channel-form";
import { useCreateChannel, channelKeys } from "@/hooks/use-channels";
import { playlistKeys } from "@/hooks/use-playlist";
import { CreateChannelRequest, PlaylistItem } from "@/lib/types/api";
import { apiClient } from "@/lib/api/client";
import { toast } from "sonner";

export default function NewChannelPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const createChannel = useCreateChannel();

  const handleSubmit = useCallback((data: CreateChannelRequest, playlistItems?: PlaylistItem[]) => {
    createChannel.mutate(data, {
      onSuccess: async (newChannel) => {
        // If there are playlist items, add them to the newly created channel
        if (playlistItems && playlistItems.length > 0) {
          const loadingToastId = toast.loading("Adding playlist items...");
          try {
            // Use bulk add endpoint - single API request for all items
            const result = await apiClient.bulkAddToPlaylist(
              newChannel.id,
              playlistItems.map((item, index) => ({
                media_id: item.media_id,
                position: index,
              }))
            );
            
            // Invalidate queries to refresh data
            queryClient.invalidateQueries({ queryKey: playlistKeys.list(newChannel.id) });
            queryClient.invalidateQueries({ queryKey: channelKeys.detail(newChannel.id) });
            
            // Dismiss the loading toast
            toast.dismiss(loadingToastId);
            
            if (result.failed > 0) {
              toast.warning(`Channel created with ${result.added} of ${result.total} media items (${result.failed} failed)`);
            } else {
              toast.success(`Channel created with ${result.added} media item${result.added > 1 ? 's' : ''}`);
            }
            
            router.push("/channels");
          } catch (error) {
            console.error("Failed to add playlist items:", error);
            toast.dismiss(loadingToastId);
            toast.error("Channel created but failed to add playlist items. You can add them manually from the channel page.");
            // Stay on the page so user can see the error and potentially retry
            // The channel was created successfully, so navigate after a delay
            setTimeout(() => {
              router.push("/channels");
            }, 3000);
          }
        } else {
          // No playlist items, just navigate (channel creation toast already shown)
          router.push("/channels");
        }
      },
    });
  }, [createChannel, queryClient, router]);

  const handleCancel = useCallback(() => {
    router.push("/channels");
  }, [router]);

  return (
    <RetroHeaderLayout>
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-foreground vcr-text">Create New Channel</h1>
        <p className="text-muted-foreground mt-1">Set up a new virtual TV channel</p>
      </div>

      <ChannelForm
        mode="create"
        onSubmit={handleSubmit}
        onCancel={handleCancel}
        isSubmitting={createChannel.isPending}
      />
    </RetroHeaderLayout>
  );
}

