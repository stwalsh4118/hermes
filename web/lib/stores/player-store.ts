import { create } from "zustand";
import { devtools, persist, createJSONStorage } from "zustand/middleware";

interface PlayerState {
  // Current playback
  currentChannelId: string | null;
  isPlaying: boolean;
  volume: number;
  isMuted: boolean;

  // Actions
  setCurrentChannel: (channelId: string | null) => void;
  setPlaying: (playing: boolean) => void;
  setVolume: (volume: number) => void;
  setMuted: (muted: boolean) => void;
  toggleMute: () => void;
  
  // Player controls
  play: (channelId: string) => void;
  pause: () => void;
  stop: () => void;
}

export const usePlayerStore = create<PlayerState>()(
  devtools(
    persist(
      (set) => ({
        // Initial state
        currentChannelId: null,
        isPlaying: false,
        volume: 80,
        isMuted: false,

        // Setters
        setCurrentChannel: (channelId) => set({ currentChannelId: channelId }),
        setPlaying: (playing) => set({ isPlaying: playing }),
        setVolume: (volume) => set({ volume, isMuted: false }),
        setMuted: (muted) => set({ isMuted: muted }),
        toggleMute: () => set((state) => ({ isMuted: !state.isMuted })),

        // Player controls
        play: (channelId) => set({ currentChannelId: channelId, isPlaying: true }),
        pause: () => set({ isPlaying: false }),
        stop: () => set({ currentChannelId: null, isPlaying: false }),
      }),
      {
        name: "player-storage",
        storage: createJSONStorage(() => localStorage),
        partialize: (state) => ({
          // Only persist volume, not playback state
          volume: state.volume,
        }),
      }
    ),
    { name: "player-store" }
  )
);

