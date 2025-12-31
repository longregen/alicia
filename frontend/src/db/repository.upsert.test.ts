import { describe, it, expect, beforeEach, vi } from 'vitest';
import { messageRepository } from './repository';
import { Message } from '../types/models';
import * as sqlite from './sqlite';

/**
 * Test suite demonstrating the SQLite upsert duplicate message bug.
 *
 * ISSUE SUMMARY:
 * When a message is created with a local optimistic ID (e.g., "local_123") and the server
 * responds with a different server-assigned ID (e.g., "server_456"), the upsert function
 * fails to find the existing record and creates a duplicate message.
 *
 * ROOT CAUSE:
 * In repository.ts, the upsert() function only uses findById(message.id) to check for
 * existing records. When the server returns a different ID, this lookup fails and the
 * function inserts a new record instead of updating the existing one.
 *
 * EXPECTED BEHAVIOR:
 * When upserting a message with a local_id, the function should:
 * 1. First try to find by message.id
 * 2. If not found, try to find by message.local_id
 * 3. Update the found record or insert if neither search succeeds
 *
 * TEST STATUS:
 * The first test uses it.fails() to mark that it's expected to fail with current code.
 * When the bug is fixed, this test should be updated to use it() instead.
 */

// Mock the sqlite module
vi.mock('./sqlite', () => ({
  getDatabase: vi.fn(),
  scheduleSave: vi.fn(),
}));

