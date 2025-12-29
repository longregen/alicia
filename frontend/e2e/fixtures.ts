import { test as base, expect } from '@playwright/test';
import type { Page, Route } from '@playwright/test';

// Mock config response for e2e tests
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
      { id: 'bf_emma', name: 'Emma', category: 'British Female' },
      { id: 'bm_george', name: 'George', category: 'British Male' },
    ],
  },
};

// In-memory storage for mock data
interface MockConversation {
  id: string;
  title: string;
  created_at: string;
  updated_at: string;
  messages: MockMessage[];
}

interface MockMessage {
  id: string;
  conversation_id: string;
  role: 'user' | 'assistant';
  content: string;
  created_at: string;
}

interface MockMCPServer {
  name: string;
  transport: string;
  command: string;
  args: string[];
  status: 'Connected' | 'Error' | 'Disconnected';
  tools: MockTool[];
}

interface MockTool {
  name: string;
  description: string;
}

// Create mock data storage per page context
function createMockStorage() {
  const conversations: Map<string, MockConversation> = new Map();
  const mcpServers: Map<string, MockMCPServer> = new Map();

  return { conversations, mcpServers };
}

// Setup all API mocks for a page
async function setupApiMocks(page: Page) {
  const storage = createMockStorage();

  // Mock /api/v1/config
  await page.route('**/api/v1/config', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockConfigResponse),
    });
  });

  // Mock GET /api/v1/conversations
  await page.route('**/api/v1/conversations', async (route: Route) => {
    if (route.request().method() === 'GET') {
      const conversationsList = Array.from(storage.conversations.values()).map(c => ({
        id: c.id,
        title: c.title,
        created_at: c.created_at,
        updated_at: c.updated_at,
      }));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(conversationsList),
      });
    } else if (route.request().method() === 'POST') {
      const id = `conv-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
      const now = new Date().toISOString();
      const conversation: MockConversation = {
        id,
        title: 'New Conversation',
        created_at: now,
        updated_at: now,
        messages: [],
      };
      storage.conversations.set(id, conversation);
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({ id, title: conversation.title, created_at: now, updated_at: now }),
      });
    } else {
      await route.continue();
    }
  });

  // Mock GET/DELETE /api/v1/conversations/:id
  await page.route(/\/api\/v1\/conversations\/[^/]+$/, async (route: Route) => {
    const url = route.request().url();
    const match = url.match(/\/api\/v1\/conversations\/([^/?]+)/);
    const conversationId = match?.[1];

    if (route.request().method() === 'GET' && conversationId) {
      const conversation = storage.conversations.get(conversationId);
      if (conversation) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(conversation),
        });
      } else {
        await route.fulfill({ status: 404, body: 'Not found' });
      }
    } else if (route.request().method() === 'DELETE' && conversationId) {
      storage.conversations.delete(conversationId);
      await route.fulfill({ status: 204 });
    } else {
      await route.continue();
    }
  });

  // Mock GET/POST /api/v1/conversations/:id/messages
  await page.route(/\/api\/v1\/conversations\/[^/]+\/messages/, async (route: Route) => {
    const url = route.request().url();
    const match = url.match(/\/api\/v1\/conversations\/([^/]+)\/messages/);
    const conversationId = match?.[1];

    if (!conversationId) {
      await route.continue();
      return;
    }

    const conversation = storage.conversations.get(conversationId);

    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(conversation?.messages || []),
      });
    } else if (route.request().method() === 'POST') {
      const body = route.request().postDataJSON();
      const messageId = `msg-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
      const now = new Date().toISOString();

      const userMessage: MockMessage = {
        id: messageId,
        conversation_id: conversationId,
        role: 'user',
        content: body?.content || '',
        created_at: now,
      };

      if (conversation) {
        conversation.messages.push(userMessage);
        conversation.updated_at = now;

        // Simulate assistant response after a short delay
        const assistantId = `msg-${Date.now() + 1}-${Math.random().toString(36).substr(2, 9)}`;
        const assistantMessage: MockMessage = {
          id: assistantId,
          conversation_id: conversationId,
          role: 'assistant',
          content: 'This is a mock response from the assistant.',
          created_at: new Date(Date.now() + 100).toISOString(),
        };
        conversation.messages.push(assistantMessage);
      }

      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify(userMessage),
      });
    } else {
      await route.continue();
    }
  });

  // Mock MCP servers endpoints
  await page.route('**/api/v1/mcp/servers', async (route: Route) => {
    if (route.request().method() === 'GET') {
      const serversList = Array.from(storage.mcpServers.values());
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(serversList),
      });
    } else if (route.request().method() === 'POST') {
      const body = route.request().postDataJSON();
      const server: MockMCPServer = {
        name: body?.name || 'Unknown',
        transport: 'stdio',
        command: body?.command || '',
        args: body?.args ? body.args.split(',').map((a: string) => a.trim()) : [],
        status: body?.command?.includes('/invalid') ? 'Error' : 'Connected',
        tools: [
          { name: 'read_file', description: 'Read a file from the filesystem' },
          { name: 'write_file', description: 'Write a file to the filesystem' },
          { name: 'list_directory', description: 'List directory contents' },
        ],
      };
      storage.mcpServers.set(server.name, server);
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify(server),
      });
    } else {
      await route.continue();
    }
  });

  // Mock DELETE /api/v1/mcp/servers/:name
  await page.route(/\/api\/v1\/mcp\/servers\/[^/]+/, async (route: Route) => {
    if (route.request().method() === 'DELETE') {
      const url = route.request().url();
      const match = url.match(/\/api\/v1\/mcp\/servers\/([^/?]+)/);
      const serverName = match?.[1];
      if (serverName) {
        storage.mcpServers.delete(decodeURIComponent(serverName));
      }
      await route.fulfill({ status: 204 });
    } else {
      await route.continue();
    }
  });

  // Mock WebSocket sync endpoint - just let it fail gracefully
  await page.route(/\/api\/v1\/conversations\/[^/]+\/sync\/ws/, async (route: Route) => {
    // WebSocket upgrades can't be mocked directly, return 101 or let it fail
    await route.fulfill({ status: 200, body: '' });
  });

  // Mock chat completion endpoint
  await page.route('**/api/v1/chat/completions', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: `chatcmpl-${Date.now()}`,
        object: 'chat.completion',
        created: Math.floor(Date.now() / 1000),
        choices: [{
          index: 0,
          message: {
            role: 'assistant',
            content: 'This is a mock response from the assistant.',
          },
          finish_reason: 'stop',
        }],
      }),
    });
  });

  // Mock audio/speech endpoint
  await page.route('**/v1/audio/speech', async (route: Route) => {
    // Return empty audio data
    await route.fulfill({
      status: 200,
      contentType: 'audio/mpeg',
      body: Buffer.from([]),
    });
  });

  return storage;
}

