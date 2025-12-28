import { useState, useEffect, useCallback, useRef } from 'react';
import { Room, RoomEvent, Track, RemoteTrack, RemoteParticipant, RemoteTrackPublication } from 'livekit-client';
import { api, PublicConfig } from '../services/api';
import { ProtocolService } from '../services/protocol';
import {
  Envelope,
  Features,
  MessageType,
  AssistantSentence,
  Transcription,
  UserMessage,
  AssistantMessage,
  ErrorMessage,
  ReasoningStep,
  ToolUseRequest,
  ToolUseResult,
  Acknowledgement,
  MemoryTrace,
  Commentary,
  StartAnswer,
  VariationType,
  AudioChunk,
  FeedbackConfirmation,
  NoteConfirmation,
  MemoryConfirmation,
  ServerInfo,
  SessionStats,
  EliteOptions,
} from '../types/protocol';
import { useMessageContext } from '../contexts/MessageContext';
import { Message } from '../types/models';
import { useFeedbackStore, VotableType, VoteType } from '../stores/feedbackStore';
import { useServerInfoStore } from '../stores/serverInfoStore';
import { useDimensionStore } from '../stores/dimensionStore';
import { setMessageSender } from '../adapters/protocolAdapter';

export interface LiveKitMessage {
  envelope: Envelope;
  timestamp: number;
}

export type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'reconnecting';

export interface UseLiveKitReturn {
  room: Room | null;
  connected: boolean;
  connectionState: ConnectionState;
  error: string | null;
  messages: LiveKitMessage[];
  sendMessage: (content: string) => Promise<void>;
  sendStop: (targetId?: string) => Promise<void>;
  sendRegenerate: (targetId: string, variationType?: VariationType) => Promise<void>;
  publishAudioTrack: (track: MediaStreamTrack) => Promise<void>;
  unpublishAudioTrack: () => Promise<void>;
  connect: () => Promise<void>;
  disconnect: () => void;
}

// Cache the config to avoid repeated fetches
let cachedConfig: PublicConfig | null = null;
let configFetchPromise: Promise<PublicConfig> | null = null;

async function getLiveKitURL(): Promise<string> {
  // Use environment variable override if set
  const envUrl = import.meta.env.VITE_LIVEKIT_URL;
  if (envUrl) {
    return envUrl;
  }

  // Fetch from server (with caching)
  if (cachedConfig?.livekit_url) {
    return cachedConfig.livekit_url;
  }

  if (!configFetchPromise) {
    configFetchPromise = api.getConfig().then(config => {
      cachedConfig = config;
      return config;
    }).catch(err => {
      console.error('Failed to fetch config:', err);
      configFetchPromise = null;
      throw err;
    });
  }

  const config = await configFetchPromise;
  return config.livekit_url || 'ws://localhost:7880';
}

