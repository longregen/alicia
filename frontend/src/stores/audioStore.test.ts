import { describe, it, expect, beforeEach } from 'vitest';
import {
  useAudioStore,
  selectCurrentlyPlayingAudio,
  selectAllAudioRefs,
} from './audioStore';
import { createAudioRefId } from '../types/streaming';
import type { AudioRef } from '../types/streaming';

describe('audioStore', () => {
  beforeEach(() => {
    useAudioStore.getState().clearAudioStore();
  });

  describe('addAudioRef', () => {
    it('should add an audio ref to the store', () => {
      const audioRef: AudioRef = {
        id: createAudioRefId('audio-1'),
        sizeBytes: 1024,
        durationMs: 5000,
        sampleRate: 44100,
      };

      useAudioStore.getState().addAudioRef(audioRef);

      const state = useAudioStore.getState();
      expect(state.audioRefs['audio-1']).toBeDefined();
      expect(state.audioRefs['audio-1']).toEqual(audioRef);
    });

    it('should store multiple audio refs independently', () => {
      const audioRef1: AudioRef = {
        id: createAudioRefId('audio-1'),
        sizeBytes: 1024,
        durationMs: 5000,
        sampleRate: 44100,
      };

      const audioRef2: AudioRef = {
        id: createAudioRefId('audio-2'),
        sizeBytes: 2048,
        durationMs: 10000,
        sampleRate: 48000,
      };

      useAudioStore.getState().addAudioRef(audioRef1);
      useAudioStore.getState().addAudioRef(audioRef2);

      const state = useAudioStore.getState();
      expect(state.audioRefs['audio-1']).toEqual(audioRef1);
      expect(state.audioRefs['audio-2']).toEqual(audioRef2);
    });

    it('should overwrite existing audio ref with same id', () => {
      const audioRef1: AudioRef = {
        id: createAudioRefId('audio-1'),
        sizeBytes: 1024,
        durationMs: 5000,
        sampleRate: 44100,
      };

      const audioRef2: AudioRef = {
        id: createAudioRefId('audio-1'),
        sizeBytes: 2048,
        durationMs: 8000,
        sampleRate: 48000,
      };

      useAudioStore.getState().addAudioRef(audioRef1);
      useAudioStore.getState().addAudioRef(audioRef2);

      const state = useAudioStore.getState();
      expect(state.audioRefs['audio-1']).toEqual(audioRef2);
      expect(state.audioRefs['audio-1'].sizeBytes).toBe(2048);
    });
  });

  describe('getAudioRef', () => {
    it('should return the audio ref by id', () => {
      const audioRef: AudioRef = {
        id: createAudioRefId('audio-1'),
        sizeBytes: 1024,
        durationMs: 5000,
        sampleRate: 44100,
      };

      useAudioStore.getState().addAudioRef(audioRef);

      const result = useAudioStore.getState().getAudioRef(createAudioRefId('audio-1'));
      expect(result).toEqual(audioRef);
    });

    it('should return undefined for non-existent audio ref', () => {
      const result = useAudioStore.getState().getAudioRef(createAudioRefId('non-existent'));
      expect(result).toBeUndefined();
    });
  });

  describe('startPlayback', () => {
    it('should start playback and set current playing id', () => {
      const audioId = createAudioRefId('audio-1');

      useAudioStore.getState().startPlayback(audioId);

      const state = useAudioStore.getState();
      expect(state.playback.currentlyPlayingId).toBe(audioId);
      expect(state.playback.isPlaying).toBe(true);
      expect(state.playback.playbackProgress).toBe(0);
    });

    it('should reset progress when starting new playback', () => {
      const audioId = createAudioRefId('audio-1');

      useAudioStore.getState().startPlayback(audioId);
      useAudioStore.getState().updatePlaybackProgress(0.5);
      useAudioStore.getState().startPlayback(createAudioRefId('audio-2'));

      const state = useAudioStore.getState();
      expect(state.playback.playbackProgress).toBe(0);
      expect(state.playback.currentlyPlayingId).toBe('audio-2');
    });
  });

  describe('stopPlayback', () => {
    it('should stop playback and reset state', () => {
      const audioId = createAudioRefId('audio-1');

      useAudioStore.getState().startPlayback(audioId);
      useAudioStore.getState().updatePlaybackProgress(0.5);
      useAudioStore.getState().stopPlayback();

      const state = useAudioStore.getState();
      expect(state.playback.isPlaying).toBe(false);
      expect(state.playback.currentlyPlayingId).toBeNull();
      expect(state.playback.playbackProgress).toBe(0);
    });

    it('should handle stopping when nothing is playing', () => {
      expect(() => {
        useAudioStore.getState().stopPlayback();
      }).not.toThrow();

      const state = useAudioStore.getState();
      expect(state.playback.isPlaying).toBe(false);
      expect(state.playback.currentlyPlayingId).toBeNull();
    });
  });

  describe('updatePlaybackProgress', () => {
    it('should update playback progress', () => {
      useAudioStore.getState().updatePlaybackProgress(0.5);

      const state = useAudioStore.getState();
      expect(state.playback.playbackProgress).toBe(0.5);
    });

    it('should clamp progress between 0 and 1', () => {
      useAudioStore.getState().updatePlaybackProgress(1.5);
      expect(useAudioStore.getState().playback.playbackProgress).toBe(1);

      useAudioStore.getState().updatePlaybackProgress(-0.5);
      expect(useAudioStore.getState().playback.playbackProgress).toBe(0);
    });

    it('should allow progress values at boundaries', () => {
      useAudioStore.getState().updatePlaybackProgress(0);
      expect(useAudioStore.getState().playback.playbackProgress).toBe(0);

      useAudioStore.getState().updatePlaybackProgress(1);
      expect(useAudioStore.getState().playback.playbackProgress).toBe(1);
    });
  });

  describe('setVolume', () => {
    it('should set volume level', () => {
      useAudioStore.getState().setVolume(0.5);

      const state = useAudioStore.getState();
      expect(state.playback.volume).toBe(0.5);
    });

    it('should clamp volume between 0 and 1', () => {
      useAudioStore.getState().setVolume(1.5);
      expect(useAudioStore.getState().playback.volume).toBe(1);

      useAudioStore.getState().setVolume(-0.5);
      expect(useAudioStore.getState().playback.volume).toBe(0);
    });

    it('should default to 1.0', () => {
      const state = useAudioStore.getState();
      expect(state.playback.volume).toBe(1.0);
    });
  });

  describe('toggleMute', () => {
    it('should toggle mute state from false to true', () => {
      useAudioStore.getState().toggleMute();

      const state = useAudioStore.getState();
      expect(state.playback.isMuted).toBe(true);
    });

    it('should toggle mute state from true to false', () => {
      useAudioStore.getState().toggleMute();
      useAudioStore.getState().toggleMute();

      const state = useAudioStore.getState();
      expect(state.playback.isMuted).toBe(false);
    });

    it('should default to false', () => {
      const state = useAudioStore.getState();
      expect(state.playback.isMuted).toBe(false);
    });
  });

  describe('clearAudioStore', () => {
    it('should reset all state to initial values', () => {
      const audioRef: AudioRef = {
        id: createAudioRefId('audio-1'),
        sizeBytes: 1024,
        durationMs: 5000,
        sampleRate: 44100,
      };

      useAudioStore.getState().addAudioRef(audioRef);
      useAudioStore.getState().startPlayback(audioRef.id);
      useAudioStore.getState().setVolume(0.5);
      useAudioStore.getState().toggleMute();
      useAudioStore.getState().updatePlaybackProgress(0.75);

      useAudioStore.getState().clearAudioStore();

      const state = useAudioStore.getState();
      expect(Object.keys(state.audioRefs)).toHaveLength(0);
      expect(state.playback.currentlyPlayingId).toBeNull();
      expect(state.playback.playbackProgress).toBe(0);
      expect(state.playback.isPlaying).toBe(false);
      expect(state.playback.volume).toBe(1.0);
      expect(state.playback.isMuted).toBe(false);
    });
  });

  describe('selectCurrentlyPlayingAudio', () => {
    it('should return the currently playing audio ref', () => {
      const audioRef: AudioRef = {
        id: createAudioRefId('audio-1'),
        sizeBytes: 1024,
        durationMs: 5000,
        sampleRate: 44100,
      };

      useAudioStore.getState().addAudioRef(audioRef);
      useAudioStore.getState().startPlayback(audioRef.id);

      const result = selectCurrentlyPlayingAudio(useAudioStore.getState());
      expect(result).toEqual(audioRef);
    });

    it('should return null when nothing is playing', () => {
      const result = selectCurrentlyPlayingAudio(useAudioStore.getState());
      expect(result).toBeNull();
    });

    it('should return null when currently playing id does not exist in audioRefs', () => {
      useAudioStore.getState().startPlayback(createAudioRefId('non-existent'));

      const result = selectCurrentlyPlayingAudio(useAudioStore.getState());
      expect(result).toBeUndefined();
    });
  });

  describe('selectAllAudioRefs', () => {
    it('should return all audio refs as an array', () => {
      const audioRef1: AudioRef = {
        id: createAudioRefId('audio-1'),
        sizeBytes: 1024,
        durationMs: 5000,
        sampleRate: 44100,
      };

      const audioRef2: AudioRef = {
        id: createAudioRefId('audio-2'),
        sizeBytes: 2048,
        durationMs: 10000,
        sampleRate: 48000,
      };

      useAudioStore.getState().addAudioRef(audioRef1);
      useAudioStore.getState().addAudioRef(audioRef2);

      const result = selectAllAudioRefs(useAudioStore.getState());
      expect(result).toHaveLength(2);
      expect(result).toContainEqual(audioRef1);
      expect(result).toContainEqual(audioRef2);
    });

    it('should return empty array when no audio refs exist', () => {
      const result = selectAllAudioRefs(useAudioStore.getState());
      expect(result).toEqual([]);
    });
  });
});
