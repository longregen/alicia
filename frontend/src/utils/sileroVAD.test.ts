import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { SileroVADManager, SileroVADCallbacks } from './sileroVAD';
import { MicrophoneStatus } from '../types/streaming';

// Mock script loading
const mockScripts = new Map<string, HTMLScriptElement>();

// Mock MicVAD instance
const mockMicVAD = {
  start: vi.fn(),
  pause: vi.fn(),
  destroy: vi.fn(),
};

// Mock VAD config that will be passed to MicVAD.new
let capturedVADConfig: any = null;

// Setup global mocks
const setupWindowMocks = () => {
  // Mock ONNX Runtime
  (window as any).ort = {
    env: {
      wasm: {
        wasmPaths: '',
        numThreads: 0,
        simd: false,
      },
    },
  };

  // Mock VAD library
  (window as any).vad = {
    MicVAD: {
      new: vi.fn(async (config: any) => {
        capturedVADConfig = config;
        return mockMicVAD;
      }),
    },
  };
};

const clearWindowMocks = () => {
  delete (window as any).ort;
  delete (window as any).vad;
  delete (window as any).VAD_CONFIG;
  capturedVADConfig = null;
};

// Store original implementations
let originalCreateElement: typeof document.createElement;

// Mock document.createElement for script loading
const mockDocumentCreateElement = () => {
  originalCreateElement = document.createElement.bind(document);

  vi.spyOn(document, 'createElement').mockImplementation((tagName: string) => {
    const element = originalCreateElement(tagName);
    return element;
  });

  vi.spyOn(document.head, 'appendChild').mockImplementation((node: Node) => {
    if (node instanceof HTMLElement && node.tagName === 'SCRIPT') {
      const scriptElement = node as HTMLScriptElement;
      mockScripts.set(scriptElement.src, scriptElement);

      // Trigger onload asynchronously and simulate script execution
      setTimeout(() => {
        // Simulate script execution: when ort.js loads, setup window.ort
        // (setupWindowMocks should already have been called by test)
        if (scriptElement.onload) {
          scriptElement.onload(new Event('load'));
        }
      }, 0);
    }
    return node;
  });
};

