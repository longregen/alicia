import { getDatabase, scheduleSave } from './sqlite';
import { Message, Conversation } from '../types/models';
import type { SqlValue } from 'sql.js';

function rowToMessage(row: unknown[]): Message {
  return {
    id: row[0] as string,
    conversation_id: row[1] as string,
    sequence_number: row[2] as number,
    role: row[3] as 'user' | 'assistant' | 'system',
    contents: row[4] as string,
    local_id: row[5] as string | undefined,
    server_id: row[6] as string | undefined,
    sync_status: row[7] as 'pending' | 'synced' | 'conflict' | undefined,
    retry_count: row[8] as number | undefined,
    created_at: row[9] as string,
    updated_at: row[10] as string,
  };
}

function rowToConversation(row: unknown[]): Conversation {
  return {
    id: row[0] as string,
    title: row[1] as string,
    status: row[2] as 'active' | 'archived' | 'deleted',
    created_at: row[3] as string,
    updated_at: row[4] as string,
    // These fields are required by the type but not stored in DB
    // They're populated separately when needed for sync operations
    last_client_stanza_id: 0,
    last_server_stanza_id: 0,
  };
}

export const messageRepository = {
  findByConversation(conversationId: string): Message[] {
    const db = getDatabase();
    const results = db.exec(
      'SELECT id, conversation_id, sequence_number, role, contents, local_id, server_id, sync_status, retry_count, created_at, updated_at FROM messages WHERE conversation_id = ? ORDER BY sequence_number ASC',
      [conversationId]
    );

    if (results.length === 0) return [];

    return results[0].values.map(rowToMessage);
  },

  findById(id: string): Message | null {
    const db = getDatabase();
    const results = db.exec(
      'SELECT id, conversation_id, sequence_number, role, contents, local_id, server_id, sync_status, retry_count, created_at, updated_at FROM messages WHERE id = ?',
      [id]
    );

    if (results.length === 0 || results[0].values.length === 0) return null;

    return rowToMessage(results[0].values[0]);
  },

  findByLocalId(localId: string): Message | null {
    const db = getDatabase();
    const results = db.exec(
      'SELECT id, conversation_id, sequence_number, role, contents, local_id, server_id, sync_status, retry_count, created_at, updated_at FROM messages WHERE local_id = ?',
      [localId]
    );

    if (results.length === 0 || results[0].values.length === 0) return null;

    return rowToMessage(results[0].values[0]);
  },

  findByServerId(serverId: string): Message | null {
    const db = getDatabase();
    const results = db.exec(
      'SELECT id, conversation_id, sequence_number, role, contents, local_id, server_id, sync_status, retry_count, created_at, updated_at FROM messages WHERE server_id = ?',
      [serverId]
    );

    if (results.length === 0 || results[0].values.length === 0) return null;

    return rowToMessage(results[0].values[0]);
  },

  insert(message: Message): void {
    const db = getDatabase();
    db.run(
      'INSERT INTO messages (id, conversation_id, sequence_number, role, contents, local_id, server_id, sync_status, retry_count, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)',
      [
        message.id,
        message.conversation_id,
        message.sequence_number,
        message.role,
        message.contents,
        message.local_id || null,
        message.server_id || null,
        message.sync_status || 'synced',
        message.retry_count || 0,
        message.created_at,
        message.updated_at,
      ]
    );
    scheduleSave();
  },

  update(id: string, updates: Partial<Message>): void {
    const db = getDatabase();
    const setClauses: string[] = [];
    const values: SqlValue[] = [];

    if (updates.contents !== undefined) {
      setClauses.push('contents = ?');
      values.push(updates.contents);
    }
    if (updates.sequence_number !== undefined) {
      setClauses.push('sequence_number = ?');
      values.push(updates.sequence_number);
    }
    if (updates.sync_status !== undefined) {
      setClauses.push('sync_status = ?');
      values.push(updates.sync_status);
    }
    if (updates.local_id !== undefined) {
      setClauses.push('local_id = ?');
      values.push(updates.local_id);
    }
    if (updates.server_id !== undefined) {
      setClauses.push('server_id = ?');
      values.push(updates.server_id);
    }
    if (updates.retry_count !== undefined) {
      setClauses.push('retry_count = ?');
      values.push(updates.retry_count);
    }

    // Always update updated_at
    setClauses.push('updated_at = ?');
    values.push(new Date().toISOString());

    values.push(id);

    // updated_at is always added, so this check is for documentation clarity
    if (setClauses.length > 0) {
      db.run(
        `UPDATE messages SET ${setClauses.join(', ')} WHERE id = ?`,
        values
      );
      scheduleSave();
    }
  },

  delete(id: string): void {
    const db = getDatabase();
    db.run('DELETE FROM messages WHERE id = ?', [id]);
    scheduleSave();
  },

  getPending(conversationId: string): Message[] {
    const db = getDatabase();
    const results = db.exec(
      'SELECT id, conversation_id, sequence_number, role, contents, local_id, server_id, sync_status, retry_count, created_at, updated_at FROM messages WHERE sync_status = ? AND conversation_id = ?',
      ['pending', conversationId]
    );

    if (results.length === 0) return [];

    return results[0].values.map(rowToMessage);
  },

  upsert(message: Message): void {
    let existing = this.findById(message.id);
    if (!existing && message.local_id) {
      existing = this.findByLocalId(message.local_id);
    }
    if (existing) {
      this.update(existing.id, message);
    } else {
      this.insert(message);
    }
  },

  incrementRetryCount(id: string): void {
    const db = getDatabase();
    db.run(
      'UPDATE messages SET retry_count = retry_count + 1 WHERE id = ?',
      [id]
    );
    scheduleSave();
  },

  /**
   * Replace a message's ID (e.g., when server assigns a new ID to a locally-created message).
   * This is needed because SQLite doesn't allow updating primary keys directly.
   *
   * Handles race conditions:
   * - If newId already exists (WebSocket inserted it first), just deletes oldId
   * - If oldId doesn't exist, returns false
   *
   * Returns true if successful, false if the old message wasn't found.
   */
  replaceId(oldId: string, newId: string, updates?: Partial<Message>): boolean {
    const existing = this.findById(oldId);
    if (!existing) return false;

    const db = getDatabase();

    // Check if the new ID already exists (WebSocket race condition)
    const newIdExists = this.findById(newId);
    if (newIdExists) {
      // WebSocket already inserted the message with the server ID
      // Just delete the old optimistic row to avoid duplicates
      db.run('DELETE FROM messages WHERE id = ?', [oldId]);
      scheduleSave();
      return true;
    }

    // Delete the old row
    db.run('DELETE FROM messages WHERE id = ?', [oldId]);

    // Insert with the new ID and any updates
    const updated = { ...existing, ...updates, id: newId };
    db.run(
      'INSERT INTO messages (id, conversation_id, sequence_number, role, contents, local_id, server_id, sync_status, retry_count, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)',
      [
        updated.id,
        updated.conversation_id,
        updated.sequence_number,
        updated.role,
        updated.contents,
        updated.local_id || null,
        updated.server_id || null,
        updated.sync_status || 'synced',
        updated.retry_count || 0,
        updated.created_at,
        new Date().toISOString(),
      ]
    );
    scheduleSave();
    return true;
  },

  getRetryable(conversationId: string, maxRetries: number = 3): Message[] {
    const db = getDatabase();
    const results = db.exec(
      'SELECT id, conversation_id, sequence_number, role, contents, local_id, server_id, sync_status, retry_count, created_at, updated_at FROM messages WHERE sync_status = ? AND conversation_id = ? AND (retry_count IS NULL OR retry_count < ?) ORDER BY created_at ASC',
      ['pending', conversationId, maxRetries]
    );

    if (results.length === 0) return [];

    return results[0].values.map(rowToMessage);
  },

  deleteFailedOperations(maxRetries: number = 5): void {
    const db = getDatabase();
    db.run(
      'DELETE FROM messages WHERE sync_status = ? AND retry_count >= ?',
      ['pending', maxRetries]
    );
    scheduleSave();
  },
};

