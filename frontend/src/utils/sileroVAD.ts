import { MicrophoneStatus } from '../types/streaming';

// Types for Silero VAD
interface VADConfig {
  model?: string;
  positiveSpeechThreshold?: number;
  negativeSpeechThreshold?: number;
  minSpeechFrames?: number;
  preSpeechPadFrames?: number;
  baseAssetPath?: string;
  onnxWASMBasePath?: string;
  modelURL?: string;
  workletURL?: string;
  onFrameProcessed?: (probs: { notSpeech: number; isSpeech: number }, audioFrame: Float32Array) => void;
  onSpeechStart?: () => void;
  onSpeechEnd?: (audio: Float32Array) => void;
  onVADMisfire?: () => void;
}

interface MicVAD {
  start: () => void;
  pause: () => void;
  destroy: () => void;
}

// Declare global window properties for VAD
declare global {
  interface Window {
    vad?: {
      MicVAD: {
        new: (config: VADConfig) => Promise<MicVAD>;
      };
    };
    ort?: {
      env: {
        wasm: {
          wasmPaths: string;
          numThreads: number;
          simd: boolean;
        };
      };
    };
  }
}

export interface SileroVADCallbacks {
  onStatusChange?: (status: MicrophoneStatus) => void;
  onSpeechProbability?: (probability: number, isSpeaking: boolean) => void;
  onFrameProcessed?: (audioFrame: Float32Array) => void;
  onSpeechStart?: () => void;
  onSpeechEnd?: (audioData: Float32Array) => void;
  onError?: (error: Error) => void;
}

export class SileroVADManager {
  private vadInstance: MicVAD | null = null;
  private isInitialized = false;
  private callbacks: SileroVADCallbacks;
  private currentStatus = MicrophoneStatus.Inactive;

  constructor(callbacks: SileroVADCallbacks = {}) {
    this.callbacks = callbacks;
  }

  /**
   * Load required scripts for Silero VAD
   */
  async loadScripts(): Promise<void> {
    // Check if already loaded
    if (window.vad && window.ort) {
      return;
    }

    try {
      // Load ONNX Runtime
      await this.loadScript('/js/lib/ort.js');

      // Configure ONNX Runtime
      if (window.ort) {
        window.ort.env.wasm.wasmPaths = '/onnx/';
        window.ort.env.wasm.numThreads = 2;
        window.ort.env.wasm.simd = true;
      }

      // Load VAD bundle
      await this.loadScript('/js/lib/vad.bundle.min.js');

      // Configure VAD globally
      if (window.vad) {
        (window as Window & { VAD_CONFIG?: { baseAssetPath: string; onnxWASMBasePath: string } }).VAD_CONFIG = {
          baseAssetPath: '/models/',
          onnxWASMBasePath: '/onnx/'
        };
      }
    } catch (error) {
      console.error('Failed to load Silero VAD scripts:', error);
      throw error;
    }
  }

  /**
   * Initialize Silero VAD
   */
  async initialize(): Promise<void> {
    if (this.isInitialized) {
      return;
    }

    try {
      this.updateStatus(MicrophoneStatus.RequestingPermission);

      // Load scripts if not already loaded
      await this.loadScripts();

      // Check if VAD is available
      if (!window.vad) {
        throw new Error('VAD library not loaded');
      }

      // Initialize VAD with configuration

      // The VAD config should be set globally
      const vadConfig = (window as Window & { VAD_CONFIG?: { baseAssetPath: string; onnxWASMBasePath: string } }).VAD_CONFIG || {
        baseAssetPath: '/models/',
        onnxWASMBasePath: '/onnx/'
      };

      this.vadInstance = await window.vad.MicVAD.new({
        positiveSpeechThreshold: 0.5,
        negativeSpeechThreshold: 0.2,
        minSpeechFrames: 4,
        preSpeechPadFrames: 0,
        model: 'v5', // Explicitly specify v5 model
        ...vadConfig,
        workletURL: vadConfig.baseAssetPath + 'vad.worklet.bundle.min.js',
        modelURL: vadConfig.baseAssetPath + 'silero_vad_v5.onnx',

        // Event handlers
        onFrameProcessed: (probs: { notSpeech: number; isSpeech: number }, audioFrame: Float32Array) => {
          // The VAD returns notSpeech and isSpeech probabilities
          const speechProbability = probs.isSpeech;
          const isSpeaking = speechProbability > 0.5;
          this.callbacks.onSpeechProbability?.(speechProbability, isSpeaking);
          this.callbacks.onFrameProcessed?.(audioFrame);
        },

        onSpeechStart: () => {
          this.callbacks.onSpeechStart?.();
        },

        onSpeechEnd: (audio: Float32Array) => {
          this.callbacks.onSpeechEnd?.(audio);
        },

        onVADMisfire: () => {
          // Misfire detected - no action needed
        }
      });

      // VAD initialized and ready - call start() to begin recording
      this.isInitialized = true;
      this.updateStatus(MicrophoneStatus.Active);

    } catch (error) {
      console.error('Failed to initialize Silero VAD:', error);
      this.updateStatus(MicrophoneStatus.Error);
      this.callbacks.onError?.(error as Error);
      throw error;
    }
  }

  /**
   * Start VAD processing
   */
  async start(): Promise<void> {
    if (!this.isInitialized) {
      await this.initialize();
    }

    if (this.vadInstance) {
      // Start VAD recording (audio context initialized during initialize())
      this.vadInstance.start();
      this.updateStatus(MicrophoneStatus.Recording);
    }
  }

  /**
   * Pause VAD processing (can be resumed with start())
   * Status remains Active as the system is ready to resume
   */
  stop(): void {
    if (this.vadInstance) {
      this.vadInstance.pause();
      this.updateStatus(MicrophoneStatus.Active);
    }
  }

  /**
   * Cleanup and destroy VAD instance
   */
  destroy(): void {
    if (this.vadInstance) {
      this.vadInstance.destroy();
      this.vadInstance = null;
      this.isInitialized = false;
      this.updateStatus(MicrophoneStatus.Inactive);
    }
  }

  /**
   * Get current status
   */
  getStatus(): MicrophoneStatus {
    return this.currentStatus;
  }

  /**
   * Update callbacks
   */
  updateCallbacks(newCallbacks: SileroVADCallbacks): void {
    this.callbacks = { ...this.callbacks, ...newCallbacks };
  }

  /**
   * Update status and notify callback
   */
  private updateStatus(status: MicrophoneStatus): void {
    this.currentStatus = status;
    this.callbacks.onStatusChange?.(status);
  }

  /**
   * Load a script dynamically
   */
  private loadScript(src: string): Promise<void> {
    return new Promise((resolve, reject) => {
      const script = document.createElement('script');
      script.src = src;
      script.onload = () => resolve();
      script.onerror = () => reject(new Error(`Failed to load script: ${src}`));
      document.head.appendChild(script);
    });
  }
}
