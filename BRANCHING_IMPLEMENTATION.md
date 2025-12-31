# Message Branching Implementation

This document describes the message branching feature implementation using the conversation tip approach.

## Overview

Message branching allows multiple conversation paths to exist from the same parent message. The system uses a `tip_message_id` column in the `alicia_conversations` table to track the current "head" of the message chain.

## Database Changes

### Migration: 002_add_tip_message

**File**: `migrations/002_add_tip_message.up.sql`

- Adds `tip_message_id` column to `alicia_conversations` table
- References `alicia_messages(id)` for foreign key integrity
- Backfills existing conversations with their latest message as the tip
- Adds index for performance

**File**: `migrations/002_add_tip_message.down.sql`

- Provides rollback by removing the column and index

## Domain Model Changes

### Conversation Model

**File**: `internal/domain/models/conversation.go`

- Added `TipMessageID *string` field to track the current message chain head

## Repository Changes

### ConversationRepository

**File**: `internal/adapters/postgres/conversation_repository.go`

- Updated all SELECT queries to include `tip_message_id`
- Updated INSERT/UPDATE queries to handle `tip_message_id`
- Added `UpdateTip(ctx, conversationID, messageID)` method
- Updated scan functions to handle the new nullable string pointer field

### MessageRepository

**File**: `internal/adapters/postgres/message_repository.go`

Added two new methods:

1. **GetChainFromTip(ctx, tipMessageID)**:
   - Uses recursive CTE to walk backwards through `previous_id` links
   - Returns messages in chronological order (oldest to newest)
   - Efficiently retrieves the entire message chain for the current branch

2. **GetSiblings(ctx, messageID)**:
   - Returns all messages that share the same `previous_id`
   - Useful for displaying alternate branches from a given point
   - Returns empty slice if message has no siblings

### Utility Functions

**File**: `internal/adapters/postgres/utils.go`

Added helpers for nullable string pointers:
- `nullStringPtr(*string)` - converts Go pointer to sql.NullString
- `getStringPtr(sql.NullString)` - extracts pointer from sql.NullString

## Application Service Changes

### MessageService

**File**: `internal/application/services/message.go`

- Updated `Create()` to set `previous_id` from conversation's `tip_message_id`
- Updates conversation tip after successfully creating a message
- Updated `GetByConversation()` to use `GetChainFromTip` when tip exists

## HTTP Handler Changes

### MessagesHandler

**File**: `internal/adapters/http/handlers/messages.go`

Updated existing endpoints:
- **List messages**: Uses `GetChainFromTip` when tip exists, falls back to `GetByConversation` for backwards compatibility
- **Send message**: Sets `previous_id` to current tip, updates tip after creation

Added new endpoints:
- **GET /api/v1/messages/{id}/siblings**: Returns all sibling messages (branches from same parent)
- **POST /api/v1/conversations/{id}/switch-branch**: Updates the conversation tip to a different message

### DTO Changes

**File**: `internal/adapters/http/dto/message.go`

- Added `SwitchBranchRequest` struct with `TipMessageID` field

### Server Routes

**File**: `internal/adapters/http/server.go`

Added new routes:
```go
r.Get("/messages/{id}/siblings", messagesHandler.GetSiblings)
r.Post("/conversations/{id}/switch-branch", messagesHandler.SwitchBranch)
```

## Message Chain Structure

```
Message chain (followed via previous_id):
am_001 ← am_002 ← am_003 ← am_004 (tip)
              └── am_005 ← am_006 (branch, could become new tip)
```

### How Branching Works

1. **Normal flow**: Each new message sets `previous_id` to current `tip_message_id`
2. **Branching**: Client can create a message with `previous_id` set to any existing message
3. **Switching branches**: Client calls `/switch-branch` to update `tip_message_id`
4. **Viewing messages**: System walks backwards from `tip_message_id` to display current branch
5. **Finding siblings**: Use `/messages/{id}/siblings` to see alternate branches

## API Examples

### Get Messages (Current Branch)
```bash
GET /api/v1/conversations/{id}/messages
```
Returns messages from the current tip backwards through the chain.

### Get Sibling Messages
```bash
GET /api/v1/messages/{message_id}/siblings
```
Returns all messages that branch from the same parent as the specified message.

### Switch to Different Branch
```bash
POST /api/v1/conversations/{id}/switch-branch
Content-Type: application/json

{
  "tip_message_id": "am_006"
}
```
Updates the conversation to show a different branch.

## Backwards Compatibility

The implementation maintains backwards compatibility:
- If `tip_message_id` is NULL (old conversations), falls back to `GetByConversation`
- Migration backfills existing conversations with their latest message
- Existing message fetching continues to work

## Future Enhancements

Possible improvements for the future:
1. Add branch metadata (names, timestamps, creators)
2. Track branch genealogy for visualization
3. Add branch merging capabilities
4. Implement branch comparison tools
5. Add branch-aware search functionality
