"use client";

import { useRef, useEffect, useState } from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import { Media } from "@/lib/types/api";
import { useMediaTree } from "@/hooks/use-media-tree";
import { MediaTreeNodeComponent } from "./media-tree-node";
import { hasChildren } from "@/lib/types/media-tree";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/common/empty-state";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { InboxIcon } from "lucide-react";
import { cn } from "@/lib/utils";
import { formatDuration, formatCount } from "@/lib/utils/format";

interface MediaTreeProps {
  /** Media items to display in tree */
  media: Media[];
  
  /** Search query to filter/highlight */
  searchQuery?: string;
  
  /** Loading state */
  isLoading?: boolean;
  
  /** Optional class name for container */
  className?: string;
  
  /** Height of the tree container (for virtual scrolling) */
  height?: number;
  
  /** Callback when selection changes */
  onSelectionChange?: (selectedMedia: Media[]) => void;
  
  /** Media IDs that should be disabled (cannot be selected) */
  disabledMediaIds?: string[];
  
  /** Enable drag-and-drop reordering */
  enableReordering?: boolean;
  
  /** Callback when items are reordered */
  onReorder?: (orderedMedia: Media[]) => void;
  
  /** Show "Show Only Added" toggle */
  showFilterToggle?: boolean;
  
  /** Initial selected media IDs (for pre-populating selection) */
  initialSelectedMediaIds?: string[];
  
  /** Callback when an episode is clicked (not just checkbox) */
  onEpisodeClick?: (media: Media) => void;
}

/**
 * Media tree component with hierarchical display and selection
 * Supports virtual scrolling for large datasets
 */
