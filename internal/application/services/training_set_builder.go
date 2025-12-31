package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/internal/prompt"
	"github.com/longregen/alicia/internal/prompt/baselines"
)

// TrainingSetBuilderConfig holds configuration for the training set builder
type TrainingSetBuilderConfig struct {
	MinVotesForReal   int     // Fallback to synthetic below this threshold
	MaxExamplesPerSet int     // Max examples per dataset
	TrainValSplit     float64 // Train/validation split ratio
}

// DefaultTrainingSetBuilderConfig returns the default configuration
func DefaultTrainingSetBuilderConfig() TrainingSetBuilderConfig {
	return TrainingSetBuilderConfig{
		MinVotesForReal:   15,
		MaxExamplesPerSet: 50,
		TrainValSplit:     0.8,
	}
}

// TrainingSetBuilderService builds GEPA training sets from user votes
type TrainingSetBuilderService struct {
	voteRepo     ports.VoteRepository
	trainingRepo ports.TrainingExampleRepository
	toolRepo     ports.ToolRepository
	memoryRepo   ports.MemoryRepository
	idGenerator  ports.IDGenerator
	config       TrainingSetBuilderConfig
}

// NewTrainingSetBuilderService creates a new training set builder service
func NewTrainingSetBuilderService(
	voteRepo ports.VoteRepository,
	trainingRepo ports.TrainingExampleRepository,
	toolRepo ports.ToolRepository,
	memoryRepo ports.MemoryRepository,
	idGenerator ports.IDGenerator,
	config TrainingSetBuilderConfig,
) *TrainingSetBuilderService {
	return &TrainingSetBuilderService{
		voteRepo:     voteRepo,
		trainingRepo: trainingRepo,
		toolRepo:     toolRepo,
		memoryRepo:   memoryRepo,
		idGenerator:  idGenerator,
		config:       config,
	}
}

// GetOrBuildToolSelectionDataset returns training/validation sets for tool selection GEPA
func (s *TrainingSetBuilderService) GetOrBuildToolSelectionDataset(ctx context.Context) (train, val []prompt.Example, err error) {
	count, err := s.voteRepo.CountByTargetType(ctx, models.VoteTargetToolUse)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count tool use votes: %w", err)
	}

	if count < s.config.MinVotesForReal {
		// Use synthetic + any available vote examples
		syntheticTrain, syntheticVal := baselines.SyntheticToolSelectionDataset()
		if count > 0 {
			voteExamples, err := s.buildToolSelectionFromVotes(ctx, count)
			if err == nil && len(voteExamples) > 0 {
				syntheticTrain = append(syntheticTrain, voteExamples...)
			}
		}
		return syntheticTrain, syntheticVal, nil
	}

	// Build from votes
	examples, err := s.buildToolSelectionFromVotes(ctx, s.config.MaxExamplesPerSet)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build tool selection from votes: %w", err)
	}

	// Split into train/val
	train, val = s.splitTrainVal(examples)
	return train, val, nil
}

// GetOrBuildMemorySelectionDataset returns training/validation sets for memory selection GEPA
func (s *TrainingSetBuilderService) GetOrBuildMemorySelectionDataset(ctx context.Context) (train, val []prompt.Example, err error) {
	count, err := s.voteRepo.CountByTargetType(ctx, models.VoteTargetMemoryUsage)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count memory usage votes: %w", err)
	}

	if count < s.config.MinVotesForReal {
		// Use synthetic datasets - for now return empty as we don't have synthetic memory selection yet
		// TODO: implement baselines.SyntheticMemorySelectionDataset()
		return []prompt.Example{}, []prompt.Example{}, nil
	}

	// Build from votes
	examples, err := s.buildMemorySelectionFromVotes(ctx, s.config.MaxExamplesPerSet)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build memory selection from votes: %w", err)
	}

	// Split into train/val
	train, val = s.splitTrainVal(examples)
	return train, val, nil
}

// GetOrBuildMemoryExtractionDataset returns training/validation sets for memory extraction GEPA
func (s *TrainingSetBuilderService) GetOrBuildMemoryExtractionDataset(ctx context.Context) (train, val []prompt.Example, err error) {
	count, err := s.voteRepo.CountByTargetType(ctx, models.VoteTargetMemoryExtraction)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count memory extraction votes: %w", err)
	}

	if count < s.config.MinVotesForReal {
		// Use synthetic datasets - for now return empty as we don't have synthetic memory extraction yet
		// TODO: implement baselines.SyntheticMemoryExtractionDataset()
		return []prompt.Example{}, []prompt.Example{}, nil
	}

	// Build from votes
	examples, err := s.buildMemoryExtractionFromVotes(ctx, s.config.MaxExamplesPerSet)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build memory extraction from votes: %w", err)
	}

	// Split into train/val
	train, val = s.splitTrainVal(examples)
	return train, val, nil
}

