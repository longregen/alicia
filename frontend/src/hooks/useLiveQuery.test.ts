import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { useLiveQuery } from './useLiveQuery';
import * as sqlite from '../db/sqlite';

vi.mock('../db/sqlite', () => ({
  getDatabase: vi.fn(),
}));

describe('useLiveQuery', () => {
  let mockDb: {
    exec: ReturnType<typeof vi.fn>;
  };

  beforeEach(() => {
    vi.clearAllMocks();

    mockDb = {
      exec: vi.fn(() => []),
    };

    (sqlite.getDatabase as any).mockReturnValue(mockDb);
  });

  it('should execute query and return data', () => {
    mockDb.exec.mockReturnValue([
      {
        columns: ['id', 'name', 'count'],
        values: [
          ['1', 'Alice', 10],
          ['2', 'Bob', 20],
        ],
      },
    ]);

    const { result } = renderHook(() =>
      useLiveQuery<{ id: string; name: string; count: number }>(
        'SELECT id, name, count FROM users',
        []
      )
    );

    expect(result.current.data).toHaveLength(2);
    expect(result.current.data[0]).toEqual({ id: '1', name: 'Alice', count: 10 });
    expect(result.current.data[1]).toEqual({ id: '2', name: 'Bob', count: 20 });
    expect(result.current.error).toBeNull();
  });

  it('should return empty array when no results', () => {
    mockDb.exec.mockReturnValue([]);

    const { result } = renderHook(() =>
      useLiveQuery('SELECT * FROM users WHERE id = ?', ['999'])
    );

    expect(result.current.data).toEqual([]);
    expect(result.current.error).toBeNull();
  });

  it('should handle query parameters', () => {
    mockDb.exec.mockReturnValue([
      {
        columns: ['id', 'name'],
        values: [['1', 'Alice']],
      },
    ]);

    renderHook(() =>
      useLiveQuery('SELECT id, name FROM users WHERE id = ?', ['1'])
    );

    expect(mockDb.exec).toHaveBeenCalledWith(
      'SELECT id, name FROM users WHERE id = ?',
      ['1']
    );
  });

  it('should handle null query', () => {
    const { result } = renderHook(() => useLiveQuery(null));

    expect(result.current.data).toEqual([]);
    expect(mockDb.exec).not.toHaveBeenCalled();
  });

  it('should handle query errors', () => {
    mockDb.exec.mockImplementation(() => {
      throw new Error('Query failed');
    });

    const { result } = renderHook(() =>
      useLiveQuery('SELECT * FROM non_existent_table')
    );

    expect(result.current.data).toEqual([]);
    expect(result.current.error).toEqual(new Error('Query failed'));
  });

  it('should provide refetch function', () => {
    mockDb.exec.mockReturnValue([
      {
        columns: ['id', 'name'],
        values: [['1', 'Alice']],
      },
    ]);

    const { result } = renderHook(() =>
      useLiveQuery('SELECT id, name FROM users')
    );

    expect(result.current.data).toHaveLength(1);

    // Change the mock data
    mockDb.exec.mockReturnValue([
      {
        columns: ['id', 'name'],
        values: [
          ['1', 'Alice'],
          ['2', 'Bob'],
        ],
      },
    ]);

    // Refetch
    act(() => {
      result.current.refetch();
    });

    expect(result.current.data).toHaveLength(2);
  });

  it('should refetch when dependencies change', async () => {
    let filter = 'active';

    mockDb.exec.mockReturnValue([
      {
        columns: ['id', 'status'],
        values: [['1', 'active']],
      },
    ]);

    const { result, rerender } = renderHook(() =>
      useLiveQuery('SELECT id, status FROM users WHERE status = ?', [filter], [filter])
    );

    expect(result.current.data).toHaveLength(1);
    expect(result.current.data[0].status).toBe('active');

    // Change filter
    filter = 'inactive';
    mockDb.exec.mockReturnValue([
      {
        columns: ['id', 'status'],
        values: [['2', 'inactive']],
      },
    ]);

    rerender();

    await waitFor(() => {
      expect(result.current.data).toHaveLength(1);
      expect(result.current.data[0].status).toBe('inactive');
    });
  });

  it('should convert rows to objects with correct column names', () => {
    mockDb.exec.mockReturnValue([
      {
        columns: ['user_id', 'full_name', 'email_address'],
        values: [['123', 'John Doe', 'john@example.com']],
      },
    ]);

    const { result } = renderHook(() =>
      useLiveQuery<{ user_id: string; full_name: string; email_address: string }>(
        'SELECT user_id, full_name, email_address FROM users'
      )
    );

    expect(result.current.data[0]).toEqual({
      user_id: '123',
      full_name: 'John Doe',
      email_address: 'john@example.com',
    });
  });

  it('should handle multiple rows correctly', () => {
    mockDb.exec.mockReturnValue([
      {
        columns: ['id', 'value'],
        values: [
          ['1', 100],
          ['2', 200],
          ['3', 300],
          ['4', 400],
          ['5', 500],
        ],
      },
    ]);

    const { result } = renderHook(() =>
      useLiveQuery<{ id: string; value: number }>('SELECT id, value FROM data')
    );

    expect(result.current.data).toHaveLength(5);
    expect(result.current.data[0].value).toBe(100);
    expect(result.current.data[4].value).toBe(500);
  });

  it('should clear error on successful refetch', () => {
    mockDb.exec.mockImplementation(() => {
      throw new Error('Initial error');
    });

    const { result } = renderHook(() =>
      useLiveQuery('SELECT * FROM users')
    );

    expect(result.current.error).not.toBeNull();

    // Fix the error
    mockDb.exec.mockReturnValue([
      {
        columns: ['id'],
        values: [['1']],
      },
    ]);

    act(() => {
      result.current.refetch();
    });

    expect(result.current.error).toBeNull();
    expect(result.current.data).toHaveLength(1);
  });

  it('should handle complex nested data types', () => {
    mockDb.exec.mockReturnValue([
      {
        columns: ['id', 'json_data', 'number', 'boolean'],
        values: [
          ['1', '{"nested": "value"}', 42, 1],
          ['2', '{"other": "data"}', 100, 0],
        ],
      },
    ]);

    const { result } = renderHook(() =>
      useLiveQuery<{ id: string; json_data: string; number: number; boolean: number }>(
        'SELECT id, json_data, number, boolean FROM complex_table'
      )
    );

    expect(result.current.data[0].json_data).toBe('{"nested": "value"}');
    expect(result.current.data[0].number).toBe(42);
    expect(result.current.data[0].boolean).toBe(1);
  });

  it('should update when query changes', async () => {
    mockDb.exec.mockReturnValueOnce([
      {
        columns: ['id', 'name'],
        values: [['1', 'Alice']],
      },
    ]);

    const { result, rerender } = renderHook(
      ({ query }) => useLiveQuery(query),
      { initialProps: { query: 'SELECT id, name FROM users WHERE id = 1' } }
    );

    expect(result.current.data).toHaveLength(1);

    // Change query
    mockDb.exec.mockReturnValueOnce([
      {
        columns: ['id', 'name'],
        values: [
          ['1', 'Alice'],
          ['2', 'Bob'],
        ],
      },
    ]);

    rerender({ query: 'SELECT id, name FROM users' });

    await waitFor(() => {
      expect(result.current.data).toHaveLength(2);
    });
  });

  it('should not execute query when query is null', () => {
    const { result } = renderHook(() => useLiveQuery(null, [], []));

    expect(result.current.data).toEqual([]);
    expect(mockDb.exec).not.toHaveBeenCalled();
    expect(result.current.error).toBeNull();
  });
});
