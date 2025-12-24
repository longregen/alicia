import { test as base, expect, Page, Locator } from '@playwright/test';
import { ArtifactCollector } from './artifact-collector';
import { FailureHandler } from './failure-handler';

export interface SidebarActions {
  createConversation(): Promise<string>;
  selectConversation(id: string): Promise<void>;
  deleteConversation(id: string): Promise<void>;
  getConversationList(): Promise<string[]>;
  waitForConversationCount(count: number): Promise<void>;
}

export interface ChatActions {
  sendMessage(text: string): Promise<void>;
  waitForUserMessage(text: string): Promise<Locator>;
  waitForAssistantResponse(timeout?: number): Promise<Locator>;
  getMessageCount(): Promise<number>;
  isTyping(): Promise<boolean>;
}

export interface VoiceActions {
  activateVoiceMode(): Promise<void>;
  deactivateVoiceMode(): Promise<void>;
  isVoiceModeActive(): Promise<boolean>;
  waitForConnection(state: 'connected' | 'connecting' | 'disconnected'): Promise<void>;
  startRecording(): Promise<void>;
  stopRecording(): Promise<void>;
}

export interface SettingsActions {
  open(): Promise<void>;
  close(): Promise<void>;
  addMcpServer(name: string, command: string, args?: string): Promise<void>;
  removeMcpServer(name: string): Promise<void>;
  waitForServerStatus(name: string, status: string): Promise<void>;
}

type TestFixtures = {
  sidebar: SidebarActions;
  chat: ChatActions;
  voice: VoiceActions;
  settings: SettingsActions;
  artifacts: ArtifactCollector;
  step: (name: string, fn: () => Promise<void>) => Promise<void>;
};

