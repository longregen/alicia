package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/longregen/alicia/internal/adapters/http/middleware"
	"github.com/longregen/alicia/internal/application/usecases"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type VoteHandler struct {
	voteRepo           ports.VoteRepository
	idGenerator        ports.IDGenerator
	memorizeFromUpvote *usecases.MemorizeFromUpvote
}

func NewVoteHandler(voteRepo ports.VoteRepository, idGenerator ports.IDGenerator, memorizeFromUpvote *usecases.MemorizeFromUpvote) *VoteHandler {
	return &VoteHandler{
		voteRepo:           voteRepo,
		idGenerator:        idGenerator,
		memorizeFromUpvote: memorizeFromUpvote,
	}
}

// triggerMemorizeFromUpvote asynchronously triggers memory extraction from upvoted content
func (h *VoteHandler) triggerMemorizeFromUpvote(ctx context.Context, targetType, targetID string, vote int) {
	if h.memorizeFromUpvote == nil {
		return
	}

	// Only trigger for upvotes on messages or conversations
	if vote != 1 {
		return
	}
	if targetType != "message" && targetType != "conversation" {
		return
	}

	// Run asynchronously with a detached context and timeout
	go func() {
		memCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()

		_, err := h.memorizeFromUpvote.Execute(memCtx, &usecases.MemorizeFromUpvoteInput{
			TargetType: targetType,
			TargetID:   targetID,
			Vote:       vote,
			MinUpvotes: 1,
		})
		if err != nil {
			log.Printf("warning: failed to extract memories from upvoted %s %s: %v\n", targetType, targetID, err)
		}
	}()
}

// VoteRequest represents a vote submission
type VoteRequest struct {
	Vote          string  `json:"vote"`           // "up", "down", "critical"
	QuickFeedback *string `json:"quick_feedback"` // Optional quick feedback
}

// VoteResponse represents aggregate vote counts
type VoteResponse struct {
	TargetID   string         `json:"target_id"`
	TargetType string         `json:"target_type"`
	Upvotes    int            `json:"upvotes"`
	Downvotes  int            `json:"downvotes"`
	UserVote   *string        `json:"user_vote"` // Current user's vote
	Special    map[string]int `json:"special,omitempty"`
}

// QuickFeedbackRequest represents quick feedback submission
type QuickFeedbackRequest struct {
	Feedback string `json:"feedback"` // "wrong_tool", "wrong_params", "unnecessary", etc.
}

// IrrelevanceReasonRequest represents memory irrelevance reason
type IrrelevanceReasonRequest struct {
	Reason string `json:"reason"` // "outdated", "wrong_context", "too_generic", "incorrect"
}

// ReasoningIssueRequest represents reasoning step issue
type ReasoningIssueRequest struct {
	Issue string `json:"issue"` // "incorrect_assumption", "missed_consideration", etc.
}

// --- Message Voting ---

// VoteOnMessage handles POST /api/v1/messages/{id}/vote
func (h *VoteHandler) VoteOnMessage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	messageID, ok := validateURLParam(r, w, "id", "Message ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[VoteRequest](r, w)
	if !ok {
		return
	}

	// Validate vote value
	if req.Vote != "up" && req.Vote != "down" {
		respondError(w, "validation_error", "Vote must be 'up' or 'down'", http.StatusBadRequest)
		return
	}

	// Convert vote string to value
	voteValue := models.VoteValueUp
	if req.Vote == "down" {
		voteValue = models.VoteValueDown
	}

	// Create vote
	vote := models.NewVote(
		h.idGenerator.GenerateVoteID(),
		models.VoteTargetMessage,
		messageID,
		messageID,
		voteValue,
	)

	if err := h.voteRepo.Create(r.Context(), vote); err != nil {
		respondError(w, "database_error", "Failed to create vote", http.StatusInternalServerError)
		return
	}

	// Trigger memory extraction for upvotes on messages (async, non-blocking)
	if voteValue == models.VoteValueUp {
		h.triggerMemorizeFromUpvote(r.Context(), "message", messageID, int(voteValue))
	}

	// Get aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetMessage, messageID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   messageID,
		TargetType: "message",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   &req.Vote,
	}

	respondJSON(w, response, http.StatusOK)
}

