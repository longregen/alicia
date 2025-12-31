import React, { useEffect, useState } from 'react';
import { cn } from '../../../lib/utils';
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
 * Follows the new design pattern with header bar and card-based layout.
 */

export interface ServerInfoPanelProps {
  /** Additional CSS classes */
  className?: string;
  /** Compact mode for reduced spacing */
  compact?: boolean;
}

// Style constants
const sectionClass = 'p-3 rounded-lg bg-surface border border-border';
const labelClass = 'text-xs font-medium text-muted mb-1';
const valueClass = 'text-sm text-foreground';

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

  // Helper functions for connection status
  const getConnectionStatusText = () => {
    switch (connectionStatus) {
      case 'connected':
        return 'Connected';
      case 'connecting':
        return 'Connecting';
      case 'reconnecting':
        return 'Reconnecting';
      case 'disconnected':
        return 'Disconnected';
      default:
        return 'Unknown';
    }
  };

  const getConnectionStatusBg = () => {
    switch (connectionStatus) {
      case 'connected':
        return 'bg-green-500/10';
      case 'connecting':
      case 'reconnecting':
        return 'bg-yellow-500/10';
      case 'disconnected':
        return 'bg-red-500/10';
      default:
        return 'bg-gray-500/10';
    }
  };

  const getConnectionStatusColor = () => {
    switch (connectionStatus) {
      case 'connected':
        return 'text-green-600 dark:text-green-400';
      case 'connecting':
      case 'reconnecting':
        return 'text-yellow-600 dark:text-yellow-400';
      case 'disconnected':
        return 'text-red-600 dark:text-red-400';
      default:
        return 'text-gray-600 dark:text-gray-400';
    }
  };

  const getQualityColor = (quality: ConnectionQuality) => {
    switch (quality) {
      case 'excellent':
        return 'text-green-600 dark:text-green-400';
      case 'good':
        return 'text-blue-600 dark:text-blue-400';
      case 'fair':
        return 'text-yellow-600 dark:text-yellow-400';
      case 'poor':
        return 'text-red-600 dark:text-red-400';
      default:
        return 'text-gray-600 dark:text-gray-400';
    }
  };

  // Helper functions for MCP server status
  const getMCPStatusBg = (status: string) => {
    switch (status) {
      case 'connected':
        return 'bg-green-500/10';
      case 'disconnected':
        return 'bg-gray-500/10';
      case 'error':
        return 'bg-red-500/10';
      default:
        return 'bg-gray-500/10';
    }
  };

  const getMCPStatusColor = (status: string) => {
    switch (status) {
      case 'connected':
        return 'text-green-600 dark:text-green-400';
      case 'disconnected':
        return 'text-gray-600 dark:text-gray-400';
      case 'error':
        return 'text-red-600 dark:text-red-400';
      default:
        return 'text-gray-600 dark:text-gray-400';
    }
  };

  // Show loading state
  if (isLoading) {
    return (
      <div className={cn(compact ? 'space-y-3' : 'space-y-4', className)}>
        <div className={cn(sectionClass, 'flex', 'items-center', 'justify-center')}>
          <div className="w-5 h-5 border-2 border-accent border-t-transparent rounded-full animate-spin" />
          <span className={cn('text-sm', 'text-muted', 'ml-2')}>Loading server info...</span>
        </div>
      </div>
    );
  }

  // Show error state
  if (error) {
    return (
      <div className={cn(compact ? 'space-y-3' : 'space-y-4', className)}>
        <div className={cn(sectionClass, 'border-error')}>
          <div className={cn('text-sm', 'text-error')}>
            {error}
          </div>
          <button
            onClick={() => window.location.reload()}
            className={cn('text-xs', 'text-accent hover:underline mt-2')}
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className={cn(compact ? 'space-y-3' : 'space-y-4', className)}>
      {/* Connection Status Section */}
      <div className={sectionClass}>
        <div className={labelClass}>Connection</div>
        <div className={cn('flex', 'items-center', 'gap-2', 'flex-row')}>
          <div
            className={cn(
              'px-2 py-1 rounded-full text-xs font-medium',
              getConnectionStatusBg(),
              getConnectionStatusColor()
            )}
          >
            {getConnectionStatusText()}
          </div>
          {isConnected && (
            <>
              <div className={cn('w-1 h-1 rounded-full bg-border')} />
              <div className={cn(valueClass, getQualityColor(connectionQuality))}>
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
            <div className={cn('text-xs', 'text-muted')}>Provider: {modelInfo.provider}</div>
          </div>
        </div>
      )}

      {/* MCP Servers Section */}
      {mcpServers.length > 0 && (
        <div className={sectionClass}>
          <div className={cn('flex', 'justify-between', 'items-center', 'mb-2')}>
            <div className={labelClass}>MCP Servers</div>
            <div className={cn('text-xs', 'text-muted')}>
              {mcpServerSummary.connected}/{mcpServerSummary.total} connected
            </div>
          </div>
          <div className={cn(compact ? 'space-y-1' : 'space-y-2')}>
            {mcpServers.map((server) => (
              <div
                key={server.name}
                className={cn('flex', 'justify-between', 'items-center', 'gap-2')}
              >
                <div className={cn(valueClass, 'truncate flex-1')}>{server.name}</div>
                <div
                  className={cn(
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
        <div className={cn('grid grid-cols-2 gap-3', compact ? 'gap-2' : '')}>
          <div>
            <div className={cn('text-xs', 'text-muted')}>Messages</div>
            <div className={cn(valueClass, 'font-medium')}>{sessionStats.messageCount}</div>
          </div>
          <div>
            <div className={cn('text-xs', 'text-muted')}>Tool Calls</div>
            <div className={cn(valueClass, 'font-medium')}>{sessionStats.toolCallCount}</div>
          </div>
          <div>
            <div className={cn('text-xs', 'text-muted')}>Memories Used</div>
            <div className={cn(valueClass, 'font-medium')}>{sessionStats.memoriesUsed}</div>
          </div>
          <div>
            <div className={cn('text-xs', 'text-muted')}>Duration</div>
            <div className={cn(valueClass, 'font-medium')}>{formattedSessionDuration}</div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ServerInfoPanel;
