import { describe, it, expect, beforeEach } from 'vitest';
import {
  useConnectionStore,
  ConnectionStatus,
  selectAllParticipants,
  selectRemoteParticipants,
  selectIsConnected,
  selectConnectionUptime,
  type ParticipantInfo,
} from './connectionStore';

describe('connectionStore', () => {
  beforeEach(() => {
    useConnectionStore.getState().clearConnection();
  });

  describe('setConnectionStatus', () => {
    it('should update connection status', () => {
      useConnectionStore.getState().setConnectionStatus(ConnectionStatus.Connecting);

      const state = useConnectionStore.getState();
      expect(state.status).toBe(ConnectionStatus.Connecting);
    });

    it('should clear error when status changes to Connected', () => {
      useConnectionStore.getState().setError('Connection failed');
      useConnectionStore.getState().setConnectionStatus(ConnectionStatus.Connected);

      const state = useConnectionStore.getState();
      expect(state.status).toBe(ConnectionStatus.Connected);
      expect(state.error).toBeNull();
    });

    it('should not clear error for other status changes', () => {
      useConnectionStore.getState().setError('Connection failed');
      useConnectionStore.getState().setConnectionStatus(ConnectionStatus.Reconnecting);

      const state = useConnectionStore.getState();
      expect(state.error).toBe('Connection failed');
    });

    it('should default to Disconnected', () => {
      const state = useConnectionStore.getState();
      expect(state.status).toBe(ConnectionStatus.Disconnected);
    });
  });

  describe('setError', () => {
    it('should set error message', () => {
      useConnectionStore.getState().setError('Network error');

      const state = useConnectionStore.getState();
      expect(state.error).toBe('Network error');
    });

    it('should set status to Error when error is set', () => {
      useConnectionStore.getState().setError('Network error');

      const state = useConnectionStore.getState();
      expect(state.status).toBe(ConnectionStatus.Error);
    });

    it('should allow clearing error with null', () => {
      useConnectionStore.getState().setError('Network error');
      useConnectionStore.getState().setError(null);

      const state = useConnectionStore.getState();
      expect(state.error).toBeNull();
    });

    it('should not change status when clearing error', () => {
      useConnectionStore.getState().setError('Network error');
      useConnectionStore.getState().setConnectionStatus(ConnectionStatus.Reconnecting);
      useConnectionStore.getState().setError(null);

      const state = useConnectionStore.getState();
      expect(state.status).toBe(ConnectionStatus.Reconnecting);
    });
  });

  describe('setRoomInfo', () => {
    it('should set room name and sid', () => {
      useConnectionStore.getState().setRoomInfo('test-room', 'room-sid-123');

      const state = useConnectionStore.getState();
      expect(state.roomName).toBe('test-room');
      expect(state.roomSid).toBe('room-sid-123');
    });

    it('should update existing room info', () => {
      useConnectionStore.getState().setRoomInfo('room-1', 'sid-1');
      useConnectionStore.getState().setRoomInfo('room-2', 'sid-2');

      const state = useConnectionStore.getState();
      expect(state.roomName).toBe('room-2');
      expect(state.roomSid).toBe('sid-2');
    });
  });

  describe('clearRoomInfo', () => {
    it('should clear room name and sid', () => {
      useConnectionStore.getState().setRoomInfo('test-room', 'room-sid-123');
      useConnectionStore.getState().clearRoomInfo();

      const state = useConnectionStore.getState();
      expect(state.roomName).toBeNull();
      expect(state.roomSid).toBeNull();
    });
  });

  describe('addParticipant', () => {
    it('should add a participant to the store', () => {
      const participant: ParticipantInfo = {
        identity: 'user-1',
        name: 'Alice',
        isSpeaking: false,
        isMuted: false,
        isLocal: false,
      };

      useConnectionStore.getState().addParticipant(participant);

      const state = useConnectionStore.getState();
      expect(state.participants['user-1']).toBeDefined();
      expect(state.participants['user-1']).toEqual(participant);
    });

    it('should overwrite existing participant with same identity', () => {
      const participant1: ParticipantInfo = {
        identity: 'user-1',
        name: 'Alice',
        isSpeaking: false,
        isMuted: false,
        isLocal: false,
      };

      const participant2: ParticipantInfo = {
        identity: 'user-1',
        name: 'Alice Updated',
        isSpeaking: true,
        isMuted: false,
        isLocal: false,
      };

      useConnectionStore.getState().addParticipant(participant1);
      useConnectionStore.getState().addParticipant(participant2);

      const state = useConnectionStore.getState();
      expect(state.participants['user-1']).toEqual(participant2);
      expect(state.participants['user-1'].name).toBe('Alice Updated');
    });
  });

  describe('removeParticipant', () => {
    it('should remove a participant from the store', () => {
      const participant: ParticipantInfo = {
        identity: 'user-1',
        name: 'Alice',
        isSpeaking: false,
        isMuted: false,
        isLocal: false,
      };

      useConnectionStore.getState().addParticipant(participant);
      useConnectionStore.getState().removeParticipant('user-1');

      const state = useConnectionStore.getState();
      expect(state.participants['user-1']).toBeUndefined();
    });

    it('should not throw when removing non-existent participant', () => {
      expect(() => {
        useConnectionStore.getState().removeParticipant('non-existent');
      }).not.toThrow();
    });
  });

  describe('updateParticipant', () => {
    it('should update participant properties', () => {
      const participant: ParticipantInfo = {
        identity: 'user-1',
        name: 'Alice',
        isSpeaking: false,
        isMuted: false,
        isLocal: false,
      };

      useConnectionStore.getState().addParticipant(participant);
      useConnectionStore.getState().updateParticipant('user-1', { isSpeaking: true });

      const state = useConnectionStore.getState();
      expect(state.participants['user-1'].isSpeaking).toBe(true);
      expect(state.participants['user-1'].name).toBe('Alice');
    });

    it('should handle updating multiple properties', () => {
      const participant: ParticipantInfo = {
        identity: 'user-1',
        name: 'Alice',
        isSpeaking: false,
        isMuted: false,
        isLocal: false,
      };

      useConnectionStore.getState().addParticipant(participant);
      useConnectionStore.getState().updateParticipant('user-1', {
        isSpeaking: true,
        isMuted: true,
        name: 'Alice Updated',
      });

      const state = useConnectionStore.getState();
      expect(state.participants['user-1'].isSpeaking).toBe(true);
      expect(state.participants['user-1'].isMuted).toBe(true);
      expect(state.participants['user-1'].name).toBe('Alice Updated');
    });

    it('should not create participant if it does not exist', () => {
      useConnectionStore.getState().updateParticipant('non-existent', { isSpeaking: true });

      const state = useConnectionStore.getState();
      expect(state.participants['non-existent']).toBeUndefined();
    });
  });

  describe('setLocalParticipant', () => {
    it('should set the local participant id', () => {
      useConnectionStore.getState().setLocalParticipant('local-user-1');

      const state = useConnectionStore.getState();
      expect(state.localParticipantId).toBe('local-user-1');
    });

    it('should update existing local participant id', () => {
      useConnectionStore.getState().setLocalParticipant('local-user-1');
      useConnectionStore.getState().setLocalParticipant('local-user-2');

      const state = useConnectionStore.getState();
      expect(state.localParticipantId).toBe('local-user-2');
    });
  });

  describe('getParticipant', () => {
    it('should return participant by identity', () => {
      const participant: ParticipantInfo = {
        identity: 'user-1',
        name: 'Alice',
        isSpeaking: false,
        isMuted: false,
        isLocal: false,
      };

      useConnectionStore.getState().addParticipant(participant);

      const result = useConnectionStore.getState().getParticipant('user-1');
      expect(result).toEqual(participant);
    });

    it('should return undefined for non-existent participant', () => {
      const result = useConnectionStore.getState().getParticipant('non-existent');
      expect(result).toBeUndefined();
    });
  });

  describe('getLocalParticipant', () => {
    it('should return local participant', () => {
      const participant: ParticipantInfo = {
        identity: 'local-user',
        name: 'Me',
        isSpeaking: false,
        isMuted: false,
        isLocal: true,
      };

      useConnectionStore.getState().addParticipant(participant);
      useConnectionStore.getState().setLocalParticipant('local-user');

      const result = useConnectionStore.getState().getLocalParticipant();
      expect(result).toEqual(participant);
    });

    it('should return undefined when no local participant is set', () => {
      const result = useConnectionStore.getState().getLocalParticipant();
      expect(result).toBeUndefined();
    });

    it('should return undefined when local participant id does not exist', () => {
      useConnectionStore.getState().setLocalParticipant('non-existent');

      const result = useConnectionStore.getState().getLocalParticipant();
      expect(result).toBeUndefined();
    });
  });

  describe('setConnectedAt', () => {
    it('should set the connected timestamp', () => {
      const now = new Date();
      useConnectionStore.getState().setConnectedAt(now);

      const state = useConnectionStore.getState();
      expect(state.connectedAt).toBe(now);
    });
  });

  describe('reconnect attempts', () => {
    it('should increment reconnect attempts', () => {
      useConnectionStore.getState().incrementReconnectAttempts();
      useConnectionStore.getState().incrementReconnectAttempts();

      const state = useConnectionStore.getState();
      expect(state.reconnectAttempts).toBe(2);
    });

    it('should reset reconnect attempts to zero', () => {
      useConnectionStore.getState().incrementReconnectAttempts();
      useConnectionStore.getState().incrementReconnectAttempts();
      useConnectionStore.getState().resetReconnectAttempts();

      const state = useConnectionStore.getState();
      expect(state.reconnectAttempts).toBe(0);
    });

    it('should default to zero', () => {
      const state = useConnectionStore.getState();
      expect(state.reconnectAttempts).toBe(0);
    });
  });

  describe('clearConnection', () => {
    it('should reset all state to initial values', () => {
      const participant: ParticipantInfo = {
        identity: 'user-1',
        name: 'Alice',
        isSpeaking: false,
        isMuted: false,
        isLocal: false,
      };

      useConnectionStore.getState().setConnectionStatus(ConnectionStatus.Connected);
      useConnectionStore.getState().setRoomInfo('test-room', 'room-sid');
      useConnectionStore.getState().addParticipant(participant);
      useConnectionStore.getState().setLocalParticipant('user-1');
      useConnectionStore.getState().setConnectedAt(new Date());
      useConnectionStore.getState().incrementReconnectAttempts();
      useConnectionStore.getState().setError('Some error');

      useConnectionStore.getState().clearConnection();

      const state = useConnectionStore.getState();
      expect(state.status).toBe(ConnectionStatus.Disconnected);
      expect(state.error).toBeNull();
      expect(state.roomName).toBeNull();
      expect(state.roomSid).toBeNull();
      expect(Object.keys(state.participants)).toHaveLength(0);
      expect(state.localParticipantId).toBeNull();
      expect(state.connectedAt).toBeNull();
      expect(state.reconnectAttempts).toBe(0);
    });
  });

  describe('selectAllParticipants', () => {
    it('should return all participants as an array', () => {
      const participant1: ParticipantInfo = {
        identity: 'user-1',
        name: 'Alice',
        isSpeaking: false,
        isMuted: false,
        isLocal: false,
      };

      const participant2: ParticipantInfo = {
        identity: 'user-2',
        name: 'Bob',
        isSpeaking: false,
        isMuted: false,
        isLocal: true,
      };

      useConnectionStore.getState().addParticipant(participant1);
      useConnectionStore.getState().addParticipant(participant2);

      const result = selectAllParticipants(useConnectionStore.getState());
      expect(result).toHaveLength(2);
      expect(result).toContainEqual(participant1);
      expect(result).toContainEqual(participant2);
    });

    it('should return empty array when no participants exist', () => {
      const result = selectAllParticipants(useConnectionStore.getState());
      expect(result).toEqual([]);
    });
  });

  describe('selectRemoteParticipants', () => {
    it('should return only remote participants', () => {
      const localParticipant: ParticipantInfo = {
        identity: 'local-user',
        name: 'Me',
        isSpeaking: false,
        isMuted: false,
        isLocal: true,
      };

      const remoteParticipant1: ParticipantInfo = {
        identity: 'user-1',
        name: 'Alice',
        isSpeaking: false,
        isMuted: false,
        isLocal: false,
      };

      const remoteParticipant2: ParticipantInfo = {
        identity: 'user-2',
        name: 'Bob',
        isSpeaking: false,
        isMuted: false,
        isLocal: false,
      };

      useConnectionStore.getState().addParticipant(localParticipant);
      useConnectionStore.getState().addParticipant(remoteParticipant1);
      useConnectionStore.getState().addParticipant(remoteParticipant2);

      const result = selectRemoteParticipants(useConnectionStore.getState());
      expect(result).toHaveLength(2);
      expect(result).toContainEqual(remoteParticipant1);
      expect(result).toContainEqual(remoteParticipant2);
      expect(result).not.toContainEqual(localParticipant);
    });

    it('should return empty array when only local participant exists', () => {
      const localParticipant: ParticipantInfo = {
        identity: 'local-user',
        name: 'Me',
        isSpeaking: false,
        isMuted: false,
        isLocal: true,
      };

      useConnectionStore.getState().addParticipant(localParticipant);

      const result = selectRemoteParticipants(useConnectionStore.getState());
      expect(result).toEqual([]);
    });
  });

  describe('selectIsConnected', () => {
    it('should return true when status is Connected', () => {
      useConnectionStore.getState().setConnectionStatus(ConnectionStatus.Connected);

      const result = selectIsConnected(useConnectionStore.getState());
      expect(result).toBe(true);
    });

    it('should return false for other statuses', () => {
      useConnectionStore.getState().setConnectionStatus(ConnectionStatus.Connecting);
      expect(selectIsConnected(useConnectionStore.getState())).toBe(false);

      useConnectionStore.getState().setConnectionStatus(ConnectionStatus.Disconnected);
      expect(selectIsConnected(useConnectionStore.getState())).toBe(false);

      useConnectionStore.getState().setConnectionStatus(ConnectionStatus.Reconnecting);
      expect(selectIsConnected(useConnectionStore.getState())).toBe(false);

      useConnectionStore.getState().setConnectionStatus(ConnectionStatus.Error);
      expect(selectIsConnected(useConnectionStore.getState())).toBe(false);
    });
  });

  describe('selectConnectionUptime', () => {
    it('should return uptime in milliseconds', () => {
      const connectedAt = new Date(Date.now() - 5000);
      useConnectionStore.getState().setConnectedAt(connectedAt);

      const result = selectConnectionUptime(useConnectionStore.getState());
      expect(result).toBeGreaterThanOrEqual(5000);
      expect(result).toBeLessThan(6000);
    });

    it('should return 0 when not connected', () => {
      const result = selectConnectionUptime(useConnectionStore.getState());
      expect(result).toBe(0);
    });

    it('should return 0 when connectedAt is null', () => {
      useConnectionStore.getState().setConnectedAt(new Date());
      useConnectionStore.getState().clearConnection();

      const result = selectConnectionUptime(useConnectionStore.getState());
      expect(result).toBe(0);
    });
  });
});
