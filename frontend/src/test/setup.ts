import { afterEach, vi, beforeEach } from 'vitest';
import { cleanup } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';

// Make React.act available globally for @testing-library/react
declare global {
  var IS_REACT_ACT_ENVIRONMENT: boolean;
}
globalThis.IS_REACT_ACT_ENVIRONMENT = true;

// Cleanup after each test
afterEach(() => {
  cleanup();
  vi.clearAllMocks();
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
  const originalFetch = global.fetch;

  // Mock fetch to intercept /api/v1/config calls
  global.fetch = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
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

    // For other requests, use the original fetch if available
    // or return a basic mock response
    if (originalFetch) {
      return originalFetch(input, options);
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
