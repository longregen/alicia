package prompt

import (
	"math"
	"testing"
)

func TestSqrt32(t *testing.T) {
	tests := []struct {
		name     string
		input    float32
		expected float32
	}{
		{"zero", 0.0, 0.0},
		{"one", 1.0, 1.0},
		{"four", 4.0, 2.0},
		{"nine", 9.0, 3.0},
		{"two", 2.0, 1.4142135},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sqrt32(tt.input)
			if math.Abs(float64(result-tt.expected)) > 0.0001 {
				t.Errorf("sqrt32(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float32
	}{
		{
			name:     "identical vectors",
			a:        []float32{1.0, 2.0, 3.0},
			b:        []float32{1.0, 2.0, 3.0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1.0, 0.0},
			b:        []float32{0.0, 1.0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1.0, 0.0},
			b:        []float32{-1.0, 0.0},
			expected: -1.0,
		},
		{
			name:     "similar vectors",
			a:        []float32{1.0, 1.0},
			b:        []float32{1.0, 0.9},
			expected: 0.9986178, // (1*1 + 1*0.9) / (sqrt(2) * sqrt(1.81))
		},
		{
			name:     "different length vectors",
			a:        []float32{1.0, 2.0},
			b:        []float32{1.0},
			expected: 0.0,
		},
		{
			name:     "zero vector",
			a:        []float32{0.0, 0.0},
			b:        []float32{1.0, 1.0},
			expected: 0.0,
		},
		{
			name:     "empty vectors",
			a:        []float32{},
			b:        []float32{},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			if math.Abs(float64(result-tt.expected)) > 0.0001 {
				t.Errorf("cosineSimilarity(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}