export interface ConversationHelpers {
  createConversation(): Promise<string>;
  sendMessage(conversationId: string, message: string): Promise<void>;
  deleteConversation(conversationId: string): Promise<void>;
  waitForMessage(messageText: string): Promise<void>;
}

export interface MCPHelpers {
  openSettings(): Promise<void>;
  addServer(name: string, command: string, args?: string): Promise<void>;
  removeServer(name: string): Promise<void>;
  expandServerTools(serverName: string): Promise<void>;
  waitForServerStatus(serverName: string, status: 'Connected' | 'Error' | 'Disconnected'): Promise<void>;
}

export interface SyncHelpers {
  waitForSync(): Promise<void>;
  getLastSyncTime(): Promise<Date | null>;
}

export interface VoiceHelpers {
  activateVoiceMode(): Promise<void>;
  deactivateVoiceMode(): Promise<void>;
  isVoiceModeActive(): Promise<boolean>;
  startRecording(): Promise<void>;
  stopRecording(): Promise<void>;
  openVoiceSelector(): Promise<void>;
  closeVoiceSelector(): Promise<void>;
  selectVoice(voiceId: string): Promise<void>;
  setSpeed(speed: number): Promise<void>;
  waitForConnectionState(state: 'connected' | 'connecting' | 'reconnecting' | 'disconnected'): Promise<void>;
}

