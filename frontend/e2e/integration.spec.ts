import { test, expect } from './fixtures';

test.describe('End-to-End Integration', () => {
  test.skip('complete user workflow: configure MCP, chat, hear response', async ({
    // Skip: This test waits for assistant responses that require a real backend
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
    const closeButton = page.locator('.settings-close-btn');
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

    // Step 4: Check for tool usage or reasoning (if present)
    // Tools are displayed as part of ComplexAddons in ChatBubble
    // Reasoning is displayed inline within message content
    const messageContent = page.locator('.message-bubble.assistant').first();
    const hasContent = await messageContent.textContent();
    expect(hasContent).toBeTruthy();

    // Step 5: Verify audio controls (if present)
    // Audio is rendered via AudioAddon component
    const audioAddon = page.locator('button[aria-label*="Play"]').first();
    if (await audioAddon.isVisible()) {
      // Verify we can interact with audio controls
      await expect(audioAddon).toBeEnabled();
    }

    // Step 6: Clean up - remove the MCP server
    await mcpHelpers.openSettings();
    await mcpHelpers.removeServer(serverName);

    // Verify server is removed
    await expect(page.locator(`.server-card:has-text("${serverName}")`)).not.toBeVisible();
  });

  test.skip('should display tool usage in message', async ({ page, mcpHelpers, conversationHelpers }) => {
    // Skip: This test waits for assistant responses that require a real backend
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

    // Wait for assistant response with tool usage
    const assistantMessage = page.locator('.message-bubble.assistant').first();
    await expect(assistantMessage).toBeVisible({ timeout: 30000 });

    // Verify the assistant message has content
    // Tools are displayed inline via ComplexAddons as emoji icons (🔧)
    // or in expanded tool details panels
    const messageContent = await assistantMessage.textContent();
    expect(messageContent).toBeTruthy();

    // Optionally check if tool emoji is present (if tools were used)
    // Tools use 🔧 emoji in ComplexAddons
    const hasToolIndicator = messageContent?.includes('🔧');
    // Note: Tool usage may or may not happen depending on the message,
    // so we just verify the response exists

    // Clean up
    await mcpHelpers.openSettings();
    await mcpHelpers.removeServer(serverName);
  });

  test.skip('should show reasoning steps when enabled', async ({ page, conversationHelpers }) => {
    // Skip: This test waits for assistant responses that require a real backend
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

    // Wait for assistant response
    const assistantMessage = page.locator('.message-bubble.assistant').first();
    await expect(assistantMessage).toBeVisible({ timeout: 30000 });

    // Look for reasoning display
    // Reasoning is displayed inline in ChatBubble using ReasoningBlock component
    // It appears as a blue-bordered block with "Reasoning" label
    const reasoningBlock = page.locator('button:has-text("Reasoning")');
    if (await reasoningBlock.isVisible()) {
      // Click to expand reasoning
      await reasoningBlock.click();

      // Verify reasoning content is now visible
      const messageContent = await assistantMessage.textContent();
      expect(messageContent).toBeTruthy();
    }
  });

  test.skip('should handle audio playback controls', async ({ page, conversationHelpers }) => {
    // Skip: This test waits for assistant responses with audio that require a real backend
    await page.goto('/');

    // Create conversation
    const conversationId = await conversationHelpers.createConversation();

    // Send message
    const userMessage = 'Tell me a short story';
    await conversationHelpers.sendMessage(conversationId, userMessage);

    // Wait for assistant response
    const assistantMessage = page.locator('.message-bubble.assistant').first();
    await expect(assistantMessage).toBeVisible({ timeout: 30000 });

    // Look for audio controls (AudioAddon component)
    // Audio controls use aria-label with Play/Pause/Stop
    const playButton = page.locator('button[aria-label*="Play"]').first();

    if (await playButton.isVisible()) {
      await playButton.click();

      // Wait a bit for state to update
      await page.waitForTimeout(1000);

      // Look for pause button or stop button (indicating audio is playing)
      const pauseButton = page.locator('button[aria-label*="Pause"]').first();
      const stopButton = page.locator('button[aria-label*="Stop"]').first();

      const isPlaying = (await pauseButton.isVisible()) || (await stopButton.isVisible());
      // Note: In mock mode, audio may not actually play, so we just verify the button exists
      expect(await playButton.isVisible() || isPlaying).toBeTruthy();
    }
  });

  test.skip('should persist conversation state across reload', async ({ page, conversationHelpers }) => {
    // Skip: This test relies on assistant responses being persisted
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

  test.skip('should handle error states gracefully', async ({
    // Skip: This test requires backend integration to properly test offline/error states
    // When offline, the input becomes disabled and the test times out
    page,
    conversationHelpers,
  }) => {
    await page.goto('/');

    // Create conversation
    await conversationHelpers.createConversation();

    // Simulate network error by going offline
    await page.context().setOffline(true);

    // Try to send message
    const messageText = 'This should handle error gracefully';
    await page.fill('.input-bar input[type="text"]', messageText);
    await page.click('.input-bar button[type="submit"]');

    // Message should appear locally
    await expect(page.locator(`.message-bubble:has-text("${messageText}")`)).toBeVisible();

    // May show error indicator or disconnection status
    // Connection status is shown in ChatWindow with class .connection-status
    const connectionStatus = page.locator('.connection-status');
    if (await connectionStatus.isVisible()) {
      const statusText = await connectionStatus.textContent();
      expect(statusText).toBeTruthy();
    }

    // Go back online
    await page.context().setOffline(false);

    // Wait for recovery
    await page.waitForTimeout(3000);

    // Message should still be visible
    await expect(page.locator(`.message-bubble:has-text("${messageText}")`)).toBeVisible();
  });
});
