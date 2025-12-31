package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type VoteRepository struct {
	BaseRepository
}

func NewVoteRepository(pool *pgxpool.Pool) *VoteRepository {
	return &VoteRepository{
		BaseRepository: NewBaseRepository(pool),
	}
}

func (r *VoteRepository) Create(ctx context.Context, vote *models.Vote) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	// Map Vote.Value to database vote string
	voteStr := "up"
	if vote.Value == models.VoteValueDown {
		voteStr = "down"
	}

	// Handle optional quick_feedback and note
	var quickFeedback, note sql.NullString
	if vote.QuickFeedback != "" {
		quickFeedback = sql.NullString{String: vote.QuickFeedback, Valid: true}
	}
	if vote.Note != "" {
		note = sql.NullString{String: vote.Note, Valid: true}
	}

	// Build query based on target type to get conversation_id from appropriate source
	var query string
	var args []interface{}

	switch vote.TargetType {
	case "message":
		// For message votes, get conversation_id from the message itself
		query = `
			INSERT INTO alicia_votes (
				id, conversation_id, message_id, target_type, target_id, vote, quick_feedback, note, created_at, updated_at
			)
			SELECT $1, m.conversation_id, $2, $3, $4, $5, $6, $7, $8, $9
			FROM alicia_messages m
			WHERE m.id = $2 AND m.deleted_at IS NULL`
		args = []interface{}{
			vote.ID, vote.MessageID, vote.TargetType, vote.TargetID,
			voteStr, quickFeedback, note, vote.CreatedAt, vote.UpdatedAt,
		}
	case "tool_use":
		// For tool_use votes, get conversation_id through the tool_use's message
		query = `
			INSERT INTO alicia_votes (
				id, conversation_id, message_id, target_type, target_id, vote, quick_feedback, note, created_at, updated_at
			)
			SELECT $1, m.conversation_id, tu.message_id, $2, $3, $4, $5, $6, $7, $8
			FROM alicia_tool_uses tu
			JOIN alicia_messages m ON m.id = tu.message_id AND m.deleted_at IS NULL
			WHERE tu.id = $3 AND tu.deleted_at IS NULL`
		args = []interface{}{
			vote.ID, vote.TargetType, vote.TargetID,
			voteStr, quickFeedback, note, vote.CreatedAt, vote.UpdatedAt,
		}
	case "reasoning":
		// For reasoning votes, get conversation_id through the reasoning step's message
		query = `
			INSERT INTO alicia_votes (
				id, conversation_id, message_id, target_type, target_id, vote, quick_feedback, note, created_at, updated_at
			)
			SELECT $1, m.conversation_id, rs.message_id, $2, $3, $4, $5, $6, $7, $8
			FROM alicia_reasoning_steps rs
			JOIN alicia_messages m ON m.id = rs.message_id AND m.deleted_at IS NULL
			WHERE rs.id = $3 AND rs.deleted_at IS NULL`
		args = []interface{}{
			vote.ID, vote.TargetType, vote.TargetID,
			voteStr, quickFeedback, note, vote.CreatedAt, vote.UpdatedAt,
		}
	case "memory":
		// For memory votes, get conversation_id through memory_used junction table
		// If no memory_used record exists, use NULL for message_id and look up any conversation
		query = `
			INSERT INTO alicia_votes (
				id, conversation_id, message_id, target_type, target_id, vote, quick_feedback, note, created_at, updated_at
			)
			SELECT $1, COALESCE(mu.conversation_id, (SELECT id FROM alicia_conversations ORDER BY created_at DESC LIMIT 1)),
			       mu.message_id, $2, $3, $4, $5, $6, $7, $8
			FROM alicia_memory mem
			LEFT JOIN alicia_memory_used mu ON mu.memory_id = mem.id
			WHERE mem.id = $3 AND mem.deleted_at IS NULL
			LIMIT 1`
		args = []interface{}{
			vote.ID, vote.TargetType, vote.TargetID,
			voteStr, quickFeedback, note, vote.CreatedAt, vote.UpdatedAt,
		}
	default:
		// Fallback to original behavior for unknown types
		query = `
			INSERT INTO alicia_votes (
				id, conversation_id, message_id, target_type, target_id, vote, quick_feedback, note, created_at, updated_at
			)
			SELECT $1, m.conversation_id, $2, $3, $4, $5, $6, $7, $8, $9
			FROM alicia_messages m
			WHERE m.id = $2 AND m.deleted_at IS NULL`
		args = []interface{}{
			vote.ID, vote.MessageID, vote.TargetType, vote.TargetID,
			voteStr, quickFeedback, note, vote.CreatedAt, vote.UpdatedAt,
		}
	}

	_, err := r.conn(ctx).Exec(ctx, query, args...)
	return err
}

