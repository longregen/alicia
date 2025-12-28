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
    sync_status: row[6] as 'pending' | 'synced' | 'conflict' | undefined,
    retry_count: row[7] as number | undefined,
    created_at: row[8] as string,
    updated_at: row[9] as string,
  };
}

function rowToConversation(row: unknown[]): Conversation {
  return {
    id: row[0] as string,
    title: row[1] as string,
    status: row[2] as 'active' | 'archived' | 'deleted',
    created_at: row[3] as string,
    updated_at: row[4] as string,
    last_client_stanza_id: 0,
    last_server_stanza_id: 0,
  };
}

export const messageRepository = {
  findByConversation(conversationId: string): Message[] {
    const db = getDatabase();
    const results = db.exec(
      'SELECT id, conversation_id, sequence_number, role, contents, local_id, sync_status, retry_count, created_at, updated_at FROM messages WHERE conversation_id = ? ORDER BY sequence_number ASC',
      [conversationId]
    );

    if (results.length === 0) return [];

    return results[0].values.map(rowToMessage);
  },

  findById(id: string): Message | null {
    const db = getDatabase();
    const results = db.exec(
      'SELECT id, conversation_id, sequence_number, role, contents, local_id, sync_status, retry_count, created_at, updated_at FROM messages WHERE id = ?',
      [id]
    );

    if (results.length === 0 || results[0].values.length === 0) return null;

    return rowToMessage(results[0].values[0]);
  },

  insert(message: Message): void {
    const db = getDatabase();
    db.run(
      'INSERT INTO messages (id, conversation_id, sequence_number, role, contents, local_id, sync_status, retry_count, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)',
      [
        message.id,
        message.conversation_id,
        message.sequence_number,
        message.role,
        message.contents,
        message.local_id || null,
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
    if (updates.retry_count !== undefined) {
      setClauses.push('retry_count = ?');
      values.push(updates.retry_count);
    }

    // Always update updated_at
    setClauses.push('updated_at = ?');
    values.push(new Date().toISOString());

    values.push(id);

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

  getPending(): Message[] {
    const db = getDatabase();
    const results = db.exec(
      'SELECT id, conversation_id, sequence_number, role, contents, local_id, sync_status, retry_count, created_at, updated_at FROM messages WHERE sync_status = ?',
      ['pending']
    );

    if (results.length === 0) return [];

    return results[0].values.map(rowToMessage);
  },

  upsert(message: Message): void {
    const existing = this.findById(message.id);
    if (existing) {
      this.update(message.id, message);
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

  getRetryable(maxRetries: number = 3): Message[] {
    const db = getDatabase();
    const results = db.exec(
      'SELECT id, conversation_id, sequence_number, role, contents, local_id, sync_status, retry_count, created_at, updated_at FROM messages WHERE sync_status = ? AND (retry_count IS NULL OR retry_count < ?) ORDER BY created_at ASC',
      ['pending', maxRetries]
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