describe('messageRepository.upsert - duplicate message bug', () => {
  let mockDb: {
    exec: ReturnType<typeof vi.fn>;
    run: ReturnType<typeof vi.fn>;
  };
  let insertedMessages: Map<string, any[]>;

  beforeEach(() => {
    vi.clearAllMocks();

    // Store inserted messages to simulate a real database
    insertedMessages = new Map();

    mockDb = {
      exec: vi.fn((sql: string, params?: any[]) => {
        // Simulate SELECT queries
        if (sql.includes('SELECT') && sql.includes('WHERE conversation_id = ?')) {
          const conversationId = params?.[0];
          const messages = Array.from(insertedMessages.values()).filter(
            (msg) => msg[1] === conversationId
          );
          if (messages.length === 0) return [];
          return [{ columns: ['id', 'conversation_id', 'sequence_number', 'role', 'contents', 'local_id', 'server_id', 'sync_status', 'retry_count', 'created_at', 'updated_at'], values: messages }];
        }
        if (sql.includes('SELECT') && sql.includes('WHERE id = ?')) {
          const id = params?.[0];
          const message = insertedMessages.get(id);
          if (!message) return [];
          return [{ columns: ['id', 'conversation_id', 'sequence_number', 'role', 'contents', 'local_id', 'server_id', 'sync_status', 'retry_count', 'created_at', 'updated_at'], values: [message] }];
        }
        if (sql.includes('SELECT') && sql.includes('WHERE local_id = ?')) {
          const localId = params?.[0];
          const message = Array.from(insertedMessages.values()).find((msg) => msg[5] === localId);
          if (!message) return [];
          return [{ columns: ['id', 'conversation_id', 'sequence_number', 'role', 'contents', 'local_id', 'server_id', 'sync_status', 'retry_count', 'created_at', 'updated_at'], values: [message] }];
        }
        return [];
      }),
      run: vi.fn((sql: string, params?: any[]) => {
        // Simulate INSERT queries
        if (sql.includes('INSERT INTO messages')) {
          const [id, conversation_id, sequence_number, role, contents, local_id, server_id, sync_status, retry_count, created_at, updated_at] = params || [];
          insertedMessages.set(id, [id, conversation_id, sequence_number, role, contents, local_id, server_id, sync_status, retry_count, created_at, updated_at]);
        }
        // Simulate UPDATE queries
        if (sql.includes('UPDATE messages')) {
          const id = params?.[params.length - 1];
          const existing = insertedMessages.get(id);
          if (existing) {
            // Parse SET clauses and update the record
            const [id_col, conversation_id, old_sequence_number, role, old_contents, old_local_id, old_server_id, old_sync_status, old_retry_count, created_at, _old_updated_at] = existing;

            // Extract field values from params (excluding the id at the end)
            const updateParams = params?.slice(0, -1) || [];
            let paramIdx = 0;

            // Build updated record based on SQL order
            let new_contents = old_contents;
            let new_sequence_number = old_sequence_number;
            let new_sync_status = old_sync_status;
            let new_local_id = old_local_id;
            let new_server_id = old_server_id;
            let new_retry_count = old_retry_count;

            // Parse fields in the order they appear in the SQL
            if (sql.includes('contents = ?')) {
              new_contents = updateParams[paramIdx++];
            }
            if (sql.includes('sequence_number = ?')) {
              new_sequence_number = updateParams[paramIdx++];
            }
            if (sql.includes('sync_status = ?')) {
              new_sync_status = updateParams[paramIdx++];
            }
            if (sql.includes('local_id = ?')) {
              new_local_id = updateParams[paramIdx++];
            }
            if (sql.includes('server_id = ?')) {
              new_server_id = updateParams[paramIdx++];
            }
            if (sql.includes('retry_count = ?')) {
              new_retry_count = updateParams[paramIdx++];
            }

            // updated_at is always updated
            const updated_at = updateParams[updateParams.length - 1];

            insertedMessages.set(id, [id_col, conversation_id, new_sequence_number, role, new_contents, new_local_id, new_server_id, new_sync_status, new_retry_count, created_at, updated_at]);
          }
        }
      }),
    };

    (sqlite.getDatabase as any).mockReturnValue(mockDb);
  });

  it('should prevent duplicate insertion when server ID differs from local ID', () => {
    const conversationId = 'conv-123';
    const timestamp = '2024-01-01T00:00:00Z';

    // Step 1: Insert a message with local optimistic ID
    // This simulates the client creating a message before server response
    const optimisticMessage: Message = {
      id: 'local_123',                  // Optimistic local ID
      conversation_id: conversationId,
      sequence_number: 1,
      role: 'user',
      contents: 'Hello, world!',
      local_id: 'local_123',           // Track the local ID
      sync_status: 'pending',
      created_at: timestamp,
      updated_at: timestamp,
    };

    messageRepository.insert(optimisticMessage);

    // Verify the message was inserted
    const messagesAfterInsert = messageRepository.findByConversation(conversationId);
    expect(messagesAfterInsert).toHaveLength(1);
    expect(messagesAfterInsert[0].id).toBe('local_123');
    expect(messagesAfterInsert[0].local_id).toBe('local_123');

    // Step 2: Simulate server response with different server-assigned ID
    // The server responds with its own ID, but keeps the same local_id for correlation
    const serverMessage: Message = {
      id: 'server_456',                 // Server-assigned ID (different!)
      conversation_id: conversationId,
      sequence_number: 1,
      role: 'user',
      contents: 'Hello, world!',
      local_id: 'local_123',           // Same local_id to correlate with original
      server_id: 'server_456',
      sync_status: 'synced',
      created_at: timestamp,
      updated_at: timestamp,
    };

    // Step 3: Call upsert with the server response
    // BUG: upsert uses findById(message.id), which searches for 'server_456'
    // It won't find the existing record with id='local_123'
    // So it will INSERT instead of UPDATE, creating a duplicate
    messageRepository.upsert(serverMessage);

    // Step 4: Verify the result
    const messagesAfterUpsert = messageRepository.findByConversation(conversationId);

    // FIXED: Should have 1 message (the original updated with server ID)
    expect(messagesAfterUpsert).toHaveLength(1);
    expect(messagesAfterUpsert[0].id).toBe('local_123'); // Original ID preserved
    expect(messagesAfterUpsert[0].local_id).toBe('local_123');
    expect(messagesAfterUpsert[0].server_id).toBe('server_456'); // Server ID added
    expect(messagesAfterUpsert[0].sync_status).toBe('synced'); // Status updated
  });

  it('should demonstrate that findByLocalId can locate the message', () => {
    const conversationId = 'conv-456';
    const timestamp = '2024-01-01T00:00:00Z';

    // Insert optimistic message
    const optimisticMessage: Message = {
      id: 'local_789',
      conversation_id: conversationId,
      sequence_number: 1,
      role: 'user',
      contents: 'Test message',
      local_id: 'local_789',
      sync_status: 'pending',
      created_at: timestamp,
      updated_at: timestamp,
    };

    messageRepository.insert(optimisticMessage);

    // Find by local_id (this works!)
    const foundByLocalId = messageRepository.findByLocalId('local_789');
    expect(foundByLocalId).not.toBeNull();
    expect(foundByLocalId?.id).toBe('local_789');
    expect(foundByLocalId?.local_id).toBe('local_789');

    // This demonstrates that we CAN find the message by local_id
    // The fix should make upsert try findByLocalId when findById fails
  });
});
