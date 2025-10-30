"use client";

import { useState } from "react";
import { useMediaScan } from "@/hooks/use-media-scan";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Progress } from "@/components/ui/progress";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { FolderSearch, Loader2, ChevronDown, ChevronRight, X } from "lucide-react";
import { cn } from "@/lib/utils";

interface LibraryScannerProps {
  onScanComplete?: () => void;
  defaultPath?: string;
}

export function LibraryScanner({ onScanComplete, defaultPath = "/media" }: LibraryScannerProps) {
  const [path, setPath] = useState(defaultPath);
  const [showPathInput, setShowPathInput] = useState(false);
  const [showProgressModal, setShowProgressModal] = useState(false);
  const [showResultsModal, setShowResultsModal] = useState(false);
  const [errorsExpanded, setErrorsExpanded] = useState(false);

  const { state, progress, error, startScan, cancelScan, reset, elapsedTime } = useMediaScan(
    () => {
      // On scan complete
      onScanComplete?.();
      setShowProgressModal(false);
      setShowResultsModal(true);
    }
  );

  const handleStartScan = async () => {
    setShowProgressModal(true);
    await startScan(path);
  };

  const handleCloseProgress = () => {
    if (state === "scanning") {
      cancelScan();
    }
    setShowProgressModal(false);
  };

  const handleCloseResults = () => {
    setShowResultsModal(false);
    reset();
  };

  const isScanning = state === "scanning";
  const percentComplete = progress
    ? progress.total_files > 0
      ? Math.round((progress.processed_files / progress.total_files) * 100)
      : 0
    : 0;

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, "0")}`;
  };

  return (
    <>
      <div className="flex gap-2 items-end">
        {showPathInput && (
          <div className="flex-1">
            <Label htmlFor="scan-path" className="text-xs">
              Library Path
            </Label>
            <Input
              id="scan-path"
              value={path}
              onChange={(e) => setPath(e.target.value)}
              placeholder="/path/to/media"
              className="mt-1"
            />
          </div>
        )}
        <Button
          onClick={() => setShowPathInput(!showPathInput)}
          variant="outline"
          size="sm"
          className="shrink-0"
        >
          {showPathInput ? "Hide" : "Show"} Path
        </Button>
        <Button
          onClick={handleStartScan}
          disabled={isScanning || !path}
          className="shrink-0"
        >
          {isScanning ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Scanning...
            </>
          ) : (
            <>
              <FolderSearch className="mr-2 h-4 w-4" />
              Scan Library
            </>
          )}
        </Button>
      </div>

      {/* Progress Modal */}
      <Dialog open={showProgressModal} onOpenChange={setShowProgressModal}>
        <DialogContent className="sm:max-w-2xl border-4 max-h-[90vh] overflow-hidden flex flex-col" onPointerDownOutside={(e) => e.preventDefault()}>
          <DialogHeader className="flex-shrink-0">
            <DialogTitle className="font-mono text-xl">Scanning Library</DialogTitle>
            <DialogDescription>
              Discovering and importing media files...
            </DialogDescription>
          </DialogHeader>

          <ScrollArea className="flex-1 max-h-[60vh] px-1">
            <div className="space-y-4 py-4 pr-4">
            {/* Progress Bar */}
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span className="font-medium">{percentComplete}% Complete</span>
                <span className="text-muted-foreground">
                  {progress?.processed_files || 0} / {progress?.total_files || 0} files
                </span>
              </div>
              <Progress value={percentComplete} className="h-3" />
            </div>

            {/* Stats Grid */}
            <div className="grid grid-cols-2 gap-3 text-sm">
              <div className="border-2 border-border rounded-lg p-3">
                <div className="text-muted-foreground text-xs">Total Files</div>
                <div className="text-2xl font-bold font-mono">
                  {progress?.total_files || 0}
                </div>
              </div>
              <div className="border-2 border-border rounded-lg p-3">
                <div className="text-muted-foreground text-xs">Processed</div>
                <div className="text-2xl font-bold font-mono">
                  {progress?.processed_files || 0}
                </div>
              </div>
              <div className="border-2 border-green-500/50 rounded-lg p-3">
                <div className="text-muted-foreground text-xs">Success</div>
                <div className="text-2xl font-bold font-mono text-green-600 dark:text-green-400">
                  {progress?.success_count || 0}
                </div>
              </div>
              <div className="border-2 border-red-500/50 rounded-lg p-3">
                <div className="text-muted-foreground text-xs">Failed</div>
                <div className="text-2xl font-bold font-mono text-red-600 dark:text-red-400">
                  {progress?.failed_count || 0}
                </div>
              </div>
            </div>

            {/* Current File */}
            {progress?.current_file && (
              <div className="border-2 border-border rounded-lg p-3">
                <div className="text-muted-foreground text-xs mb-1">Current File</div>
                <div className="font-mono text-xs break-all line-clamp-2" title={progress.current_file}>
                  {progress.current_file}
                </div>
              </div>
            )}

            {/* Elapsed Time */}
            <div className="text-center text-sm text-muted-foreground">
              Elapsed: {formatTime(elapsedTime)}
            </div>

            {/* Error Message */}
            {error && (
              <div className="border-2 border-red-500 rounded-lg p-3 text-sm text-red-600 dark:text-red-400">
                {error}
              </div>
            )}
            </div>
          </ScrollArea>

          <DialogFooter className="flex-shrink-0 border-t pt-4">
            <Button onClick={handleCloseProgress} variant="outline">
              <X className="mr-2 h-4 w-4" />
              {isScanning ? "Cancel Scan" : "Close"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Results Modal */}
      <Dialog open={showResultsModal} onOpenChange={setShowResultsModal}>
        <DialogContent className="sm:max-w-md border-4">
          <DialogHeader>
            <DialogTitle className="font-mono text-xl">
              {state === "completed" ? "Scan Complete!" : "Scan Ended"}
            </DialogTitle>
            <DialogDescription>
              {state === "completed"
                ? "Media library scan finished successfully"
                : state === "failed"
                  ? "Scan encountered errors"
                  : "Scan was cancelled"}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            {/* Summary Stats */}
            {progress && state === "completed" && (
              <div className="space-y-3">
                <div className="text-center p-4 border-2 border-border rounded-lg">
                  <div className="text-4xl font-bold font-mono text-green-600 dark:text-green-400">
                    {progress.success_count}
                  </div>
                  <div className="text-sm text-muted-foreground mt-1">
                    {progress.success_count === 1 ? "file" : "files"} added
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3 text-sm">
                  <div className="text-center p-3 border-2 border-border rounded-lg">
                    <div className="font-bold font-mono">{progress.total_files}</div>
                    <div className="text-xs text-muted-foreground">Total Scanned</div>
                  </div>
                  <div className="text-center p-3 border-2 border-border rounded-lg">
                    <div className="font-bold font-mono">
                      {progress.processed_files - progress.success_count}
                    </div>
                    <div className="text-xs text-muted-foreground">Skipped</div>
                  </div>
                </div>

                {/* Errors Section */}
                {progress.errors && progress.errors.length > 0 && (
                  <Collapsible open={errorsExpanded} onOpenChange={setErrorsExpanded}>
                    <CollapsibleTrigger className="w-full">
                      <div
                        className={cn(
                          "flex items-center justify-between p-3 border-2 rounded-lg",
                          "border-red-500/50 hover:border-red-500 transition-colors"
                        )}
                      >
                        <div className="flex items-center gap-2">
                          {errorsExpanded ? (
                            <ChevronDown className="h-4 w-4" />
                          ) : (
                            <ChevronRight className="h-4 w-4" />
                          )}
                          <span className="font-medium text-sm">
                            {progress.errors.length} Error{progress.errors.length !== 1 ? "s" : ""}
                          </span>
                        </div>
                        <span className="text-xs text-muted-foreground">
                          Click to {errorsExpanded ? "hide" : "view"}
                        </span>
                      </div>
                    </CollapsibleTrigger>
                    <CollapsibleContent>
                      <ScrollArea className="h-32 mt-2 border-2 border-border rounded-lg p-3">
                        <div className="space-y-2">
                          {progress.errors.map((err, idx) => (
                            <div
                              key={idx}
                              className="text-xs font-mono text-red-600 dark:text-red-400 pb-2 border-b border-border last:border-0 last:pb-0"
                            >
                              {err}
                            </div>
                          ))}
                        </div>
                      </ScrollArea>
                    </CollapsibleContent>
                  </Collapsible>
                )}
              </div>
            )}

            {/* Failed State */}
            {state === "failed" && (
              <div className="text-center p-4 border-2 border-red-500 rounded-lg">
                <div className="text-red-600 dark:text-red-400 font-medium">
                  {error || "An error occurred during scanning"}
                </div>
              </div>
            )}

            {/* Cancelled State */}
            {state === "cancelled" && (
              <div className="text-center p-4 border-2 border-border rounded-lg">
                <div className="text-muted-foreground">
                  Scan was cancelled. Partial results may have been saved.
                </div>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button onClick={handleCloseResults}>Close</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

