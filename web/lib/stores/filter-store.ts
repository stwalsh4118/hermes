import { create } from "zustand";
import { devtools, persist, createJSONStorage } from "zustand/middleware";

interface FilterState {
  // Media filters
  mediaSearch: string;
  mediaShowFilter: string | null;
  setMediaSearch: (search: string) => void;
  setMediaShowFilter: (show: string | null) => void;
  clearMediaFilters: () => void;

  // Channel filters
  channelSearch: string;
  setChannelSearch: (search: string) => void;
  clearChannelFilters: () => void;
}

export const useFilterStore = create<FilterState>()(
  devtools(
    persist(
      (set) => ({
        // Media filters
        mediaSearch: "",
        mediaShowFilter: null,
        setMediaSearch: (search) => set({ mediaSearch: search }),
        setMediaShowFilter: (show) => set({ mediaShowFilter: show }),
        clearMediaFilters: () => set({ mediaSearch: "", mediaShowFilter: null }),

        // Channel filters
        channelSearch: "",
        setChannelSearch: (search) => set({ channelSearch: search }),
        clearChannelFilters: () => set({ channelSearch: "" }),
      }),
      {
        name: "filter-storage",
        storage: createJSONStorage(() => localStorage),
        partialize: (state) => ({
          // Don't persist search queries, only filters
          mediaShowFilter: state.mediaShowFilter,
        }),
      }
    ),
    { name: "filter-store" }
  )
);


