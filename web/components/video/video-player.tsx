"use client";

import { useEffect, useRef, useState, forwardRef, useMemo } from "react";
import Hls from "hls.js";
import { LoadingSpinner } from "@/components/common/loading-spinner";
import { AlertCircle, Bug } from "lucide-react";
import { PlayerControls } from "./player-controls";
import { BatchDebugOverlay } from "./batch-debug-overlay";
import { Button } from "@/components/ui/button";

interface VideoPlayerProps {
  channelId: string;
  autoplay?: boolean;
  className?: string;
}

// Position reporting constants
const POSITION_REPORT_INTERVAL = 5000; // 5 seconds
const SEGMENT_DURATION = 2; // seconds, based on HLS config in API specs

/**
 * Maps HLS.js level height to quality string
 */
function getQualityFromHeight(height: number | undefined): string {
  if (!height) return "1080p";
  
  if (height >= 1080) return "1080p";
  if (height >= 720) return "720p";
  if (height >= 480) return "480p";
  
  return "1080p"; // Default fallback
}

/**
 * Gets current quality from HLS.js instance or falls back to default
 */
function getCurrentQuality(hls: Hls | null): string {
  if (!hls) return "1080p";
  
  const currentLevel = hls.currentLevel;
  const levels = hls.levels;
  
  if (currentLevel === -1) {
    // Auto mode - use highest available quality as estimate
    // (we can't reliably determine which level is currently playing in auto mode)
    if (levels && levels.length > 0) {
      // Find the highest quality level
      const highestLevel = levels.reduce((max, level) => {
        const maxHeight = max?.height || 0;
        const levelHeight = level?.height || 0;
        return levelHeight > maxHeight ? level : max;
      });
      
      if (highestLevel?.height) {
        return getQualityFromHeight(highestLevel.height);
      }
    }
    return "1080p"; // Default for auto mode
  }
  
  // Manual quality selection - get the selected level
  if (levels && currentLevel >= 0 && currentLevel < levels.length) {
    const level = levels[currentLevel];
    return getQualityFromHeight(level.height);
  }
  
  return "1080p"; // Default fallback
}