export function MediaTree({
  media,
  searchQuery = "",
  isLoading = false,
  className,
  height = 600,
  onSelectionChange,
  disabledMediaIds = [],
  enableReordering = false,
  onReorder,
  showFilterToggle = false,
  initialSelectedMediaIds = [],
  onEpisodeClick,
}: MediaTreeProps) {
  const parentRef = useRef<HTMLDivElement>(null);
  
  // Track active node for keyboard navigation (aria-activedescendant pattern)
  const [activeNodeId, setActiveNodeId] = useState<string | null>(null);
  
  // Track "Show Only Added" filter state
  const [showOnlySelected, setShowOnlySelected] = useState(false);

  // Use the media tree hook
  const {
    flattenedNodes,
    toggleNode,
    selectNode,
    getSelectedMedia,
    expandAll,
    collapseAll,
  } = useMediaTree({
    media,
    searchQuery,
    disabledMediaIds,
    showOnlySelected,
    initialSelectedMediaIds,
  });

  // Setup virtual scrolling
  const rowVirtualizer = useVirtualizer({
    count: flattenedNodes.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 48, // Approximate row height
    overscan: 10, // Render extra items for smooth scrolling
  });
  
  // Initialize active node to first item when nodes change
  useEffect(() => {
    if (flattenedNodes.length > 0 && !activeNodeId) {
      setActiveNodeId(`node-${flattenedNodes[0].node.id}`);
    }
  }, [flattenedNodes, activeNodeId]);

  // Notify parent when selection changes
  const [previousSelectionIds, setPreviousSelectionIds] = useState<string>("");
  
  useEffect(() => {
    if (onSelectionChange) {
      const selectedMedia = getSelectedMedia();
      // Create a stable ID string from selected media IDs to detect actual changes
      const currentSelectionIds = selectedMedia.map(m => m.id).sort().join(",");
      
      // Only call parent callback if selection actually changed
      if (currentSelectionIds !== previousSelectionIds) {
        setPreviousSelectionIds(currentSelectionIds);
        onSelectionChange(selectedMedia);
      }
    }
  }, [getSelectedMedia, onSelectionChange, previousSelectionIds]);
  
  // Calculate selected media stats for display
  const selectedMedia = getSelectedMedia();
  const totalDuration = selectedMedia.reduce((sum, m) => sum + (m.duration || 0), 0);
  const selectedCount = selectedMedia.length;

  // Keyboard navigation using aria-activedescendant pattern
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!parentRef.current?.contains(document.activeElement)) {
        return;
      }

      if (flattenedNodes.length === 0) return;

      // Find current active node index
      const currentIndex = flattenedNodes.findIndex((item) => 
        `node-${item.node.id}` === activeNodeId
      );
      
      let newIndex = currentIndex >= 0 ? currentIndex : 0;

      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          newIndex = Math.min(flattenedNodes.length - 1, newIndex + 1);
          break;
        case "ArrowUp":
          e.preventDefault();
          newIndex = Math.max(0, newIndex - 1);
          break;
        case "ArrowLeft":
          e.preventDefault();
          if (currentIndex >= 0) {
            const node = flattenedNodes[currentIndex].node;
            if (node.expanded && hasChildren(node)) {
              // Collapse if expanded
              toggleNode(node.id);
            }
          }
          break;
        case "ArrowRight":
          e.preventDefault();
          if (currentIndex >= 0) {
            const node = flattenedNodes[currentIndex].node;
            if (!node.expanded && hasChildren(node)) {
              // Expand if collapsed
              toggleNode(node.id);
            }
          }
          break;
        case " ":
          e.preventDefault();
          if (currentIndex >= 0) {
            const node = flattenedNodes[currentIndex].node;
            selectNode(node.id, !node.selected);
          }
          break;
        case "Enter":
          e.preventDefault();
          if (currentIndex >= 0) {
            const node = flattenedNodes[currentIndex].node;
            toggleNode(node.id);
          }
          break;
        default:
          return;
      }

      // Update active node and scroll to it
      if (e.key === "ArrowDown" || e.key === "ArrowUp" || e.key === "ArrowLeft" || e.key === "ArrowRight") {
        const newNodeId = `node-${flattenedNodes[newIndex].node.id}`;
        setActiveNodeId(newNodeId);
        rowVirtualizer.scrollToIndex(newIndex, { align: "auto" });
      }
    };

    const element = parentRef.current;
    element?.addEventListener("keydown", handleKeyDown);
    return () => element?.removeEventListener("keydown", handleKeyDown);
  }, [flattenedNodes, rowVirtualizer, toggleNode, selectNode, activeNodeId]);

  // Loading state
  if (isLoading) {
    return (
      <div
        className={cn(
          "bg-card rounded-xl border-4 border-primary/30",
          "shadow-[8px_8px_0_rgba(0,0,0,0.2)]",
          "overflow-hidden",
          className
        )}
        style={{ height }}
      >
        <div className="p-4 space-y-2">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="flex items-center gap-3">
              <Skeleton className="w-5 h-5" />
              <Skeleton className="w-5 h-5" />
              <Skeleton className="flex-1 h-6" />
            </div>
          ))}
        </div>
      </div>
    );
  }

  // Empty state - no media at all
  if (media.length === 0) {
    return (
      <div
        className={cn(
          "bg-card rounded-xl border-4 border-primary/30",
          "shadow-[8px_8px_0_rgba(0,0,0,0.2)]",
          className
        )}
        style={{ height }}
      >
        <EmptyState
          icon={InboxIcon}
          title="No media found"
          description="Your media library is empty. Scan a directory to add media files."
        />
      </div>
    );
  }

  // Empty state - search returned no results
  if (flattenedNodes.length === 0 && searchQuery) {
    return (
      <div
        className={cn(
          "bg-card rounded-xl border-4 border-primary/30",
          "shadow-[8px_8px_0_rgba(0,0,0,0.2)]",
          className
        )}
        style={{ height }}
      >
        <EmptyState
          icon={InboxIcon}
          title="No matches found"
          description={`No media matched your search for "${searchQuery}"`}
        />
      </div>
    );
  }

  const virtualItems = rowVirtualizer.getVirtualItems();
  const totalHeight = rowVirtualizer.getTotalSize();

  return (
    <div
      className={cn(
        "bg-card rounded-xl border-4 border-primary/30",
        "shadow-[8px_8px_0_rgba(0,0,0,0.2)]",
        "overflow-hidden",
        className
      )}
    >
      {/* Toolbar */}
      <div className="border-b-4 border-primary/30 px-4 py-3 bg-muted/20">
        <div className="flex items-center justify-between gap-4 flex-wrap">
          {/* Left: Stats */}
          <div className="font-mono text-sm text-muted-foreground">
            <span>{flattenedNodes.length} items</span>
            {selectedCount > 0 && (
              <>
                <span className="mx-2">•</span>
                <span className="text-primary font-bold">
                  {formatCount(selectedCount, "selected")} • {formatDuration(totalDuration)}
                </span>
              </>
            )}
          </div>
          
          {/* Right: Controls */}
          <div className="flex items-center gap-4">
            {/* Show Only Added Toggle */}
            {showFilterToggle && selectedCount > 0 && (
              <div className="flex items-center gap-2">
                <Switch
                  id="show-only-added"
                  checked={showOnlySelected}
                  onCheckedChange={setShowOnlySelected}
                  className="data-[state=checked]:bg-primary"
                />
                <Label
                  htmlFor="show-only-added"
                  className="text-xs font-mono font-bold text-foreground cursor-pointer whitespace-nowrap"
                >
                  SHOW ONLY ADDED
                </Label>
              </div>
            )}
            
            {/* Expand/Collapse */}
            {!showOnlySelected && (
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={expandAll}
                  className="text-xs font-mono font-bold text-primary hover:text-primary/80 transition-colors whitespace-nowrap"
                >
                  EXPAND ALL
                </button>
                <span className="text-muted-foreground">|</span>
                <button
                  type="button"
                  onClick={collapseAll}
                  className="text-xs font-mono font-bold text-primary hover:text-primary/80 transition-colors whitespace-nowrap"
                >
                  COLLAPSE ALL
                </button>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Virtual scrolling container */}
      <div
        ref={parentRef}
        style={{ height: height - 60, overflow: "auto" }}
        role="tree"
        aria-label="Media library tree"
        aria-activedescendant={activeNodeId || undefined}
        tabIndex={0}
        className="focus:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-inset"
      >
        <div
          style={{
            height: `${totalHeight}px`,
            width: "100%",
            position: "relative",
          }}
        >
          {virtualItems.map((virtualRow) => {
            const { node, depth } = flattenedNodes[virtualRow.index];
            const nodeId = `node-${node.id}`;
            const isActive = nodeId === activeNodeId;
            
            return (
              <div
                key={virtualRow.key}
                id={nodeId}
                data-node-id={node.id}
                style={{
                  position: "absolute",
                  top: 0,
                  left: 0,
                  width: "100%",
                  transform: `translateY(${virtualRow.start}px)`,
                }}
              >
                <MediaTreeNodeComponent
                  node={node}
                  depth={depth}
                  onToggle={toggleNode}
                  onSelect={selectNode}
                  searchQuery={searchQuery}
                  isActive={isActive}
                  enableReordering={enableReordering}
                  onEpisodeClick={onEpisodeClick}
                />
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

