"use client"

import { useState } from "react"
import Link from "next/link"
import { RefreshCw } from "lucide-react"
import { RetroHeaderLayout } from "@/components/layout/retro-header-layout"
import { useMedia, useScanMedia } from "@/hooks/use-media"
import { Skeleton } from "@/components/ui/skeleton"

export default function LibraryPage() {
  const [searchQuery, setSearchQuery] = useState("")
  const [typeFilter, setTypeFilter] = useState("all")
  const [sortOrder, setSortOrder] = useState("newest")

  const { data: mediaResponse, isLoading, isError } = useMedia()
  const scanMediaMutation = useScanMedia()

  const mediaItems = mediaResponse?.items || []

  // Filter and sort media items
  const filteredAndSortedItems = mediaItems
    .filter((item) => {
      const matchesSearch = item.title.toLowerCase().includes(searchQuery.toLowerCase())
      // For now, consider items with season/episode as episodes, others as media
      const itemType = item.season !== null || item.episode !== null ? "episode" : "media"
      const matchesType = typeFilter === "all" || itemType === typeFilter
      return matchesSearch && matchesType
    })
    .sort((a, b) => {
      if (sortOrder === "newest") {
        return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
      } else if (sortOrder === "oldest") {
        return new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
      } else if (sortOrder === "a-z") {
        return a.title.localeCompare(b.title)
      } else if (sortOrder === "z-a") {
        return b.title.localeCompare(a.title)
      }
      return 0
    })

  // Calculate stats
  const totalItems = mediaItems.length
  const totalSize = mediaItems.reduce((sum, item) => sum + (item.file_size || 0), 0)
  const episodes = mediaItems.filter((item) => item.season !== null || item.episode !== null).length
  const movies = mediaItems.filter((item) => item.season === null && item.episode === null).length

  // Format bytes to human readable
  const formatBytes = (bytes: number) => {
    if (bytes === 0) return "0 Bytes"
    const k = 1024
    const sizes = ["Bytes", "KB", "MB", "GB", "TB"]
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + " " + sizes[i]
  }

  // Format duration
  const formatDuration = (seconds: number) => {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    const secs = seconds % 60
    if (hours > 0) {
      return `${hours}:${minutes.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`
    }
    return `${minutes}:${secs.toString().padStart(2, "0")}`
  }

  const handleScanLibrary = () => {
    scanMediaMutation.mutate("/media")
  }

  return (
    <RetroHeaderLayout>
      {/* Page Title and Actions */}
      <div className="mb-8 flex items-center justify-between">
        <div>
          <h2 className="font-mono text-4xl font-bold uppercase tracking-wider vcr-text">Media Library</h2>
          <p className="mt-2 text-muted-foreground">Manage your video collection</p>
        </div>
        <button
          onClick={handleScanLibrary}
          disabled={scanMediaMutation.isPending}
          className="retro-button bg-primary text-primary-foreground hover:bg-primary/80 px-6 py-3 rounded-lg font-bold border-2 border-primary-foreground/20 shadow-[6px_6px_0_rgba(0,0,0,0.2)] hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)] transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
        >
          <RefreshCw className={`w-5 h-5 ${scanMediaMutation.isPending ? "animate-spin" : ""}`} />
          {scanMediaMutation.isPending ? "SCANNING..." : "SCAN LIBRARY"}
        </button>
      </div>

      {/* Search and Filter */}
      <div className="mb-6 flex gap-4">
        <div className="flex-1">
          <input
            type="text"
            placeholder="Search media..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full rounded-lg border-4 border-primary/20 bg-card px-4 py-3 font-mono uppercase tracking-wider shadow-[4px_4px_0_rgba(0,0,0,0.2)] focus:border-primary focus:outline-none"
          />
        </div>
        <select
          value={typeFilter}
          onChange={(e) => setTypeFilter(e.target.value)}
          className="rounded-lg border-4 border-primary/20 bg-card px-6 py-3 font-mono uppercase tracking-wider shadow-[4px_4px_0_rgba(0,0,0,0.2)] focus:border-primary focus:outline-none"
        >
          <option value="all">ALL TYPES</option>
          <option value="episode">EPISODES</option>
          <option value="media">MEDIA</option>
        </select>
        <select
          value={sortOrder}
          onChange={(e) => setSortOrder(e.target.value)}
          className="rounded-lg border-4 border-primary/20 bg-card px-6 py-3 font-mono uppercase tracking-wider shadow-[4px_4px_0_rgba(0,0,0,0.2)] focus:border-primary focus:outline-none"
        >
          <option value="newest">NEWEST FIRST</option>
          <option value="oldest">OLDEST FIRST</option>
          <option value="a-z">A-Z</option>
          <option value="z-a">Z-A</option>
        </select>
      </div>

      {/* Stats */}
      <div className="mb-8 grid grid-cols-4 gap-6">
        <div className="rounded-lg border-4 border-primary/20 bg-card p-6 shadow-[8px_8px_0_rgba(0,0,0,0.2)]">
          <div className="text-sm text-muted-foreground vcr-text">Total Items</div>
          <div className="mt-2 font-mono text-4xl font-bold">{totalItems}</div>
        </div>
        <div className="rounded-lg border-4 border-primary/20 bg-card p-6 shadow-[8px_8px_0_rgba(0,0,0,0.2)]">
          <div className="text-sm text-muted-foreground vcr-text">Total Size</div>
          <div className="mt-2 font-mono text-4xl font-bold text-accent">{formatBytes(totalSize)}</div>
        </div>
        <div className="rounded-lg border-4 border-primary/20 bg-card p-6 shadow-[8px_8px_0_rgba(0,0,0,0.2)]">
          <div className="text-sm text-muted-foreground vcr-text">Episodes</div>
          <div className="mt-2 font-mono text-4xl font-bold text-secondary">{episodes}</div>
        </div>
        <div className="rounded-lg border-4 border-primary/20 bg-card p-6 shadow-[8px_8px_0_rgba(0,0,0,0.2)]">
          <div className="text-sm text-muted-foreground vcr-text">Movies</div>
          <div className="mt-2 font-mono text-4xl font-bold text-primary">{movies}</div>
        </div>
      </div>

      {/* Loading State */}
      {isLoading && (
        <div className="grid grid-cols-3 gap-6">
          {[1, 2, 3, 4, 5, 6].map((i) => (
            <div key={i} className="rounded-lg border-4 border-primary/20 bg-card shadow-[8px_8px_0_rgba(0,0,0,0.2)]">
              <Skeleton className="aspect-video w-full" />
              <div className="p-4 space-y-3">
                <Skeleton className="h-6 w-3/4" />
                <Skeleton className="h-4 w-full" />
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Error State */}
      {isError && (
        <div className="bg-card rounded-xl p-8 border-4 border-destructive shadow-[8px_8px_0_rgba(0,0,0,0.6)] text-center">
          <p className="text-destructive font-bold text-lg vcr-text">Failed to load media library</p>
          <p className="text-muted-foreground mt-2">Please try again later</p>
        </div>
      )}

      {/* Media Grid */}
      {!isLoading && !isError && (
        <>
          {filteredAndSortedItems && filteredAndSortedItems.length > 0 ? (
            <div className="grid grid-cols-3 gap-6">
              {filteredAndSortedItems.map((item) => (
                <div
                  key={item.id}
                  className="group overflow-hidden rounded-lg border-4 border-primary/20 bg-card shadow-[8px_8px_0_rgba(0,0,0,0.2)] transition-all hover:shadow-[4px_4px_0_rgba(0,0,0,0.2)]"
                >
                  {/* Thumbnail */}
                  <div className="crt-screen relative aspect-video overflow-hidden bg-muted">
                    <div className="w-full h-full flex items-center justify-center">
                      <svg className="w-16 h-16 text-muted-foreground/20" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
                      </svg>
                    </div>
                    <div className="absolute inset-0 bg-black/40 opacity-0 transition-opacity group-hover:opacity-100" />
                    <div className="absolute inset-0 flex items-center justify-center opacity-0 transition-opacity group-hover:opacity-100">
                      <button className="retro-button bg-primary text-primary-foreground hover:bg-primary/80 px-6 py-3 rounded-lg font-bold border-2 border-primary-foreground/20 shadow-[4px_4px_0_rgba(0,0,0,0.2)] hover:shadow-[2px_2px_0_rgba(0,0,0,0.2)] transition-all">
                        PREVIEW
                      </button>
                    </div>
                    {item.duration && (
                      <div className="absolute right-2 top-2 rounded bg-black/80 px-2 py-1 font-mono text-xs font-bold">
                        {formatDuration(item.duration)}
                      </div>
                    )}
                  </div>

                  {/* Info */}
                  <div className="p-4">
                    <h3 className="mb-2 font-mono text-lg font-bold uppercase tracking-wider line-clamp-1">
                      {item.title}
                    </h3>
                    <div className="mb-4 flex items-center gap-4 text-sm text-muted-foreground">
                      <span>{item.season !== null || item.episode !== null ? "Episode" : "Media"}</span>
                      <span>•</span>
                      <span>{formatBytes(item.file_size || 0)}</span>
                      <span>•</span>
                      <span>{new Date(item.created_at).toLocaleDateString()}</span>
                    </div>
                    <div className="flex gap-2">
                      <Link href={`/library/${item.id}`} className="flex-1">
                        <button className="retro-button w-full bg-accent text-accent-foreground hover:bg-accent/80 px-4 py-2 rounded-lg font-bold text-sm border-2 border-accent-foreground/20 shadow-[4px_4px_0_rgba(0,0,0,0.2)] hover:shadow-[2px_2px_0_rgba(0,0,0,0.2)] transition-all">
                          EDIT
                        </button>
                      </Link>
                      <button className="retro-button bg-destructive/20 text-destructive hover:bg-destructive/40 px-4 py-2 rounded-lg font-bold text-sm border-2 border-destructive shadow-[4px_4px_0_rgba(0,0,0,0.2)] hover:shadow-[2px_2px_0_rgba(0,0,0,0.2)] transition-all">
                        DELETE
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="bg-card rounded-xl p-12 border-4 border-primary/20 shadow-[8px_8px_0_rgba(0,0,0,0.2)] text-center">
              <p className="text-muted-foreground font-mono text-lg">No media items found</p>
              <button
                onClick={handleScanLibrary}
                className="mt-4 retro-button bg-primary text-primary-foreground hover:bg-primary/80 px-6 py-3 rounded-lg font-bold border-2 border-primary-foreground/20 shadow-[6px_6px_0_rgba(0,0,0,0.2)] hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)] transition-all"
              >
                SCAN LIBRARY
              </button>
            </div>
          )}
        </>
      )}
    </RetroHeaderLayout>
  )
}
