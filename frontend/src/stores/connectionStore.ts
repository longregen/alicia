import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';

export enum ConnectionStatus {
  Disconnected = 'disconnected',
  Connecting = 'connecting',
  Connected = 'connected',
  Reconnecting = 'reconnecting',
  Error = 'error',
}

export interface ParticipantInfo {
  identity: string;
  name?: string;
  isSpeaking: boolean;
  isMuted: boolean;
  isLocal: boolean;
}

interface ConnectionStoreState {
  // Connection state
  status: ConnectionStatus;
  error: string | null;

  // Room state
  roomName: string | null;
  roomSid: string | null;

  // Participant info
  participants: Record<string, ParticipantInfo>;
  localParticipantId: string | null;

  // Connection metadata
  connectedAt: Date | null;
  reconnectAttempts: number;
}

interface ConnectionStoreActions {
  // Connection actions
  setConnectionStatus: (status: ConnectionStatus) => void;
  setError: (error: string | null) => void;

  // Room actions
  setRoomInfo: (roomName: string, roomSid: string) => void;
  clearRoomInfo: () => void;

  // Participant actions
  addParticipant: (participant: ParticipantInfo) => void;
  removeParticipant: (identity: string) => void;
  updateParticipant: (identity: string, update: Partial<ParticipantInfo>) => void;
  setLocalParticipant: (identity: string) => void;

  // Connection metadata actions
  setConnectedAt: (date: Date) => void;
  incrementReconnectAttempts: () => void;
  resetReconnectAttempts: () => void;

  // Bulk operations
  clearConnection: () => void;

  // Selectors
  getParticipant: (identity: string) => ParticipantInfo | undefined;
  getLocalParticipant: () => ParticipantInfo | undefined;
}

type ConnectionStore = ConnectionStoreState & ConnectionStoreActions;

const initialState: ConnectionStoreState = {
  status: ConnectionStatus.Disconnected,
  error: null,
  roomName: null,
  roomSid: null,
  participants: {},
  localParticipantId: null,
  connectedAt: null,
  reconnectAttempts: 0,
};

// Check for E2E test mock - this allows e2e tests to set initial connection state
function getInitialState(): ConnectionStoreState {
  if (typeof window !== 'undefined' && (window as any).__E2E_CONNECTION_MOCK__) {
    // Create a fresh copy to avoid read-only property errors
    const mock = (window as any).__E2E_CONNECTION_MOCK__;
    return {
      status: mock.status,
      error: mock.error,
      roomName: mock.roomName,
      roomSid: mock.roomSid,
      participants: { ...mock.participants },
      localParticipantId: mock.localParticipantId,
      connectedAt: mock.connectedAt ? new Date(mock.connectedAt) : null,
      reconnectAttempts: mock.reconnectAttempts,
    };
  }
  return initialState;
}

export const useConnectionStore = create<ConnectionStore>()(
  immer((set, get) => ({
    ...getInitialState(),

    // Connection actions
    setConnectionStatus: (status) =>
      set((state) => {
        state.status = status;
        if (status === ConnectionStatus.Connected) {
          state.error = null;
        }
      }),

    setError: (error) =>
      set((state) => {
        state.error = error;
        if (error) {
          state.status = ConnectionStatus.Error;
        }
      }),

    // Room actions
    setRoomInfo: (roomName, roomSid) =>
      set((state) => {
        state.roomName = roomName;
        state.roomSid = roomSid;
      }),

    clearRoomInfo: () =>
      set((state) => {
        state.roomName = null;
        state.roomSid = null;
      }),

    // Participant actions
    addParticipant: (participant) =>
      set((state) => {
        state.participants[participant.identity] = participant;
      }),

    removeParticipant: (identity) =>
      set((state) => {
        delete state.participants[identity];
      }),

    updateParticipant: (identity, update) =>
      set((state) => {
        if (state.participants[identity]) {
          Object.assign(state.participants[identity], update);
        }
      }),

    setLocalParticipant: (identity) =>
      set((state) => {
        state.localParticipantId = identity;
      }),

    // Connection metadata actions
    setConnectedAt: (date) =>
      set((state) => {
        state.connectedAt = date;
      }),

    incrementReconnectAttempts: () =>
      set((state) => {
        state.reconnectAttempts += 1;
      }),

    resetReconnectAttempts: () =>
      set((state) => {
        state.reconnectAttempts = 0;
      }),

    // Bulk operations
    clearConnection: () =>
      set((state) => {
        Object.assign(state, initialState);
      }),

    // Selectors
    getParticipant: (identity) => {
      return get().participants[identity];
    },

    getLocalParticipant: () => {
      const state = get();
      return state.localParticipantId
        ? state.participants[state.localParticipantId]
        : undefined;
    },
  }))
);

// Utility selectors
export const selectAllParticipants = (state: ConnectionStore) =>
  Object.values(state.participants);

export const selectRemoteParticipants = (state: ConnectionStore) =>
  Object.values(state.participants).filter((p) => !p.isLocal);

export const selectIsConnected = (state: ConnectionStore) =>
  state.status === ConnectionStatus.Connected;

export const selectConnectionUptime = (state: ConnectionStore) =>
  state.connectedAt
    ? Date.now() - state.connectedAt.getTime()
    : 0;
