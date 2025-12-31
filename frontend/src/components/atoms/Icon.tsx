import React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '../../lib/utils';

/**
 * Icon wrapper component for consistent icon sizing and styling.
 * Wraps any SVG icon component with standardized size and color options.
 */

const iconVariants = cva('shrink-0', {
  variants: {
    size: {
      xs: 'size-3',      // 12px
      sm: 'size-4',      // 16px
      md: 'size-5',      // 20px
      lg: 'size-6',      // 24px
      xl: 'size-8',      // 32px
    },
    color: {
      default: 'text-default',
      muted: 'text-muted-foreground',
      accent: 'text-accent',
      error: 'text-error',
      success: 'text-success',
      warning: 'text-warning',
      inherit: '',
    },
  },
  defaultVariants: {
    size: 'sm',
    color: 'inherit',
  },
});

export interface IconProps
  extends Omit<React.HTMLAttributes<HTMLSpanElement>, 'color'>,
    VariantProps<typeof iconVariants> {
  /** The icon component to render */
  icon: React.ComponentType<React.SVGProps<SVGSVGElement>>;
  /** Whether the icon should spin (for loading states) */
  spin?: boolean;
}

const Icon = React.forwardRef<HTMLSpanElement, IconProps>(
  ({ className, size, color, icon: IconComponent, spin, ...props }, ref) => {
    return (
      <span
        ref={ref}
        className={cn(iconVariants({ size, color }), spin && 'animate-spin', className)}
        {...props}
      >
        <IconComponent className="size-full" />
      </span>
    );
  }
);

Icon.displayName = 'Icon';

export default Icon;
export { iconVariants };
