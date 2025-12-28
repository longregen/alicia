/**
 * VAD Audio Processor Worklet
 *
 * This AudioWorklet processor receives Float32Array audio frames from
 * Silero VAD and converts them into audio stream output for LiveKit.
 *
 * Messages from main thread:
 * - { type: 'audio', data: Float32Array } - Individual audio frame
 * - { type: 'speech', data: Float32Array } - Complete speech segment
 */

class VADProcessor extends AudioWorkletProcessor {
  constructor() {
    super();

    // Buffer to hold incoming audio data
    this.buffer = [];
    this.bufferSize = 0;

    // Listen for messages from main thread
    this.port.onmessage = (event) => {
      const { type, data } = event.data;

      if (type === 'audio' || type === 'speech') {
        // Add audio data to buffer
        if (data && data.length > 0) {
          this.buffer.push(new Float32Array(data));
          this.bufferSize += data.length;
        }
      }
    };
  }

  /**
   * Process audio - called by the audio rendering thread
   * This is where we convert buffered Float32Array data to output
   *
   * @param {Float32Array[][]} inputs - Input audio (not used, we use our buffer)
   * @param {Float32Array[][]} outputs - Output audio to write to
   * @param {Object} parameters - Audio parameters (not used)
   * @returns {boolean} - true to keep processor alive
   */
  process(inputs, outputs, parameters) {
    const output = outputs[0];

    // If no output channel, skip
    if (!output || !output[0]) {
      return true;
    }

    const outputChannel = output[0];
    const frameSize = outputChannel.length;
    let written = 0;

    // Fill output buffer from our internal buffer
    while (written < frameSize && this.buffer.length > 0) {
      const chunk = this.buffer[0];
      const remaining = frameSize - written;
      const toWrite = Math.min(chunk.length, remaining);

      // Copy data to output
      outputChannel.set(chunk.subarray(0, toWrite), written);
      written += toWrite;

      // Update buffer
      if (toWrite >= chunk.length) {
        // Consumed entire chunk
        this.buffer.shift();
        this.bufferSize -= chunk.length;
      } else {
        // Partial chunk consumed, keep remainder
        this.buffer[0] = chunk.subarray(toWrite);
        this.bufferSize -= toWrite;
      }
    }

    // Fill remaining output with silence if needed
    if (written < frameSize) {
      outputChannel.fill(0, written);
    }

    // Keep processor alive
    return true;
  }
}

// Register the processor
registerProcessor('vad-processor', VADProcessor);
