/**
 * @deprecated This component has been replaced by MediaTree in task 9-2B.
 * Use MediaTree for playlist management instead.
 * This file is kept for reference only and will be removed in a future cleanup.
 * 
 * Migration: Replace <PlaylistEditor> with <MediaTree enableReordering={true} showFilterToggle={true} />
 */

"use client";

import { useState, useCallback } from "react";
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from "@dnd-kit/core";
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { PlaylistItem, Media, AddToPlaylistRequest } from "@/lib/types/api";
import { GripVertical, X } from "lucide-react";
import { MediaBrowser } from "./media-browser";
import { useBulkAddToPlaylist } from "@/hooks/use-playlist";

interface PlaylistEditorProps {
  items: PlaylistItem[];
  channelId: string;
  onReorder: (items: PlaylistItem[]) => void;
  onAdd: (media: Media, position: number) => void;
  onRemove: (itemId: string) => void;
}

interface SortableItemProps {
  item: PlaylistItem;
  onRemove: (itemId: string) => void;
}

function SortableItem({ item, onRemove }: SortableItemProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: item.id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  const formatDuration = (seconds: number) => {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;
    
    if (hours > 0) {
      return `${hours}:${minutes.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`;
    }
    return `${minutes}:${secs.toString().padStart(2, "0")}`;
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="flex items-center gap-4 p-4 bg-muted/50 border-2 border-primary/30 rounded-lg hover:bg-muted/80 transition-colors"
    >
      {/* Drag Handle */}
      <button
        type="button"
        className="cursor-grab active:cursor-grabbing shrink-0 hover:text-primary transition-colors"
        {...attributes}
        {...listeners}
      >
        <GripVertical className="w-5 h-5" />
      </button>

      {/* Thumbnail Placeholder */}
      <div className="w-24 h-16 shrink-0 rounded-lg overflow-hidden bg-muted border-2 border-primary/20 shadow-[4px_4px_0_rgba(0,0,0,0.2)] flex items-center justify-center">
        <svg
          className="w-8 h-8 text-muted-foreground/20"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"
          />
        </svg>
      </div>

      {/* Media Info */}
      <div className="flex-1 min-w-0">
        <h3 className="font-bold text-sm truncate vcr-text">
          {item.media?.title || "Unknown Title"}
        </h3>
        <div className="flex items-center gap-2 text-xs text-muted-foreground mt-1">
          {item.media?.show_name && (
            <span className="truncate">{item.media.show_name}</span>
          )}
          {item.media?.season && item.media?.episode && (
            <span>
              S{item.media.season.toString().padStart(2, "0")}E
              {item.media.episode.toString().padStart(2, "0")}
            </span>
          )}
          {item.media?.duration && (
            <>
              <span>•</span>
              <span>{formatDuration(item.media.duration)}</span>
            </>
          )}
          {item.media?.resolution && (
            <>
              <span>•</span>
              <span>{item.media.resolution}</span>
            </>
          )}
        </div>
      </div>

      {/* Remove Button */}
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => onRemove(item.id)}
        className="retro-button bg-destructive/20 text-destructive hover:bg-destructive/40 shadow-[4px_4px_0_rgba(0,0,0,0.2)] hover:shadow-[2px_2px_0_rgba(0,0,0,0.2)]"
      >
        <X className="w-4 h-4" />
      </Button>
    </div>
  );
}

export function PlaylistEditor({ items, channelId, onReorder, onAdd, onRemove }: PlaylistEditorProps) {
  const [mediaBrowserOpen, setMediaBrowserOpen] = useState(false);
  const bulkAddMutation = useBulkAddToPlaylist();
  
  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;

    if (over && active.id !== over.id) {
      const oldIndex = items.findIndex((item) => item.id === active.id);
      const newIndex = items.findIndex((item) => item.id === over.id);

      const reorderedItems = arrayMove(items, oldIndex, newIndex);
      onReorder(reorderedItems);
    }
  };

  const handleBulkAdd = useCallback(async (mediaItems: Media[]) => {
    if (mediaItems.length === 0) return;

    // Single source of truth for bulk adds:
    // - Edit mode (channelId exists): Use bulk API mutation for efficiency
    // - Create mode (no channelId): Update local state only via onAdd
    if (channelId) {
      // Edit mode: Use efficient bulk API call
      const bulkItems: AddToPlaylistRequest[] = mediaItems.map((media, idx) => ({
        media_id: media.id,
        position: items.length + idx,
      }));

      try {
        await bulkAddMutation.mutateAsync({ channelId, items: bulkItems });
        // Mutation hook handles cache invalidation, no need to call onAdd
      } catch (error) {
        console.error("Bulk add failed:", error);
      }
    } else {
      // Create mode: Update local state via onAdd callbacks
      mediaItems.forEach((media, idx) => {
        onAdd(media, items.length + idx);
      });
    }
  }, [channelId, items.length, bulkAddMutation, onAdd]);

  const totalDuration = items.reduce((sum, item) => {
    return sum + (item.media?.duration || 0);
  }, 0);

  const formatTotalDuration = (seconds: number) => {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return `${hours}h ${minutes}m`;
  };

  const playlistMediaIds = items.map((item) => item.media_id);

  return (
    <>
      <Card className="border-4 border-primary bg-card shadow-[8px_8px_0_rgba(0,0,0,0.2)]">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="vcr-text uppercase tracking-wider">
              Playlist ({items.length} items)
            </CardTitle>
            <Button
              type="button"
              onClick={() => setMediaBrowserOpen(true)}
              className="retro-button bg-accent text-accent-foreground hover:bg-accent/80 shadow-[6px_6px_0_rgba(0,0,0,0.2)] hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)]"
            >
              + Add Media
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {items.length === 0 ? (
            <div className="text-center py-12">
              <p className="text-muted-foreground vcr-text text-lg mb-4">
                No items in playlist
              </p>
              <Button
                type="button"
                onClick={() => setMediaBrowserOpen(true)}
                className="retro-button bg-primary text-primary-foreground hover:bg-primary/80 shadow-[6px_6px_0_rgba(0,0,0,0.2)] hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)]"
              >
                Add Media from Library
              </Button>
            </div>
          ) : (
            <>
              <DndContext
                sensors={sensors}
                collisionDetection={closestCenter}
                onDragEnd={handleDragEnd}
              >
                <SortableContext
                  items={items.map((item) => item.id)}
                  strategy={verticalListSortingStrategy}
                >
                  <div className="space-y-3">
                    {items.map((item) => (
                      <SortableItem key={item.id} item={item} onRemove={onRemove} />
                    ))}
                  </div>
                </SortableContext>
              </DndContext>

              {/* Total Duration */}
              <div className="mt-6 pt-6 border-t-2 border-primary/20">
                <div className="flex justify-between items-center vcr-text">
                  <span className="text-muted-foreground">Total Duration:</span>
                  <span className="text-lg font-bold text-primary">
                    {formatTotalDuration(totalDuration)}
                  </span>
                </div>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      <MediaBrowser
        open={mediaBrowserOpen}
        onOpenChange={setMediaBrowserOpen}
        onBulkAdd={handleBulkAdd}
        playlistMediaIds={playlistMediaIds}
        isSubmitting={bulkAddMutation.isPending}
      />
    </>
  );
}

