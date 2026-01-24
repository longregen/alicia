/**
 * VAD Audio Processor Worklet
 *
 * This AudioWorklet processor receives Float32Array audio frames from
 * Silero VAD (at 16kHz) and upsamples them to the output sample rate
 * (typically 48kHz) for LiveKit compatibility.
 *
 * Messages from main thread:
 * - { type: 'audio', data: Float32Array } - Individual audio frame
 * - { type: 'speech', data: Float32Array } - Complete speech segment
 */

class VADProcessor extends AudioWorkletProcessor {
  constructor(options) {
    super();

    // Get sample rates from processor options
    const processorOptions = options.processorOptions || {};
    this.inputSampleRate = processorOptions.inputSampleRate || 16000;
    this.outputSampleRate = processorOptions.outputSampleRate || sampleRate;

    // Calculate resampling ratio
    this.resampleRatio = this.outputSampleRate / this.inputSampleRate;

    // Resampled buffer (at output sample rate)
    this.outputBuffer = [];
    this.outputBufferSize = 0;

    // Listen for messages from main thread
    this.port.onmessage = (event) => {
      const { type, data } = event.data;

      if (type === 'audio' || type === 'speech') {
        // Add audio data to buffer and resample
        if (data && data.length > 0) {
          const resampled = this.resample(data);
          this.outputBuffer.push(resampled);
          this.outputBufferSize += resampled.length;
        }
      }
    };
  }

  /**
   * Resample audio from input sample rate to output sample rate
   * Uses linear interpolation for simplicity and low latency
   *
   * @param {Float32Array} input - Audio at input sample rate
   * @returns {Float32Array} - Audio at output sample rate
   */
  resample(input) {
    if (this.resampleRatio === 1) {
      return new Float32Array(input);
    }

    const outputLength = Math.ceil(input.length * this.resampleRatio);
    const output = new Float32Array(outputLength);

    for (let i = 0; i < outputLength; i++) {
      const inputPos = i / this.resampleRatio;
      const inputIndex = Math.floor(inputPos);
      const fraction = inputPos - inputIndex;

      const sample1 = input[inputIndex] || 0;
      const sample2 = input[inputIndex + 1] !== undefined ? input[inputIndex + 1] : sample1;

      // Linear interpolation
      output[i] = sample1 + fraction * (sample2 - sample1);
    }

    return output;
  }

  /**
   * Process audio - called by the audio rendering thread
   *
   * @param {Float32Array[][]} inputs - Input audio (not used)
   * @param {Float32Array[][]} outputs - Output audio to write to
   * @param {Object} parameters - Audio parameters (not used)
   * @returns {boolean} - true to keep processor alive
   */
  process(inputs, outputs, parameters) {
    const output = outputs[0];

    if (!output || !output[0]) {
      return true;
    }

    const outputChannel = output[0];
    const frameSize = outputChannel.length;
    let written = 0;

    // Fill output buffer from our resampled buffer
    while (written < frameSize && this.outputBuffer.length > 0) {
      const chunk = this.outputBuffer[0];
      const remaining = frameSize - written;
      const toWrite = Math.min(chunk.length, remaining);

      outputChannel.set(chunk.subarray(0, toWrite), written);
      written += toWrite;

      if (toWrite >= chunk.length) {
        this.outputBuffer.shift();
        this.outputBufferSize -= chunk.length;
      } else {
        this.outputBuffer[0] = chunk.subarray(toWrite);
        this.outputBufferSize -= toWrite;
      }
    }

    // Fill remaining with silence
    if (written < frameSize) {
      outputChannel.fill(0, written);
    }

    return true;
  }
}

registerProcessor('vad-processor', VADProcessor);
