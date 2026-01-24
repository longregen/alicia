import { useState, useEffect, useCallback, useRef } from 'react';
import { Room, RoomEvent, Track, RemoteTrack, RemoteParticipant, RemoteTrackPublication } from 'livekit-client';
import { api } from '../services/api';
import { useWebSocket } from '../contexts/WebSocketContext';

export type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'reconnecting';

export interface UseLiveKitOptions {
  audioOutputEnabled?: boolean;
}

export interface UseLiveKitReturn {
  room: Room | null;
  connected: boolean;
  connectionState: ConnectionState;
  error: string | null;
  publishAudioTrack: (track: MediaStreamTrack) => Promise<void>;
  unpublishAudioTrack: () => Promise<void>;
  connect: () => Promise<void>;
  disconnect: () => void;
}

function getLiveKitURL(): string {
  return import.meta.env.VITE_LIVEKIT_URL || 'ws://localhost:7880';
}

function cleanupAudioElements(elements: Map<string, HTMLAudioElement>) {
  for (const audioEl of elements.values()) {
    audioEl.pause();
    audioEl.srcObject = null;
    audioEl.remove();
  }
  elements.clear();
}

export function useLiveKit(conversationId: string | null, options: UseLiveKitOptions = {}): UseLiveKitReturn {
  const { audioOutputEnabled = true } = options;
  const [room, setRoom] = useState<Room | null>(null);
  const [connected, setConnected] = useState(false);
  const [connectionState, setConnectionState] = useState<ConnectionState>('disconnected');
  const [error, setError] = useState<string | null>(null);

  const { sendVoiceJoinRequest, sendVoiceLeaveRequest } = useWebSocket();

  const audioTrackRef = useRef<MediaStreamTrack | null>(null);
  const roomRef = useRef<Room | null>(null);
  const pendingTrackRef = useRef<MediaStreamTrack | null>(null);
  const remoteAudioElementsRef = useRef<Map<string, HTMLAudioElement>>(new Map());
  const audioOutputEnabledRef = useRef(audioOutputEnabled);

  audioOutputEnabledRef.current = audioOutputEnabled;

  const handleTrackSubscribed = useCallback((
    track: RemoteTrack,
    publication: RemoteTrackPublication,
    _participant: RemoteParticipant
  ) => {
    if (track.kind !== Track.Kind.Audio) return;

    const audioEl = track.attach();
    audioEl.muted = !audioOutputEnabledRef.current;
    remoteAudioElementsRef.current.set(publication.trackSid, audioEl);
  }, []);

  const handleTrackUnsubscribed = useCallback((
    track: RemoteTrack,
    publication: RemoteTrackPublication,
    _participant: RemoteParticipant
  ) => {
    if (track.kind !== Track.Kind.Audio) return;

    track.detach();
    const audioEl = remoteAudioElementsRef.current.get(publication.trackSid);
    if (audioEl) {
      audioEl.pause();
      audioEl.srcObject = null;
      audioEl.remove();
      remoteAudioElementsRef.current.delete(publication.trackSid);
    }
  }, []);

  useEffect(() => {
    for (const audioEl of remoteAudioElementsRef.current.values()) {
      audioEl.muted = !audioOutputEnabled;
    }
  }, [audioOutputEnabled]);

  const connect = useCallback(async () => {
    if (!conversationId || roomRef.current) return;

    try {
      setError(null);
      setConnectionState('connecting');

      const token = await api.getLiveKitToken(conversationId);
      const liveKitUrl = getLiveKitURL();

      const newRoom = new Room({
        adaptiveStream: true,
        dynacast: true,
      });

      newRoom.on(RoomEvent.Connected, () => {
        setConnected(true);
        setConnectionState('connected');
        setError(null);

        sendVoiceJoinRequest(conversationId);

        if (pendingTrackRef.current) {
          newRoom.localParticipant.publishTrack(pendingTrackRef.current, {
            name: 'microphone',
            source: Track.Source.Microphone,
          }).then(() => {
            audioTrackRef.current = pendingTrackRef.current;
            pendingTrackRef.current = null;
          }).catch((err) => {
            console.error('Failed to publish queued audio track:', err);
          });
        }
      });

      newRoom.on(RoomEvent.Disconnected, () => {
        setConnected(false);
        setConnectionState('disconnected');
      });

      newRoom.on(RoomEvent.Reconnecting, () => {
        setConnectionState('reconnecting');
        setError('Connection lost. Reconnecting...');
      });

      newRoom.on(RoomEvent.Reconnected, () => {
        setConnected(true);
        setConnectionState('connected');
        setError(null);

        if (pendingTrackRef.current) {
          newRoom.localParticipant.publishTrack(pendingTrackRef.current, {
            name: 'microphone',
            source: Track.Source.Microphone,
          }).then(() => {
            audioTrackRef.current = pendingTrackRef.current;
            pendingTrackRef.current = null;
          }).catch((err) => {
            console.error('Failed to re-publish queued audio track:', err);
          });
        }
      });

      newRoom.on(RoomEvent.TrackSubscribed, handleTrackSubscribed);
      newRoom.on(RoomEvent.TrackUnsubscribed, handleTrackUnsubscribed);

      await newRoom.connect(liveKitUrl, token);
      roomRef.current = newRoom;
      setRoom(newRoom);
    } catch (err) {
      console.error('Failed to connect to LiveKit:', err);

      let errorMessage = 'Failed to connect to voice service';
      if (err instanceof Error) {
        if (err.message.includes('Network error')) {
          errorMessage = 'Network error: Unable to reach the voice service.';
        } else if (err.message.includes('token')) {
          errorMessage = 'Authentication failed. Please try creating a new conversation.';
        } else if (err.message.includes('timeout')) {
          errorMessage = 'Connection timeout. The voice service may be unavailable.';
        } else {
          errorMessage = err.message;
        }
      }

      setError(errorMessage);
      setConnectionState('disconnected');
    }
  }, [conversationId, handleTrackSubscribed, handleTrackUnsubscribed, sendVoiceJoinRequest]);

  const disconnect = useCallback(() => {
    if (roomRef.current) {
      if (conversationId) {
        sendVoiceLeaveRequest(conversationId);
      }

      roomRef.current.disconnect();
      roomRef.current = null;
      setRoom(null);
      setConnected(false);
      setConnectionState('disconnected');
      cleanupAudioElements(remoteAudioElementsRef.current);
    }
  }, [conversationId, sendVoiceLeaveRequest]);

  const publishAudioTrack = useCallback(async (track: MediaStreamTrack) => {
    if (!roomRef.current) {
      pendingTrackRef.current = track;
      return;
    }

    try {
      await roomRef.current.localParticipant.publishTrack(track, {
        name: 'microphone',
        source: Track.Source.Microphone,
      });
      audioTrackRef.current = track;
      pendingTrackRef.current = null;
    } catch (err) {
      console.error('Failed to publish audio track:', err);
      const errorMessage = err instanceof Error ? err.message : 'Failed to publish audio';
      setError(`Microphone access failed: ${errorMessage}`);
      throw err;
    }
  }, []);

  const unpublishAudioTrack = useCallback(async () => {
    if (!roomRef.current || !audioTrackRef.current) return;

    try {
      const publications = roomRef.current.localParticipant.audioTrackPublications;
      for (const [, publication] of publications) {
        if (publication.track) {
          await roomRef.current.localParticipant.unpublishTrack(publication.track);
        }
      }
      audioTrackRef.current?.stop();
      audioTrackRef.current = null;
    } catch (err) {
      console.error('Failed to unpublish audio track:', err);
    }
  }, []);

  const currentConvIdRef = useRef<string | null>(null);

  useEffect(() => {
    if (currentConvIdRef.current !== conversationId && roomRef.current) {
      if (currentConvIdRef.current) {
        sendVoiceLeaveRequest(currentConvIdRef.current);
      }

      roomRef.current.disconnect();
      roomRef.current = null;
      setRoom(null);
      setConnected(false);
      setConnectionState('disconnected');
      cleanupAudioElements(remoteAudioElementsRef.current);
    }
    currentConvIdRef.current = conversationId;

    if (conversationId && !roomRef.current) {
      connect();
    }
  }, [conversationId, connect, sendVoiceLeaveRequest]);

  useEffect(() => {
    const audioElements = remoteAudioElementsRef.current;
    return () => {
      if (roomRef.current) {
        if (currentConvIdRef.current) {
          sendVoiceLeaveRequest(currentConvIdRef.current);
        }
        roomRef.current.disconnect();
        roomRef.current = null;
      }
      cleanupAudioElements(audioElements);
    };
  }, [sendVoiceLeaveRequest]);

  return {
    room,
    connected,
    connectionState,
    error,
    publishAudioTrack,
    unpublishAudioTrack,
    connect,
    disconnect,
  };
}
