"use client";

import { useEffect, useState } from "react";
import { X, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface BatchDebugData {
  channel_id: string;
  client_count: number;
  furthest_segment: number;
  has_batch: boolean;
  trigger_threshold: number;
  batch: {
    batch_number: number;
    start_segment: number;
    end_segment: number;
    is_complete: boolean;
    segments_remaining: number;
    video_source_path: string;
    video_start_offset: number;
    generation_started: string;
    generation_ended: string | null;
    generation_duration_seconds?: number;
  } | null;
  client_positions: Array<{
    session_id: string;
    segment_number: number;
    quality: string;
    last_updated: string;
  }>;
}

interface BatchDebugOverlayProps {
  channelId: string;
  onClose: () => void;
}

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
const POLL_INTERVAL = 1000; // 1 second

export function BatchDebugOverlay({ channelId, onClose }: BatchDebugOverlayProps) {
  const [data, setData] = useState<BatchDebugData | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);

  const fetchDebugData = async () => {
    try {
      setIsRefreshing(true);
      const response = await fetch(`${API_BASE_URL}/api/stream/${channelId}/debug`);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      const json = await response.json();
      setData(json);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch debug data");
    } finally {
      setIsRefreshing(false);
    }
  };

  useEffect(() => {
    // Initial fetch
    fetchDebugData();

    // Poll every second
    const interval = setInterval(fetchDebugData, POLL_INTERVAL);

    return () => clearInterval(interval);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [channelId]);

  if (!data && !error) {
    return (
      <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
        <Card className="w-full max-w-4xl m-4">
          <CardHeader>
            <CardTitle>Batch Debug</CardTitle>
          </CardHeader>
          <CardContent>
            <p>Loading...</p>
          </CardContent>
        </Card>
      </div>
    );
  }

  const batch = data?.batch;
  const furthestSegment = data?.furthest_segment ?? 0;
  const segmentsRemaining = batch ? batch.segments_remaining : 0;
  const triggerThreshold = data?.trigger_threshold ?? 7; // Fallback to 7 if not provided
  const shouldTrigger = batch ? segmentsRemaining <= triggerThreshold : false;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <Card className="w-full max-w-4xl max-h-[90vh] overflow-y-auto">
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
          <CardTitle>Batch Generation Debug</CardTitle>
          <div className="flex gap-2">
            <Button
              variant="ghost"
              size="icon"
              onClick={fetchDebugData}
              disabled={isRefreshing}
            >
              <RefreshCw className={`h-4 w-4 ${isRefreshing ? "animate-spin" : ""}`} />
            </Button>
            <Button variant="ghost" size="icon" onClick={onClose}>
              <X className="h-4 w-4" />
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {error && (
            <div className="bg-destructive/10 text-destructive p-3 rounded-md">
              Error: {error}
            </div>
          )}

          {data && (
            <>
              {/* Summary */}
              <div className="grid grid-cols-3 gap-4">
                <div className="space-y-1">
                  <p className="text-sm text-muted-foreground">Clients</p>
                  <p className="text-2xl font-bold">{data.client_count}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm text-muted-foreground">Furthest Segment</p>
                  <p className="text-2xl font-bold">{furthestSegment}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm text-muted-foreground">Segments Remaining</p>
                  <p className={`text-2xl font-bold ${shouldTrigger ? "text-yellow-500" : segmentsRemaining <= 3 ? "text-red-500" : "text-green-500"}`}>
                    {segmentsRemaining}
                  </p>
                </div>
              </div>

              {/* Batch State */}
              {batch ? (
                <div className="space-y-4">
                  <div>
                    <h3 className="text-lg font-semibold mb-2">Current Batch</h3>
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <p className="text-sm text-muted-foreground">Batch Number</p>
                        <p className="text-xl font-bold">#{batch.batch_number}</p>
                      </div>
                      <div>
                        <p className="text-sm text-muted-foreground">Status</p>
                        <p className={`text-xl font-bold ${batch.is_complete ? "text-green-500" : "text-yellow-500"}`}>
                          {batch.is_complete ? "Complete" : "Generating..."}
                        </p>
                      </div>
                      <div>
                        <p className="text-sm text-muted-foreground">Segment Range</p>
                        <p className="text-lg">{batch.start_segment} - {batch.end_segment}</p>
                      </div>
                      {batch.generation_duration_seconds && (
                        <div>
                          <p className="text-sm text-muted-foreground">Generation Time</p>
                          <p className="text-lg">{batch.generation_duration_seconds.toFixed(2)}s</p>
                        </div>
                      )}
                    </div>
                  </div>

                  {/* Visual Buffer Representation */}
                  <div>
                    <h3 className="text-lg font-semibold mb-2">Buffer Visualization</h3>
                    <div className="space-y-2">
                      <div className="flex items-center gap-2">
                        <span className="text-sm w-32">Batch Range:</span>
                        <div className="flex-1 flex gap-1">
                          {Array.from({ length: batch.end_segment - batch.start_segment + 1 }, (_, i) => {
                            const segmentNum = batch.start_segment + i;
                            const isPlayed = segmentNum <= furthestSegment;
                            const isCurrent = segmentNum === furthestSegment;
                            const isRemaining = segmentNum > furthestSegment && segmentNum <= batch.end_segment;
                            
                            return (
                              <div
                                key={segmentNum}
                                className={`h-8 flex-1 rounded text-xs flex items-center justify-center font-mono ${
                                  isCurrent
                                    ? "bg-blue-500 text-white ring-2 ring-blue-300"
                                    : isPlayed
                                    ? "bg-gray-600 text-white"
                                    : isRemaining
                                    ? segmentsRemaining <= triggerThreshold && segmentNum > batch.end_segment - triggerThreshold
                                      ? "bg-yellow-500 text-white"
                                      : "bg-green-500 text-white"
                                    : "bg-gray-300 text-gray-600"
                                }`}
                                title={`Segment ${segmentNum}`}
                              >
                                {segmentNum}
                              </div>
                            );
                          })}
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="text-sm w-32">Trigger Zone:</span>
                        <div className="flex-1 flex gap-1">
                          {Array.from({ length: batch.end_segment - batch.start_segment + 1 }, (_, i) => {
                            const segmentNum = batch.start_segment + i;
                            const isInTriggerZone = segmentNum > batch.end_segment - triggerThreshold;
                            
                            return (
                              <div
                                key={segmentNum}
                                className={`h-6 flex-1 rounded text-xs flex items-center justify-center ${
                                  isInTriggerZone
                                    ? "bg-yellow-500/30 border border-yellow-500"
                                    : "bg-transparent"
                                }`}
                              >
                                {isInTriggerZone ? "⚠" : ""}
                              </div>
                            );
                          })}
                        </div>
                      </div>
                      <div className="text-xs text-muted-foreground space-y-1">
                        <div className="flex items-center gap-2">
                          <div className="w-4 h-4 bg-gray-600 rounded"></div>
                          <span>Played</span>
                        </div>
                        <div className="flex items-center gap-2">
                          <div className="w-4 h-4 bg-blue-500 rounded ring-2 ring-blue-300"></div>
                          <span>Current</span>
                        </div>
                        <div className="flex items-center gap-2">
                          <div className="w-4 h-4 bg-green-500 rounded"></div>
                          <span>Available</span>
                        </div>
                        <div className="flex items-center gap-2">
                          <div className="w-4 h-4 bg-yellow-500 rounded"></div>
                          <span>Trigger Zone (≤{triggerThreshold} remaining)</span>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Trigger Status */}
                  <div className={`p-3 rounded-md ${shouldTrigger ? "bg-yellow-500/20 border border-yellow-500" : "bg-green-500/20 border border-green-500"}`}>
                    <p className="font-semibold">
                      {shouldTrigger ? (
                        <>⚠️ Next batch should trigger (segments remaining ≤ {triggerThreshold})</>
                      ) : (
                        <>✓ Buffer healthy (segments remaining: {segmentsRemaining}, threshold: {triggerThreshold})</>
                      )}
                    </p>
                  </div>

                  {/* Video Source Info */}
                  <div>
                    <h3 className="text-lg font-semibold mb-2">Video Source</h3>
                    <div className="space-y-1 text-sm">
                      <p className="font-mono break-all">{batch.video_source_path}</p>
                      <p className="text-muted-foreground">Offset: {batch.video_start_offset}s</p>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="text-center py-8 text-muted-foreground">
                  No batch data available. Stream may be starting...
                </div>
              )}

              {/* Client Positions */}
              {data.client_positions.length > 0 && (
                <div>
                  <h3 className="text-lg font-semibold mb-2">Client Positions</h3>
                  <div className="space-y-2">
                    {data.client_positions.map((client) => (
                      <div key={client.session_id} className="flex items-center justify-between p-2 bg-muted rounded">
                        <div>
                          <p className="font-mono text-xs">{client.session_id.substring(0, 8)}...</p>
                          <p className="text-sm text-muted-foreground">{client.quality}</p>
                        </div>
                        <div className="text-right">
                          <p className="font-bold">Segment {client.segment_number}</p>
                          <p className="text-xs text-muted-foreground">
                            {new Date(client.last_updated).toLocaleTimeString()}
                          </p>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