// RemoveMessageVote handles DELETE /api/v1/messages/{id}/vote
func (h *VoteHandler) RemoveMessageVote(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	messageID, ok := validateURLParam(r, w, "id", "Message ID")
	if !ok {
		return
	}

	// Delete vote
	if err := h.voteRepo.Delete(r.Context(), models.VoteTargetMessage, messageID); err != nil {
		respondError(w, "database_error", "Failed to delete vote", http.StatusInternalServerError)
		return
	}

	// Get updated aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetMessage, messageID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   messageID,
		TargetType: "message",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   nil,
	}

	respondJSON(w, response, http.StatusOK)
}

// GetMessageVotes handles GET /api/v1/messages/{id}/votes
func (h *VoteHandler) GetMessageVotes(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	messageID, ok := validateURLParam(r, w, "id", "Message ID")
	if !ok {
		return
	}

	// Get aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetMessage, messageID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   messageID,
		TargetType: "message",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   nil,
	}

	respondJSON(w, response, http.StatusOK)
}

// --- Tool Use Voting ---

// VoteOnToolUse handles POST /api/v1/tool-uses/{id}/vote
func (h *VoteHandler) VoteOnToolUse(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	toolUseID, ok := validateURLParam(r, w, "id", "Tool use ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[VoteRequest](r, w)
	if !ok {
		return
	}

	if req.Vote != "up" && req.Vote != "down" {
		respondError(w, "validation_error", "Vote must be 'up' or 'down'", http.StatusBadRequest)
		return
	}

	// Convert vote string to value
	voteValue := models.VoteValueUp
	if req.Vote == "down" {
		voteValue = models.VoteValueDown
	}

	// Create vote (for tool_use, we use empty string for messageID since it's not message-specific)
	vote := models.NewVote(
		h.idGenerator.GenerateVoteID(),
		models.VoteTargetToolUse,
		toolUseID,
		"",
		voteValue,
	)

	if err := h.voteRepo.Create(r.Context(), vote); err != nil {
		respondError(w, "database_error", "Failed to create vote", http.StatusInternalServerError)
		return
	}

	// Get aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetToolUse, toolUseID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   toolUseID,
		TargetType: "tool_use",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   &req.Vote,
	}

	respondJSON(w, response, http.StatusOK)
}

// GetToolUseVotes handles GET /api/v1/tool-uses/{id}/votes
func (h *VoteHandler) GetToolUseVotes(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	toolUseID, ok := validateURLParam(r, w, "id", "Tool use ID")
	if !ok {
		return
	}

	// Get aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetToolUse, toolUseID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   toolUseID,
		TargetType: "tool_use",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   nil,
	}

	respondJSON(w, response, http.StatusOK)
}

// RemoveToolUseVote handles DELETE /api/v1/tool-uses/{id}/vote
func (h *VoteHandler) RemoveToolUseVote(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	toolUseID, ok := validateURLParam(r, w, "id", "Tool use ID")
	if !ok {
		return
	}

	// Delete vote
	if err := h.voteRepo.Delete(r.Context(), models.VoteTargetToolUse, toolUseID); err != nil {
		respondError(w, "database_error", "Failed to delete vote", http.StatusInternalServerError)
		return
	}

	// Get updated aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetToolUse, toolUseID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   toolUseID,
		TargetType: "tool_use",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   nil,
	}

	respondJSON(w, response, http.StatusOK)
}

// ToolUseQuickFeedback handles POST /api/v1/tool-uses/{id}/quick-feedback
func (h *VoteHandler) ToolUseQuickFeedback(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	toolUseID, ok := validateURLParam(r, w, "id", "Tool use ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[QuickFeedbackRequest](r, w)
	if !ok {
		return
	}

	// Validate feedback type
	validFeedback := map[string]bool{
		"wrong_tool":      true,
		"wrong_params":    true,
		"unnecessary":     true,
		"missing_context": true,
		"perfect":         true,
	}

	if !validFeedback[req.Feedback] {
		respondError(w, "validation_error", "Invalid feedback type", http.StatusBadRequest)
		return
	}

	// Determine vote value: perfect is positive, others are negative
	voteValue := models.VoteValueDown
	if req.Feedback == "perfect" {
		voteValue = models.VoteValueUp
	}

	// Create vote with quick feedback
	vote := models.NewVoteWithFeedback(
		h.idGenerator.GenerateVoteID(),
		models.VoteTargetToolUse,
		toolUseID,
		"", // No parent message ID for quick feedback
		voteValue,
		req.Feedback,
		"",
	)

	if err := h.voteRepo.Create(r.Context(), vote); err != nil {
		respondError(w, "database_error", "Failed to record quick feedback", http.StatusInternalServerError)
		return
	}

	// Get updated aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetToolUse, toolUseID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   toolUseID,
		TargetType: "tool_use",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   &req.Feedback,
	}

	respondJSON(w, response, http.StatusOK)
}

