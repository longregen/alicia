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
  tools: string[];  // Array of tool names, not tool objects
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

  // Mock sql.js database initialization by injecting a script before page load
  await page.addInitScript(() => {
    // In-memory storage for the mock database
    interface MockRow {
      [key: string]: unknown;
    }
    interface MockTables {
      messages: MockRow[];
      conversations: MockRow[];
      [key: string]: MockRow[];
    }
    const tables: MockTables = {
      messages: [],
      conversations: [],
    };

    // Simple SQL parser for common operations
    const mockDatabase = {
      run: (sql: string, params?: unknown[]) => {
        const sqlLower = sql.toLowerCase().trim();

        // Handle INSERT
        if (sqlLower.startsWith('insert into messages')) {
          const row: MockRow = {
            id: params?.[0],
            conversation_id: params?.[1],
            sequence_number: params?.[2],
            role: params?.[3],
            contents: params?.[4],
            local_id: params?.[5],
            server_id: params?.[6],
            sync_status: params?.[7],
            retry_count: params?.[8],
            created_at: params?.[9],
            updated_at: params?.[10],
          };
          tables.messages.push(row);

          // Auto-generate assistant response for user messages (synchronously)
          if (row.role === 'user') {
            const assistantRow: MockRow = {
              id: `assistant-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
              conversation_id: row.conversation_id,
              sequence_number: (row.sequence_number as number) + 1,
              role: 'assistant',
              contents: 'This is a mock assistant response for testing purposes.',
              local_id: null,
              server_id: `srv-${Date.now()}`,
              sync_status: 'synced',
              retry_count: 0,
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString(),
            };
            tables.messages.push(assistantRow);
          }
        } else if (sqlLower.startsWith('insert into conversations')) {
          const row: MockRow = {
            id: params?.[0],
            title: params?.[1],
            status: params?.[2],
            created_at: params?.[3],
            updated_at: params?.[4],
          };
          tables.conversations.push(row);
        } else if (sqlLower.startsWith('update messages')) {
          // Simple update - find by id (last param) and update fields
          const id = params?.[params.length - 1];
          const idx = tables.messages.findIndex(m => m.id === id);
          if (idx >= 0) {
            // Parse SET clause to determine which fields to update
            if (sqlLower.includes('sync_status')) {
              tables.messages[idx].sync_status = params?.[0];
            }
            if (sqlLower.includes('contents')) {
              tables.messages[idx].contents = params?.[0];
            }
          }
        } else if (sqlLower.startsWith('delete from messages')) {
          const id = params?.[0];
          const idx = tables.messages.findIndex(m => m.id === id);
          if (idx >= 0) {
            tables.messages.splice(idx, 1);
          }
        } else if (sqlLower.startsWith('create table')) {
          // Ignore schema creation
        }
      },
      exec: (sql: string, params?: unknown[]) => {
        const sqlLower = sql.toLowerCase().trim();

        // Handle SELECT from messages
        if (sqlLower.includes('from messages') && sqlLower.includes('where conversation_id')) {
          const conversationId = params?.[0];
          const filtered = tables.messages.filter(m => m.conversation_id === conversationId);
          if (filtered.length === 0) return [];
          return [{
            columns: ['id', 'conversation_id', 'sequence_number', 'role', 'contents', 'local_id', 'server_id', 'sync_status', 'retry_count', 'created_at', 'updated_at'],
            values: filtered.map(m => [
              m.id, m.conversation_id, m.sequence_number, m.role, m.contents,
              m.local_id, m.server_id, m.sync_status, m.retry_count,
              m.created_at, m.updated_at
            ])
          }];
        }

        // Handle SELECT from messages by id
        if (sqlLower.includes('from messages') && sqlLower.includes('where id')) {
          const id = params?.[0];
          const found = tables.messages.find(m => m.id === id);
          if (!found) return [];
          return [{
            columns: ['id', 'conversation_id', 'sequence_number', 'role', 'contents', 'local_id', 'server_id', 'sync_status', 'retry_count', 'created_at', 'updated_at'],
            values: [[
              found.id, found.conversation_id, found.sequence_number, found.role, found.contents,
              found.local_id, found.server_id, found.sync_status, found.retry_count,
              found.created_at, found.updated_at
            ]]
          }];
        }

        // Handle SELECT from conversations
        if (sqlLower.includes('from conversations')) {
          if (tables.conversations.length === 0) return [];
          return [{
            columns: ['id', 'title', 'status', 'created_at', 'updated_at'],
            values: tables.conversations.map(c => [c.id, c.title, c.status, c.created_at, c.updated_at])
          }];
        }

        return [];
      },
      export: () => new Uint8Array(),
      close: () => {},
    };

    // Override the dynamic import of sql.js
    (window as unknown as Record<string, unknown>).__SQL_JS_MOCK__ = {
      Database: function() { return mockDatabase; },
    };

    // Mock initSqlJs to return our mock
    (window as unknown as Record<string, unknown>).initSqlJs = async () => {
      return (window as unknown as Record<string, unknown>).__SQL_JS_MOCK__;
    };
  });

  // Mock the sql-wasm.wasm file request
  await page.route('**/sql-wasm.wasm', async (route: Route) => {
    // Return empty response - the mock above will handle initialization
    await route.fulfill({
      status: 200,
      contentType: 'application/wasm',
      body: Buffer.from([]),
    });
  });

  // Mock /api/v1/config
  await page.route('**/api/v1/config', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockConfigResponse),
    });
  });

  // Mock GET/POST /api/v1/conversations (exact path only, not sub-paths)
  // Using regex to ensure we don't match /conversations/123/messages etc.
  await page.route(/\/api\/v1\/conversations\/?$/, async (route: Route) => {
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

  // Routes for specific conversation operations (use ** to match sub-paths like /messages)
  await page.route('**/api/v1/conversations/**', async (route: Route) => {
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

    // Handle events endpoint (SSE)
    if (url.includes('/events')) {
      await route.fulfill({
        status: 200,
        contentType: 'text/event-stream',
        body: '',
      });
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
      const server: MockMCPServer = {
        name: body.name,
        transport: 'stdio',
        command: body.command,
        args: body.args ? (typeof body.args === 'string' ? body.args.split(',').map((a: string) => a.trim()) : body.args) : [],
        status: body.command?.includes('/invalid') ? 'error' : 'connected',
        tools: ['read_file', 'write_file', 'list_directory'],  // Array of tool names
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

  // MCP tools endpoint - returns all available tools
  await page.route('**/api/v1/mcp/tools', async (route: Route) => {
    // Build tools by server name - the response format is Record<serverName, MCPTool[]>
    const allTools: Record<string, MockTool[]> = {};
    mockState.mcpServers.forEach((server) => {
      // Convert tool names to tool objects
      const toolObjects: MockTool[] = server.tools.map(toolName => ({
        name: toolName,
        description: toolName === 'read_file' ? 'Read a file from the filesystem' :
                     toolName === 'write_file' ? 'Write a file to the filesystem' :
                     toolName === 'list_directory' ? 'List directory contents' :
                     `Description for ${toolName}`,
      }));
      allTools[server.name] = toolObjects;
    });
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ tools: allTools, total: Object.values(allTools).flat().length }),
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
  closeSettings(): Promise<void>;
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
        // Click the delete button for the conversation (deletes immediately, no confirmation)
        await page.click(`[data-conversation-id="${conversationId}"] .delete-btn`);

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

      async closeSettings() {
        const closeButton = page.locator('.settings-close-btn, button[title="Close settings"]');
        await closeButton.click();
        await page.waitForSelector('.settings-modal-overlay', { state: 'hidden', timeout: 5000 });
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

        // Set up dialog handler BEFORE clicking (dialog appears synchronously after click)
        page.once('dialog', dialog => dialog.accept());

        await serverCard.locator('.remove-server-btn').click();

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
