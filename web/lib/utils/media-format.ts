import { Media } from "@/lib/types/api";

/**
 * Format duration in seconds to HH:MM:SS or MM:SS format
 * @param seconds - Duration in seconds
 * @returns Formatted string like "1:32:45" or "45:30"
 */
export function formatDurationDetailed(seconds: number): string {
  if (seconds === 0) return "0:00";

  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = seconds % 60;

  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, "0")}:${String(secs).padStart(2, "0")}`;
  } else {
    return `${minutes}:${String(secs).padStart(2, "0")}`;
  }
}

/**
 * Format file size in bytes to human-readable format
 * @param bytes - File size in bytes
 * @returns Formatted string like "1.2 GB" or "345 MB"
 */
export function formatFileSize(bytes: number | null | undefined): string {
  if (bytes == null) return "Unknown";
  if (bytes === 0) return "0 B";

  const units = ["B", "KB", "MB", "GB", "TB"];
  const k = 1024;
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return `${(bytes / Math.pow(k, i)).toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

/**
 * Format ISO date string to human-readable format
 * @param dateString - ISO date string
 * @returns Formatted string like "Oct 28, 2025 3:45 PM"
 */
export function formatDate(dateString: string): string {
  const date = new Date(dateString);

  const options: Intl.DateTimeFormatOptions = {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    hour12: true,
  };

  return date.toLocaleString("en-US", options);
}

/**
 * Determine if media requires transcoding and why
 * @param media - Media object
 * @returns Object with compatible flag and array of reasons
 */
export function determineTranscoding(media: Media): {
  compatible: boolean;
  reasons: string[];
} {
  const reasons: string[] = [];

  // Normalize codec strings to lowercase for comparison
  const videoCodec = media.video_codec?.toLowerCase();
  const audioCodec = media.audio_codec?.toLowerCase();

  // Check video codec
  if (!videoCodec) {
    reasons.push("Video codec: unknown → h264 required");
  } else if (videoCodec !== "h264") {
    reasons.push(`Video codec: ${videoCodec} → h264 required`);
  }

  // Check audio codec
  if (!audioCodec) {
    reasons.push("Audio codec: unknown → aac required");
  } else if (audioCodec !== "aac") {
    reasons.push(`Audio codec: ${audioCodec} → aac required`);
  }

  return {
    compatible: reasons.length === 0,
    reasons,
  };
}

