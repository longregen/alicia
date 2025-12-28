import initSqlJs, { Database } from 'sql.js';

const DB_NAME = 'alicia_messages';
const DB_VERSION = 1;

let db: Database | null = null;

export async function initDatabase(): Promise<Database> {
  if (db) return db;

  const SQL = await initSqlJs({
    locateFile: (file: string) => `https://sql.js.org/dist/${file}`
  });

  db = new SQL.Database();

  // Create schema
  db.run(`
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
      retry_count INTEGER DEFAULT 0,
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL,
      FOREIGN KEY (conversation_id) REFERENCES conversations(id)
    );

    CREATE INDEX IF NOT EXISTS idx_messages_conv ON messages(conversation_id, sequence_number);
  `);

  return db;
}

export function getDatabase(): Database {
  if (!db) {
    throw new Error('Database not initialized. Call initDatabase() first.');
  }
  return db;
}

export async function saveToIndexedDB(): Promise<void> {
  if (!db) return;

  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION);

    request.onerror = () => reject(request.error);

    request.onsuccess = () => {
      const idb = request.result;
      const transaction = idb.transaction(['database'], 'readwrite');
      const store = transaction.objectStore('database');

      // Export database to Uint8Array
      const data = db!.export();
      store.put(data, 'sqliteDb');

      transaction.oncomplete = () => {
        idb.close();
        resolve();
      };

      transaction.onerror = () => reject(transaction.error);
    };

    request.onupgradeneeded = (event) => {
      const idb = (event.target as IDBOpenDBRequest).result;
      if (!idb.objectStoreNames.contains('database')) {
        idb.createObjectStore('database');
      }
    };
  });
}

export async function loadFromIndexedDB(): Promise<void> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION);

    request.onerror = () => reject(request.error);

    request.onsuccess = () => {
      const idb = request.result;

      // Check if database object store exists
      if (!idb.objectStoreNames.contains('database')) {
        idb.close();
        resolve(); // No stored data yet
        return;
      }

      const transaction = idb.transaction(['database'], 'readonly');
      const store = transaction.objectStore('database');
      const getRequest = store.get('sqliteDb');

      getRequest.onsuccess = async () => {
        if (getRequest.result && db) {
          try {
            // Load stored data into existing database
            const SQL = await initSqlJs({
              locateFile: (file: string) => `https://sql.js.org/dist/${file}`
            });
            const loadedDb = new SQL.Database(getRequest.result);
            db = loadedDb;
          } catch (error) {
            console.error('Error loading database from IndexedDB:', error);
          }
        }
        idb.close();
        resolve();
      };

      getRequest.onerror = () => {
        idb.close();
        reject(getRequest.error);
      };
    };

    request.onupgradeneeded = (event) => {
      const idb = (event.target as IDBOpenDBRequest).result;
      if (!idb.objectStoreNames.contains('database')) {
        idb.createObjectStore('database');
      }
    };
  });
}

// Auto-save to IndexedDB periodically
let saveTimer: NodeJS.Timeout | null = null;

export function scheduleSave(delayMs = 1000): void {
  if (saveTimer) {
    clearTimeout(saveTimer);
  }

  saveTimer = setTimeout(() => {
    saveToIndexedDB().catch(error => {
      console.error('Failed to save database to IndexedDB:', error);
    });
  }, delayMs);
}
