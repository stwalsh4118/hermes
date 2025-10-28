"use client";

import { useState, useCallback } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { useMedia } from "@/hooks/use-media";
import { Media } from "@/lib/types/api";
import { Search, X } from "lucide-react";
import { LoadingSpinner } from "@/components/common/loading-spinner";
import { MediaTree } from "@/components/media/media-tree";

interface MediaBrowserProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onBulkAdd: (media: Media[]) => void;
  playlistMediaIds: string[];
  isSubmitting?: boolean;
}

export function MediaBrowser({
  open,
  onOpenChange,
  onBulkAdd,
  playlistMediaIds,
  isSubmitting = false,
}: MediaBrowserProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedMedia, setSelectedMedia] = useState<Media[]>([]);
  
  const { data: mediaResponse, isLoading, isError } = useMedia({
    limit: 1000,
  });

  const handleSelectionChange = useCallback((media: Media[]) => {
    setSelectedMedia(media);
  }, []);

  const handleBulkAdd = useCallback(() => {
    if (selectedMedia.length === 0) return;
    
    // Pass selected media to parent - parent handles API calls vs local state
    // Single source of truth: parent decides create mode vs edit mode behavior
    onBulkAdd(selectedMedia);
    setSelectedMedia([]);
    setSearchQuery("");
    onOpenChange(false);
  }, [selectedMedia, onBulkAdd, onOpenChange]);

  const handleClearSelection = useCallback(() => {
    setSelectedMedia([]);
  }, []);

  const handleClose = useCallback(() => {
    setSelectedMedia([]);
    setSearchQuery("");
    onOpenChange(false);
  }, [onOpenChange]);

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-4xl max-h-[85vh] border-4 border-primary shadow-[8px_8px_0_rgba(0,0,0,0.6)] flex flex-col">
        <DialogHeader>
          <DialogTitle className="vcr-text uppercase tracking-wider text-2xl">
            Add Media to Playlist
          </DialogTitle>
        </DialogHeader>

        {/* Search Bar */}
        <div className="relative">
          <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search shows, seasons, episodes..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10 border-2 border-primary font-mono"
          />
        </div>

        {/* Selection Info Bar */}
        {selectedMedia.length > 0 && (
          <div className="flex items-center justify-between px-4 py-2 bg-accent/10 border-2 border-accent rounded-lg">
            <span className="font-mono font-bold text-sm text-accent-foreground">
              {selectedMedia.length} {selectedMedia.length === 1 ? "item" : "items"} selected
            </span>
            <Button
              variant="ghost"
              size="sm"
              onClick={handleClearSelection}
              className="h-7 px-2 font-mono font-bold text-xs hover:bg-accent/20"
            >
              <X className="w-3 h-3 mr-1" />
              CLEAR
            </Button>
          </div>
        )}

        {/* Media Tree */}
        <div className="flex-1 min-h-0">
          {isLoading && (
            <div className="flex items-center justify-center h-full">
              <LoadingSpinner />
            </div>
          )}

          {isError && (
            <div className="flex items-center justify-center h-full">
              <div className="text-center py-12 px-6">
                <p className="text-destructive font-bold vcr-text text-lg mb-2">
                  FAILED TO LOAD MEDIA LIBRARY
                </p>
                <p className="text-muted-foreground text-sm font-mono">
                  Please try again or check your connection
                </p>
              </div>
            </div>
          )}

          {!isLoading && !isError && mediaResponse && (
            <>
              {mediaResponse.items.length === 0 ? (
                <div className="flex items-center justify-center h-full">
                  <div className="text-center py-12 px-6">
                    <p className="text-muted-foreground font-mono vcr-text text-lg mb-2">
                      NO MEDIA FOUND
                    </p>
                    <p className="text-muted-foreground text-sm font-mono">
                      Scan your library first to add media files
                    </p>
                  </div>
                </div>
              ) : (
                <MediaTree
                  media={mediaResponse.items}
                  searchQuery={searchQuery}
                  height={400}
                  onSelectionChange={handleSelectionChange}
                  disabledMediaIds={playlistMediaIds}
                />
              )}
            </>
          )}
        </div>

        {/* Footer */}
        <div className="flex justify-between items-center pt-4 border-t-2 border-primary/20">
          <div className="text-sm text-muted-foreground font-mono">
            {mediaResponse && `${mediaResponse.total} total items`}
            {playlistMediaIds.length > 0 && ` • ${playlistMediaIds.length} already in playlist`}
          </div>
          <div className="flex gap-3">
            <Button
              variant="outline"
              onClick={handleClose}
              className="retro-button border-2 border-primary/30 shadow-[4px_4px_0_rgba(0,0,0,0.2)] hover:shadow-[2px_2px_0_rgba(0,0,0,0.2)] font-mono"
            >
              CLOSE
            </Button>
            <Button
              onClick={handleBulkAdd}
              disabled={selectedMedia.length === 0 || isSubmitting}
              className="retro-button bg-primary text-primary-foreground hover:bg-primary/80 shadow-[6px_6px_0_rgba(0,0,0,0.2)] hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)] font-mono disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isSubmitting ? (
                <>
                  <LoadingSpinner size="sm" />
                  <span className="ml-2">ADDING...</span>
                </>
              ) : (
                <>
                  ADD SELECTED ({selectedMedia.length})
                </>
              )}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
