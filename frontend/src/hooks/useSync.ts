import { useEffect, useState, useRef, useCallback } from 'react';
import { api } from '../services/api';
import { useMessageContext } from '../contexts/MessageContext';
import { SyncState, SyncRequest, SyncMessageRequest } from '../types/sync';
import { Message } from '../types/models';
import { useSSE } from './useSSE';

const BASE_SYNC_INTERVAL_MS = 5000; // Base interval: 5 seconds
const MAX_SYNC_INTERVAL_MS = 60000; // Max interval when idle: 60 seconds
const MAX_RETRY_DELAY_MS = 30000; // Max 30 seconds backoff
const INITIAL_RETRY_DELAY_MS = 1000; // Start with 1 second
const IDLE_THRESHOLD_MS = 30000; // Consider idle after 30 seconds of no new messages

// When SSE is connected, reduce polling frequency significantly
const SSE_CONNECTED_SYNC_INTERVAL_MS = 30000; // 30 seconds when SSE is active

interface UseSyncResult extends SyncState {
  syncNow: () => Promise<void>;
  isSSEConnected: boolean;
}

export function useSync(conversationId: string | null): UseSyncResult {
  const { messages, mergeMessages, updateMessage } = useMessageContext();
  const [isSyncing, setIsSyncing] = useState(false);
  const [lastSyncTime, setLastSyncTime] = useState<Date | null>(null);
  const [syncError, setSyncError] = useState<string | null>(null);
  const retryDelayRef = useRef(INITIAL_RETRY_DELAY_MS);
  const lastMessageCountRef = useRef(0);
  const lastActivityTimeRef = useRef(Date.now());

  const syncIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const isMountedRef = useRef(true);
  const isSyncingRef = useRef(false);
  const conversationIdRef = useRef<string | null>(null);

  // SSE integration for real-time updates
  const { isConnected: isSSEConnected, error: sseError } = useSSE(conversationId, {
    onMessage: (message: Message) => {
      // Real-time message received via SSE
      console.log('SSE: Received message', message.id);
      mergeMessages([message]);
      lastActivityTimeRef.current = Date.now();
    },
    onSync: () => {
      // Sync event received, trigger a sync
      console.log('SSE: Sync event received');
      performSync();
    },
    enabled: !!conversationId,
  });

  // Update refs when values change
  useEffect(() => {
    conversationIdRef.current = conversationId;
  }, [conversationId]);

  // Track message activity for adaptive polling
  useEffect(() => {
    if (messages.length !== lastMessageCountRef.current) {
      lastActivityTimeRef.current = Date.now();
      lastMessageCountRef.current = messages.length;
    }
  }, [messages.length]);

  // Calculate sync interval with exponential backoff when idle
  // If SSE is connected, use much longer intervals since we get real-time updates
  const getSyncInterval = useCallback(() => {
    // If SSE is connected, use longer polling interval (safety net only)
    if (isSSEConnected) {
      return SSE_CONNECTED_SYNC_INTERVAL_MS;
    }

    // Fallback to adaptive polling when SSE is not connected
    const timeSinceActivity = Date.now() - lastActivityTimeRef.current;

    if (timeSinceActivity < IDLE_THRESHOLD_MS) {
      // Active: use base interval
      return BASE_SYNC_INTERVAL_MS;
    } else {
      // Idle: use exponential backoff (2x for every 30s idle, up to max)
      const idlePeriods = Math.floor(timeSinceActivity / IDLE_THRESHOLD_MS);
      const backoffInterval = BASE_SYNC_INTERVAL_MS * Math.pow(2, idlePeriods);
      return Math.min(backoffInterval, MAX_SYNC_INTERVAL_MS);
    }
  }, [isSSEConnected]);

  // Convert local messages to SyncMessageRequest format
  const buildSyncRequest = useCallback((localMessages: Message[]): SyncRequest => {
    const syncMessages: SyncMessageRequest[] = localMessages.map(msg => ({
      local_id: msg.local_id || msg.id,
      sequence_number: msg.sequence_number,
      previous_id: msg.previous_id,
      role: msg.role,
      contents: msg.contents,
      created_at: msg.created_at,
      updated_at: msg.updated_at,
    }));

    return { messages: syncMessages };
  }, []);

  // Sync function using proper sync protocol
  const performSync = useCallback(async () => {
    const currentConversationId = conversationIdRef.current;

    if (!currentConversationId || isSyncingRef.current) {
      return;
    }

    try {
      isSyncingRef.current = true;
      setIsSyncing(true);
      setSyncError(null);

      // Build sync request with local message state
      const syncRequest = buildSyncRequest(messages);

      // Use sync endpoint instead of getMessages
      const syncResponse = await api.syncConversation(currentConversationId, syncRequest);

      // Only process if we're still mounted and conversation hasn't changed
      if (isMountedRef.current && conversationIdRef.current === currentConversationId) {
        // Handle sync response with conflict resolution
        const serverMessages: Message[] = [];

        for (const syncedMsg of syncResponse.synced_messages) {
          if (syncedMsg.status === 'synced' && syncedMsg.message) {
            // Message was synced successfully
            serverMessages.push(syncedMsg.message);

            // Update local message if it had a temporary local_id
            if (syncedMsg.local_id !== syncedMsg.server_id && syncedMsg.message) {
              const localMsg = messages.find(m => (m.local_id || m.id) === syncedMsg.local_id);
              if (localMsg && localMsg.local_id) {
                // Update to use server ID and content
                updateMessage(localMsg.id, { contents: syncedMsg.message.contents });
              }
            }
          } else if (syncedMsg.status === 'conflict' && syncedMsg.conflict) {
            // Conflict detected - use server version as authoritative
            if (syncedMsg.conflict.server_message) {
              serverMessages.push(syncedMsg.conflict.server_message);
              console.warn(`Sync conflict for message ${syncedMsg.local_id}: ${syncedMsg.conflict.reason}`);
            }
          }
        }

        // Merge server messages into local state
        if (serverMessages.length > 0) {
          mergeMessages(serverMessages);
        }

        setLastSyncTime(new Date());
        retryDelayRef.current = INITIAL_RETRY_DELAY_MS; // Reset backoff on success
      }
    } catch (error) {
      if (isMountedRef.current) {
        const errorMessage = error instanceof Error ? error.message : 'Sync failed';
        setSyncError(errorMessage);

        // Exponential backoff on error
        retryDelayRef.current = Math.min(retryDelayRef.current * 2, MAX_RETRY_DELAY_MS);
      }
    } finally {
      isSyncingRef.current = false;
      if (isMountedRef.current) {
        setIsSyncing(false);
      }
    }
  }, [messages, buildSyncRequest, mergeMessages, updateMessage]);

  // Expose syncNow as a stable callback
  const syncNow = useCallback(async () => {
    await performSync();
  }, [performSync]);

  // Set up adaptive polling interval with exponential backoff when idle
  useEffect(() => {
    // Clear any existing interval
    if (syncIntervalRef.current) {
      clearInterval(syncIntervalRef.current);
      syncIntervalRef.current = null;
    }

    // Don't poll if no conversation is active
    if (!conversationId) {
      setLastSyncTime(null);
      setSyncError(null);
      return;
    }

    // Initial sync when conversation changes
    performSync();

    // Set up adaptive polling interval
    const scheduleNextSync = () => {
      const interval = getSyncInterval();
      syncIntervalRef.current = setTimeout(() => {
        performSync().finally(() => {
          // Schedule next sync after current one completes
          if (conversationIdRef.current) {
            scheduleNextSync();
          }
        });
      }, interval);
    };

    scheduleNextSync();

    // Cleanup on unmount or conversation change
    return () => {
      if (syncIntervalRef.current) {
        clearTimeout(syncIntervalRef.current);
        syncIntervalRef.current = null;
      }
    };
  }, [conversationId, performSync, getSyncInterval]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  return {
    isSyncing,
    lastSyncTime,
    syncError: syncError || sseError?.message || null,
    syncNow,
    isSSEConnected,
  };
}
