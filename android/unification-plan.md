# Android Unification Plan

Plan to add conversations, memory support, and notes to the Android app, aligning it with the web client's capabilities while keeping the voice-first UX simple and fast.

## Design Decisions

### Transport
- **LiveKit for voice only** (unchanged from current behavior).
- **REST API for text conversations** — Android uses synchronous HTTP requests to the Alicia API. No WebSocket connection.
- A blocking variant of the message creation endpoint returns the full assistant response in one round-trip (used with `use_pareto: false`).

### Offline Mode
- Fully server-dependent. Remove the direct Qwen3 8B endpoint (`/v1/chat/completions`).
- No offline fallback or message queuing.
- When the server is unreachable and no local skill matches, the request simply fails (no special degradation UX).

### Notes vs Memories
- **Notes**: User-written reference documents (title + content). The LLM can read them but never creates, edits, or deletes them.
- **Memories**: LLM-extracted from conversations. Not meant for direct user consumption.
- These are separate concepts and stay separate. Voice notes on Android remain local recordings, unrelated to the notes system.

### Note Retrieval
- Server generates embeddings for notes (pgvector, 1024 dimensions).
- Notes are retrieved semantically like memories — included in agent context if similarity exceeds a configurable threshold.
- Embedding-similarity only. No full-text search.
- User preference controls the threshold (`notes_similarity_threshold`).
- No size limit on note content. Content is truncated before embedding generation if it exceeds the embedding model's context window. One embedding per note.
- Note IDs are client-generated (UUID).

### Branching
- Android uses linear chat only. No branching, no sibling navigation.
- Regenerate replaces the last message rather than creating a branch. This is destructive (no undo) — the server retains branches but Android only shows the current one.

