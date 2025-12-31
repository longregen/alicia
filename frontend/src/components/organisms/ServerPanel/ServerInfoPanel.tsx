import React, { useEffect, useState } from 'react';
import { cls } from '../../../utils/cls';
import { useServerInfo } from '../../../hooks/useServerInfo';
import type { ConnectionQuality } from '../../../hooks/useServerInfo';
import { api } from '../../../services/api';
import { useServerInfoStore } from '../../../stores/serverInfoStore';

/**
 * ServerInfoPanel organism component.
 *
 * Displays comprehensive server information including:
 * - Connection status with latency and quality indicator
 * - Model information (name and provider)
 * - MCP server statuses
 * - Session statistics (message count, tool calls, memories used, duration)
 *
 * Uses the useServerInfo hook for reactive state management.
 */

export interface ServerInfoPanelProps {
  /** Additional CSS classes */
  className?: string;
  /** Compact mode for reduced spacing */
  compact?: boolean;
}

const ServerInfoPanel: React.FC<ServerInfoPanelProps> = ({
  className = '',
  compact = false,
}) => {
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const {
    connectionStatus,
    latency,
    isConnected,
    isConnecting,
    connectionQuality,
    modelInfo,
    mcpServers,
    sessionStats,
    mcpServerSummary,
    formattedSessionDuration,
  } = useServerInfo();

  // Fetch server info on mount
  useEffect(() => {
    const fetchServerInfo = async () => {
      try {
        setIsLoading(true);
        setError(null);

        const [infoResponse, statsResponse] = await Promise.all([
          api.getServerInfo(),
          api.getGlobalStats(),
        ]);

        const store = useServerInfoStore.getState();

        // Update connection info
        store.setConnectionStatus(
          infoResponse.connection.status as 'connected' | 'connecting' | 'disconnected' | 'reconnecting'
        );
        store.setLatency(infoResponse.connection.latency);

        // Update model info
        store.setModelInfo({
          name: infoResponse.model.name,
          provider: infoResponse.model.provider,
        });

        // Update MCP servers
        store.setMCPServers(
          infoResponse.mcpServers.map((s) => ({
            name: s.name,
            status: s.status as 'connected' | 'disconnected' | 'error',
          }))
        );

        // Update session stats
        store.setSessionStats({
          messageCount: statsResponse.messageCount,
          toolCallCount: statsResponse.toolCallCount,
          memoriesUsed: statsResponse.memoriesUsed,
          sessionDuration: statsResponse.sessionDuration,
        });
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch server info');
      } finally {
        setIsLoading(false);
      }
    };

    fetchServerInfo();
  }, []);

  const getConnectionStatusColor = () => {
    if (isConnected) return 'text-success';
    if (isConnecting) return 'text-warning';
    return 'text-error';
  };

  const getConnectionStatusBg = () => {
    if (isConnected) return 'bg-success-subtle';
    if (isConnecting) return 'bg-warning-subtle';
    return 'bg-error-subtle';
  };

  const getConnectionStatusText = () => {
    switch (connectionStatus) {
      case 'connected':
        return 'Connected';
      case 'connecting':
        return 'Connecting...';
      case 'reconnecting':
        return 'Reconnecting...';
      case 'disconnected':
        return 'Disconnected';
      default:
        return 'Unknown';
    }
  };

  const getQualityColor = (quality: ConnectionQuality) => {
    switch (quality) {
      case 'excellent':
        return 'text-success';
      case 'good':
        return 'text-accent';
      case 'fair':
        return 'text-warning';
      case 'poor':
        return 'text-error';
    }
  };

  const getMCPStatusColor = (status: string) => {
    switch (status) {
      case 'connected':
        return 'text-success';
      case 'disconnected':
        return 'text-muted';
      case 'error':
        return 'text-error';
      default:
        return 'text-muted';
    }
  };

  const getMCPStatusBg = (status: string) => {
    switch (status) {
      case 'connected':
        return 'bg-success-subtle';
      case 'disconnected':
        return 'bg-surface';
      case 'error':
        return 'bg-error-subtle';
      default:
        return 'bg-surface';
    }
  };

  const sectionClass = cls(
    'p-3',
    'rounded',
    'bg-surface',
    'border',
    compact ? 'space-y-2' : 'space-y-3'
  );

  const labelClass = cls('text-xs', 'font-medium', 'text-muted', 'uppercase tracking-wide');

  const valueClass = cls('text-sm', 'text-default');

  // Show loading state
  if (isLoading) {
    return (
      <div className={cls(compact ? 'space-y-3' : 'space-y-4', className)}>
        <div className={cls(sectionClass, 'flex', 'items-center', 'justify-center')}>
          <div className="w-5 h-5 border-2 border-accent border-t-transparent rounded-full animate-spin" />
          <span className={cls('text-sm', 'text-muted', 'ml-2')}>Loading server info...</span>
        </div>
      </div>
    );
  }

  // Show error state
  if (error) {
    return (
      <div className={cls(compact ? 'space-y-3' : 'space-y-4', className)}>
        <div className={cls(sectionClass, 'border-error')}>
          <div className={cls('text-sm', 'text-error')}>
            {error}
          </div>
          <button
            onClick={() => window.location.reload()}
            className={cls('text-xs', 'text-accent hover:underline mt-2')}
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className={cls(compact ? 'space-y-3' : 'space-y-4', className)}>
      {/* Connection Status Section */}
      <div className={sectionClass}>
        <div className={labelClass}>Connection</div>
        <div className={cls('flex', 'items-center', 'gap-2', 'flex-row')}>
          <div
            className={cls(
              'px-2 py-1 rounded-full text-xs font-medium',
              getConnectionStatusBg(),
              getConnectionStatusColor()
            )}
          >
            {getConnectionStatusText()}
          </div>
          {isConnected && (
            <>
              <div className={cls('w-1 h-1 rounded-full bg-border')} />
              <div className={cls(valueClass, getQualityColor(connectionQuality))}>
                {latency}ms ({connectionQuality})
              </div>
            </>
          )}
          {isConnecting && (
            <div className="w-3 h-3 border-2 border-warning border-t-transparent rounded-full animate-spin" />
          )}
        </div>
      </div>

      {/* Model Info Section */}
      {modelInfo && (
        <div className={sectionClass}>
          <div className={labelClass}>Model</div>
          <div className={valueClass}>
            <div className="font-medium">{modelInfo.name}</div>
            <div className={cls('text-xs', 'text-muted')}>Provider: {modelInfo.provider}</div>
          </div>
        </div>
      )}

      {/* MCP Servers Section */}
      {mcpServers.length > 0 && (
        <div className={sectionClass}>
          <div className={cls('flex', 'justify-between', 'items-center', 'mb-2')}>
            <div className={labelClass}>MCP Servers</div>
            <div className={cls('text-xs', 'text-muted')}>
              {mcpServerSummary.connected}/{mcpServerSummary.total} connected
            </div>
          </div>
          <div className={cls(compact ? 'space-y-1' : 'space-y-2')}>
            {mcpServers.map((server) => (
              <div
                key={server.name}
                className={cls('flex', 'justify-between', 'items-center', 'gap-2')}
              >
                <div className={cls(valueClass, 'truncate flex-1')}>{server.name}</div>
                <div
                  className={cls(
                    'px-2 py-0.5 rounded-full text-xs font-medium whitespace-nowrap',
                    getMCPStatusBg(server.status),
                    getMCPStatusColor(server.status)
                  )}
                >
                  {server.status}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Session Statistics Section */}
      <div className={sectionClass}>
        <div className={labelClass}>Session Statistics</div>
        <div className={cls('grid grid-cols-2 gap-3', compact ? 'gap-2' : '')}>
          <div>
            <div className={cls('text-xs', 'text-muted')}>Messages</div>
            <div className={cls(valueClass, 'font-medium')}>{sessionStats.messageCount}</div>
          </div>
          <div>
            <div className={cls('text-xs', 'text-muted')}>Tool Calls</div>
            <div className={cls(valueClass, 'font-medium')}>{sessionStats.toolCallCount}</div>
          </div>
          <div>
            <div className={cls('text-xs', 'text-muted')}>Memories Used</div>
            <div className={cls(valueClass, 'font-medium')}>{sessionStats.memoriesUsed}</div>
          </div>
          <div>
            <div className={cls('text-xs', 'text-muted')}>Duration</div>
            <div className={cls(valueClass, 'font-medium')}>{formattedSessionDuration}</div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ServerInfoPanel;
