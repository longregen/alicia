import * as React from 'react';
import * as TooltipPrimitive from '@radix-ui/react-tooltip';
import { cn } from '../../lib/utils';
import type { BaseComponentProps } from '../../types/components';

/**
 * Tooltip atom component for displaying contextual information on hover.
 * Built with Radix UI primitives for accessibility and proper positioning.
 */

export type TooltipPosition = 'top' | 'bottom' | 'left' | 'right';

// Provider component
function TooltipProvider({
  delayDuration = 0,
  ...props
}: React.ComponentProps<typeof TooltipPrimitive.Provider>) {
  return (
    <TooltipPrimitive.Provider
      data-slot="tooltip-provider"
      delayDuration={delayDuration}
      {...props}
    />
  );
}

// Root component with built-in provider
function TooltipRoot({
  ...props
}: React.ComponentProps<typeof TooltipPrimitive.Root>) {
  return (
    <TooltipProvider>
      <TooltipPrimitive.Root data-slot="tooltip" {...props} />
    </TooltipProvider>
  );
}

// Trigger component
function TooltipTrigger({
  ...props
}: React.ComponentProps<typeof TooltipPrimitive.Trigger>) {
  return <TooltipPrimitive.Trigger data-slot="tooltip-trigger" {...props} />;
}

// Content component
function TooltipContent({
  className,
  sideOffset = 4,
  children,
  ...props
}: React.ComponentProps<typeof TooltipPrimitive.Content>) {
  return (
    <TooltipPrimitive.Portal>
      <TooltipPrimitive.Content
        data-slot="tooltip-content"
        sideOffset={sideOffset}
        className={cn(
          'z-50 origin-(--radix-tooltip-content-transform-origin) rounded-md px-3 py-2 text-xs',
          'bg-overlay backdrop-blur-sm text-on-emphasis',
          'border border-border-emphasis shadow-lg',
          'animate-in fade-in-0 zoom-in-95',
          'data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95',
          'data-[side=bottom]:slide-in-from-top-2',
          'data-[side=left]:slide-in-from-right-2',
          'data-[side=right]:slide-in-from-left-2',
          'data-[side=top]:slide-in-from-bottom-2',
          className
        )}
        {...props}
      >
        {children}
        <TooltipPrimitive.Arrow className="fill-overlay" />
      </TooltipPrimitive.Content>
    </TooltipPrimitive.Portal>
  );
}

// Backwards-compatible wrapper component
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
  delay = 0,
  disabled = false,
  className = '',
  children,
}) => {
  if (disabled || !content) {
    return <>{children}</>;
  }

  return (
    <TooltipProvider delayDuration={delay}>
      <TooltipPrimitive.Root data-slot="tooltip">
        <TooltipPrimitive.Trigger data-slot="tooltip-trigger" asChild>
          <span className={cn('inline-block', className)}>{children}</span>
        </TooltipPrimitive.Trigger>
        <TooltipContent side={position}>{content}</TooltipContent>
      </TooltipPrimitive.Root>
    </TooltipProvider>
  );
};

// Named exports for compound component pattern
export { TooltipProvider, TooltipRoot, TooltipTrigger, TooltipContent };

// Default export for backwards compatibility
export default Tooltip;
