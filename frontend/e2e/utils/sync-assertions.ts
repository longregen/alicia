import { expect, Page } from '@playwright/test';

/**
 * Custom assertions for sync state verification in E2E tests
 */
export class SyncAssertions {
  constructor(private page: Page) {}

  /**
   * Assert that sync is in progress
   */
  async assertSyncing(timeoutMs = 5000): Promise<void> {
    await expect(
      this.page.locator('.sync-status, [data-testid="sync-status"]')
    ).toContainText(/syncing/i, { timeout: timeoutMs });
  }

  /**
   * Assert that sync has completed
   */
  async assertSynced(timeoutMs = 10000): Promise<void> {
    await expect(
      this.page.locator('.sync-status, [data-testid="sync-status"]')
    ).toContainText(/synced/i, { timeout: timeoutMs });
  }

  /**
   * Assert that sync has failed
   */
  async assertSyncError(timeoutMs = 5000): Promise<void> {
    await expect(
      this.page.locator('.sync-status, [data-testid="sync-status"]')
    ).toContainText(/error|failed/i, { timeout: timeoutMs });
  }

  /**
   * Assert that a message has a specific sync status
   */
  async assertMessageSyncStatus(
    messageId: string,
    status: 'pending' | 'synced' | 'conflict',
    timeoutMs = 5000
  ): Promise<void> {
    const statusLocator = this.page.locator(
      `[data-message-id="${messageId}"] .sync-status, [data-message-id="${messageId}"][data-sync-status]`
    );

    await statusLocator.waitFor({ state: 'visible', timeout: timeoutMs });

    const actualStatus = await statusLocator.evaluate((el) => {
      return (
        el.getAttribute('data-sync-status') ||
        el.textContent?.toLowerCase() ||
        ''
      );
    });

    expect(actualStatus).toContain(status);
  }

  /**
   * Assert that all messages in a conversation are synced
   */
  async assertAllMessagesSynced(
    conversationId: string,
    timeoutMs = 10000
  ): Promise<void> {
    const startTime = Date.now();

    while (Date.now() - startTime < timeoutMs) {
      const pendingCount = await this.page
        .locator(
          `[data-conversation-id="${conversationId}"] [data-sync-status="pending"]`
        )
        .count();

      if (pendingCount === 0) {
        return;
      }

      await this.page.waitForTimeout(500);
    }

    throw new Error(
      `Not all messages synced within ${timeoutMs}ms for conversation ${conversationId}`
    );
  }

  /**
   * Assert that a message exists and is synced across devices
   */
  async assertMessageSyncedAcrossDevices(
    messageText: string,
    pages: Page[],
    timeoutMs = 10000
  ): Promise<void> {
    for (const page of pages) {
      await expect(
        page.locator(`.message-bubble:has-text("${messageText}")`)
      ).toBeVisible({ timeout: timeoutMs });

      // Verify it's marked as synced
      const syncStatus = await page
        .locator(`.message-bubble:has-text("${messageText}")`)
        .getAttribute('data-sync-status');

      expect(syncStatus).toBe('synced');
    }
  }

  /**
   * Assert sync conflict is displayed
   */
  async assertSyncConflict(messageId: string, timeoutMs = 5000): Promise<void> {
    await expect(
      this.page.locator(
        `[data-message-id="${messageId}"] .conflict-indicator, [data-message-id="${messageId}"][data-sync-status="conflict"]`
      )
    ).toBeVisible({ timeout: timeoutMs });
  }

  /**
   * Assert last sync time is recent
   */
  async assertRecentSync(maxAgeMs = 60000): Promise<void> {
    const lastSyncTime = await this.getLastSyncTime();

    if (!lastSyncTime) {
      throw new Error('No last sync time found');
    }

    const ageMs = Date.now() - lastSyncTime.getTime();
    expect(ageMs).toBeLessThan(maxAgeMs);
  }

  /**
   * Assert conversation is synced across devices
   */
  async assertConversationSynced(
    conversationId: string,
    pages: Page[],
    timeoutMs = 10000
  ): Promise<void> {
    for (const page of pages) {
      await expect(
        page.locator(`[data-conversation-id="${conversationId}"]`)
      ).toBeVisible({ timeout: timeoutMs });
    }
  }

