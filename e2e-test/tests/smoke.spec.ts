import { test, expect } from '../lib/fixtures';

test.describe('Alicia Smoke Test', () => {
  test.describe.configure({ mode: 'serial' });

  let conversationId: string;

  test.beforeAll(async ({ page }) => {
    const maxAttempts = 30;
    const delayMs = 2000;

    for (let i = 0; i < maxAttempts; i++) {
      try {
        const response = await page.request.get('/health');
        if (response.ok()) {
          return;
        }
      } catch {
        // Server not ready yet
      }
      await page.waitForTimeout(delayMs);
    }
    throw new Error('Server did not become ready within timeout');
  });

  test('01 - Application loads successfully', async ({ page, artifacts }) => {
    await page.goto('/');

    await expect(page.locator('.app')).toBeVisible();
    await expect(page.locator('.sidebar')).toBeVisible();
    await expect(page.locator('button:has-text("New Chat")')).toBeVisible();

    await artifacts.screenshot('01-app-loaded');

    const errors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error' && !msg.text().includes('net::ERR_')) {
        errors.push(msg.text());
      }
    });

    await page.waitForTimeout(1000);
    expect(errors).toEqual([]);
  });

  test('02 - Create new conversation', async ({ page, sidebar, artifacts }) => {
    await page.goto('/');

    await artifacts.screenshot('02-before-new-conversation');

    conversationId = await sidebar.createConversation();

    await expect(
      page.locator(`[data-conversation-id="${conversationId}"]`)
    ).toBeVisible();

    await expect(page.locator('.chat-window')).toBeVisible();
    await expect(page.locator('.input-bar')).toBeVisible();

    await artifacts.screenshot('02-conversation-created');
  });

  test('03 - Send text message', async ({ page, chat, artifacts }) => {
    await page.goto('/');
    await page.click(`[data-conversation-id="${conversationId}"]`);

    const testMessage = 'Hello Alicia, this is a test message.';

    await chat.sendMessage(testMessage);

    const userMsg = await chat.waitForUserMessage(testMessage);
    await expect(userMsg).toBeVisible();

    await artifacts.screenshot('03-message-sent');
  });

  test('04 - Receive AI response', async ({ page, chat, artifacts }) => {
    await page.goto('/');
    await page.click(`[data-conversation-id="${conversationId}"]`);

    let msgCount = await chat.getMessageCount();

    if (msgCount < 2) {
      await chat.sendMessage('Please respond briefly.');
    }

    const response = await chat.waitForAssistantResponse(60000);
    await expect(response).toBeVisible();

    const responseText = await response.textContent();
    expect(responseText?.length).toBeGreaterThan(0);

    await artifacts.screenshot('04-ai-response-received');
  });

  test('05 - Navigate to settings', async ({ page, settings, artifacts }) => {
    await page.goto('/');

    await settings.open();

    await expect(page.locator('.mcp-settings h2')).toContainText('Settings');

    await artifacts.screenshot('05-settings-opened');

    await settings.close();
    await expect(page.locator('.mcp-settings')).not.toBeVisible();
  });

  test('06 - Voice mode activation', async ({ page, sidebar, voice, artifacts }) => {
    await page.goto('/');

    if (!conversationId) {
      conversationId = await sidebar.createConversation();
    } else {
      await page.click(`[data-conversation-id="${conversationId}"]`);
    }

    await artifacts.screenshot('06a-before-voice-mode');

    await voice.activateVoiceMode();

    const isActive = await voice.isVoiceModeActive();
    expect(isActive).toBe(true);

    await expect(page.locator('.voice-controls')).toBeVisible();
    await expect(page.locator('.audio-input')).toBeVisible();

    await artifacts.screenshot('06b-voice-mode-activated');

    const connectionStatus = page.locator('.connection-status');
    await expect(connectionStatus).toBeVisible();
    await expect(connectionStatus).toContainText(/Connecting|Connected/);

    await voice.deactivateVoiceMode();
    await expect(page.locator('.voice-controls')).not.toBeVisible();

    await artifacts.screenshot('06c-voice-mode-deactivated');
  });

  test('07 - Voice interaction flow', async ({ page, sidebar, voice, artifacts }) => {
    await page.goto('/');

    if (!conversationId) {
      conversationId = await sidebar.createConversation();
    } else {
      await page.click(`[data-conversation-id="${conversationId}"]`);
    }

    await voice.activateVoiceMode();

    try {
      await voice.waitForConnection('connected');
    } catch {
      console.log('LiveKit connection not available in test environment');
    }

    const recordBtn = page.locator('.record-btn');
    if (await recordBtn.isEnabled()) {
      await voice.startRecording();
      await artifacts.screenshot('07a-recording-started');

      await page.waitForTimeout(1000);

      await voice.stopRecording();
      await artifacts.screenshot('07b-recording-stopped');
    }

    const voiceSelector = page.locator('.voice-selector-toggle');
    if (await voiceSelector.isVisible()) {
      await voiceSelector.click();
      await expect(page.locator('.voice-selector-panel')).toBeVisible();

      await artifacts.screenshot('07c-voice-selector-open');

      await page.keyboard.press('Escape');
    }

    await voice.deactivateVoiceMode();
  });

  test('08 - Multiple conversations', async ({ page, sidebar, chat, artifacts }) => {
    await page.goto('/');

    const conv1 = await sidebar.createConversation();
    await chat.sendMessage('Message in conversation 1');

    await artifacts.screenshot('08a-first-conversation');

    const conv2 = await sidebar.createConversation();
    await chat.sendMessage('Message in conversation 2');

    await artifacts.screenshot('08b-second-conversation');

    await sidebar.selectConversation(conv1);

    await expect(
      page.locator('.message-bubble:has-text("Message in conversation 1")')
    ).toBeVisible();
    await expect(
      page.locator('.message-bubble:has-text("Message in conversation 2")')
    ).not.toBeVisible();

    await artifacts.screenshot('08c-switched-to-first');

    await sidebar.deleteConversation(conv2);
    await sidebar.deleteConversation(conv1);
  });

  test('09 - Error handling - offline mode', async ({ page, sidebar, chat, artifacts }) => {
    await page.goto('/');

    const id = await sidebar.createConversation();

    await page.context().setOffline(true);

    await chat.sendMessage('Offline test message');

    await artifacts.screenshot('09a-offline-message-sent');

    await expect(
      page.locator('.message-bubble:has-text("Offline test message")')
    ).toBeVisible();

    const errorBanner = page.locator('.error-banner, .sync-error');
    if (await errorBanner.isVisible()) {
      await artifacts.screenshot('09b-error-indicator');
    }

    await page.context().setOffline(false);
    await page.waitForTimeout(2000);

    await artifacts.screenshot('09c-back-online');

    await sidebar.deleteConversation(id);
  });

  test('10 - Error handling - invalid input', async ({ page, sidebar, artifacts }) => {
    await page.goto('/');
    await sidebar.createConversation();

    const input = page.locator('.input-bar input[type="text"]');
    const submitBtn = page.locator('.input-bar button[type="submit"]');

    await input.fill('');
    await submitBtn.click();

    await page.waitForTimeout(500);

    await artifacts.screenshot('10-empty-message-handled');

    await expect(input).toBeEnabled();
  });

  test('11 - Persistence across reload', async ({ page, sidebar, chat, artifacts }) => {
    await page.goto('/');

    const id = await sidebar.createConversation();
    const persistMsg = `Persist test ${Date.now()}`;
    await chat.sendMessage(persistMsg);

    await chat.waitForUserMessage(persistMsg);
    await artifacts.screenshot('11a-before-reload');

    await page.reload();
    await page.waitForSelector('.sidebar', { state: 'visible' });

    await expect(page.locator(`[data-conversation-id="${id}"]`)).toBeVisible();

    await sidebar.selectConversation(id);
    await expect(
      page.locator(`.message-bubble:has-text("${persistMsg}")`)
    ).toBeVisible();

    await artifacts.screenshot('11b-after-reload');

    await sidebar.deleteConversation(id);
  });

  test('12 - Cleanup and final state', async ({ page, artifacts }) => {
    await page.goto('/');

    const items = page.locator('.conversation-item');
    const count = await items.count();

    for (let i = count - 1; i >= 0; i--) {
      const item = items.nth(i);
      const id = await item.getAttribute('data-conversation-id');

      if (id) {
        await item.locator('.delete-btn').click();
        await page.click('button:has-text("Delete")');
        await page.waitForTimeout(300);
      }
    }

    await artifacts.screenshot('12-final-clean-state');
  });
});
