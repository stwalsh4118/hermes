"use client";

import { ChevronRight, ChevronDown, Folder, Film } from "lucide-react";
import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";
import {
  MediaTreeNode,
  isEpisodeNode,
  hasChildren,
  getEpisodeMetadata,
} from "@/lib/types/media-tree";

interface MediaTreeNodeProps {
  node: MediaTreeNode;
  depth: number;
  onToggle: (nodeId: string) => void;
  onSelect: (nodeId: string, selected: boolean) => void;
  searchQuery?: string;
  isActive?: boolean;
}

/**
 * Individual node in the media tree
 */
export function MediaTreeNodeComponent({
  node,
  depth,
  onToggle,
  onSelect,
  searchQuery = "",
  isActive = false,
}: MediaTreeNodeProps) {
  const hasChildNodes = hasChildren(node);
  const isEpisode = isEpisodeNode(node);
  const indentSize = 24; // 24px per depth level
  const paddingLeft = depth * indentSize;

  const handleCheckboxChange = (checked: boolean | "indeterminate") => {
    onSelect(node.id, checked === true);
  };

  const handleToggle = () => {
    if (hasChildNodes) {
      onToggle(node.id);
    }
  };

  const handleRowClick = (e: React.MouseEvent) => {
    // Don't toggle if clicking on checkbox
    const target = e.target as HTMLElement;
    if (target.closest('[role="checkbox"]')) {
      return;
    }
    
    if (hasChildNodes) {
      onToggle(node.id);
    }
  };

  // Highlight search matches
  const highlightText = (text: string) => {
    if (!searchQuery) return text;
    
    const lowerText = text.toLowerCase();
    const lowerQuery = searchQuery.toLowerCase();
    const index = lowerText.indexOf(lowerQuery);
    
    if (index === -1) return text;
    
    return (
      <>
        {text.slice(0, index)}
        <mark className="bg-accent/40 text-accent-foreground font-bold">
          {text.slice(index, index + searchQuery.length)}
        </mark>
        {text.slice(index + searchQuery.length)}
      </>
    );
  };

  return (
    <div
      className={cn(
        "flex items-center gap-2 py-2 px-3 font-mono text-sm",
        "border-b border-border/50",
        "transition-colors",
        node.disabled 
          ? "opacity-50 cursor-not-allowed" 
          : "hover:bg-muted/30 cursor-pointer",
        "select-none",
        isActive && "bg-accent/20 ring-2 ring-primary ring-inset"
      )}
      style={{ paddingLeft: `${paddingLeft + 12}px` }}
      onClick={node.disabled ? undefined : handleRowClick}
      role="treeitem"
      aria-expanded={hasChildNodes ? node.expanded : undefined}
      aria-selected={node.selected}
      aria-level={depth + 1}
      aria-disabled={node.disabled}
    >
      {/* Expand/collapse chevron */}
      <div className="w-5 h-5 flex items-center justify-center shrink-0">
        {hasChildNodes ? (
          <button
            onClick={(e) => {
              e.stopPropagation();
              handleToggle();
            }}
            className="hover:text-primary transition-colors"
            aria-label={node.expanded ? "Collapse" : "Expand"}
          >
            {node.expanded ? (
              <ChevronDown className="w-4 h-4" />
            ) : (
              <ChevronRight className="w-4 h-4" />
            )}
          </button>
        ) : (
          <div className="w-4 h-4" />
        )}
      </div>

      {/* Checkbox */}
      <div
        onClick={(e) => e.stopPropagation()}
        className="flex items-center"
      >
        <Checkbox
          checked={node.selected ? true : node.indeterminate ? "indeterminate" : false}
          onCheckedChange={handleCheckboxChange}
          disabled={node.disabled}
          aria-label={`Select ${node.label}`}
        />
      </div>

      {/* Icon */}
      <div className="w-5 h-5 flex items-center justify-center shrink-0 text-muted-foreground">
        {isEpisode ? (
          <Film className="w-4 h-4" />
        ) : (
          <Folder className="w-4 h-4" />
        )}
      </div>

      {/* Content */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-3">
          {/* Label */}
          <span
            className={cn(
              "font-bold truncate",
              node.type === "show" && "text-base",
              node.type === "season" && "text-sm",
              node.type === "episode" && "text-sm"
            )}
          >
            {highlightText(node.label)}
          </span>

          {/* Episode count for show/season */}
          {node.episodeCount !== undefined && (
            <span className="text-xs text-muted-foreground whitespace-nowrap">
              {node.episodeCount} {node.episodeCount === 1 ? "episode" : "episodes"}
            </span>
          )}
          
          {/* Already Added badge for disabled items */}
          {node.disabled && (
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-muted text-muted-foreground border border-border font-bold text-xs whitespace-nowrap">
              ALREADY ADDED
            </span>
          )}
        </div>

        {/* Episode metadata (resolution, codec, duration) */}
        {isEpisode && node.media && (
          <div className="text-xs text-muted-foreground mt-0.5 truncate hidden sm:block">
            {getEpisodeMetadata(node.media)}
          </div>
        )}
      </div>
    </div>
  );
}

