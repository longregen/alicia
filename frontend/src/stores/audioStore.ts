import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';
import { AudioRef, AudioRefId } from '../types/streaming';

export interface AudioPlaybackState {
  currentlyPlayingId: AudioRefId | null;
  playbackProgress: number; // 0-1
  isPlaying: boolean;
  volume: number; // 0-1
  isMuted: boolean;
}

interface AudioStoreState {
  // Audio metadata
  audioRefs: Record<string, AudioRef>;

  // Playback state
  playback: AudioPlaybackState;
}

interface AudioStoreActions {
  // Audio ref actions
  addAudioRef: (audioRef: AudioRef) => void;
  getAudioRef: (id: AudioRefId) => AudioRef | undefined;

  // Playback control actions
  startPlayback: (id: AudioRefId) => void;
  stopPlayback: () => void;
  updatePlaybackProgress: (progress: number) => void;
  setVolume: (volume: number) => void;
  toggleMute: () => void;

  // Bulk operations
  clearAudioStore: () => void;
}

type AudioStore = AudioStoreState & AudioStoreActions;

const initialState: AudioStoreState = {
  audioRefs: {},
  playback: {
    currentlyPlayingId: null,
    playbackProgress: 0,
    isPlaying: false,
    volume: 1.0,
    isMuted: false,
  },
};

export const useAudioStore = create<AudioStore>()(
  immer((set, get) => ({
    ...initialState,

    // Audio ref actions
    addAudioRef: (audioRef) =>
      set((state) => {
        state.audioRefs[audioRef.id] = audioRef;
      }),

    getAudioRef: (id) => {
      return get().audioRefs[id];
    },

    // Playback control actions
    startPlayback: (id) =>
      set((state) => {
        state.playback.currentlyPlayingId = id;
        state.playback.isPlaying = true;
        state.playback.playbackProgress = 0;
      }),

    stopPlayback: () =>
      set((state) => {
        state.playback.isPlaying = false;
        state.playback.currentlyPlayingId = null;
        state.playback.playbackProgress = 0;
      }),

    updatePlaybackProgress: (progress) =>
      set((state) => {
        state.playback.playbackProgress = Math.max(0, Math.min(1, progress));
      }),

    setVolume: (volume) =>
      set((state) => {
        state.playback.volume = Math.max(0, Math.min(1, volume));
      }),

    toggleMute: () =>
      set((state) => {
        state.playback.isMuted = !state.playback.isMuted;
      }),

    // Bulk operations
    clearAudioStore: () =>
      set((state) => {
        Object.assign(state, initialState);
      }),
  }))
);

// Utility selectors
export const selectCurrentlyPlayingAudio = (state: AudioStore) =>
  state.playback.currentlyPlayingId
    ? state.audioRefs[state.playback.currentlyPlayingId]
    : null;

export const selectAllAudioRefs = (state: AudioStore) =>
  Object.values(state.audioRefs);
