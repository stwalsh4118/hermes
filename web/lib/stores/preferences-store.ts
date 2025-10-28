import { create } from "zustand";
import { devtools, persist, createJSONStorage } from "zustand/middleware";

export type ViewMode = "grid" | "list";
export type SortOrder = "name" | "date" | "duration";

interface PreferencesState {
  // View preferences
  mediaViewMode: ViewMode;
  channelViewMode: ViewMode;
  setMediaViewMode: (mode: ViewMode) => void;
  setChannelViewMode: (mode: ViewMode) => void;

  // Sort preferences
  mediaSortOrder: SortOrder;
  channelSortOrder: SortOrder;
  setMediaSortOrder: (order: SortOrder) => void;
  setChannelSortOrder: (order: SortOrder) => void;

  // Reset
  resetPreferences: () => void;
}

const defaultState = {
  mediaViewMode: "grid" as ViewMode,
  channelViewMode: "grid" as ViewMode,
  mediaSortOrder: "name" as SortOrder,
  channelSortOrder: "name" as SortOrder,
};

export const usePreferencesStore = create<PreferencesState>()(
  devtools(
    persist(
      (set) => ({
        ...defaultState,

        setMediaViewMode: (mode) => set({ mediaViewMode: mode }),
        setChannelViewMode: (mode) => set({ channelViewMode: mode }),
        setMediaSortOrder: (order) => set({ mediaSortOrder: order }),
        setChannelSortOrder: (order) => set({ channelSortOrder: order }),
        resetPreferences: () => {
          // Clear persisted storage
          if (typeof window !== "undefined") {
            localStorage.removeItem("preferences-storage");
          }
          // Reset in-memory state
          set(defaultState);
        },
      }),
      {
        name: "preferences-storage",
        storage: createJSONStorage(() => localStorage),
      }
    ),
    { name: "preferences-store" }
  )
);


