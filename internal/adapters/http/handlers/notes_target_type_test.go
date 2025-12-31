package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/internal/domain/models"
)

// TestNoteHandler_UpdateNote_InfersTargetType verifies that UpdateNote correctly infers
// the target type from the note's message_id (which contains the target ID)
func TestNoteHandler_UpdateNote_InfersTargetType(t *testing.T) {
	tests := []struct {
		name         string
		messageID    string // This is the target ID stored in note.MessageID
		expectedType string
		description  string
	}{
		{
			name:         "message target",
			messageID:    "am_abc123",
			expectedType: "message",
			description:  "Message IDs have prefix 'am_'",
		},
		{
			name:         "tool_use target",
			messageID:    "atu_xyz789",
			expectedType: "tool_use",
			description:  "Tool use IDs have prefix 'atu_'",
		},
		{
			name:         "reasoning target",
			messageID:    "ar_def456",
			expectedType: "reasoning",
			description:  "Reasoning step IDs have prefix 'ar_'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noteRepo := newMockNoteRepo()
			idGen := newMockIDGenerator()

			// Create a note with the target ID stored in MessageID field
			note := models.NewNote("an_test123", tt.messageID, "Original content", "general")
			noteRepo.notes["an_test123"] = note

			handler := NewNoteHandler(noteRepo, idGen)

			body := `{"content": "Updated content"}`
			req := httptest.NewRequest("PUT", "/api/v1/notes/an_test123", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			req = addUserContext(req, "test-user")

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", "an_test123")
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()
			handler.UpdateNote(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rr.Code)
			}

			var response NoteResponse
			if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			// Verify the target type was correctly inferred
			if response.TargetType != tt.expectedType {
				t.Errorf("expected target_type '%s', got '%s' (test: %s)",
					tt.expectedType, response.TargetType, tt.description)
			}

			// Verify the target ID matches the message ID
			if response.TargetID != tt.messageID {
				t.Errorf("expected target_id '%s', got '%s'", tt.messageID, response.TargetID)
			}

			// Verify content was updated
			if response.Content != "Updated content" {
				t.Errorf("expected content 'Updated content', got '%s'", response.Content)
			}
		})
	}
}

// TestInferTargetType tests the inferTargetType helper function directly
func TestInferTargetType(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected string
	}{
		{
			name:     "message ID",
			id:       "am_abc123xyz",
			expected: "message",
		},
		{
			name:     "tool use ID",
			id:       "atu_abc123xyz",
			expected: "tool_use",
		},
		{
			name:     "reasoning ID",
			id:       "ar_abc123xyz",
			expected: "reasoning",
		},
		{
			name:     "short string defaults to message",
			id:       "ab",
			expected: "message",
		},
		{
			name:     "empty string defaults to message",
			id:       "",
			expected: "message",
		},
		{
			name:     "unknown prefix defaults to message",
			id:       "xyz_abc123",
			expected: "message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferTargetType(tt.id)
			if result != tt.expected {
				t.Errorf("inferTargetType(%q) = %q, want %q", tt.id, result, tt.expected)
			}
		})
	}
}
