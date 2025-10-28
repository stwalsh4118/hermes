"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import {
  useUIStore,
  useFilterStore,
  usePreferencesStore,
  usePlayerStore,
} from "@/lib/stores";
import { useHydration } from "@/hooks/use-hydration";

export default function StoresTestPage() {
  const hydrated = useHydration();
  // UI Store
  const { sidebarOpen, toggleSidebar, activeModal, openModal, closeModal } = useUIStore();

  // Filter Store
  const { mediaSearch, setMediaSearch, clearMediaFilters } = useFilterStore();

  // Preferences Store
  const { mediaViewMode, setMediaViewMode, resetPreferences } = usePreferencesStore();

  // Player Store
  const { currentChannelId, isPlaying, volume, setVolume, play, pause } = usePlayerStore();

  // Don't render persisted values until hydration is complete
  if (!hydrated) {
    return (
      <div className="container mx-auto py-10">
        <div className="text-center">
          <p className="text-muted-foreground">Loading...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto py-10 space-y-8">
      <div>
        <h1 className="text-4xl font-bold mb-2">Zustand Stores Test</h1>
        <p className="text-muted-foreground">Testing client state management</p>
      </div>

      <Separator />

      <Card>
        <CardHeader>
          <CardTitle>UI Store</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <p>Sidebar Open: {sidebarOpen ? "Yes" : "No"}</p>
            <Button onClick={toggleSidebar}>Toggle Sidebar</Button>
          </div>
          <div>
            <p>Active Modal: {activeModal || "None"}</p>
            <div className="flex gap-2">
              <Button onClick={() => openModal("test-modal")}>Open Modal</Button>
              <Button variant="secondary" onClick={closeModal}>
                Close Modal
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Filter Store</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <Label>Media Search</Label>
            <Input
              value={mediaSearch}
              onChange={(e) => setMediaSearch(e.target.value)}
              placeholder="Search media..."
            />
            <p className="text-sm text-muted-foreground mt-2">
              Current search: {mediaSearch || "(empty)"}
            </p>
          </div>
          <Button variant="secondary" onClick={clearMediaFilters}>
            Clear Filters
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Preferences Store (Persisted)</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <p>Media View Mode: {mediaViewMode}</p>
            <div className="flex gap-2">
              <Button
                variant={mediaViewMode === "grid" ? "default" : "secondary"}
                onClick={() => setMediaViewMode("grid")}
              >
                Grid
              </Button>
              <Button
                variant={mediaViewMode === "list" ? "default" : "secondary"}
                onClick={() => setMediaViewMode("list")}
              >
                List
              </Button>
            </div>
          </div>
          <Button variant="destructive" onClick={resetPreferences}>
            Reset Preferences
          </Button>
          <p className="text-sm text-muted-foreground">
            Refresh the page - your preferences will persist!
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Player Store</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <p>Current Channel: {currentChannelId || "None"}</p>
            <p>Playing: {isPlaying ? "Yes" : "No"}</p>
            <p>Volume: {volume}%</p>
          </div>
          <div className="flex gap-2">
            <Button onClick={() => play("channel-123")}>Play Channel</Button>
            <Button variant="secondary" onClick={pause}>
              Pause
            </Button>
          </div>
          <div>
            <Label>Volume</Label>
            <Input
              type="range"
              min="0"
              max="100"
              value={volume}
              onChange={(e) => setVolume(Number(e.target.value))}
            />
            <p className="text-sm text-muted-foreground mt-2">
              Volume persists across page refreshes
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}


