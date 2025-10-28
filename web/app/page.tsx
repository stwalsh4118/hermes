"use client"

import { Card } from "@/components/ui/card"
import { Play, Plus, Settings, Radio } from "lucide-react"
import { ThemeToggle } from "@/components/theme-toggle"
import Link from "next/link"
import { useChannels } from "@/hooks/use-channels"
import { Skeleton } from "@/components/ui/skeleton"

export default function DashboardPage() {
  const { data: channels, isLoading, isError } = useChannels()

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b-4 border-primary backdrop-blur-sm sticky top-0 z-50 bg-card shadow-lg">
        <div className="container mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="w-12 h-12 rounded-lg bg-primary border-4 border-primary/30 flex items-center justify-center shadow-lg">
                <Radio className="w-7 h-7 text-primary-foreground" />
              </div>
              <h1 className="text-2xl font-bold text-foreground vcr-text">Virtual TV</h1>
            </div>
            <nav className="flex items-center gap-4">
              <Link href="/channels">
                <button className="retro-button bg-muted text-foreground hover:bg-muted/60 px-4 py-2 rounded-lg font-bold text-sm border-2 border-primary/30 shadow-[6px_6px_0_rgba(0,0,0,0.2)] hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)] transition-all">
                  CHANNELS
                </button>
              </Link>
              <Link href="/library">
                <button className="retro-button bg-muted text-foreground hover:bg-muted/60 px-4 py-2 rounded-lg font-bold text-sm border-2 border-primary/30 shadow-[6px_6px_0_rgba(0,0,0,0.2)] hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)] transition-all">
                  LIBRARY
                </button>
              </Link>
              <ThemeToggle />
              <Link href="/settings">
                <button className="retro-button bg-muted text-foreground hover:bg-muted/60 p-3 rounded-lg border-2 border-primary/30 shadow-[6px_6px_0_rgba(0,0,0,0.2)] hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)] transition-all">
                  <Settings className="w-5 h-5" />
                </button>
              </Link>
            </nav>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="container mx-auto px-6 py-8">
        {/* Hero Section */}
        <div className="mb-12">
          <div className="flex items-center justify-between mb-6">
            <div>
              <h2 className="text-4xl font-bold text-foreground mb-2 text-balance vcr-text">Your Channels</h2>
              <p className="text-lg text-muted-foreground font-mono">Saturday morning cartoons, anytime you want</p>
            </div>
            <Link href="/channels/new">
              <button className="retro-button bg-primary text-primary-foreground hover:bg-primary/80 px-6 py-3 rounded-lg font-bold border-2 border-primary-foreground/20 shadow-[8px_8px_0_rgba(0,0,0,0.2)] hover:shadow-[4px_4px_0_rgba(0,0,0,0.2)] transition-all">
                <Plus className="w-5 h-5 inline mr-2" />
                CREATE CHANNEL
              </button>
            </Link>
          </div>
        </div>

        {/* Channel Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
          {isLoading ? (
            // Loading skeletons
            <>
              {[1, 2, 3].map((i) => (
                <Card key={i} className="overflow-hidden border-4 border-primary/30">
                  <Skeleton className="aspect-video w-full" />
                  <div className="p-6 space-y-3">
                    <Skeleton className="h-6 w-3/4" />
                    <Skeleton className="h-4 w-full" />
                    <div className="flex justify-between">
                      <Skeleton className="h-4 w-1/3" />
                      <Skeleton className="h-4 w-1/4" />
                    </div>
                  </div>
                </Card>
              ))}
            </>
          ) : isError ? (
            // Error state
            <div className="col-span-full text-center py-12">
              <p className="text-destructive font-mono text-lg">Failed to load channels</p>
            </div>
          ) : channels && channels.length > 0 ? (
            // Render actual channels
            <>
              {channels.map((channel) => (
                <Link key={channel.id} href={`/channels/${channel.id}`}>
                  <Card className="overflow-hidden group cursor-pointer transition-smooth hover:scale-[1.02] border-4 border-primary/30 shadow-[8px_8px_0_rgba(0,0,0,0.2)] hover:shadow-[4px_4px_0_rgba(0,0,0,0.2)] bg-card">
                    <div className="aspect-video bg-muted relative overflow-hidden crt-screen">
                      {channel.icon ? (
                        <div className="w-full h-full flex items-center justify-center bg-muted">
                          <span className="text-6xl">{channel.icon}</span>
                        </div>
                      ) : (
                        <div className="w-full h-full flex items-center justify-center bg-muted">
                          <Radio className="w-16 h-16 text-muted-foreground/20" />
                        </div>
                      )}
                      <div className="absolute top-3 left-3 flex items-center gap-2 bg-destructive backdrop-blur-sm px-4 py-2 border-2 border-white shadow-lg">
                        <div className="w-3 h-3 rounded-full bg-white live-pulse" />
                        <span className="text-xs font-bold text-white uppercase tracking-wider font-mono">
                          Live
                        </span>
                      </div>
                      <div className="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center">
                        <button className="retro-button bg-primary text-primary-foreground hover:bg-primary/80 px-6 py-3 rounded-lg font-bold border-2 border-primary-foreground/20 shadow-[6px_6px_0_rgba(0,0,0,0.2)] hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)] transition-all w-20 h-20 flex items-center justify-center">
                          <Play className="w-10 h-10 fill-current" />
                        </button>
                      </div>
                    </div>
                    <div className="p-6 bg-card border-t-4 border-primary/20">
                      <h3 className="text-xl font-bold text-foreground mb-2 font-mono uppercase tracking-wide">
                        {channel.name}
                      </h3>
                      <p className="text-sm text-muted-foreground mb-4 font-mono">
                        Loop: {channel.loop ? 'Yes' : 'No'} • Start: {new Date(channel.start_time).toLocaleTimeString()}
                      </p>
                      <div className="flex items-center justify-between">
                        <span className="text-xs text-muted-foreground font-mono uppercase tracking-wide">
                          24/7 Broadcast
                        </span>
                        <span className="text-xs font-bold text-primary font-mono px-3 py-1 bg-primary/10 border-2 border-primary rounded">
                          Live
                        </span>
                      </div>
                    </div>
                  </Card>
                </Link>
              ))}
              
              {/* Create New Channel Card */}
              <Link href="/channels/new">
                <Card className="overflow-hidden group cursor-pointer transition-smooth hover:scale-[1.02] border-4 border-dashed border-primary/50 shadow-[8px_8px_0_rgba(0,0,0,0.2)] hover:shadow-[4px_4px_0_rgba(0,0,0,0.2)] bg-card/50">
                  <div className="aspect-video bg-muted/30 relative overflow-hidden flex items-center justify-center">
                    <div className="text-center p-6">
                      <div className="w-20 h-20 rounded-lg bg-primary/20 border-4 border-primary/40 flex items-center justify-center mx-auto mb-4 shadow-lg">
                        <Plus className="w-10 h-10 text-primary" />
                      </div>
                      <h3 className="text-lg font-bold text-foreground mb-2 font-mono uppercase tracking-wide">
                        Create New Channel
                      </h3>
                      <p className="text-sm text-muted-foreground font-mono">Start broadcasting your content</p>
                    </div>
                  </div>
                  <div className="p-6 border-t-4 border-primary/20">
                    <button className="retro-button w-full bg-primary text-primary-foreground hover:bg-primary/80 px-6 py-3 rounded-lg font-bold border-2 border-primary-foreground/20 shadow-[8px_8px_0_rgba(0,0,0,0.2)] hover:shadow-[4px_4px_0_rgba(0,0,0,0.2)] transition-all">
                      GET STARTED
                    </button>
                  </div>
                </Card>
              </Link>
            </>
          ) : (
            // Empty state - only show create card
            <Link href="/channels/new">
              <Card className="overflow-hidden group cursor-pointer transition-smooth hover:scale-[1.02] border-4 border-dashed border-primary/50 shadow-[8px_8px_0_rgba(0,0,0,0.2)] hover:shadow-[4px_4px_0_rgba(0,0,0,0.2)] bg-card/50">
                <div className="aspect-video bg-muted/30 relative overflow-hidden flex items-center justify-center">
                  <div className="text-center p-6">
                    <div className="w-20 h-20 rounded-lg bg-primary/20 border-4 border-primary/40 flex items-center justify-center mx-auto mb-4 shadow-lg">
                      <Plus className="w-10 h-10 text-primary" />
                    </div>
                    <h3 className="text-lg font-bold text-foreground mb-2 font-mono uppercase tracking-wide">
                      Create Your First Channel
                    </h3>
                    <p className="text-sm text-muted-foreground font-mono">Start broadcasting your content</p>
                  </div>
                </div>
                <div className="p-6 border-t-4 border-primary/20">
                  <button className="retro-button w-full bg-primary text-primary-foreground hover:bg-primary/80 px-6 py-3 rounded-lg font-bold border-2 border-primary-foreground/20 shadow-[8px_8px_0_rgba(0,0,0,0.2)] hover:shadow-[4px_4px_0_rgba(0,0,0,0.2)] transition-all">
                    GET STARTED
                  </button>
                </div>
              </Card>
            </Link>
          )}
        </div>
      </main>
    </div>
  )
}
