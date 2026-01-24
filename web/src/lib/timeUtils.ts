/**
 * Format a timestamp as relative time (e.g., "5m ago", "2h ago", "Yesterday")
 */
export function formatRelativeTime(timestamp: string | Date): string {
  const date = typeof timestamp === 'string' ? new Date(timestamp) : timestamp;
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSeconds = Math.floor(diffMs / 1000);
  const diffMinutes = Math.floor(diffSeconds / 60);
  const diffHours = Math.floor(diffMinutes / 60);
  const diffDays = Math.floor(diffHours / 24);

  // Less than a minute
  if (diffSeconds < 60) {
    return 'Just now';
  }

  // Less than an hour
  if (diffMinutes < 60) {
    return `${diffMinutes}m ago`;
  }

  // Less than a day
  if (diffHours < 24) {
    return `${diffHours}h ago`;
  }

  // Yesterday
  if (diffDays === 1) {
    return 'Yesterday';
  }

  // Less than a week
  if (diffDays < 7) {
    return `${diffDays}d ago`;
  }

  // More than a week - show date
  const month = date.getMonth() + 1;
  const day = date.getDate();
  return `${month}/${day}`;
}
