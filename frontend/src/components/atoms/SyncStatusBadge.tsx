import React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cls } from '../../utils/cls';
import type { SyncStatus } from '../../types/models';

const syncStatusBadgeVariants = cva(
  'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium transition-colors',
  {
    variants: {
      status: {
        pending: 'bg-warning/20 text-warning',
        synced: 'bg-success/20 text-success',
        conflict: 'bg-destructive/20 text-destructive cursor-pointer hover:bg-destructive/30',
      },
    },
    defaultVariants: {
      status: 'pending',
    },
  }
);

export interface SyncStatusBadgeProps
  extends Omit<React.HTMLAttributes<HTMLSpanElement>, 'children'>,
    VariantProps<typeof syncStatusBadgeVariants> {
  status: SyncStatus;
  showLabel?: boolean;
}

const statusLabels: Record<SyncStatus, string> = {
  pending: 'Syncing...',
  synced: 'Synced',
  conflict: 'Conflict',
};

const statusIcons: Record<SyncStatus, React.ReactNode> = {
  pending: (
    <svg className="size-3 animate-spin" viewBox="0 0 24 24" fill="none">
      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
      <path
        className="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
      />
    </svg>
  ),
  synced: (
    <svg className="size-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <polyline points="20 6 9 17 4 12" />
    </svg>
  ),
  conflict: (
    <svg className="size-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
    </svg>
  ),
};

const SyncStatusBadge = React.forwardRef<HTMLSpanElement, SyncStatusBadgeProps>(
  ({ className, status, showLabel = true, ...props }, ref) => {
    return (
      <span
        ref={ref}
        className={cls(syncStatusBadgeVariants({ status }), className)}
        {...props}
      >
        {statusIcons[status]}
        {showLabel && <span>{statusLabels[status]}</span>}
      </span>
    );
  }
);

SyncStatusBadge.displayName = 'SyncStatusBadge';

export default SyncStatusBadge;
