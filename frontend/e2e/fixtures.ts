import { test as base, expect, Page } from '@playwright/test';

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
    // Intercept /api/v1/config requests before each test
    await page.route('**/api/v1/config', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockConfigResponse),
      });
    });

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