func (r *VoteRepository) Delete(ctx context.Context, targetType string, targetID string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		UPDATE alicia_votes
		SET deleted_at = NOW()
		WHERE target_type = $1 AND target_id = $2 AND deleted_at IS NULL`

	_, err := r.conn(ctx).Exec(ctx, query, targetType, targetID)
	return err
}

func (r *VoteRepository) GetByTarget(ctx context.Context, targetType string, targetID string) ([]*models.Vote, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, target_type, target_id, message_id, vote, quick_feedback, note, created_at, updated_at
		FROM alicia_votes
		WHERE target_type = $1 AND target_id = $2 AND deleted_at IS NULL
		ORDER BY created_at DESC`

	rows, err := r.conn(ctx).Query(ctx, query, targetType, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanVotes(rows)
}

func (r *VoteRepository) GetByMessage(ctx context.Context, messageID string) ([]*models.Vote, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, target_type, target_id, message_id, vote, quick_feedback, note, created_at, updated_at
		FROM alicia_votes
		WHERE message_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`

	rows, err := r.conn(ctx).Query(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanVotes(rows)
}

func (r *VoteRepository) GetAggregates(ctx context.Context, targetType string, targetID string) (*models.VoteAggregates, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT
			target_type,
			target_id,
			COUNT(*) FILTER (WHERE vote = 'up') as upvotes,
			COUNT(*) FILTER (WHERE vote = 'down') as downvotes
		FROM alicia_votes
		WHERE target_type = $1 AND target_id = $2 AND deleted_at IS NULL
		GROUP BY target_type, target_id`

	var aggregates models.VoteAggregates
	var upvotes, downvotes int

	err := r.conn(ctx).QueryRow(ctx, query, targetType, targetID).Scan(
		&aggregates.TargetType,
		&aggregates.TargetID,
		&upvotes,
		&downvotes,
	)

	if err != nil {
		if checkNoRows(err) {
			// No votes found, return zero aggregates
			return &models.VoteAggregates{
				TargetType: targetType,
				TargetID:   targetID,
				Upvotes:    0,
				Downvotes:  0,
				NetScore:   0,
			}, nil
		}
		return nil, err
	}

	aggregates.Upvotes = upvotes
	aggregates.Downvotes = downvotes
	aggregates.NetScore = upvotes - downvotes

	return &aggregates, nil
}

func (r *VoteRepository) scanVotes(rows pgx.Rows) ([]*models.Vote, error) {
	votes := make([]*models.Vote, 0)

	for rows.Next() {
		var v models.Vote
		var voteStr string
		var messageID, quickFeedback, note sql.NullString

		err := rows.Scan(
			&v.ID,
			&v.TargetType,
			&v.TargetID,
			&messageID,
			&voteStr,
			&quickFeedback,
			&note,
			&v.CreatedAt,
			&v.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Map database vote string to Vote.Value
		if voteStr == "up" {
			v.Value = models.VoteValueUp
		} else if voteStr == "down" {
			v.Value = models.VoteValueDown
		}

		v.MessageID = getString(messageID)
		v.QuickFeedback = getString(quickFeedback)
		v.Note = getString(note)

		votes = append(votes, &v)
	}

	return votes, rows.Err()
}

func (r *VoteRepository) CountByTargetType(ctx context.Context, targetType string) (int, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `SELECT COUNT(*) FROM alicia_votes WHERE target_type = $1 AND deleted_at IS NULL`

	var count int
	err := r.conn(ctx).QueryRow(ctx, query, targetType).Scan(&count)
	return count, err
}

func (r *VoteRepository) GetToolUseVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithToolContext, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT
			v.id, v.target_type, v.target_id, v.message_id, v.vote, v.quick_feedback, v.note, v.created_at, v.updated_at,
			tu.id as tu_id, tu.message_id as tu_message_id, tu.tool_name, tu.tool_arguments, tu.tool_result, tu.status,
			tu.error_message, tu.sequence_number, tu.completed_at, tu.created_at as tu_created_at, tu.updated_at as tu_updated_at,
			m.conversation_id,
			m_user.contents as user_message
		FROM alicia_votes v
		JOIN alicia_tool_uses tu ON v.target_id = tu.id AND tu.deleted_at IS NULL
		JOIN alicia_messages m ON tu.message_id = m.id AND m.deleted_at IS NULL
		LEFT JOIN alicia_messages m_user ON m.previous_id = m_user.id AND m_user.deleted_at IS NULL
		WHERE v.target_type = 'tool_use' AND v.deleted_at IS NULL
		ORDER BY v.created_at DESC
		LIMIT $1`

	rows, err := r.conn(ctx).Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*ports.VoteWithToolContext

	for rows.Next() {
		var v models.Vote
		var tu models.ToolUse
		var voteStr string
		var messageID, quickFeedback, note sql.NullString
		var tuArguments, tuResult []byte
		var tuErrorMessage sql.NullString
		var tuCompletedAt sql.NullTime
		var conversationID string
		var userMessage sql.NullString

		err := rows.Scan(
			&v.ID, &v.TargetType, &v.TargetID, &messageID, &voteStr, &quickFeedback, &note, &v.CreatedAt, &v.UpdatedAt,
			&tu.ID, &tu.MessageID, &tu.ToolName, &tuArguments, &tuResult, &tu.Status,
			&tuErrorMessage, &tu.SequenceNumber, &tuCompletedAt, &tu.CreatedAt, &tu.UpdatedAt,
			&conversationID,
			&userMessage,
		)
		if err != nil {
			return nil, err
		}

		// Map vote fields
		if voteStr == "up" {
			v.Value = models.VoteValueUp
		} else if voteStr == "down" {
			v.Value = models.VoteValueDown
		}
		v.MessageID = getString(messageID)
		v.QuickFeedback = getString(quickFeedback)
		v.Note = getString(note)

		// Unmarshal tool use JSON fields
		if len(tuArguments) > 0 {
			if err := json.Unmarshal(tuArguments, &tu.Arguments); err != nil {
				tu.Arguments = make(map[string]any)
			}
		} else {
			tu.Arguments = make(map[string]any)
		}

		if len(tuResult) > 0 {
			var res any
			if err := json.Unmarshal(tuResult, &res); err == nil {
				tu.Result = res
			}
		}

		tu.ErrorMessage = getString(tuErrorMessage)
		tu.CompletedAt = getTimePtr(tuCompletedAt)

		result := &ports.VoteWithToolContext{
			Vote:           &v,
			ToolUse:        &tu,
			UserMessage:    getString(userMessage),
			ConversationID: conversationID,
			AvailableTools: []*models.Tool{}, // Populated separately if needed
		}

		results = append(results, result)
	}

	return results, rows.Err()
}

func (r *VoteRepository) GetMemoryVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithMemoryContext, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT
			v.id, v.target_type, v.target_id, v.message_id, v.vote, v.quick_feedback, v.note, v.created_at, v.updated_at,
			mem.id as mem_id, mem.content, mem.importance, mem.confidence, mem.user_rating, mem.created_by,
			mem.source_type, mem.source_info, mem.tags, mem.pinned, mem.archived,
			mem.created_at as mem_created_at, mem.updated_at as mem_updated_at,
			mu.id as mu_id, mu.conversation_id, mu.message_id as mu_message_id, mu.memory_id,
			mu.query_prompt, mu.query_prompt_meta, mu.similarity_score, mu.meta, mu.position_in_results,
			mu.created_at as mu_created_at, mu.updated_at as mu_updated_at,
			m.contents as user_message
		FROM alicia_votes v
		JOIN alicia_memory mem ON v.target_id = mem.id AND mem.deleted_at IS NULL
		LEFT JOIN alicia_memory_used mu ON mu.memory_id = mem.id
			AND (v.message_id = mu.message_id OR v.message_id IS NULL)
			AND mu.deleted_at IS NULL
		LEFT JOIN alicia_messages m ON mu.message_id = m.id AND m.deleted_at IS NULL
		WHERE v.target_type = 'memory' AND v.deleted_at IS NULL
		ORDER BY v.created_at DESC
		LIMIT $1`

	rows, err := r.conn(ctx).Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*ports.VoteWithMemoryContext

	for rows.Next() {
		var v models.Vote
		var mem models.Memory
		var mu models.MemoryUsage
		var voteStr string
		var messageID, quickFeedback, note sql.NullString
		var userRating sql.NullInt32
		var createdBy, sourceType sql.NullString
		var sourceInfo, tags []byte
		var muID, muConversationID, muMessageID, muMemoryID sql.NullString
		var queryPrompt sql.NullString
		var queryPromptMeta, muMeta []byte
		var similarityScore sql.NullFloat64
		var positionInResults sql.NullInt32
		var muCreatedAt, muUpdatedAt sql.NullTime
		var userMessage sql.NullString

		err := rows.Scan(
			&v.ID, &v.TargetType, &v.TargetID, &messageID, &voteStr, &quickFeedback, &note, &v.CreatedAt, &v.UpdatedAt,
			&mem.ID, &mem.Content, &mem.Importance, &mem.Confidence, &userRating, &createdBy,
			&sourceType, &sourceInfo, &tags, &mem.Pinned, &mem.Archived,
			&mem.CreatedAt, &mem.UpdatedAt,
			&muID, &muConversationID, &muMessageID, &muMemoryID,
			&queryPrompt, &queryPromptMeta, &similarityScore, &muMeta, &positionInResults,
			&muCreatedAt, &muUpdatedAt,
			&userMessage,
		)
		if err != nil {
			return nil, err
		}

		// Map vote fields
		if voteStr == "up" {
			v.Value = models.VoteValueUp
		} else if voteStr == "down" {
			v.Value = models.VoteValueDown
		}
		v.MessageID = getString(messageID)
		v.QuickFeedback = getString(quickFeedback)
		v.Note = getString(note)

		// Map memory fields
		if userRating.Valid {
			rating := int(userRating.Int32)
			mem.UserRating = &rating
		}
		mem.CreatedBy = getString(createdBy)
		mem.SourceType = getString(sourceType)

		if len(sourceInfo) > 0 {
			var si models.SourceInfo
			if err := json.Unmarshal(sourceInfo, &si); err == nil {
				mem.SourceInfo = &si
			}
		}

		if len(tags) > 0 {
			if err := json.Unmarshal(tags, &mem.Tags); err != nil {
				mem.Tags = []string{}
			}
		} else {
			mem.Tags = []string{}
		}

		// Map memory usage fields (if exists)
		var memoryUsage *models.MemoryUsage
		if muID.Valid {
			mu.ID = getString(muID)
			mu.ConversationID = getString(muConversationID)
			mu.MessageID = getString(muMessageID)
			mu.MemoryID = getString(muMemoryID)
			mu.QueryPrompt = getString(queryPrompt)

			if len(queryPromptMeta) > 0 {
				if err := json.Unmarshal(queryPromptMeta, &mu.QueryPromptMeta); err != nil {
					mu.QueryPromptMeta = make(map[string]any)
				}
			} else {
				mu.QueryPromptMeta = make(map[string]any)
			}

			if similarityScore.Valid {
				mu.SimilarityScore = float32(similarityScore.Float64)
			}

			if len(muMeta) > 0 {
				if err := json.Unmarshal(muMeta, &mu.Meta); err != nil {
					mu.Meta = make(map[string]any)
				}
			} else {
				mu.Meta = make(map[string]any)
			}

			if positionInResults.Valid {
				mu.PositionInResults = int(positionInResults.Int32)
			}

			if muCreatedAt.Valid {
				mu.CreatedAt = muCreatedAt.Time
			}
			if muUpdatedAt.Valid {
				mu.UpdatedAt = muUpdatedAt.Time
			}

			memoryUsage = &mu
		}

		result := &ports.VoteWithMemoryContext{
			Vote:              &v,
			Memory:            &mem,
			MemoryUsage:       memoryUsage,
			UserMessage:       getString(userMessage),
			ConversationID:    getString(muConversationID),
			SimilarityScore:   float32(similarityScore.Float64),
			CandidateMemories: []*models.Memory{}, // Populated separately if needed
		}

		results = append(results, result)
	}

	return results, rows.Err()
}

func (r *VoteRepository) GetMemoryUsageVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithMemoryContext, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT
			v.id, v.target_type, v.target_id, v.message_id, v.vote, v.quick_feedback, v.note, v.created_at, v.updated_at,
			mu.id as mu_id, mu.conversation_id, mu.message_id as mu_message_id, mu.memory_id,
			mu.query_prompt, mu.query_prompt_meta, mu.similarity_score, mu.meta, mu.position_in_results,
			mu.created_at as mu_created_at, mu.updated_at as mu_updated_at,
			mem.id as mem_id, mem.content, mem.importance, mem.confidence, mem.user_rating, mem.created_by,
			mem.source_type, mem.source_info, mem.tags, mem.pinned, mem.archived,
			mem.created_at as mem_created_at, mem.updated_at as mem_updated_at,
			msg.contents as user_message
		FROM alicia_votes v
		JOIN alicia_memory_used mu ON v.target_id = mu.id AND mu.deleted_at IS NULL
		JOIN alicia_memory mem ON mu.memory_id = mem.id AND mem.deleted_at IS NULL
		JOIN alicia_messages msg ON mu.message_id = msg.id AND msg.deleted_at IS NULL
		WHERE v.target_type = 'memory_usage' AND v.deleted_at IS NULL
		ORDER BY v.created_at DESC
		LIMIT $1`

	rows, err := r.conn(ctx).Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*ports.VoteWithMemoryContext

	for rows.Next() {
		var v models.Vote
		var mu models.MemoryUsage
		var mem models.Memory
		var voteStr string
		var messageID, quickFeedback, note sql.NullString
		var userRating sql.NullInt32
		var createdBy, sourceType sql.NullString
		var sourceInfo, tags []byte
		var queryPrompt sql.NullString
		var queryPromptMeta, muMeta []byte
		var positionInResults sql.NullInt32
		var userMessage sql.NullString

		err := rows.Scan(
			&v.ID, &v.TargetType, &v.TargetID, &messageID, &voteStr, &quickFeedback, &note, &v.CreatedAt, &v.UpdatedAt,
			&mu.ID, &mu.ConversationID, &mu.MessageID, &mu.MemoryID,
			&queryPrompt, &queryPromptMeta, &mu.SimilarityScore, &muMeta, &positionInResults,
			&mu.CreatedAt, &mu.UpdatedAt,
			&mem.ID, &mem.Content, &mem.Importance, &mem.Confidence, &userRating, &createdBy,
			&sourceType, &sourceInfo, &tags, &mem.Pinned, &mem.Archived,
			&mem.CreatedAt, &mem.UpdatedAt,
			&userMessage,
		)
		if err != nil {
			return nil, err
		}

		// Map vote fields
		if voteStr == "up" {
			v.Value = models.VoteValueUp
		} else if voteStr == "down" {
			v.Value = models.VoteValueDown
		}
		v.MessageID = getString(messageID)
		v.QuickFeedback = getString(quickFeedback)
		v.Note = getString(note)

		// Map memory usage fields
		mu.QueryPrompt = getString(queryPrompt)

		if len(queryPromptMeta) > 0 {
			if err := json.Unmarshal(queryPromptMeta, &mu.QueryPromptMeta); err != nil {
				mu.QueryPromptMeta = make(map[string]any)
			}
		} else {
			mu.QueryPromptMeta = make(map[string]any)
		}

		if len(muMeta) > 0 {
			if err := json.Unmarshal(muMeta, &mu.Meta); err != nil {
				mu.Meta = make(map[string]any)
			}
		} else {
			mu.Meta = make(map[string]any)
		}

		if positionInResults.Valid {
			mu.PositionInResults = int(positionInResults.Int32)
		}

		// Map memory fields
		if userRating.Valid {
			rating := int(userRating.Int32)
			mem.UserRating = &rating
		}
		mem.CreatedBy = getString(createdBy)
		mem.SourceType = getString(sourceType)

		if len(sourceInfo) > 0 {
			var si models.SourceInfo
			if err := json.Unmarshal(sourceInfo, &si); err == nil {
				mem.SourceInfo = &si
			}
		}

		if len(tags) > 0 {
			if err := json.Unmarshal(tags, &mem.Tags); err != nil {
				mem.Tags = []string{}
			}
		} else {
			mem.Tags = []string{}
		}

		result := &ports.VoteWithMemoryContext{
			Vote:              &v,
			Memory:            &mem,
			MemoryUsage:       &mu,
			UserMessage:       getString(userMessage),
			ConversationID:    mu.ConversationID,
			SimilarityScore:   mu.SimilarityScore,
			CandidateMemories: []*models.Memory{}, // Populated separately if needed
		}

		results = append(results, result)
	}

	return results, rows.Err()
}

func (r *VoteRepository) GetMemoryExtractionVotesWithContext(ctx context.Context, limit int) ([]*ports.VoteWithExtractionContext, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT
			v.id, v.target_type, v.target_id, v.message_id, v.vote, v.quick_feedback, v.note, v.created_at, v.updated_at,
			mem.id as mem_id, mem.content, mem.importance, mem.confidence, mem.user_rating, mem.created_by,
			mem.source_type, mem.source_info, mem.tags, mem.pinned, mem.archived,
			mem.created_at as mem_created_at, mem.updated_at as mem_updated_at,
			msg.id as msg_id, msg.conversation_id, msg.sequence_number, msg.previous_id, msg.role, msg.contents,
			msg.local_id, msg.server_id, msg.sync_status, msg.synced_at, msg.completion_status,
			msg.created_at as msg_created_at, msg.updated_at as msg_updated_at
		FROM alicia_votes v
		JOIN alicia_memory mem ON v.target_id = mem.id AND mem.deleted_at IS NULL
		JOIN alicia_messages msg ON v.message_id = msg.id AND msg.deleted_at IS NULL
		WHERE v.target_type = 'memory_extraction' AND v.deleted_at IS NULL
		ORDER BY v.created_at DESC
		LIMIT $1`

	rows, err := r.conn(ctx).Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*ports.VoteWithExtractionContext

	for rows.Next() {
		var v models.Vote
		var mem models.Memory
		var msg models.Message
		var voteStr string
		var messageID, quickFeedback, note sql.NullString
		var userRating sql.NullInt32
		var createdBy, sourceType sql.NullString
		var sourceInfo, tags []byte
		var previousID, localID, serverID, syncStatus sql.NullString
		var syncedAt sql.NullTime
		var completionStatus sql.NullString

		err := rows.Scan(
			&v.ID, &v.TargetType, &v.TargetID, &messageID, &voteStr, &quickFeedback, &note, &v.CreatedAt, &v.UpdatedAt,
			&mem.ID, &mem.Content, &mem.Importance, &mem.Confidence, &userRating, &createdBy,
			&sourceType, &sourceInfo, &tags, &mem.Pinned, &mem.Archived,
			&mem.CreatedAt, &mem.UpdatedAt,
			&msg.ID, &msg.ConversationID, &msg.SequenceNumber, &previousID, &msg.Role, &msg.Contents,
			&localID, &serverID, &syncStatus, &syncedAt, &completionStatus,
			&msg.CreatedAt, &msg.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Map vote fields
		if voteStr == "up" {
			v.Value = models.VoteValueUp
		} else if voteStr == "down" {
			v.Value = models.VoteValueDown
		}
		v.MessageID = getString(messageID)
		v.QuickFeedback = getString(quickFeedback)
		v.Note = getString(note)

		// Map memory fields
		if userRating.Valid {
			rating := int(userRating.Int32)
			mem.UserRating = &rating
		}
		mem.CreatedBy = getString(createdBy)
		mem.SourceType = getString(sourceType)

		if len(sourceInfo) > 0 {
			var si models.SourceInfo
			if err := json.Unmarshal(sourceInfo, &si); err == nil {
				mem.SourceInfo = &si
			}
		}

		if len(tags) > 0 {
			if err := json.Unmarshal(tags, &mem.Tags); err != nil {
				mem.Tags = []string{}
			}
		} else {
			mem.Tags = []string{}
		}

		// Map message fields
		msg.PreviousID = getString(previousID)
		msg.LocalID = getString(localID)
		msg.ServerID = getString(serverID)
		if syncStatus.Valid {
			msg.SyncStatus = models.SyncStatus(syncStatus.String)
		}
		msg.SyncedAt = getTimePtr(syncedAt)
		if completionStatus.Valid {
			msg.CompletionStatus = models.CompletionStatus(completionStatus.String)
		}

		result := &ports.VoteWithExtractionContext{
			Vote:           &v,
			Memory:         &mem,
			SourceMessage:  &msg,
			ConversationID: msg.ConversationID,
		}

		results = append(results, result)
	}

	return results, rows.Err()
}
