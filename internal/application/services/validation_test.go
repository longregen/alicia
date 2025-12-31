package services

import (
	"testing"
	"time"
)

func TestValidateID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		entityType string
		wantError  bool
	}{
		{
			name:       "valid ID",
			id:         "msg_123",
			entityType: "message",
			wantError:  false,
		},
		{
			name:       "empty ID",
			id:         "",
			entityType: "message",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateID(tt.id, tt.entityType)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateID() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateNotDeleted(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		deletedAt  *time.Time
		entityType string
		wantError  bool
	}{
		{
			name:       "not deleted",
			deletedAt:  nil,
			entityType: "message",
			wantError:  false,
		},
		{
			name:       "deleted",
			deletedAt:  &now,
			entityType: "message",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNotDeleted(tt.deletedAt, tt.entityType)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateNotDeleted() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		fieldName string
		wantError bool
	}{
		{
			name:      "valid value",
			value:     "hello",
			fieldName: "content",
			wantError: false,
		},
		{
			name:      "empty value",
			value:     "",
			fieldName: "content",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequired(tt.value, tt.fieldName)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateRequired() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidatePositive(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		fieldName string
		wantError bool
	}{
		{
			name:      "positive value",
			value:     10,
			fieldName: "count",
			wantError: false,
		},
		{
			name:      "zero value",
			value:     0,
			fieldName: "count",
			wantError: true,
		},
		{
			name:      "negative value",
			value:     -5,
			fieldName: "count",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePositive(tt.value, tt.fieldName)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidatePositive() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateStringLength(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		fieldName string
		minLen    int
		maxLen    int
		wantError bool
	}{
		{
			name:      "valid length",
			value:     "hello",
			fieldName: "content",
			minLen:    3,
			maxLen:    10,
			wantError: false,
		},
		{
			name:      "too short",
			value:     "hi",
			fieldName: "content",
			minLen:    3,
			maxLen:    10,
			wantError: true,
		},
		{
			name:      "too long",
			value:     "hello world this is too long",
			fieldName: "content",
			minLen:    3,
			maxLen:    10,
			wantError: true,
		},
		{
			name:      "no min constraint",
			value:     "a",
			fieldName: "content",
			minLen:    0,
			maxLen:    10,
			wantError: false,
		},
		{
			name:      "no max constraint",
			value:     "hello world this is a very long string",
			fieldName: "content",
			minLen:    3,
			maxLen:    0,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStringLength(tt.value, tt.fieldName, tt.minLen, tt.maxLen)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateStringLength() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateRange(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		fieldName string
		min       int
		max       int
		wantError bool
	}{
		{
			name:      "valid range",
			value:     5,
			fieldName: "score",
			min:       0,
			max:       10,
			wantError: false,
		},
		{
			name:      "below min",
			value:     -1,
			fieldName: "score",
			min:       0,
			max:       10,
			wantError: true,
		},
		{
			name:      "above max",
			value:     11,
			fieldName: "score",
			min:       0,
			max:       10,
			wantError: true,
		},
		{
			name:      "at min boundary",
			value:     0,
			fieldName: "score",
			min:       0,
			max:       10,
			wantError: false,
		},
		{
			name:      "at max boundary",
			value:     10,
			fieldName: "score",
			min:       0,
			max:       10,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRange(tt.value, tt.fieldName, tt.min, tt.max)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateRange() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateJSONSize(t *testing.T) {
	tests := []struct {
		name         string
		value        interface{}
		fieldName    string
		maxSizeBytes int
		wantError    bool
	}{
		{
			name:         "small object",
			value:        map[string]string{"key": "value"},
			fieldName:    "metadata",
			maxSizeBytes: 100,
			wantError:    false,
		},
		{
			name: "large object",
			value: map[string]string{
				"key1": "very long value that will make this JSON exceed the size limit",
				"key2": "another very long value that adds to the total size",
				"key3": "yet another long value to push us over the limit",
			},
			fieldName:    "metadata",
			maxSizeBytes: 50,
			wantError:    true,
		},
		{
			name:         "nil value",
			value:        nil,
			fieldName:    "metadata",
			maxSizeBytes: 100,
			wantError:    false,
		},
		{
			name:         "array",
			value:        []int{1, 2, 3, 4, 5},
			fieldName:    "numbers",
			maxSizeBytes: 100,
			wantError:    false,
		},
		{
			name:         "string",
			value:        "hello world",
			fieldName:    "text",
			maxSizeBytes: 100,
			wantError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJSONSize(tt.value, tt.fieldName, tt.maxSizeBytes)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateJSONSize() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateConversationIDFormat(t *testing.T) {
	tests := []struct {
		name           string
		conversationID string
		wantError      bool
	}{
		{
			name:           "valid conversation ID",
			conversationID: "ac_123456",
			wantError:      false,
		},
		{
			name:           "empty conversation ID",
			conversationID: "",
			wantError:      true,
		},
		{
			name:           "wrong prefix",
			conversationID: "conv_123",
			wantError:      true,
		},
		{
			name:           "prefix only",
			conversationID: "ac_",
			wantError:      true,
		},
		{
			name:           "no prefix",
			conversationID: "123456",
			wantError:      true,
		},
		{
			name:           "valid long ID",
			conversationID: "ac_abcdef1234567890",
			wantError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConversationIDFormat(tt.conversationID)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateConversationIDFormat() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateStringLength_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		fieldName string
		minLen    int
		maxLen    int
		wantError bool
	}{
		{
			name:      "exact min length",
			value:     "abc",
			fieldName: "field",
			minLen:    3,
			maxLen:    10,
			wantError: false,
		},
		{
			name:      "exact max length",
			value:     "0123456789",
			fieldName: "field",
			minLen:    3,
			maxLen:    10,
			wantError: false,
		},
		{
			name:      "one below min",
			value:     "ab",
			fieldName: "field",
			minLen:    3,
			maxLen:    10,
			wantError: true,
		},
		{
			name:      "one above max",
			value:     "01234567890",
			fieldName: "field",
			minLen:    3,
			maxLen:    10,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStringLength(tt.value, tt.fieldName, tt.minLen, tt.maxLen)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateStringLength() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateJSONSize_InvalidJSON(t *testing.T) {
	// Create a value that cannot be marshaled to JSON
	type invalidType struct {
		Channel chan int
	}

	err := ValidateJSONSize(invalidType{Channel: make(chan int)}, "field", 100)
	if err == nil {
		t.Fatal("expected error for un-marshalable value, got nil")
	}
}
