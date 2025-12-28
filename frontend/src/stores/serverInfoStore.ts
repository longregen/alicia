import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';

export type ConnectionStatus = 'connected' | 'connecting' | 'disconnected' | 'reconnecting';
export type MCPServerStatus = 'connected' | 'disconnected' | 'error';

export interface ModelInfo {
  name: string;
  provider: string;
}

export interface MCPServerInfo {
  name: string;
  status: MCPServerStatus;
}

export interface SessionStats {
  messageCount: number;
  toolCallCount: number;
  memoriesUsed: number;
  sessionDuration: number;
}

interface ServerInfoStoreState {
  connectionStatus: ConnectionStatus;
  latency: number;
  modelInfo: ModelInfo | null;
  mcpServers: MCPServerInfo[];
  sessionStats: SessionStats;
}

interface ServerInfoStoreActions {
  // Connection actions
  setConnectionStatus: (status: ConnectionStatus) => void;
  setLatency: (latency: number) => void;

  // Model actions
  setModelInfo: (modelInfo: ModelInfo) => void;

  // MCP server actions
  setMCPServers: (servers: MCPServerInfo[]) => void;
  updateMCPServer: (name: string, status: MCPServerStatus) => void;

  // Session stats actions
  setSessionStats: (stats: SessionStats) => void;
  updateSessionStats: (updates: Partial<SessionStats>) => void;
  incrementMessageCount: () => void;
  incrementToolCallCount: () => void;
  incrementMemoriesUsed: (count?: number) => void;
  updateSessionDuration: (duration: number) => void;

  // Bulk operations
  resetServerInfo: () => void;
}

type ServerInfoStore = ServerInfoStoreState & ServerInfoStoreActions;

const initialState: ServerInfoStoreState = {
  connectionStatus: 'disconnected',
  latency: 0,
  modelInfo: null,
  mcpServers: [],
  sessionStats: {
    messageCount: 0,
    toolCallCount: 0,
    memoriesUsed: 0,
    sessionDuration: 0,
  },
};

export const useServerInfoStore = create<ServerInfoStore>()(
  immer((set) => ({
    ...initialState,

    // Connection actions
    setConnectionStatus: (status) =>
      set((state) => {
        state.connectionStatus = status;
      }),

    setLatency: (latency) =>
      set((state) => {
        state.latency = latency;
      }),

    // Model actions
    setModelInfo: (modelInfo) =>
      set((state) => {
        state.modelInfo = modelInfo;
      }),

    // MCP server actions
    setMCPServers: (servers) =>
      set((state) => {
        state.mcpServers = servers;
      }),

    updateMCPServer: (name, status) =>
      set((state) => {
        const server = state.mcpServers.find((s) => s.name === name);
        if (server) {
          server.status = status;
        } else {
          state.mcpServers.push({ name, status });
        }
      }),

    // Session stats actions
    setSessionStats: (stats) =>
      set((state) => {
        state.sessionStats = stats;
      }),

    updateSessionStats: (updates) =>
      set((state) => {
        Object.assign(state.sessionStats, updates);
      }),

    incrementMessageCount: () =>
      set((state) => {
        state.sessionStats.messageCount += 1;
      }),

    incrementToolCallCount: () =>
      set((state) => {
        state.sessionStats.toolCallCount += 1;
      }),

    incrementMemoriesUsed: (count = 1) =>
      set((state) => {
        state.sessionStats.memoriesUsed += count;
      }),

    updateSessionDuration: (duration) =>
      set((state) => {
        state.sessionStats.sessionDuration = duration;
      }),

    // Bulk operations
    resetServerInfo: () =>
      set((state) => {
        Object.assign(state, initialState);
      }),
  }))
);

// Utility selectors
export const selectConnectionStatus = (state: ServerInfoStore) => state.connectionStatus;
export const selectLatency = (state: ServerInfoStore) => state.latency;
export const selectModelInfo = (state: ServerInfoStore) => state.modelInfo;
export const selectMCPServers = (state: ServerInfoStore) => state.mcpServers;
export const selectSessionStats = (state: ServerInfoStore) => state.sessionStats;

export const selectConnectedMCPServers = (state: ServerInfoStore) =>
  state.mcpServers.filter((s) => s.status === 'connected');

export const selectDisconnectedMCPServers = (state: ServerInfoStore) =>
  state.mcpServers.filter((s) => s.status === 'disconnected' || s.status === 'error');
