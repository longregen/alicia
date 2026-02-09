package langfuse

import (
	"context"
	"time"
)

// GoldenQADataset is the name of the dataset for high-quality Q&A examples.
const GoldenQADataset = "alicia-golden-qa"

// GoldenQADatasetDescription describes the purpose of the golden QA dataset.
const GoldenQADatasetDescription = "High-quality Q&A pairs collected from positive user feedback for evaluation and fine-tuning."

// EnsureGoldenDataset creates the golden Q&A dataset if it doesn't exist.
// This operation is idempotent - it won't fail if the dataset already exists.
func EnsureGoldenDataset(ctx context.Context, client *Client) error {
	err := client.CreateDataset(ctx, DatasetParams{
		Name:        GoldenQADataset,
		Description: GoldenQADatasetDescription,
		Metadata: map[string]any{
			"source":     "user_feedback",
			"type":       "qa_pairs",
			"created_by": "alicia-api",
		},
	})
	if err != nil {
		return err
	}
	client.log.Printf("langfuse: ensured golden QA dataset %q exists", GoldenQADataset)
	return nil
}

// GoldenExampleInput represents the input format for a golden Q&A example.
type GoldenExampleInput struct {
	UserQuery string `json:"user_query"`
}

// GoldenExampleOutput represents the expected output format for a golden Q&A example.
type GoldenExampleOutput struct {
	AssistantResponse string `json:"assistant_response"`
}

// AddGoldenExample adds a high-quality Q&A pair to the golden dataset.
// This should be called when a message receives positive feedback.
func AddGoldenExample(ctx context.Context, client *Client, userQuery, assistantResponse string, metadata map[string]any) error {
	input := GoldenExampleInput{
		UserQuery: userQuery,
	}

	expectedOutput := GoldenExampleOutput{
		AssistantResponse: assistantResponse,
	}

	// Merge default metadata with provided metadata
	fullMetadata := map[string]any{
		"source":       "positive_feedback",
		"collected_at": time.Now().UTC().Format(time.RFC3339),
	}
	for k, v := range metadata {
		fullMetadata[k] = v
	}

	err := client.CreateDatasetItem(ctx, DatasetItemParams{
		DatasetName:    GoldenQADataset,
		Input:          input,
		ExpectedOutput: expectedOutput,
		Metadata:       fullMetadata,
	})
	if err != nil {
		client.log.Printf("langfuse: failed to add golden example: %v", err)
		return err
	}

	client.log.Printf("langfuse: added golden example to dataset (conv=%v)", metadata["conversation_id"])
	return nil
}
