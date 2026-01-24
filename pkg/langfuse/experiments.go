package langfuse

import (
	"context"
)

// ExperimentConfig holds configuration for running an experiment.
type ExperimentConfig struct {
	DatasetName   string // Name of the dataset to run
	RunName       string // Name for this experiment run
	PromptVersion int    // Optional: specific prompt version to use
}

// ExperimentItem represents a single item to be evaluated in an experiment.
// It wraps the DatasetItem with easier access to the input/output fields.
type ExperimentItem struct {
	ID             string
	Input          GoldenExampleInput
	ExpectedOutput GoldenExampleOutput
	Metadata       map[string]any
}

// GetExperimentItems retrieves all items from a dataset formatted for experiment runs.
// This is a helper for manual experiment runs - it fetches items and converts them
// to the expected format for evaluation.
func GetExperimentItems(ctx context.Context, client *Client, datasetName string) ([]ExperimentItem, error) {
	items, err := client.GetDatasetItems(ctx, datasetName)
	if err != nil {
		return nil, err
	}

	results := make([]ExperimentItem, 0, len(items))
	for _, item := range items {
		expItem := ExperimentItem{
			ID:       item.ID,
			Metadata: item.Metadata,
		}

		// Parse input
		if inputMap, ok := item.Input.(map[string]any); ok {
			if query, ok := inputMap["user_query"].(string); ok {
				expItem.Input.UserQuery = query
			}
		}

		// Parse expected output
		if outputMap, ok := item.ExpectedOutput.(map[string]any); ok {
			if response, ok := outputMap["assistant_response"].(string); ok {
				expItem.ExpectedOutput.AssistantResponse = response
			}
		}

		results = append(results, expItem)
	}

	return results, nil
}

// RecordExperimentResult records the result of running a dataset item through the system.
// This creates a dataset run item that can be compared against the expected output.
func RecordExperimentResult(ctx context.Context, client *Client, config ExperimentConfig, datasetItemID string, actualOutput string, scores map[string]float64) error {
	output := GoldenExampleOutput{
		AssistantResponse: actualOutput,
	}

	return client.CreateDatasetRunItem(ctx, DatasetRunItemParams{
		DatasetItemID: datasetItemID,
		RunName:       config.RunName,
		Output:        output,
		Scores:        scores,
		Metadata: map[string]any{
			"prompt_version": config.PromptVersion,
		},
	})
}
