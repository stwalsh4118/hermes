import { useMemo, useState, useCallback, useEffect } from "react";
import { Media } from "@/lib/types/api";
import {
  MediaTreeNode,
  FlattenedNode,
  UseMediaTreeResult,
  formatEpisodeLabel,
} from "@/lib/types/media-tree";

const UNCATEGORIZED_SHOW = "Uncategorized";

interface UseMediaTreeOptions {
  /** Initial media items */
  media: Media[];
  
  /** Search query to filter nodes */
  searchQuery?: string;
  
  /** Media IDs that should be disabled (cannot be selected) */
  disabledMediaIds?: string[];
  
  /** Show only selected items (filter to selected media) */
  showOnlySelected?: boolean;
  
  /** Initial selected media IDs (for pre-populating selection) */
  initialSelectedMediaIds?: string[];
}

/**
 * Hook for organizing media into a hierarchical tree structure
 * and managing selection/expansion state
 */
export function useMediaTree({
  media,
  searchQuery = "",
  disabledMediaIds = [],
  showOnlySelected = false,
  initialSelectedMediaIds = [],
}: UseMediaTreeOptions): UseMediaTreeResult {
  // Track expanded node IDs
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());
  
  // Track selected node IDs - initialize with episode IDs from initialSelectedMediaIds
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => {
    const initial = new Set<string>();
    initialSelectedMediaIds.forEach(mediaId => {
      initial.add(`episode:${mediaId}`);
    });
    return initial;
  });

  /**
   * Build the tree structure from flat media array
   */
  const tree = useMemo(() => {
    // Group media by show, then season
    const showMap = new Map<string, Map<number | null, Media[]>>();
    
    media.forEach((item) => {
      const showName = item.show_name || UNCATEGORIZED_SHOW;
      
      if (!showMap.has(showName)) {
        showMap.set(showName, new Map());
      }
      
      const seasonMap = showMap.get(showName)!;
      const season = item.season;
      
      if (!seasonMap.has(season)) {
        seasonMap.set(season, []);
      }
      
      seasonMap.get(season)!.push(item);
    });
    
    // Build tree nodes
    const treeNodes: MediaTreeNode[] = [];
    
    // Sort shows alphabetically
    const sortedShows = Array.from(showMap.keys()).sort((a, b) => {
      // Uncategorized always goes last
      if (a === UNCATEGORIZED_SHOW) return 1;
      if (b === UNCATEGORIZED_SHOW) return -1;
      return a.localeCompare(b);
    });
    
    sortedShows.forEach((showName) => {
      const seasonMap = showMap.get(showName)!;
      
      // Count total episodes in show
      let totalEpisodes = 0;
      seasonMap.forEach((episodes) => {
        totalEpisodes += episodes.length;
      });
      
      const showId = `show:${showName}`;
      const seasonNodes: MediaTreeNode[] = [];
      
      // Sort seasons numerically (null seasons go last)
      const sortedSeasons = Array.from(seasonMap.keys()).sort((a, b) => {
        if (a === null) return 1;
        if (b === null) return -1;
        return a - b;
      });
      
      sortedSeasons.forEach((seasonNum) => {
        const episodes = seasonMap.get(seasonNum)!;
        
        // Sort episodes by episode number, then by title
        const sortedEpisodes = [...episodes].sort((a, b) => {
          if (a.episode !== null && b.episode !== null) {
            return a.episode - b.episode;
          }
          if (a.episode !== null) return -1;
          if (b.episode !== null) return 1;
          return a.title.localeCompare(b.title);
        });
        
        const seasonId = seasonNum !== null 
          ? `season:${showName}:${seasonNum}`
          : `season:${showName}:unspecified`;
        
        const seasonLabel = seasonNum !== null 
          ? `Season ${seasonNum}`
          : "No Season";
        
        // Create episode nodes
        const episodeNodes: MediaTreeNode[] = sortedEpisodes.map((episode) => {
          const episodeId = `episode:${episode.id}`;
          const isDisabled = disabledMediaIds.includes(episode.id);
          return {
            id: episodeId,
            type: "episode",
            label: formatEpisodeLabel(episode),
            media: episode,
            expanded: false,
            selected: selectedIds.has(episodeId) && !isDisabled,
            indeterminate: false,
            disabled: isDisabled,
            depth: 2,
            parentId: seasonId,
          };
        });
        
        // Check if all episodes are selected (excluding disabled)
        const enabledEpisodes = episodeNodes.filter((n) => !n.disabled);
        const allSelected = enabledEpisodes.length > 0 && enabledEpisodes.every((n) => n.selected);
        const someSelected = enabledEpisodes.some((n) => n.selected);
        const allDisabled = episodeNodes.every((n) => n.disabled);
        
        // Create season node
        const seasonNode: MediaTreeNode = {
          id: seasonId,
          type: "season",
          label: seasonLabel,
          children: episodeNodes,
          episodeCount: episodes.length,
          expanded: expandedIds.has(seasonId),
          selected: allSelected,
          indeterminate: !allSelected && someSelected,
          disabled: allDisabled,
          depth: 1,
          parentId: showId,
        };
        
        seasonNodes.push(seasonNode);
      });
      
      // Check if all seasons are selected (excluding disabled)
      const enabledSeasons = seasonNodes.filter((n) => !n.disabled);
      const allSeasonsSelected = enabledSeasons.length > 0 && enabledSeasons.every((n) => n.selected);
      const someSeasonsSelected = enabledSeasons.some((n) => n.selected || n.indeterminate);
      const allSeasonsDisabled = seasonNodes.every((n) => n.disabled);
      
      // Create show node
      const showNode: MediaTreeNode = {
        id: showId,
        type: "show",
        label: showName,
        children: seasonNodes,
        episodeCount: totalEpisodes,
        expanded: expandedIds.has(showId),
        selected: allSeasonsSelected,
        indeterminate: !allSeasonsSelected && someSeasonsSelected,
        disabled: allSeasonsDisabled,
        depth: 0,
      };
      
      treeNodes.push(showNode);
    });
    
    return treeNodes;
  }, [media, expandedIds, selectedIds, disabledMediaIds]);

  /**
   * Filter and flatten tree based on search query and showOnlySelected
   */
  const filteredAndFlattenedNodes = useMemo(() => {
    const flatten = (nodes: MediaTreeNode[]): FlattenedNode[] => {
      const result: FlattenedNode[] = [];
      
      const traverse = (node: MediaTreeNode) => {
        // Check if node has selected descendants (for showOnlySelected filter)
        const hasSelectedDescendant = (n: MediaTreeNode): boolean => {
          if (n.type === "episode") {
            return n.selected;
          }
          return n.children?.some((child) => 
            child.selected || hasSelectedDescendant(child)
          ) || false;
        };
        
        // Apply showOnlySelected filter
        if (showOnlySelected) {
          const shouldShowForSelection = node.selected || hasSelectedDescendant(node);
          if (!shouldShowForSelection) {
            return;
          }
        }
        
        // Check if node matches search
        const matchesSearch = searchQuery === "" || 
          node.label.toLowerCase().includes(searchQuery.toLowerCase()) ||
          (node.media?.title.toLowerCase().includes(searchQuery.toLowerCase()));
        
        // Check if any descendant matches search
        const hasMatchingDescendant = (n: MediaTreeNode): boolean => {
          if (n.type === "episode") {
            return n.label.toLowerCase().includes(searchQuery.toLowerCase());
          }
          return n.children?.some((child) => 
            hasMatchingDescendant(child) ||
            child.label.toLowerCase().includes(searchQuery.toLowerCase())
          ) || false;
        };
        
        const shouldShow = searchQuery === "" || 
          matchesSearch || 
          hasMatchingDescendant(node);
        
        if (!shouldShow) {
          return;
        }
        
        // Add this node
        result.push({ node, depth: node.depth });
        
        // If expanded (or forced by search/filter), add children
        const shouldExpand = node.expanded || 
          (searchQuery !== "" && hasMatchingDescendant(node)) ||
          (showOnlySelected && hasSelectedDescendant(node));
        
        if (shouldExpand && node.children) {
          node.children.forEach(traverse);
        }
      };
      
      nodes.forEach(traverse);
      return result;
    };
    
    return flatten(tree);
  }, [tree, searchQuery, showOnlySelected]);

  /**
   * Toggle expand/collapse state of a node
   */
  const toggleNode = useCallback((nodeId: string) => {
    setExpandedIds((prev) => {
      const next = new Set(prev);
      if (next.has(nodeId)) {
        next.delete(nodeId);
      } else {
        next.add(nodeId);
      }
      return next;
    });
  }, []);

  /**
   * Select or deselect a node and cascade to children
   */
  const selectNode = useCallback((nodeId: string, selected: boolean) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      
      // Find the node in tree
      const findNode = (nodes: MediaTreeNode[]): MediaTreeNode | null => {
        for (const node of nodes) {
          if (node.id === nodeId) return node;
          if (node.children) {
            const found = findNode(node.children);
            if (found) return found;
          }
        }
        return null;
      };
      
      const node = findNode(tree);
      if (!node || node.disabled) return prev;
      
      // Update this node
      if (selected) {
        next.add(nodeId);
      } else {
        next.delete(nodeId);
      }
      
      // Cascade to all descendants (skip disabled nodes)
      const cascadeToChildren = (n: MediaTreeNode) => {
        if (n.children) {
          n.children.forEach((child) => {
            if (child.disabled) return; // Skip disabled nodes
            
            if (selected) {
              next.add(child.id);
            } else {
              next.delete(child.id);
            }
            cascadeToChildren(child);
          });
        }
      };
      
      cascadeToChildren(node);
      
      return next;
    });
  }, [tree]);

  /**
   * Get all selected media items
   */
  const getSelectedMedia = useCallback((): Media[] => {
    const selectedMedia: Media[] = [];
    
    const collectSelected = (nodes: MediaTreeNode[]) => {
      nodes.forEach((node) => {
        if (node.type === "episode" && node.media && selectedIds.has(node.id)) {
          selectedMedia.push(node.media);
        }
        if (node.children) {
          collectSelected(node.children);
        }
      });
    };
    
    collectSelected(tree);
    return selectedMedia;
  }, [tree, selectedIds]);

  /**
   * Get array of selected media IDs
   */
  const getSelectedIds = useCallback((): string[] => {
    const ids: string[] = [];
    
    const collectIds = (nodes: MediaTreeNode[]) => {
      nodes.forEach((node) => {
        if (node.type === "episode" && node.media && selectedIds.has(node.id)) {
          ids.push(node.media.id);
        }
        if (node.children) {
          collectIds(node.children);
        }
      });
    };
    
    collectIds(tree);
    return ids;
  }, [tree, selectedIds]);

  /**
   * Calculate playlist positions for selected nodes (depth-first traversal)
   * Mutates the tree to add playlistPosition to selected episode nodes
   */
  const calculatePlaylistPositions = useCallback(() => {
    let position = 0;
    
    const assignPositions = (nodes: MediaTreeNode[]) => {
      nodes.forEach((node) => {
        if (node.type === "episode" && node.selected && node.media) {
          node.playlistPosition = position;
          position++;
        }
        if (node.children) {
          assignPositions(node.children);
        }
      });
    };
    
    assignPositions(tree);
  }, [tree]);
  
  // Calculate positions whenever selection changes
  useEffect(() => {
    calculatePlaylistPositions();
  }, [selectedIds, calculatePlaylistPositions]);

  /**
   * Expand all nodes
   */
  const expandAll = useCallback(() => {
    const allIds = new Set<string>();
    
    const collectIds = (nodes: MediaTreeNode[]) => {
      nodes.forEach((node) => {
        if (node.children && node.children.length > 0) {
          allIds.add(node.id);
          collectIds(node.children);
        }
      });
    };
    
    collectIds(tree);
    setExpandedIds(allIds);
  }, [tree]);

  /**
   * Collapse all nodes
   */
  const collapseAll = useCallback(() => {
    setExpandedIds(new Set());
  }, []);

  /**
   * Clear all selections
   */
  const clearSelection = useCallback(() => {
    setSelectedIds(new Set());
  }, []);

  /**
   * Select all items
   */
  const selectAll = useCallback(() => {
    const allIds = new Set<string>();
    
    const collectIds = (nodes: MediaTreeNode[]) => {
      nodes.forEach((node) => {
        allIds.add(node.id);
        if (node.children) {
          collectIds(node.children);
        }
      });
    };
    
    collectIds(tree);
    setSelectedIds(allIds);
  }, [tree]);

  return {
    tree,
    flattenedNodes: filteredAndFlattenedNodes,
    toggleNode,
    selectNode,
    getSelectedMedia,
    getSelectedIds,
    expandAll,
    collapseAll,
    clearSelection,
    selectAll,
  };
}

