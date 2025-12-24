import { useEffect, useState } from 'react';
import { useMessageContext } from '../contexts/MessageContext';
import { ErrorMessage, Severity } from '../types/protocol';
import './ErrorNotification.css';

export function ErrorNotification() {
  const { errors } = useMessageContext();
  const [visibleErrors, setVisibleErrors] = useState<ErrorMessage[]>([]);

  useEffect(() => {
    // Clear visible errors when context errors are cleared (conversation switch)
    if (errors.length === 0) {
      setVisibleErrors([]);
      return;
    }

    // Show new errors
    const newErrors = errors.filter(
      error => !visibleErrors.some(visible => visible.id === error.id)
    );

    if (newErrors.length > 0) {
      setVisibleErrors(prev => [...prev, ...newErrors]);

      // Auto-dismiss recoverable errors after 5 seconds
      newErrors.forEach(error => {
        if (error.recoverable) {
          setTimeout(() => {
            setVisibleErrors(prev => prev.filter(e => e.id !== error.id));
          }, 5000);
        }
      });
    }
  }, [errors, visibleErrors]);

  const dismissError = (errorId: string) => {
    setVisibleErrors(prev => prev.filter(e => e.id !== errorId));
  };

  const getSeverityColor = (severity: Severity): string => {
    switch (severity) {
      case Severity.Info:
        return '#2196f3';
      case Severity.Warning:
        return '#ff9800';
      case Severity.Error:
        return '#f44336';
      case Severity.Critical:
        return '#9c27b0';
      default:
        return '#757575';
    }
  };

  const getSeverityLabel = (severity: Severity): string => {
    switch (severity) {
      case Severity.Info:
        return 'INFO';
      case Severity.Warning:
        return 'WARNING';
      case Severity.Error:
        return 'ERROR';
      case Severity.Critical:
        return 'CRITICAL';
      default:
        return 'UNKNOWN';
    }
  };

  if (visibleErrors.length === 0) {
    return null;
  }

  return (
    <div className="error-notifications">
      {visibleErrors.map(error => (
        <div
          key={error.id}
          className="error-notification"
          style={{ borderLeftColor: getSeverityColor(error.severity) }}
        >
          <div className="error-header">
            <span
              className="error-severity"
              style={{ backgroundColor: getSeverityColor(error.severity) }}
            >
              {getSeverityLabel(error.severity)}
            </span>
            <button
              className="error-dismiss"
              onClick={() => dismissError(error.id)}
              aria-label="Dismiss"
            >
              ×
            </button>
          </div>
          <div className="error-message">{error.message}</div>
          {error.code && (
            <div className="error-code">Error code: {error.code}</div>
          )}
        </div>
      ))}
    </div>
  );
}
