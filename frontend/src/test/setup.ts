import { afterEach, vi, beforeEach } from 'vitest';
import { cleanup } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';
import 'fake-indexeddb/auto';

// Clear VITE_API_URL to ensure tests use relative URLs
// This prevents the global env from affecting unit tests
delete import.meta.env.VITE_API_URL;

// Store imports for cleanup
import { useConversationStore } from '../stores/conversationStore';
import { useFeedbackStore } from '../stores/feedbackStore';
import { useMemoryStore } from '../stores/memoryStore';
import { useNotesStore } from '../stores/notesStore';
import { useConnectionStore } from '../stores/connectionStore';
import { useAudioStore } from '../stores/audioStore';
import { useDimensionStore } from '../stores/dimensionStore';
import { useServerInfoStore } from '../stores/serverInfoStore';

// Make React.act available globally for @testing-library/react
declare global {
  var IS_REACT_ACT_ENVIRONMENT: boolean;
}
globalThis.IS_REACT_ACT_ENVIRONMENT = true;

// Store original fetch to restore later
const originalFetch = global.fetch;

// Cleanup after each test
afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  vi.restoreAllMocks();

  // Restore original fetch
  global.fetch = originalFetch;

  // Reset all Zustand stores to prevent memory accumulation
  useConversationStore.getState().clearConversation();
  useFeedbackStore.getState().clearFeedback();
  useMemoryStore.getState().clearMemories();
  useNotesStore.getState().clearNotes();
  useConnectionStore.getState().clearConnection();
  useAudioStore.getState().clearAudioStore();
  useDimensionStore.getState().resetToBalanced();
  useServerInfoStore.getState().resetServerInfo();
});

// Mock window.matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
});

// Global mock config response for all tests
// Tests can override this by mocking fetch or the api module directly
const mockConfigResponse = {
  livekit_url: 'ws://localhost:7880',
  tts_enabled: true,
  asr_enabled: true,
  tts: {
    endpoint: '/v1/audio/speech',
    model: 'kokoro',
    default_voice: 'af_sarah',
    default_speed: 1.0,
    speed_min: 0.5,
    speed_max: 2.0,
    speed_step: 0.1,
    voices: [
      { id: 'af_sarah', name: 'Sarah', category: 'American Female' },
      { id: 'am_adam', name: 'Adam', category: 'American Male' },
      { id: 'af_nicole', name: 'Nicole', category: 'American Female' },
      { id: 'am_michael', name: 'Michael', category: 'American Male' },
    ],
  },
};

// Setup global fetch mock before each test
beforeEach(() => {
  // Mock fetch to intercept API calls
  global.fetch = vi.fn(async (input: RequestInfo | URL, _options?: RequestInit) => {
    const url = typeof input === 'string' ? input : input.toString();

    // If this is a config endpoint request, return mock config
    if (url.includes('/api/v1/config')) {
      return {
        ok: true,
        json: async () => mockConfigResponse,
        text: async () => JSON.stringify(mockConfigResponse),
        status: 200,
        statusText: 'OK',
      } as Response;
    }

    // Handle tool-uses votes endpoint (used by ComplexAddons)
    if (url.includes('/api/v1/tool-uses/') && url.includes('/votes')) {
      return {
        ok: true,
        json: async () => ({ upvotes: 0, downvotes: 0 }),
        text: async () => JSON.stringify({ upvotes: 0, downvotes: 0 }),
        status: 200,
        statusText: 'OK',
      } as Response;
    }

    // For relative URLs (starting with /), return a generic mock response
    // since originalFetch doesn't work with relative URLs in tests
    if (url.startsWith('/')) {
      return {
        ok: true,
        json: async () => ({}),
        text: async () => '',
        status: 200,
        statusText: 'OK',
      } as Response;
    }

    // For absolute URLs, try original fetch if available
    if (originalFetch) {
      return originalFetch(input, _options);
    }

    return {
      ok: true,
      json: async () => ({}),
      text: async () => '',
      status: 200,
      statusText: 'OK',
    } as Response;
  });
});
