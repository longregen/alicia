import React from 'react';
import { cls } from '../../utils/cls';

export interface StarRatingProps {
  rating: number | null;
  onRate: (rating: number) => void;
  isLoading?: boolean;
  className?: string;
  compact?: boolean;
  readOnly?: boolean;
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
      if (rating === starIndex) {
        onRate(0);
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

  const starSize = compact ? 'w-5 h-5' : 'w-6 h-6';

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
              'transition-all duration-150',
              compact ? 'p-1' : 'p-1.5',
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

      {showValue && rating !== null && rating > 0 && (
        <span className={cls(
          'ml-1 text-muted-foreground',
          compact ? 'text-xs' : 'text-sm'
        )}>
          {rating}/5
        </span>
      )}

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

export function starToImportance(stars: number): number {
  if (stars <= 0) return 0.5;
  return Math.min(1.0, Math.max(0.2, stars * 0.2));
}

export function importanceToStar(importance: number): number {
  if (importance <= 0) return 0;
  return Math.round(importance * 5);
}