// --- Memory Voting ---

// VoteOnMemory handles POST /api/v1/memories/{id}/vote
func (h *VoteHandler) VoteOnMemory(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	memoryID, ok := validateURLParam(r, w, "id", "Memory ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[VoteRequest](r, w)
	if !ok {
		return
	}

	// Validate vote value (memories support "critical" as well)
	if req.Vote != "up" && req.Vote != "down" && req.Vote != "critical" {
		respondError(w, "validation_error", "Vote must be 'up', 'down', or 'critical'", http.StatusBadRequest)
		return
	}

	// Convert vote string to value (note: "critical" is treated as upvote for now)
	voteValue := models.VoteValueUp
	if req.Vote == "down" {
		voteValue = models.VoteValueDown
	}

	// Create vote
	vote := models.NewVote(
		h.idGenerator.GenerateVoteID(),
		models.VoteTargetMemory,
		memoryID,
		"",
		voteValue,
	)

	if err := h.voteRepo.Create(r.Context(), vote); err != nil {
		respondError(w, "database_error", "Failed to create vote", http.StatusInternalServerError)
		return
	}

	// Get aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetMemory, memoryID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   memoryID,
		TargetType: "memory",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   &req.Vote,
		Special: map[string]int{
			"critical": 0,
		},
	}

	respondJSON(w, response, http.StatusOK)
}

// GetMemoryVotes handles GET /api/v1/memories/{id}/votes
func (h *VoteHandler) GetMemoryVotes(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	memoryID, ok := validateURLParam(r, w, "id", "Memory ID")
	if !ok {
		return
	}

	// Get aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetMemory, memoryID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   memoryID,
		TargetType: "memory",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   nil,
		Special: map[string]int{
			"critical": 0,
		},
	}

	respondJSON(w, response, http.StatusOK)
}

// RemoveMemoryVote handles DELETE /api/v1/memories/{id}/vote
func (h *VoteHandler) RemoveMemoryVote(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	memoryID, ok := validateURLParam(r, w, "id", "Memory ID")
	if !ok {
		return
	}

	// Delete vote
	if err := h.voteRepo.Delete(r.Context(), models.VoteTargetMemory, memoryID); err != nil {
		respondError(w, "database_error", "Failed to delete vote", http.StatusInternalServerError)
		return
	}

	// Get updated aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetMemory, memoryID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   memoryID,
		TargetType: "memory",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   nil,
	}

	respondJSON(w, response, http.StatusOK)
}

// MemoryIrrelevanceReason handles POST /api/v1/memories/{id}/irrelevance-reason
func (h *VoteHandler) MemoryIrrelevanceReason(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	memoryID, ok := validateURLParam(r, w, "id", "Memory ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[IrrelevanceReasonRequest](r, w)
	if !ok {
		return
	}

	// Validate reason
	validReasons := map[string]bool{
		"outdated":      true,
		"wrong_context": true,
		"too_generic":   true,
		"incorrect":     true,
	}

	if !validReasons[req.Reason] {
		respondError(w, "validation_error", "Invalid irrelevance reason", http.StatusBadRequest)
		return
	}

	// Create vote with irrelevance reason as quick feedback
	vote := models.NewVoteWithFeedback(
		h.idGenerator.GenerateVoteID(),
		models.VoteTargetMemory,
		memoryID,
		"",                   // No parent message ID
		models.VoteValueDown, // Irrelevance is negative feedback
		req.Reason,
		"",
	)

	if err := h.voteRepo.Create(r.Context(), vote); err != nil {
		respondError(w, "database_error", "Failed to record irrelevance reason", http.StatusInternalServerError)
		return
	}

	// Get updated aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetMemory, memoryID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   memoryID,
		TargetType: "memory",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   &req.Reason,
	}

	respondJSON(w, response, http.StatusOK)
}

