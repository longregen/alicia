import initSqlJs, { Database } from 'sql.js';

/**
 * In-memory SQLite database for testing
 */
export class TestDatabase {
  private db: Database | null = null;

  /**
   * Initialize the test database with schema
   */
  async init(): Promise<Database> {
    if (this.db) return this.db;

    const SQL = await initSqlJs({
      locateFile: (file: string) => `https://sql.js.org/dist/${file}`,
    });

    this.db = new SQL.Database();

    // Create schema matching the production schema
    this.db.run(`
      CREATE TABLE IF NOT EXISTS conversations (
        id TEXT PRIMARY KEY,
        title TEXT NOT NULL DEFAULT '',
        status TEXT NOT NULL DEFAULT 'active',
        created_at TEXT NOT NULL,
        updated_at TEXT NOT NULL
      );

      CREATE TABLE IF NOT EXISTS messages (
        id TEXT PRIMARY KEY,
        conversation_id TEXT NOT NULL,
        sequence_number INTEGER NOT NULL,
        role TEXT NOT NULL,
        contents TEXT NOT NULL DEFAULT '',
        local_id TEXT,
        sync_status TEXT DEFAULT 'synced',
        created_at TEXT NOT NULL,
        updated_at TEXT NOT NULL,
        FOREIGN KEY (conversation_id) REFERENCES conversations(id)
      );

      CREATE INDEX IF NOT EXISTS idx_messages_conv
        ON messages(conversation_id, sequence_number);
      CREATE INDEX IF NOT EXISTS idx_messages_sync
        ON messages(sync_status);
      CREATE INDEX IF NOT EXISTS idx_messages_local
        ON messages(local_id);
    `);

    return this.db;
  }

  /**
   * Get the database instance
   */
  getDb(): Database {
    if (!this.db) {
      throw new Error('Database not initialized. Call init() first.');
    }
    return this.db;
  }

  /**
   * Clear all data from tables
   */
  clearAll(): void {
    if (!this.db) return;

    this.db.run('DELETE FROM messages');
    this.db.run('DELETE FROM conversations');
  }

  /**
   * Drop all tables
   */
  dropAll(): void {
    if (!this.db) return;

    this.db.run('DROP TABLE IF EXISTS messages');
    this.db.run('DROP TABLE IF EXISTS conversations');
  }

  /**
   * Close and destroy the database
   */
  close(): void {
    if (this.db) {
      this.db.close();
      this.db = null;
    }
  }

  /**
   * Export database to Uint8Array for inspection
   */
  export(): Uint8Array {
    if (!this.db) {
      throw new Error('Database not initialized');
    }
    return this.db.export();
  }

  /**
   * Get table row count
   */
  getRowCount(tableName: 'conversations' | 'messages'): number {
    if (!this.db) return 0;

    const result = this.db.exec(`SELECT COUNT(*) as count FROM ${tableName}`);
    if (result.length === 0) return 0;

    return result[0].values[0][0] as number;
  }

  /**
   * Get all conversations
   */
  getAllConversations(): any[] {
    if (!this.db) return [];

    const result = this.db.exec(
      'SELECT * FROM conversations ORDER BY updated_at DESC'
    );

    if (result.length === 0) return [];

    return result[0].values.map((row) => ({
      id: row[0],
      title: row[1],
      status: row[2],
      created_at: row[3],
      updated_at: row[4],
    }));
  }

  /**
   * Get all messages
   */
  getAllMessages(): any[] {
    if (!this.db) return [];

    const result = this.db.exec(
      'SELECT * FROM messages ORDER BY sequence_number ASC'
    );

    if (result.length === 0) return [];

    return result[0].values.map((row) => ({
      id: row[0],
      conversation_id: row[1],
      sequence_number: row[2],
      role: row[3],
      contents: row[4],
      local_id: row[5],
      sync_status: row[6],
      created_at: row[7],
      updated_at: row[8],
    }));
  }

  /**
   * Get messages by conversation
   */
  getMessagesByConversation(conversationId: string): any[] {
    if (!this.db) return [];

    const result = this.db.exec(
      'SELECT * FROM messages WHERE conversation_id = ? ORDER BY sequence_number ASC',
      [conversationId]
    );

    if (result.length === 0) return [];

    return result[0].values.map((row) => ({
      id: row[0],
      conversation_id: row[1],
      sequence_number: row[2],
      role: row[3],
      contents: row[4],
      local_id: row[5],
      sync_status: row[6],
      created_at: row[7],
      updated_at: row[8],
    }));
  }

  /**
   * Get pending messages
   */
  getPendingMessages(): any[] {
    if (!this.db) return [];

    const result = this.db.exec(
      'SELECT * FROM messages WHERE sync_status = ? ORDER BY sequence_number ASC',
      ['pending']
    );

    if (result.length === 0) return [];

    return result[0].values.map((row) => ({
      id: row[0],
      conversation_id: row[1],
      sequence_number: row[2],
      role: row[3],
      contents: row[4],
      local_id: row[5],
      sync_status: row[6],
      created_at: row[7],
      updated_at: row[8],
    }));
  }
}

/**
 * Create a new test database instance
 */
export async function createTestDatabase(): Promise<TestDatabase> {
  const db = new TestDatabase();
  await db.init();
  return db;
}

/**
 * Mock the global database functions for testing
 */
export function mockDatabaseFunctions(testDb: TestDatabase) {
  const originalDb = { ...(globalThis as any).db };

  (globalThis as any).getDatabase = () => testDb.getDb();
  (globalThis as any).initDatabase = () => Promise.resolve(testDb.getDb());

  return {
    restore: () => {
      (globalThis as any).db = originalDb;
    },
  };
}

/**
 * Setup function for tests using the database
 */
export async function setupTestDatabase(): Promise<{
  db: TestDatabase;
  cleanup: () => void;
}> {
  const db = await createTestDatabase();

  const cleanup = () => {
    db.clearAll();
    db.close();
  };

  return { db, cleanup };
}