export const conversationRepository = {
  findAll(): Conversation[] {
    const db = getDatabase();
    const results = db.exec(
      'SELECT id, title, status, created_at, updated_at FROM conversations ORDER BY updated_at DESC'
    );

    if (results.length === 0) return [];

    return results[0].values.map(rowToConversation);
  },

  findById(id: string): Conversation | null {
    const db = getDatabase();
    const results = db.exec(
      'SELECT id, title, status, created_at, updated_at FROM conversations WHERE id = ?',
      [id]
    );

    if (results.length === 0 || results[0].values.length === 0) return null;

    return rowToConversation(results[0].values[0]);
  },

  insert(conversation: Conversation): void {
    const db = getDatabase();
    db.run(
      'INSERT INTO conversations (id, title, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)',
      [
        conversation.id,
        conversation.title,
        conversation.status,
        conversation.created_at,
        conversation.updated_at,
      ]
    );
    scheduleSave();
  },

  update(id: string, updates: Partial<Conversation>): void {
    const db = getDatabase();
    const setClauses: string[] = [];
    const values: SqlValue[] = [];

    if (updates.title !== undefined) {
      setClauses.push('title = ?');
      values.push(updates.title);
    }
    if (updates.status !== undefined) {
      setClauses.push('status = ?');
      values.push(updates.status);
    }

    // Always update updated_at
    setClauses.push('updated_at = ?');
    values.push(new Date().toISOString());

    values.push(id);

    // updated_at is always added, so this check is for documentation clarity
    if (setClauses.length > 0) {
      db.run(
        `UPDATE conversations SET ${setClauses.join(', ')} WHERE id = ?`,
        values
      );
      scheduleSave();
    }
  },

  upsert(conversation: Conversation): void {
    const existing = this.findById(conversation.id);
    if (existing) {
      this.update(conversation.id, conversation);
    } else {
      this.insert(conversation);
    }
  },
};
