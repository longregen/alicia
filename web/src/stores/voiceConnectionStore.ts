import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';

export enum VoiceConnectionStatus {
  Idle = 'idle',
  Connecting = 'connecting',
  Connected = 'connected',
  Error = 'error',
  Retrying = 'retrying',
}

export interface VoiceSpeakingState {
  speaking: boolean;
  messageId: string | null;
  sentenceSeq: number | null;
}

interface VoiceConnectionState {
  status: VoiceConnectionStatus;
  retryCount: number;
  maxRetries: number;
  error: string | null;
  conversationId: string | null;
  speakingState: VoiceSpeakingState;
}

interface VoiceConnectionActions {
  setConnecting: (conversationId: string) => void;
  setConnected: () => void;
  setRetrying: (retryCount: number) => void;
  setError: (error: string) => void;
  setSpeaking: (speaking: boolean, messageId: string | null, sentenceSeq: number | null) => void;
  reset: () => void;
}

type VoiceConnectionStore = VoiceConnectionState & VoiceConnectionActions;

const MAX_RETRIES = 3;

const initialState: VoiceConnectionState = {
  status: VoiceConnectionStatus.Idle,
  retryCount: 0,
  maxRetries: MAX_RETRIES,
  error: null,
  conversationId: null,
  speakingState: {
    speaking: false,
    messageId: null,
    sentenceSeq: null,
  },
};

export const useVoiceConnectionStore = create<VoiceConnectionStore>()(
  immer((set) => ({
    ...initialState,

    setConnecting: (conversationId: string) =>
      set((state) => {
        state.status = VoiceConnectionStatus.Connecting;
        state.conversationId = conversationId;
        state.error = null;
      }),

    setConnected: () =>
      set((state) => {
        state.status = VoiceConnectionStatus.Connected;
        state.retryCount = 0;
        state.error = null;
      }),

    setRetrying: (retryCount: number) =>
      set((state) => {
        state.status = VoiceConnectionStatus.Retrying;
        state.retryCount = retryCount;
        state.error = null;
      }),

    setError: (error: string) =>
      set((state) => {
        state.status = VoiceConnectionStatus.Error;
        state.error = error;
      }),

    setSpeaking: (speaking: boolean, messageId: string | null, sentenceSeq: number | null) =>
      set((state) => {
        state.speakingState.speaking = speaking;
        state.speakingState.messageId = messageId;
        state.speakingState.sentenceSeq = sentenceSeq;
      }),

    reset: () =>
      set((state) => {
        Object.assign(state, initialState);
      }),
  }))
);
