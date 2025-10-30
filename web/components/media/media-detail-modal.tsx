"use client";

import { useState } from "react";
import { Media } from "@/lib/types/api";
import {
  formatDurationDetailed,
  formatFileSize,
  formatDate,
  determineTranscoding,
} from "@/lib/utils/media-format";
import { useDeleteMedia } from "@/hooks/use-media";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { ConfirmDialog } from "@/components/common/confirm-dialog";
import { Separator } from "@/components/ui/separator";
import {
  Copy,
  Edit,
  Trash2,
  CheckCircle2,
  AlertTriangle,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";

interface MediaDetailModalProps {
  media: Media | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onEdit: () => void;
  onDeleted: () => void;
}

/**
 * Modal for displaying complete media item details
 * Shows metadata, file information, and transcoding requirements
 */
export function MediaDetailModal({
  media,
  open,
  onOpenChange,
  onEdit,
  onDeleted,
}: MediaDetailModalProps) {
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const deleteMutation = useDeleteMedia();

  // Handle delete confirmation
  const handleDelete = async () => {
    if (!media) return;

    try {
      await deleteMutation.mutateAsync(media.id);
      toast.success("Media deleted successfully");
      setShowDeleteConfirm(false);
      onOpenChange(false);
      onDeleted();
    } catch (error) {
      toast.error("Failed to delete media");
      console.error("Delete error:", error);
    }
  };

  // Handle copy file path to clipboard
  const handleCopyPath = () => {
    if (media?.file_path) {
      navigator.clipboard.writeText(media.file_path);
      toast.success("File path copied to clipboard");
    }
  };

  if (!media) return null;

  const transcoding = determineTranscoding(media);
  const isDeleting = deleteMutation.isPending;

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent
          className={cn(
            "max-w-3xl max-h-[90vh] border-4 border-primary/30",
            "shadow-[8px_8px_0_rgba(0,0,0,0.3)]",
            "font-mono"
          )}
        >
          <DialogHeader className="border-b-4 border-primary/20 pb-4">
            <DialogTitle className="vcr-text text-2xl font-bold uppercase tracking-wider text-primary">
              Media Details
            </DialogTitle>
          </DialogHeader>

          <ScrollArea className="max-h-[calc(90vh-200px)] pr-4">
            <div className="space-y-6 py-2">
              {/* Section 1: Metadata Display */}
              <div>
                <h3 className="vcr-text text-lg font-bold uppercase mb-4 text-foreground">
                  Metadata
                </h3>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  {/* Title - Full width */}
                  <div className="col-span-1 sm:col-span-2">
                    <div className="text-xs text-muted-foreground uppercase tracking-wider mb-1">
                      Title
                    </div>
                    <div className="text-lg font-bold text-foreground break-words">
                      {media.title}
                    </div>
                  </div>

                  {/* Show Name */}
                  <div>
                    <div className="text-xs text-muted-foreground uppercase tracking-wider mb-1">
                      Show Name
                    </div>
                    <div className="text-sm font-bold">
                      {media.show_name || "—"}
                    </div>
                  </div>

                  {/* Season / Episode */}
                  <div>
                    <div className="text-xs text-muted-foreground uppercase tracking-wider mb-1">
                      Season / Episode
                    </div>
                    <div className="text-sm font-bold">
                      {media.season != null && media.episode != null
                        ? `S${String(media.season).padStart(2, "0")}E${String(media.episode).padStart(2, "0")}`
                        : "—"}
                    </div>
                  </div>

                  {/* Duration */}
                  <div>
                    <div className="text-xs text-muted-foreground uppercase tracking-wider mb-1">
                      Duration
                    </div>
                    <div className="text-sm font-bold text-primary">
                      {formatDurationDetailed(media.duration)}
                    </div>
                  </div>

                  {/* Resolution */}
                  <div>
                    <div className="text-xs text-muted-foreground uppercase tracking-wider mb-1">
                      Resolution
                    </div>
                    <div className="text-sm font-bold">
                      {media.resolution || "—"}
                    </div>
                  </div>

                  {/* Video Codec */}
                  <div>
                    <div className="text-xs text-muted-foreground uppercase tracking-wider mb-1">
                      Video Codec
                    </div>
                    <div className="text-sm font-bold uppercase">
                      {media.video_codec || "—"}
                    </div>
                  </div>

                  {/* Audio Codec */}
                  <div>
                    <div className="text-xs text-muted-foreground uppercase tracking-wider mb-1">
                      Audio Codec
                    </div>
                    <div className="text-sm font-bold uppercase">
                      {media.audio_codec || "—"}
                    </div>
                  </div>

                  {/* Date Added */}
                  <div>
                    <div className="text-xs text-muted-foreground uppercase tracking-wider mb-1">
                      Date Added
                    </div>
                    <div className="text-sm font-bold">
                      {formatDate(media.created_at)}
                    </div>
                  </div>

                  {/* File Size */}
                  <div>
                    <div className="text-xs text-muted-foreground uppercase tracking-wider mb-1">
                      File Size
                    </div>
                    <div className="text-sm font-bold">
                      {formatFileSize(media.file_size)}
                    </div>
                  </div>
                </div>
              </div>

              <Separator className="border-2 border-primary/20" />

              {/* Section 2: File Information */}
              <div>
                <h3 className="vcr-text text-lg font-bold uppercase mb-4 text-foreground">
                  File Information
                </h3>
                <div className="space-y-3">
                  {/* File Path */}
                  <div>
                    <div className="text-xs text-muted-foreground uppercase tracking-wider mb-1">
                      File Path
                    </div>
                    <div className="flex items-start gap-2">
                      <div className="flex-1 text-sm font-bold break-all bg-muted/30 px-3 py-2 rounded border border-border">
                        {media.file_path}
                      </div>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={handleCopyPath}
                        className="shrink-0 border-2"
                      >
                        <Copy className="w-4 h-4" />
                      </Button>
                    </div>
                  </div>
                </div>
              </div>

              <Separator className="border-2 border-primary/20" />

              {/* Section 3: Transcoding Requirements */}
              <div>
                <h3 className="vcr-text text-lg font-bold uppercase mb-4 text-foreground">
                  Transcoding Requirements
                </h3>
                <div className="space-y-3">
                  {/* Compatibility Badge */}
                  <div className="flex items-center gap-3">
                    {transcoding.compatible ? (
                      <>
                        <CheckCircle2 className="w-5 h-5 text-green-500" />
                        <Badge
                          variant="outline"
                          className="border-2 border-green-500/50 bg-green-500/10 text-green-700 dark:text-green-400 font-bold uppercase"
                        >
                          Compatible
                        </Badge>
                        <span className="text-sm text-muted-foreground">
                          No transcoding required
                        </span>
                      </>
                    ) : (
                      <>
                        <AlertTriangle className="w-5 h-5 text-orange-500" />
                        <Badge
                          variant="outline"
                          className="border-2 border-orange-500/50 bg-orange-500/10 text-orange-700 dark:text-orange-400 font-bold uppercase"
                        >
                          Requires Transcoding
                        </Badge>
                      </>
                    )}
                  </div>

                  {/* Transcoding Reasons */}
                  {!transcoding.compatible && transcoding.reasons.length > 0 && (
                    <div className="bg-muted/30 rounded border-2 border-border p-4">
                      <div className="text-xs text-muted-foreground uppercase tracking-wider mb-2 font-bold">
                        Reasons:
                      </div>
                      <ul className="space-y-1">
                        {transcoding.reasons.map((reason, index) => (
                          <li
                            key={index}
                            className="text-sm flex items-start gap-2"
                          >
                            <span className="text-orange-500">•</span>
                            <span>{reason}</span>
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}

                  {/* Compatibility Info */}
                  <div className="text-xs text-muted-foreground bg-muted/20 rounded p-3 border border-border">
                    <strong>Note:</strong> Compatible media uses H.264 video
                    codec and AAC audio codec, which can be streamed directly
                    without transcoding.
                  </div>
                </div>
              </div>
            </div>
          </ScrollArea>

          <DialogFooter className="border-t-4 border-primary/20 pt-4 gap-2">
            <Button
              variant="outline"
              onClick={() => onOpenChange(false)}
              className="border-2 font-bold uppercase"
            >
              Close
            </Button>
            <Button
              variant="outline"
              onClick={onEdit}
              className="border-2 font-bold uppercase"
            >
              <Edit className="w-4 h-4 mr-2" />
              Edit
            </Button>
            <Button
              variant="destructive"
              onClick={() => setShowDeleteConfirm(true)}
              disabled={isDeleting}
              className="border-2 font-bold uppercase"
            >
              {isDeleting ? (
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              ) : (
                <Trash2 className="w-4 h-4 mr-2" />
              )}
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <ConfirmDialog
        open={showDeleteConfirm}
        onOpenChange={setShowDeleteConfirm}
        title="Delete media?"
        description={`Are you sure you want to delete "${media.title}"? This action cannot be undone and will remove the media from your library.`}
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={handleDelete}
      />
    </>
  );
}

