package encoding

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNegotiateContentType(t *testing.T) {
	tests := []struct {
		name           string
		acceptHeader   string
		expectedType   string
	}{
		{
			name:         "Empty Accept header defaults to JSON",
			acceptHeader: "",
			expectedType: ContentTypeJSON,
		},
		{
			name:         "Explicit MessagePack request",
			acceptHeader: "application/msgpack",
			expectedType: ContentTypeMsgpack,
		},
		{
			name:         "Explicit JSON request",
			acceptHeader: "application/json",
			expectedType: ContentTypeJSON,
		},
		{
			name:         "Wildcard defaults to JSON",
			acceptHeader: "*/*",
			expectedType: ContentTypeJSON,
		},
		{
			name:         "Multiple types with MessagePack",
			acceptHeader: "application/json, application/msgpack",
			expectedType: ContentTypeMsgpack,
		},
		{
			name:         "Quality values with MessagePack preferred",
			acceptHeader: "application/json;q=0.9, application/msgpack;q=1.0",
			expectedType: ContentTypeMsgpack,
		},
		{
			name:         "Unknown content type defaults to JSON",
			acceptHeader: "application/xml",
			expectedType: ContentTypeJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}

			contentType := NegotiateContentType(req)
			if contentType != tt.expectedType {
				t.Errorf("expected content type %s, got %s", tt.expectedType, contentType)
			}
		})
	}
}

func TestWriteMsgpack(t *testing.T) {
	type TestData struct {
		Name  string `msgpack:"name"`
		Value int    `msgpack:"value"`
	}

	tests := []struct {
		name         string
		data         interface{}
		status       int
		expectError  bool
	}{
		{
			name: "Encode simple struct",
			data: TestData{
				Name:  "test",
				Value: 123,
			},
			status:      http.StatusOK,
			expectError: false,
		},
		{
			name: "Encode map",
			data: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
			status:      http.StatusCreated,
			expectError: false,
		},
		{
			name: "Encode slice",
			data: []string{"a", "b", "c"},
			status:      http.StatusOK,
			expectError: false,
		},
		{
			name:        "Encode nil",
			data:        nil,
			status:      http.StatusOK,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			err := WriteMsgpack(w, tt.status, tt.data)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if w.Code != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != ContentTypeMsgpack {
				t.Errorf("expected Content-Type %s, got %s", ContentTypeMsgpack, contentType)
			}
		})
	}
}

func TestReadMsgpack(t *testing.T) {
	type TestData struct {
		Name  string `msgpack:"name"`
		Value int    `msgpack:"value"`
	}

	tests := []struct {
		name        string
		input       TestData
		expectError bool
	}{
		{
			name: "Decode simple struct",
			input: TestData{
				Name:  "test",
				Value: 123,
			},
			expectError: false,
		},
		{
			name: "Decode empty struct",
			input: TestData{
				Name:  "",
				Value: 0,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First encode the test data
			var buf bytes.Buffer
			w := httptest.NewRecorder()
			w.Body = &buf
			err := WriteMsgpack(w, http.StatusOK, tt.input)
			if err != nil {
				t.Fatalf("failed to encode test data: %v", err)
			}

			// Create a request with the encoded data
			req := httptest.NewRequest("POST", "/test", bytes.NewReader(w.Body.Bytes()))
			req.Header.Set("Content-Type", ContentTypeMsgpack)

			// Decode the data
			var output TestData
			err = ReadMsgpack(req, &output)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				if output.Name != tt.input.Name {
					t.Errorf("expected name %s, got %s", tt.input.Name, output.Name)
				}
				if output.Value != tt.input.Value {
					t.Errorf("expected value %d, got %d", tt.input.Value, output.Value)
				}
			}
		})
	}
}

func TestReadMsgpack_InvalidData(t *testing.T) {
	type TestData struct {
		Name  string `msgpack:"name"`
		Value int    `msgpack:"value"`
	}

	// Create a request with invalid MessagePack data
	req := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte{0xFF, 0xFE, 0xFD}))
	req.Header.Set("Content-Type", ContentTypeMsgpack)

	var output TestData
	err := ReadMsgpack(req, &output)

	if err == nil {
		t.Error("expected error when decoding invalid MessagePack data")
	}
}

func TestRoundtrip(t *testing.T) {
	type ComplexData struct {
		String  string            `msgpack:"string"`
		Int     int               `msgpack:"int"`
		Float   float64           `msgpack:"float"`
		Bool    bool              `msgpack:"bool"`
		Slice   []string          `msgpack:"slice"`
		Map     map[string]string `msgpack:"map"`
		Pointer *string           `msgpack:"pointer,omitempty"`
	}

	testStr := "pointer value"
	original := ComplexData{
		String:  "test",
		Int:     42,
		Float:   3.14,
		Bool:    true,
		Slice:   []string{"a", "b", "c"},
		Map:     map[string]string{"key": "value"},
		Pointer: &testStr,
	}

	// Encode
	var buf bytes.Buffer
	w := httptest.NewRecorder()
	w.Body = &buf
	err := WriteMsgpack(w, http.StatusOK, original)
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}

	// Decode
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(w.Body.Bytes()))
	var decoded ComplexData
	err = ReadMsgpack(req, &decoded)
	if err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	// Verify
	if decoded.String != original.String {
		t.Errorf("String mismatch: expected %s, got %s", original.String, decoded.String)
	}
	if decoded.Int != original.Int {
		t.Errorf("Int mismatch: expected %d, got %d", original.Int, decoded.Int)
	}
	if decoded.Float != original.Float {
		t.Errorf("Float mismatch: expected %f, got %f", original.Float, decoded.Float)
	}
	if decoded.Bool != original.Bool {
		t.Errorf("Bool mismatch: expected %v, got %v", original.Bool, decoded.Bool)
	}
	if len(decoded.Slice) != len(original.Slice) {
		t.Errorf("Slice length mismatch: expected %d, got %d", len(original.Slice), len(decoded.Slice))
	}
	if len(decoded.Map) != len(original.Map) {
		t.Errorf("Map length mismatch: expected %d, got %d", len(original.Map), len(decoded.Map))
	}
	if decoded.Pointer == nil || *decoded.Pointer != *original.Pointer {
		t.Error("Pointer value mismatch")
	}
}
