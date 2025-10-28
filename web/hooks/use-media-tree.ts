import { useMemo, useState, useCallback } from "react";
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
}

/**
 * Hook for organizing media into a hierarchical tree structure
 * and managing selection/expansion state
 */
export function useMediaTree({
  media,
  searchQuery = "",
}: UseMediaTreeOptions): UseMediaTreeResult {
  // Track expanded node IDs
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());
  
  // Track selected node IDs
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

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
        const episodeNodes: MediaTreeNode[] = sortedEpisodes.map((episode) => ({
          id: `episode:${episode.id}`,
          type: "episode",
          label: formatEpisodeLabel(episode),
          media: episode,
          expanded: false,
          selected: selectedIds.has(`episode:${episode.id}`),
          indeterminate: false,
          depth: 2,
          parentId: seasonId,
        }));
        
        // Check if all episodes are selected
        const allSelected = episodeNodes.every((n) => n.selected);
        const someSelected = episodeNodes.some((n) => n.selected);
        
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
          depth: 1,
          parentId: showId,
        };
        
        seasonNodes.push(seasonNode);
      });
      
      // Check if all seasons are selected
      const allSeasonsSelected = seasonNodes.every((n) => n.selected);
      const someSeasonsSelected = seasonNodes.some((n) => n.selected || n.indeterminate);
      
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
        depth: 0,
      };
      
      treeNodes.push(showNode);
    });
    
    return treeNodes;
  }, [media, expandedIds, selectedIds]);

  /**
   * Filter and flatten tree based on search query
   */
  const filteredAndFlattenedNodes = useMemo(() => {
    const flatten = (nodes: MediaTreeNode[]): FlattenedNode[] => {
      const result: FlattenedNode[] = [];
      
      const traverse = (node: MediaTreeNode) => {
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
        
        // If expanded (or forced by search), add children
        const shouldExpand = node.expanded || (searchQuery !== "" && hasMatchingDescendant(node));
        
        if (shouldExpand && node.children) {
          node.children.forEach(traverse);
        }
      };
      
      nodes.forEach(traverse);
      return result;
    };
    
    return flatten(tree);
  }, [tree, searchQuery]);

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
      if (!node) return prev;
      
      // Update this node
      if (selected) {
        next.add(nodeId);
      } else {
        next.delete(nodeId);
      }
      
      // Cascade to all descendants
      const cascadeToChildren = (n: MediaTreeNode) => {
        if (n.children) {
          n.children.forEach((child) => {
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

