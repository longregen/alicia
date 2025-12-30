package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/longregen/alicia/internal/prompt"
)

func TestFeedbackHandler_SubmitFeedback(t *testing.T) {
	mockVoteRepo := &MockVoteRepository{}
	mockOptService := &MockOptimizationService{}

	handler := NewFeedbackHandler(mockVoteRepo, mockOptService)

	tests := []struct {
		name           string
		request        SubmitFeedbackRequest
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "upvote on message",
			request: SubmitFeedbackRequest{
				TargetType: "message",
				TargetID:   "msg_123",
				Vote:       "up",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]any
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}

				if resp["feedback_type"] != string(prompt.FeedbackGreatAnswer) {
					t.Errorf("expected feedback_type to be %s, got %v", prompt.FeedbackGreatAnswer, resp["feedback_type"])
				}

				if _, ok := resp["new_weights"]; !ok {
					t.Error("expected new_weights in response")
				}
			},
		},
		{
			name: "downvote on tool_use",
			request: SubmitFeedbackRequest{
				TargetType: "tool_use",
				TargetID:   "tool_456",
				Vote:       "down",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]any
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}

				if resp["feedback_type"] != string(prompt.FeedbackWrongTool) {
					t.Errorf("expected feedback_type to be %s, got %v", prompt.FeedbackWrongTool, resp["feedback_type"])
				}
			},
		},
		{
			name: "downvote with quick feedback",
			request: SubmitFeedbackRequest{
				TargetType:    "tool_use",
				TargetID:      "tool_789",
				Vote:          "down",
				QuickFeedback: "wrong_params",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]any
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}

				if resp["feedback_type"] != string(prompt.FeedbackWrongParams) {
					t.Errorf("expected feedback_type to be %s, got %v", prompt.FeedbackWrongParams, resp["feedback_type"])
				}
			},
		},
		{
			name: "missing required fields",
			request: SubmitFeedbackRequest{
				TargetType: "message",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock state
			mockOptService.weights = map[string]float64{
				"successRate":    0.25,
				"quality":        0.20,
				"efficiency":     0.15,
				"robustness":     0.15,
				"generalization": 0.10,
				"diversity":      0.10,
				"innovation":     0.05,
			}

			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req = setTestUserID(req, "test-user")

			rec := httptest.NewRecorder()
			handler.SubmitFeedback(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.checkResponse != nil && rec.Code == http.StatusOK {
				tt.checkResponse(t, rec.Body.Bytes())
			}
		})
	}
}

func TestFeedbackHandler_GetDimensionWeights(t *testing.T) {
	mockVoteRepo := &MockVoteRepository{}
	mockOptService := &MockOptimizationService{
		weights: map[string]float64{
			"successRate":    0.30,
			"quality":        0.25,
			"efficiency":     0.15,
			"robustness":     0.10,
			"generalization": 0.10,
			"diversity":      0.05,
			"innovation":     0.05,
		},
	}

	handler := NewFeedbackHandler(mockVoteRepo, mockOptService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/feedback/dimensions", nil)
	req = setTestUserID(req, "test-user")

	rec := httptest.NewRecorder()
	handler.GetDimensionWeights(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp DimensionWeightsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.SuccessRate != 0.30 {
		t.Errorf("expected SuccessRate to be 0.30, got %f", resp.SuccessRate)
	}

	if resp.Quality != 0.25 {
		t.Errorf("expected Quality to be 0.25, got %f", resp.Quality)
	}
}

func TestFeedbackHandler_UpdateDimensionWeights(t *testing.T) {
	mockVoteRepo := &MockVoteRepository{}
	mockOptService := &MockOptimizationService{}

	handler := NewFeedbackHandler(mockVoteRepo, mockOptService)

	request := DimensionWeightsResponse{
		SuccessRate:    0.40,
		Quality:        0.30,
		Efficiency:     0.10,
		Robustness:     0.10,
		Generalization: 0.05,
		Diversity:      0.03,
		Innovation:     0.02,
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/feedback/dimensions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req = setTestUserID(req, "test-user")

	rec := httptest.NewRecorder()
	handler.UpdateDimensionWeights(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp DimensionWeightsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Weights should be normalized, so they should sum to approximately 1.0
	sum := resp.SuccessRate + resp.Quality + resp.Efficiency + resp.Robustness +
		resp.Generalization + resp.Diversity + resp.Innovation

	if sum < 0.99 || sum > 1.01 {
		t.Errorf("expected weights to sum to 1.0, got %f", sum)
	}
}
