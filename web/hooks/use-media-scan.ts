import { useState, useEffect, useCallback } from "react";
import { useScanMedia, useScanStatus } from "./use-media";
import { ScanProgress } from "@/lib/types/api";
import { toast } from "sonner";

export type ScanState = "idle" | "scanning" | "completed" | "failed" | "cancelled";

interface UseScanResult {
  state: ScanState;
  scanId: string | null;
  progress: ScanProgress | null;
  error: string | null;
  startScan: (path: string) => Promise<void>;
  cancelScan: () => void;
  reset: () => void;
  elapsedTime: number;
}

export function useMediaScan(onComplete?: () => void): UseScanResult {
  const [state, setState] = useState<ScanState>("idle");
  const [scanId, setScanId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [elapsedTime, setElapsedTime] = useState(0);
  const [startTime, setStartTime] = useState<number | null>(null);

  const scanMutation = useScanMedia();
  
  // Use the auto-polling query hook for scan status
  const { data: progress } = useScanStatus(scanId || "");

  // Calculate elapsed time
  useEffect(() => {
    if (state === "scanning" && startTime) {
      const interval = setInterval(() => {
        setElapsedTime(Math.floor((Date.now() - startTime) / 1000));
      }, 1000);
      return () => clearInterval(interval);
    }
  }, [state, startTime]);

  // Monitor scan status changes
  useEffect(() => {
    if (!progress || state !== "scanning") return;

    switch (progress.status) {
      case "completed":
        setState("completed");
        toast.success(`Scan complete! ${progress.success_count} files added`);
        onComplete?.();
        break;
      case "failed":
        setState("failed");
        setError("Scan failed");
        toast.error("Scan failed");
        break;
      case "cancelled":
        setState("cancelled");
        toast.info("Scan cancelled");
        break;
      case "running":
        // Still scanning, do nothing
        break;
    }
  }, [progress, state, onComplete]);

  const startScan = useCallback(
    async (path: string) => {
      try {
        setError(null);
        setState("scanning");
        setStartTime(Date.now());
        setElapsedTime(0);

        const response = await scanMutation.mutateAsync(path);
        setScanId(response.scan_id);
      } catch (err: any) {
        setState("failed");
        setError(err.message || "Failed to start scan");
        
        // Handle specific error cases
        if (err.status === 409) {
          toast.error("A scan is already running");
        } else if (err.status === 400) {
          toast.error("Invalid path provided");
        } else {
          toast.error(err.message || "Failed to start scan");
        }
      }
    },
    [scanMutation]
  );

  const cancelScan = useCallback(() => {
    // Note: Cancel API endpoint doesn't exist yet
    // For now, just update local state
    setState("cancelled");
    toast.info("Scan cancelled (client-side)");
  }, []);

  const reset = useCallback(() => {
    setState("idle");
    setScanId(null);
    setError(null);
    setElapsedTime(0);
    setStartTime(null);
  }, []);

  return {
    state,
    scanId,
    progress: progress || null,
    error,
    startScan,
    cancelScan,
    reset,
    elapsedTime,
  };
}

