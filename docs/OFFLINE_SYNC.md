# Offline Mode and Synchronization

This document describes the offline mode capabilities and synchronization protocol for the Alicia voice assistant.

## Overview

Alicia supports offline mode, allowing clients to create and store messages locally when disconnected from the server. When the connection is restored, the client can sync these messages to the server using the sync API.

The offline sync system operates independently from the real-time reconnection semantics described in `protocol/05-reconnection-semantics.md`. The reconnection protocol handles brief connection drops during active conversations, while offline sync handles longer periods of disconnection where messages are created and stored entirely offline.

## Architecture

### Message Sync Tracking

Each message in the system includes sync tracking fields to support offline operation:

- **LocalID**: Client-generated unique identifier created when the message is first saved offline
- **ServerID**: Server-assigned canonical identifier (assigned during sync)
- **SyncStatus**: Current synchronization state (pending, synced, or conflict)
- **SyncedAt**: Timestamp when the message was last synced with the server

### Sync Status States

```
pending  → Message exists locally but hasn't been synced to server
synced   → Message is synchronized with the server
conflict → A conflict occurs when local and server versions differ
```

## Client Workflow

### 1. Creating Messages Offline

When the client is offline or chooses to create messages locally:

```javascript
// Client generates a local ID using nanoid or similar
const localID = generateLocalID(); // e.g., "local_abc123"

const message = {
  local_id: localID,
  conversation_id: "conv_xyz",
  sequence_number: 42,
  role: "user",
  contents: "What's the weather like?",
  created_at: new Date().toISOString(),
  sync_status: "pending"
};

// Store in local database (IndexedDB, SQLite, etc.)
await localDB.messages.add(message);
```

### 2. Syncing When Online

When connectivity is restored, the client retrieves pending messages and syncs them:

```javascript
// Get all pending messages for the conversation
const pendingMessages = await localDB.messages
  .where('sync_status').equals('pending')
  .toArray();

// Sync to server
const response = await fetch(`/api/v1/conversations/${conversationID}/sync`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    messages: pendingMessages.map(msg => ({
      local_id: msg.local_id,
      sequence_number: msg.sequence_number,
      previous_id: msg.previous_id,
      role: msg.role,
      contents: msg.contents,
      created_at: msg.created_at,
      updated_at: msg.updated_at
    }))
  })
});

const syncResult = await response.json();

// Process sync results
for (const syncedMsg of syncResult.synced_messages) {
  if (syncedMsg.status === 'synced') {
    // Update local message with server ID
    await localDB.messages.update(syncedMsg.local_id, {
      server_id: syncedMsg.server_id,
      sync_status: 'synced',
      synced_at: syncResult.synced_at
    });
  } else if (syncedMsg.status === 'conflict') {
    // Handle conflict (see conflict resolution below)
    await handleConflict(syncedMsg);
  }
}
```

## Server API

### POST /api/v1/conversations/{id}/sync

Synchronizes client messages with the server.

**Content Types:** Both `application/json` and `application/msgpack` are supported. All field names use camelCase in the wire format.

**Request Body:**
```json
{
  "messages": [
    {
      "local_id": "local_abc123",
      "sequence_number": 42,
      "previous_id": "am_previous",
      "role": "user",
      "contents": "What's the weather like?",
      "created_at": "2025-12-21T10:30:00Z",
      "updated_at": "2025-12-21T10:30:00Z"
    }
  ]
}
```

**Response:**
```json
{
  "synced_messages": [
    {
      "local_id": "local_abc123",
      "server_id": "am_xyz789",
      "status": "synced",
      "message": {
        "id": "am_xyz789",
        "conversation_id": "conv_xyz",
        "sequence_number": 42,
        "role": "user",
        "contents": "What's the weather like?",
        "local_id": "local_abc123",
        "server_id": "am_xyz789",
        "sync_status": "synced",
        "synced_at": "2025-12-21T10:35:00Z",
        "created_at": "2025-12-21T10:30:00Z",
        "updated_at": "2025-12-21T10:30:00Z"
      }
    }
  ],
  "synced_at": "2025-12-21T10:35:00Z"
}
```

### GET /api/v1/conversations/{id}/sync/status

Gets synchronization status for a conversation.

**Content Types:** Both `application/json` and `application/msgpack` are supported. All field names use camelCase in the wire format.