// --- Reasoning Voting ---

// VoteOnReasoning handles POST /api/v1/reasoning/{id}/vote
func (h *VoteHandler) VoteOnReasoning(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	reasoningID, ok := validateURLParam(r, w, "id", "Reasoning ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[VoteRequest](r, w)
	if !ok {
		return
	}

	if req.Vote != "up" && req.Vote != "down" {
		respondError(w, "validation_error", "Vote must be 'up' or 'down'", http.StatusBadRequest)
		return
	}

	// Convert vote string to value
	voteValue := models.VoteValueUp
	if req.Vote == "down" {
		voteValue = models.VoteValueDown
	}

	// Create vote
	vote := models.NewVote(
		h.idGenerator.GenerateVoteID(),
		models.VoteTargetReasoning,
		reasoningID,
		"",
		voteValue,
	)

	if err := h.voteRepo.Create(r.Context(), vote); err != nil {
		respondError(w, "database_error", "Failed to create vote", http.StatusInternalServerError)
		return
	}

	// Get aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetReasoning, reasoningID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   reasoningID,
		TargetType: "reasoning",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   &req.Vote,
	}

	respondJSON(w, response, http.StatusOK)
}

// GetReasoningVotes handles GET /api/v1/reasoning/{id}/votes
func (h *VoteHandler) GetReasoningVotes(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	reasoningID, ok := validateURLParam(r, w, "id", "Reasoning ID")
	if !ok {
		return
	}

	// Get aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetReasoning, reasoningID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   reasoningID,
		TargetType: "reasoning",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   nil,
	}

	respondJSON(w, response, http.StatusOK)
}

// RemoveReasoningVote handles DELETE /api/v1/reasoning/{id}/vote
func (h *VoteHandler) RemoveReasoningVote(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	reasoningID, ok := validateURLParam(r, w, "id", "Reasoning ID")
	if !ok {
		return
	}

	// Delete vote
	if err := h.voteRepo.Delete(r.Context(), models.VoteTargetReasoning, reasoningID); err != nil {
		respondError(w, "database_error", "Failed to delete vote", http.StatusInternalServerError)
		return
	}

	// Get updated aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetReasoning, reasoningID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   reasoningID,
		TargetType: "reasoning",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   nil,
	}

	respondJSON(w, response, http.StatusOK)
}

// ReasoningIssue handles POST /api/v1/reasoning/{id}/issue
func (h *VoteHandler) ReasoningIssue(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	reasoningID, ok := validateURLParam(r, w, "id", "Reasoning ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[ReasoningIssueRequest](r, w)
	if !ok {
		return
	}

	// Validate issue type
	validIssues := map[string]bool{
		"incorrect_assumption": true,
		"missed_consideration": true,
		"overcomplicated":      true,
		"wrong_direction":      true,
	}

	if !validIssues[req.Issue] {
		respondError(w, "validation_error", "Invalid reasoning issue type", http.StatusBadRequest)
		return
	}

	// Create vote with reasoning issue as quick feedback
	vote := models.NewVoteWithFeedback(
		h.idGenerator.GenerateVoteID(),
		models.VoteTargetReasoning,
		reasoningID,
		"",                   // No parent message ID
		models.VoteValueDown, // Reporting an issue is negative feedback
		req.Issue,
		"",
	)

	if err := h.voteRepo.Create(r.Context(), vote); err != nil {
		respondError(w, "database_error", "Failed to record reasoning issue", http.StatusInternalServerError)
		return
	}

	// Get updated aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetReasoning, reasoningID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   reasoningID,
		TargetType: "reasoning",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   &req.Issue,
	}

	respondJSON(w, response, http.StatusOK)
}

// --- Memory Usage (Selection) Voting ---

