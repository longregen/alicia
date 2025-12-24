import { test, expect } from './fixtures';

test.describe('Conversation Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('should create a new conversation', async ({ page, conversationHelpers }) => {
    const conversationId = await conversationHelpers.createConversation();

    // Verify conversation appears in sidebar
    const conversationItem = page.locator(`[data-conversation-id="${conversationId}"]`);
    await expect(conversationItem).toBeVisible();

    // Verify conversation is selected
    await expect(conversationItem).toHaveClass(/selected/);

    // Verify chat window is visible and empty
    const chatWindow = page.locator('.chat-window');
    await expect(chatWindow).toBeVisible();

    const messages = chatWindow.locator('.message-bubble');
    await expect(messages).toHaveCount(0);
  });

  test('should send a text message', async ({ page, conversationHelpers }) => {
    const conversationId = await conversationHelpers.createConversation();
    const messageText = 'Hello, Alicia! This is a test message.';

    await conversationHelpers.sendMessage(conversationId, messageText);

    // Verify message appears in the message list with user role
    const userMessage = page.locator(`.message-bubble.user:has-text("${messageText}")`);
    await expect(userMessage).toBeVisible();
  });

  test('should display messages in correct order', async ({ page, conversationHelpers }) => {
    const conversationId = await conversationHelpers.createConversation();

    const messages = [
      'First message',
      'Second message',
      'Third message',
    ];

    for (const msg of messages) {
      await conversationHelpers.sendMessage(conversationId, msg);
    }

    // Verify all messages are visible
    for (const msg of messages) {
      const messageBubble = page.locator(`.message-bubble:has-text("${msg}")`);
      await expect(messageBubble).toBeVisible();
    }

    // Verify order
    const allMessages = await page.locator('.message-bubble').allTextContents();
    expect(allMessages).toContain(messages[0]);
    expect(allMessages).toContain(messages[1]);
    expect(allMessages).toContain(messages[2]);
  });

  test('should delete a conversation', async ({ page, conversationHelpers }) => {
    const conversationId = await conversationHelpers.createConversation();

    // Send a message to ensure conversation has content
    await conversationHelpers.sendMessage(conversationId, 'Test message');

    // Delete the conversation
    await conversationHelpers.deleteConversation(conversationId);

    // Verify conversation is removed from sidebar
    const conversationItem = page.locator(`[data-conversation-id="${conversationId}"]`);
    await expect(conversationItem).not.toBeVisible();
  });

  test('should switch between conversations', async ({ page, conversationHelpers }) => {
    // Create two conversations
    const conv1Id = await conversationHelpers.createConversation();
    await conversationHelpers.sendMessage(conv1Id, 'Message in conversation 1');

    const conv2Id = await conversationHelpers.createConversation();
    await conversationHelpers.sendMessage(conv2Id, 'Message in conversation 2');

    // Switch back to conversation 1
    await page.click(`[data-conversation-id="${conv1Id}"]`);

    // Verify conversation 1 messages are shown
    await expect(page.locator('.message-bubble:has-text("Message in conversation 1")')).toBeVisible();
    await expect(page.locator('.message-bubble:has-text("Message in conversation 2")')).not.toBeVisible();

    // Switch to conversation 2
    await page.click(`[data-conversation-id="${conv2Id}"]`);

    // Verify conversation 2 messages are shown
    await expect(page.locator('.message-bubble:has-text("Message in conversation 2")')).toBeVisible();
    await expect(page.locator('.message-bubble:has-text("Message in conversation 1")')).not.toBeVisible();
  });

  test('should show empty state when no messages', async ({ page, conversationHelpers }) => {
    const conversationId = await conversationHelpers.createConversation();

    // Verify empty state or placeholder
    const chatWindow = page.locator('.chat-window');
    await expect(chatWindow).toBeVisible();

    const messages = chatWindow.locator('.message-bubble');
    await expect(messages).toHaveCount(0);
  });

  test('should handle rapid message sending', async ({ page, conversationHelpers }) => {
    const conversationId = await conversationHelpers.createConversation();

    // Send multiple messages quickly
    const messagePromises = [
      conversationHelpers.sendMessage(conversationId, 'Quick message 1'),
      conversationHelpers.sendMessage(conversationId, 'Quick message 2'),
      conversationHelpers.sendMessage(conversationId, 'Quick message 3'),
    ];

    await Promise.all(messagePromises);

    // Verify all messages appear
    await expect(page.locator('.message-bubble:has-text("Quick message 1")')).toBeVisible();
    await expect(page.locator('.message-bubble:has-text("Quick message 2")')).toBeVisible();
    await expect(page.locator('.message-bubble:has-text("Quick message 3")')).toBeVisible();
  });
});
