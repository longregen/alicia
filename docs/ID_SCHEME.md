# Alicia ID Prefix Scheme

This document describes the ID prefix system used throughout the Alicia codebase for entity identification.

## Overview

Alicia uses a consistent ID format across all entities to improve type safety, debugging, and code clarity. Each ID consists of a **prefix** that identifies the entity type, followed by an underscore and a **21-character NanoID**.

### ID Format

```
<prefix>_<nanoid>

Examples:
ac_k9mX2pL7qR3vN5pQ8tYzW
am_7gH4mN9pL2kR6vT5xB8wZ
ams_3nQ5kL8pR7mV2tY9xC6wH
```

### NanoID Properties

- **Length**: 21 characters
- **Character Set**: URL-safe alphabet (A-Za-z0-9_-)
- **Collision Probability**: ~1% chance in 47,000 years at 1000 IDs/hour ([NanoID calculator](https://zelark.github.io/nano-id-cc/))
- **Library**: `github.com/matoous/go-nanoid/v2`

## ID Prefixes

### Core Entities

| Prefix | Entity | Database Table | Description |
|--------|--------|----------------|-------------|
| `ac_` | Conversation | `alicia_conversations` | A conversation thread between user and assistant |
| `am_` | Message | `alicia_messages` | A single message (user or assistant) within a conversation |
| `ams_` | Sentence | `alicia_sentences` | A sentence fragment within an assistant message (for streaming) |
| `aa_` | Audio | `alicia_audio` | Audio recording metadata for voice input/output |
| `amem_` | Memory | `alicia_memory` | A stored memory entry with vector embeddings |
| `amu_` | Memory Usage | `alicia_memory_used` | A record of when a memory was retrieved and used |
| `at_` | Tool | `alicia_tools` | A tool definition (e.g., web_search, calculator) |
| `atu_` | Tool Use | `alicia_tool_uses` | A specific invocation of a tool within a message |
| `ar_` | Reasoning Step | `alicia_reasoning_steps` | A chain-of-thought reasoning step (for extended thinking) |
| `aucc_` | User Commentary | `alicia_user_conversation_commentaries` | User feedback or annotations on conversations |
| `amt_` | Meta | `alicia_meta` | Generic key-value metadata storage |

### Integration & Infrastructure

| Prefix | Entity | Database Table | Description |
|--------|--------|----------------|-------------|
| `amcp_` | MCP Server | `alicia_mcp_servers` | Model Context Protocol server configuration |
| `room_` | LiveKit Room | - | LiveKit room name for real-time audio/video (Note: `GenerateLiveKitRoomName()` exists in `Generator` but is NOT part of the `IDGenerator` interface) |

### Feedback & Analytics

| Prefix | Entity | Database Table | Description |
|--------|--------|----------------|-------------|
| `av_` | Vote | `alicia_votes` | User vote (up/down/critical) on messages, tools, memories |
| `an_` | Note | `alicia_notes` | User annotation or note on a message |
| `ass_` | Session Stats | `alicia_session_stats` | Aggregated statistics for a conversation session |

### Prompt Optimization (GEPA)

| Prefix | Entity | Database Table | Description |
|--------|--------|----------------|-------------|
| `aor_` | Optimization Run | `prompt_optimization_runs` | A prompt optimization session |
| `apc_` | Prompt Candidate | `prompt_candidates` | A candidate prompt in the optimization process |
| `ape_` | Prompt Evaluation | `prompt_evaluations` | Evaluation result for a prompt candidate |
| `gte_` | Training Example | `gepa_training_examples` | Training example for GEPA derived from user votes |
| `spv_` | System Prompt Version | `system_prompt_versions` | Version of a system prompt for A/B testing |

## Why Use ID Prefixes?

### 1. Type Safety

Prefixes provide visual type checking and reduce the chance of passing wrong IDs:

```go
// Clear what type of ID is expected
func GetMessage(messageID string) (*Message, error) {
    if !strings.HasPrefix(messageID, "am_") {
        return nil, errors.New("invalid message ID")
    }
    // ...
}
```

### 2. Debugging & Logging

Prefixes make logs immediately readable:

```
❌ Without prefixes:
ERROR: Failed to load k9mX2pL7qR3vN5pQ8tYzW for 7gH4mN9pL2kR6vT5xB8wZ

✅ With prefixes:
ERROR: Failed to load conversation ac_k9mX2pL7qR3vN5pQ8tYzW for message am_7gH4mN9pL2kR6vT5xB8wZ
```

### 3. Database Integrity

Prefixes help catch cross-table reference errors:

```sql
-- Immediately visible that this is wrong
SELECT * FROM alicia_messages WHERE conversation_id = 'am_7gH4mN9pL2kR6vT5xB8wZ';
                                                    -- ^ Should be ac_, not am_
```

### 4. API Clarity

REST API endpoints become self-documenting:

```
GET /conversations/ac_k9mX2pL7qR3vN5pQ8tYzW
GET /conversations/ac_k9mX2pL7qR3vN5pQ8tYzW/messages/am_7gH4mN9pL2kR6vT5xB8wZ
```

## Implementation

### Go Code Generation

IDs are generated using the `IDGenerator` interface defined in `internal/ports/repositories.go`:

```go
type IDGenerator interface {
    GenerateConversationID() string  // Returns "ac_<nanoid>"
    GenerateMessageID() string       // Returns "am_<nanoid>"
    GenerateSentenceID() string      // Returns "ams_<nanoid>"
    // ... etc
}
```

The concrete implementation is in `internal/adapters/id/generator.go`:

```go
func (g *Generator) generate(prefix string) string {
    id, err := gonanoid.New(21)
    if err != nil {
        return prefix + "_fallback"  // Fallback on error (rare)
    }
    return prefix + "_" + id
}

func (g *Generator) GenerateConversationID() string {
    return g.generate("ac")
}
```

### Database Default Values

PostgreSQL generates IDs automatically if not provided:

```sql
CREATE TABLE alicia_conversations (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('ac'),
    -- ...
);
```

The `generate_random_id()` function is defined in the migration:

```sql
CREATE OR REPLACE FUNCTION generate_random_id(prefix TEXT) RETURNS TEXT AS $$
BEGIN
    RETURN prefix || '_' || encode(gen_random_bytes(12), 'base64');
END;
$$ LANGUAGE plpgsql;
```

**Note**: The database-generated IDs use base64-encoded random bytes (16 chars), while Go-generated IDs use NanoID (21 chars). Both are unique, but Go-generated IDs are preferred for consistency.

## Examples

### Conversation Flow

```
Conversation: ac_k9mX2pL7qR3vN5pQ8tYzW
├─ Message 1 (user): am_7gH4mN9pL2kR6vT5xB8wZ
│  └─ Audio: aa_2nP5kL8pR7mV2tY9xC6wH
│
├─ Message 2 (assistant): am_9vT2kL5pR8mN4xB7wZ3qH
   ├─ Reasoning Step 1: ar_8mQ5nL7pR3kV2tY9xB6wH
   ├─ Reasoning Step 2: ar_6wT3kL9pR5mN2xB8vZ4qH
   ├─ Tool Use: atu_4xQ7kL2pR9mV5tY3wB8nH
   ├─ Sentence 1: ams_3nQ5kL8pR7mV2tY9xC6wH
   ├─ Sentence 2: ams_5vT8kL3pR2mN7xB4wZ9qH
   └─ Sentence 3: ams_7wQ4kL6pR9mV3tY8xB2nH
```

### Memory Retrieval

```
Query message: am_9vT2kL5pR8mN4xB7wZ3qH

Retrieved memories:
├─ Memory: amem_8nQ5kL7pR2mV9tY3xB6wH (similarity: 0.92)
│  └─ Usage record: amu_4xT7kL9pR3mN5wB2vZ8qH
│
└─ Memory: amem_6wT3kL4pR8mV2tY7xB9nH (similarity: 0.87)
   └─ Usage record: amu_2nQ9kL5pR7mV4tY8xB3wH
```

### Tool Execution

```
Assistant message: am_9vT2kL5pR8mN4xB7wZ3qH
└─ Tool Use: atu_4xQ7kL2pR9mV5tY3wB8nH
   ├─ Tool ID: at_8nQ5kL7pR2mV9tY3xB6wH
   ├─ Tool Name: web_search_builtin
   ├─ Status: success
   └─ Result: {...}
```

## Best Practices

### 1. Always Use Generated IDs

❌ **Don't** create IDs manually:
```go
conversationID := "ac_" + uuid.New().String()  // Wrong format
```

✅ **Do** use the IDGenerator:
```go
conversationID := idGen.GenerateConversationID()
```

### 2. Validate ID Prefixes

When accepting IDs from external sources (API, protocol messages), validate the prefix:

```go
func (s *Service) GetConversation(id string) (*Conversation, error) {
    if !strings.HasPrefix(id, "ac_") {
        return nil, fmt.Errorf("invalid conversation ID: expected ac_ prefix, got %s", id)
    }
    // ...
}
```

### 3. Use Typed Parameters

Make function signatures explicit about ID types:

```go
// Clear parameter names
func CreateMessage(conversationID, previousMessageID string, content string) (*Message, error)

// Even better: use type aliases
type ConversationID string
type MessageID string

func CreateMessage(conversationID ConversationID, previousID MessageID, content string) (*Message, error)
```

### 4. Include IDs in Logs

Always log relevant IDs for troubleshooting:

```go
logger.Info("processing message",
    zap.String("conversation_id", conversationID),
    zap.String("message_id", messageID),
)
```

### 5. Fallback Behavior

The ID generator includes fallback logic for the rare case when NanoID generation fails:

```go
func (g *Generator) generate(prefix string) string {
    id, err := gonanoid.New(21)
    if err != nil {
        return prefix + "_fallback"  // Returns "<prefix>_fallback" on error
    }
    return prefix + "_" + id
}
```

**Important**: If you encounter IDs like `ac_fallback`, `am_fallback`, etc. in your logs, this indicates an issue with the random number generator or NanoID library. While these fallback IDs are valid and won't cause crashes, they:

- **Are not unique**: Multiple failures will produce the same fallback ID
- **Indicate a system problem**: Check system entropy, library compatibility, or resource constraints
- **Should be investigated**: This should never happen in normal operation

If you see fallback IDs in production, file an incident report immediately.

## Related Documentation

- [Database Schema](DATABASE.md)
- [Architecture Overview](ARCHITECTURE.md)
- [Protocol Specification](protocol/index.md)
