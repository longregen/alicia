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

func respondJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, errorType string, message string, status int) {
	if status >= 400 && status < 500 {
		log.Printf("HTTP %d: type=%s message=%s", status, errorType, message)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(dto.NewErrorResponse(errorType, message, status))
}

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

func validateURLParam(r *http.Request, w http.ResponseWriter, paramName, errorField string) (string, bool) {
	value := chi.URLParam(r, paramName)
	if value == "" {
		respondError(w, "invalid_request", errorField+" is required", http.StatusBadRequest)
		return "", false
	}
	return value, true
}

func decodeJSON[T any](r *http.Request, w http.ResponseWriter) (*T, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	var req T
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid_request", "Invalid request body", http.StatusBadRequest)
		return nil, false
	}
	return &req, true
}

func requireActiveConversation(conv *models.Conversation, w http.ResponseWriter) bool {
	if !conv.IsActive() {
		respondError(w, "conversation_inactive", "Conversation is not active", http.StatusBadRequest)
		return false
	}
	return true
}

func respondMsgpack(w http.ResponseWriter, data interface{}, status int) {
	if err := encoding.WriteMsgpack(w, status, data); err != nil {
		respondError(w, "internal_error", "Failed to encode response", http.StatusInternalServerError)
	}
}

func decodeMsgpack[T any](r *http.Request, w http.ResponseWriter) (*T, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

	var req T
	decoder := msgpack.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		respondError(w, "invalid_request", "Invalid request body", http.StatusBadRequest)
		return nil, false
	}
	return &req, true
}
