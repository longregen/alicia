import initSqlJs, { Database } from 'sql.js';

interface ConversationRow {
  id: string;
  title: string;
  status: string;
  created_at: string;
  updated_at: string;
}

interface MessageRow {
  id: string;
  conversation_id: string;
  sequence_number: number;
  role: string;
  contents: string;
  local_id: string | null;
  sync_status: string;
  created_at: string;
  updated_at: string;
}

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
  getAllConversations(): ConversationRow[] {
    if (!this.db) return [];

    const result = this.db.exec(
      'SELECT * FROM conversations ORDER BY updated_at DESC'
    );

    if (result.length === 0) return [];

    return result[0].values.map((row) => ({
      id: row[0] as string,
      title: row[1] as string,
      status: row[2] as string,
      created_at: row[3] as string,
      updated_at: row[4] as string,
    }));
  }

  /**
   * Get all messages
   */
  getAllMessages(): MessageRow[] {
    if (!this.db) return [];

    const result = this.db.exec(
      'SELECT * FROM messages ORDER BY sequence_number ASC'
    );

    if (result.length === 0) return [];

    return result[0].values.map((row) => ({
      id: row[0] as string,
      conversation_id: row[1] as string,
      sequence_number: row[2] as number,
      role: row[3] as string,
      contents: row[4] as string,
      local_id: row[5] as string | null,
      sync_status: row[6] as string,
      created_at: row[7] as string,
      updated_at: row[8] as string,
    }));
  }

  /**
   * Get messages by conversation
   */
  getMessagesByConversation(conversationId: string): MessageRow[] {
    if (!this.db) return [];

    const result = this.db.exec(
      'SELECT * FROM messages WHERE conversation_id = ? ORDER BY sequence_number ASC',
      [conversationId]
    );

    if (result.length === 0) return [];

    return result[0].values.map((row) => ({
      id: row[0] as string,
      conversation_id: row[1] as string,
      sequence_number: row[2] as number,
      role: row[3] as string,
      contents: row[4] as string,
      local_id: row[5] as string | null,
      sync_status: row[6] as string,
      created_at: row[7] as string,
      updated_at: row[8] as string,
    }));
  }

  /**
   * Get pending messages
   */
  getPendingMessages(): MessageRow[] {
    if (!this.db) return [];

    const result = this.db.exec(
      'SELECT * FROM messages WHERE sync_status = ? ORDER BY sequence_number ASC',
      ['pending']
    );

    if (result.length === 0) return [];

    return result[0].values.map((row) => ({
      id: row[0] as string,
      conversation_id: row[1] as string,
      sequence_number: row[2] as number,
      role: row[3] as string,
      contents: row[4] as string,
      local_id: row[5] as string | null,
      sync_status: row[6] as string,
      created_at: row[7] as string,
      updated_at: row[8] as string,
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
  interface GlobalWithDb {
    db?: Database;
    getDatabase?: () => Database;
    initDatabase?: () => Promise<Database>;
  }

  const globalWithDb = globalThis as unknown as GlobalWithDb;
  const originalDb = globalWithDb.db;

  globalWithDb.getDatabase = () => testDb.getDb();
  globalWithDb.initDatabase = () => Promise.resolve(testDb.getDb());

  return {
    restore: () => {
      globalWithDb.db = originalDb;
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