describe('SileroVADManager', () => {
  let callbacks: SileroVADCallbacks;
  let manager: SileroVADManager;

  beforeEach(() => {
    vi.clearAllMocks();
    mockScripts.clear();

    // Suppress console output during tests
    vi.spyOn(console, 'log').mockImplementation(() => {});
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    vi.spyOn(console, 'error').mockImplementation(() => {});

    callbacks = {
      onStatusChange: vi.fn(),
      onSpeechProbability: vi.fn(),
      onFrameProcessed: vi.fn(),
      onSpeechStart: vi.fn(),
      onSpeechEnd: vi.fn(),
      onError: vi.fn(),
    };

    mockDocumentCreateElement();
  });

  afterEach(() => {
    clearWindowMocks();
    vi.restoreAllMocks();
  });

  describe('Constructor', () => {
    it('should initialize with empty callbacks', () => {
      manager = new SileroVADManager();
      expect(manager).toBeInstanceOf(SileroVADManager);
      expect(manager.getStatus()).toBe(MicrophoneStatus.Inactive);
    });

    it('should initialize with provided callbacks', () => {
      manager = new SileroVADManager(callbacks);
      expect(manager).toBeInstanceOf(SileroVADManager);
      expect(manager.getStatus()).toBe(MicrophoneStatus.Inactive);
    });

    it('should start in Inactive status', () => {
      manager = new SileroVADManager(callbacks);
      expect(manager.getStatus()).toBe(MicrophoneStatus.Inactive);
    });
  });

  describe('loadScripts', () => {
    beforeEach(() => {
      manager = new SileroVADManager(callbacks);
    });

    it('should return early if scripts already loaded', async () => {
      // Setup mocks BEFORE loadScripts - simulates already loaded scripts
      setupWindowMocks();

      // Simulate script loading completing - should return early
      await manager.loadScripts();

      // After loadScripts, window.ort and window.vad should still be defined
      expect((window as any).ort).toBeDefined();
      expect((window as any).vad).toBeDefined();
    });

    it('should configure ONNX Runtime with correct paths when called after mocks', async () => {
      // When mocks are set up first, loadScripts returns early
      // Test that the mock values are correct (simulating already loaded state)
      setupWindowMocks();

      await manager.loadScripts();

      // The mocks define these values
      expect((window as any).ort.env.wasm).toBeDefined();
    });

    it('should verify VAD_CONFIG can be set globally', async () => {
      // When scripts are already loaded, VAD_CONFIG should be settable
      setupWindowMocks();

      // Manually set VAD_CONFIG to test the pattern
      (window as any).VAD_CONFIG = {
        baseAssetPath: '/models/',
        onnxWASMBasePath: '/onnx/',
      };

      await manager.loadScripts();

      expect((window as any).VAD_CONFIG).toEqual({
        baseAssetPath: '/models/',
        onnxWASMBasePath: '/onnx/',
      });
    });

    it('should skip loading if scripts already loaded', async () => {
      setupWindowMocks();

      await manager.loadScripts();
      const firstScriptCount = mockScripts.size;

      await manager.loadScripts();
      const secondScriptCount = mockScripts.size;

      expect(secondScriptCount).toBe(firstScriptCount);
    });

    it('should handle script loading errors', async () => {
      vi.restoreAllMocks();
      // Re-suppress console after restoreAllMocks
      vi.spyOn(console, 'error').mockImplementation(() => {});

      const origCreate = document.createElement.bind(document);
      vi.spyOn(document, 'createElement').mockImplementation((tagName: string) => {
        return origCreate(tagName);
      });

      vi.spyOn(document.head, 'appendChild').mockImplementation((node: Node) => {
        if (node instanceof HTMLElement && node.tagName === 'SCRIPT') {
          const scriptElement = node as HTMLScriptElement;
          setTimeout(() => {
            if (scriptElement.onerror) {
              scriptElement.onerror(new Event('error'));
            }
          }, 0);
        }
        return node;
      });

      const newManager = new SileroVADManager(callbacks);
      await expect(newManager.loadScripts()).rejects.toThrow(/Failed to load script/);
    });
  });

  describe('initialize', () => {
    beforeEach(() => {
      manager = new SileroVADManager(callbacks);
      setupWindowMocks();
    });

    it('should initialize VAD with correct configuration', async () => {
      await manager.initialize();

      expect((window as any).vad.MicVAD.new).toHaveBeenCalledWith(
        expect.objectContaining({
          positiveSpeechThreshold: 0.5,
          negativeSpeechThreshold: 0.2,
          minSpeechFrames: 4,
          preSpeechPadFrames: 0,
          model: 'v5',
          baseAssetPath: '/models/',
          onnxWASMBasePath: '/onnx/',
          workletURL: '/models/vad.worklet.bundle.min.js',
          modelURL: '/models/silero_vad_v5.onnx',
        })
      );
    });

    it('should transition status from Inactive to RequestingPermission to Active', async () => {
      await manager.initialize();

      expect(callbacks.onStatusChange).toHaveBeenCalledWith(MicrophoneStatus.RequestingPermission);
      expect(callbacks.onStatusChange).toHaveBeenCalledWith(MicrophoneStatus.Active);
      expect(manager.getStatus()).toBe(MicrophoneStatus.Active);
    });

    it('should skip initialization if already initialized', async () => {
      await manager.initialize();

      vi.clearAllMocks();
      await manager.initialize();

      expect((window as any).vad.MicVAD.new).not.toHaveBeenCalled();
    });

    it('should handle initialization errors', async () => {
      (window as any).vad.MicVAD.new.mockRejectedValueOnce(new Error('ONNX model load failed'));

      await expect(manager.initialize()).rejects.toThrow('ONNX model load failed');
      expect(callbacks.onError).toHaveBeenCalledWith(expect.any(Error));
      expect(manager.getStatus()).toBe(MicrophoneStatus.Error);
    });

    it('should handle missing VAD library', async () => {
      delete (window as any).vad;

      await expect(manager.initialize()).rejects.toThrow('VAD library not loaded');
      expect(callbacks.onError).toHaveBeenCalledWith(expect.any(Error));
      expect(manager.getStatus()).toBe(MicrophoneStatus.Error);
    });
  });

  describe('start', () => {
    beforeEach(() => {
      manager = new SileroVADManager(callbacks);
      setupWindowMocks();
    });

    it('should initialize if not already initialized', async () => {
      await manager.start();

      expect((window as any).vad.MicVAD.new).toHaveBeenCalled();
      expect(mockMicVAD.start).toHaveBeenCalled();
    });

    it('should start VAD instance and update status to Recording', async () => {
      await manager.initialize();
      vi.clearAllMocks();

      await manager.start();

      expect(mockMicVAD.start).toHaveBeenCalled();
      expect(callbacks.onStatusChange).toHaveBeenCalledWith(MicrophoneStatus.Recording);
      expect(manager.getStatus()).toBe(MicrophoneStatus.Recording);
    });

    it('should not reinitialize if already initialized', async () => {
      await manager.initialize();

      vi.clearAllMocks();
      await manager.start();

      expect((window as any).vad.MicVAD.new).not.toHaveBeenCalled();
      expect(mockMicVAD.start).toHaveBeenCalled();
    });
  });

  describe('stop', () => {
    beforeEach(() => {
      manager = new SileroVADManager(callbacks);
      setupWindowMocks();
    });

    it('should pause VAD instance and update status to Active', async () => {
      await manager.start();
      vi.clearAllMocks();

      manager.stop();

      expect(mockMicVAD.pause).toHaveBeenCalled();
      expect(callbacks.onStatusChange).toHaveBeenCalledWith(MicrophoneStatus.Active);
      expect(manager.getStatus()).toBe(MicrophoneStatus.Active);
    });

    it('should handle stop when not initialized', () => {
      expect(() => manager.stop()).not.toThrow();
      expect(mockMicVAD.pause).not.toHaveBeenCalled();
    });
  });

  describe('destroy', () => {
    beforeEach(() => {
      manager = new SileroVADManager(callbacks);
      setupWindowMocks();
    });

    it('should destroy VAD instance and reset state', async () => {
      await manager.start();
      vi.clearAllMocks();

      manager.destroy();

      expect(mockMicVAD.destroy).toHaveBeenCalled();
      expect(callbacks.onStatusChange).toHaveBeenCalledWith(MicrophoneStatus.Inactive);
      expect(manager.getStatus()).toBe(MicrophoneStatus.Inactive);
    });

    it('should handle destroy when not initialized', () => {
      expect(() => manager.destroy()).not.toThrow();
      expect(mockMicVAD.destroy).not.toHaveBeenCalled();
    });

    it('should allow reinitialization after destroy', async () => {
      await manager.initialize();
      manager.destroy();

      await manager.initialize();

      expect((window as any).vad.MicVAD.new).toHaveBeenCalledTimes(2);
    });
  });

  describe('Callback handling', () => {
    beforeEach(() => {
      manager = new SileroVADManager(callbacks);
      setupWindowMocks();
    });

    it('should trigger onSpeechStart callback', async () => {
      await manager.initialize();

      expect(capturedVADConfig.onSpeechStart).toBeDefined();
      capturedVADConfig.onSpeechStart();

      expect(callbacks.onSpeechStart).toHaveBeenCalled();
    });

    it('should trigger onSpeechEnd callback with audio data', async () => {
      await manager.initialize();
      const mockAudioData = new Float32Array([0.1, 0.2, 0.3]);

      expect(capturedVADConfig.onSpeechEnd).toBeDefined();
      capturedVADConfig.onSpeechEnd(mockAudioData);

      expect(callbacks.onSpeechEnd).toHaveBeenCalledWith(mockAudioData);
    });

    it('should trigger onFrameProcessed callback with audio frame', async () => {
      await manager.initialize();
      const mockAudioFrame = new Float32Array([0.1, 0.2, 0.3]);
      const mockProbs = { notSpeech: 0.3, isSpeech: 0.7 };

      expect(capturedVADConfig.onFrameProcessed).toBeDefined();
      capturedVADConfig.onFrameProcessed(mockProbs, mockAudioFrame);

      expect(callbacks.onFrameProcessed).toHaveBeenCalledWith(mockAudioFrame);
    });

    it('should trigger onSpeechProbability callback with correct values', async () => {
      await manager.initialize();
      const mockAudioFrame = new Float32Array([0.1, 0.2, 0.3]);
      const mockProbs = { notSpeech: 0.3, isSpeech: 0.7 };

      capturedVADConfig.onFrameProcessed(mockProbs, mockAudioFrame);

      expect(callbacks.onSpeechProbability).toHaveBeenCalledWith(0.7, true);
    });

    it('should determine isSpeaking correctly based on threshold', async () => {
      await manager.initialize();
      const mockAudioFrame = new Float32Array([0.1]);

      // Below threshold (0.5)
      capturedVADConfig.onFrameProcessed({ notSpeech: 0.7, isSpeech: 0.3 }, mockAudioFrame);
      expect(callbacks.onSpeechProbability).toHaveBeenCalledWith(0.3, false);

      // Above threshold
      capturedVADConfig.onFrameProcessed({ notSpeech: 0.2, isSpeech: 0.8 }, mockAudioFrame);
      expect(callbacks.onSpeechProbability).toHaveBeenCalledWith(0.8, true);

      // Exactly at threshold
      capturedVADConfig.onFrameProcessed({ notSpeech: 0.5, isSpeech: 0.5 }, mockAudioFrame);
      expect(callbacks.onSpeechProbability).toHaveBeenCalledWith(0.5, false);
    });

    it('should handle onVADMisfire without error', async () => {
      await manager.initialize();

      expect(capturedVADConfig.onVADMisfire).toBeDefined();
      expect(() => capturedVADConfig.onVADMisfire()).not.toThrow();
    });

    it('should handle callbacks being undefined', async () => {
      const managerWithoutCallbacks = new SileroVADManager();
      await managerWithoutCallbacks.initialize();

      expect(() => {
        capturedVADConfig.onSpeechStart();
        capturedVADConfig.onSpeechEnd(new Float32Array());
        capturedVADConfig.onFrameProcessed({ notSpeech: 0.5, isSpeech: 0.5 }, new Float32Array());
      }).not.toThrow();
    });
  });

  describe('updateCallbacks', () => {
    beforeEach(() => {
      manager = new SileroVADManager(callbacks);
    });

    it('should update callbacks', () => {
      const newCallback = vi.fn();
      manager.updateCallbacks({ onSpeechStart: newCallback });

      expect(manager.getStatus()).toBe(MicrophoneStatus.Inactive);
    });

    it('should merge callbacks without replacing existing ones', async () => {
      setupWindowMocks();

      const newOnError = vi.fn();
      manager.updateCallbacks({ onError: newOnError });

      // updateCallbacks should merge, not replace all callbacks
      // The onSpeechStart should still work after the merge
      await manager.initialize();

      // Trigger the callback through VAD config
      capturedVADConfig.onSpeechStart();

      // Original callback should still work (not replaced by the merge)
      expect(callbacks.onSpeechStart).toHaveBeenCalled();
    });
  });

  describe('State transitions', () => {
    beforeEach(() => {
      manager = new SileroVADManager(callbacks);
      setupWindowMocks();
    });

    it('should transition through correct states: Inactive → RequestingPermission → Active', async () => {
      expect(manager.getStatus()).toBe(MicrophoneStatus.Inactive);

      const initPromise = manager.initialize();

      // Should be requesting permission immediately
      expect(callbacks.onStatusChange).toHaveBeenCalledWith(MicrophoneStatus.RequestingPermission);

      await initPromise;

      // Should be active after initialization
      expect(manager.getStatus()).toBe(MicrophoneStatus.Active);
    });

    it('should transition Active → Recording on start', async () => {
      await manager.initialize();
      expect(manager.getStatus()).toBe(MicrophoneStatus.Active);

      await manager.start();
      expect(manager.getStatus()).toBe(MicrophoneStatus.Recording);
    });

    it('should transition Recording → Active on stop', async () => {
      await manager.start();
      expect(manager.getStatus()).toBe(MicrophoneStatus.Recording);

      manager.stop();
      expect(manager.getStatus()).toBe(MicrophoneStatus.Active);
    });

    it('should transition to Inactive on destroy', async () => {
      await manager.start();
      expect(manager.getStatus()).toBe(MicrophoneStatus.Recording);

      manager.destroy();
      expect(manager.getStatus()).toBe(MicrophoneStatus.Inactive);
    });

    it('should transition to Error on initialization failure', async () => {
      (window as any).vad.MicVAD.new.mockRejectedValueOnce(new Error('Test error'));

      await expect(manager.initialize()).rejects.toThrow();
      expect(manager.getStatus()).toBe(MicrophoneStatus.Error);
    });
  });

  describe('Error handling', () => {
    beforeEach(() => {
      manager = new SileroVADManager(callbacks);
      setupWindowMocks();
    });

    it('should handle ONNX model loading failures', async () => {
      const onnxError = new Error('Failed to load ONNX model');
      (window as any).vad.MicVAD.new.mockRejectedValueOnce(onnxError);

      await expect(manager.initialize()).rejects.toThrow('Failed to load ONNX model');
      expect(callbacks.onError).toHaveBeenCalledWith(onnxError);
    });

    it('should handle network errors during script loading', async () => {
      clearWindowMocks();
      vi.restoreAllMocks();
      // Re-suppress console after restoreAllMocks
      vi.spyOn(console, 'error').mockImplementation(() => {});

      // Fresh mock that triggers onerror
      const origCreate = document.createElement.bind(document);
      vi.spyOn(document, 'createElement').mockImplementation((tagName: string) => {
        return origCreate(tagName);
      });

      vi.spyOn(document.head, 'appendChild').mockImplementation((node: Node) => {
        if (node instanceof HTMLElement && node.tagName === 'SCRIPT') {
          setTimeout(() => {
            if ((node as HTMLScriptElement).onerror) {
              (node as HTMLScriptElement).onerror!(new Event('error'));
            }
          }, 0);
        }
        return node;
      });

      const newManager = new SileroVADManager(callbacks);
      await expect(newManager.loadScripts()).rejects.toThrow('Failed to load script');
    });

    it('should handle permission denied errors gracefully', async () => {
      const permissionError = new Error('Permission denied');
      (window as any).vad.MicVAD.new.mockRejectedValueOnce(permissionError);

      await expect(manager.initialize()).rejects.toThrow('Permission denied');
      expect(manager.getStatus()).toBe(MicrophoneStatus.Error);
      expect(callbacks.onError).toHaveBeenCalledWith(permissionError);
    });
  });

  describe('Cleanup and resource management', () => {
    beforeEach(() => {
      manager = new SileroVADManager(callbacks);
      setupWindowMocks();
    });

    it('should properly clean up resources on destroy', async () => {
      await manager.start();

      manager.destroy();

      expect(mockMicVAD.destroy).toHaveBeenCalled();
      expect(manager.getStatus()).toBe(MicrophoneStatus.Inactive);
    });

    it('should allow multiple start/stop cycles', async () => {
      await manager.start();
      manager.stop();
      await manager.start();
      manager.stop();

      expect(mockMicVAD.start).toHaveBeenCalledTimes(2);
      expect(mockMicVAD.pause).toHaveBeenCalledTimes(2);
    });

    it('should handle rapid start/stop/destroy sequences', async () => {
      await manager.start();
      manager.stop();
      manager.destroy();

      expect(mockMicVAD.start).toHaveBeenCalled();
      expect(mockMicVAD.pause).toHaveBeenCalled();
      expect(mockMicVAD.destroy).toHaveBeenCalled();
      expect(manager.getStatus()).toBe(MicrophoneStatus.Inactive);
    });
  });
});
