/**
 * Format duration in seconds to human-readable string
 * @param seconds - Duration in seconds
 * @returns Formatted string like "2h 30m" or "45m" or "1h"
 */
export function formatDuration(seconds: number): string {
  if (seconds === 0) return "0m";
  
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  
  if (hours > 0 && minutes > 0) {
    return `${hours}h ${minutes}m`;
  } else if (hours > 0) {
    return `${hours}h`;
  } else {
    return `${minutes}m`;
  }
}

/**
 * Format a count of items with proper pluralization
 * @param count - Number of items
 * @param singular - Singular form (e.g., "item")
 * @param plural - Plural form (optional, defaults to singular + "s")
 * @returns Formatted string like "1 item" or "5 items"
 */
export function formatCount(count: number, singular: string, plural?: string): string {
  const word = count === 1 ? singular : (plural || `${singular}s`);
  return `${count} ${word}`;
}


