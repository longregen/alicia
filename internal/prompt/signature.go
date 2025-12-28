package prompt

import (
	"fmt"
	"strings"

	"github.com/XiaoConstantine/dspy-go/pkg/core"
)

// Signature wraps dspy-go's signature with Alicia-specific features
type Signature struct {
	core.Signature
	Name        string
	Description string
	Version     int
}

// MustParseSignature creates a signature from a string or panics
func MustParseSignature(sig string) Signature {
	s, err := ParseSignature(sig)
	if err != nil {
		panic(fmt.Sprintf("failed to parse signature: %v", err))
	}
	return s
}

// ParseSignature creates a signature from a string like "input1, input2 -> output1, output2"
func ParseSignature(sig string) (Signature, error) {
	parts := strings.Split(sig, "->")
	if len(parts) != 2 {
		return Signature{}, fmt.Errorf("invalid signature format: %s", sig)
	}

	inputFields := parseFields(strings.TrimSpace(parts[0]))
	outputFields := parseFields(strings.TrimSpace(parts[1]))

	// Convert to InputField and OutputField
	inputs := make([]core.InputField, len(inputFields))
	for i, f := range inputFields {
		inputs[i] = core.InputField{Field: f}
	}

	outputs := make([]core.OutputField, len(outputFields))
	for i, f := range outputFields {
		outputs[i] = core.OutputField{Field: f}
	}

	coreSig := core.NewSignature(inputs, outputs)

	return Signature{
		Signature: coreSig,
		Name:      generateName(sig),
		Version:   1,
	}, nil
}

// parseFields converts comma-separated field definitions into InputField and OutputField slices
func parseFields(fieldStr string) []core.Field {
	if fieldStr == "" {
		return nil
	}

	parts := strings.Split(fieldStr, ",")
	fields := make([]core.Field, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Parse field format: "name: type" or just "name"
		var name string
		if strings.Contains(part, ":") {
			fieldParts := strings.SplitN(part, ":", 2)
			name = strings.TrimSpace(fieldParts[0])
			// Type information is stored in Field.Type (FieldType)
		} else {
			name = part
		}

		fields = append(fields, core.NewField(name))
	}

	return fields
}

// generateName creates a name from the signature string
func generateName(sig string) string {
	// Remove special characters and spaces
	name := strings.ReplaceAll(sig, "->", "_to_")
	name = strings.ReplaceAll(name, ",", "_")
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, ":", "_")
	return name
}

// Predefined signatures for common Alicia use cases
var (
	ConversationResponse = MustParseSignature(
		"context, user_message, memories -> response",
	)

	ToolSelection = MustParseSignature(
		"user_intent, available_tools -> tool_name, reasoning",
	)

	MemoryExtraction = MustParseSignature(
		"conversation -> key_facts: list[str], importance: int",
	)
)
