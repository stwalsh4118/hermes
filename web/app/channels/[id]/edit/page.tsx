"use client";

import { use, useState } from "react";
import { useRouter } from "next/navigation";
import { RetroHeaderLayout } from "@/components/layout/retro-header-layout";
import { ChannelForm } from "@/components/channel/channel-form";
import { useChannel, useUpdateChannel, useDeleteChannel } from "@/hooks/use-channels";
import { usePlaylist } from "@/hooks/use-playlist";
import { UpdateChannelRequest } from "@/lib/types/api";
import { LoadingSpinner } from "@/components/common/loading-spinner";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/common/confirm-dialog";
import { Trash2 } from "lucide-react";

export default function EditChannelPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const router = useRouter();
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const { id } = use(params);

  const { data: channel, isLoading: channelLoading, isError: channelError } = useChannel(id);
  const { data: playlistData, isLoading: playlistLoading } = usePlaylist(id);
  const updateChannel = useUpdateChannel(id);
  const deleteChannel = useDeleteChannel();

  const handleSubmit = (data: UpdateChannelRequest, _playlistItems?: unknown) => {
    // In edit mode, playlist items are handled directly by the form, so we ignore the second parameter
    updateChannel.mutate(data, {
      onSuccess: () => {
        router.push("/channels");
      },
    });
  };

  const handleCancel = () => {
    router.push("/channels");
  };

  const handleDelete = () => {
    deleteChannel.mutate(id, {
      onSuccess: () => {
        setShowDeleteDialog(false);
        router.push("/channels");
      },
      onError: () => {
        setShowDeleteDialog(false);
      },
    });
  };

  if (channelLoading || playlistLoading) {
    return (
      <RetroHeaderLayout>
        <div className="flex items-center justify-center py-12">
          <LoadingSpinner />
        </div>
      </RetroHeaderLayout>
    );
  }

  if (channelError || !channel) {
    return (
      <RetroHeaderLayout>
        <div className="bg-card rounded-xl p-8 border-4 border-destructive shadow-[8px_8px_0_rgba(0,0,0,0.6)] text-center">
          <p className="text-destructive font-bold text-lg vcr-text">Failed to load channel</p>
          <p className="text-muted-foreground mt-2">Channel not found or an error occurred</p>
          <Button
            onClick={() => router.push("/channels")}
            className="mt-4 retro-button"
          >
            Back to Channels
          </Button>
        </div>
      </RetroHeaderLayout>
    );
  }

  return (
    <RetroHeaderLayout>
      <div className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-foreground vcr-text">Edit Channel</h1>
          <p className="text-muted-foreground mt-1">Update {channel.name} settings and playlist</p>
        </div>
        <Button
          variant="destructive"
          onClick={() => setShowDeleteDialog(true)}
          className="retro-button bg-destructive text-destructive-foreground hover:bg-destructive/80 shadow-[6px_6px_0_rgba(0,0,0,0.2)] hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)]"
        >
          <Trash2 className="w-4 h-4 mr-2" />
          Delete Channel
        </Button>
      </div>

      <ChannelForm
        mode="edit"
        channel={channel}
        playlist={playlistData?.items || []}
        onSubmit={handleSubmit}
        onCancel={handleCancel}
        isSubmitting={updateChannel.isPending}
      />

      <ConfirmDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title="Delete Channel"
        description={`Are you sure you want to delete "${channel.name}"? This action cannot be undone and will remove all playlist items.`}
        confirmLabel="Delete"
        onConfirm={handleDelete}
        variant="destructive"
      />
    </RetroHeaderLayout>
  );
}

