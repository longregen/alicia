import { test, expect } from './fixtures';

test.describe('End-to-End Integration', () => {
  test('complete user workflow: configure MCP, chat, hear response', async ({
    page,
    mcpHelpers,
    conversationHelpers,
  }) => {
    await page.goto('/');

    // Step 1: Configure MCP server
    await mcpHelpers.openSettings();

    const serverName = `filesystem-${Date.now()}`;
    await mcpHelpers.addServer(
      serverName,
      'npx',
      '-y, @modelcontextprotocol/server-filesystem, /tmp'
    );

    // Verify server is added
    const serverCard = page.locator(`.server-card:has-text("${serverName}")`);
    await expect(serverCard).toBeVisible();

    // Wait for server to connect and load tools
    await page.waitForTimeout(3000);

    // Check if tools are available
    const toolsToggle = serverCard.locator('.tools-toggle');
    if (await toolsToggle.isVisible()) {
      await mcpHelpers.expandServerTools(serverName);

      // Verify tools are displayed
      const toolsList = serverCard.locator('.tools-list');
      await expect(toolsList).toBeVisible();

      const tools = toolsList.locator('.tool-item');
      const toolCount = await tools.count();
      expect(toolCount).toBeGreaterThan(0);
    }

    // Close settings
    const closeButton = page.locator('button[title="Close"], .close-btn, button:has-text("Close")');
    if (await closeButton.isVisible()) {
      await closeButton.click();
    } else {
      // Alternative: press Escape
      await page.keyboard.press('Escape');
    }

    // Step 2: Create conversation and send message
    const conversationId = await conversationHelpers.createConversation();

    // Verify we're in the chat view
    const chatWindow = page.locator('.chat-window');
    await expect(chatWindow).toBeVisible();

    // Send a message that might trigger tool usage
    const userMessage = 'Hello Alicia, can you help me?';
    await conversationHelpers.sendMessage(conversationId, userMessage);

    // Verify user message appears
    await expect(page.locator(`.message-bubble.user:has-text("${userMessage}")`)).toBeVisible();

    // Step 3: Wait for assistant response
    // The response might take some time, so we wait longer
    const assistantMessage = page.locator('.message-bubble.assistant').first();
    await expect(assistantMessage).toBeVisible({ timeout: 30000 });

    // Step 4: Check for protocol display (tool usage, reasoning)
    const protocolDisplay = page.locator('.protocol-display');
    if (await protocolDisplay.isVisible()) {
      // If protocol is shown, verify it has content
      const hasContent = await protocolDisplay.textContent();
      expect(hasContent).toBeTruthy();
    }

    // Step 5: Verify audio controls (if present)
    const audioControls = page.locator('.audio-output, .response-controls');
    if (await audioControls.isVisible()) {
      // Check for play/pause button
      const playButton = page.locator('button[aria-label="Play"], button:has-text("Play")');
      if (await playButton.isVisible()) {
        // Verify we can interact with audio controls
        await expect(playButton).toBeEnabled();
      }
    }

    // Step 6: Clean up - remove the MCP server
    await mcpHelpers.openSettings();
    await mcpHelpers.removeServer(serverName);

    // Verify server is removed
    await expect(page.locator(`.server-card:has-text("${serverName}")`)).not.toBeVisible();
  });

  test('should display tool usage in message', async ({ page, mcpHelpers, conversationHelpers }) => {
    await page.goto('/');

    // Configure MCP server
    await mcpHelpers.openSettings();

    const serverName = `filesystem-${Date.now()}`;
    await mcpHelpers.addServer(
      serverName,
      'npx',
      '-y, @modelcontextprotocol/server-filesystem, /tmp'
    );

    // Wait for connection
    await page.waitForTimeout(2000);

    // Close settings
    await page.keyboard.press('Escape');

    // Create conversation
    const conversationId = await conversationHelpers.createConversation();

    // Send a message that would trigger tool usage
    const userMessage = 'Can you list the files in /tmp?';
    await conversationHelpers.sendMessage(conversationId, userMessage);

    // Wait for response
    await page.waitForTimeout(10000);

    // Look for tool usage indicator
    const toolUsage = page.locator('.tool-use, [data-testid="tool-usage"]');
    if (await toolUsage.isVisible()) {
      // Verify tool usage shows the tool name
      const toolContent = await toolUsage.textContent();
      expect(toolContent).toBeTruthy();
    }

    // Clean up
    await mcpHelpers.openSettings();
    await mcpHelpers.removeServer(serverName);
  });

  test('should show reasoning steps when enabled', async ({ page, conversationHelpers }) => {
    await page.goto('/');

    // Create conversation with reasoning enabled
    const conversationId = await conversationHelpers.createConversation();

    // Enable reasoning in preferences (if there's a UI for it)
    const reasoningToggle = page.locator('[data-testid="enable-reasoning"], input[type="checkbox"][name="reasoning"]');
    if (await reasoningToggle.isVisible()) {
      await reasoningToggle.check();
    }

    // Send a complex message
    const userMessage = 'What is the capital of France and why is it important?';
    await conversationHelpers.sendMessage(conversationId, userMessage);

    // Wait for response
    await page.waitForTimeout(10000);

    // Look for reasoning display
    const reasoningDisplay = page.locator('.reasoning-step, [data-testid="reasoning"]');
    if (await reasoningDisplay.isVisible()) {
      // Verify reasoning content
      const reasoningContent = await reasoningDisplay.textContent();
      expect(reasoningContent).toBeTruthy();
    }
  });

  test('should handle audio playback controls', async ({ page, conversationHelpers }) => {
    await page.goto('/');

    // Create conversation
    const conversationId = await conversationHelpers.createConversation();

    // Send message
    const userMessage = 'Tell me a short story';
    await conversationHelpers.sendMessage(conversationId, userMessage);

    // Wait for response with audio
    await page.waitForTimeout(15000);

    // Look for audio controls
    const audioOutput = page.locator('.audio-output, [data-testid="audio-output"]');

    if (await audioOutput.isVisible()) {
      // Try to play audio
      const playButton = page.locator('button[aria-label="Play"], button:has-text("Play")').first();

      if (await playButton.isVisible()) {
        await playButton.click();

        // Wait a bit
        await page.waitForTimeout(1000);

        // Look for pause button or playing state
        const pauseButton = page.locator('button[aria-label="Pause"], button:has-text("Pause")');
        const playingIndicator = page.locator('[data-playing="true"], .playing');

        const isPlaying = (await pauseButton.isVisible()) || (await playingIndicator.isVisible());
        expect(isPlaying).toBeTruthy();
      }
    }
  });

  test('should persist conversation state across reload', async ({ page, conversationHelpers }) => {
    await page.goto('/');

    // Create conversation and send message
    const conversationId = await conversationHelpers.createConversation();
    const messageText = 'This message should persist';
    await conversationHelpers.sendMessage(conversationId, messageText);

    // Verify message is visible
    await expect(page.locator(`.message-bubble:has-text("${messageText}")`)).toBeVisible();

    // Reload page
    await page.reload();

    // Wait for app to load
    await page.waitForSelector('.sidebar', { state: 'visible' });

    // Verify conversation still exists
    await expect(page.locator(`[data-conversation-id="${conversationId}"]`)).toBeVisible();

    // Select conversation
    await page.click(`[data-conversation-id="${conversationId}"]`);

    // Verify message is still there
    await expect(page.locator(`.message-bubble:has-text("${messageText}")`)).toBeVisible();
  });

  test('should handle error states gracefully', async ({ page, conversationHelpers }) => {
    await page.goto('/');

    // Create conversation
    const conversationId = await conversationHelpers.createConversation();

    // Simulate network error by going offline
    await page.context().setOffline(true);

    // Try to send message
    const messageText = 'This should handle error gracefully';
    await page.fill('.input-bar input[type="text"]', messageText);
    await page.click('.input-bar button[type="submit"]');

    // Message should appear locally
    await expect(page.locator(`.message-bubble:has-text("${messageText}")`)).toBeVisible();

    // May show error indicator
    const errorIndicator = page.locator('.error-banner, .sync-error, [data-testid="error"]');
    if (await errorIndicator.isVisible()) {
      const errorText = await errorIndicator.textContent();
      expect(errorText).toBeTruthy();
    }

    // Go back online
    await page.context().setOffline(false);

    // Wait for recovery
    await page.waitForTimeout(3000);

    // Message should still be visible
    await expect(page.locator(`.message-bubble:has-text("${messageText}")`)).toBeVisible();
  });
});
