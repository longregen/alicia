'use client';

import * as React from 'react';
import * as PopoverPrimitive from '@radix-ui/react-popover';
import { cls } from '../../utils/cls';
import { useIsTouchDevice } from '../../hooks/useIsTouchDevice';

export interface HoverPopoverProps {
  children: React.ReactNode;
  content: React.ReactNode;
  side?: 'top' | 'bottom' | 'left' | 'right';
  align?: 'start' | 'center' | 'end';
  sideOffset?: number;
  alignOffset?: number;
  openDelay?: number;
  closeDelay?: number;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  contentClassName?: string;
  width?: string;
  interactive?: boolean;
}

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

  const openTimeoutRef = React.useRef<ReturnType<typeof setTimeout> | null>(null);
  const closeTimeoutRef = React.useRef<ReturnType<typeof setTimeout> | null>(null);

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

  const handleTriggerMouseEnter = React.useCallback(() => {
    if (isTouchDevice) return;

    clearTimeouts();
    openTimeoutRef.current = setTimeout(() => {
      setOpen(true);
    }, openDelay);
  }, [isTouchDevice, clearTimeouts, openDelay, setOpen]);

  const handleTriggerMouseLeave = React.useCallback(() => {
    if (isTouchDevice) return;

    clearTimeouts();
    closeTimeoutRef.current = setTimeout(() => {
      setOpen(false);
    }, closeDelay);
  }, [isTouchDevice, clearTimeouts, closeDelay, setOpen]);

  const handleContentMouseEnter = React.useCallback(() => {
    if (isTouchDevice || !interactive) return;
    clearTimeouts();
  }, [isTouchDevice, interactive, clearTimeouts]);

  const handleContentMouseLeave = React.useCallback(() => {
    if (isTouchDevice) return;

    clearTimeouts();
    closeTimeoutRef.current = setTimeout(() => {
      setOpen(false);
    }, closeDelay);
  }, [isTouchDevice, clearTimeouts, closeDelay, setOpen]);

  const handleTriggerClick = React.useCallback(
    (e: React.MouseEvent) => {
      if (isTouchDevice) {
        e.preventDefault();
        setOpen(!isOpen);
      }
    },
    [isTouchDevice, isOpen, setOpen]
  );

  React.useEffect(() => {
    return () => {
      clearTimeouts();
    };
  }, [clearTimeouts]);

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
            'bg-popover text-popover-foreground',
            'rounded-lg border border-border-emphasis p-4 shadow-xl',
            'backdrop-blur-sm',
            'data-[state=open]:animate-in data-[state=closed]:animate-out',
            'data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0',
            'data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95',
            'data-[side=bottom]:slide-in-from-top-2',
            'data-[side=left]:slide-in-from-right-2',
            'data-[side=right]:slide-in-from-left-2',
            'data-[side=top]:slide-in-from-bottom-2',
            'z-50',
            width,
            'max-w-[calc(100vw-32px)]',
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