// buildToolSelectionFromVotes transforms tool_use votes into GEPA examples
func (s *TrainingSetBuilderService) buildToolSelectionFromVotes(ctx context.Context, limit int) ([]prompt.Example, error) {
	votesWithContext, err := s.voteRepo.GetToolUseVotesWithContext(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool use votes with context: %w", err)
	}

	// Get available tools for context
	tools, err := s.toolRepo.ListEnabled(ctx)
	if err != nil {
		// Continue without tools rather than fail
		tools = []*models.Tool{}
	}

	var examples []prompt.Example
	for _, vc := range votesWithContext {
		example := s.toolUseVoteToExample(vc, tools)
		examples = append(examples, example)
	}

	return examples, nil
}

// buildMemorySelectionFromVotes transforms memory_usage votes into GEPA examples
func (s *TrainingSetBuilderService) buildMemorySelectionFromVotes(ctx context.Context, limit int) ([]prompt.Example, error) {
	votesWithContext, err := s.voteRepo.GetMemoryUsageVotesWithContext(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get memory usage votes with context: %w", err)
	}

	var examples []prompt.Example
	for _, vc := range votesWithContext {
		example := s.memoryVoteToExample(vc)
		examples = append(examples, example)
	}

	return examples, nil
}

// toolUseVoteToExample converts a vote with tool context to a GEPA example
func (s *TrainingSetBuilderService) toolUseVoteToExample(vc *ports.VoteWithToolContext, tools []*models.Tool) prompt.Example {
	isPositive := vc.Vote.Value == models.VoteValueUp

	// Build inputs
	inputs := map[string]any{
		"user_message":    vc.UserMessage,
		"context":         "", // could fetch conversation history
		"available_tools": s.toolsToJSON(tools),
	}

	// Build outputs - this is what the model produced
	outputs := map[string]any{
		"selected_tool": vc.ToolUse.ToolName,
		"arguments":     vc.ToolUse.Arguments,
		"reasoning":     "", // Could extract from tool use if we store it
	}

	// Create example with metadata for vote-aware scoring
	example := prompt.Example{
		Inputs:  inputs,
		Outputs: outputs,
	}

	// Store vote metadata for scoring - note: prompt.Example doesn't have SetMetadata
	// so we'll embed this in the outputs for the metric to use
	if !isPositive {
		// For downvotes, add diagnostic feedback to outputs
		feedback := BuildDiagnosticFeedback(
			vc.Vote.QuickFeedback,
			vc.ToolUse.ToolName,
			vc.ToolUse.Arguments,
		)
		outputs["_vote_feedback"] = feedback
		outputs["_vote_value"] = vc.Vote.Value
		outputs["_quick_feedback"] = vc.Vote.QuickFeedback
	}

	return example
}

// memoryVoteToExample converts a memory vote to a GEPA example
func (s *TrainingSetBuilderService) memoryVoteToExample(vc *ports.VoteWithMemoryContext) prompt.Example {
	isPositive := vc.Vote.Value == models.VoteValueUp

	// Build candidate memories list
	candidateMemories := make([]map[string]any, len(vc.CandidateMemories))
	for i, mem := range vc.CandidateMemories {
		candidateMemories[i] = map[string]any{
			"id":         mem.ID,
			"content":    mem.Content,
			"importance": mem.Importance,
		}
	}

	// Build inputs
	inputs := map[string]any{
		"user_message":       vc.UserMessage,
		"context":            "", // could fetch conversation history
		"candidate_memories": candidateMemories,
	}

	// Build outputs
	outputs := map[string]any{
		"selected_memory_id": vc.Memory.ID,
		"reasoning":          "", // Could add reasoning if we track it
	}

	// Add vote metadata for scoring
	if !isPositive {
		feedback := BuildDiagnosticFeedback(
			vc.Vote.QuickFeedback,
			"", // not applicable for memory
			nil,
		)
		outputs["_vote_feedback"] = feedback
		outputs["_vote_value"] = vc.Vote.Value
		outputs["_quick_feedback"] = vc.Vote.QuickFeedback
	}

	return prompt.Example{
		Inputs:  inputs,
		Outputs: outputs,
	}
}

// buildMemoryExtractionFromVotes transforms memory_extraction votes into GEPA examples
func (s *TrainingSetBuilderService) buildMemoryExtractionFromVotes(ctx context.Context, limit int) ([]prompt.Example, error) {
	votesWithContext, err := s.voteRepo.GetMemoryExtractionVotesWithContext(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get memory extraction votes with context: %w", err)
	}

	var examples []prompt.Example
	for _, vc := range votesWithContext {
		example := s.memoryExtractionVoteToExample(vc)
		examples = append(examples, example)
	}

	return examples, nil
}

