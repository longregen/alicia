import { useState, useEffect, useCallback, useRef } from 'react';
import { getDatabase } from '../db/sqlite';
import type { SqlValue } from 'sql.js';

export function useLiveQuery<T>(
  query: string | null,
  params: SqlValue[] = [],
  deps: unknown[] = []
): { data: T[]; refetch: () => void; error: Error | null } {
  const [data, setData] = useState<T[]>([]);
  const [error, setError] = useState<Error | null>(null);

  // Use refs to capture latest values without triggering refetch on every change
  const paramsRef = useRef(params);
  paramsRef.current = params;

  const refetch = useCallback(() => {
    if (!query) {
      setData([]);
      return;
    }

    try {
      const db = getDatabase();
      const results = db.exec(query, paramsRef.current);

      if (results.length === 0) {
        setData([]);
        return;
      }

      // Convert rows to objects based on column names
      const columns = results[0].columns;
      const rows = results[0].values;

      const objects = rows.map((row: unknown[]) => {
        const obj: Record<string, unknown> = {};
        columns.forEach((col: string, idx: number) => {
          obj[col] = row[idx];
        });
        return obj as T;
      });

      setData(objects);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Query failed'));
      setData([]);
    }
  }, [query]);

  // Trigger refetch when query changes or when caller-provided deps change
  useEffect(() => {
    refetch();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refetch, ...deps]);

  return { data, refetch, error };
}
