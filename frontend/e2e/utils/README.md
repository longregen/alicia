# E2E Test Utilities

Playwright utilities for testing WebSocket sync protocol in end-to-end tests.

## Overview

This directory contains utilities for E2E testing:

- `websocket-helper.ts` - WebSocket connection monitoring and helpers
- `sync-assertions.ts` - Custom assertions for sync state verification

## Usage Examples

### WebSocket Helper

```typescript
import { test } from '@playwright/test';
import { createWebSocketHelper } from './utils';

test('should sync messages via WebSocket', async ({ page }) => {
  const wsHelper = createWebSocketHelper(page);

  // Setup monitoring before navigating
  await wsHelper.setupMonitoring();

  // Navigate to the app
  await page.goto('/');

  // Wait for WebSocket to connect
  await wsHelper.waitForConnection();

  // Perform actions that trigger WebSocket messages
  await page.click('button:has-text("Send Message")');

  // Wait for specific message to be sent
  await wsHelper.waitForSentMessage(
    (msg) => msg.type === 'sync_request',
    5000
  );

  // Check message counts
  const counts = await wsHelper.getMessageCount();
  expect(counts.sent).toBe(1);
  expect(counts.received).toBeGreaterThan(0);

  // Get all messages
  const messages = await wsHelper.getMessages();
  console.log('WebSocket messages:', messages);
});
```

### Sync Assertions

```typescript
import { test } from '@playwright/test';
import { createSyncAssertions } from './utils';

test('should show sync status', async ({ page }) => {
  await page.goto('/');

  const syncAssert = createSyncAssertions(page);

  // Send a message
  await page.fill('.input-bar input', 'Hello');
  await page.click('.input-bar button[type="submit"]');

  // Assert syncing started
  await syncAssert.assertSyncing();

  // Assert sync completed
  await syncAssert.assertSynced(10000);

  // Assert all messages are synced
  await syncAssert.assertAllMessagesSynced('conv-123');

  // Assert recent sync
  await syncAssert.assertRecentSync(60000); // within last 60s
});
```

### Multi-Device Sync Testing

```typescript
import { test } from '@playwright/test';
import { createSyncAssertions } from './utils';

test('should sync across devices', async ({ browser }) => {
  const context1 = await browser.newContext();
  const context2 = await browser.newContext();

  const page1 = await context1.newPage();
  const page2 = await context2.newPage();

  const syncAssert1 = createSyncAssertions(page1);
  const syncAssert2 = createSyncAssertions(page2);

  try {
    await page1.goto('/');
    await page2.goto('/');

    // Send message from device 1
    await page1.fill('.input-bar input', 'Test sync');
    await page1.click('.input-bar button[type="submit"]');

    // Assert synced on both devices
    await syncAssert1.assertMessageSyncedAcrossDevices(
      'Test sync',
      [page1, page2]
    );
  } finally {
    await context1.close();
    await context2.close();
  }
});
```

### WebSocket Message Inspection

```typescript
import { test } from '@playwright/test';
import {
  assertWebSocketMessageSent,
  assertWebSocketMessageReceived,
} from './utils';

test('should send and receive sync messages', async ({ page }) => {
  await page.goto('/');

  // Trigger sync
  await page.click('button:has-text("Sync Now")');

  // Assert sync request was sent
  await assertWebSocketMessageSent(
    page,
    (msg) => {
      try {
        const data = typeof msg.data === 'string'
          ? JSON.parse(msg.data)
          : msg.data;
        return data.type === 'sync_request';
      } catch {
        return false;
      }
    }
  );

  // Assert sync response was received
  await assertWebSocketMessageReceived(
    page,
    (msg) => {
      try {
        const data = typeof msg.data === 'string'
          ? JSON.parse(msg.data)
          : msg.data;
        return data.type === 'sync_response';
      } catch {
        return false;
      }
    }
  );
});
```

### Database State Verification

```typescript
import { test } from '@playwright/test';
import { assertDatabaseSyncState } from './utils';

test('should update database sync state', async ({ page }) => {
  await page.goto('/');

  const conversationId = 'conv-123';

  // Before sync - should have pending messages
  await assertDatabaseSyncState(page, conversationId, 2);

  // Trigger sync
  await page.click('button:has-text("Sync Now")');
  await page.waitForTimeout(2000);

  // After sync - should have no pending messages
  await assertDatabaseSyncState(page, conversationId, 0);
});
```

## Advanced Usage

### Mock WebSocket Responses

```typescript
import { mockWebSocketMessages } from './utils';

test('should handle custom WebSocket responses', async ({ page }) => {
  await mockWebSocketMessages(page, (message) => {
    // Intercept and modify messages
    if (message.type === 'sync_request') {
      return {
        type: 'sync_response',
        payload: { messages: [] },
      };
    }
    return null; // Don't modify
  });

  await page.goto('/');
  // WebSocket messages will be intercepted
});
```

### Wait for Sync Helper

```typescript
import { waitForWebSocketSync } from './utils';

test('should wait for sync to complete', async ({ page }) => {
  await page.goto('/');

  // Trigger action that causes sync
  await page.click('button:has-text("Create Conversation")');

  // Wait for sync to complete via WebSocket
  await waitForWebSocketSync(page, 10000);

  // Continue with test
  expect(await page.locator('.conversation-item').count()).toBe(1);
});
```

## Integration with Existing Fixtures

These utilities can be combined with existing fixtures from `fixtures.ts`:

```typescript
import { test } from './fixtures';
import { createSyncAssertions } from './utils';

test('should sync with conversation helpers', async ({
  page,
  conversationHelpers
}) => {
  await page.goto('/');

  const syncAssert = createSyncAssertions(page);
  const convId = await conversationHelpers.createConversation();

  await syncAssert.waitForSyncComplete();
  await syncAssert.assertConversationSynced(convId, [page]);
});
```
