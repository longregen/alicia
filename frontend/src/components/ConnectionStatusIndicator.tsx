import { useConnectionStore, ConnectionStatus } from '../stores/connectionStore';
import { cn } from '../lib/utils';

interface ConnectionStatusIndicatorProps {
  isCollapsed: boolean;
}

export function ConnectionStatusIndicator({ isCollapsed }: ConnectionStatusIndicatorProps) {
  const status = useConnectionStore((state) => state.status);
  const error = useConnectionStore((state) => state.error);

  const getStatusColor = () => {
    switch (status) {
      case ConnectionStatus.Connected:
        return 'bg-green-500';
      case ConnectionStatus.Connecting:
      case ConnectionStatus.Reconnecting:
        return 'bg-yellow-500';
      case ConnectionStatus.Disconnected:
      case ConnectionStatus.Error:
        return 'bg-red-500';
      default:
        return 'bg-gray-500';
    }
  };

  const getStatusText = () => {
    switch (status) {
      case ConnectionStatus.Connected:
        return 'Connected';
      case ConnectionStatus.Connecting:
        return 'Connecting...';
      case ConnectionStatus.Reconnecting:
        return 'Reconnecting...';
      case ConnectionStatus.Disconnected:
        return 'Disconnected';
      case ConnectionStatus.Error:
        return 'Error';
      default:
        return 'Unknown';
    }
  };

  const isAnimated = status === ConnectionStatus.Connecting || status === ConnectionStatus.Reconnecting;
  const showError = status === ConnectionStatus.Error && error;

  return (
    <div
      className={cn(
        'layout-center-gap p-2 rounded transition-colors',
        isCollapsed && 'justify-center'
      )}
      title={isCollapsed ? getStatusText() : showError ? error : undefined}
      data-testid="connection-status-indicator"
    >
      <div className="relative">
        <div
          className={cn(
            'w-2 h-2 rounded-full',
            getStatusColor(),
            isAnimated && 'animate-pulse'
          )}
          data-status={status}
        />
      </div>
      {!isCollapsed && (
        <span className="text-xs text-muted-foreground">{getStatusText()}</span>
      )}
    </div>
  );
}
