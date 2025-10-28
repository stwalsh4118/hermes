// Backend model types matching Go structs

export interface Channel {
  id: string;
  name: string;
  icon: string | null;
  start_time: string;
  loop: boolean;
  created_at: string;
  updated_at: string;
}

export interface Media {
  id: string;
  file_path: string;
  title: string;
  show_name: string | null;
  season: number | null;
  episode: number | null;
  duration: number;
  video_codec: string | null;
  audio_codec: string | null;
  resolution: string | null;
  file_size: number | null;
  created_at: string;
}

export interface PlaylistItem {
  id: string;
  channel_id: string;
  media_id: string;
  position: number;
  created_at: string;
  media?: Media;
}

export interface Settings {
  id: number;
  media_library_path: string;
  transcode_quality: "high" | "medium" | "low";
  hardware_accel: "none" | "nvenc" | "qsv" | "vaapi" | "videotoolbox";
  server_port: number;
  updated_at: string;
}

// Request types

export interface CreateChannelRequest {
  name: string;
  icon?: string;
  start_time: string;
  loop: boolean;
}

export interface UpdateChannelRequest {
  name?: string;
  icon?: string;
  start_time?: string;
  loop?: boolean;
}

export interface UpdateMediaRequest {
  title?: string;
  show_name?: string;
  season?: number;
  episode?: number;
}

export interface ScanMediaRequest {
  path: string;
}

export interface MediaQueryParams {
  limit?: number;
  offset?: number;
  show?: string;
}

export interface AddToPlaylistRequest {
  media_id: string;
  position: number;
}

export interface ReorderPlaylistRequest {
  items: {
    item_id: string;
    position: number;
  }[];
}

// Response types

export interface ApiError {
  error: string;
  message: string;
  status: number;
}

export interface HealthResponse {
  status: string;
  database: string;
  time: string;
  details?: {
    database_error?: string;
  };
}

export interface ScanResponse {
  scan_id: string;
  message: string;
}

export interface ScanProgress {
  scan_id: string;
  status: "running" | "completed" | "cancelled" | "failed";
  total_files: number;
  processed_files: number;
  success_count: number;
  failed_count: number;
  current_file: string;
  start_time: string;
  end_time: string | null;
  errors: string[];
}

export interface PaginatedMediaResponse {
  items: Media[];
  total: number;
  limit: number;
  offset: number;
}

export interface MessageResponse {
  message: string;
}

export interface PlaylistResponse {
  items: PlaylistItem[];
  total_duration_seconds: number;
}

export interface ChannelsResponse {
  channels: Channel[];
}

