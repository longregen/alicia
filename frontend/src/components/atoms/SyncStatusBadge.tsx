import React from 'react';
import { Clock, AlertCircle } from 'lucide-react';
import Badge from './Badge';
import { cn } from '../../lib/utils';

/**
 * SyncStatusBadge component for displaying message sync status.
 *
 * Shows visual indicators for:
 * - pending: Clock icon with neutral styling
 * - synced: No indicator (default state)
 * - conflict: Warning icon with alert styling
 */

export interface SyncStatusBadgeProps {
  status?: 'pending' | 'synced' | 'conflict';
  className?: string;
}

const SyncStatusBadge: React.FC<SyncStatusBadgeProps> = ({ status, className }) => {
  // Don't render anything for synced or undefined status
  if (!status || status === 'synced') {
    return null;
  }

  if (status === 'pending') {
    return (
      <Badge
        variant="secondary"
        className={cn('text-xs gap-1', className)}
        icon={<Clock className="w-3 h-3" />}
      >
        Pending
      </Badge>
    );
  }

  if (status === 'conflict') {
    return (
      <Badge
        variant="warning"
        className={cn('text-xs gap-1', className)}
        icon={<AlertCircle className="w-3 h-3" />}
      >
        Conflict
      </Badge>
    );
  }

  return null;
};

export default SyncStatusBadge;
