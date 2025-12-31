import React from 'react';
import { cls } from '../../utils/cls';

/**
 * SectionHeader component for consistent header patterns across the app.
 * Provides a title with optional action buttons and divider.
 */

export interface SectionHeaderProps extends React.HTMLAttributes<HTMLDivElement> {
  /** The header title */
  title: string;
  /** Optional subtitle or description */
  subtitle?: string;
  /** Action element (typically buttons) to display on the right */
  action?: React.ReactNode;
  /** Whether to show a bottom border */
  bordered?: boolean;
  /** Size variant */
  size?: 'sm' | 'md' | 'lg';
}

const sizeStyles = {
  sm: {
    container: 'px-3 py-2',
    title: 'text-sm font-medium',
    subtitle: 'text-xs',
  },
  md: {
    container: 'px-4 py-3',
    title: 'text-base font-semibold',
    subtitle: 'text-sm',
  },
  lg: {
    container: 'px-6 py-4',
    title: 'text-lg font-semibold',
    subtitle: 'text-sm',
  },
};

const SectionHeader = React.forwardRef<HTMLDivElement, SectionHeaderProps>(
  ({ className, title, subtitle, action, bordered = true, size = 'md', ...props }, ref) => {
    const styles = sizeStyles[size];

    return (
      <div
        ref={ref}
        className={cls(
          'layout-between',
          styles.container,
          bordered && 'border-b border-default',
          className
        )}
        {...props}
      >
        <div className="layout-stack">
          <h2 className={cls(styles.title, 'text-default')}>{title}</h2>
          {subtitle && (
            <p className={cls(styles.subtitle, 'text-muted-foreground')}>{subtitle}</p>
          )}
        </div>
        {action && <div className="layout-center-gap">{action}</div>}
      </div>
    );
  }
);

SectionHeader.displayName = 'SectionHeader';

export default SectionHeader;
