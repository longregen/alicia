import { useState, useCallback } from 'react';
import { useWebSocketSync } from './useWebSocketSync';
import { SyncState } from '../types/sync';

interface UseSyncOptions {
  onSync?: () => void;
}

interface UseSyncResult extends SyncState {
  syncNow: () => void;
  isSSEConnected: boolean;
}

export function useSync(conversationId: string | null, options?: UseSyncOptions): UseSyncResult {
  const [lastSyncTime, setLastSyncTime] = useState<Date | null>(null);

  const handleSync = useCallback(() => {
    setLastSyncTime(new Date());
    options?.onSync?.();
  }, [options]);

  const { isConnected, error, syncNow } = useWebSocketSync(conversationId, {
    onSync: handleSync,
    enabled: !!conversationId,
  });

  return {
    isSyncing: false, // WebSocket is always syncing when connected
    lastSyncTime,
    syncError: error?.message || null,
    syncNow,
    isSSEConnected: isConnected, // For compatibility
  };
}