// VoteOnMemoryUsage handles POST /api/v1/memory-usages/{id}/vote
func (h *VoteHandler) VoteOnMemoryUsage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	memoryUsageID, ok := validateURLParam(r, w, "id", "Memory usage ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[VoteRequest](r, w)
	if !ok {
		return
	}

	if req.Vote != "up" && req.Vote != "down" {
		respondError(w, "validation_error", "Vote must be 'up' or 'down'", http.StatusBadRequest)
		return
	}

	// Convert vote string to value
	voteValue := models.VoteValueUp
	if req.Vote == "down" {
		voteValue = models.VoteValueDown
	}

	// Create vote
	vote := models.NewVote(
		h.idGenerator.GenerateVoteID(),
		models.VoteTargetMemoryUsage,
		memoryUsageID,
		"",
		voteValue,
	)

	if err := h.voteRepo.Create(r.Context(), vote); err != nil {
		respondError(w, "database_error", "Failed to create vote", http.StatusInternalServerError)
		return
	}

	// Get aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetMemoryUsage, memoryUsageID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   memoryUsageID,
		TargetType: "memory_usage",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   &req.Vote,
	}

	respondJSON(w, response, http.StatusOK)
}

// RemoveMemoryUsageVote handles DELETE /api/v1/memory-usages/{id}/vote
func (h *VoteHandler) RemoveMemoryUsageVote(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	memoryUsageID, ok := validateURLParam(r, w, "id", "Memory usage ID")
	if !ok {
		return
	}

	// Delete vote
	if err := h.voteRepo.Delete(r.Context(), models.VoteTargetMemoryUsage, memoryUsageID); err != nil {
		respondError(w, "database_error", "Failed to delete vote", http.StatusInternalServerError)
		return
	}

	// Get updated aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetMemoryUsage, memoryUsageID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   memoryUsageID,
		TargetType: "memory_usage",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   nil,
	}

	respondJSON(w, response, http.StatusOK)
}

// GetMemoryUsageVotes handles GET /api/v1/memory-usages/{id}/votes
func (h *VoteHandler) GetMemoryUsageVotes(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	memoryUsageID, ok := validateURLParam(r, w, "id", "Memory usage ID")
	if !ok {
		return
	}

	// Get aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetMemoryUsage, memoryUsageID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   memoryUsageID,
		TargetType: "memory_usage",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   nil,
	}

	respondJSON(w, response, http.StatusOK)
}

// MemoryUsageIrrelevanceReason handles POST /api/v1/memory-usages/{id}/irrelevance-reason
func (h *VoteHandler) MemoryUsageIrrelevanceReason(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	memoryUsageID, ok := validateURLParam(r, w, "id", "Memory usage ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[IrrelevanceReasonRequest](r, w)
	if !ok {
		return
	}

	// Validate reason
	validReasons := map[string]bool{
		"outdated":      true,
		"wrong_context": true,
		"too_general":   true,
		"too_specific":  true,
		"incorrect":     true,
	}

	if !validReasons[req.Reason] {
		respondError(w, "validation_error", "Invalid irrelevance reason", http.StatusBadRequest)
		return
	}

	// Create vote with irrelevance reason as quick feedback
	vote := models.NewVoteWithFeedback(
		h.idGenerator.GenerateVoteID(),
		models.VoteTargetMemoryUsage,
		memoryUsageID,
		"",                   // No parent message ID
		models.VoteValueDown, // Irrelevance is negative feedback
		req.Reason,
		"",
	)

	if err := h.voteRepo.Create(r.Context(), vote); err != nil {
		respondError(w, "database_error", "Failed to record irrelevance reason", http.StatusInternalServerError)
		return
	}

	// Get updated aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetMemoryUsage, memoryUsageID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   memoryUsageID,
		TargetType: "memory_usage",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   &req.Reason,
	}

	respondJSON(w, response, http.StatusOK)
}

// --- Memory Extraction Voting ---

