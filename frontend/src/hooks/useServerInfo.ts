import { useMemo } from 'react';
import {
  useServerInfoStore,
  selectConnectionStatus,
  selectLatency,
  selectModelInfo,
  selectMCPServers,
  selectSessionStats,
} from '../stores/serverInfoStore';

export type ConnectionQuality = 'excellent' | 'good' | 'fair' | 'poor';

/**
 * Hook for accessing server information and connection state.
 * Provides server status, model info, MCP servers, and session statistics.
 *
 * @returns Object with server info state and computed properties
 *
 * @example
 * ```tsx
 * function ServerInfoPanel() {
 *   const {
 *     isConnected,
 *     connectionQuality,
 *     latency,
 *     modelInfo,
 *     mcpServers,
 *     sessionStats
 *   } = useServerInfo();
 *
 *   return (
 *     <div>
 *       <div>Status: {isConnected ? 'Connected' : 'Disconnected'}</div>
 *       <div>Latency: {latency}ms ({connectionQuality})</div>
 *       <div>Model: {modelInfo?.name}</div>
 *       <div>Messages: {sessionStats.messageCount}</div>
 *     </div>
 *   );
 * }
 * ```
 */
export function useServerInfo() {
  // Get store state
  const connectionStatus = useServerInfoStore(selectConnectionStatus);
  const latency = useServerInfoStore(selectLatency);
  const modelInfo = useServerInfoStore(selectModelInfo);
  const mcpServers = useServerInfoStore(selectMCPServers);
  const sessionStats = useServerInfoStore(selectSessionStats);

  // Compute filtered servers to avoid subscribing to filter functions that return new arrays
  const connectedMCPServers = useMemo(
    () => mcpServers.filter((s) => s.status === 'connected'),
    [mcpServers]
  );

  const disconnectedMCPServers = useMemo(
    () => mcpServers.filter((s) => s.status === 'disconnected' || s.status === 'error'),
    [mcpServers]
  );

  // Computed: is connected
  const isConnected = useMemo(
    () => connectionStatus === 'connected',
    [connectionStatus]
  );

  // Computed: is connecting
  const isConnecting = useMemo(
    () => connectionStatus === 'connecting' || connectionStatus === 'reconnecting',
    [connectionStatus]
  );

  // Computed: connection quality based on latency
  const connectionQuality: ConnectionQuality = useMemo(() => {
    if (!isConnected) return 'poor';
    if (latency < 50) return 'excellent';
    if (latency < 100) return 'good';
    if (latency < 200) return 'fair';
    return 'poor';
  }, [isConnected, latency]);

  // Computed: MCP server summary
  const mcpServerSummary = useMemo(
    () => ({
      total: mcpServers.length,
      connected: connectedMCPServers.length,
      disconnected: disconnectedMCPServers.length,
    }),
    [mcpServers.length, connectedMCPServers.length, disconnectedMCPServers.length]
  );

  // Computed: formatted session duration
  const formattedSessionDuration = useMemo(() => {
    const { sessionDuration } = sessionStats;
    if (sessionDuration < 60) {
      return `${sessionDuration}s`;
    }
    const minutes = Math.floor(sessionDuration / 60);
    const seconds = sessionDuration % 60;
    if (minutes < 60) {
      return `${minutes}m ${seconds}s`;
    }
    const hours = Math.floor(minutes / 60);
    const remainingMinutes = minutes % 60;
    return `${hours}h ${remainingMinutes}m`;
  }, [sessionStats]);

  return {
    // Raw state
    connectionStatus,
    latency,
    modelInfo,
    mcpServers,
    sessionStats,

    // Computed properties
    isConnected,
    isConnecting,
    connectionQuality,
    mcpServerSummary,
    formattedSessionDuration,

    // Filtered lists
    connectedMCPServers,
    disconnectedMCPServers,
  };
}

/**
 * Hook for accessing only connection status and quality.
 * Lighter alternative to useServerInfo when only connection info is needed.
 *
 * @example
 * ```tsx
 * function ConnectionIndicator() {
 *   const { isConnected, connectionQuality, latency } = useConnectionStatus();
 *
 *   return (
 *     <div className={isConnected ? 'connected' : 'disconnected'}>
 *       {latency}ms ({connectionQuality})
 *     </div>
 *   );
 * }
 * ```
 */
export function useConnectionStatus() {
  const connectionStatus = useServerInfoStore(selectConnectionStatus);
  const latency = useServerInfoStore(selectLatency);

  const isConnected = useMemo(
    () => connectionStatus === 'connected',
    [connectionStatus]
  );

  const isConnecting = useMemo(
    () => connectionStatus === 'connecting' || connectionStatus === 'reconnecting',
    [connectionStatus]
  );

  const connectionQuality: ConnectionQuality = useMemo(() => {
    if (!isConnected) return 'poor';
    if (latency < 50) return 'excellent';
    if (latency < 100) return 'good';
    if (latency < 200) return 'fair';
    return 'poor';
  }, [isConnected, latency]);

  return {
    connectionStatus,
    latency,
    isConnected,
    isConnecting,
    connectionQuality,
  };
}

/**
 * Hook for accessing only session statistics.
 * Lighter alternative to useServerInfo when only stats are needed.
 *
 * @example
 * ```tsx
 * function SessionStatsDisplay() {
 *   const { messageCount, toolCallCount, formattedDuration } = useSessionStats();
 *
 *   return (
 *     <div>
 *       <div>Messages: {messageCount}</div>
 *       <div>Tool Calls: {toolCallCount}</div>
 *       <div>Duration: {formattedDuration}</div>
 *     </div>
 *   );
 * }
 * ```
 */
export function useSessionStats() {
  const sessionStats = useServerInfoStore(selectSessionStats);

  const formattedDuration = useMemo(() => {
    const { sessionDuration } = sessionStats;
    if (sessionDuration < 60) {
      return `${sessionDuration}s`;
    }
    const minutes = Math.floor(sessionDuration / 60);
    const seconds = sessionDuration % 60;
    if (minutes < 60) {
      return `${minutes}m ${seconds}s`;
    }
    const hours = Math.floor(minutes / 60);
    const remainingMinutes = minutes % 60;
    return `${hours}h ${remainingMinutes}m`;
  }, [sessionStats]);

  return {
    ...sessionStats,
    formattedDuration,
  };
}
