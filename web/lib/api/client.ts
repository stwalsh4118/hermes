import {
  ApiError,
  Channel,
  CreateChannelRequest,
  HealthResponse,
  Media,
  MediaQueryParams,
  MessageResponse,
  PaginatedMediaResponse,
  ScanProgress,
  ScanResponse,
  UpdateChannelRequest,
  UpdateMediaRequest,
} from "@/lib/types/api";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

class ApiClient {
  private baseURL: string;

  constructor(baseURL: string) {
    this.baseURL = baseURL;
  }

  private async request<T>(endpoint: string, options?: RequestInit): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;

    const config: RequestInit = {
      ...options,
      headers: {
        "Content-Type": "application/json",
        ...options?.headers,
      },
    };

    try {
      const response = await fetch(url, config);

      if (!response.ok) {
        const error: ApiError = await response.json().catch(() => ({
          error: "Unknown Error",
          message: response.statusText,
          status: response.status,
        }));
        throw error;
      }

      // Handle 204 No Content
      if (response.status === 204) {
        return {} as T;
      }

      return response.json();
    } catch (error) {
      if ((error as ApiError).status !== undefined) {
        // Already an ApiError, rethrow
        throw error;
      }
      // Network or other error
      if (error instanceof Error) {
        throw {
          error: "Network Error",
          message: error.message,
          status: 0,
        } as ApiError;
      }
      throw error;
    }
  }

  // Health check
  async health() {
    return this.request<HealthResponse>("/api/health");
  }

  // Channel endpoints
  async getChannels() {
    return this.request<Channel[]>("/api/channels");
  }

  async getChannel(id: string) {
    return this.request<Channel>(`/api/channels/${id}`);
  }

  async createChannel(data: CreateChannelRequest) {
    return this.request<Channel>("/api/channels", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  async updateChannel(id: string, data: UpdateChannelRequest) {
    return this.request<Channel>(`/api/channels/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    });
  }

  async deleteChannel(id: string) {
    return this.request<MessageResponse>(`/api/channels/${id}`, {
      method: "DELETE",
    });
  }

  // Media endpoints
  async getMedia(params?: MediaQueryParams) {
    const queryParams = new URLSearchParams();
    if (params?.limit) queryParams.append("limit", params.limit.toString());
    if (params?.offset) queryParams.append("offset", params.offset.toString());
    if (params?.show) queryParams.append("show", params.show);

    const query = queryParams.toString();
    const endpoint = query ? `/api/media?${query}` : "/api/media";

    return this.request<PaginatedMediaResponse>(endpoint);
  }

  async getMediaItem(id: string) {
    return this.request<Media>(`/api/media/${id}`);
  }

  async updateMedia(id: string, data: UpdateMediaRequest) {
    return this.request<Media>(`/api/media/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    });
  }

  async deleteMedia(id: string) {
    return this.request<MessageResponse>(`/api/media/${id}`, {
      method: "DELETE",
    });
  }

  async scanMedia(path: string) {
    return this.request<ScanResponse>("/api/media/scan", {
      method: "POST",
      body: JSON.stringify({ path }),
    });
  }

  async getScanStatus(scanId: string) {
    return this.request<ScanProgress>(`/api/media/scan/${scanId}/status`);
  }
}

export const apiClient = new ApiClient(API_BASE_URL);