export const VideoPlayer = forwardRef<HTMLVideoElement, VideoPlayerProps>(
  ({ channelId, autoplay = true, className = "" }, ref) => {
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [needsInteraction, setNeedsInteraction] = useState(false);
    const [showDebug, setShowDebug] = useState(false);
    const hlsRef = useRef<Hls | null>(null);
    const videoRef = useRef<HTMLVideoElement | null>(null);
    
    // Generate or retrieve a unique session ID for this player instance
    // Use sessionStorage to persist across page refreshes within the same tab
    // useMemo ensures this only runs once per channelId
    const sessionId = useMemo(() => {
      const storageKey = `hermes-player-session-${channelId}`;
      
      // Try to get existing session from sessionStorage
      if (typeof window !== 'undefined') {
        const existingSession = sessionStorage.getItem(storageKey);
        if (existingSession) {
          console.log(`[VideoPlayer] Reusing existing session: ${existingSession}`);
          return existingSession;
        }
      }
      
      // Generate new session ID
      const newSessionId = `${Date.now()}-${Math.random().toString(36).substring(2, 15)}`;
      console.log(`[VideoPlayer] Generated new session: ${newSessionId}`);
      
      // Store in sessionStorage
      if (typeof window !== 'undefined') {
        sessionStorage.setItem(storageKey, newSessionId);
      }
      
      return newSessionId;
    }, [channelId]);
    
    // Track if we've actually registered (prevents StrictMode cleanup from unregistering prematurely)
    const registeredRef = useRef<boolean>(false);

    // API URL from environment variable
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
    const streamUrl = `${apiUrl}/api/stream/${channelId}/master.m3u8?session_id=${sessionId}`;

    // Handle user interaction to start playback
    const handlePlayClick = () => {
      console.log("[VideoPlayer] Play button clicked");
      if (videoRef.current) {
        videoRef.current.play()
          .then(() => {
            console.log("[VideoPlayer] Playback started successfully");
            setNeedsInteraction(false);
          })
          .catch((err) => {
            console.error("[VideoPlayer] Failed to start playback:", err);
            setError("Failed to start playback. Please try again.");
          });
      } else {
        console.error("[VideoPlayer] Video element not found");
      }
    };

    // Debug logging for needsInteraction state
    useEffect(() => {
      console.log("[VideoPlayer] needsInteraction state changed:", needsInteraction);
    }, [needsInteraction]);

    // Hide play button when video starts playing (regardless of how it started)
    useEffect(() => {
      const video = videoRef.current;
      if (!video) return;

      const handlePlay = () => {
        console.log("[VideoPlayer] Video started playing, hiding play button");
        setNeedsInteraction(false);
      };

      video.addEventListener('play', handlePlay);
      return () => video.removeEventListener('play', handlePlay);
    }, []);

    // Position reporting: Send current playback position to backend every 5 seconds
    useEffect(() => {
      const video = videoRef.current;
      if (!video) return;

      let intervalId: NodeJS.Timeout | null = null;

      const reportPosition = async () => {
        // Only report when video is playing
        if (video.paused || video.ended) {
          return;
        }

        try {
          // Calculate segment number from current time
          const currentTime = video.currentTime;
          const segmentNumber = Math.floor(currentTime / SEGMENT_DURATION);

          // Get current quality
          const quality = getCurrentQuality(hlsRef.current);

          // Prepare request body
          const requestBody = {
            session_id: sessionId,
            segment_number: segmentNumber,
            quality: quality,
            timestamp: new Date().toISOString(),
          };

          // Send position update to backend
          const positionUrl = `${apiUrl}/api/stream/${channelId}/position`;
          const response = await fetch(positionUrl, {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
            },
            body: JSON.stringify(requestBody),
          });

          if (!response.ok) {
            // Log errors but don't interrupt playback
            if (response.status === 404) {
              console.debug(`[VideoPlayer] Stream not found for position reporting (may be expected)`);
            } else {
              console.warn(`[VideoPlayer] Position reporting failed: ${response.status} ${response.statusText}`);
            }
            return;
          }

          // Optionally log successful reports in development
          if (process.env.NODE_ENV === "development") {
            const data = await response.json().catch(() => null);
            if (data) {
              console.debug(`[VideoPlayer] Position reported: segment ${segmentNumber}, quality ${quality}, batch ${data.current_batch}, ${data.segments_remaining} remaining`);
            }
          }
        } catch (error) {
          // Log errors but don't interrupt playback
          console.debug(`[VideoPlayer] Position reporting error (non-fatal):`, error);
        }
      };

      // Set up interval to report position every 5 seconds
      const startReporting = () => {
        // Clear any existing interval
        if (intervalId) {
          clearInterval(intervalId);
        }

        // Report immediately on play
        reportPosition();

        // Then report every 5 seconds
        intervalId = setInterval(() => {
          reportPosition();
        }, POSITION_REPORT_INTERVAL);
      };

      // Stop reporting when paused or ended
      const stopReporting = () => {
        if (intervalId) {
          clearInterval(intervalId);
          intervalId = null;
        }
      };

      // Start reporting when video starts playing
      video.addEventListener('play', startReporting);
      
      // Stop reporting when video is paused or ended
      video.addEventListener('pause', stopReporting);
      video.addEventListener('ended', stopReporting);

      // Initial check: if video is already playing, start reporting
      if (!video.paused && !video.ended) {
        startReporting();
      }

      // Cleanup: stop reporting and remove event listeners
      return () => {
        stopReporting();
        video.removeEventListener('play', startReporting);
        video.removeEventListener('pause', stopReporting);
        video.removeEventListener('ended', stopReporting);
      };
    }, [channelId, sessionId, apiUrl]);

    useEffect(() => {
      const video = videoRef.current;
      if (!video) return;

      // Check browser support
      if (Hls.isSupported()) {
        // Use HLS.js for browsers that support MSE (Chrome, Firefox, Edge)
        const hls = new Hls({
          enableWorker: true,
          lowLatencyMode: false, // VOD-like experience for channel streaming

          startPosition: 0,
        });

        hlsRef.current = hls;

        hls.loadSource(streamUrl);
        hls.attachMedia(video);

        hls.on(Hls.Events.MANIFEST_PARSED, () => {
          console.log("[VideoPlayer] Manifest parsed, attempting autoplay");
          registeredRef.current = true; // Mark as successfully registered
          setIsLoading(false);
          if (autoplay) {
            video.play()
              .then(() => {
                console.log("[VideoPlayer] Autoplay succeeded");
              })
              .catch((err) => {
                console.log("[VideoPlayer] Autoplay blocked by browser, showing play button:", err.message);
                setNeedsInteraction(true);
              });
          }
        });

        hls.on(Hls.Events.ERROR, (event, data) => {
          console.error("HLS error:", data);
          if (data.fatal) {
            switch (data.type) {
              case Hls.ErrorTypes.NETWORK_ERROR:
                setError("Network error. Please check your connection.");
                break;
              case Hls.ErrorTypes.MEDIA_ERROR:
                setError("Media error. The stream may be corrupted.");
                // Try to recover from media errors
                hls.recoverMediaError();
                break;
              default:
                setError("An error occurred while loading the stream.");
                break;
            }
          }
        });
      } else if (video.canPlayType("application/vnd.apple.mpegurl")) {
        // Native HLS support (Safari)
        video.src = streamUrl;

        video.addEventListener("loadedmetadata", () => {
          console.log("[VideoPlayer] Metadata loaded (Safari native HLS)");
          registeredRef.current = true; // Mark as successfully registered
          setIsLoading(false);
          if (autoplay) {
            video.play().catch((err) => {
              console.log("Autoplay blocked by browser, user interaction required:", err);
              setNeedsInteraction(true);
            });
          }
        });

        video.addEventListener("error", () => {
          setError("Failed to load video stream.");
        });
      } else {
        // Browser doesn't support HLS
        setError(
          "Your browser doesn't support HLS streaming. Please use a modern browser."
        );
        setIsLoading(false);
      }

      // Cleanup function
      return () => {
        console.log(`[VideoPlayer] Cleanup triggered for session ${sessionId}, registered: ${registeredRef.current}`);
        
        // Only unregister if we actually successfully registered
        // This prevents React StrictMode's test unmount from unregistering prematurely
        if (registeredRef.current) {
          console.log(`[VideoPlayer] Unregistering client ${sessionId} for channel ${channelId}`);
          
          // Unregister client from backend
          const unregisterUrl = `${apiUrl}/api/stream/${channelId}/client?session_id=${sessionId}`;
          
          // Use fetch with keepalive for better reliability during page unload
          fetch(unregisterUrl, {
            method: "DELETE",
            keepalive: true, // Ensures request completes even if page unloads
          })
            .then(() => {
              console.log(`[VideoPlayer] Successfully unregistered session ${sessionId} from channel ${channelId}`);
              // Clear session from storage on successful unregister
              if (typeof window !== 'undefined') {
                sessionStorage.removeItem(`hermes-player-session-${channelId}`);
              }
            })
            .catch((err) => {
              // 404 is expected if stream already stopped
              console.debug(`[VideoPlayer] Unregister response (may be expected):`, err);
            });
        } else {
          console.log(`[VideoPlayer] Skipping unregister for session ${sessionId} (not yet registered)`);
        }

        // Cleanup HLS instance
        if (hlsRef.current) {
          console.log(`[VideoPlayer] Destroying HLS instance for channel ${channelId}`);
          hlsRef.current.destroy();
          hlsRef.current = null;
        }
      };
    }, [streamUrl, autoplay, apiUrl, channelId, sessionId]);

    return (
      <div className={`relative w-full bg-black ${className}`}>
        {/* 16:9 Aspect Ratio Container */}
        <div className="relative w-full" style={{ paddingBottom: "56.25%" }}>
          <video
            ref={(el) => {
              videoRef.current = el;
              // Forward ref to parent if provided
              if (typeof ref === "function") {
                ref(el);
              } else if (ref) {
                ref.current = el;
              }
            }}
            className="absolute top-0 left-0 w-full h-full"
            playsInline
          />

          {/* Custom Player Controls - hide when interaction needed */}
          {!isLoading && !error && !needsInteraction && (
            <PlayerControls
              videoRef={videoRef}
              hlsRef={hlsRef}
              channelId={channelId}
            />
          )}

          {/* Loading State */}
          {isLoading && (
            <div className="absolute inset-0 flex items-center justify-center bg-black/80">
              <div className="text-center">
                <LoadingSpinner size="lg" />
                <p className="mt-4 text-white text-sm">Loading stream...</p>
              </div>
            </div>
          )}

          {/* Error State */}
          {error && (
            <div className="absolute inset-0 flex items-center justify-center bg-black/90 p-4">
              <div className="text-center max-w-md">
                <AlertCircle className="w-12 h-12 text-red-500 mx-auto mb-4" />
                <p className="text-white text-lg font-semibold mb-2">
                  Playback Error
                </p>
                <p className="text-gray-300 text-sm">{error}</p>
              </div>
            </div>
          )}

          {/* User Interaction Required (Autoplay Blocked) */}
          {needsInteraction && !error && (
            <div 
              className="absolute inset-0 flex items-center justify-center bg-black/70 backdrop-blur-sm z-50 cursor-pointer"
              onClick={handlePlayClick}
            >
              <button
                onClick={handlePlayClick}
                className="flex items-center justify-center w-20 h-20 rounded-full bg-white/90 hover:bg-white transition-all hover:scale-110 active:scale-95 shadow-2xl pointer-events-auto"
                aria-label="Play video"
                type="button"
              >
                <svg
                  className="w-8 h-8 text-black ml-1"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                >
                  <path d="M6.3 2.841A1.5 1.5 0 004 4.11v11.78a1.5 1.5 0 002.3 1.269l9.344-5.89a1.5 1.5 0 000-2.538L6.3 2.84z" />
                </svg>
              </button>
            </div>
          )}

          {/* Debug Toggle Button */}
          <Button
            variant="outline"
            size="icon"
            className="absolute top-4 right-4 z-40 bg-black/50 hover:bg-black/70 border-white/20 text-white"
            onClick={() => setShowDebug(!showDebug)}
            title="Toggle batch debug overlay"
          >
            <Bug className="h-4 w-4" />
          </Button>
        </div>

        {/* Debug Overlay */}
        {showDebug && (
          <BatchDebugOverlay
            channelId={channelId}
            onClose={() => setShowDebug(false)}
          />
        )}
      </div>
    );
  }
);

VideoPlayer.displayName = "VideoPlayer";