// VoteOnMemoryExtraction handles POST /api/v1/messages/{messageId}/extracted-memories/{memoryId}/vote
func (h *VoteHandler) VoteOnMemoryExtraction(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	messageID, ok := validateURLParam(r, w, "messageId", "Message ID")
	if !ok {
		return
	}

	memoryID, ok := validateURLParam(r, w, "memoryId", "Memory ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[VoteRequest](r, w)
	if !ok {
		return
	}

	if req.Vote != "up" && req.Vote != "down" {
		respondError(w, "validation_error", "Vote must be 'up' or 'down'", http.StatusBadRequest)
		return
	}

	// Convert vote string to value
	voteValue := models.VoteValueUp
	if req.Vote == "down" {
		voteValue = models.VoteValueDown
	}

	// Create vote (use memoryID as targetID, messageID as parent)
	vote := models.NewVote(
		h.idGenerator.GenerateVoteID(),
		models.VoteTargetMemoryExtraction,
		memoryID,
		messageID,
		voteValue,
	)

	if err := h.voteRepo.Create(r.Context(), vote); err != nil {
		respondError(w, "database_error", "Failed to create vote", http.StatusInternalServerError)
		return
	}

	// Get aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetMemoryExtraction, memoryID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   memoryID,
		TargetType: "memory_extraction",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   &req.Vote,
	}

	respondJSON(w, response, http.StatusOK)
}

// RemoveMemoryExtractionVote handles DELETE /api/v1/messages/{messageId}/extracted-memories/{memoryId}/vote
func (h *VoteHandler) RemoveMemoryExtractionVote(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	_, ok := validateURLParam(r, w, "messageId", "Message ID")
	if !ok {
		return
	}

	memoryID, ok := validateURLParam(r, w, "memoryId", "Memory ID")
	if !ok {
		return
	}

	// Delete vote
	if err := h.voteRepo.Delete(r.Context(), models.VoteTargetMemoryExtraction, memoryID); err != nil {
		respondError(w, "database_error", "Failed to delete vote", http.StatusInternalServerError)
		return
	}

	// Get updated aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetMemoryExtraction, memoryID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   memoryID,
		TargetType: "memory_extraction",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   nil,
	}

	respondJSON(w, response, http.StatusOK)
}

// GetMemoryExtractionVotes handles GET /api/v1/messages/{messageId}/extracted-memories/{memoryId}/votes
func (h *VoteHandler) GetMemoryExtractionVotes(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	_, ok := validateURLParam(r, w, "messageId", "Message ID")
	if !ok {
		return
	}

	memoryID, ok := validateURLParam(r, w, "memoryId", "Memory ID")
	if !ok {
		return
	}

	// Get aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetMemoryExtraction, memoryID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   memoryID,
		TargetType: "memory_extraction",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   nil,
	}

	respondJSON(w, response, http.StatusOK)
}

// MemoryExtractionQualityFeedback handles POST /api/v1/messages/{messageId}/extracted-memories/{memoryId}/quality-feedback
func (h *VoteHandler) MemoryExtractionQualityFeedback(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	messageID, ok := validateURLParam(r, w, "messageId", "Message ID")
	if !ok {
		return
	}

	memoryID, ok := validateURLParam(r, w, "memoryId", "Memory ID")
	if !ok {
		return
	}

	req, ok := decodeJSON[QuickFeedbackRequest](r, w)
	if !ok {
		return
	}

	// Validate feedback type
	validFeedback := map[string]bool{
		"accurate":     true,
		"too_specific": true,
		"too_generic":  true,
		"incorrect":    true,
		"redundant":    true,
		"irrelevant":   true,
	}

	if !validFeedback[req.Feedback] {
		respondError(w, "validation_error", "Invalid quality feedback type", http.StatusBadRequest)
		return
	}

	// Determine vote value: accurate is positive, others are negative
	voteValue := models.VoteValueDown
	if req.Feedback == "accurate" {
		voteValue = models.VoteValueUp
	}

	// Create vote with quality feedback
	vote := models.NewVoteWithFeedback(
		h.idGenerator.GenerateVoteID(),
		models.VoteTargetMemoryExtraction,
		memoryID,
		messageID,
		voteValue,
		req.Feedback,
		"",
	)

	if err := h.voteRepo.Create(r.Context(), vote); err != nil {
		respondError(w, "database_error", "Failed to record quality feedback", http.StatusInternalServerError)
		return
	}

	// Get updated aggregates
	aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetMemoryExtraction, memoryID)
	if err != nil {
		respondError(w, "database_error", "Failed to get vote aggregates", http.StatusInternalServerError)
		return
	}

	response := &VoteResponse{
		TargetID:   memoryID,
		TargetType: "memory_extraction",
		Upvotes:    aggregates.Upvotes,
		Downvotes:  aggregates.Downvotes,
		UserVote:   &req.Feedback,
	}

	respondJSON(w, response, http.StatusOK)
}
