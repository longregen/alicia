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

func (h *VoteHandler) triggerMemorizeFromUpvote(ctx context.Context, targetType, targetID string, vote int) {
	if h.memorizeFromUpvote == nil {
		return
	}

	if vote != 1 {
		return
	}
	if targetType != "message" && targetType != "conversation" {
		return
	}

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

type VoteRequest struct {
	Vote          string  `json:"vote"`
	QuickFeedback *string `json:"quick_feedback"`
}

type VoteResponse struct {
	TargetID   string         `json:"target_id"`
	TargetType string         `json:"target_type"`
	Upvotes    int            `json:"upvotes"`
	Downvotes  int            `json:"downvotes"`
	UserVote   *string        `json:"user_vote"`
	Special    map[string]int `json:"special,omitempty"`
}

type QuickFeedbackRequest struct {
	Feedback string `json:"feedback"`
}

type IrrelevanceReasonRequest struct {
	Reason string `json:"reason"`
}

type ReasoningIssueRequest struct {
	Issue string `json:"issue"`
}

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

	if req.Vote != "up" && req.Vote != "down" {
		respondError(w, "validation_error", "Vote must be 'up' or 'down'", http.StatusBadRequest)
		return
	}

	voteValue := models.VoteValueUp
	if req.Vote == "down" {
		voteValue = models.VoteValueDown
	}

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

	if voteValue == models.VoteValueUp {
		h.triggerMemorizeFromUpvote(r.Context(), "message", messageID, int(voteValue))
	}

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

	if err := h.voteRepo.Delete(r.Context(), models.VoteTargetMessage, messageID); err != nil {
		respondError(w, "database_error", "Failed to delete vote", http.StatusInternalServerError)
		return
	}

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

type BatchVotesRequest struct {
	MessageIDs []string `json:"message_ids"`
}

type BatchVotesResponse struct {
	Votes map[string]*VoteResponse `json:"votes"`
}

func (h *VoteHandler) GetBatchMessageVotes(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, "auth_error", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	req, ok := decodeJSON[BatchVotesRequest](r, w)
	if !ok {
		return
	}

	if len(req.MessageIDs) == 0 {
		respondJSON(w, &BatchVotesResponse{Votes: make(map[string]*VoteResponse)}, http.StatusOK)
		return
	}

	if len(req.MessageIDs) > 100 {
		respondError(w, "validation_error", "Maximum 100 message IDs per batch", http.StatusBadRequest)
		return
	}

	votes := make(map[string]*VoteResponse)
	for _, messageID := range req.MessageIDs {
		aggregates, err := h.voteRepo.GetAggregates(r.Context(), models.VoteTargetMessage, messageID)
		if err != nil {
			// Skip messages that fail, don't fail the whole batch
			continue
		}
		votes[messageID] = &VoteResponse{
			TargetID:   messageID,
			TargetType: "message",
			Upvotes:    aggregates.Upvotes,
			Downvotes:  aggregates.Downvotes,
			UserVote:   nil,
		}
	}

	respondJSON(w, &BatchVotesResponse{Votes: votes}, http.StatusOK)
}

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

	if err := h.voteRepo.Delete(r.Context(), models.VoteTargetToolUse, toolUseID); err != nil {
		respondError(w, "database_error", "Failed to delete vote", http.StatusInternalServerError)
		return
	}

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

	voteValue := models.VoteValueDown
	if req.Feedback == "perfect" {
		voteValue = models.VoteValueUp
	}

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

	if req.Vote != "up" && req.Vote != "down" && req.Vote != "critical" {
		respondError(w, "validation_error", "Vote must be 'up', 'down', or 'critical'", http.StatusBadRequest)
		return
	}

	voteValue := models.VoteValueUp
	if req.Vote == "down" {
		voteValue = models.VoteValueDown
	}

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

	if err := h.voteRepo.Delete(r.Context(), models.VoteTargetMemory, memoryID); err != nil {
		respondError(w, "database_error", "Failed to delete vote", http.StatusInternalServerError)
		return
	}

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

	voteValue := models.VoteValueUp
	if req.Vote == "down" {
		voteValue = models.VoteValueDown
	}

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

	if err := h.voteRepo.Delete(r.Context(), models.VoteTargetReasoning, reasoningID); err != nil {
		respondError(w, "database_error", "Failed to delete vote", http.StatusInternalServerError)
		return
	}

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

	voteValue := models.VoteValueUp
	if req.Vote == "down" {
		voteValue = models.VoteValueDown
	}

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

	if err := h.voteRepo.Delete(r.Context(), models.VoteTargetMemoryUsage, memoryUsageID); err != nil {
		respondError(w, "database_error", "Failed to delete vote", http.StatusInternalServerError)
		return
	}

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

	voteValue := models.VoteValueUp
	if req.Vote == "down" {
		voteValue = models.VoteValueDown
	}

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

	if err := h.voteRepo.Delete(r.Context(), models.VoteTargetMemoryExtraction, memoryID); err != nil {
		respondError(w, "database_error", "Failed to delete vote", http.StatusInternalServerError)
		return
	}

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

	voteValue := models.VoteValueDown
	if req.Feedback == "accurate" {
		voteValue = models.VoteValueUp
	}

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
