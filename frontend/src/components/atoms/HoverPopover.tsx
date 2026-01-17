'use client';

import * as React from 'react';
import * as PopoverPrimitive from '@radix-ui/react-popover';
import { cls } from '../../utils/cls';
import { useIsTouchDevice } from '../../hooks/useIsTouchDevice';

export interface HoverPopoverProps {
  /** The trigger element */
  children: React.ReactNode;
  /** The popover content */
  content: React.ReactNode;
  /** Preferred side for the popover */
  side?: 'top' | 'bottom' | 'left' | 'right';
  /** Alignment relative to the trigger */
  align?: 'start' | 'center' | 'end';
  /** Offset from the trigger (perpendicular to side) */
  sideOffset?: number;
  /** Offset along the alignment axis */
  alignOffset?: number;
  /** Delay before opening on hover (desktop only, ms) */
  openDelay?: number;
  /** Delay before closing when mouse leaves (desktop only, ms) */
  closeDelay?: number;
  /** Controlled open state */
  open?: boolean;
  /** Callback when open state changes */
  onOpenChange?: (open: boolean) => void;
  /** Additional class name for the content */
  contentClassName?: string;
  /** Width of the popover content */
  width?: string;
  /** Whether the popover content is interactive (affects close behavior) */
  interactive?: boolean;
}

/**
 * HoverPopover - A popover that opens on hover (desktop) or tap (mobile)
 *
 * Features:
 * - Desktop: Opens on hover after configurable delay, stays open when moving to content
 * - Mobile: Opens on tap, closes on tap elsewhere
 * - Positioned "up and to the right" by default (side="top", align="start", alignOffset=8)
 * - Automatic collision detection and repositioning
 * - Accessible: keyboard navigation, screen reader support
 */
export const HoverPopover: React.FC<HoverPopoverProps> = ({
  children,
  content,
  side = 'top',
  align = 'start',
  sideOffset = 8,
  alignOffset = 8,
  openDelay = 150,
  closeDelay = 300,
  open: controlledOpen,
  onOpenChange,
  contentClassName,
  width = 'w-80',
  interactive = true,
}) => {
  const isTouchDevice = useIsTouchDevice();
  const [internalOpen, setInternalOpen] = React.useState(false);

  // Use controlled or internal state
  const isControlled = controlledOpen !== undefined;
  const isOpen = isControlled ? controlledOpen : internalOpen;

  const setOpen = React.useCallback(
    (value: boolean) => {
      if (!isControlled) {
        setInternalOpen(value);
      }
      onOpenChange?.(value);
    },
    [isControlled, onOpenChange]
  );

  // Refs for timers
  const openTimeoutRef = React.useRef<ReturnType<typeof setTimeout> | null>(null);
  const closeTimeoutRef = React.useRef<ReturnType<typeof setTimeout> | null>(null);

  // Clear all timeouts
  const clearTimeouts = React.useCallback(() => {
    if (openTimeoutRef.current) {
      clearTimeout(openTimeoutRef.current);
      openTimeoutRef.current = null;
    }
    if (closeTimeoutRef.current) {
      clearTimeout(closeTimeoutRef.current);
      closeTimeoutRef.current = null;
    }
  }, []);

  // Handle mouse enter on trigger
  const handleTriggerMouseEnter = React.useCallback(() => {
    if (isTouchDevice) return;

    clearTimeouts();
    openTimeoutRef.current = setTimeout(() => {
      setOpen(true);
    }, openDelay);
  }, [isTouchDevice, clearTimeouts, openDelay, setOpen]);

  // Handle mouse leave on trigger
  const handleTriggerMouseLeave = React.useCallback(() => {
    if (isTouchDevice) return;

    clearTimeouts();
    closeTimeoutRef.current = setTimeout(() => {
      setOpen(false);
    }, closeDelay);
  }, [isTouchDevice, clearTimeouts, closeDelay, setOpen]);

  // Handle mouse enter on content (keeps popover open)
  const handleContentMouseEnter = React.useCallback(() => {
    if (isTouchDevice || !interactive) return;
    clearTimeouts();
  }, [isTouchDevice, interactive, clearTimeouts]);

  // Handle mouse leave on content
  const handleContentMouseLeave = React.useCallback(() => {
    if (isTouchDevice) return;

    clearTimeouts();
    closeTimeoutRef.current = setTimeout(() => {
      setOpen(false);
    }, closeDelay);
  }, [isTouchDevice, clearTimeouts, closeDelay, setOpen]);

  // Handle click for mobile (toggle)
  const handleTriggerClick = React.useCallback(
    (e: React.MouseEvent) => {
      if (isTouchDevice) {
        e.preventDefault();
        setOpen(!isOpen);
      }
    },
    [isTouchDevice, isOpen, setOpen]
  );

  // Cleanup timeouts on unmount
  React.useEffect(() => {
    return () => {
      clearTimeouts();
    };
  }, [clearTimeouts]);

  // Handle escape key
  const handleEscapeKeyDown = React.useCallback(() => {
    setOpen(false);
  }, [setOpen]);

  return (
    <PopoverPrimitive.Root open={isOpen} onOpenChange={setOpen}>
      <PopoverPrimitive.Trigger asChild>
        <div
          onMouseEnter={handleTriggerMouseEnter}
          onMouseLeave={handleTriggerMouseLeave}
          onClick={handleTriggerClick}
          className="inline-block"
        >
          {children}
        </div>
      </PopoverPrimitive.Trigger>

      <PopoverPrimitive.Portal>
        <PopoverPrimitive.Content
          data-slot="hover-popover-content"
          side={side}
          align={align}
          sideOffset={sideOffset}
          alignOffset={alignOffset}
          avoidCollisions={true}
          collisionPadding={16}
          onMouseEnter={handleContentMouseEnter}
          onMouseLeave={handleContentMouseLeave}
          onEscapeKeyDown={handleEscapeKeyDown}
          onPointerDownOutside={() => {
            if (isTouchDevice) {
              setOpen(false);
            }
          }}
          className={cls(
            // Base styling
            'bg-popover text-popover-foreground',
            'rounded-lg border border-border-emphasis p-4 shadow-xl',
            'backdrop-blur-sm',
            // Animation
            'data-[state=open]:animate-in data-[state=closed]:animate-out',
            'data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0',
            'data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95',
            // Slide direction based on actual position
            'data-[side=bottom]:slide-in-from-top-2',
            'data-[side=left]:slide-in-from-right-2',
            'data-[side=right]:slide-in-from-left-2',
            'data-[side=top]:slide-in-from-bottom-2',
            // Z-index and sizing
            'z-50',
            width,
            'max-w-[calc(100vw-32px)]',
            // Focus outline
            'outline-hidden focus:outline-none',
            contentClassName
          )}
        >
          {content}
        </PopoverPrimitive.Content>
      </PopoverPrimitive.Portal>
    </PopoverPrimitive.Root>
  );
};

export default HoverPopover;
