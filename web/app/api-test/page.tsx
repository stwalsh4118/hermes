"use client";

import { useState } from "react";
import { useChannels } from "@/hooks/use-channels";
import { useMedia, useScanMedia, useScanStatus } from "@/hooks/use-media";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";

export default function ApiTestPage() {
  const [scanPath, setScanPath] = useState("/path/to/media");
  const [activeScanId, setActiveScanId] = useState<string | null>(null);

  const {
    data: channels,
    isLoading: channelsLoading,
    error: channelsError,
  } = useChannels();
  const { data: mediaData, isLoading: mediaLoading, error: mediaError } = useMedia();
  const scanMutation = useScanMedia();
  const {
    data: scanStatus,
    isLoading: scanStatusLoading,
  } = useScanStatus(activeScanId || "");

  const handleStartScan = () => {
    scanMutation.mutate(scanPath, {
      onSuccess: (data) => {
        setActiveScanId(data.scan_id);
        toast.success(`Scan started with ID: ${data.scan_id}`);
      },
    });
  };

  return (
    <div className="container mx-auto py-10 space-y-8">
      <div>
        <h1 className="text-4xl font-bold mb-2">API Test</h1>
        <p className="text-muted-foreground">
          Testing API client and TanStack Query integration
        </p>
      </div>

      <div className="flex gap-4">
        <Button onClick={() => toast.success("Toast notification works!")}>
          Test Toast
        </Button>
        <Button variant="outline" onClick={() => toast.error("Error toast works!")}>
          Test Error Toast
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Channels</CardTitle>
        </CardHeader>
        <CardContent>
          {channelsLoading && (
            <div className="space-y-2">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-4 w-1/2" />
            </div>
          )}
          {channelsError && (
            <div className="text-destructive">
              Error: {(channelsError as { message: string }).message}
            </div>
          )}
          {channels && (
            <div>
              <p className="text-sm text-muted-foreground mb-2">
                Found {channels.length} channel{channels.length !== 1 ? "s" : ""}
              </p>
              {channels.length > 0 ? (
                <pre className="text-sm bg-muted p-4 rounded overflow-auto max-h-96">
                  {JSON.stringify(channels, null, 2)}
                </pre>
              ) : (
                <p className="text-sm text-muted-foreground">
                  No channels found. The channel API endpoints may not be implemented yet.
                </p>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Media</CardTitle>
        </CardHeader>
        <CardContent>
          {mediaLoading && (
            <div className="space-y-2">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-4 w-1/2" />
            </div>
          )}
          {mediaError && (
            <div className="text-destructive">
              Error: {(mediaError as { message: string }).message}
            </div>
          )}
          {mediaData && (
            <div>
              <p className="text-sm text-muted-foreground mb-2">
                Found {mediaData.total} media item{mediaData.total !== 1 ? "s" : ""} (showing{" "}
                {mediaData.items.length})
              </p>
              {mediaData.items.length > 0 ? (
                <pre className="text-sm bg-muted p-4 rounded overflow-auto max-h-96">
                  {JSON.stringify(mediaData.items, null, 2)}
                </pre>
              ) : (
                <p className="text-sm text-muted-foreground">
                  No media found. Try scanning your media library first.
                </p>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Media Scan Test</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="scan-path">Media Library Path</Label>
            <div className="flex gap-2">
              <Input
                id="scan-path"
                value={scanPath}
                onChange={(e) => setScanPath(e.target.value)}
                placeholder="/path/to/media"
                disabled={scanMutation.isPending}
              />
              <Button
                onClick={handleStartScan}
                disabled={scanMutation.isPending || !scanPath}
              >
                {scanMutation.isPending ? "Starting..." : "Start Scan"}
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">
              Enter a valid media library path to scan for video files.
            </p>
          </div>

          {activeScanId && (
            <div className="space-y-3 p-4 border rounded-lg bg-muted/50">
              <div className="flex items-center justify-between">
                <h4 className="text-sm font-medium">Active Scan</h4>
                {scanStatus && (
                  <Badge
                    variant={
                      scanStatus.status === "running"
                        ? "default"
                        : scanStatus.status === "completed"
                          ? "secondary"
                          : "destructive"
                    }
                  >
                    {scanStatus.status}
                  </Badge>
                )}
              </div>

              {scanStatusLoading ? (
                <div className="space-y-2">
                  <Skeleton className="h-4 w-full" />
                  <Skeleton className="h-4 w-3/4" />
                </div>
              ) : scanStatus ? (
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Scan ID:</span>
                    <span className="font-mono text-xs">{scanStatus.scan_id}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Progress:</span>
                    <span>
                      {scanStatus.processed_files} / {scanStatus.total_files} files
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Success:</span>
                    <span className="text-green-600 dark:text-green-400">
                      {scanStatus.success_count}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Failed:</span>
                    <span className="text-red-600 dark:text-red-400">
                      {scanStatus.failed_count}
                    </span>
                  </div>
                  {scanStatus.current_file && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Current:</span>
                      <span className="font-mono text-xs truncate max-w-xs" title={scanStatus.current_file}>
                        {scanStatus.current_file}
                      </span>
                    </div>
                  )}
                  {scanStatus.errors && scanStatus.errors.length > 0 && (
                    <div className="mt-2">
                      <p className="text-muted-foreground mb-1">Errors:</p>
                      <div className="text-xs text-destructive space-y-1 max-h-32 overflow-auto">
                        {scanStatus.errors.map((error, i) => (
                          <div key={i} className="p-2 bg-destructive/10 rounded">
                            {error}
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                  {scanStatus.status !== "running" && (
                    <Button
                      variant="outline"
                      size="sm"
                      className="mt-2 w-full"
                      onClick={() => setActiveScanId(null)}
                    >
                      Clear Scan Status
                    </Button>
                  )}
                </div>
              ) : null}
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>React Query DevTools</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            Look for the React Query DevTools icon in the bottom-left corner of the screen. Click
            it to inspect query states, cache, and more.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

