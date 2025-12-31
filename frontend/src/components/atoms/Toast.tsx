import React, { useEffect, useState, useCallback } from 'react';
import { cls } from '../../utils/cls';
import type { BaseComponentProps, Variant } from '../../types/components';

/**
 * Toast atom component for displaying temporary notifications.
 * Auto-dismisses after a specified duration.
 */

export interface ToastProps extends BaseComponentProps {
  /** Toast message content */
  message: string;
  /** Toast variant determines styling */
  variant?: Variant;
  /** Auto-dismiss duration in ms (0 = no auto-dismiss) */
  duration?: number;
  /** Callback when toast is dismissed */
  onDismiss?: () => void;
  /** Whether toast is visible */
  visible?: boolean;
  /** Show close button */
  showClose?: boolean;
}

const Toast: React.FC<ToastProps> = ({
  message,
  variant = 'default',
  duration = 3000,
  onDismiss,
  visible = true,
  showClose = true,
  className = '',
}) => {
  const [isVisible, setIsVisible] = useState(visible);
  const [isExiting, setIsExiting] = useState(false);

  useEffect(() => {
    setIsVisible(visible);
    if (visible) {
      setIsExiting(false);
    }
  }, [visible]);

  const handleDismiss = useCallback(() => {
    setIsExiting(true);
    setTimeout(() => {
      setIsVisible(false);
      onDismiss?.();
    }, 200); // Match transition duration
  }, [onDismiss]);

  useEffect(() => {
    if (isVisible && duration > 0) {
      const timer = setTimeout(() => {
        handleDismiss();
      }, duration);

      return () => clearTimeout(timer);
    }
  }, [isVisible, duration, handleDismiss]);

  if (!isVisible) {
    return null;
  }

  const variantStyles = {
    default: 'bg-surface text-default border',
    primary: 'bg-accent text-on-emphasis border-accent',
    success: 'bg-success text-on-emphasis border-success',
    warning: 'bg-warning text-on-emphasis border-warning',
    error: 'bg-error text-on-emphasis border-error',
  };

  const icons = {
    default: (
      <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
      </svg>
    ),
    primary: (
      <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
      </svg>
    ),
    success: (
      <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
      </svg>
    ),
    warning: (
      <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
      </svg>
    ),
    error: (
      <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
      </svg>
    ),
  };

  const toastClasses = cls(
    // Base styles
    'flex items-center gap-3 p-4',
    'rounded-lg border shadow-lg backdrop-blur-sm',
    'min-w-[300px] max-w-md',

    // Variant
    variantStyles[variant],

    // Animation
    'transition-all duration-200 ease-out',
    isExiting ? 'opacity-0 translate-x-full' : 'opacity-100 translate-x-0',

    // Custom classes
    className
  );

  return (
    <div className={toastClasses} role="alert">
      {/* Icon */}
      <div className="flex-shrink-0">
        {icons[variant]}
      </div>

      {/* Message */}
      <div className="flex flex-col flex-1">
        <p className="text-sm font-medium">{message}</p>
      </div>

      {/* Close button */}
      {showClose && (
        <button
          onClick={handleDismiss}
          className={cls(
            'flex-shrink-0 rounded-md p-1',
            'transition-colors duration-200',
            'hover:bg-black/10',
            'focus:outline-none focus:ring-2 focus:ring-accent'
          )}
          aria-label="Dismiss"
        >
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
          </svg>
        </button>
      )}
    </div>
  );
};

export default Toast;