export const test = base.extend<TestFixtures>({
  artifacts: async ({ page }, use, testInfo) => {
    const artifactDir = process.env.ARTIFACT_DIR || './test-results';
    const collector = new ArtifactCollector(page, testInfo, artifactDir);

    await use(collector);

    await collector.saveAllLogs();

    if (testInfo.status !== testInfo.expectedStatus) {
      const failureHandler = new FailureHandler(page, testInfo, collector, artifactDir);
      await failureHandler.captureAll();
    }
  },

  step: async ({ artifacts }, use) => {
    const stepFn = async (name: string, fn: () => Promise<void>): Promise<void> => {
      await fn();
      await artifacts.screenshot(name);
    };
    await use(stepFn);
  },

  sidebar: async ({ page }, use) => {
    const actions: SidebarActions = {
      async createConversation() {
        await page.click('button:has-text("New Chat")');
        await page.waitForSelector('.chat-window', { state: 'visible' });

        const selected = page.locator('.conversation-item.selected').first();
        const id = await selected.getAttribute('data-conversation-id');

        if (!id) throw new Error('Failed to get conversation ID');
        return id;
      },

      async selectConversation(id: string) {
        await page.click(`[data-conversation-id="${id}"]`);
        await page.waitForSelector('.chat-window', { state: 'visible' });
      },

      async deleteConversation(id: string) {
        await page.click(`[data-conversation-id="${id}"] .delete-btn`);
        await page.click('button:has-text("Delete")');
        await page.waitForSelector(`[data-conversation-id="${id}"]`, {
          state: 'hidden',
          timeout: 5000,
        });
      },

      async getConversationList() {
        const items = page.locator('.conversation-item');
        const count = await items.count();
        const ids: string[] = [];

        for (let i = 0; i < count; i++) {
          const id = await items.nth(i).getAttribute('data-conversation-id');
          if (id) ids.push(id);
        }
        return ids;
      },

      async waitForConversationCount(count: number) {
        await expect(page.locator('.conversation-item')).toHaveCount(count, {
          timeout: 10000,
        });
      },
    };
    await use(actions);
  },

  chat: async ({ page }, use) => {
    const actions: ChatActions = {
      async sendMessage(text: string) {
        await page.fill('.input-bar input[type="text"]', text);
        await page.click('.input-bar button[type="submit"]');
      },

      async waitForUserMessage(text: string) {
        const msg = page.locator(`.message-bubble.user:has-text("${text}")`);
        await expect(msg).toBeVisible({ timeout: 5000 });
        return msg;
      },

      async waitForAssistantResponse(timeout = 30000) {
        const msg = page.locator('.message-bubble.assistant').first();
        await expect(msg).toBeVisible({ timeout });
        return msg;
      },

      async getMessageCount() {
        return page.locator('.message-bubble').count();
      },

      async isTyping() {
        const typing = page.locator('.typing-indicator, .streaming-response');
        return typing.isVisible();
      },
    };
    await use(actions);
  },

  voice: async ({ page }, use) => {
    await page.addInitScript(() => {
      navigator.mediaDevices.getUserMedia = async () => {
        const audioContext = new AudioContext();
        const oscillator = audioContext.createOscillator();
        const destination = audioContext.createMediaStreamDestination();
        oscillator.connect(destination);
        oscillator.start();
        return destination.stream;
      };
    });

    const actions: VoiceActions = {
      async activateVoiceMode() {
        const toggle = page.locator('.voice-mode-toggle');
        const isActive = await toggle.evaluate(el =>
          el.classList.contains('active')
        );

        if (!isActive) {
          await toggle.click();
          await page.waitForTimeout(500);
        }
      },

      async deactivateVoiceMode() {
        const toggle = page.locator('.voice-mode-toggle');
        const isActive = await toggle.evaluate(el =>
          el.classList.contains('active')
        );

        if (isActive) {
          await toggle.click();
          await page.waitForTimeout(500);
        }
      },

      async isVoiceModeActive() {
        const toggle = page.locator('.voice-mode-toggle');
        return toggle.evaluate(el => el.classList.contains('active'));
      },

      async waitForConnection(state) {
        const stateText = {
          connected: 'Connected',
          connecting: 'Connecting',
          disconnected: 'Disconnected',
        }[state];

        await page.waitForSelector(
          `.connection-status:has-text("${stateText}")`,
          { timeout: 15000 }
        );
      },

      async startRecording() {
        const btn = page.locator('.record-btn');
        const isRecording = await btn.evaluate(el =>
          el.classList.contains('recording')
        );

        if (!isRecording) {
          await btn.click();
        }
      },

      async stopRecording() {
        const btn = page.locator('.record-btn');
        const isRecording = await btn.evaluate(el =>
          el.classList.contains('recording')
        );

        if (isRecording) {
          await btn.click();
        }
      },
    };
    await use(actions);
  },

  settings: async ({ page }, use) => {
    const actions: SettingsActions = {
      async open() {
        await page.click('button[title="Settings"]');
        await page.waitForSelector('.mcp-settings', { state: 'visible' });
      },

      async close() {
        await page.keyboard.press('Escape');
        await page.waitForSelector('.mcp-settings', { state: 'hidden' });
      },

      async addMcpServer(name: string, command: string, args?: string) {
        await page.click('button:has-text("Add Server")');
        await page.fill('#server-name', name);
        await page.fill('#command', command);
        if (args) {
          await page.fill('#args', args);
        }
        await page.click('button[type="submit"]:has-text("Add Server")');
        await page.waitForSelector('.toast-success', { timeout: 5000 });
      },

      async removeMcpServer(name: string) {
        const card = page.locator(`.server-card:has-text("${name}")`);
        await card.locator('.remove-server-btn').click();
        await page.click('button:has-text("Confirm")');
        await page.waitForSelector(`.server-card:has-text("${name}")`, {
          state: 'hidden',
        });
      },

      async waitForServerStatus(name: string, status: string) {
        await page.waitForSelector(
          `.server-card:has-text("${name}") .status-badge:has-text("${status}")`,
          { timeout: 10000 }
        );
      },
    };
    await use(actions);
  },
});

export { expect };
