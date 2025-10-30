"use client";

import { useState, useMemo } from "react";
import { RetroHeaderLayout } from "@/components/layout/retro-header-layout";
import { useMedia } from "@/hooks/use-media";
import { Media } from "@/lib/types/api";
import { LibraryScanner, MediaTree } from "@/components/media";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { formatDuration } from "@/lib/utils/format";
import { X, Library as LibraryIcon } from "lucide-react";

export default function LibraryPage() {
  // Filter states
  const [searchQuery, setSearchQuery] = useState("");
  const [showFilter, setShowFilter] = useState<string | null>(null);
  const [needsTranscodingFilter, setNeedsTranscodingFilter] = useState(false);

  // Fetch all media (unlimited)
  const { data: mediaResponse, isLoading, isError, refetch } = useMedia({ limit: -1 });

  const mediaItems = mediaResponse?.items || [];

  // Extract unique show names for filter dropdown
  const showNames = useMemo(() => {
    const names = new Set<string>();
    mediaItems.forEach((item) => {
      if (item.show_name) {
        names.add(item.show_name);
      }
    });
    return Array.from(names).sort();
  }, [mediaItems]);

  // Apply filters to media
  const filteredMedia = useMemo(() => {
    return mediaItems.filter((item) => {
      // Search filter - check title, show_name, and file_path
      if (searchQuery) {
        const query = searchQuery.toLowerCase();
        const matchesSearch =
          item.title?.toLowerCase().includes(query) ||
          item.show_name?.toLowerCase().includes(query) ||
          item.file_path?.toLowerCase().includes(query);

        if (!matchesSearch) {
          return false;
        }
      }

      // Show name filter
      if (showFilter && item.show_name !== showFilter) {
        return false;
      }

      // Needs transcoding filter
      if (needsTranscodingFilter) {
        const needsTranscode =
          item.video_codec?.toLowerCase() !== "h264" ||
          item.audio_codec?.toLowerCase() !== "aac";
        if (!needsTranscode) {
          return false;
        }
      }

      return true;
    });
  }, [mediaItems, searchQuery, showFilter, needsTranscodingFilter]);

  // Calculate stats
  const stats = useMemo(() => {
    const uniqueShows = new Set<string>();
    let episodeCount = 0;
    let totalDuration = 0;

    filteredMedia.forEach((item) => {
      if (item.show_name) {
        uniqueShows.add(item.show_name);
      }
      if (item.season != null || item.episode != null) {
        episodeCount++;
      }
      totalDuration += item.duration || 0;
    });

    return {
      totalShows: uniqueShows.size,
      totalEpisodes: episodeCount,
      totalDuration,
      totalItems: filteredMedia.length,
    };
  }, [filteredMedia]);

  // Clear all filters
  const clearFilters = () => {
    setSearchQuery("");
    setShowFilter(null);
    setNeedsTranscodingFilter(false);
  };

  const hasActiveFilters = searchQuery || showFilter || needsTranscodingFilter;

  // TODO: Integration point for MediaDetailModal (Task 9-5)
  // When task 9-5 is complete, add state and handler:
  // const [selectedMedia, setSelectedMedia] = useState<Media | null>(null);
  // Then pass to MediaTree: onEpisodeClick={(media) => setSelectedMedia(media)}
  // And render: {selectedMedia && <MediaDetailModal media={selectedMedia} onClose={...} />}

  return (
    <RetroHeaderLayout>
      {/* Page Title and Actions */}
      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="vcr-text font-mono text-4xl font-bold uppercase tracking-wider">
            Media Library
          </h2>
          <p className="mt-2 text-muted-foreground">
            Browse your video collection
          </p>
        </div>
        <LibraryScanner onScanComplete={() => refetch()} defaultPath="/media" />
      </div>

      {/* Search and Filter Bar */}
      <div className="mb-6 flex flex-col gap-4 lg:flex-row">
        {/* Search Input */}
        <div className="flex-1">
          <Input
            type="text"
            placeholder="Search media..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="h-12 border-4 border-primary/20 font-mono uppercase tracking-wider shadow-[4px_4px_0_rgba(0,0,0,0.2)] focus-visible:border-primary"
          />
        </div>

        {/* Show Filter */}
        <Select value={showFilter || "all"} onValueChange={(value) => setShowFilter(value === "all" ? null : value)}>
          <SelectTrigger className="h-12 w-full border-4 border-primary/20 font-mono uppercase tracking-wider shadow-[4px_4px_0_rgba(0,0,0,0.2)] lg:w-[240px]">
            <SelectValue placeholder="All Shows" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">ALL SHOWS</SelectItem>
            {showNames.map((name) => (
              <SelectItem key={name} value={name}>
                {name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        {/* Needs Transcoding Filter */}
        <div className="flex h-12 items-center gap-2 rounded-lg border-4 border-primary/20 bg-card px-4 shadow-[4px_4px_0_rgba(0,0,0,0.2)]">
          <Checkbox
            id="needs-transcoding"
            checked={needsTranscodingFilter}
            onCheckedChange={(checked) => setNeedsTranscodingFilter(checked === true)}
          />
          <Label
            htmlFor="needs-transcoding"
            className="cursor-pointer font-mono text-sm font-bold uppercase tracking-wider"
          >
            Needs Transcoding
          </Label>
        </div>

        {/* Clear Filters */}
        {hasActiveFilters && (
          <Button
            variant="outline"
            onClick={clearFilters}
            className="h-12 border-4 font-mono font-bold uppercase tracking-wider shadow-[4px_4px_0_rgba(0,0,0,0.2)]"
          >
            <X className="mr-2 h-4 w-4" />
            Clear
          </Button>
        )}
      </div>

      {/* Stats Summary */}
      {!isLoading && !isError && mediaItems.length > 0 && (
        <div className="mb-8 grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-4">
          <div className="rounded-lg border-4 border-primary/20 bg-card p-4 shadow-[8px_8px_0_rgba(0,0,0,0.2)] md:p-6">
            <div className="vcr-text text-xs text-muted-foreground sm:text-sm">
              Total Items
            </div>
            <div className="mt-2 font-mono text-3xl font-bold md:text-4xl">
              {stats.totalItems}
            </div>
          </div>
          <div className="rounded-lg border-4 border-primary/20 bg-card p-4 shadow-[8px_8px_0_rgba(0,0,0,0.2)] md:p-6">
            <div className="vcr-text text-xs text-muted-foreground sm:text-sm">
              Total Shows
            </div>
            <div className="mt-2 font-mono text-3xl font-bold text-accent md:text-4xl">
              {stats.totalShows}
            </div>
          </div>
          <div className="rounded-lg border-4 border-primary/20 bg-card p-4 shadow-[8px_8px_0_rgba(0,0,0,0.2)] md:p-6">
            <div className="vcr-text text-xs text-muted-foreground sm:text-sm">
              Total Episodes
            </div>
            <div className="mt-2 font-mono text-3xl font-bold text-secondary md:text-4xl">
              {stats.totalEpisodes}
            </div>
          </div>
          <div className="rounded-lg border-4 border-primary/20 bg-card p-4 shadow-[8px_8px_0_rgba(0,0,0,0.2)] md:p-6">
            <div className="vcr-text text-xs text-muted-foreground sm:text-sm">
              Total Duration
            </div>
            <div className="mt-2 font-mono text-3xl font-bold text-primary md:text-4xl">
              {formatDuration(stats.totalDuration)}
            </div>
          </div>
        </div>
      )}

      {/* Loading State */}
      {isLoading && (
        <div className="space-y-6">
          {/* Stats Skeleton */}
          <div className="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-4">
            {[1, 2, 3, 4].map((i) => (
              <div
                key={i}
                className="rounded-lg border-4 border-primary/20 bg-card p-4 shadow-[8px_8px_0_rgba(0,0,0,0.2)] md:p-6"
              >
                <Skeleton className="h-4 w-20" />
                <Skeleton className="mt-2 h-10 w-16" />
              </div>
            ))}
          </div>
          {/* Tree Skeleton */}
          <div className="rounded-xl border-4 border-primary/30 bg-card p-4 shadow-[8px_8px_0_rgba(0,0,0,0.2)]">
            <div className="space-y-2">
              {Array.from({ length: 8 }).map((_, i) => (
                <div key={i} className="flex items-center gap-3">
                  <Skeleton className="h-5 w-5" />
                  <Skeleton className="h-5 w-5" />
                  <Skeleton className="h-6 flex-1" />
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Error State */}
      {isError && (
        <div className="rounded-xl border-4 border-destructive bg-card p-8 text-center shadow-[8px_8px_0_rgba(0,0,0,0.6)]">
          <p className="vcr-text text-lg font-bold text-destructive">
            Failed to load media library
          </p>
          <p className="mt-2 text-muted-foreground">Please try again later</p>
          <Button onClick={() => refetch()} className="mt-4" variant="outline">
            Retry
          </Button>
        </div>
      )}

      {/* Empty State - No Media */}
      {!isLoading && !isError && mediaItems.length === 0 && (
        <div className="rounded-xl border-4 border-primary/20 bg-card p-12 text-center shadow-[8px_8px_0_rgba(0,0,0,0.2)]">
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-muted">
            <LibraryIcon className="h-8 w-8 text-muted-foreground" />
          </div>
          <h3 className="vcr-text mb-2 text-xl font-bold">
            Your library is empty
          </h3>
          <p className="mb-6 text-muted-foreground">
            Scan a directory to get started importing media files
          </p>
          <LibraryScanner onScanComplete={() => refetch()} defaultPath="/media" />
        </div>
      )}

      {/* Media Tree View */}
      {!isLoading && !isError && mediaItems.length > 0 && (
        <MediaTree
          media={filteredMedia}
          searchQuery={searchQuery}
          height={600}
          className="mb-8"
        />
      )}

      {/* Empty State - No Results After Filtering */}
      {!isLoading &&
        !isError &&
        mediaItems.length > 0 &&
        filteredMedia.length === 0 && (
          <div className="rounded-xl border-4 border-primary/20 bg-card p-12 text-center shadow-[8px_8px_0_rgba(0,0,0,0.2)]">
            <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-muted">
              <LibraryIcon className="h-8 w-8 text-muted-foreground" />
            </div>
            <h3 className="vcr-text mb-2 text-xl font-bold">No matches found</h3>
            <p className="mb-6 text-muted-foreground">
              No media matched your current filters
            </p>
            <Button onClick={clearFilters} variant="outline">
              <X className="mr-2 h-4 w-4" />
              Clear Filters
            </Button>
          </div>
        )}
    </RetroHeaderLayout>
  );
}