**Response:**
```json
{
  "conversation_id": "conv_xyz",
  "pending_count": 0,
  "synced_count": 42,
  "conflict_count": 1,
  "last_synced_at": "2025-12-21T10:35:00Z"
}
```

### GET /api/v1/conversations/{id}/sync/ws

Real-time WebSocket synchronization endpoint for bidirectional message sync.

**Protocol:** Binary MessagePack over WebSocket

**Features:**
- Automatic sync of pending messages on connection establishment
- Real-time broadcast of new messages to all connected clients
- Ping/pong heartbeat for connection health monitoring
- Cross-device message synchronization via `WebSocketBroadcaster`

**Connection Flow:**
1. Client establishes WebSocket connection
2. Server automatically syncs all pending messages for the conversation
3. Server broadcasts new messages to all connected WebSocket clients
4. Client receives real-time updates from other devices/sessions

**Reconnection Behavior:**
- Exponential backoff reconnection (minimum 1 second, maximum 30 seconds)
- Automatic pending message resync on reconnect
- No message loss during brief disconnections

**Message Format:** All messages use MessagePack binary encoding with camelCase field names. See [WebSocket Protocol Specification](protocol/index.md) for detailed message type definitions.

## Conflict Resolution

### Conflict Detection

Conflicts occur when:
1. A message with the same `local_id` already exists on the server with different content
2. Sequence number collisions occur
3. Invalid message dependencies (e.g., `previous_id` points to non-existent message)

### Manual Resolution

The system uses a **manual resolution** strategy:

1. When a conflict is detected, the server marks the message with `sync_status = 'conflict'`
2. The conflict details are returned to the client:
   ```json
   {
     "local_id": "local_abc123",
     "status": "conflict",
     "conflict": {
       "reason": "Content mismatch with existing message",
       "server_message": { /* existing server message */ },
       "resolution": "manual"
     }
   }
   ```
3. The client is responsible for presenting the conflict to the user and resolving it
4. The user can choose to:
   - Keep the local version (update server)
   - Keep the server version (discard local version)
   - Merge both versions manually

## Database Schema

The sync tracking is implemented with these additional columns in `alicia_messages`:

```sql
ALTER TABLE alicia_messages
    ADD COLUMN local_id TEXT,
    ADD COLUMN server_id TEXT,
    ADD COLUMN sync_status sync_status NOT NULL DEFAULT 'synced',
    ADD COLUMN synced_at TIMESTAMP;

CREATE INDEX idx_messages_local_id ON alicia_messages(local_id)
    WHERE local_id IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX idx_messages_sync_status ON alicia_messages(conversation_id, sync_status)
    WHERE deleted_at IS NULL;
```

## Implementation Notes

### Server-Side

1. **Message Creation**: The sync handler creates messages with both `local_id` and `server_id`
2. **Idempotency**: Syncing the same message multiple times is safe - if a message with the same `local_id` exists, it's treated as a duplicate
3. **Validation**: The server validates message content, role, and relationships
4. **Atomic Operations**: Each message sync is independent - failures don't roll back successful syncs

### Client-Side

1. **Local Storage**: Clients should use a robust offline storage mechanism (IndexedDB, SQLite, etc.)
2. **ID Generation**: Use a reliable ID generation strategy (nanoid, uuid) with a prefix (e.g., "local_")
3. **Sync Timing**: Sync can happen:
   - Automatically when connection is restored
   - Periodically in the background
   - On-demand when user triggers it
4. **Conflict UI**: Provide clear UI for conflict resolution
5. **Retry Logic**: Implement exponential backoff for failed sync attempts

## Differences from Real-Time Reconnection

| Feature | Offline Sync | Real-Time Reconnection |
|---------|--------------|------------------------|
| Use Case | Long periods offline | Brief connection drops |
| Message Storage | Client-side database | Server-side only |
| Sync Direction | Client → Server (bidirectional via WebSocket) | Server → Client |
| Conflict Potential | High | Low |
| Protocol | HTTP REST + WebSocket (MessagePack) | LiveKit data channel |
| ID Management | Local + Server IDs | Server IDs only |

## See Also

- [Database Schema](DATABASE.md) - Sync-related tables
- [WebSocket Protocol Specification](protocol/index.md) - Detailed protocol message types and envelope format
- [Reconnection Semantics](protocol/05-reconnection-semantics.md) - Real-time reconnection during active conversations

