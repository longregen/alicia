package prompt

import (
	"testing"
)

func TestParseSignature(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "simple signature",
			input:   "question -> answer",
			wantErr: false,
		},
		{
			name:    "multiple inputs and outputs",
			input:   "context, user_message, memories -> response",
			wantErr: false,
		},
		{
			name:    "with types",
			input:   "question: str -> answer: str, reasoning: str",
			wantErr: false,
		},
		{
			name:    "invalid signature",
			input:   "question",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig, err := ParseSignature(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSignature() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && sig.Name == "" {
				t.Errorf("ParseSignature() returned signature with empty name")
			}
		})
	}
}

func TestMustParseSignature(t *testing.T) {
	// Should not panic for valid signature
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustParseSignature() panicked on valid signature: %v", r)
		}
	}()

	sig := MustParseSignature("question -> answer")
	if sig.Name == "" {
		t.Errorf("MustParseSignature() returned signature with empty name")
	}
}

func TestMustParseSignaturePanic(t *testing.T) {
	// Should panic for invalid signature
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MustParseSignature() did not panic on invalid signature")
		}
	}()

	_ = MustParseSignature("invalid")
}

func TestPredefinedSignatures(t *testing.T) {
	signatures := []struct {
		name string
		sig  Signature
	}{
		{"ConversationResponse", ConversationResponse},
		{"ToolSelection", ToolSelection},
		{"MemoryExtraction", MemoryExtraction},
	}

	for _, tt := range signatures {
		t.Run(tt.name, func(t *testing.T) {
			if tt.sig.Name == "" {
				t.Errorf("%s has empty name", tt.name)
			}
			if len(tt.sig.Inputs) == 0 {
				t.Errorf("%s has no inputs", tt.name)
			}
			if len(tt.sig.Outputs) == 0 {
				t.Errorf("%s has no outputs", tt.name)
			}
		})
	}
}
