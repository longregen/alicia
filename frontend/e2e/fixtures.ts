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

// Mock state for conversations and messages (per-test isolation)
interface MockConversation {
  id: string;
  title: string;
  status: string;
  last_client_stanza_id: number;
  last_server_stanza_id: number;
  created_at: string;
  updated_at: string;
  messages: MockMessage[];
}

interface MockMessage {
  id: string;
  conversation_id: string;
  sequence_number: number;
  role: 'user' | 'assistant';
  content: string;
  contents: string;
  created_at: string;
  updated_at: string;
}

interface MockMCPServer {
  name: string;
  transport: string;
  command: string;
  args: string[];
  status: 'connected' | 'error' | 'disconnected';
  tools: string[]; // Array of tool names, not full tool objects
  toolObjects?: MockTool[]; // Keep the full tool objects for the tools endpoint
}

interface MockTool {
  name: string;
  description: string;
}

interface MockState {
  conversations: Map<string, MockConversation>;
  messages: Map<string, MockMessage[]>;
  mcpServers: Map<string, MockMCPServer>;
}

// Helper function to create a fresh mock state for each test
function createMockState(): MockState {
  return {
    conversations: new Map(),
    messages: new Map(),
    mcpServers: new Map(),
  };
}

// Setup all API mocks for a page
async function setupApiMocks(page: Page, mockState: MockState) {
  let messageCounter = 0;

  // Note: sql.js WASM loads from Vite dev server - no mocking needed
  // The real sql.js database is used for e2e tests

  // Mock /api/v1/config
  await page.route('**/api/v1/config', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockConfigResponse),
    });
  });

  // Mock GET/POST /api/v1/conversations
  await page.route('**/api/v1/conversations', async (route: Route) => {
    if (route.request().method() === 'GET') {
      const conversationsList = Array.from(mockState.conversations.values()).map(c => ({
        id: c.id,
        title: c.title,
        status: c.status,
        created_at: c.created_at,
        updated_at: c.updated_at,
      }));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ conversations: conversationsList }),
      });
    } else if (route.request().method() === 'POST') {
      const body = route.request().postDataJSON() || {};
      const id = `conv-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
      const now = new Date().toISOString();
      const conversation: MockConversation = {
        id,
        title: body.title || 'New Conversation',
        status: 'active',
        last_client_stanza_id: 0,
        last_server_stanza_id: 0,
        created_at: now,
        updated_at: now,
        messages: [],
      };
      mockState.conversations.set(id, conversation);
      mockState.messages.set(id, []);
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify(conversation),
      });
    } else {
      await route.continue();
    }
  });

  // Routes for specific conversation operations
  await page.route('**/api/v1/conversations/*', async (route: Route) => {
    const request = route.request();
    const url = request.url();
    const method = request.method();

    // Extract conversation ID from URL
    const conversationIdMatch = url.match(/\/conversations\/([^/]+)/);
    if (!conversationIdMatch) {
      await route.fulfill({ status: 404, body: 'Not found' });
      return;
    }
    const conversationId = conversationIdMatch[1];

    // Handle messages endpoint
    if (url.includes('/messages')) {
      if (method === 'GET') {
        const messages = mockState.messages.get(conversationId) || [];
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ messages }),
        });
      } else if (method === 'POST') {
        const body = request.postDataJSON() || {};
        const now = new Date().toISOString();
        messageCounter++;
        const message: MockMessage = {
          id: body.local_id || `msg-${Date.now()}-${messageCounter}`,
          conversation_id: conversationId,
          sequence_number: messageCounter,
          role: 'user',
          content: body.content || body.contents || '',
          contents: body.contents || body.content || '',
          created_at: now,
          updated_at: now,
        };
        const messages = mockState.messages.get(conversationId) || [];
        messages.push(message);
        mockState.messages.set(conversationId, messages);

        const conversation = mockState.conversations.get(conversationId);
        if (conversation) {
          conversation.messages.push(message);
          conversation.updated_at = now;

          // Simulate assistant response after a short delay
          setTimeout(() => {
            const assistantMessage: MockMessage = {
              id: `msg-${Date.now()}-assistant`,
              conversation_id: conversationId,
              sequence_number: ++messageCounter,
              role: 'assistant',
              content: 'This is a mock assistant response for testing purposes.',
              contents: 'This is a mock assistant response for testing purposes.',
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString(),
            };
            messages.push(assistantMessage);
            conversation.messages.push(assistantMessage);
          }, 100);
        }

        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify(message),
        });
      } else {
        await route.continue();
      }
      return;
    }

    // Handle token endpoint
    if (url.includes('/token')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ token: 'mock-livekit-token' }),
      });
      return;
    }

    // Handle sync endpoints
    if (url.includes('/sync')) {
      if (url.includes('/status')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'synced', last_sync: new Date().toISOString() }),
        });
      } else if (url.includes('/ws')) {
        // WebSocket upgrades can't be mocked directly
        await route.fulfill({ status: 200, body: '' });
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ success: true }),
        });
      }
      return;
    }

    // Handle conversation CRUD
    if (method === 'GET') {
      const conversation = mockState.conversations.get(conversationId);
      if (conversation) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(conversation),
        });
      } else {
        await route.fulfill({ status: 404, body: 'Not found' });
      }
    } else if (method === 'DELETE') {
      mockState.conversations.delete(conversationId);
      mockState.messages.delete(conversationId);
      await route.fulfill({ status: 204 });
    } else if (method === 'PATCH') {
      const conversation = mockState.conversations.get(conversationId);
      if (conversation) {
        const body = request.postDataJSON() || {};
        Object.assign(conversation, body, { updated_at: new Date().toISOString() });
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(conversation),
        });
      } else {
        await route.fulfill({ status: 404, body: 'Not found' });
      }
    } else {
      await route.continue();
    }
  });

  // MCP servers endpoint
  await page.route('**/api/v1/mcp/servers', async (route: Route) => {
    if (route.request().method() === 'GET') {
      const servers = Array.from(mockState.mcpServers.values());
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ servers }),
      });
    } else if (route.request().method() === 'POST') {
      const body = route.request().postDataJSON() || {};
      const toolObjects: MockTool[] = [
        { name: 'read_file', description: 'Read a file from the filesystem' },
        { name: 'write_file', description: 'Write a file to the filesystem' },
        { name: 'list_directory', description: 'List directory contents' },
      ];

      const server: MockMCPServer = {
        name: body.name,
        transport: 'stdio',
        command: body.command,
        args: body.args ? (typeof body.args === 'string' ? body.args.split(',').map((a: string) => a.trim()) : body.args) : [],
        status: body.command?.includes('/invalid') ? 'error' : 'connected',
        tools: toolObjects.map(t => t.name), // Store only tool names in the server
        toolObjects: toolObjects, // Keep full objects for the tools endpoint
      };
      mockState.mcpServers.set(body.name, server);
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify(server),
      });
    } else {
      await route.continue();
    }
  });

  // MCP server delete endpoint
  await page.route('**/api/v1/mcp/servers/*', async (route: Route) => {
    if (route.request().method() === 'DELETE') {
      const url = route.request().url();
      const serverName = decodeURIComponent(url.split('/').pop() || '');
      mockState.mcpServers.delete(serverName);
      await route.fulfill({ status: 204 });
    } else {
      await route.continue();
    }
  });

  // MCP tools endpoint
  await page.route('**/api/v1/mcp/tools', async (route: Route) => {
    const allTools: Record<string, MockTool[]> = {};
    mockState.mcpServers.forEach((server) => {
      if (server.toolObjects) {
        allTools[server.name] = server.toolObjects;
      }
    });
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ tools: allTools }),
    });
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
    await route.fulfill({
      status: 200,
      contentType: 'audio/mpeg',
      body: Buffer.from([]),
    });
  });

  return mockState;
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
    // Create isolated mock state for this test
    const mockState = createMockState();

    // Setup all API mocks before each test
    await setupApiMocks(page, mockState);

    // Mock the connectionStore to always be connected in tests
    // This prevents the input field from being disabled with "Connecting..." placeholder
    await page.addInitScript(() => {
      // Override the connectionStore's initial state to be Connected
      // This runs before the page loads, ensuring the store starts in the right state
      (window as any).__E2E_CONNECTION_MOCK__ = {
        status: 'connected',
        error: null,
        roomName: null,
        roomSid: null,
        participants: {},
        localParticipantId: null,
        connectedAt: new Date(),
        reconnectAttempts: 0,
      };
    });

    await use(page);
  },

  conversationHelpers: async ({ page }, use) => {
    const helpers: ConversationHelpers = {
      async createConversation() {
        await page.click('[data-testid="new-chat-btn"]');

        // Wait for conversation to be selected
        await page.waitForSelector('.conversation-item.bg-sidebar-accent', { state: 'visible', timeout: 5000 });

        // Get the conversation ID from the selected item
        const selectedConv = await page.locator('.conversation-item.bg-sidebar-accent').first();
        const conversationId = await selectedConv.getAttribute('data-conversation-id');

        if (!conversationId) {
          throw new Error('Failed to get conversation ID');
        }

        return conversationId;
      },

      async sendMessage(conversationId: string, message: string) {
        // Make sure the conversation is selected
        await page.click(`[data-conversation-id="${conversationId}"]`);
        await page.keyboard.press('Escape');

        // Wait for input to be visible and enabled
        const inputSelector = '.input-bar input[type="text"]';
        await page.waitForSelector(inputSelector, { state: 'visible' });

        // Wait for the input to be enabled (not disabled)
        // Use a longer timeout for connection to be established
        const input = page.locator(inputSelector);
        await page.waitForFunction(
          (selector) => {
            const el = document.querySelector(selector) as HTMLInputElement;
            return el && !el.disabled;
          },
          inputSelector,
          { timeout: 15000 }
        );

        // Type and send message
        await input.fill(message);
        await page.click('.input-bar button[type="submit"]');

        // Wait for the user message to appear in the list
        await page.locator('div.user').filter({ hasText: message }).first().waitFor({
          timeout: 10000,
        });

        // Note: We don't wait for assistant response because the frontend uses
        // a local SQL.js database and the mocked assistant response may not appear
        // in the UI (it's only in the mock API state, not the local DB)
      },

      async deleteConversation(conversationId: string) {
        // Click the conversation item to open dropdown menu
        const conversationItem = page.locator(`[data-conversation-id="${conversationId}"]`);
        await conversationItem.click();

        // Click the delete menu item
        await page.click('[data-testid="delete-conversation-menu-item"]');

        // Wait for conversation to be removed from the DOM
        await page.waitForSelector(`[data-conversation-id="${conversationId}"]`, {
          state: 'hidden',
          timeout: 5000,
        });
      },

      async waitForMessage(messageText: string) {
        await page.locator('div.user, div.assistant, div.system').filter({ hasText: messageText }).first().waitFor({
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

        // Register dialog handler BEFORE clicking to catch the confirm dialog
        page.once('dialog', dialog => dialog.accept());

        // Click the remove button
        await serverCard.locator('.remove-server-btn').click();

        // Wait for success toast to appear
        await page.waitForSelector('.toast-success', { timeout: 5000 });

        // Wait for server to be removed from the list
        await page.waitForSelector(`.server-card:has-text("${name}")`, {
          state: 'hidden',
          timeout: 5000,
        });
      },

      async expandServerTools(serverName: string) {
        const serverCard = page.locator(`.server-card:has-text("${serverName}")`);
        const toolsToggle = serverCard.locator('.tools-toggle');

        // Wait for the toggle button to be visible and enabled
        await toolsToggle.waitFor({ state: 'visible' });

        // Click to expand
        await toolsToggle.click();

        // Wait a bit for React state to update
        await page.waitForTimeout(300);

        // Wait for tools list to be visible
        await serverCard.locator('.tools-list').waitFor({ state: 'visible', timeout: 10000 });
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
