import { test, expect } from './fixtures';

test.describe('Console Errors', () => {
  test('should have no console errors on page load', async ({ page }) => {
    const consoleErrors: string[] = [];
    const consoleWarnings: string[] = [];

    // Capture console messages before navigating
    page.on('console', (msg) => {
      const text = msg.text();
      if (msg.type() === 'error') {
        // Ignore known benign errors (e.g., failed network requests in test env)
        if (
          !text.includes('net::ERR_') &&
          !text.includes('Failed to load resource') &&
          !text.includes('WebSocket')
        ) {
          consoleErrors.push(text);
        }
      } else if (msg.type() === 'warning') {
        // Ignore React dev mode warnings about strict mode double-renders
        if (!text.includes('ReactDOM.render is no longer supported')) {
          consoleWarnings.push(text);
        }
      }
    });

    // Also capture uncaught exceptions
    page.on('pageerror', (err) => {
      consoleErrors.push(`Uncaught: ${err.message}`);
    });

    // Navigate to the app
    await page.goto('/');

    // Wait for the app to fully load
    await page.waitForSelector('.app', { state: 'visible', timeout: 10000 });

    // Give React time to finish any async effects
    await page.waitForTimeout(1000);

    // Assert no console errors
    expect(consoleErrors, `Console errors found: ${consoleErrors.join('\n')}`).toEqual([]);
  });

  test('should have no console errors when creating a conversation', async ({
    page,
    conversationHelpers,
  }) => {
    const consoleErrors: string[] = [];

    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        const text = msg.text();
        if (
          !text.includes('net::ERR_') &&
          !text.includes('Failed to load resource') &&
          !text.includes('WebSocket')
        ) {
          consoleErrors.push(text);
        }
      }
    });

    page.on('pageerror', (err) => {
      consoleErrors.push(`Uncaught: ${err.message}`);
    });

    await page.goto('/');
    await page.waitForSelector('.app', { state: 'visible' });

    // Create a new conversation
    await conversationHelpers.createConversation();

    // Wait for UI to settle
    await page.waitForTimeout(500);

    expect(consoleErrors, `Console errors found: ${consoleErrors.join('\n')}`).toEqual([]);
  });

  test('should have no console errors when navigating between conversations', async ({
    page,
    conversationHelpers,
  }) => {
    const consoleErrors: string[] = [];

    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        const text = msg.text();
        if (
          !text.includes('net::ERR_') &&
          !text.includes('Failed to load resource') &&
          !text.includes('WebSocket')
        ) {
          consoleErrors.push(text);
        }
      }
    });

    page.on('pageerror', (err) => {
      consoleErrors.push(`Uncaught: ${err.message}`);
    });

    await page.goto('/');
    await page.waitForSelector('.app', { state: 'visible' });

    // Create first conversation
    const conv1 = await conversationHelpers.createConversation();
    await page.waitForTimeout(300);

    // Create second conversation
    const conv2 = await conversationHelpers.createConversation();
    await page.waitForTimeout(300);

    // Navigate back to first conversation
    await page.click(`[data-conversation-id="${conv1}"]`);
    await page.keyboard.press('Escape');
    await page.waitForTimeout(300);

    // Navigate to second conversation
    await page.click(`[data-conversation-id="${conv2}"]`);
    await page.keyboard.press('Escape');
    await page.waitForTimeout(300);

    expect(consoleErrors, `Console errors found: ${consoleErrors.join('\n')}`).toEqual([]);
  });

  test('should have no console errors when opening settings', async ({ page, mcpHelpers }) => {
    const consoleErrors: string[] = [];

    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        const text = msg.text();
        if (
          !text.includes('net::ERR_') &&
          !text.includes('Failed to load resource') &&
          !text.includes('WebSocket')
        ) {
          consoleErrors.push(text);
        }
      }
    });

    page.on('pageerror', (err) => {
      consoleErrors.push(`Uncaught: ${err.message}`);
    });

    await page.goto('/');
    await page.waitForSelector('.app', { state: 'visible' });

    // Open settings
    await mcpHelpers.openSettings();
    await page.waitForTimeout(500);

    // Close settings
    await page.keyboard.press('Escape');
    await page.waitForTimeout(300);

    expect(consoleErrors, `Console errors found: ${consoleErrors.join('\n')}`).toEqual([]);
  });
});
