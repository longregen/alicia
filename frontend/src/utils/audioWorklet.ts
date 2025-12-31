/**
 * AudioWorklet Utilities
 *
 * Helper functions and types for working with the AudioWorklet API,
 * specifically for the VAD-LiveKit bridge pipeline.
 */

/**
 * Message types sent from main thread to AudioWorklet processor
 */
export interface WorkletMessage {
  type: 'audio' | 'speech';
  data: Float32Array;
}

/**
 * Message types received from AudioWorklet processor (if needed)
 */
export interface WorkletResponse {
  type: 'status' | 'error';
  payload?: unknown;
}

/**
 * Options for loading an AudioWorklet module
 */
export interface LoadWorkletOptions {
  /** AudioContext to load the worklet into */
  context: AudioContext;
  /** URL/path to the worklet processor script */
  moduleURL: string;
  /** Processor name to instantiate */
  processorName: string;
}

/**
 * Result of loading an AudioWorklet module
 */
export interface LoadWorkletResult {
  /** The created AudioWorkletNode */
  node: AudioWorkletNode;
  /** Function to send typed messages to the worklet */
  postMessage: (message: WorkletMessage) => void;
  /** Function to cleanup and disconnect the worklet */
  cleanup: () => void;
}

/**
 * Load an AudioWorklet module and create a node
 *
 * @param options Configuration for loading the worklet
 * @returns Promise resolving to worklet node and helper functions
 *
 * @example
 * ```typescript
 * const { node, postMessage, cleanup } = await loadAudioWorklet({
 *   context: audioContext,
 *   moduleURL: '/vad-processor.js',
 *   processorName: 'vad-processor',
 * });
 *
 * // Send audio data to worklet
 * postMessage({ type: 'audio', data: audioFrame });
 *
 * // Cleanup when done
 * cleanup();
 * ```
 */
export async function loadAudioWorklet(
  options: LoadWorkletOptions
): Promise<LoadWorkletResult> {
  const { context, moduleURL, processorName } = options;

  try {
    // Load the worklet module into the AudioContext
    await context.audioWorklet.addModule(moduleURL);

    // Create the worklet node
    const node = new AudioWorkletNode(context, processorName);

    // Create typed postMessage helper
    const postMessage = (message: WorkletMessage): void => {
      try {
        // Transfer underlying ArrayBuffer for zero-copy performance
        const transferable = message.data.buffer;
        node.port.postMessage(message, [transferable]);
      } catch (error) {
        // If transfer fails (buffer already detached), clone the data
        console.warn('Failed to transfer audio buffer, cloning data:', error);
        node.port.postMessage({ ...message, data: new Float32Array(message.data) });
      }
    };

    // Create cleanup helper
    const cleanup = (): void => {
      try {
        node.disconnect();
        node.port.close();
      } catch (error) {
        console.warn('Error during worklet cleanup:', error);
      }
    };

    return { node, postMessage, cleanup };
  } catch (error) {
    console.error(`Failed to load AudioWorklet module ${moduleURL}:`, error);
    throw new Error(
      `AudioWorklet initialization failed: ${error instanceof Error ? error.message : 'Unknown error'}`
    );
  }
}

/**
 * Create a MediaStreamTrack from an AudioWorkletNode
 *
 * This connects the worklet to a MediaStreamAudioDestinationNode
 * and returns the resulting audio track.
 *
 * @param context AudioContext to use
 * @param workletNode The AudioWorkletNode to connect
 * @returns MediaStreamTrack that can be published to LiveKit
 *
 * @example
 * ```typescript
 * const { node } = await loadAudioWorklet({ ... });
 * const track = createTrackFromWorklet(audioContext, node);
 * await liveKit.publishAudioTrack(track);
 * ```
 */
export function createTrackFromWorklet(
  context: AudioContext,
  workletNode: AudioWorkletNode
): MediaStreamTrack {
  // Create destination node that provides a MediaStream
  const destination = context.createMediaStreamDestination();

  // Connect worklet to destination
  workletNode.connect(destination);

  // Get the audio track from the stream
  const tracks = destination.stream.getAudioTracks();
  if (tracks.length === 0) {
    throw new Error('No audio tracks available from destination node');
  }

  return tracks[0];
}

/**
 * Setup message handler for worklet responses
 *
 * @param workletNode The AudioWorkletNode to listen to
 * @param handler Callback for handling messages from the worklet
 * @returns Cleanup function to remove the listener
 *
 * @example
 * ```typescript
 * const removeListener = setupWorkletMessageHandler(node, (message) => {
 *   console.log('Worklet message:', message);
 * });
 *
 * // Later: remove listener
 * removeListener();
 * ```
 */
export function setupWorkletMessageHandler(
  workletNode: AudioWorkletNode,
  handler: (message: WorkletResponse) => void
): () => void {
  const messageHandler = (event: MessageEvent<WorkletResponse>) => {
    handler(event.data);
  };

  workletNode.port.addEventListener('message', messageHandler);
  workletNode.port.start();

  // Return cleanup function
  return () => {
    workletNode.port.removeEventListener('message', messageHandler);
  };
}

/**
 * Check if AudioWorklet API is supported in the current browser
 *
 * @returns true if AudioWorklet is supported
 */
export function isAudioWorkletSupported(): boolean {
  return typeof AudioContext !== 'undefined' && 'audioWorklet' in AudioContext.prototype;
}

/**
 * Create an AudioContext with optimal settings for VAD processing
 *
 * @param sampleRate Sample rate for the context (default: 16000 for VAD)
 * @returns Configured AudioContext
 *
 * @example
 * ```typescript
 * const context = createVADAudioContext();
 * // ... use context for VAD pipeline
 * ```
 */
export function createVADAudioContext(sampleRate = 16000): AudioContext {
  if (!isAudioWorkletSupported()) {
    throw new Error('AudioWorklet API is not supported in this browser');
  }

  return new AudioContext({
    sampleRate,
    latencyHint: 'interactive',
  });
}

/**
 * Error types that can occur during AudioWorklet operations
 */
export class AudioWorkletError extends Error {
  constructor(
    message: string,
    public readonly code: 'LOAD_FAILED' | 'NOT_SUPPORTED' | 'CONNECTION_FAILED'
  ) {
    super(message);
    this.name = 'AudioWorkletError';
  }
}
