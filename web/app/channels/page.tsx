"use client"

import { useState } from "react"
import Link from "next/link"
import { RetroHeaderLayout } from "@/components/layout/retro-header-layout"
import { useChannels, useDeleteChannel } from "@/hooks/use-channels"
import { Skeleton } from "@/components/ui/skeleton"
import { ConfirmDialog } from "@/components/common/confirm-dialog"

export default function ChannelsPage() {
  const { data: channels, isLoading, isError } = useChannels()
  const deleteChannel = useDeleteChannel()
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [channelToDelete, setChannelToDelete] = useState<{ id: string; name: string } | null>(null)

  const handleDeleteClick = (id: string, name: string) => {
    setChannelToDelete({ id, name })
    setDeleteDialogOpen(true)
  }

  const handleConfirmDelete = () => {
    if (channelToDelete) {
      deleteChannel.mutate(channelToDelete.id)
    }
    setDeleteDialogOpen(false)
    setChannelToDelete(null)
  }

  // Calculate stats
  const totalChannels = channels?.length || 0
  const totalViewers = 0 // Not implemented yet in API
  const totalContent = 0 // Not implemented yet in API

  return (
    <RetroHeaderLayout>
      {/* Page Header */}
      <div className="mb-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-foreground vcr-text">Channel Management</h1>
            <p className="text-muted-foreground mt-1">Manage your virtual TV channels</p>
          </div>
          <Link href="/channels/new">
            <button className="retro-button bg-primary text-primary-foreground hover:bg-primary/80 px-6 py-3 rounded-lg font-bold border-2 border-primary-foreground/20 shadow-[8px_8px_0_rgba(0,0,0,0.2)] hover:shadow-[4px_4px_0_rgba(0,0,0,0.2)] transition-all">
              + CREATE CHANNEL
            </button>
          </Link>
        </div>
      </div>

      {/* Loading State */}
      {isLoading && (
        <div className="space-y-8">
          <div className="bg-card rounded-xl overflow-hidden border-4 border-primary shadow-[8px_8px_0_rgba(0,0,0,0.6)]">
            <Skeleton className="h-96 w-full" />
          </div>
        </div>
      )}

      {/* Error State */}
      {isError && (
        <div className="bg-card rounded-xl p-8 border-4 border-destructive shadow-[8px_8px_0_rgba(0,0,0,0.6)] text-center">
          <p className="text-destructive font-bold text-lg vcr-text">Failed to load channels</p>
          <p className="text-muted-foreground mt-2">Please try again later</p>
        </div>
      )}

      {/* Channels Table */}
      {!isLoading && !isError && (
        <>
          <div className="bg-card rounded-xl overflow-hidden border-4 border-primary shadow-[8px_8px_0_rgba(0,0,0,0.6)]">
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-muted/50 border-b-4 border-primary">
                  <tr>
                    <th className="text-left px-6 py-4 font-bold text-foreground vcr-text">Channel</th>
                    <th className="text-left px-6 py-4 font-bold text-foreground vcr-text">Status</th>
                    <th className="text-left px-6 py-4 font-bold text-foreground vcr-text">Uptime</th>
                    <th className="text-right px-6 py-4 font-bold text-foreground vcr-text">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {channels && channels.length > 0 ? (
                    channels.map((channel) => (
                      <tr key={channel.id} className="border-b border-border hover:bg-muted/30 transition-colors">
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-4">
                            <div className="relative w-24 h-16 rounded-lg overflow-hidden border-2 border-primary/30 shadow-[4px_4px_0_rgba(0,0,0,0.2)]">
                              {channel.icon ? (
                                <div className="w-full h-full flex items-center justify-center bg-muted text-3xl">
                                  {channel.icon}
                                </div>
                              ) : (
                                <div className="w-full h-full flex items-center justify-center bg-muted">
                                  <svg className="w-8 h-8 text-muted-foreground/20" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
                                  </svg>
                                </div>
                              )}
                            </div>
                            <div>
                              <div className="font-bold text-foreground vcr-text">{channel.name}</div>
                              <div className="text-sm text-muted-foreground mt-1">
                                Loop: {channel.loop ? "Yes" : "No"} • Start: {new Date(channel.start_time).toLocaleTimeString()}
                              </div>
                            </div>
                          </div>
                        </td>
                        <td className="px-6 py-4">
                          <span className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-primary/20 text-primary border-2 border-primary font-bold text-sm">
                            <span className="w-2 h-2 bg-primary rounded-full"></span>
                            READY
                          </span>
                        </td>
                        <td className="px-6 py-4">
                          <div className="text-muted-foreground vcr-text">24/7 Broadcast</div>
                        </td>
                        <td className="px-6 py-4">
                          <div className="flex items-center justify-end gap-2">
                            <Link href={`/channels/${channel.id}/edit`}>
                              <button className="retro-button bg-accent text-accent-foreground hover:bg-accent/80 px-4 py-2 rounded-lg font-bold text-sm border-2 border-accent-foreground/20 shadow-[6px_6px_0_rgba(0,0,0,0.2)] hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)] transition-all">
                                EDIT
                              </button>
                            </Link>
                            <button
                              onClick={() => handleDeleteClick(channel.id, channel.name)}
                              className="retro-button bg-destructive/20 text-destructive hover:bg-destructive/40 px-4 py-2 rounded-lg font-bold text-sm border-2 border-destructive shadow-[6px_6px_0_rgba(0,0,0,0.2)] hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)] transition-all"
                            >
                              DELETE
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan={4} className="px-6 py-12 text-center">
                        <p className="text-muted-foreground font-mono text-lg">No channels found</p>
                        <Link href="/channels/new">
                          <button className="mt-4 retro-button bg-primary text-primary-foreground hover:bg-primary/80 px-6 py-3 rounded-lg font-bold border-2 border-primary-foreground/20 shadow-[6px_6px_0_rgba(0,0,0,0.2)] hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)] transition-all">
                            CREATE YOUR FIRST CHANNEL
                          </button>
                        </Link>
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>

          {/* Stats Cards */}
          {channels && channels.length > 0 && (
            <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mt-8">
              <div className="bg-card rounded-xl p-6 border-4 border-primary shadow-[8px_8px_0_rgba(0,0,0,0.6)]">
                <div className="text-muted-foreground text-sm mb-2 vcr-text">Total Channels</div>
                <div className="text-4xl font-bold text-foreground vcr-text">{totalChannels}</div>
              </div>
              <div className="bg-card rounded-xl p-6 border-4 border-accent shadow-[8px_8px_0_rgba(0,0,0,0.6)]">
                <div className="text-muted-foreground text-sm mb-2 vcr-text">Content Hours</div>
                <div className="text-4xl font-bold text-accent vcr-text">0</div>
              </div>
              <div className="bg-card rounded-xl p-6 border-4 border-accent shadow-[8px_8px_0_rgba(0,0,0,0.6)]">
                <div className="text-muted-foreground text-sm mb-2 vcr-text">Total Viewers</div>
                <div className="text-4xl font-bold text-accent vcr-text">{totalViewers}</div>
              </div>
              <div className="bg-card rounded-xl p-6 border-4 border-secondary shadow-[8px_8px_0_rgba(0,0,0,0.6)]">
                <div className="text-muted-foreground text-sm mb-2 vcr-text">Total Content</div>
                <div className="text-4xl font-bold text-secondary vcr-text">{totalContent}</div>
              </div>
            </div>
          )}
        </>
      )}

      <ConfirmDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        title="Delete Channel"
        description={
          channelToDelete
            ? `Are you sure you want to delete "${channelToDelete.name}"? This action cannot be undone and will remove all playlist items.`
            : ""
        }
        confirmLabel="Delete"
        onConfirm={handleConfirmDelete}
        variant="destructive"
      />
    </RetroHeaderLayout>
  )
}
