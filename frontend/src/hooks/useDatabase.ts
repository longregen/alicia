import { useState, useEffect } from 'react';
import { initDatabase, loadFromIndexedDB } from '../db/sqlite';

export function useDatabase() {
  const [isReady, setIsReady] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    let mounted = true;

    async function init() {
      try {
        await initDatabase();
        await loadFromIndexedDB();
        if (mounted) {
          setIsReady(true);
        }
      } catch (err) {
        if (mounted) {
          setError(err instanceof Error ? err : new Error('Failed to initialize database'));
        }
      }
    }

    init();

    return () => {
      mounted = false;
    };
  }, []);

  return { isReady, error };
}