### Pareto Optimization
- Per-request flag: `use_pareto` sent with each `GenerationRequest`.
- Android always sends `use_pareto: false` for faster responses.
- Web sends `use_pareto: true` (or based on user's Pareto preference in settings).
- This is the smallest change — no server-side device state needed.

### Android UI Transparency
- Show tool name only (e.g. "Used: web_search") as a small badge.
- No expandable tool details or memory traces on Android.
- Users can inspect full details on the web client.

### Notes Structure
- Flat list: title + content. No tags, categories, or folders.
- Keep it simple.

### Notes Web UI
- Sidebar icon (similar to Settings/Memory icons) linking to `/notes`.
- Dedicated full-page editor at `/notes` route.

---

## Gap Analysis: Android vs Web

| Feature | Web | Android (Current) | Android (Planned) |
|---------|-----|-------------------|-------------------|
| Conversations | Full persistence, branching, streaming | In-memory only (10 turns), lost on close | Persistent, linear, synchronous |
| Backend connection | WebSocket + REST to Alicia API | Direct HTTP to raw Qwen3 8B | REST only to Alicia API |
| Memory | Vector retrieval + auto-extraction | None | Server-side (transparent to user) |
| Tools (MCP) | Full tool calling with UI | None | Tool calling, show tool name badge |
| Response delivery | Sentence-by-sentence via WS | Single-shot response | Single-shot (blocking POST) |
| Preferences | Synced to agent in real-time | Local-only (DataStore) | Synced via REST |
| Notes | Does not exist | Does not exist | Full CRUD (create, edit, delete) |
| Pareto | Enabled (multi-generation optimization) | N/A (bypasses agent) | Disabled per-request for speed |

---

## Implementation Plan

### Part 1: Notes System (API + Agent + Web)

#### Database

New migration adding `notes` table:

```sql
CREATE TABLE notes (
    id         TEXT PRIMARY KEY,  -- client-generated UUID
    user_id    TEXT NOT NULL,
    title      TEXT NOT NULL DEFAULT '',
    content    TEXT NOT NULL,
    embedding  vector(1024),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_notes_user ON notes(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_notes_embed ON notes USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100) WHERE deleted_at IS NULL;
```

Embedding generated server-side on create/update using the same embedding model as memories. Content is truncated to the model's context window before embedding.

#### API Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/api/v1/notes` | Create (title, content) - generates embedding |
| GET | `/api/v1/notes` | List user's notes |
| GET | `/api/v1/notes/{id}` | Get single note |
| PUT | `/api/v1/notes/{id}` | Update (regenerates embedding) |
| DELETE | `/api/v1/notes/{id}` | Soft delete |

#### Agent Integration

- During the memory retrieval phase (`agent.go:450-464`), also query notes by embedding similarity.
- Include relevant notes in system prompt, clearly marked:
  ```
  [User Note: {title}]
  {content}
  [/User Note]
  ```
- Agent system prompt instructs it to reference notes but never modify them.
- User preferences control inclusion:
  - `notes_similarity_threshold` (float, default 0.7)
  - `notes_max_count` (int, default 3)

#### Web UI

- New sidebar icon linking to `/notes`.
- `/notes` page: note list (left panel), editor (right panel).
- Simple CRUD: create, edit title/content, delete.
- No tags, folders, or categorization.

---

### Part 2: Pareto Bypass (Shared Protocol Change)

#### Protocol

Add `UsePareto` field to `GenerationRequest` in `shared/protocol/types.go`:

```go
type GenerationRequest struct {
    ConversationID string `msgpack:"conversation_id" json:"conversation_id"`
    MessageID      string `msgpack:"message_id" json:"message_id"`
    PreviousID     string `msgpack:"previous_id" json:"previous_id"`
    RequestType    string `msgpack:"request_type" json:"request_type"`
    EnableTools    bool   `msgpack:"enable_tools" json:"enable_tools"`
    EnableReasoning bool  `msgpack:"enable_reasoning" json:"enable_reasoning"`
    EnableStreaming bool   `msgpack:"enable_streaming" json:"enable_streaming"`
    UsePareto      bool   `msgpack:"use_pareto" json:"use_pareto"` // NEW
    // ... trace context fields
}
```

#### Agent

Check the flag in `agent.go` at the decision points (lines 218, 260, 312):

```go
if cfg.ParetoMode && req.UsePareto {
    generateParetoResponse(...)
} else {
    generateResponse(...)
}
```

#### Clients

- **Web**: Sends `use_pareto: true` (or based on existing Pareto preference in settings).
- **Android**: Always sends `use_pareto: false`.

---

### Part 3: Android Conversations & Memory

#### Remove Direct LLM Calls

Delete the current `LlmClient.kt` that calls `/v1/chat/completions` directly with Qwen3 8B. All LLM interaction goes through the Alicia API and agent.

#### New API Endpoint: Synchronous Message Creation

New blocking variant of the message creation endpoint:

```
POST /api/v1/conversations/{id}/messages?sync=true
Body: { "content": "...", "use_pareto": false }
Response: { "user_message": {...}, "assistant_message": {...} }
```

- Creates the user message, sends `GenerationRequest` to agent, blocks until the agent finishes.
- Returns both the user message and the complete assistant message (including any tool use info).
- Intended for `use_pareto: false` requests where latency is acceptable in a single round-trip.
- The existing async flow (WebSocket-based streaming) remains unchanged for the web client.

#### New Android Components

| Component | Purpose |
|-----------|---------|
| `service/AliciaApiClient.kt` | REST client for Alicia API (conversations, messages, preferences). Replaces `LlmClient.kt`. |
| `storage/ConversationRepository.kt` | REST calls for conversation CRUD + message fetching. Local Room DB cache. |
| `ConversationListActivity.kt` | UI: list conversations, create new, tap to open. |
| Chat UI in `MainActivity.kt` | Show linear message history, text input, send button. |

#### Message Flow

1. User types or speaks (transcription via Whisper, unchanged).
2. `POST /api/v1/conversations/{id}/messages?sync=true` with body `{content, use_pareto: false}`.
3. API creates user message, sends `GenerationRequest` to agent, waits for completion.
4. Response contains the full assistant message (content + tool use metadata).
5. Android sends response text to Kokoro TTS (existing `TtsManager`, unchanged).

#### Voice via LiveKit

- LiveKit audio streaming for real-time voice (unchanged, separate from text conversations).
- Voice interactions are not persisted to the conversation API. Voice remains ephemeral by design.

#### Memory

- Fully server-side and transparent. The agent extracts memories from Android conversations automatically (same as web conversations).
- No memory UI on Android.

#### Tool Transparency

- The assistant message response includes tool use metadata (tool name, success/error).
- Android shows a small chip: "Used: {tool_name}" for each tool invocation.
- No expandable details on Android.

#### Local Persistence

- Room DB caches conversations and messages for offline viewing.
- When online, server is source of truth — local cache is overwritten on fetch.
- Messages not editable offline; send requires connectivity.
- Pagination deferred to a later design pass.

---

### Part 4: User Preferences Updates

Add to `user_preferences` table and preference sync:

| Preference | Type | Default | Purpose |
|------------|------|---------|---------|
| `notes_similarity_threshold` | float | 0.7 | Min cosine similarity for note inclusion |
| `notes_max_count` | int | 3 | Max notes to include per request |

These sync to the agent via the existing `TypePreferencesUpdate` WebSocket mechanism.

---

## Changes by Component

| Component | Changes |
|-----------|---------|
| **shared/protocol** | Add `UsePareto` to `GenerationRequest` |
| **API: migration** | New `notes` table with embedding index |
| **API: handlers** | New `notes.go` handler (CRUD) |
| **API: handlers** | Synchronous message creation endpoint (`?sync=true`) |
| **API: services** | New `notes.go` service (embedding generation, truncation) |
| **API: store** | New `notes.go` store (DB queries, vector search) |
| **API: ws.go** | Pass `UsePareto` from message creation to `GenerationRequest` |
| **API: preferences** | Add `notes_similarity_threshold`, `notes_max_count` validation |
| **Agent: agent.go** | Check `req.UsePareto` flag at Pareto decision points |
| **Agent: agent.go** | Query notes alongside memories during retrieval |
| **Agent: db.go** | Add `SearchNotes()` function |
| **Web: sidebar** | Add Notes icon |
| **Web: pages** | New `/notes` page with list + editor |
| **Web: stores** | New `notesStore.ts` |
| **Web: hooks** | New `useNotes.ts` hook for CRUD |
| **Web: services/api.ts** | Add notes endpoints |
| **Web: chat** | Send `use_pareto` based on preference |
| **Android: service** | New `AliciaApiClient.kt` (REST client for Alicia API) |
| **Android: storage** | New `ConversationRepository.kt` (REST + Room DB) |
| **Android: UI** | New conversation list, chat view |
| **Android: service** | Remove `LlmClient.kt` (direct Qwen3 calls) |
| **Android: skills** | Update `SkillRouter.kt` to use `AliciaApiClient` instead of `LlmClient` |
| **Android: session** | Update `AliciaInteractionSession.kt` to use synchronous API calls |

---

## Implementation Order

1. **Pareto bypass** — Smallest change, unblocks Android work. Protocol + agent change.
2. **Synchronous message endpoint** — Blocking `?sync=true` variant of message creation. Unblocks Android.
3. **Notes: API + database** — CRUD endpoints, embedding generation with truncation.
4. **Notes: Agent integration** — Retrieve notes alongside memories during response generation.
5. **Notes: Web UI** — Sidebar icon, /notes page, editor.
6. **Android: API client + conversations** — New `AliciaApiClient.kt`, conversation CRUD, Room DB cache.
7. **Android: Chat UI** — Conversation list, message display, send via sync endpoint.
8. **Android: Remove direct LLM** — Delete `LlmClient.kt`, wire `SkillRouter` through `AliciaApiClient`.
9. **User preferences** — Add notes thresholds, sync to agent.
