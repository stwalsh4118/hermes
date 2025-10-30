"use client";

import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { Media } from "@/lib/types/api";
import { useUpdateMedia } from "@/hooks/use-media";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { ConfirmDialog } from "@/components/common/confirm-dialog";
import { Loader2, Save, X } from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";

// Validation schema - handles string inputs that will be converted to numbers
const mediaEditSchema = z.object({
  title: z.string().min(1, "Title is required"),
  show_name: z.string().optional(),
  season: z
    .string()
    .optional()
    .refine(
      (val) => {
        if (!val || val === "") return true;
        const num = Number(val);
        return !isNaN(num) && Number.isInteger(num) && num > 0;
      },
      { message: "Season must be a positive integer" }
    ),
  episode: z
    .string()
    .optional()
    .refine(
      (val) => {
        if (!val || val === "") return true;
        const num = Number(val);
        return !isNaN(num) && Number.isInteger(num) && num > 0;
      },
      { message: "Episode must be a positive integer" }
    ),
});

type MediaEditFormValues = z.infer<typeof mediaEditSchema>;

interface MediaEditorModalProps {
  media: Media | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved: (updatedMedia: Media) => void;
}

/**
 * Modal for editing media metadata
 * Provides form fields for title, show name, season, and episode with validation
 */
export function MediaEditorModal({
  media,
  open,
  onOpenChange,
  onSaved,
}: MediaEditorModalProps) {
  const [showDiscardConfirm, setShowDiscardConfirm] = useState(false);
  const updateMutation = useUpdateMedia(media?.id || "");

  const form = useForm<MediaEditFormValues>({
    resolver: zodResolver(mediaEditSchema),
    defaultValues: {
      title: "",
      show_name: "",
      season: "",
      episode: "",
    },
  });

  // Reset form when media changes or modal opens
  useEffect(() => {
    if (media && open) {
      form.reset({
        title: media.title || "",
        show_name: media.show_name || "",
        season: media.season != null ? String(media.season) : "",
        episode: media.episode != null ? String(media.episode) : "",
      });
    }
  }, [media, open, form]);

  // Handle form submission
  const onSubmit = async (values: MediaEditFormValues) => {
    if (!media) return;

    try {
      // Prepare update data - only include non-empty values
      const updateData: {
        title?: string;
        show_name?: string;
        season?: number;
        episode?: number;
      } = {
        title: values.title,
      };

      // Only include show_name if it's not empty
      if (values.show_name && values.show_name.trim() !== "") {
        updateData.show_name = values.show_name;
      }

      // Convert and include season if it has a value
      if (values.season && values.season.trim() !== "") {
        const seasonNum = Number(values.season);
        if (!isNaN(seasonNum)) {
          updateData.season = seasonNum;
        }
      }

      // Convert and include episode if it has a value
      if (values.episode && values.episode.trim() !== "") {
        const episodeNum = Number(values.episode);
        if (!isNaN(episodeNum)) {
          updateData.episode = episodeNum;
        }
      }

      const updatedMedia = await updateMutation.mutateAsync(updateData);
      toast.success("Media updated successfully");
      onSaved(updatedMedia);
      onOpenChange(false);
    } catch (error) {
      toast.error("Failed to update media");
      console.error("Update error:", error);
    }
  };

  // Handle modal close with unsaved changes check
  const handleClose = () => {
    if (form.formState.isDirty) {
      setShowDiscardConfirm(true);
    } else {
      onOpenChange(false);
    }
  };

  // Handle discard confirmation
  const handleDiscard = () => {
    form.reset();
    setShowDiscardConfirm(false);
    onOpenChange(false);
  };

  if (!media) return null;

  const isSaving = updateMutation.isPending;

  return (
    <>
      <Dialog open={open} onOpenChange={handleClose}>
        <DialogContent
          className={cn(
            "max-w-2xl max-h-[90vh] border-4 border-primary/30",
            "shadow-[8px_8px_0_rgba(0,0,0,0.3)]",
            "font-mono"
          )}
        >
          <DialogHeader className="border-b-4 border-primary/20 pb-4">
            <DialogTitle className="vcr-text text-2xl font-bold uppercase tracking-wider text-primary">
              Edit Media
            </DialogTitle>
          </DialogHeader>

          <ScrollArea className="max-h-[calc(90vh-200px)] pr-4">
            <Form {...form}>
              <form
                onSubmit={form.handleSubmit(onSubmit)}
                className="space-y-6 py-2"
              >
                {/* Title Field */}
                <FormField
                  control={form.control}
                  name="title"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className="text-xs uppercase tracking-wider font-bold">
                        Title *
                      </FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          disabled={isSaving}
                          className="border-2 font-mono"
                          placeholder="Enter title"
                          autoFocus
                        />
                      </FormControl>
                      <FormMessage className="text-xs" />
                    </FormItem>
                  )}
                />

                {/* Show Name Field */}
                <FormField
                  control={form.control}
                  name="show_name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className="text-xs uppercase tracking-wider font-bold">
                        Show Name
                      </FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          disabled={isSaving}
                          className="border-2 font-mono"
                          placeholder="Enter show name (optional)"
                        />
                      </FormControl>
                      <FormMessage className="text-xs" />
                    </FormItem>
                  )}
                />

                {/* Season and Episode Fields - Grid Layout */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  {/* Season Field */}
                  <FormField
                    control={form.control}
                    name="season"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel className="text-xs uppercase tracking-wider font-bold">
                          Season
                        </FormLabel>
                        <FormControl>
                          <Input
                            {...field}
                            type="number"
                            disabled={isSaving}
                            className="border-2 font-mono"
                            placeholder="Season #"
                            min="1"
                          />
                        </FormControl>
                        <FormMessage className="text-xs" />
                      </FormItem>
                    )}
                  />

                  {/* Episode Field */}
                  <FormField
                    control={form.control}
                    name="episode"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel className="text-xs uppercase tracking-wider font-bold">
                          Episode
                        </FormLabel>
                        <FormControl>
                          <Input
                            {...field}
                            type="number"
                            disabled={isSaving}
                            className="border-2 font-mono"
                            placeholder="Episode #"
                            min="1"
                          />
                        </FormControl>
                        <FormMessage className="text-xs" />
                      </FormItem>
                    )}
                  />
                </div>

                {/* Help Text */}
                <div className="text-xs text-muted-foreground bg-muted/20 rounded p-3 border border-border">
                  <strong>Note:</strong> Fields marked with * are required. Season
                  and Episode must be positive integers if provided.
                </div>
              </form>
            </Form>
          </ScrollArea>

          <DialogFooter className="border-t-4 border-primary/20 pt-4 gap-2">
            <Button
              variant="outline"
              onClick={handleClose}
              disabled={isSaving}
              className="border-2 font-bold uppercase"
            >
              <X className="w-4 h-4 mr-2" />
              Cancel
            </Button>
            <Button
              onClick={form.handleSubmit(onSubmit)}
              disabled={isSaving || !form.formState.isValid}
              className="border-2 font-bold uppercase"
            >
              {isSaving ? (
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              ) : (
                <Save className="w-4 h-4 mr-2" />
              )}
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Discard Changes Confirmation Dialog */}
      <ConfirmDialog
        open={showDiscardConfirm}
        onOpenChange={setShowDiscardConfirm}
        title="Discard changes?"
        description="You have unsaved changes. Are you sure you want to discard them?"
        confirmLabel="Discard"
        cancelLabel="Keep Editing"
        variant="destructive"
        onConfirm={handleDiscard}
      />
    </>
  );
}

