import { describe, it, expect, beforeEach } from 'vitest';
import {
  useServerInfoStore,
  selectConnectionStatus,
  selectLatency,
  selectModelInfo,
  selectMCPServers,
  selectSessionStats,
  selectConnectedMCPServers,
  selectDisconnectedMCPServers,
  type ConnectionStatus,
  type MCPServerStatus,
  type ModelInfo,
  type MCPServerInfo,
} from './serverInfoStore';

describe('serverInfoStore', () => {
  beforeEach(() => {
    useServerInfoStore.getState().resetServerInfo();
  });

  describe('initial state', () => {
    it('should have correct initial state', () => {
      const state = useServerInfoStore.getState();
      expect(state.connectionStatus).toBe('disconnected');
      expect(state.latency).toBe(0);
      expect(state.modelInfo).toBeNull();
      expect(state.mcpServers).toEqual([]);
      expect(state.sessionStats).toEqual({
        messageCount: 0,
        toolCallCount: 0,
        memoriesUsed: 0,
        sessionDuration: 0,
      });
    });
  });

  describe('setConnectionStatus', () => {
    it('should update connection status', () => {
      useServerInfoStore.getState().setConnectionStatus('connecting');

      const state = useServerInfoStore.getState();
      expect(state.connectionStatus).toBe('connecting');
    });

    it('should handle all connection statuses', () => {
      const statuses: ConnectionStatus[] = [
        'disconnected',
        'connecting',
        'connected',
        'reconnecting',
      ];

      statuses.forEach((status) => {
        useServerInfoStore.getState().setConnectionStatus(status);
        const state = useServerInfoStore.getState();
        expect(state.connectionStatus).toBe(status);
      });
    });
  });

  describe('setLatency', () => {
    it('should update latency value', () => {
      useServerInfoStore.getState().setLatency(150);

      const state = useServerInfoStore.getState();
      expect(state.latency).toBe(150);
    });

    it('should handle zero latency', () => {
      useServerInfoStore.getState().setLatency(100);
      useServerInfoStore.getState().setLatency(0);

      const state = useServerInfoStore.getState();
      expect(state.latency).toBe(0);
    });

    it('should handle high latency values', () => {
      useServerInfoStore.getState().setLatency(5000);

      const state = useServerInfoStore.getState();
      expect(state.latency).toBe(5000);
    });
  });

  describe('setModelInfo', () => {
    it('should set model information', () => {
      const modelInfo: ModelInfo = {
        name: 'claude-3-opus',
        provider: 'anthropic',
      };

      useServerInfoStore.getState().setModelInfo(modelInfo);

      const state = useServerInfoStore.getState();
      expect(state.modelInfo).toEqual(modelInfo);
    });

    it('should overwrite existing model info', () => {
      const modelInfo1: ModelInfo = {
        name: 'gpt-4',
        provider: 'openai',
      };

      const modelInfo2: ModelInfo = {
        name: 'claude-3-opus',
        provider: 'anthropic',
      };

      useServerInfoStore.getState().setModelInfo(modelInfo1);
      useServerInfoStore.getState().setModelInfo(modelInfo2);

      const state = useServerInfoStore.getState();
      expect(state.modelInfo).toEqual(modelInfo2);
    });
  });

  describe('setMCPServers', () => {
    it('should set MCP servers array', () => {
      const servers: MCPServerInfo[] = [
        { name: 'server-1', status: 'connected' },
        { name: 'server-2', status: 'disconnected' },
      ];

      useServerInfoStore.getState().setMCPServers(servers);

      const state = useServerInfoStore.getState();
      expect(state.mcpServers).toEqual(servers);
    });

    it('should overwrite existing servers', () => {
      const servers1: MCPServerInfo[] = [{ name: 'server-1', status: 'connected' }];

      const servers2: MCPServerInfo[] = [
        { name: 'server-2', status: 'connected' },
        { name: 'server-3', status: 'disconnected' },
      ];

      useServerInfoStore.getState().setMCPServers(servers1);
      useServerInfoStore.getState().setMCPServers(servers2);

      const state = useServerInfoStore.getState();
      expect(state.mcpServers).toEqual(servers2);
      expect(state.mcpServers).toHaveLength(2);
    });

    it('should handle empty servers array', () => {
      useServerInfoStore.getState().setMCPServers([
        { name: 'server-1', status: 'connected' },
      ]);
      useServerInfoStore.getState().setMCPServers([]);

      const state = useServerInfoStore.getState();
      expect(state.mcpServers).toEqual([]);
    });
  });

  describe('updateMCPServer', () => {
    it('should update existing server status', () => {
      const servers: MCPServerInfo[] = [
        { name: 'server-1', status: 'disconnected' },
        { name: 'server-2', status: 'connected' },
      ];

      useServerInfoStore.getState().setMCPServers(servers);
      useServerInfoStore.getState().updateMCPServer('server-1', 'connected');

      const state = useServerInfoStore.getState();
      const server1 = state.mcpServers.find((s) => s.name === 'server-1');
      expect(server1?.status).toBe('connected');
    });

    it('should add new server if it does not exist', () => {
      useServerInfoStore.getState().updateMCPServer('new-server', 'connected');

      const state = useServerInfoStore.getState();
      expect(state.mcpServers).toHaveLength(1);
      expect(state.mcpServers[0]).toEqual({
        name: 'new-server',
        status: 'connected',
      });
    });

    it('should handle all MCP server statuses', () => {
      const statuses: MCPServerStatus[] = ['connected', 'disconnected', 'error'];

      statuses.forEach((status, index) => {
        useServerInfoStore.getState().updateMCPServer(`server-${index}`, status);
      });

      const state = useServerInfoStore.getState();
      expect(state.mcpServers).toHaveLength(3);
      expect(state.mcpServers[0].status).toBe('connected');
      expect(state.mcpServers[1].status).toBe('disconnected');
      expect(state.mcpServers[2].status).toBe('error');
    });
  });

  describe('setSessionStats', () => {
    it('should set session stats', () => {
      const stats = {
        messageCount: 10,
        toolCallCount: 5,
        memoriesUsed: 3,
        sessionDuration: 120000,
      };

      useServerInfoStore.getState().setSessionStats(stats);

      const state = useServerInfoStore.getState();
      expect(state.sessionStats).toEqual(stats);
    });

    it('should overwrite existing session stats', () => {
      const stats1 = {
        messageCount: 5,
        toolCallCount: 2,
        memoriesUsed: 1,
        sessionDuration: 60000,
      };

      const stats2 = {
        messageCount: 10,
        toolCallCount: 5,
        memoriesUsed: 3,
        sessionDuration: 120000,
      };

      useServerInfoStore.getState().setSessionStats(stats1);
      useServerInfoStore.getState().setSessionStats(stats2);

      const state = useServerInfoStore.getState();
      expect(state.sessionStats).toEqual(stats2);
    });
  });

  describe('updateSessionStats', () => {
    it('should update specific session stats fields', () => {
      useServerInfoStore.getState().setSessionStats({
        messageCount: 5,
        toolCallCount: 2,
        memoriesUsed: 1,
        sessionDuration: 60000,
      });

      useServerInfoStore.getState().updateSessionStats({
        messageCount: 10,
        toolCallCount: 5,
      });

      const state = useServerInfoStore.getState();
      expect(state.sessionStats.messageCount).toBe(10);
      expect(state.sessionStats.toolCallCount).toBe(5);
      expect(state.sessionStats.memoriesUsed).toBe(1);
      expect(state.sessionStats.sessionDuration).toBe(60000);
    });

    it('should handle partial updates', () => {
      useServerInfoStore.getState().updateSessionStats({
        messageCount: 15,
      });

      const state = useServerInfoStore.getState();
      expect(state.sessionStats.messageCount).toBe(15);
      expect(state.sessionStats.toolCallCount).toBe(0);
    });
  });

  describe('incrementMessageCount', () => {
    it('should increment message count by 1', () => {
      useServerInfoStore.getState().incrementMessageCount();
      useServerInfoStore.getState().incrementMessageCount();

      const state = useServerInfoStore.getState();
      expect(state.sessionStats.messageCount).toBe(2);
    });

    it('should increment from initial zero', () => {
      useServerInfoStore.getState().incrementMessageCount();

      const state = useServerInfoStore.getState();
      expect(state.sessionStats.messageCount).toBe(1);
    });
  });

  describe('incrementToolCallCount', () => {
    it('should increment tool call count by 1', () => {
      useServerInfoStore.getState().incrementToolCallCount();
      useServerInfoStore.getState().incrementToolCallCount();
      useServerInfoStore.getState().incrementToolCallCount();

      const state = useServerInfoStore.getState();
      expect(state.sessionStats.toolCallCount).toBe(3);
    });
  });

  describe('incrementMemoriesUsed', () => {
    it('should increment memories used by 1 by default', () => {
      useServerInfoStore.getState().incrementMemoriesUsed();
      useServerInfoStore.getState().incrementMemoriesUsed();

      const state = useServerInfoStore.getState();
      expect(state.sessionStats.memoriesUsed).toBe(2);
    });

    it('should increment memories used by specified count', () => {
      useServerInfoStore.getState().incrementMemoriesUsed(5);

      const state = useServerInfoStore.getState();
      expect(state.sessionStats.memoriesUsed).toBe(5);
    });

    it('should accumulate custom increments', () => {
      useServerInfoStore.getState().incrementMemoriesUsed(3);
      useServerInfoStore.getState().incrementMemoriesUsed(2);

      const state = useServerInfoStore.getState();
      expect(state.sessionStats.memoriesUsed).toBe(5);
    });
  });

  describe('updateSessionDuration', () => {
    it('should update session duration', () => {
      useServerInfoStore.getState().updateSessionDuration(120000);

      const state = useServerInfoStore.getState();
      expect(state.sessionStats.sessionDuration).toBe(120000);
    });

    it('should overwrite previous duration', () => {
      useServerInfoStore.getState().updateSessionDuration(60000);
      useServerInfoStore.getState().updateSessionDuration(90000);

      const state = useServerInfoStore.getState();
      expect(state.sessionStats.sessionDuration).toBe(90000);
    });
  });

  describe('resetServerInfo', () => {
    it('should reset all state to initial values', () => {
      const modelInfo: ModelInfo = {
        name: 'claude-3-opus',
        provider: 'anthropic',
      };

      const servers: MCPServerInfo[] = [{ name: 'server-1', status: 'connected' }];

      useServerInfoStore.getState().setConnectionStatus('connected');
      useServerInfoStore.getState().setLatency(150);
      useServerInfoStore.getState().setModelInfo(modelInfo);
      useServerInfoStore.getState().setMCPServers(servers);
      useServerInfoStore.getState().incrementMessageCount();
      useServerInfoStore.getState().incrementToolCallCount();

      useServerInfoStore.getState().resetServerInfo();

      const state = useServerInfoStore.getState();
      expect(state.connectionStatus).toBe('disconnected');
      expect(state.latency).toBe(0);
      expect(state.modelInfo).toBeNull();
      expect(state.mcpServers).toEqual([]);
      expect(state.sessionStats).toEqual({
        messageCount: 0,
        toolCallCount: 0,
        memoriesUsed: 0,
        sessionDuration: 0,
      });
    });
  });

  describe('selectConnectionStatus', () => {
    it('should return connection status', () => {
      useServerInfoStore.getState().setConnectionStatus('connected');

      const result = selectConnectionStatus(useServerInfoStore.getState());
      expect(result).toBe('connected');
    });
  });

  describe('selectLatency', () => {
    it('should return latency value', () => {
      useServerInfoStore.getState().setLatency(200);

      const result = selectLatency(useServerInfoStore.getState());
      expect(result).toBe(200);
    });
  });

  describe('selectModelInfo', () => {
    it('should return model info', () => {
      const modelInfo: ModelInfo = {
        name: 'claude-3-opus',
        provider: 'anthropic',
      };

      useServerInfoStore.getState().setModelInfo(modelInfo);

      const result = selectModelInfo(useServerInfoStore.getState());
      expect(result).toEqual(modelInfo);
    });

    it('should return null when no model info set', () => {
      const result = selectModelInfo(useServerInfoStore.getState());
      expect(result).toBeNull();
    });
  });

  describe('selectMCPServers', () => {
    it('should return MCP servers array', () => {
      const servers: MCPServerInfo[] = [
        { name: 'server-1', status: 'connected' },
        { name: 'server-2', status: 'disconnected' },
      ];

      useServerInfoStore.getState().setMCPServers(servers);

      const result = selectMCPServers(useServerInfoStore.getState());
      expect(result).toEqual(servers);
    });
  });

  describe('selectSessionStats', () => {
    it('should return session stats', () => {
      const stats = {
        messageCount: 10,
        toolCallCount: 5,
        memoriesUsed: 3,
        sessionDuration: 120000,
      };

      useServerInfoStore.getState().setSessionStats(stats);

      const result = selectSessionStats(useServerInfoStore.getState());
      expect(result).toEqual(stats);
    });
  });

  describe('selectConnectedMCPServers', () => {
    it('should return only connected servers', () => {
      const servers: MCPServerInfo[] = [
        { name: 'server-1', status: 'connected' },
        { name: 'server-2', status: 'disconnected' },
        { name: 'server-3', status: 'connected' },
        { name: 'server-4', status: 'error' },
      ];

      useServerInfoStore.getState().setMCPServers(servers);

      const result = selectConnectedMCPServers(useServerInfoStore.getState());
      expect(result).toHaveLength(2);
      expect(result.every((s) => s.status === 'connected')).toBe(true);
    });

    it('should return empty array when no servers connected', () => {
      const servers: MCPServerInfo[] = [
        { name: 'server-1', status: 'disconnected' },
        { name: 'server-2', status: 'error' },
      ];

      useServerInfoStore.getState().setMCPServers(servers);

      const result = selectConnectedMCPServers(useServerInfoStore.getState());
      expect(result).toEqual([]);
    });
  });

  describe('selectDisconnectedMCPServers', () => {
    it('should return disconnected and error servers', () => {
      const servers: MCPServerInfo[] = [
        { name: 'server-1', status: 'connected' },
        { name: 'server-2', status: 'disconnected' },
        { name: 'server-3', status: 'error' },
        { name: 'server-4', status: 'disconnected' },
      ];

      useServerInfoStore.getState().setMCPServers(servers);

      const result = selectDisconnectedMCPServers(useServerInfoStore.getState());
      expect(result).toHaveLength(3);
      expect(result.every((s) => s.status === 'disconnected' || s.status === 'error')).toBe(
        true
      );
    });

    it('should return empty array when all servers connected', () => {
      const servers: MCPServerInfo[] = [
        { name: 'server-1', status: 'connected' },
        { name: 'server-2', status: 'connected' },
      ];

      useServerInfoStore.getState().setMCPServers(servers);

      const result = selectDisconnectedMCPServers(useServerInfoStore.getState());
      expect(result).toEqual([]);
    });
  });
});
