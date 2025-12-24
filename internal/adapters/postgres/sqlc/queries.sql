-- ==============================================================================
-- Conversations
-- ==============================================================================

-- name: CreateConversation :one
INSERT INTO alicia_conversations (
    id, title, status, livekit_room_name, preferences, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetConversationByID :one
SELECT * FROM alicia_conversations
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetConversationByLiveKitRoom :one
SELECT * FROM alicia_conversations
WHERE livekit_room_name = $1 AND deleted_at IS NULL;

-- name: UpdateConversation :exec
UPDATE alicia_conversations
SET title = $2,
    status = $3,
    livekit_room_name = $4,
    preferences = $5,
    updated_at = $6
WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteConversation :exec
UPDATE alicia_conversations
SET deleted_at = $2
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListConversations :many
SELECT * FROM alicia_conversations
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListActiveConversations :many
SELECT * FROM alicia_conversations
WHERE status = 'active' AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- ==============================================================================
-- Messages
-- ==============================================================================

-- name: CreateMessage :one
INSERT INTO alicia_messages (
    id, conversation_id, sequence_number, previous_id, message_role, contents, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetMessageByID :one
SELECT * FROM alicia_messages
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateMessage :exec
UPDATE alicia_messages
SET previous_id = $2,
    contents = $3,
    updated_at = $4
WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteMessage :exec
UPDATE alicia_messages
SET deleted_at = $2
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetMessagesByConversation :many
SELECT * FROM alicia_messages
WHERE conversation_id = $1 AND deleted_at IS NULL
ORDER BY sequence_number ASC;

-- name: GetLatestMessagesByConversation :many
SELECT * FROM alicia_messages
WHERE conversation_id = $1 AND deleted_at IS NULL
ORDER BY sequence_number DESC
LIMIT $2;

-- name: GetNextMessageSequenceNumber :one
SELECT COALESCE(MAX(sequence_number), 0) + 1 as next_sequence
FROM alicia_messages
WHERE conversation_id = $1 AND deleted_at IS NULL;

-- ==============================================================================
-- Sentences
-- ==============================================================================

-- name: CreateSentence :one
INSERT INTO alicia_sentences (
    id, message_id, sentence_sequence_number, text, audio_type, audio_format,
    duration_ms, audio_bytesize, audio_data, meta, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING *;

-- name: GetSentenceByID :one
SELECT * FROM alicia_sentences
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateSentence :exec
UPDATE alicia_sentences
SET text = $2,
    audio_type = $3,
    audio_format = $4,
    duration_ms = $5,
    audio_bytesize = $6,
    audio_data = $7,
    meta = $8,
    updated_at = $9
WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteSentence :exec
UPDATE alicia_sentences
SET deleted_at = $2
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetSentencesByMessage :many
SELECT * FROM alicia_sentences
WHERE message_id = $1 AND deleted_at IS NULL
ORDER BY sentence_sequence_number ASC;

-- name: GetNextSentenceSequenceNumber :one
SELECT COALESCE(MAX(sentence_sequence_number), 0) + 1 as next_sequence
FROM alicia_sentences
WHERE message_id = $1 AND deleted_at IS NULL;

-- ==============================================================================
-- Audio
-- ==============================================================================

-- name: CreateAudio :one
INSERT INTO alicia_audio (
    id, message_id, audio_type, audio_format, audio_data, duration_ms,
    transcription, livekit_track_sid, transcription_meta, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetAudioByID :one
SELECT * FROM alicia_audio
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetAudioByMessage :one
SELECT * FROM alicia_audio
WHERE message_id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: GetAudioByLiveKitTrack :one
SELECT * FROM alicia_audio
WHERE livekit_track_sid = $1 AND deleted_at IS NULL;

-- name: UpdateAudio :exec
UPDATE alicia_audio
SET message_id = $2,
    audio_data = $3,
    duration_ms = $4,
    transcription = $5,
    livekit_track_sid = $6,
    transcription_meta = $7,
    updated_at = $8
WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteAudio :exec
UPDATE alicia_audio
SET deleted_at = $2
WHERE id = $1 AND deleted_at IS NULL;

-- ==============================================================================
-- Memory
-- ==============================================================================

-- name: CreateMemory :one
INSERT INTO alicia_memory (
    id, content, embeddings, embeddings_info, importance, confidence,
    user_rating, created_by, source_type, source_info, tags, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
) RETURNING *;

-- name: GetMemoryByID :one
SELECT * FROM alicia_memory
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateMemory :exec
UPDATE alicia_memory
SET content = $2,
    embeddings = $3,
    embeddings_info = $4,
    importance = $5,
    confidence = $6,
    user_rating = $7,
    created_by = $8,
    source_type = $9,
    source_info = $10,
    tags = $11,
    updated_at = $12
WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteMemory :exec
UPDATE alicia_memory
SET deleted_at = $2
WHERE id = $1 AND deleted_at IS NULL;

-- name: SearchMemoryByEmbedding :many
SELECT *, 1 - (embeddings <=> $1) as similarity_score
FROM alicia_memory
WHERE deleted_at IS NULL AND embeddings IS NOT NULL
ORDER BY embeddings <=> $1
LIMIT $2;

-- name: SearchMemoryByEmbeddingWithThreshold :many
SELECT *, 1 - (embeddings <=> $1) as similarity_score
FROM alicia_memory
WHERE deleted_at IS NULL
  AND embeddings IS NOT NULL
  AND 1 - (embeddings <=> $1) >= $2
ORDER BY embeddings <=> $1
LIMIT $3;

-- name: GetMemoriesByTags :many
SELECT * FROM alicia_memory
WHERE deleted_at IS NULL AND tags && $1
ORDER BY importance DESC, created_at DESC
LIMIT $2;

-- ==============================================================================
-- Memory Usage
-- ==============================================================================

-- name: CreateMemoryUsage :one
INSERT INTO alicia_memory_used (
    id, conversation_id, message_id, memory_id, query_prompt, query_prompt_meta,
    similarity_score, meta, position_in_results, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetMemoryUsageByMessage :many
SELECT * FROM alicia_memory_used
WHERE message_id = $1 AND deleted_at IS NULL
ORDER BY position_in_results ASC;

-- name: GetMemoryUsageByConversation :many
SELECT * FROM alicia_memory_used
WHERE conversation_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetMemoryUsageByMemory :many
SELECT * FROM alicia_memory_used
WHERE memory_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- ==============================================================================
-- Tools
-- ==============================================================================

-- name: CreateTool :one
INSERT INTO alicia_tools (
    id, name, description, schema, enabled, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetToolByID :one
SELECT * FROM alicia_tools
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetToolByName :one
SELECT * FROM alicia_tools
WHERE name = $1 AND deleted_at IS NULL;

-- name: UpdateTool :exec
UPDATE alicia_tools
SET name = $2,
    description = $3,
    schema = $4,
    enabled = $5,
    updated_at = $6
WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteTool :exec
UPDATE alicia_tools
SET deleted_at = $2
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListEnabledTools :many
SELECT * FROM alicia_tools
WHERE enabled = true AND deleted_at IS NULL
ORDER BY name ASC;

-- name: ListAllTools :many
SELECT * FROM alicia_tools
WHERE deleted_at IS NULL
ORDER BY name ASC;

-- ==============================================================================
-- Tool Uses
-- ==============================================================================

-- name: CreateToolUse :one
INSERT INTO alicia_tool_uses (
    id, message_id, tool_name, tool_arguments, tool_result, status,
    error_message, sequence_number, completed_at, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetToolUseByID :one
SELECT * FROM alicia_tool_uses
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateToolUse :exec
UPDATE alicia_tool_uses
SET tool_result = $2,
    status = $3,
    error_message = $4,
    completed_at = $5,
    updated_at = $6
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetToolUsesByMessage :many
SELECT * FROM alicia_tool_uses
WHERE message_id = $1 AND deleted_at IS NULL
ORDER BY sequence_number ASC;

-- name: GetPendingToolUses :many
SELECT * FROM alicia_tool_uses
WHERE status = 'pending' AND deleted_at IS NULL
ORDER BY created_at ASC
LIMIT $1;

-- ==============================================================================
-- Reasoning Steps
-- ==============================================================================

-- name: CreateReasoningStep :one
INSERT INTO alicia_reasoning_steps (
    id, message_id, content, sequence_number, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetReasoningStepsByMessage :many
SELECT * FROM alicia_reasoning_steps
WHERE message_id = $1 AND deleted_at IS NULL
ORDER BY sequence_number ASC;

-- name: GetNextReasoningStepSequenceNumber :one
SELECT COALESCE(MAX(sequence_number), 0) + 1 as next_sequence
FROM alicia_reasoning_steps
WHERE message_id = $1 AND deleted_at IS NULL;

-- ==============================================================================
-- Commentaries
-- ==============================================================================

-- name: CreateCommentary :one
INSERT INTO alicia_user_conversation_commentaries (
    id, content, conversation_id, message_id, created_by, meta, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetCommentaryByID :one
SELECT * FROM alicia_user_conversation_commentaries
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetCommentariesByConversation :many
SELECT * FROM alicia_user_conversation_commentaries
WHERE conversation_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetCommentariesByMessage :many
SELECT * FROM alicia_user_conversation_commentaries
WHERE message_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- ==============================================================================
-- Meta
-- ==============================================================================

-- name: UpsertMeta :one
INSERT INTO alicia_meta (
    id, ref, key, value, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6
)
ON CONFLICT (ref, key) WHERE deleted_at IS NULL
DO UPDATE SET
    value = EXCLUDED.value,
    updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: GetMeta :one
SELECT * FROM alicia_meta
WHERE ref = $1 AND key = $2 AND deleted_at IS NULL;

-- name: GetAllMetaByRef :many
SELECT * FROM alicia_meta
WHERE ref = $1 AND deleted_at IS NULL
ORDER BY key ASC;

-- name: DeleteMeta :exec
UPDATE alicia_meta
SET deleted_at = $2
WHERE ref = $1 AND key = $3 AND deleted_at IS NULL;

-- name: DeleteAllMetaByRef :exec
UPDATE alicia_meta
SET deleted_at = $2
WHERE ref = $1 AND deleted_at IS NULL;
