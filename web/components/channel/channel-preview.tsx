"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { PlaylistItem } from "@/lib/types/api";

interface ChannelPreviewProps {
  playlist: PlaylistItem[];
  startTime: string;
}

export function ChannelPreview({ playlist, startTime }: ChannelPreviewProps) {
  return (
    <Card className="border-4 border-secondary shadow-[8px_8px_0_rgba(0,0,0,0.2)]">
      <CardHeader>
        <CardTitle className="vcr-text uppercase tracking-wider">
          Channel Preview
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="text-center py-8 space-y-4">
          <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-secondary/20 text-secondary border-2 border-secondary">
            <span className="w-2 h-2 bg-secondary rounded-full animate-pulse"></span>
            <span className="vcr-text font-bold">PREVIEW MODE</span>
          </div>
          <p className="text-muted-foreground">
            Timeline preview will be available in PBI 4
          </p>
          <div className="text-sm text-muted-foreground space-y-1">
            <p>Start time: {new Date(startTime).toLocaleString()}</p>
            <p>Playlist items: {playlist.length}</p>
            {playlist.length > 0 && (
              <p className="text-xs mt-2">
                First item: {playlist[0]?.media?.title || "Unknown"}
              </p>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

