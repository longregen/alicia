import { useConnectionStore, ConnectionStatus } from '../stores/connectionStore';
import { cls } from '../utils/cls';

interface ConnectionStatusIndicatorProps {
  isCollapsed: boolean;
}

export function ConnectionStatusIndicator({ isCollapsed }: ConnectionStatusIndicatorProps) {
  const status = useConnectionStore((state) => state.status);
  const error = useConnectionStore((state) => state.error);

  const getStatusColor = () => {
    switch (status) {
      case ConnectionStatus.Connected:
        return 'bg-success';
      case ConnectionStatus.Connecting:
      case ConnectionStatus.Reconnecting:
        return 'bg-warning';
      case ConnectionStatus.Disconnected:
      case ConnectionStatus.Error:
        return 'bg-destructive';
      default:
        return 'bg-muted-foreground';
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
      className={cls(
        'row-2 p-2 rounded transition-colors',
        isCollapsed && 'justify-center'
      )}
      title={isCollapsed ? getStatusText() : showError ? error : undefined}
      data-testid="connection-status-indicator"
    >
      <div
        className={cls(
          'status-dot',
          getStatusColor(),
          isAnimated && 'animate-pulse'
        )}
        data-status={status}
      />
      {!isCollapsed && (
        <span className="text-xs text-muted-foreground">{getStatusText()}</span>
      )}
    </div>
  );
}
