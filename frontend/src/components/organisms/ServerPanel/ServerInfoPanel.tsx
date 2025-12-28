import React, { useEffect, useState } from 'react';
import { cls } from '../../../utils/cls';
import { CSS } from '../../../utils/constants';
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
    if (isConnected) return 'text-green-600 dark:text-green-400';
    if (isConnecting) return 'text-yellow-600 dark:text-yellow-400';
    return 'text-red-600 dark:text-red-400';
  };

  const getConnectionStatusBg = () => {
    if (isConnected) return 'bg-green-100 dark:bg-green-900/30';
    if (isConnecting) return 'bg-yellow-100 dark:bg-yellow-900/30';
    return 'bg-red-100 dark:bg-red-900/30';
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
        return 'text-green-600 dark:text-green-400';
      case 'good':
        return 'text-blue-600 dark:text-blue-400';
      case 'fair':
        return 'text-yellow-600 dark:text-yellow-400';
      case 'poor':
        return 'text-red-600 dark:text-red-400';
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

  const getMCPStatusBg = (status: string) => {
    switch (status) {
      case 'connected':
        return 'bg-green-100 dark:bg-green-900/30';
      case 'disconnected':
        return 'bg-gray-100 dark:bg-gray-900/30';
      case 'error':
        return 'bg-red-100 dark:bg-red-900/30';
      default:
        return 'bg-gray-100 dark:bg-gray-900/30';
    }
  };

  const sectionClass = cls(
    CSS.p3,
    CSS.rounded,
    'bg-surface-bg',
    'border border-gray-200 dark:border-gray-700',
    compact ? CSS.spaceY2 : CSS.spaceY3
  );

  const labelClass = cls(CSS.textXs, CSS.fontMedium, CSS.textMuted, 'uppercase tracking-wide');

  const valueClass = cls(CSS.textSm, CSS.textPrimary);

  // Show loading state
  if (isLoading) {
    return (
      <div className={cls(compact ? CSS.spaceY3 : CSS.spaceY4, className)}>
        <div className={cls(sectionClass, CSS.flex, CSS.itemsCenter, CSS.justifyCenter)}>
          <div className="w-5 h-5 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
          <span className={cls(CSS.textSm, CSS.textMuted, 'ml-2')}>Loading server info...</span>
        </div>
      </div>
    );
  }

  // Show error state
  if (error) {
    return (
      <div className={cls(compact ? CSS.spaceY3 : CSS.spaceY4, className)}>
        <div className={cls(sectionClass, 'border-red-300 dark:border-red-700')}>
          <div className={cls(CSS.textSm, 'text-red-600 dark:text-red-400')}>
            {error}
          </div>
          <button
            onClick={() => window.location.reload()}
            className={cls(CSS.textXs, 'text-blue-600 dark:text-blue-400 hover:underline mt-2')}
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className={cls(compact ? CSS.spaceY3 : CSS.spaceY4, className)}>
      {/* Connection Status Section */}
      <div className={sectionClass}>
        <div className={labelClass}>Connection</div>
        <div className={cls(CSS.flex, CSS.itemsCenter, CSS.gap2, CSS.flexRow)}>
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
              <div className={cls('w-1 h-1 rounded-full bg-gray-300 dark:bg-gray-600')} />
              <div className={cls(valueClass, getQualityColor(connectionQuality))}>
                {latency}ms ({connectionQuality})
              </div>
            </>
          )}
          {isConnecting && (
            <div className="w-3 h-3 border-2 border-yellow-500 border-t-transparent rounded-full animate-spin" />
          )}
        </div>
      </div>

      {/* Model Info Section */}
      {modelInfo && (
        <div className={sectionClass}>
          <div className={labelClass}>Model</div>
          <div className={valueClass}>
            <div className={CSS.fontMedium}>{modelInfo.name}</div>
            <div className={cls(CSS.textXs, CSS.textMuted)}>Provider: {modelInfo.provider}</div>
          </div>
        </div>
      )}

      {/* MCP Servers Section */}
      {mcpServers.length > 0 && (
        <div className={sectionClass}>
          <div className={cls(CSS.flex, CSS.justifyBetween, CSS.itemsCenter, CSS.mb2)}>
            <div className={labelClass}>MCP Servers</div>
            <div className={cls(CSS.textXs, CSS.textMuted)}>
              {mcpServerSummary.connected}/{mcpServerSummary.total} connected
            </div>
          </div>
          <div className={cls(compact ? 'space-y-1' : CSS.spaceY2)}>
            {mcpServers.map((server) => (
              <div
                key={server.name}
                className={cls(CSS.flex, CSS.justifyBetween, CSS.itemsCenter, CSS.gap2)}
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
            <div className={cls(CSS.textXs, CSS.textMuted)}>Messages</div>
            <div className={cls(valueClass, CSS.fontMedium)}>{sessionStats.messageCount}</div>
          </div>
          <div>
            <div className={cls(CSS.textXs, CSS.textMuted)}>Tool Calls</div>
            <div className={cls(valueClass, CSS.fontMedium)}>{sessionStats.toolCallCount}</div>
          </div>
          <div>
            <div className={cls(CSS.textXs, CSS.textMuted)}>Memories Used</div>
            <div className={cls(valueClass, CSS.fontMedium)}>{sessionStats.memoriesUsed}</div>
          </div>
          <div>
            <div className={cls(CSS.textXs, CSS.textMuted)}>Duration</div>
            <div className={cls(valueClass, CSS.fontMedium)}>{formattedSessionDuration}</div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ServerInfoPanel;
