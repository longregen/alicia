import React, { useState, useRef, useEffect } from 'react';
import { cls } from '../../utils/cls';
import type { BaseComponentProps } from '../../types/components';

/**
 * Tooltip atom component for displaying contextual information on hover.
 * Supports multiple positions and delay options.
 */

export type TooltipPosition = 'top' | 'bottom' | 'left' | 'right';

export interface TooltipProps extends BaseComponentProps {
  /** Tooltip content text */
  content: string;
  /** Position relative to the trigger element */
  position?: TooltipPosition;
  /** Delay before showing tooltip (ms) */
  delay?: number;
  /** Whether tooltip is disabled */
  disabled?: boolean;
  /** Element that triggers the tooltip */
  children: React.ReactNode;
}

const Tooltip: React.FC<TooltipProps> = ({
  content,
  position = 'top',
  delay = 200,
  disabled = false,
  className = '',
  children,
}) => {
  const [isVisible, setIsVisible] = useState(false);
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);

  const handleMouseEnter = () => {
    if (disabled) return;

    timeoutRef.current = setTimeout(() => {
      setIsVisible(true);
    }, delay);
  };

  const handleMouseLeave = () => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
    setIsVisible(false);
  };

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  const positionClasses = {
    top: 'bottom-full left-1/2 -translate-x-1/2 mb-2',
    bottom: 'top-full left-1/2 -translate-x-1/2 mt-2',
    left: 'right-full top-1/2 -translate-y-1/2 mr-2',
    right: 'left-full top-1/2 -translate-y-1/2 ml-2',
  };

  const arrowClasses = {
    top: 'top-full left-1/2 -translate-x-1/2 border-t-gray-900/95',
    bottom: 'bottom-full left-1/2 -translate-x-1/2 border-b-gray-900/95',
    left: 'left-full top-1/2 -translate-y-1/2 border-l-gray-900/95',
    right: 'right-full top-1/2 -translate-y-1/2 border-r-gray-900/95',
  };

  const tooltipClasses = cls(
    'absolute',
    positionClasses[position],
    'px-3',
    'py-2',
    'text-xs',
    'text-white',
    'bg-gray-900/95',
    'backdrop-blur-sm',
    'rounded-md',
    'opacity-0',
    'transition-opacity',
    'duration-200',
    'pointer-events-none',
    'whitespace-nowrap',
    'z-50',
    'shadow-lg',
    'border',
    'border-gray-700/50',
    'min-w-[8rem]',
    'max-w-xs',
    isVisible ? 'opacity-100' : ''
  );

  return (
    <div
      className={cls('relative inline-block', className)}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
      {children}
      {!disabled && content && (
        <div className={tooltipClasses}>
          {content}
          {/* Tooltip arrow */}
          <div
            className={cls(
              'absolute',
              arrowClasses[position],
              'border-4',
              'border-transparent'
            )}
          />
        </div>
      )}
    </div>
  );
};

export default Tooltip;
