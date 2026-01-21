package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/longregen/alicia/internal/adapters/http/dto"
	"github.com/longregen/alicia/internal/adapters/http/encoding"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/vmihailenco/msgpack/v5"
)

// respondJSON writes a JSON response with the given status code
func respondJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError writes an error JSON response
func respondError(w http.ResponseWriter, errorType string, message string, status int) {
	// Log 4xx client errors for debugging
	if status >= 400 && status < 500 {
		log.Printf("HTTP %d: type=%s message=%s", status, errorType, message)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(dto.NewErrorResponse(errorType, message, status))
}

// parseIntQuery parses an integer query parameter with a default value
func parseIntQuery(r *http.Request, name string, defaultValue int) int {
	value := r.URL.Query().Get(name)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}

// validateURLParam validates and returns a URL parameter
func validateURLParam(r *http.Request, w http.ResponseWriter, paramName, errorField string) (string, bool) {
	value := chi.URLParam(r, paramName)
	if value == "" {
		respondError(w, "invalid_request", errorField+" is required", http.StatusBadRequest)
		return "", false
	}
	return value, true
}

// decodeJSON decodes JSON request body with error handling
func decodeJSON[T any](r *http.Request, w http.ResponseWriter) (*T, bool) {
	// Add request body size limit
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024) // 1MB limit

	var req T
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid_request", "Invalid request body", http.StatusBadRequest)
		return nil, false
	}
	return &req, true
}

// requireActiveConversation checks if conversation is active
func requireActiveConversation(conv *models.Conversation, w http.ResponseWriter) bool {
	if !conv.IsActive() {
		respondError(w, "conversation_inactive", "Conversation is not active", http.StatusBadRequest)
		return false
	}
	return true
}

// respondMsgpack writes a MessagePack response with the given status code
func respondMsgpack(w http.ResponseWriter, data interface{}, status int) {
	if err := encoding.WriteMsgpack(w, status, data); err != nil {
		respondError(w, "internal_error", "Failed to encode response", http.StatusInternalServerError)
	}
}

// decodeMsgpack decodes MessagePack request body with error handling
func decodeMsgpack[T any](r *http.Request, w http.ResponseWriter) (*T, bool) {
	// Add request body size limit
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024) // 10MB limit

	var req T
	decoder := msgpack.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		respondError(w, "invalid_request", "Invalid request body", http.StatusBadRequest)
		return nil, false
	}
	return &req, true
}
