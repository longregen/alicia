import React from 'react';
import { cls } from '../../utils/cls';

/**
 * StarRating atom component for 1-click importance ranking.
 * Displays 1-5 stars that can be clicked to set importance.
 * Maps star rating to importance: 1 star = 0.2, 5 stars = 1.0
 */

export interface StarRatingProps {
  /** Current rating (1-5) or null if unrated */
  rating: number | null;
  /** Callback when user clicks a star */
  onRate: (rating: number) => void;
  /** Whether the rating is being processed */
  isLoading?: boolean;
  /** Additional CSS classes */
  className?: string;
  /** Compact mode for inline display */
  compact?: boolean;
  /** Read-only mode (no interactions) */
  readOnly?: boolean;
  /** Show numeric value beside stars */
  showValue?: boolean;
}

const StarRating: React.FC<StarRatingProps> = ({
  rating,
  onRate,
  isLoading = false,
  className = '',
  compact = false,
  readOnly = false,
  showValue = false,
}) => {
  const [hoverRating, setHoverRating] = React.useState<number | null>(null);

  const handleClick = (starIndex: number) => {
    if (!isLoading && !readOnly) {
      // Toggle off if clicking the same rating
      if (rating === starIndex) {
        onRate(0); // Clear rating
      } else {
        onRate(starIndex);
      }
    }
  };

  const handleMouseEnter = (starIndex: number) => {
    if (!isLoading && !readOnly) {
      setHoverRating(starIndex);
    }
  };

  const handleMouseLeave = () => {
    setHoverRating(null);
  };

  const displayRating = hoverRating ?? rating ?? 0;

  const starSize = compact ? 'w-3.5 h-3.5' : 'w-5 h-5';

  return (
    <div className={cls('flex items-center gap-0.5', className)}>
      {[1, 2, 3, 4, 5].map((starIndex) => {
        const isFilled = starIndex <= displayRating;

        return (
          <button
            key={starIndex}
            type="button"
            onClick={() => handleClick(starIndex)}
            onMouseEnter={() => handleMouseEnter(starIndex)}
            onMouseLeave={handleMouseLeave}
            disabled={isLoading || readOnly}
            aria-label={`Rate ${starIndex} star${starIndex > 1 ? 's' : ''}`}
            className={cls(
              'p-0.5 transition-all duration-150',
              isLoading || readOnly
                ? 'cursor-default'
                : 'cursor-pointer hover:scale-110',
              isLoading && 'opacity-50'
            )}
          >
            <svg
              className={cls(
                starSize,
                'transition-colors duration-150',
                isFilled
                  ? 'text-amber-400 fill-amber-400'
                  : 'text-muted-foreground/40 fill-transparent',
                !readOnly && !isLoading && hoverRating !== null && 'drop-shadow-sm'
              )}
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={1.5}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M11.48 3.499a.562.562 0 011.04 0l2.125 5.111a.563.563 0 00.475.345l5.518.442c.499.04.701.663.321.988l-4.204 3.602a.563.563 0 00-.182.557l1.285 5.385a.562.562 0 01-.84.61l-4.725-2.885a.563.563 0 00-.586 0L6.982 20.54a.562.562 0 01-.84-.61l1.285-5.386a.562.562 0 00-.182-.557l-4.204-3.602a.563.563 0 01.321-.988l5.518-.442a.563.563 0 00.475-.345L11.48 3.5z"
              />
            </svg>
          </button>
        );
      })}

      {/* Optional numeric display */}
      {showValue && rating !== null && rating > 0 && (
        <span className={cls(
          'ml-1 text-muted-foreground',
          compact ? 'text-xs' : 'text-sm'
        )}>
          {rating}/5
        </span>
      )}

      {/* Loading indicator */}
      {isLoading && (
        <div className={cls(
          'ml-1 border-2 border-amber-400 border-t-transparent rounded-full animate-spin',
          compact ? 'w-3 h-3' : 'w-4 h-4'
        )} />
      )}
    </div>
  );
};

export default StarRating;

/**
 * Utility to convert star rating (1-5) to importance (0.2-1.0)
 */
export function starToImportance(stars: number): number {
  if (stars <= 0) return 0.5; // Default
  return Math.min(1.0, Math.max(0.2, stars * 0.2));
}

/**
 * Utility to convert importance (0-1) to star rating (1-5)
 */
export function importanceToStar(importance: number): number {
  if (importance <= 0) return 0;
  return Math.round(importance * 5);
}
