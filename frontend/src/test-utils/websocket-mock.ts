import { unpack } from 'msgpackr';

export interface WebSocketMessage {
  data: ArrayBuffer;
  timestamp: number;
}

export interface MockWebSocketOptions {
  autoConnect?: boolean;
  latency?: number;
  onSend?: (data: ArrayBuffer) => void;
}

/**
 * Mock WebSocket implementation for testing
 * Supports message recording, playback, and simulated responses
 */
export class MockWebSocket {
  public url: string;
  public readyState: number = WebSocket.CONNECTING;
  public binaryType: BinaryType = 'arraybuffer';

  public onopen: ((event: Event) => void) | null = null;
  public onclose: ((event: CloseEvent) => void) | null = null;
  public onerror: ((event: Event) => void) | null = null;
  public onmessage: ((event: MessageEvent) => void) | null = null;

  private sentMessages: WebSocketMessage[] = [];
  private receivedMessages: WebSocketMessage[] = [];
  private options: MockWebSocketOptions;
  private responseQueue: ArrayBuffer[] = [];

  constructor(url: string, options: MockWebSocketOptions = {}) {
    this.url = url;
    this.options = {
      autoConnect: true,
      latency: 0,
      ...options,
    };

    if (this.options.autoConnect) {
      setTimeout(() => this.connect(), 0);
    }
  }

  /**
   * Simulate connection opening
   */
  connect(): void {
    this.readyState = WebSocket.OPEN;
    if (this.onopen) {
      this.onopen(new Event('open'));
    }
  }

  /**
   * Send data through the mock WebSocket
   */
  send(data: string | ArrayBuffer | Blob): void {
    if (this.readyState !== WebSocket.OPEN) {
      throw new Error('WebSocket is not open');
    }

    let arrayBuffer: ArrayBuffer;

    if (typeof data === 'string') {
      const encoder = new TextEncoder();
      arrayBuffer = encoder.encode(data).buffer;
    } else if (data instanceof Blob) {
      throw new Error('Blob not supported in mock');
    } else {
      arrayBuffer = data;
    }

    const message: WebSocketMessage = {
      data: arrayBuffer,
      timestamp: Date.now(),
    };

    this.sentMessages.push(message);

    if (this.options.onSend) {
      this.options.onSend(arrayBuffer);
    }

    // Process response queue
    this.processResponseQueue();
  }

  /**
   * Simulate receiving a message
   */
  receive(data: ArrayBuffer): void {
    if (this.readyState !== WebSocket.OPEN) {
      return;
    }

    const message: WebSocketMessage = {
      data,
      timestamp: Date.now(),
    };

    this.receivedMessages.push(message);

    const delay = this.options.latency || 0;
    setTimeout(() => {
      if (this.onmessage) {
        this.onmessage(new MessageEvent('message', { data }));
      }
    }, delay);
  }

  /**
   * Queue a response to be sent after the next send()
   */
  queueResponse(data: ArrayBuffer): void {
    this.responseQueue.push(data);
  }

  /**
   * Queue multiple responses
   */
  queueResponses(dataArray: ArrayBuffer[]): void {
    this.responseQueue.push(...dataArray);
  }

  /**
   * Process queued responses
   */
  private processResponseQueue(): void {
    if (this.responseQueue.length > 0) {
      const response = this.responseQueue.shift()!;
      setTimeout(() => this.receive(response), this.options.latency || 0);
    }
  }

  /**
   * Close the WebSocket connection
   */
  close(code?: number, reason?: string): void {
    this.readyState = WebSocket.CLOSED;
    if (this.onclose) {
      this.onclose(new CloseEvent('close', { code, reason }));
    }
  }

  /**
   * Simulate an error
   */
  error(message?: string): void {
    if (this.onerror) {
      const event = new Event('error') as Event & { message?: string };
      event.message = message;
      this.onerror(event);
    }
  }

  /**
   * Get all sent messages
   */
  getSentMessages(): WebSocketMessage[] {
    return [...this.sentMessages];
  }

  /**
   * Get all received messages
   */
  getReceivedMessages(): WebSocketMessage[] {
    return [...this.receivedMessages];
  }

  /**
   * Decode a sent message as MessagePack
   */
  decodeSentMessage<T>(index: number): T {
    if (index >= this.sentMessages.length) {
      throw new Error(`No message at index ${index}`);
    }
    return unpack(new Uint8Array(this.sentMessages[index].data)) as T;
  }

  /**
   * Get the last sent message decoded as MessagePack
   */
  getLastSentMessage<T>(): T | null {
    if (this.sentMessages.length === 0) return null;
    return this.decodeSentMessage<T>(this.sentMessages.length - 1);
  }

  /**
   * Clear all recorded messages
   */
  clearMessages(): void {
    this.sentMessages = [];
    this.receivedMessages = [];
  }

  /**
   * Check if a specific message was sent
   */
  hasSentMessage(predicate: (msg: unknown) => boolean): boolean {
    return this.sentMessages.some((msg) => {
      const decoded = unpack(new Uint8Array(msg.data));
      return predicate(decoded);
    });
  }

  /**
   * Wait for a specific message to be sent
   */
  async waitForSentMessage(
    predicate: (msg: unknown) => boolean,
    timeoutMs = 5000
  ): Promise<unknown> {
    const startTime = Date.now();

    while (Date.now() - startTime < timeoutMs) {
      const message = this.sentMessages.find((msg) => {
        const decoded = unpack(new Uint8Array(msg.data));
        return predicate(decoded);
      });

      if (message) {
        return unpack(new Uint8Array(message.data));
      }

      await new Promise((resolve) => setTimeout(resolve, 100));
    }

    throw new Error('Timeout waiting for message');
  }
}

/**
 * Create a mock WebSocket for testing
 */
export function createMockWebSocket(
  url = 'ws://localhost/test',
  options?: MockWebSocketOptions
): MockWebSocket {
  return new MockWebSocket(url, options);
}

/**
 * Replace global WebSocket with mock
 */
export function installWebSocketMock(): typeof MockWebSocket {
  interface GlobalWithWebSocket {
    WebSocket: typeof WebSocket;
  }
  (globalThis as unknown as GlobalWithWebSocket).WebSocket = MockWebSocket as unknown as typeof WebSocket;
  return MockWebSocket;
}

/**
 * Restore original WebSocket
 */
export function uninstallWebSocketMock(originalWebSocket: typeof WebSocket): void {
  interface GlobalWithWebSocket {
    WebSocket: typeof WebSocket;
  }
  (globalThis as unknown as GlobalWithWebSocket).WebSocket = originalWebSocket;
}
