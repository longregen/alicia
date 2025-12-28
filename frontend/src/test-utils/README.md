# Frontend Test Utilities

Shared test utilities for MessagePack + WebSocket sync protocol testing.

## Overview

This directory contains reusable test utilities for unit and integration tests:

- `websocket-mock.ts` - Mock WebSocket implementation with message recording
- `msgpack-fixtures.ts` - Pre-encoded MessagePack test data and fixtures
- `sync-protocol.ts` - Builders for sync protocol messages
- `sqlite-mock.ts` - In-memory SQLite database for testing

## Usage Examples

### WebSocket Mock

```typescript
import { createMockWebSocket, MockWebSocket } from '@/test-utils';
import { pack } from 'msgpackr';

// Create a mock WebSocket
const ws = createMockWebSocket('ws://localhost/test', {
  autoConnect: true,
  latency: 50, // Simulate 50ms latency
});

// Queue a response to be sent after next send()
const response = pack({ type: 'sync_response', payload: { messages: [] } });
ws.queueResponse(response);

// Send a message
const request = pack({ type: 'sync_request', payload: { messages: [] } });
ws.send(request);

// Wait for response
await ws.waitForSentMessage((msg) => msg.type === 'sync_request');

// Check sent messages
expect(ws.getSentMessages()).toHaveLength(1);
expect(ws.getLastSentMessage()).toEqual({ type: 'sync_request', ... });
```

### MessagePack Fixtures

```typescript
import {
  userMessageFixture,
  createUserMessage,
  createMessageBatch,
  encodedFixtures,
} from '@/test-utils';

// Use predefined fixtures
const message = userMessageFixture;

// Create custom messages
const customMessage = createUserMessage({
  contents: 'Custom message',
  sync_status: 'pending',
});

// Create multiple messages
const messages = createMessageBatch(10, 'conv-123', 'user');

// Use pre-encoded MessagePack data
const encoded = encodedFixtures.syncRequest;
```

### Sync Protocol Builders

```typescript
import { SyncProtocolBuilder, SyncFlowSimulator } from '@/test-utils';

// Build sync request
const messages = [userMessageFixture];
const request = SyncProtocolBuilder.createSyncRequest(messages);
const requestBinary = SyncProtocolBuilder.createSyncRequestBinary(messages);

// Build sync response
const syncedMessages = [{ local_id: 'local-1', server_id: 'msg-1', status: 'synced', message: userMessageFixture }];
const response = SyncProtocolBuilder.createSyncResponse(syncedMessages);

// Simulate sync flow
const simulator = new SyncFlowSimulator();
simulator.sendForSync(messages);
simulator.receiveSync(syncedMessages.map(sm => sm.message));
expect(simulator.getPendingMessages()).toHaveLength(0);
```

### SQLite Mock

```typescript
import { createTestDatabase, setupTestDatabase } from '@/test-utils';

describe('Database tests', () => {
  let db: TestDatabase;
  let cleanup: () => void;

  beforeEach(async () => {
    ({ db, cleanup } = await setupTestDatabase());
  });

  afterEach(() => {
    cleanup();
  });

  it('should store messages', () => {
    const database = db.getDb();
    database.run('INSERT INTO messages (...) VALUES (...)');

    expect(db.getRowCount('messages')).toBe(1);
    expect(db.getAllMessages()).toHaveLength(1);
  });
});
```

## Integration with Vitest

These utilities are designed to work seamlessly with Vitest:

```typescript
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { createMockWebSocket, createUserMessage } from '@/test-utils';

describe('useWebSocketSync', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should sync messages', async () => {
    const ws = createMockWebSocket();
    // ... test implementation
  });
});
```

## Type Safety

All utilities are fully typed with TypeScript and use the shared types from `@/types`:

- `Message` from `@/types/models`
- `Envelope`, `MessageType` from `@/types/protocol`
- `SyncRequest`, `SyncResponse` from `@/types/sync`
