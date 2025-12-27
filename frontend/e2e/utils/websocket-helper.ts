import { Page } from '@playwright/test';

/**
 * WebSocket connection helper for Playwright tests
 */
export class WebSocketHelper {
  private wsMessages: any[] = [];
  private wsConnected = false;
  private wsError: string | null = null;

  constructor(private page: Page) {}

  /**
   * Setup WebSocket monitoring in the browser context
   * Call this before navigating to capture WebSocket events
   */
  async setupMonitoring(): Promise<void> {
    await this.page.addInitScript(() => {
      // Store original WebSocket
      const OriginalWebSocket = window.WebSocket;
      const messages: any[] = [];
      let connected = false;
      let error: string | null = null;

      // Override WebSocket constructor
      (window as any).WebSocket = function (
        url: string | URL,
        protocols?: string | string[]
      ) {
        const ws = new OriginalWebSocket(url, protocols);

        // Track connection state
        ws.addEventListener('open', () => {
          connected = true;
          (window as any).__wsConnected = true;
        });

        ws.addEventListener('close', () => {
          connected = false;
          (window as any).__wsConnected = false;
        });

        ws.addEventListener('error', (event) => {
          error = 'WebSocket error occurred';
          (window as any).__wsError = error;
        });

        // Track messages
        ws.addEventListener('message', (event) => {
          messages.push({
            data: event.data,
            timestamp: Date.now(),
            type: 'received',
          });
          (window as any).__wsMessages = messages;
        });

        // Track sent messages
        const originalSend = ws.send.bind(ws);
        ws.send = function (data: string | ArrayBuffer | Blob) {
          messages.push({
            data,
            timestamp: Date.now(),
            type: 'sent',
          });
          (window as any).__wsMessages = messages;
          return originalSend(data);
        };

        return ws;
      };

      // Preserve WebSocket constants
      (window as any).WebSocket.CONNECTING = OriginalWebSocket.CONNECTING;
      (window as any).WebSocket.OPEN = OriginalWebSocket.OPEN;
      (window as any).WebSocket.CLOSING = OriginalWebSocket.CLOSING;
      (window as any).WebSocket.CLOSED = OriginalWebSocket.CLOSED;
    });
  }

  /**
   * Wait for WebSocket to connect
   */
  async waitForConnection(timeoutMs = 10000): Promise<void> {
    const startTime = Date.now();

    while (Date.now() - startTime < timeoutMs) {
      const connected = await this.page.evaluate(
        () => (window as any).__wsConnected === true
      );

      if (connected) {
        this.wsConnected = true;
        return;
      }

      await this.page.waitForTimeout(100);
    }

    throw new Error('WebSocket connection timeout');
  }

  /**
   * Wait for WebSocket to disconnect
   */
  async waitForDisconnection(timeoutMs = 10000): Promise<void> {
    const startTime = Date.now();

    while (Date.now() - startTime < timeoutMs) {
      const connected = await this.page.evaluate(
        () => (window as any).__wsConnected === true
      );

      if (!connected) {
        this.wsConnected = false;
        return;
      }

      await this.page.waitForTimeout(100);
    }

    throw new Error('WebSocket disconnection timeout');
  }

  /**
   * Check if WebSocket is currently connected
   */
  async isConnected(): Promise<boolean> {
    return this.page.evaluate(
      () => (window as any).__wsConnected === true
    );
  }

  /**
   * Get all captured WebSocket messages
   */
  async getMessages(): Promise<any[]> {
    return this.page.evaluate(() => (window as any).__wsMessages || []);
  }

  /**
   * Get sent WebSocket messages
   */
  async getSentMessages(): Promise<any[]> {
    const messages = await this.getMessages();
    return messages.filter((msg) => msg.type === 'sent');
  }

  /**
   * Get received WebSocket messages
   */
  async getReceivedMessages(): Promise<any[]> {
    const messages = await this.getMessages();
    return messages.filter((msg) => msg.type === 'received');
  }

  /**
   * Wait for a specific message to be sent
   */
  async waitForSentMessage(
    predicate: (msg: any) => boolean,
    timeoutMs = 5000
  ): Promise<any> {
    const startTime = Date.now();

    while (Date.now() - startTime < timeoutMs) {
      const messages = await this.getSentMessages();
      const found = messages.find(predicate);

      if (found) {
        return found;
      }

      await this.page.waitForTimeout(100);
    }

    throw new Error('Timeout waiting for sent message');
  }

  /**
   * Wait for a specific message to be received
   */
  async waitForReceivedMessage(
    predicate: (msg: any) => boolean,
    timeoutMs = 5000
  ): Promise<any> {
    const startTime = Date.now();

    while (Date.now() - startTime < timeoutMs) {
      const messages = await this.getReceivedMessages();
      const found = messages.find(predicate);

      if (found) {
        return found;
      }

      await this.page.waitForTimeout(100);
    }

    throw new Error('Timeout waiting for received message');
  }

  /**
   * Clear all captured messages
   */
  async clearMessages(): Promise<void> {
    await this.page.evaluate(() => {
      (window as any).__wsMessages = [];
    });
  }

  /**
   * Get the count of messages
   */
  async getMessageCount(): Promise<{ sent: number; received: number }> {
    const messages = await this.getMessages();
    return {
      sent: messages.filter((msg) => msg.type === 'sent').length,
      received: messages.filter((msg) => msg.type === 'received').length,
    };
  }

  /**
   * Simulate WebSocket disconnection (close from client side)
   */
  async disconnect(): Promise<void> {
    await this.page.evaluate(() => {
      // Find and close all WebSocket connections
      const websockets = (window as any).__websockets || [];
      websockets.forEach((ws: WebSocket) => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.close();
        }
      });
    });
  }
}

/**
 * Create a WebSocket helper for a page
 */
export function createWebSocketHelper(page: Page): WebSocketHelper {
  return new WebSocketHelper(page);
}

/**
 * Wait for sync to complete via WebSocket
 */
export async function waitForWebSocketSync(
  page: Page,
  timeoutMs = 10000
): Promise<void> {
  const helper = new WebSocketHelper(page);
  await helper.setupMonitoring();

  // Wait for connection
  await helper.waitForConnection(timeoutMs);

  // Wait for sync_response message
  await helper.waitForReceivedMessage(
    (msg) => {
      try {
        if (typeof msg.data === 'string') {
          const parsed = JSON.parse(msg.data);
          return parsed.type === 'sync_response';
        }
        return false;
      } catch {
        return false;
      }
    },
    timeoutMs
  );
}

/**
 * Intercept and mock WebSocket messages
 */
export async function mockWebSocketMessages(
  page: Page,
  messageHandler: (message: any) => any | null
): Promise<void> {
  await page.addInitScript((handler) => {
    const OriginalWebSocket = window.WebSocket;

    (window as any).WebSocket = function (
      url: string | URL,
      protocols?: string | string[]
    ) {
      const ws = new OriginalWebSocket(url, protocols);

      // Intercept received messages
      ws.addEventListener('message', (event) => {
        try {
          const data =
            typeof event.data === 'string'
              ? JSON.parse(event.data)
              : event.data;
          const modified = (handler as any)(data);

          if (modified) {
            // Dispatch modified message
            const newEvent = new MessageEvent('message', {
              data:
                typeof event.data === 'string'
                  ? JSON.stringify(modified)
                  : modified,
            });
            ws.dispatchEvent(newEvent);
          }
        } catch (err) {
          console.error('Error in message handler:', err);
        }
      });

      return ws;
    };
  }, messageHandler);
}