  /**
   * Assert message order is correct
   */
  async assertMessageOrder(
    conversationId: string,
    expectedOrder: string[]
  ): Promise<void> {
    const messageElements = this.page.locator(
      `[data-conversation-id="${conversationId}"] .message-bubble`
    );

    const count = await messageElements.count();
    expect(count).toBe(expectedOrder.length);

    for (let i = 0; i < count; i++) {
      const text = await messageElements.nth(i).textContent();
      expect(text).toContain(expectedOrder[i]);
    }
  }

  /**
   * Assert pending message count
   */
  async assertPendingMessageCount(
    expected: number,
    conversationId?: string
  ): Promise<void> {
    const selector = conversationId
      ? `[data-conversation-id="${conversationId}"] [data-sync-status="pending"]`
      : '[data-sync-status="pending"]';

    const count = await this.page.locator(selector).count();
    expect(count).toBe(expected);
  }

  /**
   * Wait for sync to complete and assert success
   */
  async waitForSyncComplete(timeoutMs = 10000): Promise<void> {
    await this.assertSyncing(5000);
    await this.assertSynced(timeoutMs);
  }

  /**
   * Get last sync time from UI
   */
  private async getLastSyncTime(): Promise<Date | null> {
    const syncTimeText = await this.page
      .locator('.last-sync-time, [data-testid="last-sync-time"]')
      .textContent()
      .catch(() => null);

    if (!syncTimeText) return null;

    // Parse ISO timestamp or relative time
    const isoMatch = syncTimeText.match(/\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/);
    if (isoMatch) {
      return new Date(isoMatch[0]);
    }

    // Try to parse "X seconds/minutes ago"
    const relativeMatch = syncTimeText.match(/(\d+)\s*(second|minute|hour)s?\s*ago/i);
    if (relativeMatch) {
      const value = parseInt(relativeMatch[1]);
      const unit = relativeMatch[2].toLowerCase();
      const now = new Date();

      if (unit === 'second') {
        return new Date(now.getTime() - value * 1000);
      } else if (unit === 'minute') {
        return new Date(now.getTime() - value * 60000);
      } else if (unit === 'hour') {
        return new Date(now.getTime() - value * 3600000);
      }
    }

    return null;
  }
}

/**
 * Create sync assertions for a page
 */
export function createSyncAssertions(page: Page): SyncAssertions {
  return new SyncAssertions(page);
}

/**
 * Assert that WebSocket is connected
 */
export async function assertWebSocketConnected(
  page: Page,
  timeoutMs = 10000
): Promise<void> {
  const connected = await page.evaluate(
    () => (window as any).__wsConnected === true
  );

  if (!connected) {
    throw new Error('WebSocket is not connected');
  }
}

/**
 * Assert that WebSocket message was sent
 */
export async function assertWebSocketMessageSent(
  page: Page,
  predicate: (msg: any) => boolean,
  timeoutMs = 5000
): Promise<void> {
  const startTime = Date.now();

  while (Date.now() - startTime < timeoutMs) {
    const messages = await page.evaluate(
      () => (window as any).__wsMessages || []
    );

    const sentMessages = messages.filter((msg: any) => msg.type === 'sent');
    const found = sentMessages.some(predicate);

    if (found) {
      return;
    }

    await page.waitForTimeout(100);
  }

  throw new Error('WebSocket message was not sent');
}

/**
 * Assert that WebSocket message was received
 */
export async function assertWebSocketMessageReceived(
  page: Page,
  predicate: (msg: any) => boolean,
  timeoutMs = 5000
): Promise<void> {
  const startTime = Date.now();

  while (Date.now() - startTime < timeoutMs) {
    const messages = await page.evaluate(
      () => (window as any).__wsMessages || []
    );

    const receivedMessages = messages.filter(
      (msg: any) => msg.type === 'received'
    );
    const found = receivedMessages.some(predicate);

    if (found) {
      return;
    }

    await page.waitForTimeout(100);
  }

  throw new Error('WebSocket message was not received');
}

/**
 * Assert sync state in local database
 */
export async function assertDatabaseSyncState(
  page: Page,
  conversationId: string,
  expectedPending: number
): Promise<void> {
  const pendingCount = await page.evaluate(
    (convId) => {
      // Access the in-memory SQLite database
      const db = (window as any).getDatabase?.();
      if (!db) return -1;

      const results = db.exec(
        'SELECT COUNT(*) FROM messages WHERE conversation_id = ? AND sync_status = ?',
        [convId, 'pending']
      );

      if (results.length === 0) return 0;
      return results[0].values[0][0];
    },
    conversationId
  );

  expect(pendingCount).toBe(expectedPending);
}
