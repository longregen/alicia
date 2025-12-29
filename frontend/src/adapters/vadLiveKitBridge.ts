/**
 * VAD-LiveKit Bridge Adapter
 *
 * Converts Float32Array audio frames from Silero VAD to a MediaStreamTrack
 * that can be published to LiveKit for server-side transcription.
 *
 * Architecture:
 * Microphone → Silero VAD (Float32Array frames)
 *           → AudioWorkletNode (process & forward)
 *           → MediaStreamAudioDestinationNode
 *           → MediaStreamTrack
 *           → LiveKit publishTrack()
 */

/**
 * VADLiveKitBridge manages the audio pipeline from VAD to LiveKit
 */
export class VADLiveKitBridge {
  private audioContext: AudioContext | null = null;
  private destination: MediaStreamAudioDestinationNode | null = null;
  private workletNode: AudioWorkletNode | null = null;
  private isInitialized = false;

  /**
   * Initialize the audio pipeline and return a MediaStreamTrack
   * ready to be published to LiveKit
   */
  async initialize(): Promise<MediaStreamTrack> {
    if (this.isInitialized) {
      throw new Error('VADLiveKitBridge already initialized');
    }

    try {
      // Create AudioContext with 16kHz sample rate (matches VAD output)
      this.audioContext = new AudioContext({ sampleRate: 16000 });

      // Create destination node that provides a MediaStream
      this.destination = this.audioContext.createMediaStreamDestination();

      // Load and create AudioWorklet for efficient Float32Array → MediaStream conversion
      await this.audioContext.audioWorklet.addModule('/vad-processor.js');
      this.workletNode = new AudioWorkletNode(this.audioContext, 'vad-processor');

      // Connect worklet to destination
      this.workletNode.connect(this.destination);

      this.isInitialized = true;

      // Return the audio track from the destination stream
      const tracks = this.destination.stream.getAudioTracks();
      if (tracks.length === 0) {
        throw new Error('No audio tracks available from destination');
      }

      return tracks[0];
    } catch (error) {
      console.error('Failed to initialize VADLiveKitBridge:', error);
      this.cleanup();
      throw error;
    }
  }

  /**
   * Push an audio frame from VAD to the LiveKit stream
   * Called by Silero VAD onFrameProcessed callback
   *
   * @param audioData Float32Array audio samples from VAD
   */
  pushAudioFrame(audioData: Float32Array): void {
    if (!this.isInitialized || !this.workletNode) {
      console.warn('VADLiveKitBridge not initialized, ignoring audio frame');
      return;
    }

    try {
      // Send audio data to the AudioWorklet processor
      this.workletNode.port.postMessage({
        type: 'audio',
        data: audioData,
      });
    } catch (error) {
      console.error('Failed to push audio frame:', error);
    }
  }

  /**
   * Push a complete speech segment from VAD to the LiveKit stream
   * Called by Silero VAD onSpeechEnd callback
   *
   * @param audioData Float32Array complete speech segment
   */
  pushSpeechSegment(audioData: Float32Array): void {
    if (!this.isInitialized || !this.workletNode) {
      console.warn('VADLiveKitBridge not initialized, ignoring speech segment');
      return;
    }

    try {
      // Send speech segment to the AudioWorklet processor
      this.workletNode.port.postMessage({
        type: 'speech',
        data: audioData,
      });
    } catch (error) {
      console.error('Failed to push speech segment:', error);
    }
  }

  /**
   * Clean up resources and release the audio pipeline
   */
  cleanup(): void {
    try {
      if (this.workletNode) {
        this.workletNode.disconnect();
        this.workletNode.port.close();
        this.workletNode = null;
      }

      if (this.destination) {
        this.destination.disconnect();
        this.destination = null;
      }

      if (this.audioContext) {
        this.audioContext.close();
        this.audioContext = null;
      }

      this.isInitialized = false;
    } catch (error) {
      console.error('Error during VADLiveKitBridge cleanup:', error);
    }
  }

  /**
   * Check if the bridge is initialized and ready to accept audio
   */
  get initialized(): boolean {
    return this.isInitialized;
  }
}
