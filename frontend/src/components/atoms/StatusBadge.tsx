import React from 'react';
import Badge, { type BadgeProps } from './Badge';
import type { Variant } from '../../types/components';

/**
 * StatusBadge atom component for displaying status indicators.
 * Pre-configured Badge with status-specific styling and dot indicator.
 */

export type StatusType = 'idle' | 'running' | 'completed' | 'error' | 'warning';

export interface StatusBadgeProps extends Omit<BadgeProps, 'variant' | 'showDot'> {
  /** Status type determines the variant and styling */
  status: StatusType;
}

const statusToVariant: Record<StatusType, Variant> = {
  idle: 'default',
  running: 'primary',
  completed: 'success',
  error: 'error',
  warning: 'warning',
};

const statusLabels: Record<StatusType, string> = {
  idle: 'Idle',
  running: 'Running',
  completed: 'Completed',
  error: 'Error',
  warning: 'Warning',
};

const StatusBadge: React.FC<StatusBadgeProps> = ({
  status,
  children,
  ...props
}) => {
  const variant = statusToVariant[status];
  const label = children || statusLabels[status];

  return (
    <Badge {...props} variant={variant} showDot>
      {label}
    </Badge>
  );
};

export default StatusBadge;
