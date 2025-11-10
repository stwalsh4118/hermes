"use client";

import { useEffect, useRef, useState } from "react";
import Hls from "hls.js";
import {
  Volume2,
  VolumeX,
  Maximize,
  Minimize,
  Settings,
  Check,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Slider } from "@/components/ui/slider";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { usePlayerStore } from "@/lib/stores";
import { cn } from "@/lib/utils";

interface PlayerControlsProps {
  videoRef: React.RefObject<HTMLVideoElement>;
  hlsRef: React.RefObject<Hls | null>;
  channelId: string;
  className?: string;
}

const QUALITY_LABELS: Record<number, string> = {
  1080: "1080p",
  720: "720p",
  480: "480p",
  360: "360p",
};

export function PlayerControls({
  videoRef,
  hlsRef,
  channelId,
  className = "",
}: PlayerControlsProps) {
  const [showControls, setShowControls] = useState(true);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [currentQuality, setCurrentQuality] = useState<number>(-1); // -1 = Auto
  const [availableQualities, setAvailableQualities] = useState<
    Array<{ index: number; height: number; label: string }>
  >([]);
  const hideTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  // Get volume state from Zustand store
  const { volume, isMuted, setVolume, setMuted, toggleMute } = usePlayerStore();

  // Handle mouse/touch movement to show controls
  const handleInteraction = () => {
    setShowControls(true);
    clearTimeout(hideTimeoutRef.current);

    // Hide controls after 3 seconds of inactivity
    hideTimeoutRef.current = setTimeout(() => {
      setShowControls(false);
    }, 3000);
  };

  // Setup available quality levels from HLS
  useEffect(() => {
    const hls = hlsRef.current;
    if (!hls) return;

    const updateQualityLevels = () => {
      const levels = hls.levels;
      if (levels && levels.length > 0) {
        const qualities = levels
          .map((level, index) => ({
            index,
            height: level.height,
            label: QUALITY_LABELS[level.height] || `${level.height}p`,
          }))
          .sort((a, b) => b.height - a.height); // Sort high to low

        setAvailableQualities(qualities);
      }
    };

    hls.on(Hls.Events.MANIFEST_PARSED, updateQualityLevels);

    return () => {
      hls.off(Hls.Events.MANIFEST_PARSED, updateQualityLevels);
    };
  }, [hlsRef]);

  // Sync volume to video element
  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    video.volume = volume / 100;
    video.muted = isMuted;
  }, [volume, isMuted, videoRef]);

  // Handle fullscreen changes
  useEffect(() => {
    const handleFullscreenChange = () => {
      setIsFullscreen(!!document.fullscreenElement);
    };

    document.addEventListener("fullscreenchange", handleFullscreenChange);
    return () => {
      document.removeEventListener("fullscreenchange", handleFullscreenChange);
    };
  }, []);

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const video = videoRef.current;
      if (!video) return;

      switch (e.key.toLowerCase()) {
        case " ": // Space - play/pause
          e.preventDefault();
          if (video.paused) {
            video.play();
          } else {
            video.pause();
          }
          break;
        case "f": // F - fullscreen
          e.preventDefault();
          handleFullscreenToggle();
          break;
        case "m": // M - mute
          e.preventDefault();
          toggleMute();
          break;
        case "arrowup": // Arrow Up - increase volume
          e.preventDefault();
          setVolume(Math.min(100, volume + 5));
          break;
        case "arrowdown": // Arrow Down - decrease volume
          e.preventDefault();
          setVolume(Math.max(0, volume - 5));
          break;
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [volume, videoRef, toggleMute, setVolume]);

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      clearTimeout(hideTimeoutRef.current);
    };
  }, []);

  const handleVolumeChange = (value: number[]) => {
    setVolume(value[0]);
  };

  const handleMuteToggle = () => {
    toggleMute();
  };

  const handleQualityChange = (index: number) => {
    const hls = hlsRef.current;
    if (!hls) return;

    hls.currentLevel = index;
    setCurrentQuality(index);
  };

  const handleFullscreenToggle = () => {
    const container = containerRef.current?.parentElement;
    if (!container) return;

    if (!document.fullscreenElement) {
      container.requestFullscreen().catch((err) => {
        console.error("Error attempting to enable fullscreen:", err);
      });
    } else {
      document.exitFullscreen();
    }
  };

  const getCurrentQualityLabel = () => {
    if (currentQuality === -1) return "Auto";
    const quality = availableQualities.find((q) => q.index === currentQuality);
    return quality?.label || "Auto";
  };

  const getVolumeIcon = () => {
    if (isMuted || volume === 0) {
      return <VolumeX className="w-5 h-5" />;
    }
    return <Volume2 className="w-5 h-5" />;
  };

  return (
    <div
      ref={containerRef}
      className={cn("absolute inset-0 z-10", className)}
      onMouseMove={handleInteraction}
      onTouchStart={handleInteraction}
      onClick={handleInteraction}
    >
      {/* Live Indicator (Always Visible) */}
      <div className="absolute top-4 right-4 z-20">
        <span className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-destructive/20 text-destructive border-2 border-destructive font-bold text-sm shadow-[4px_4px_0_rgba(0,0,0,0.4)]">
          <span className="w-2 h-2 bg-destructive rounded-full animate-pulse" />
          LIVE
        </span>
      </div>

      {/* Controls Overlay (Fades Out) */}
      <div
        className={cn(
          "absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black via-black/80 to-transparent p-6 transition-opacity duration-300",
          showControls ? "opacity-100" : "opacity-0 pointer-events-none"
        )}
      >
        <div className="flex items-center gap-4">
          {/* Volume Control */}
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="icon"
              onClick={handleMuteToggle}
              className="text-white hover:bg-white/20 border-2 border-white/20 shadow-[4px_4px_0_rgba(0,0,0,0.2)] hover:shadow-[2px_2px_0_rgba(0,0,0,0.2)] transition-all"
              aria-label={isMuted ? "Unmute" : "Mute"}
            >
              {getVolumeIcon()}
            </Button>
            <Slider
              value={[isMuted ? 0 : volume]}
              onValueChange={handleVolumeChange}
              max={100}
              step={1}
              className="w-24"
              aria-label="Volume"
            />
            <span className="text-white text-sm font-mono min-w-[3ch]">
              {isMuted ? 0 : volume}
            </span>
          </div>

          <div className="flex-1" />

          {/* Quality Selector */}
          {availableQualities.length > 0 && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-white hover:bg-white/20 border-2 border-white/20 shadow-[4px_4px_0_rgba(0,0,0,0.2)] hover:shadow-[2px_2px_0_rgba(0,0,0,0.2)] transition-all font-bold"
                  aria-label="Quality settings"
                >
                  <Settings className="w-4 h-4 mr-2" />
                  {getCurrentQualityLabel()}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent
                align="end"
                className="bg-card border-2 border-primary shadow-[6px_6px_0_rgba(0,0,0,0.4)]"
              >
                <DropdownMenuItem
                  onClick={() => handleQualityChange(-1)}
                  className="font-mono cursor-pointer"
                >
                  <Check
                    className={cn(
                      "w-4 h-4 mr-2",
                      currentQuality === -1 ? "visible" : "invisible"
                    )}
                  />
                  Auto
                </DropdownMenuItem>
                {availableQualities.map((quality) => (
                  <DropdownMenuItem
                    key={quality.index}
                    onClick={() => handleQualityChange(quality.index)}
                    className="font-mono cursor-pointer"
                  >
                    <Check
                      className={cn(
                        "w-4 h-4 mr-2",
                        currentQuality === quality.index
                          ? "visible"
                          : "invisible"
                      )}
                    />
                    {quality.label}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          )}

          {/* Fullscreen Toggle */}
          <Button
            variant="ghost"
            size="icon"
            onClick={handleFullscreenToggle}
            className="text-white hover:bg-white/20 border-2 border-white/20 shadow-[4px_4px_0_rgba(0,0,0,0.2)] hover:shadow-[2px_2px_0_rgba(0,0,0,0.2)] transition-all"
            aria-label={isFullscreen ? "Exit fullscreen" : "Enter fullscreen"}
          >
            {isFullscreen ? (
              <Minimize className="w-5 h-5" />
            ) : (
              <Maximize className="w-5 h-5" />
            )}
          </Button>
        </div>
      </div>
    </div>
  );
}