// memoryExtractionVoteToExample converts a memory extraction vote to a GEPA example
func (s *TrainingSetBuilderService) memoryExtractionVoteToExample(vc *ports.VoteWithExtractionContext) prompt.Example {
	// Build inputs
	inputs := map[string]any{
		"source_message": vc.SourceMessage.Contents,
	}

	// Build outputs
	outputs := map[string]any{
		"extracted_memory": vc.Memory.Content,
		"importance":       vc.Memory.Importance,
		"_vote_value":      vc.Vote.Value,
		"_quick_feedback":  vc.Vote.QuickFeedback,
	}

	return prompt.Example{
		Inputs:  inputs,
		Outputs: outputs,
	}
}

// toolsToJSON serializes tools to JSON for GEPA examples
func (s *TrainingSetBuilderService) toolsToJSON(tools []*models.Tool) string {
	toolData := make([]map[string]any, len(tools))
	for i, t := range tools {
		toolData[i] = map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"parameters":  t.Schema,
		}
	}

	jsonBytes, err := json.Marshal(toolData)
	if err != nil {
		return "[]"
	}
	return string(jsonBytes)
}

// splitTrainVal splits examples into training and validation sets
func (s *TrainingSetBuilderService) splitTrainVal(examples []prompt.Example) (train, val []prompt.Example) {
	if len(examples) == 0 {
		return []prompt.Example{}, []prompt.Example{}
	}

	splitIdx := int(float64(len(examples)) * s.config.TrainValSplit)
	if splitIdx < 1 {
		splitIdx = 1
	}
	if splitIdx >= len(examples) {
		// All to train, empty val
		return examples, []prompt.Example{}
	}

	return examples[:splitIdx], examples[splitIdx:]
}

// GetTrainingStats returns statistics about available training data
func (s *TrainingSetBuilderService) GetTrainingStats(ctx context.Context) (*TrainingStats, error) {
	stats := &TrainingStats{}

	toolVotes, err := s.voteRepo.CountByTargetType(ctx, models.VoteTargetToolUse)
	if err == nil {
		stats.ToolSelectionVotes = toolVotes
	}

	memoryUsageVotes, err := s.voteRepo.CountByTargetType(ctx, models.VoteTargetMemoryUsage)
	if err == nil {
		stats.MemoryUsageVotes = memoryUsageVotes
	}

	memoryExtractionVotes, err := s.voteRepo.CountByTargetType(ctx, models.VoteTargetMemoryExtraction)
	if err == nil {
		stats.MemoryExtractionVotes = memoryExtractionVotes
	}

	toolExamples, err := s.trainingRepo.CountByTaskType(ctx, models.TaskTypeToolSelection)
	if err == nil {
		stats.ToolSelectionExamples = toolExamples
	}

	memorySelectionExamples, err := s.trainingRepo.CountByTaskType(ctx, models.TaskTypeMemorySelection)
	if err == nil {
		stats.MemorySelectionExamples = memorySelectionExamples
	}

	memoryExtractionExamples, err := s.trainingRepo.CountByTaskType(ctx, models.TaskTypeMemoryExtraction)
	if err == nil {
		stats.MemoryExtractionExamples = memoryExtractionExamples
	}

	stats.MinVotesRequired = s.config.MinVotesForReal

	return stats, nil
}

// TrainingStats contains statistics about training data availability
type TrainingStats struct {
	ToolSelectionVotes       int `json:"tool_selection_votes"`
	MemoryUsageVotes         int `json:"memory_usage_votes"`
	MemoryExtractionVotes    int `json:"memory_extraction_votes"`
	ToolSelectionExamples    int `json:"tool_selection_examples"`
	MemorySelectionExamples  int `json:"memory_selection_examples"`
	MemoryExtractionExamples int `json:"memory_extraction_examples"`
	MinVotesRequired         int `json:"min_votes_required"`
}

// BuildDiagnosticFeedback generates rich diagnostic feedback from quick_feedback for GEPA reflection
func BuildDiagnosticFeedback(quickFeedback string, toolName string, args map[string]any) string {
	switch quickFeedback {
	case "wrong_tool":
		return fmt.Sprintf("The selected tool '%s' was incorrect for this query. Consider what the user actually needed.", toolName)
	case "unnecessary":
		return "A tool was used when none was needed. The query could have been answered directly without tool use."
	case "wrong_params":
		return fmt.Sprintf("The tool '%s' was correct but the arguments %v were wrong. Review how to extract parameters from user intent.", toolName, args)
	case "missing_context":
		return "The tool selection lacked important context from the conversation. Consider the full conversation history."
	case "wrong_context":
		return "This memory was retrieved but wasn't relevant to the user's actual intent."
	case "too_generic":
		return "This memory was too generic to be useful. More specific memories should be prioritized."
	case "outdated":
		return "This memory contains outdated information. Consider recency when selecting memories."
	case "incorrect":
		return "This memory contains incorrect information that should not have been used."
	default:
		return "The output was marked as incorrect by the user."
	}
}
