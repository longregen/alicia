package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/shared/db"
	"github.com/pgvector/pgvector-go"
)

func ConnectDB(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	return db.ConnectSimple(ctx, databaseURL)
}

// --- Messages ---

// Avoids N+1 queries by batching tool uses and memories.
func LoadConversationFull(ctx context.Context, pool *pgxpool.Pool, conversationID string) ([]Message, error) {
	// Query 1: All messages
	rows, err := pool.Query(ctx, `
		SELECT id, conversation_id, previous_id, branch_index, role, content, reasoning, status
		FROM messages
		WHERE conversation_id = $1 AND deleted_at IS NULL
		ORDER BY created_at
	`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	msgIndex := make(map[string]int) // message ID -> index in slice
	for rows.Next() {
		var m Message
		var prevID *string
		if err := rows.Scan(&m.ID, &m.ConversationID, &prevID, &m.BranchIndex, &m.Role, &m.Content, &m.Reasoning, &m.Status); err != nil {
			return nil, err
		}
		if prevID != nil {
			m.PreviousID = *prevID
		}
		msgIndex[m.ID] = len(messages)
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		return messages, nil
	}

	// Query 2: All tool uses for this conversation's messages
	tuRows, err := pool.Query(ctx, `
		SELECT tu.id, tu.message_id, tu.tool_name, tu.arguments, tu.result, tu.status, tu.error
		FROM tool_uses tu
		JOIN messages m ON m.id = tu.message_id
		WHERE m.conversation_id = $1 AND m.deleted_at IS NULL
		ORDER BY tu.created_at
	`, conversationID)
	if err != nil {
		return nil, err
	}
	defer tuRows.Close()

	for tuRows.Next() {
		var tu ToolUse
		var msgID string
		var argsJSON, resultJSON []byte
		var status, errorStr string
		if err := tuRows.Scan(&tu.ID, &msgID, &tu.ToolName, &argsJSON, &resultJSON, &status, &errorStr); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(argsJSON, &tu.Arguments); err != nil {
			slog.Warn("failed to unmarshal tool arguments", "tool_use_id", tu.ID, "error", err)
		}
		if err := json.Unmarshal(resultJSON, &tu.Result); err != nil {
			slog.Warn("failed to unmarshal tool result", "tool_use_id", tu.ID, "error", err)
		}
		tu.Success = status == "success"
		tu.Error = errorStr

		if idx, ok := msgIndex[msgID]; ok {
			messages[idx].ToolUses = append(messages[idx].ToolUses, tu)
		}
	}
	if err := tuRows.Err(); err != nil {
		return nil, err
	}

	// Query 3: All memories used for this conversation's messages
	memRows, err := pool.Query(ctx, `
		SELECT mu.message_id, mem.id, mem.content, mu.similarity
		FROM memory_uses mu
		JOIN memories mem ON mem.id = mu.memory_id
		JOIN messages m ON m.id = mu.message_id
		WHERE m.conversation_id = $1 AND m.deleted_at IS NULL AND mem.deleted_at IS NULL
		ORDER BY mu.similarity DESC
	`, conversationID)
	if err != nil {
		return nil, err
	}
	defer memRows.Close()

	for memRows.Next() {
		var msgID string
		var mem Memory
		if err := memRows.Scan(&msgID, &mem.ID, &mem.Content, &mem.Similarity); err != nil {
			return nil, err
		}
		if idx, ok := msgIndex[msgID]; ok {
			messages[idx].Memories = append(messages[idx].Memories, mem)
		}
	}

	return messages, memRows.Err()
}

func GetMessage(ctx context.Context, pool *pgxpool.Pool, messageID string) (*Message, error) {
	var m Message
	var prevID *string
	err := pool.QueryRow(ctx, `
		SELECT id, conversation_id, previous_id, branch_index, role, content, reasoning, status
		FROM messages
		WHERE id = $1 AND deleted_at IS NULL
	`, messageID).Scan(&m.ID, &m.ConversationID, &prevID, &m.BranchIndex, &m.Role, &m.Content, &m.Reasoning, &m.Status)
	if err != nil {
		return nil, err
	}
	if prevID != nil {
		m.PreviousID = *prevID
	}
	return &m, nil
}

func GetConversationIDForMessage(ctx context.Context, pool *pgxpool.Pool, messageID string) (string, error) {
	var convID string
	err := pool.QueryRow(ctx, `
		SELECT conversation_id FROM messages WHERE id = $1
	`, messageID).Scan(&convID)
	return convID, err
}

func GetPreviousUserMessage(ctx context.Context, pool *pgxpool.Pool, assistantMessageID string) (*Message, error) {
	msg, err := GetMessage(ctx, pool, assistantMessageID)
	if err != nil {
		return nil, err
	}
	if msg.PreviousID == "" {
		return nil, nil
	}
	return GetMessage(ctx, pool, msg.PreviousID)
}

func CreateMessage(ctx context.Context, pool *pgxpool.Pool, id, convID, role, content, reasoning string, previousID *string) error {
	// Use separate parameters for subquery to avoid pgx type inference issues
	var err error
	if previousID == nil {
		_, err = pool.Exec(ctx, `
			INSERT INTO messages (id, conversation_id, previous_id, branch_index, role, content, reasoning, status, created_at)
			VALUES ($1, $2, NULL,
				COALESCE((
					SELECT MAX(branch_index) + 1
					FROM messages
					WHERE conversation_id = $7
					  AND previous_id IS NULL
					  AND deleted_at IS NULL
				), 0),
				$3, $4, $5, 'pending', $6)
		`, id, convID, role, content, reasoning, time.Now().UTC(), convID)
	} else {
		_, err = pool.Exec(ctx, `
			INSERT INTO messages (id, conversation_id, previous_id, branch_index, role, content, reasoning, status, created_at)
			VALUES ($1, $2, $3,
				COALESCE((
					SELECT MAX(branch_index) + 1
					FROM messages
					WHERE conversation_id = $8
					  AND previous_id = $9
					  AND deleted_at IS NULL
				), 0),
				$4, $5, $6, 'pending', $7)
		`, id, convID, *previousID, role, content, reasoning, time.Now().UTC(), convID, *previousID)
	}
	return err
}

func UpdateMessage(ctx context.Context, pool *pgxpool.Pool, messageID, content, reasoning, status string) error {
	_, err := pool.Exec(ctx, `
		UPDATE messages SET content = $2, reasoning = $3, status = $4 WHERE id = $1
	`, messageID, content, reasoning, status)
	return err
}

func UpdateMessageTraceID(ctx context.Context, pool *pgxpool.Pool, messageID, traceID string) error {
	_, err := pool.Exec(ctx, `
		UPDATE messages SET trace_id = $2 WHERE id = $1
	`, messageID, traceID)
	return err
}

// --- Tool Uses ---

func SaveToolUse(ctx context.Context, pool *pgxpool.Pool, messageID string, tu ToolUse) error {
	argsJSON, err := json.Marshal(tu.Arguments)
	if err != nil {
		return fmt.Errorf("marshal tool arguments: %w", err)
	}
	resultJSON, err := json.Marshal(tu.Result)
	if err != nil {
		return fmt.Errorf("marshal tool result: %w", err)
	}
	status := "error"
	if tu.Success {
		status = "success"
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO tool_uses (id, message_id, tool_name, arguments, result, status, error, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`, tu.ID, messageID, tu.ToolName, argsJSON, resultJSON, status, tu.Error)
	return err
}

// --- Memories ---

func SearchMemories(ctx context.Context, pool *pgxpool.Pool, embedding []float32, threshold float32, limit int) ([]Memory, error) {
	vec := pgvector.NewVector(embedding)
	rows, err := pool.Query(ctx, `
		SELECT id, content, 1 - (embedding <=> $1) as similarity
		FROM memories
		WHERE deleted_at IS NULL AND archived = false
		  AND embedding IS NOT NULL
		  AND 1 - (embedding <=> $1) >= $2
		ORDER BY embedding <=> $1
		LIMIT $3
	`, vec, threshold, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []Memory
	for rows.Next() {
		var m Memory
		if err := rows.Scan(&m.ID, &m.Content, &m.Similarity); err != nil {
			return nil, err
		}
		memories = append(memories, m)
	}
	return memories, rows.Err()
}

func RecordMemoryUse(ctx context.Context, pool *pgxpool.Pool, id, memoryID, messageID, conversationID string, similarity float32) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO memory_uses (id, memory_id, message_id, conversation_id, similarity, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, id, memoryID, messageID, conversationID, similarity)
	return err
}

func CreateMemory(ctx context.Context, pool *pgxpool.Pool, id, content string, embedding []float32, importance float32) error {
	vec := pgvector.NewVector(embedding)
	_, err := pool.Exec(ctx, `
		INSERT INTO memories (id, content, embedding, importance, pinned, archived, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, false, false, '{}', NOW(), NOW())
	`, id, content, vec, importance)
	return err
}

// --- Memory Generations ---

type MemoryGeneration struct {
	ID                      string
	ConversationID          string
	MessageID               string
	MemoryContent           string
	ExtractPromptName       string
	ExtractPromptVersion    int
	ImportanceRating        int
	ImportanceThinking      string
	ImportancePromptName    string
	ImportancePromptVersion int
	HistoricalRating        int
	HistoricalThinking      string
	HistoricalPromptName    string
	HistoricalPromptVersion int
	PersonalRating          int
	PersonalThinking        string
	PersonalPromptName      string
	PersonalPromptVersion   int
	FactualRating           int
	FactualThinking         string
	FactualPromptName       string
	FactualPromptVersion    int
	RerankDecision          string
	RerankPromptName        string
	RerankPromptVersion     int
	Accepted                bool
	MemoryID                *string
}

func CreateMemoryGeneration(ctx context.Context, pool *pgxpool.Pool, g MemoryGeneration) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO memory_generations (
			id, conversation_id, message_id, memory_content,
			extract_prompt_name, extract_prompt_version,
			importance_rating, importance_thinking, importance_prompt_name, importance_prompt_version,
			historical_rating, historical_thinking, historical_prompt_name, historical_prompt_version,
			personal_rating, personal_thinking, personal_prompt_name, personal_prompt_version,
			factual_rating, factual_thinking, factual_prompt_name, factual_prompt_version,
			rerank_decision, rerank_prompt_name, rerank_prompt_version,
			accepted, memory_id
		) VALUES (
			$1, $2, $3, $4,
			$5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14,
			$15, $16, $17, $18,
			$19, $20, $21, $22,
			$23, $24, $25,
			$26, $27
		)
	`,
		g.ID, g.ConversationID, g.MessageID, g.MemoryContent,
		nilIfEmpty(g.ExtractPromptName), nilIfZero(g.ExtractPromptVersion),
		g.ImportanceRating, nilIfEmpty(g.ImportanceThinking), nilIfEmpty(g.ImportancePromptName), nilIfZero(g.ImportancePromptVersion),
		g.HistoricalRating, nilIfEmpty(g.HistoricalThinking), nilIfEmpty(g.HistoricalPromptName), nilIfZero(g.HistoricalPromptVersion),
		g.PersonalRating, nilIfEmpty(g.PersonalThinking), nilIfEmpty(g.PersonalPromptName), nilIfZero(g.PersonalPromptVersion),
		g.FactualRating, nilIfEmpty(g.FactualThinking), nilIfEmpty(g.FactualPromptName), nilIfZero(g.FactualPromptVersion),
		nilIfEmpty(g.RerankDecision), nilIfEmpty(g.RerankPromptName), nilIfZero(g.RerankPromptVersion),
		g.Accepted, g.MemoryID,
	)
	return err
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nilIfZero(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

// --- Notes ---

type Note struct {
	ID         string
	Title      string
	Content    string
	Similarity float32
}

func SearchNotes(ctx context.Context, pool *pgxpool.Pool, userID string, embedding []float32, threshold float32, limit int) ([]Note, error) {
	vec := pgvector.NewVector(embedding)
	rows, err := pool.Query(ctx, `
		SELECT id, title, content, 1 - (embedding <=> $1) as similarity
		FROM notes
		WHERE user_id = $2 AND deleted_at IS NULL AND embedding IS NOT NULL
		      AND 1 - (embedding <=> $1) >= $3
		ORDER BY embedding <=> $1
		LIMIT $4
	`, vec, userID, threshold, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.Similarity); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// --- Tools ---

func LoadTools(ctx context.Context, pool *pgxpool.Pool) ([]Tool, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, name, description, schema
		FROM tools
		WHERE enabled = true
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tools []Tool
	for rows.Next() {
		var t Tool
		var schemaJSON []byte
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &schemaJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(schemaJSON, &t.Schema); err != nil {
			slog.Warn("failed to unmarshal tool schema", "tool_id", t.ID, "error", err)
		}
		tools = append(tools, t)
	}
	return tools, rows.Err()
}

// --- Conversations ---

func GetConversationTitle(ctx context.Context, pool *pgxpool.Pool, conversationID string) (string, error) {
	var title string
	err := pool.QueryRow(ctx, `SELECT title FROM conversations WHERE id = $1`, conversationID).Scan(&title)
	return title, err
}

func UpdateConversationTitle(ctx context.Context, pool *pgxpool.Pool, conversationID, title string) error {
	_, err := pool.Exec(ctx, `UPDATE conversations SET title = $2, updated_at = NOW() WHERE id = $1`, conversationID, title)
	return err
}

func UpdateConversationTip(ctx context.Context, pool *pgxpool.Pool, conversationID, messageID string) error {
	_, err := pool.Exec(ctx, `
		UPDATE conversations SET tip_message_id = $2, updated_at = NOW() WHERE id = $1
	`, conversationID, messageID)
	return err
}

// --- MCP Servers ---

type MCPServerConfig struct {
	ID            string
	Name          string
	TransportType string // "stdio" or "sse"
	Command       string
	Args          []string
	URL           string
	Enabled       bool
}

func LoadEnabledMCPServers(ctx context.Context, pool *pgxpool.Pool) ([]MCPServerConfig, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, name, transport_type, command, args, url, enabled
		FROM mcp_servers
		WHERE enabled = true AND deleted_at IS NULL
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []MCPServerConfig
	for rows.Next() {
		var s MCPServerConfig
		if err := rows.Scan(&s.ID, &s.Name, &s.TransportType, &s.Command, &s.Args, &s.URL, &s.Enabled); err != nil {
			return nil, err
		}
		servers = append(servers, s)
	}
	return servers, rows.Err()
}

