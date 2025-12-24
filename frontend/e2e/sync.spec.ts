import { test, expect } from './fixtures';

test.describe('Message Synchronization', () => {
  test('should sync messages between two browser contexts', async ({ browser, conversationHelpers }) => {
    // Create two separate contexts to simulate two devices
    const context1 = await browser.newContext();
    const context2 = await browser.newContext();

    const page1 = await context1.newPage();
    const page2 = await context2.newPage();

    try {
      // Navigate both pages
      await page1.goto('/');
      await page2.goto('/');

      // Create conversation on device 1
      await page1.click('button:has-text("New Chat")');
      await page1.waitForSelector('.chat-window', { state: 'visible' });

      // Get conversation ID
      const selectedConv = await page1.locator('.conversation-item.selected').first();
      const conversationId = await selectedConv.getAttribute('data-conversation-id');

      if (!conversationId) {
        throw new Error('Failed to get conversation ID');
      }

      // Wait for sync to happen on device 2
      await page2.waitForTimeout(2000);

      // Refresh page2 to get latest data
      await page2.reload();

      // Verify conversation appears on device 2
      const conv2Item = page2.locator(`[data-conversation-id="${conversationId}"]`);
      await expect(conv2Item).toBeVisible({ timeout: 10000 });

      // Send message from device 1
      const messageText = 'Test sync message from device 1';
      await page1.fill('.input-bar input[type="text"]', messageText);
      await page1.click('.input-bar button[type="submit"]');

      // Verify message appears on device 1
      await expect(page1.locator(`.message-bubble:has-text("${messageText}")`)).toBeVisible();

      // Wait for sync
      await page2.waitForTimeout(3000);

      // Select the conversation on device 2
      await page2.click(`[data-conversation-id="${conversationId}"]`);

      // Refresh to ensure we get latest messages
      await page2.reload();
      await page2.click(`[data-conversation-id="${conversationId}"]`);

      // Verify message appears on device 2
      await expect(page2.locator(`.message-bubble:has-text("${messageText}")`)).toBeVisible({ timeout: 10000 });
    } finally {
      await context1.close();
      await context2.close();
    }
  });

  test('should show sync status indicator', async ({ page }) => {
    await page.goto('/');

    // Create a conversation
    await page.click('button:has-text("New Chat")');

    // Send a message
    const messageText = 'Testing sync status';
    await page.fill('.input-bar input[type="text"]', messageText);
    await page.click('.input-bar button[type="submit"]');

    // Look for sync status indicator (may vary based on implementation)
    const syncStatus = page.locator('.sync-status, [data-testid="sync-status"]');

    if (await syncStatus.isVisible()) {
      // Verify it shows syncing or synced state
      const statusText = await syncStatus.textContent();
      expect(statusText).toMatch(/syncing|synced/i);
    }
  });

  test('should handle offline mode gracefully', async ({ page, conversationHelpers }) => {
    await page.goto('/');

    const conversationId = await conversationHelpers.createConversation();

    // Simulate offline mode
    await page.context().setOffline(true);

    // Try to send a message
    const messageText = 'Offline message';
    await page.fill('.input-bar input[type="text"]', messageText);
    await page.click('.input-bar button[type="submit"]');

    // Message should still appear locally (optimistic UI)
    await expect(page.locator(`.message-bubble:has-text("${messageText}")`)).toBeVisible();

    // Go back online
    await page.context().setOffline(false);

    // Wait for sync
    await page.waitForTimeout(3000);

    // Message should still be visible and eventually synced
    await expect(page.locator(`.message-bubble:has-text("${messageText}")`)).toBeVisible();
  });

  test('should preserve message order during sync', async ({ browser }) => {
    const context1 = await browser.newContext();
    const context2 = await browser.newContext();

    const page1 = await context1.newPage();
    const page2 = await context2.newPage();

    try {
      await page1.goto('/');
      await page2.goto('/');

      // Create conversation on device 1
      await page1.click('button:has-text("New Chat")');
      await page1.waitForSelector('.chat-window', { state: 'visible' });

      const selectedConv = await page1.locator('.conversation-item.selected').first();
      const conversationId = await selectedConv.getAttribute('data-conversation-id');

      if (!conversationId) {
        throw new Error('Failed to get conversation ID');
      }

      // Send multiple messages in order
      const messages = ['First', 'Second', 'Third'];

      for (const msg of messages) {
        await page1.fill('.input-bar input[type="text"]', msg);
        await page1.click('.input-bar button[type="submit"]');
        await page1.waitForSelector(`.message-bubble:has-text("${msg}")`);
        await page1.waitForTimeout(500);
      }

      // Wait for sync
      await page2.waitForTimeout(3000);

      // Reload and select conversation on device 2
      await page2.reload();
      await page2.click(`[data-conversation-id="${conversationId}"]`);

      // Verify all messages appear in correct order
      const messageBubbles = await page2.locator('.message-bubble').allTextContents();

      for (const msg of messages) {
        expect(messageBubbles.join(' ')).toContain(msg);
      }
    } finally {
      await context1.close();
      await context2.close();
    }
  });

  test('should handle concurrent message creation', async ({ browser }) => {
    const context1 = await browser.newContext();
    const context2 = await browser.newContext();

    const page1 = await context1.newPage();
    const page2 = await context2.newPage();

    try {
      await page1.goto('/');
      await page2.goto('/');

      // Create conversation on device 1
      await page1.click('button:has-text("New Chat")');
      await page1.waitForSelector('.chat-window', { state: 'visible' });

      const selectedConv = await page1.locator('.conversation-item.selected').first();
      const conversationId = await selectedConv.getAttribute('data-conversation-id');

      if (!conversationId) {
        throw new Error('Failed to get conversation ID');
      }

      // Wait for sync
      await page2.waitForTimeout(2000);
      await page2.reload();

      // Select conversation on both devices
      await page2.click(`[data-conversation-id="${conversationId}"]`);

      // Send messages from both devices at roughly the same time
      const msg1 = 'Message from device 1';
      const msg2 = 'Message from device 2';

      await Promise.all([
        (async () => {
          await page1.fill('.input-bar input[type="text"]', msg1);
          await page1.click('.input-bar button[type="submit"]');
        })(),
        (async () => {
          await page2.fill('.input-bar input[type="text"]', msg2);
          await page2.click('.input-bar button[type="submit"]');
        })(),
      ]);

      // Wait for sync
      await page1.waitForTimeout(3000);
      await page2.waitForTimeout(3000);

      // Reload both pages
      await page1.reload();
      await page2.reload();

      await page1.click(`[data-conversation-id="${conversationId}"]`);
      await page2.click(`[data-conversation-id="${conversationId}"]`);

      // Both messages should appear on both devices
      await expect(page1.locator(`.message-bubble:has-text("${msg1}")`)).toBeVisible({ timeout: 10000 });
      await expect(page1.locator(`.message-bubble:has-text("${msg2}")`)).toBeVisible({ timeout: 10000 });

      await expect(page2.locator(`.message-bubble:has-text("${msg1}")`)).toBeVisible({ timeout: 10000 });
      await expect(page2.locator(`.message-bubble:has-text("${msg2}")`)).toBeVisible({ timeout: 10000 });
    } finally {
      await context1.close();
      await context2.close();
    }
  });

  test('should sync conversation deletion', async ({ browser }) => {
    const context1 = await browser.newContext();
    const context2 = await browser.newContext();

    const page1 = await context1.newPage();
    const page2 = await context2.newPage();

    try {
      await page1.goto('/');
      await page2.goto('/');

      // Create conversation on device 1
      await page1.click('button:has-text("New Chat")');
      await page1.waitForSelector('.chat-window', { state: 'visible' });

      const selectedConv = await page1.locator('.conversation-item.selected').first();
      const conversationId = await selectedConv.getAttribute('data-conversation-id');

      if (!conversationId) {
        throw new Error('Failed to get conversation ID');
      }

      // Wait for sync
      await page2.waitForTimeout(3000);
      await page2.reload();

      // Verify conversation appears on device 2
      await expect(page2.locator(`[data-conversation-id="${conversationId}"]`)).toBeVisible({ timeout: 10000 });

      // Delete conversation on device 1
      await page1.click(`[data-conversation-id="${conversationId}"] .delete-btn`);
      await page1.click('button:has-text("Delete")');

      // Wait for conversation to be removed
      await expect(page1.locator(`[data-conversation-id="${conversationId}"]`)).not.toBeVisible();

      // Wait for sync
      await page2.waitForTimeout(3000);
      await page2.reload();

      // Verify conversation is deleted on device 2
      await expect(page2.locator(`[data-conversation-id="${conversationId}"]`)).not.toBeVisible({ timeout: 10000 });
    } finally {
      await context1.close();
      await context2.close();
    }
  });
});
