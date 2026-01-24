// Package jsonutil provides common JSON helper functions.
package jsonutil

import (
	"encoding/json"
)

// MustJSON marshals v to a JSON string.
// Returns an empty string on error.
func MustJSON(v any) string {
	if v == nil {
		return "{}"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// ParseJSON parses a JSON string into a map.
// Returns nil on error.
func ParseJSON(s string) map[string]any {
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil
	}
	return m
}

// MustMarshalIndent marshals v to a pretty-printed JSON string.
// Returns an empty string on error.
func MustMarshalIndent(v any) string {
	if v == nil {
		return "{}"
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}
