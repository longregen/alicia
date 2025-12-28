import { useState, useCallback, useRef, useEffect } from 'react';
import { useWebSocketSync } from './useWebSocketSync';
import { SyncState } from '../types/sync';
import { Message } from '../types/models';

interface UseSyncOptions {
  onMessage?: (message: Message) => void;
  onSync?: () => void;
}

interface UseSyncResult extends SyncState {
  syncNow: () => void;
  isSSEConnected: boolean;
}

export function useSync(conversationId: string | null, options?: UseSyncOptions): UseSyncResult {
  const [lastSyncTime, setLastSyncTime] = useState<Date | null>(null);

  // Store callbacks in refs to maintain stable references
  const onSyncRef = useRef(options?.onSync);
  const onMessageRef = useRef(options?.onMessage);

  useEffect(() => {
    onSyncRef.current = options?.onSync;
  }, [options?.onSync]);

  useEffect(() => {
    onMessageRef.current = options?.onMessage;
  }, [options?.onMessage]);

  const handleSync = useCallback(() => {
    setLastSyncTime(new Date());
    onSyncRef.current?.();
  }, []);

  const handleMessage = useCallback((message: Message) => {
    onMessageRef.current?.(message);
  }, []);

  const { isConnected, error, syncNow } = useWebSocketSync(conversationId, {
    onSync: handleSync,
    onMessage: handleMessage,
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