type TestFixtures = {
  conversationHelpers: ConversationHelpers;
  mcpHelpers: MCPHelpers;
  syncHelpers: SyncHelpers;
  voiceHelpers: VoiceHelpers;
};

const test = base.extend<TestFixtures>({
  page: async ({ page }, use) => {
    // Setup all API mocks before each test
    await setupApiMocks(page);

    await use(page);
  },

  conversationHelpers: async ({ page }, use) => {
    const helpers: ConversationHelpers = {
      async createConversation() {
        await page.click('button:has-text("New Chat")');

        // Wait for conversation to be created and selected
        await page.waitForSelector('.chat-window', { state: 'visible' });

        // Get the conversation ID from the selected item
        const selectedConv = await page.locator('.conversation-item.selected').first();
        const conversationId = await selectedConv.getAttribute('data-conversation-id');

        if (!conversationId) {
          throw new Error('Failed to get conversation ID');
        }

        return conversationId;
      },

      async sendMessage(conversationId: string, message: string) {
        // Make sure the conversation is selected
        await page.click(`[data-conversation-id="${conversationId}"]`);

        // Type and send message
        await page.fill('.input-bar input[type="text"]', message);
        await page.click('.input-bar button[type="submit"]');

        // Wait for message to appear in the list
        await page.waitForSelector(`.message-bubble:has-text("${message}")`, {
          timeout: 5000,
        });
      },

      async deleteConversation(conversationId: string) {
        // Click the delete button for the conversation
        await page.click(`[data-conversation-id="${conversationId}"] .delete-btn`);

        // Confirm deletion if there's a confirmation dialog
        await page.click('button:has-text("Delete")');

        // Wait for conversation to be removed
        await page.waitForSelector(`[data-conversation-id="${conversationId}"]`, {
          state: 'hidden',
          timeout: 5000,
        });
      },

      async waitForMessage(messageText: string) {
        await page.waitForSelector(`.message-bubble:has-text("${messageText}")`, {
          timeout: 10000,
        });
      },
    };

    await use(helpers);
  },

  mcpHelpers: async ({ page }, use) => {
    const helpers: MCPHelpers = {
      async openSettings() {
        await page.click('button[title="Settings"]');
        await page.waitForSelector('.mcp-settings', { state: 'visible' });
      },

      async addServer(name: string, command: string, args?: string) {
        await page.click('button:has-text("Add Server")');

        await page.fill('#server-name', name);
        await page.fill('#command', command);

        if (args) {
          await page.fill('#args', args);
        }

        await page.click('button[type="submit"]:has-text("Add Server")');

        // Wait for success toast
        await page.waitForSelector('.toast-success', { timeout: 5000 });

        // Wait for the server to appear in the list
        await page.waitForSelector(`.server-card:has-text("${name}")`, {
          timeout: 5000,
        });
      },

      async removeServer(name: string) {
        const serverCard = page.locator(`.server-card:has-text("${name}")`);
        await serverCard.locator('.remove-server-btn').click();

        // Confirm removal
        page.once('dialog', dialog => dialog.accept());

        // Wait for server to be removed
        await page.waitForSelector(`.server-card:has-text("${name}")`, {
          state: 'hidden',
          timeout: 5000,
        });
      },

      async expandServerTools(serverName: string) {
        const serverCard = page.locator(`.server-card:has-text("${serverName}")`);
        await serverCard.locator('.tools-toggle').click();

        // Wait for tools list to be visible
        await serverCard.locator('.tools-list').waitFor({ state: 'visible' });
      },

      async waitForServerStatus(serverName: string, status: 'Connected' | 'Error' | 'Disconnected') {
        await page.waitForSelector(
          `.server-card:has-text("${serverName}") .status-badge:has-text("${status}")`,
          { timeout: 10000 }
        );
      },
    };

    await use(helpers);
  },

  syncHelpers: async ({ page }, use) => {
    const helpers: SyncHelpers = {
      async waitForSync() {
        // Look for sync indicator to show syncing
        await page.waitForSelector('.sync-status:has-text("Syncing")', {
          timeout: 5000,
        });

        // Wait for sync to complete
        await page.waitForSelector('.sync-status:has-text("Synced")', {
          timeout: 10000,
        });
      },

      async getLastSyncTime() {
        const syncTimeElement = await page.locator('.last-sync-time').textContent();
        if (!syncTimeElement) return null;

        // Parse the time from the element text
        const match = syncTimeElement.match(/(\d{2}:\d{2}:\d{2})/);
        if (!match) return null;

        const [hours, minutes, seconds] = match[1].split(':').map(Number);
        const now = new Date();
        now.setHours(hours, minutes, seconds, 0);

        return now;
      },
    };

    await use(helpers);
  },

  voiceHelpers: async ({ page }, use) => {
    const helpers: VoiceHelpers = {
      async activateVoiceMode() {
        const voiceModeToggle = page.locator('.voice-mode-toggle');
        const isActive = await voiceModeToggle.evaluate((el) =>
          el.classList.contains('active')
        );

        if (!isActive) {
          await voiceModeToggle.click();
          await page.waitForTimeout(500); // Allow time for activation
        }
      },

      async deactivateVoiceMode() {
        const voiceModeToggle = page.locator('.voice-mode-toggle');
        const isActive = await voiceModeToggle.evaluate((el) =>
          el.classList.contains('active')
        );

        if (isActive) {
          await voiceModeToggle.click();
          await page.waitForTimeout(500); // Allow time for deactivation
        }
      },

      async isVoiceModeActive() {
        const voiceModeToggle = page.locator('.voice-mode-toggle');
        return await voiceModeToggle.evaluate((el) =>
          el.classList.contains('active')
        );
      },

      async startRecording() {
        const recordBtn = page.locator('.record-btn');
        const isRecording = await recordBtn.evaluate((el) =>
          el.classList.contains('recording')
        );

        if (!isRecording) {
          await recordBtn.click();
          await page.waitForTimeout(300);
        }
      },

      async stopRecording() {
        const recordBtn = page.locator('.record-btn');
        const isRecording = await recordBtn.evaluate((el) =>
          el.classList.contains('recording')
        );

        if (isRecording) {
          await recordBtn.click();
          await page.waitForTimeout(300);
        }
      },

      async openVoiceSelector() {
        const voiceSelectorToggle = page.locator('.voice-selector-toggle');
        await voiceSelectorToggle.click();
        await page.waitForSelector('.voice-selector-panel', { state: 'visible' });
      },

      async closeVoiceSelector() {
        const closeBtn = page.locator('.voice-selector-close');
        await closeBtn.click();
        await page.waitForSelector('.voice-selector-panel', { state: 'hidden' });
      },

      async selectVoice(voiceId: string) {
        const voiceSelect = page.locator('.voice-select');
        await voiceSelect.selectOption(voiceId);
        await page.waitForTimeout(300);
      },

      async setSpeed(speed: number) {
        const speedSlider = page.locator('.speed-slider');
        await speedSlider.fill(speed.toString());
        await page.waitForTimeout(300);
      },

      async waitForConnectionState(state: 'connected' | 'connecting' | 'reconnecting' | 'disconnected') {
        const stateMap = {
          connected: 'Connected',
          connecting: 'Connecting',
          reconnecting: 'Reconnecting',
          disconnected: 'Disconnected',
        };

        await page.waitForSelector(
          `.connection-status:has-text("${stateMap[state]}")`,
          { timeout: 10000 }
        );
      },
    };

    await use(helpers);
  },
});

export { test, expect };
