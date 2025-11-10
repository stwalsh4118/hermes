"use client";

import { use } from "react";
import Link from "next/link";
import { RetroHeaderLayout } from "@/components/layout/retro-header-layout";
import { useChannel } from "@/hooks/use-channels";
import { LoadingSpinner } from "@/components/common/loading-spinner";
import { Button } from "@/components/ui/button";
import { ArrowLeft } from "lucide-react";
import { VideoPlayer } from "@/components/video";

export default function ChannelPlayerPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const { data: channel, isLoading, isError } = useChannel(id);

  // Loading State
  if (isLoading) {
    return (
      <RetroHeaderLayout>
        <div className="flex items-center justify-center py-12">
          <LoadingSpinner />
        </div>
      </RetroHeaderLayout>
    );
  }

  // Error State
  if (isError || !channel) {
    return (
      <RetroHeaderLayout>
        <div className="bg-card rounded-xl p-8 border-4 border-destructive shadow-[8px_8px_0_rgba(0,0,0,0.6)] text-center">
          <p className="text-destructive font-bold text-lg vcr-text">Channel not found</p>
          <p className="text-muted-foreground mt-2">
            The channel you&apos;re looking for doesn&apos;t exist or has been deleted
          </p>
          <Button
            asChild
            className="mt-4 retro-button"
          >
            <Link href="/channels">
              <ArrowLeft className="w-4 h-4 mr-2" />
              Back to Channels
            </Link>
          </Button>
        </div>
      </RetroHeaderLayout>
    );
  }

  // Success State
  return (
    <RetroHeaderLayout>
      {/* Page Header with Channel Name and Back Button */}
      <div className="mb-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Button
              asChild
              variant="outline"
              className="retro-button"
            >
              <Link href="/channels">
                <ArrowLeft className="w-4 h-4 mr-2" />
                Back
              </Link>
            </Button>
            <h1 className="text-3xl font-bold text-foreground vcr-text">{channel.name}</h1>
          </div>
        </div>
      </div>

      {/* Video Player */}
      <div className="rounded-xl overflow-hidden border-4 border-primary shadow-[8px_8px_0_rgba(0,0,0,0.6)]">
        <VideoPlayer channelId={channel.id} autoplay={true} />
      </div>
    </RetroHeaderLayout>
  );
}

