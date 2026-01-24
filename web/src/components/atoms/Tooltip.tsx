import * as React from 'react';
import * as TooltipPrimitive from '@radix-ui/react-tooltip';
import { cls } from '../../utils/cls';
import type { BaseComponentProps } from '../../types/components';

export type TooltipPosition = 'top' | 'bottom' | 'left' | 'right';

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
        className={cls(
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

export interface TooltipProps extends BaseComponentProps {
  content: string;
  position?: TooltipPosition;
  delay?: number;
  disabled?: boolean;
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
          <span className={cls('inline-block', className)}>{children}</span>
        </TooltipPrimitive.Trigger>
        <TooltipContent side={position}>{content}</TooltipContent>
      </TooltipPrimitive.Root>
    </TooltipProvider>
  );
};

export { TooltipProvider, TooltipContent, Tooltip };
