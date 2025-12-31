import React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '../../lib/utils';

/**
 * Base Badge component for displaying status, scores, and counts.
 * Uses CVA for variant management.
 */

const badgeVariants = cva(
  'inline-flex items-center rounded-md border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2',
  {
    variants: {
      variant: {
        default: 'border-transparent bg-primary text-primary-foreground shadow hover:bg-primary/80',
        secondary: 'border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80',
        destructive: 'border-transparent bg-destructive text-destructive-foreground shadow hover:bg-destructive/80',
        outline: 'text-foreground',
        success: 'border-transparent bg-green-500 text-white shadow hover:bg-green-600',
        warning: 'border-transparent bg-yellow-500 text-white shadow hover:bg-yellow-600',
        error: 'border-transparent bg-red-500 text-white shadow hover:bg-red-600',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  }
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {
  /** Optional icon or prefix */
  icon?: React.ReactNode;
  /** Optional dot indicator */
  showDot?: boolean;
  /** Dot color (only shown if showDot is true) */
  dotColor?: string;
}

const Badge = React.forwardRef<HTMLDivElement, BadgeProps>(
  ({ className, variant, icon, showDot = false, dotColor, children, ...props }, ref) => {
    const dotColorClass = dotColor || 'bg-current';

    return (
      <div
        ref={ref}
        className={cn(badgeVariants({ variant }), className)}
        data-slot="badge"
        {...props}
      >
        {showDot && (
          <span className={cn('w-1.5 h-1.5 rounded-full', dotColorClass)} />
        )}
        {icon && <span className="flex items-center">{icon}</span>}
        {children}
      </div>
    );
  }
);

Badge.displayName = 'Badge';

export default Badge;
export { badgeVariants };
