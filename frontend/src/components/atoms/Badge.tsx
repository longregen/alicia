import React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cls } from '../../utils/cls';

/**
 * Base Badge component for displaying status, scores, and counts.
 * Uses CVA for variant management.
 */

const badgeVariants = cva(
  'inline-flex items-center gap-1.5 rounded-md border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2',
  {
    variants: {
      variant: {
        default: 'border-transparent bg-primary text-primary-foreground hover:bg-primary/80',
        secondary: 'border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80',
        destructive: 'border-transparent bg-destructive text-destructive-foreground hover:bg-destructive/80',
        outline: 'border-border text-foreground',
        success: 'border-transparent bg-success text-success-foreground hover:bg-success/80',
        warning: 'border-transparent bg-warning text-warning-foreground hover:bg-warning/80',
        error: 'border-transparent bg-destructive text-destructive-foreground hover:bg-destructive/80',
        muted: 'border-transparent bg-muted text-muted-foreground',
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
  /** Dot color class name (Tailwind class, only shown if showDot is true) */
  dotColor?: string;
}

const Badge = React.forwardRef<HTMLDivElement, BadgeProps>(
  ({ className, variant, icon, showDot = false, dotColor, children, ...props }, ref) => {
    const dotColorClass = dotColor || 'bg-current';

    return (
      <div
        ref={ref}
        className={cls(badgeVariants({ variant }), className)}
        data-slot="badge"
        {...props}
      >
        {showDot && (
          <span className={cls('w-1.5 h-1.5 rounded-full', dotColorClass)} />
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
