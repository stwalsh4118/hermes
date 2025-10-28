"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useMedia } from "@/hooks/use-media";
import { Media } from "@/lib/types/api";
import { Search, Plus, Check } from "lucide-react";
import { LoadingSpinner } from "@/components/common/loading-spinner";

interface MediaBrowserProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSelectMedia: (media: Media) => void;
  playlistMediaIds: string[];
}

export function MediaBrowser({
  open,
  onOpenChange,
  onSelectMedia,
  playlistMediaIds,
}: MediaBrowserProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const { data: mediaResponse, isLoading, isError } = useMedia({
    show: searchQuery || undefined,
    limit: 100,
  });

  const handleSelect = (media: Media) => {
    onSelectMedia(media);
  };

  const isInPlaylist = (mediaId: string) => playlistMediaIds.includes(mediaId);

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
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[80vh] border-4 border-primary shadow-[8px_8px_0_rgba(0,0,0,0.6)]">
        <DialogHeader>
          <DialogTitle className="vcr-text uppercase tracking-wider text-2xl">
            Add Media to Playlist
          </DialogTitle>
        </DialogHeader>

        {/* Search Bar */}
        <div className="relative">
          <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search by show name..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10 border-2 border-primary"
          />
        </div>

        {/* Media List */}
        <ScrollArea className="h-[400px] pr-4">
          {isLoading && (
            <div className="flex items-center justify-center py-12">
              <LoadingSpinner />
            </div>
          )}

          {isError && (
            <div className="text-center py-12 text-destructive vcr-text">
              Failed to load media library
            </div>
          )}

          {!isLoading && !isError && mediaResponse && (
            <div className="space-y-2">
              {mediaResponse.items.length === 0 ? (
                <div className="text-center py-12 text-muted-foreground vcr-text">
                  No media found. Scan your library first.
                </div>
              ) : (
                mediaResponse.items.map((media) => {
                  const inPlaylist = isInPlaylist(media.id);
                  return (
                    <div
                      key={media.id}
                      className={`
                        flex items-center gap-4 p-4 rounded-lg border-2 transition-colors
                        ${
                          inPlaylist
                            ? "bg-muted/50 border-primary/30 opacity-60"
                            : "bg-card border-primary/30 hover:bg-muted/30 cursor-pointer"
                        }
                      `}
                      onClick={() => !inPlaylist && handleSelect(media)}
                    >
                      {/* Thumbnail Placeholder */}
                      <div className="w-20 h-14 flex-shrink-0 rounded bg-muted border-2 border-primary/20 flex items-center justify-center">
                        <svg
                          className="w-6 h-6 text-muted-foreground/40"
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
                          {media.title}
                        </h3>
                        <div className="flex items-center gap-2 text-xs text-muted-foreground mt-1">
                          {media.show_name && (
                            <span className="truncate">{media.show_name}</span>
                          )}
                          {media.season && media.episode && (
                            <span>
                              S{media.season.toString().padStart(2, "0")}E
                              {media.episode.toString().padStart(2, "0")}
                            </span>
                          )}
                          <span>•</span>
                          <span>{formatDuration(media.duration)}</span>
                          {media.resolution && (
                            <>
                              <span>•</span>
                              <span>{media.resolution}</span>
                            </>
                          )}
                        </div>
                      </div>

                      {/* Action Button */}
                      {inPlaylist ? (
                        <div className="flex items-center gap-2 text-primary vcr-text font-bold">
                          <Check className="w-4 h-4" />
                          <span className="text-sm">IN PLAYLIST</span>
                        </div>
                      ) : (
                        <Button
                          size="sm"
                          className="retro-button bg-primary text-primary-foreground hover:bg-primary/80 shadow-[4px_4px_0_rgba(0,0,0,0.2)] hover:shadow-[2px_2px_0_rgba(0,0,0,0.2)]"
                          onClick={(e) => {
                            e.stopPropagation();
                            handleSelect(media);
                          }}
                        >
                          <Plus className="w-4 h-4 mr-1" />
                          ADD
                        </Button>
                      )}
                    </div>
                  );
                })
              )}
            </div>
          )}
        </ScrollArea>

        {/* Footer */}
        <div className="flex justify-between items-center pt-4 border-t-2 border-primary/20">
          <div className="text-sm text-muted-foreground vcr-text">
            {mediaResponse && `${mediaResponse.total} total items`}
          </div>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            className="retro-button border-2 border-primary shadow-[4px_4px_0_rgba(0,0,0,0.2)] hover:shadow-[2px_2px_0_rgba(0,0,0,0.2)]"
          >
            Close
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

