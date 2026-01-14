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
 * - Session statistics
 */

export interface ServerInfoPanelProps {
  className?: string;
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

        store.setConnectionStatus(
          infoResponse.connection.status as 'connected' | 'connecting' | 'disconnected' | 'reconnecting'
        );
        store.setLatency(infoResponse.connection.latency);

        store.setModelInfo({
          name: infoResponse.model.name,
          provider: infoResponse.model.provider,
        });

        store.setMCPServers(
          infoResponse.mcpServers.map((s) => ({
            name: s.name,
            status: s.status as 'connected' | 'disconnected' | 'error',
          }))
        );

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

  const getConnectionStatusText = () => {
    switch (connectionStatus) {
      case 'connected': return 'Connected';
      case 'connecting': return 'Connecting';
      case 'reconnecting': return 'Reconnecting';
      case 'disconnected': return 'Disconnected';
      default: return 'Unknown';
    }
  };

  const getStatusClasses = (status: string) => {
    switch (status) {
      case 'connected':
        return 'bg-success/10 text-success';
      case 'connecting':
      case 'reconnecting':
        return 'bg-warning/10 text-warning';
      case 'disconnected':
      case 'error':
        return 'bg-destructive/10 text-destructive';
      default:
        return 'bg-muted text-muted-foreground';
    }
  };

  const getQualityClasses = (quality: ConnectionQuality) => {
    switch (quality) {
      case 'excellent': return 'text-success';
      case 'good': return 'text-primary';
      case 'fair': return 'text-warning';
      case 'poor': return 'text-destructive';
      default: return 'text-muted-foreground';
    }
  };

  if (isLoading) {
    return (
      <div className={cls('stack-4', className)}>
        <div className="panel flex-center gap-2">
          <div className="w-4 h-4 border-2 border-primary border-t-transparent rounded-full animate-spin" />
          <span className="text-sm text-muted-foreground">Loading...</span>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={cls('stack-4', className)}>
        <div className="panel border border-destructive/20">
          <p className="text-sm text-destructive">{error}</p>
          <button
            onClick={() => window.location.reload()}
            className="text-xs text-primary hover:underline mt-2"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className={cls(compact ? 'stack-2' : 'stack-4', className)}>
      {/* Connection */}
      <div className="panel">
        <div className="panel-header">Connection</div>
        <div className="row-2">
          <span className={cls('badge', getStatusClasses(connectionStatus))}>
            {getConnectionStatusText()}
          </span>
          {isConnected && (
            <>
              <span className="text-muted-foreground">·</span>
              <span className={cls('text-sm', getQualityClasses(connectionQuality))}>
                {latency}ms ({connectionQuality})
              </span>
            </>
          )}
          {isConnecting && (
            <div className="w-3 h-3 border-2 border-warning border-t-transparent rounded-full animate-spin" />
          )}
        </div>
      </div>

      {/* Model */}
      {modelInfo && (
        <div className="panel">
          <div className="panel-header">Model</div>
          <div className="text-sm font-medium">{modelInfo.name}</div>
          <div className="text-xs text-muted-foreground">Provider: {modelInfo.provider}</div>
        </div>
      )}

      {/* MCP Servers */}
      {mcpServers.length > 0 && (
        <div className="panel">
          <div className="flex-between mb-2">
            <div className="panel-header mb-0">MCP Servers</div>
            <span className="text-xs text-muted-foreground">
              {mcpServerSummary.connected}/{mcpServerSummary.total}
            </span>
          </div>
          <div className={compact ? 'stack' : 'stack-2'}>
            {mcpServers.map((server) => (
              <div key={server.name} className="flex-between gap-2">
                <span className="text-sm truncate flex-1">{server.name}</span>
                <span className={cls('badge text-xs', getStatusClasses(server.status))}>
                  {server.status}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Session Stats */}
      <div className="panel">
        <div className="panel-header">Session</div>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <div className="text-xs text-muted-foreground">Messages</div>
            <div className="text-sm font-medium">{sessionStats.messageCount}</div>
          </div>
          <div>
            <div className="text-xs text-muted-foreground">Tool Calls</div>
            <div className="text-sm font-medium">{sessionStats.toolCallCount}</div>
          </div>
          <div>
            <div className="text-xs text-muted-foreground">Memories</div>
            <div className="text-sm font-medium">{sessionStats.memoriesUsed}</div>
          </div>
          <div>
            <div className="text-xs text-muted-foreground">Duration</div>
            <div className="text-sm font-medium">{formattedSessionDuration}</div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ServerInfoPanel;