export function useLiveKit(conversationId: string | null): UseLiveKitReturn {
  const [room, setRoom] = useState<Room | null>(null);
  const [connected, setConnected] = useState(false);
  const [connectionState, setConnectionState] = useState<ConnectionState>('disconnected');
  const [error, setError] = useState<string | null>(null);
  const [messages, setMessages] = useState<LiveKitMessage[]>([]);
  const [lastSeenStanzaId, setLastSeenStanzaId] = useState<number>(0);

  const messageContext = useMessageContext();
  const messageContextRef = useRef(messageContext);
  const protocolRef = useRef(new ProtocolService());
  const audioTrackRef = useRef<MediaStreamTrack | null>(null);
  const roomRef = useRef<Room | null>(null);

  // Keep messageContext ref updated (no deps - runs every render)
  useEffect(() => {
    messageContextRef.current = messageContext;
  });

  // Handle incoming data messages
  const handleDataReceived = useCallback((payload: Uint8Array) => {
    try {
      const envelope = protocolRef.current.decode(payload);
      const timestamp = Date.now();

      // Track last seen stanza ID from server messages (negative IDs)
      if (envelope.stanzaId < 0) {
        setLastSeenStanzaId(Math.abs(envelope.stanzaId));
      }

      // Add to LiveKit messages for protocol handling
      setMessages(prev => [...prev, { envelope, timestamp }]);

      // Update unified message store based on message type
      switch (envelope.type) {
        case MessageType.AssistantSentence: {
          const sentence = envelope.body as AssistantSentence;
          messageContextRef.current.updateStreamingSentence(sentence.sequence, sentence.text);

          if (sentence.isFinal) {
            // When streaming is complete, create a final message
            const fullContent = Array.from(messageContextRef.current.streamingMessages.values()).join(' ');
            if (fullContent.trim()) {
              const finalMessage: Message = {
                id: sentence.id || `assistant-${timestamp}`,
                conversation_id: envelope.conversationId,
                sequence_number: 0, // Will be set by backend
                role: 'assistant',
                contents: fullContent,
                created_at: new Date(timestamp).toISOString(),
                updated_at: new Date(timestamp).toISOString(),
              };
              messageContextRef.current.finalizeStreamingMessage(finalMessage);
            }
          }
          break;
        }

        case MessageType.Transcription: {
          const transcription = envelope.body as Transcription;
          messageContextRef.current.setTranscription(transcription.text);

          if (transcription.final) {
            // Clear transcription after a delay
            setTimeout(() => {
              messageContextRef.current.clearTranscription();
            }, 1000);

            // Add user message when transcription is finalized
            if (transcription.text.trim()) {
              const userMessage: Message = {
                id: transcription.id,
                conversation_id: envelope.conversationId,
                sequence_number: 0, // Will be set by backend
                role: 'user',
                contents: transcription.text,
                created_at: new Date(timestamp).toISOString(),
                updated_at: new Date(timestamp).toISOString(),
              };
              messageContextRef.current.addMessage(userMessage);
            }
          }
          break;
        }

        case MessageType.StartAnswer: {
          // Clear streaming sentences when new answer starts
          const startAnswer = envelope.body as StartAnswer;
          messageContextRef.current.clearStreamingSentences();
          messageContextRef.current.setIsGenerating(true, startAnswer.id);
          break;
        }

        case MessageType.UserMessage: {
          const userMsg = envelope.body as UserMessage;
          const message: Message = {
            id: userMsg.id,
            conversation_id: envelope.conversationId,
            sequence_number: 0, // Will be set by backend
            role: 'user',
            contents: userMsg.content,
            created_at: new Date(userMsg.timestamp || timestamp).toISOString(),
            updated_at: new Date(userMsg.timestamp || timestamp).toISOString(),
          };
          messageContextRef.current.addMessage(message);
          break;
        }

        case MessageType.AssistantMessage: {
          const assistantMsg = envelope.body as AssistantMessage;
          const message: Message = {
            id: assistantMsg.id,
            conversation_id: envelope.conversationId,
            sequence_number: 0, // Will be set by backend
            role: 'assistant',
            contents: assistantMsg.content,
            created_at: new Date(assistantMsg.timestamp || timestamp).toISOString(),
            updated_at: new Date(assistantMsg.timestamp || timestamp).toISOString(),
          };
          messageContextRef.current.addMessage(message);
          break;
        }

        case MessageType.ErrorMessage: {
          const error = envelope.body as ErrorMessage;
          messageContextRef.current.addError(error);
          break;
        }

        case MessageType.ReasoningStep: {
          const step = envelope.body as ReasoningStep;
          messageContextRef.current.addReasoningStep(step);
          break;
        }

        case MessageType.ToolUseRequest: {
          const request = envelope.body as ToolUseRequest;
          messageContextRef.current.addToolUsage({ request, result: null });
          break;
        }

        case MessageType.ToolUseResult: {
          const result = envelope.body as ToolUseResult;
          messageContextRef.current.updateToolUsageResult(result);
          break;
        }

        case MessageType.Acknowledgement: {
          const ack = envelope.body as Acknowledgement;
          messageContextRef.current.handleAcknowledgement(ack);
          break;
        }

        case MessageType.MemoryTrace: {
          const trace = envelope.body as MemoryTrace;
          messageContextRef.current.addMemoryTrace(trace);
          break;
        }

        case MessageType.Commentary: {
          const commentary = envelope.body as Commentary;
          messageContextRef.current.addCommentary(commentary);
          break;
        }

        case MessageType.AudioChunk: {
          const audioChunk = envelope.body as AudioChunk;
          // Handle audio metadata for synchronization
          // This is primarily for debugging/logging audio stream state
          console.debug('Audio chunk received:', {
            format: audioChunk.format,
            sequence: audioChunk.sequence,
            durationMs: audioChunk.durationMs,
            isLast: audioChunk.isLast,
          });

          // If isLast is true, this signals end of TTS output
          if (audioChunk.isLast) {
            // Could emit an event or update state to indicate speaking finished
          }
          break;
        }

        case MessageType.ControlVariation: {
          // Control variations can be handled by the UI layer if needed
          break;
        }

        // Feedback protocol message types (20-27)
        case MessageType.FeedbackConfirmation: {
          const confirmation = envelope.body as FeedbackConfirmation;
          // Update the local vote state with server-confirmed data
          const targetType = confirmation.targetType as VotableType;
          const voteValue = confirmation.userVote as VoteType | null;
          const feedbackStore = useFeedbackStore.getState();

          // Update user's vote
          if (voteValue) {
            feedbackStore.addVote(targetType, confirmation.targetId, voteValue);
          } else {
            feedbackStore.removeVote(targetType, confirmation.targetId);
          }

          // Update server aggregates
          if (confirmation.aggregates) {
            feedbackStore.setAggregates(targetType, confirmation.targetId, {
              upvotes: confirmation.aggregates.upvotes,
              downvotes: confirmation.aggregates.downvotes,
              special: confirmation.aggregates.specialVotes,
            });
          }
          break;
        }

        case MessageType.NoteConfirmation: {
          const confirmation = envelope.body as NoteConfirmation;
          // Note confirmations are informational - the note was already added locally
          // If it failed, we could remove the local note, but for now just log
          if (!confirmation.success) {
            console.warn('Note creation failed for message:', confirmation.messageId);
          }
          break;
        }

        case MessageType.MemoryConfirmation: {
          const confirmation = envelope.body as MemoryConfirmation;
          // Memory confirmations are informational
          if (!confirmation.success) {
            console.warn('Memory action failed:', confirmation.action, confirmation.memoryId);
          }
          break;
        }

        case MessageType.ServerInfo: {
          const serverInfo = envelope.body as ServerInfo;
          const serverStore = useServerInfoStore.getState();

          // Update connection info
          serverStore.setConnectionStatus(
            serverInfo.connection.status as 'connected' | 'connecting' | 'disconnected' | 'reconnecting'
          );
          serverStore.setLatency(serverInfo.connection.latency);

          // Update model info
          serverStore.setModelInfo({
            name: serverInfo.model.name,
            provider: serverInfo.model.provider,
          });

          // Update MCP servers
          serverStore.setMCPServers(
            serverInfo.mcpServers.map((s) => ({
              name: s.name,
              status: s.status as 'connected' | 'disconnected' | 'error',
            }))
          );
          break;
        }

        case MessageType.SessionStats: {
          const stats = envelope.body as SessionStats;
          useServerInfoStore.getState().setSessionStats({
            messageCount: stats.messageCount,
            toolCallCount: stats.toolCallCount,
            memoriesUsed: stats.memoriesUsed,
            sessionDuration: stats.sessionDuration,
          });
          break;
        }

        // Dimension optimization message types (29-31)
        case MessageType.EliteOptions: {
          // Server broadcasts available elite solutions from Pareto archive
          const eliteOptions = envelope.body as EliteOptions;
          const dimensionStore = useDimensionStore.getState();
          dimensionStore.updateElites(eliteOptions.elites, eliteOptions.currentEliteId);
          break;
        }

        // Note: DimensionPreference (29) and EliteSelect (30) are client -> server only
        // The server doesn't send these back; instead it responds with EliteOptions (31)
      }
    } catch (err) {
      console.error('Failed to decode message:', err);
    }
  }, []);

  // Handle audio track subscribed
  const handleTrackSubscribed = useCallback((
    _track: RemoteTrack,
    _publication: RemoteTrackPublication,
    _participant: RemoteParticipant
  ) => {
    // Audio will be automatically played by LiveKit
  }, []);

  // Connect to LiveKit room
  const connect = useCallback(async () => {
    if (!conversationId || room) return;

    try {
      setError(null);
      setConnectionState('connecting');

      const [token, liveKitUrl] = await Promise.all([
        api.getLiveKitToken(conversationId),
        getLiveKitURL(),
      ]);

      const newRoom = new Room({
        adaptiveStream: true,
        dynacast: true,
      });

      // Set up event listeners
      newRoom.on(RoomEvent.Connected, () => {
        setConnected(true);
        setConnectionState('connected');
        setError(null);

        // Wire up the message sender for the protocol adapter
        // This allows components to send messages (DimensionPreference, EliteSelect, etc.)
        setMessageSender((envelope: Envelope) => {
          const data = protocolRef.current.encode(envelope);
          newRoom.localParticipant.publishData(data, { reliable: true });
        });

        // Send configuration message (initial connection)
        const configEnvelope = protocolRef.current.createConfiguration(
          conversationId,
          [Features.STREAMING, Features.AUDIO_OUTPUT, Features.PARTIAL_RESPONSES],
          0  // First connection, no messages seen yet
        );
        const data = protocolRef.current.encode(configEnvelope);
        newRoom.localParticipant.publishData(data, { reliable: true });
      });

      newRoom.on(RoomEvent.Disconnected, () => {
        setConnected(false);
        setConnectionState('disconnected');
        // Clear the message sender when disconnected
        setMessageSender(null);
      });

      newRoom.on(RoomEvent.Reconnecting, () => {
        setConnectionState('reconnecting');
        setError('Connection lost. Reconnecting...');
      });

      newRoom.on(RoomEvent.Reconnected, () => {
        setConnected(true);
        setConnectionState('connected');
        setError(null);

        // Re-wire the message sender after reconnection
        setMessageSender((envelope: Envelope) => {
          const data = protocolRef.current.encode(envelope);
          newRoom.localParticipant.publishData(data, { reliable: true });
        });

        // Send Configuration with lastSequenceSeen to trigger message replay
        const configEnvelope = protocolRef.current.createConfiguration(
          conversationId,
          [Features.STREAMING, Features.AUDIO_OUTPUT, Features.PARTIAL_RESPONSES],
          lastSeenStanzaId
        );
        const data = protocolRef.current.encode(configEnvelope);
        newRoom.localParticipant.publishData(data, { reliable: true });
      });

      newRoom.on(RoomEvent.DataReceived, (payload: Uint8Array) => {
        handleDataReceived(payload);
      });

      newRoom.on(RoomEvent.TrackSubscribed, handleTrackSubscribed);

      // Connect to the room
      await newRoom.connect(liveKitUrl, token);
      roomRef.current = newRoom;
      setRoom(newRoom);
    } catch (err) {
      console.error('Failed to connect to LiveKit:', err);

      let errorMessage = 'Failed to connect to voice service';
      if (err instanceof Error) {
        if (err.message.includes('Network error')) {
          errorMessage = 'Network error: Unable to reach the voice service. Please check your connection.';
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
  }, [conversationId, room, handleDataReceived, handleTrackSubscribed, lastSeenStanzaId]);

  // Disconnect from room
  const disconnect = useCallback(() => {
    if (room) {
      room.disconnect();
      roomRef.current = null;
      setRoom(null);
      setConnected(false);
      setConnectionState('disconnected');
      setMessages([]);
      setLastSeenStanzaId(0);
      messageContextRef.current.clearStreamingSentences();
      messageContextRef.current.clearTranscription();
      // Clear the message sender
      setMessageSender(null);
    }
  }, [room]);

  // Send a text message
  const sendMessage = useCallback(async (content: string) => {
    if (!room || !conversationId || !content.trim()) return;

    try {
      const envelope = protocolRef.current.createUserMessage(conversationId, content);
      const data = protocolRef.current.encode(envelope);
      await room.localParticipant.publishData(data, { reliable: true });
    } catch (err) {
      console.error('Failed to send message:', err);
      const errorMessage = err instanceof Error ? err.message : 'Failed to send message';
      setError(`Message send failed: ${errorMessage}`);
      throw err;
    }
  }, [room, conversationId]);

  // Send a stop control message
  const sendStop = useCallback(async (targetId?: string) => {
    if (!room || !conversationId) return;

    try {
      const envelope = protocolRef.current.createControlStop(conversationId, targetId);
      const data = protocolRef.current.encode(envelope);
      await room.localParticipant.publishData(data, { reliable: true });

      // Update local state to reflect stopping
      messageContextRef.current.setIsGenerating(false);
    } catch (err) {
      console.error('Failed to send stop:', err);
      const errorMessage = err instanceof Error ? err.message : 'Failed to send stop';
      setError(`Stop command failed: ${errorMessage}`);
      throw err;
    }
  }, [room, conversationId]);

  // Send a regenerate (variation) control message
  const sendRegenerate = useCallback(async (targetId: string, variationType: VariationType = 'regenerate') => {
    if (!room || !conversationId) return;

    try {
      const envelope = protocolRef.current.createControlVariation(conversationId, targetId, variationType);
      const data = protocolRef.current.encode(envelope);
      await room.localParticipant.publishData(data, { reliable: true });

      // Set generating state as we expect a new response
      messageContextRef.current.setIsGenerating(true);
    } catch (err) {
      console.error('Failed to send regenerate:', err);
      const errorMessage = err instanceof Error ? err.message : 'Failed to send regenerate';
      setError(`Regenerate command failed: ${errorMessage}`);
      throw err;
    }
  }, [room, conversationId]);

  // Publish audio track
  const publishAudioTrack = useCallback(async (track: MediaStreamTrack) => {
    if (!room) return;

    try {
      await room.localParticipant.publishTrack(track, {
        name: 'microphone',
        source: Track.Source.Microphone,
      });
      audioTrackRef.current = track;
    } catch (err) {
      console.error('Failed to publish audio track:', err);
      const errorMessage = err instanceof Error ? err.message : 'Failed to publish audio';
      setError(`Microphone access failed: ${errorMessage}`);
      throw err;
    }
  }, [room]);

  // Unpublish audio track
  const unpublishAudioTrack = useCallback(async () => {
    if (!room || !audioTrackRef.current) return;

    try {
      const publications = room.localParticipant.audioTrackPublications;
      for (const [, publication] of publications) {
        await room.localParticipant.unpublishTrack(publication.track!);
      }
      audioTrackRef.current?.stop();
      audioTrackRef.current = null;
    } catch (err) {
      console.error('Failed to unpublish audio track:', err);
    }
  }, [room]);

  // Auto-connect when conversation changes
  useEffect(() => {
    if (conversationId && !room) {
      connect();
    }
  }, [conversationId, room, connect]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (roomRef.current) {
        roomRef.current.disconnect();
        roomRef.current = null;
      }
    };
  }, []);

  return {
    room,
    connected,
    connectionState,
    error,
    messages,
    sendMessage,
    sendStop,
    sendRegenerate,
    publishAudioTrack,
    unpublishAudioTrack,
    connect,
    disconnect,
  };
}
