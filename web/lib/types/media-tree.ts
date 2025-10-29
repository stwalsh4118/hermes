import { Media } from "./api";

/**
 * Type of node in the media tree
 */
export type MediaTreeNodeType = "show" | "season" | "episode";

/**
 * A node in the media tree hierarchy
 */
export interface MediaTreeNode {
  /** Unique identifier for this node (e.g., "show:Friends", "season:Friends:1", "episode:uuid") */
  id: string;
  
  /** Type of node */
  type: MediaTreeNodeType;
  
  /** Display label for this node */
  label: string;
  
  /** Child nodes (undefined for episode nodes) */
  children?: MediaTreeNode[];
  
  /** Reference to the media item (only for episode nodes) */
  media?: Media;
  
  /** Total episode count (for show and season nodes) */
  episodeCount?: number;
  
  /** Whether this node is currently expanded */
  expanded: boolean;
  
  /** Whether this node is selected */
  selected: boolean;
  
  /** Whether this node is in an indeterminate state (some but not all children selected) */
  indeterminate: boolean;
  
  /** Whether this node is disabled (cannot be selected) */
  disabled: boolean;
  
  /** Depth level in the tree (0 = root) */
  depth: number;
  
  /** Parent node ID (undefined for root nodes) */
  parentId?: string;
  
  /** Position in the playlist (for selected episode nodes, 0-indexed) */
  playlistPosition?: number;
}

/**
 * Flattened node for virtual scrolling
 */
export interface FlattenedNode {
  node: MediaTreeNode;
  depth: number;
}

/**
 * Return type for the useMediaTree hook
 */
export interface UseMediaTreeResult {
  /** The hierarchical tree structure */
  tree: MediaTreeNode[];
  
  /** Flattened array of visible nodes (for virtual scrolling) */
  flattenedNodes: FlattenedNode[];
  
  /** Toggle expand/collapse state of a node */
  toggleNode: (nodeId: string) => void;
  
  /** Select or deselect a node (cascades to children) */
  selectNode: (nodeId: string, selected: boolean) => void;
  
  /** Get all selected media items */
  getSelectedMedia: () => Media[];
  
  /** Get array of selected media IDs */
  getSelectedIds: () => string[];
  
  /** Expand all nodes */
  expandAll: () => void;
  
  /** Collapse all nodes */
  collapseAll: () => void;
  
  /** Clear all selections */
  clearSelection: () => void;
  
  /** Select all items */
  selectAll: () => void;
}

/**
 * Type guard: Check if node is a show node
 */
export function isShowNode(node: MediaTreeNode): boolean {
  return node.type === "show";
}

/**
 * Type guard: Check if node is a season node
 */
export function isSeasonNode(node: MediaTreeNode): boolean {
  return node.type === "season";
}

/**
 * Type guard: Check if node is an episode node
 */
export function isEpisodeNode(node: MediaTreeNode): boolean {
  return node.type === "episode";
}

/**
 * Check if node has children
 */
export function hasChildren(node: MediaTreeNode): boolean {
  return node.children !== undefined && node.children.length > 0;
}

/**
 * Format duration in seconds to HH:MM:SS or MM:SS
 */
export function formatDuration(seconds: number): string {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = seconds % 60;
  
  if (hours > 0) {
    return `${hours}:${minutes.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`;
  }
  
  return `${minutes}:${secs.toString().padStart(2, "0")}`;
}

/**
 * Format episode label (e.g., "S01E05 - Episode Title")
 */
export function formatEpisodeLabel(media: Media): string {
  const parts: string[] = [];
  
  if (media.season !== null && media.episode !== null) {
    const seasonStr = media.season.toString().padStart(2, "0");
    const episodeStr = media.episode.toString().padStart(2, "0");
    parts.push(`S${seasonStr}E${episodeStr}`);
  }
  
  parts.push(media.title);
  
  return parts.join(" - ");
}

/**
 * Get metadata string for episode node (resolution, codec, duration)
 */
export function getEpisodeMetadata(media: Media): string {
  const parts: string[] = [];
  
  if (media.resolution) {
    parts.push(media.resolution);
  }
  
  if (media.video_codec) {
    parts.push(media.video_codec.toUpperCase());
  }
  
  parts.push(formatDuration(media.duration));
  
  return parts.join(" • ");
}

