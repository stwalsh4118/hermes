"use client";

import { useEffect, useState, useCallback, memo } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { MediaTree } from "@/components/media";
import { ChannelPreview } from "./channel-preview";
import { Channel, PlaylistItem, Media } from "@/lib/types/api";
import { useMediaAll } from "@/hooks/use-media";
import { useAddToPlaylist, useRemoveFromPlaylist, useReorderPlaylist, useBulkAddToPlaylist } from "@/hooks/use-playlist";
import { toast } from "sonner";

const channelFormSchema = z.object({
  name: z.string().min(1, "Channel name is required").max(100, "Channel name must be 100 characters or less"),
  icon: z.string().refine(
    (val) => val === "" || z.string().url().safeParse(val).success,
    { message: "Must be a valid URL" }
  ),
  start_time: z.string().min(1, "Start time is required"),
  loop: z.boolean(),
});

type ChannelFormData = z.infer<typeof channelFormSchema>;

interface ChannelFormProps {
  mode: "create" | "edit";
  channel?: Channel;
  playlist?: PlaylistItem[];
  onSubmit: (data: ChannelFormData, playlistItems?: PlaylistItem[]) => void;
  onCancel: () => void;
  isSubmitting?: boolean;
}

export const ChannelForm = memo(function ChannelForm({
  mode,
  channel,
  playlist = [],
  onSubmit,
  onCancel,
  isSubmitting = false,
}: ChannelFormProps) {
  const [localPlaylist, setLocalPlaylist] = useState<PlaylistItem[]>(playlist);
  const [isInitialLoad, setIsInitialLoad] = useState(true);

  // Fetch all media for the tree (no pagination limits)
  const { data: mediaResponse, isLoading: isLoadingMedia } = useMediaAll();
  const allMedia = mediaResponse?.items || [];

  const addToPlaylist = useAddToPlaylist();
  const removeFromPlaylist = useRemoveFromPlaylist();
  const reorderPlaylist = useReorderPlaylist();
  const bulkAddToPlaylist = useBulkAddToPlaylist();

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<ChannelFormData>({
    resolver: zodResolver(channelFormSchema),
    defaultValues: {
      name: channel?.name || "",
      icon: channel?.icon || "",
      start_time: channel?.start_time
        ? new Date(channel.start_time).toISOString().slice(0, 16)
        : new Date().toISOString().slice(0, 16),
      loop: channel?.loop ?? true,
    },
  });

  // Sync playlist in edit mode
  useEffect(() => {
    if (mode === "edit") {
      setLocalPlaylist(playlist);
    }
  }, [mode, playlist]);

  const startTime = watch("start_time");

  const handlePlaylistReorder = useCallback((reorderedItems: PlaylistItem[]) => {
    setLocalPlaylist(reorderedItems);

    // If editing, save to backend immediately
    if (mode === "edit" && channel) {
      const items = reorderedItems.map((item, index) => ({
        item_id: item.id,
        position: index,
      }));
      reorderPlaylist.mutate({
        channelId: channel.id,
        data: { items },
      });
    }
  }, [mode, channel, reorderPlaylist]);

  const handleAddToPlaylist = useCallback((media: Media, position: number) => {
    if (mode === "edit" && channel) {
      // Save to backend immediately in edit mode
      addToPlaylist.mutate({
        channelId: channel.id,
        data: {
          media_id: media.id,
          position,
        },
      });
    } else {
      // In create mode, insert at the specified position
      const newItem: PlaylistItem = {
        id: `temp-${crypto.randomUUID()}`,
        channel_id: channel?.id || "",
        media_id: media.id,
        position,
        created_at: new Date().toISOString(),
        media,
      };
      const updatedPlaylist = [...localPlaylist];
      updatedPlaylist.splice(position, 0, newItem);
      // Update positions for all items
      updatedPlaylist.forEach((item, index) => {
        item.position = index;
      });
      setLocalPlaylist(updatedPlaylist);
    }
  }, [mode, channel, addToPlaylist, localPlaylist]);

  const handleRemoveFromPlaylist = useCallback((itemId: string) => {
    if (mode === "edit" && channel) {
      // Save to backend immediately in edit mode
      removeFromPlaylist.mutate({
        channelId: channel.id,
        itemId,
      });
    } else {
      // In create mode, just update local state
      setLocalPlaylist(localPlaylist.filter((item) => item.id !== itemId));
    }
  }, [mode, channel, removeFromPlaylist, localPlaylist]);

  // Handle MediaTree selection changes
  const handleTreeSelectionChange = useCallback((selectedMedia: Media[]) => {
    // Skip the initial load to avoid spurious API calls
    if (isInitialLoad) {
      setIsInitialLoad(false);
      return;
    }
    
    if (mode === "edit" && channel) {
      // In edit mode, calculate what was added/removed and update via API
      const currentMediaIds = new Set(localPlaylist.map(item => item.media_id));
      const selectedMediaIds = new Set(selectedMedia.map(m => m.id));
      
      // Find newly added items
      const addedMedia = selectedMedia.filter(m => !currentMediaIds.has(m.id));
      
      // Find removed items
      const removedItems = localPlaylist.filter(item => !selectedMediaIds.has(item.media_id));
      
      // Bulk add new items
      if (addedMedia.length > 0) {
        const items = addedMedia.map((media, index) => ({
          media_id: media.id,
          position: localPlaylist.length + index,
        }));
        bulkAddToPlaylist.mutate({ channelId: channel.id, items });
      }
      
      // Remove unselected items
      removedItems.forEach(item => {
        removeFromPlaylist.mutate({ channelId: channel.id, itemId: item.id });
      });
    } else {
      // In create mode, update local state
      const newPlaylist: PlaylistItem[] = selectedMedia.map((media, index) => ({
        id: `temp-${media.id}`,
        channel_id: channel?.id || "",
        media_id: media.id,
        position: index,
        created_at: new Date().toISOString(),
        media,
      }));
      setLocalPlaylist(newPlaylist);
    }
  }, [mode, channel, localPlaylist, bulkAddToPlaylist, removeFromPlaylist, isInitialLoad]);

  const handleFormSubmit = (data: ChannelFormData) => {
    // Convert datetime-local to ISO string
    const submitData = {
      name: data.name,
      start_time: new Date(data.start_time).toISOString(),
      loop: data.loop,
      icon: data.icon && data.icon !== "" ? data.icon : undefined,
    };
    // In create mode, pass localPlaylist to parent so it can add items after channel creation
    onSubmit(submitData, mode === "create" ? localPlaylist : undefined);
  };

  const showEmptyPlaylistWarning = localPlaylist.length === 0;

  return (
    <form onSubmit={handleSubmit(handleFormSubmit)} className="space-y-6">
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Channel Settings */}
        <div className="lg:col-span-1">
          <Card className="border-4 border-primary bg-card shadow-[8px_8px_0_rgba(0,0,0,0.6)]">
            <CardHeader>
              <CardTitle className="vcr-text uppercase tracking-wider">
                Channel Settings
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              {/* Channel Name */}
              <div className="space-y-2">
                <Label htmlFor="name" className="vcr-text uppercase text-sm">
                  Channel Name *
                </Label>
                <Input
                  id="name"
                  {...register("name")}
                  className="border-2 border-primary"
                  placeholder="e.g., Saturday Morning Classics"
                />
                {errors.name && (
                  <p className="text-sm text-destructive">{errors.name.message}</p>
                )}
              </div>

              {/* Icon URL */}
              <div className="space-y-2">
                <Label htmlFor="icon" className="vcr-text uppercase text-sm">
                  Icon URL (Optional)
                </Label>
                <Input
                  id="icon"
                  {...register("icon")}
                  className="border-2 border-primary"
                  placeholder="https://example.com/icon.png"
                />
                {errors.icon && (
                  <p className="text-sm text-destructive">{errors.icon.message}</p>
                )}
                <p className="text-xs text-muted-foreground">
                  Provide a URL to an image or emoji
                </p>
              </div>

              {/* Start Time */}
              <div className="space-y-2">
                <Label htmlFor="start_time" className="vcr-text uppercase text-sm">
                  Start Time *
                </Label>
                <Input
                  id="start_time"
                  type="datetime-local"
                  {...register("start_time")}
                  className="border-2 border-primary"
                />
                {errors.start_time && (
                  <p className="text-sm text-destructive">{errors.start_time.message}</p>
                )}
                <p className="text-xs text-muted-foreground">
                  When did/will the channel start broadcasting?
                </p>
              </div>

              {/* Loop Setting */}
              <div className="space-y-4 pt-4 border-t-2 border-primary/20">
                <div className="flex items-center gap-3">
                  <input
                    type="checkbox"
                    id="loop"
                    {...register("loop")}
                    className="w-5 h-5 rounded border-2 border-primary bg-background checked:bg-primary checked:border-primary focus:ring-2 focus:ring-ring focus:ring-offset-2 cursor-pointer"
                  />
                  <div className="space-y-0.5">
                    <Label htmlFor="loop" className="vcr-text uppercase text-sm cursor-pointer">
                      Loop Playlist
                    </Label>
                    <p className="text-xs text-muted-foreground">
                      Restart playlist when it ends
                    </p>
                  </div>
                </div>
              </div>

              {/* Empty Playlist Warning */}
              {showEmptyPlaylistWarning && (
                <div className="pt-4 border-t-2 border-primary/20">
                  <div className="bg-destructive/10 border-2 border-destructive/50 rounded-lg p-3">
                    <p className="text-sm text-destructive vcr-text">
                      ⚠️ Playlist is empty. Add media to make your channel functional.
                    </p>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Channel Preview */}
          <div className="mt-6">
            <ChannelPreview
              playlist={localPlaylist}
              startTime={startTime || new Date().toISOString()}
            />
          </div>
        </div>

        {/* Media Tree for Playlist Management */}
        <div className="lg:col-span-2">
          <Card className="border-4 border-primary bg-card shadow-[8px_8px_0_rgba(0,0,0,0.6)]">
            <CardHeader>
              <CardTitle className="vcr-text uppercase tracking-wider">
                Playlist Content
              </CardTitle>
            </CardHeader>
            <CardContent>
              <MediaTree
                media={allMedia}
                isLoading={isLoadingMedia}
                height={600}
                enableReordering={false}
                showFilterToggle={true}
                onSelectionChange={handleTreeSelectionChange}
                disabledMediaIds={[]}
                initialSelectedMediaIds={localPlaylist.map(item => item.media_id)}
              />
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Form Actions */}
      <div className="flex items-center justify-end gap-4 pt-6 border-t-4 border-primary">
        <Button
          type="button"
          variant="outline"
          onClick={onCancel}
          disabled={isSubmitting}
          className="retro-button bg-muted text-muted-foreground hover:bg-muted/80 shadow-[6px_6px_0_rgba(0,0,0,0.2)] hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)]"
        >
          Cancel
        </Button>
        <Button
          type="submit"
          disabled={isSubmitting}
          className="retro-button bg-primary text-primary-foreground hover:bg-primary/80 shadow-[6px_6px_0_rgba(0,0,0,0.2)] hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)]"
        >
          {isSubmitting ? "Saving..." : mode === "create" ? "Create Channel" : "Save Changes"}
        </Button>
      </div>
    </form>
  );
});

