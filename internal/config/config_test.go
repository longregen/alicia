package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// LLM defaults
	if cfg.LLM.URL == "" {
		t.Error("LLM URL should not be empty")
	}
	if cfg.LLM.Model == "" {
		t.Error("LLM Model should not be empty")
	}
	if cfg.LLM.MaxTokens <= 0 {
		t.Error("LLM MaxTokens should be positive")
	}
	if cfg.LLM.Temperature < 0 || cfg.LLM.Temperature > 2 {
		t.Error("LLM Temperature should be between 0 and 2")
	}

	// LiveKit defaults
	if cfg.LiveKit.URL == "" {
		t.Error("LiveKit URL should not be empty")
	}
	if cfg.LiveKit.WorkerCount <= 0 {
		t.Error("LiveKit WorkerCount should be positive")
	}
	if cfg.LiveKit.WorkQueueSize <= 0 {
		t.Error("LiveKit WorkQueueSize should be positive")
	}

	// Server defaults
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		t.Error("Server Port should be valid")
	}
	if cfg.Server.Host == "" {
		t.Error("Server Host should not be empty")
	}

	// Database defaults
	if cfg.Database.Path == "" {
		t.Error("Database Path should not be empty")
	}

	// MCP defaults
	if cfg.MCP.Servers == nil {
		t.Error("MCP Servers should be initialized")
	}
}

func TestEnvString(t *testing.T) {
	target := "original"

	t.Run("sets value when env var exists", func(t *testing.T) {
		t.Setenv("TEST_VAR", "new_value")
		envString("TEST_VAR", &target)
		if target != "new_value" {
			t.Errorf("expected 'new_value', got '%s'", target)
		}
	})

	t.Run("does not change value when env var is empty", func(t *testing.T) {
		t.Setenv("TEST_VAR", "")
		target = "original"
		envString("TEST_VAR", &target)
		if target != "original" {
			t.Errorf("expected 'original', got '%s'", target)
		}
	})

	t.Run("does not change value when env var is unset", func(t *testing.T) {
		target = "original"
		envString("NONEXISTENT_VAR", &target)
		if target != "original" {
			t.Errorf("expected 'original', got '%s'", target)
		}
	})
}

func TestEnvInt(t *testing.T) {
	target := 42

	t.Run("sets value when env var is valid int", func(t *testing.T) {
		t.Setenv("TEST_INT", "100")
		envInt("TEST_INT", &target)
		if target != 100 {
			t.Errorf("expected 100, got %d", target)
		}
	})

	t.Run("does not change value when env var is invalid", func(t *testing.T) {
		t.Setenv("TEST_INT", "not_a_number")
		target = 42
		envInt("TEST_INT", &target)
		if target != 42 {
			t.Errorf("expected 42, got %d", target)
		}
	})

	t.Run("does not change value when env var is empty", func(t *testing.T) {
		t.Setenv("TEST_INT", "")
		target = 42
		envInt("TEST_INT", &target)
		if target != 42 {
			t.Errorf("expected 42, got %d", target)
		}
	})
}

func TestEnvFloat(t *testing.T) {
	target := 0.5

	t.Run("sets value when env var is valid float", func(t *testing.T) {
		t.Setenv("TEST_FLOAT", "0.8")
		envFloat("TEST_FLOAT", &target)
		if target != 0.8 {
			t.Errorf("expected 0.8, got %f", target)
		}
	})

	t.Run("does not change value when env var is invalid", func(t *testing.T) {
		t.Setenv("TEST_FLOAT", "not_a_float")
		target = 0.5
		envFloat("TEST_FLOAT", &target)
		if target != 0.5 {
			t.Errorf("expected 0.5, got %f", target)
		}
	})

	t.Run("does not change value when env var is empty", func(t *testing.T) {
		t.Setenv("TEST_FLOAT", "")
		target = 0.5
		envFloat("TEST_FLOAT", &target)
		if target != 0.5 {
			t.Errorf("expected 0.5, got %f", target)
		}
	})
}

func TestEnvStringSlice(t *testing.T) {
	target := []string{"original"}

	t.Run("parses comma-separated values", func(t *testing.T) {
		t.Setenv("TEST_SLICE", "a,b,c")
		envStringSlice("TEST_SLICE", &target)
		if len(target) != 3 || target[0] != "a" || target[1] != "b" || target[2] != "c" {
			t.Errorf("expected [a b c], got %v", target)
		}
	})

	t.Run("trims whitespace from values", func(t *testing.T) {
		t.Setenv("TEST_SLICE", " a , b , c ")
		target = []string{"original"}
		envStringSlice("TEST_SLICE", &target)
		if len(target) != 3 || target[0] != "a" || target[1] != "b" || target[2] != "c" {
			t.Errorf("expected [a b c], got %v", target)
		}
	})

	t.Run("filters empty values", func(t *testing.T) {
		t.Setenv("TEST_SLICE", "a,,b,  ,c")
		target = []string{"original"}
		envStringSlice("TEST_SLICE", &target)
		if len(target) != 3 || target[0] != "a" || target[1] != "b" || target[2] != "c" {
			t.Errorf("expected [a b c], got %v", target)
		}
	})

	t.Run("does not change value when env var is empty", func(t *testing.T) {
		t.Setenv("TEST_SLICE", "")
		target = []string{"original"}
		envStringSlice("TEST_SLICE", &target)
		if len(target) != 1 || target[0] != "original" {
			t.Errorf("expected [original], got %v", target)
		}
	})
}

func TestValidate_ServerPort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"valid port 80", 80, false},
		{"valid port 8080", 8080, false},
		{"valid port 65535", 65535, false},
		{"invalid port 0", 0, true},
		{"invalid port -1", -1, true},
		{"invalid port 65536", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Server.Port = tt.port
			// Clear LiveKit URL to avoid validation errors
			cfg.LiveKit.URL = ""
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), "server port") {
				t.Errorf("error should mention server port, got: %v", err)
			}
		})
	}
}

func TestValidate_LLMTemperature(t *testing.T) {
	tests := []struct {
		name        string
		temperature float64
		wantErr     bool
	}{
		{"valid temp 0", 0, false},
		{"valid temp 0.7", 0.7, false},
		{"valid temp 2.0", 2.0, false},
		{"invalid temp -0.1", -0.1, true},
		{"invalid temp 2.1", 2.1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.LLM.Temperature = tt.temperature
			// Clear LiveKit URL to avoid validation errors
			cfg.LiveKit.URL = ""
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), "temperature") {
				t.Errorf("error should mention temperature, got: %v", err)
			}
		})
	}
}

func TestValidate_LLMMaxTokens(t *testing.T) {
	cfg := DefaultConfig()
	// Clear LiveKit URL to avoid validation errors
	cfg.LiveKit.URL = ""
	cfg.LLM.MaxTokens = 0
	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for zero max_tokens")
	}
	if !strings.Contains(err.Error(), "max_tokens") {
		t.Errorf("error should mention max_tokens, got: %v", err)
	}

	cfg.LLM.MaxTokens = -1
	err = cfg.Validate()
	if err == nil {
		t.Error("expected error for negative max_tokens")
	}
}

func TestValidate_LLMURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid http URL", "http://localhost:8000", false},
		{"valid https URL", "https://api.example.com/v1", false},
		{"empty URL", "", true},
		{"invalid URL without scheme", "localhost:8000", true},
		{"invalid URL without host", "http://", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.LLM.URL = tt.url
			// Clear LiveKit URL to avoid validation errors
			cfg.LiveKit.URL = ""
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), "LLM URL") {
				t.Errorf("error should mention LLM URL, got: %v", err)
			}
		})
	}
}

func TestValidate_Database(t *testing.T) {
	t.Run("requires either Path or PostgresURL", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Database.Path = ""
		cfg.Database.PostgresURL = ""
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error when both database fields are empty")
		}
		if !strings.Contains(err.Error(), "PostgreSQL URL or database path") {
			t.Errorf("error should mention database requirement, got: %v", err)
		}
	})

	t.Run("validates PostgresURL format", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Database.PostgresURL = "invalid-url"
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for invalid PostgresURL")
		}
		if !strings.Contains(err.Error(), "PostgreSQL URL") {
			t.Errorf("error should mention PostgreSQL URL, got: %v", err)
		}
	})

	t.Run("accepts valid PostgresURL", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Database.PostgresURL = "postgresql://user:pass@localhost/db"
		// Clear LiveKit URL to avoid validation errors
		cfg.LiveKit.URL = ""
		err := cfg.Validate()
		if err != nil {
			t.Errorf("unexpected error for valid PostgresURL: %v", err)
		}
	})
}

func TestValidate_LiveKit(t *testing.T) {
	t.Run("requires credentials when URL is set", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.LiveKit.URL = "ws://localhost:7880"
		cfg.LiveKit.APIKey = ""
		cfg.LiveKit.APISecret = ""
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error when LiveKit credentials are missing")
		}
		if !strings.Contains(err.Error(), "API key and secret") {
			t.Errorf("error should mention API credentials, got: %v", err)
		}
	})

	t.Run("validates URL format", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.LiveKit.URL = "invalid-url"
		cfg.LiveKit.APIKey = "key"
		cfg.LiveKit.APISecret = "secret"
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for invalid LiveKit URL")
		}
		if !strings.Contains(err.Error(), "LiveKit URL") {
			t.Errorf("error should mention LiveKit URL, got: %v", err)
		}
	})

	t.Run("validates worker count", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.LiveKit.WorkerCount = 0
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for zero worker count")
		}
		if !strings.Contains(err.Error(), "worker count") {
			t.Errorf("error should mention worker count, got: %v", err)
		}
	})

	t.Run("validates work queue size", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.LiveKit.WorkQueueSize = 0
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for zero work queue size")
		}
		if !strings.Contains(err.Error(), "work queue size") {
			t.Errorf("error should mention work queue size, got: %v", err)
		}
	})
}

func TestValidate_OptionalServices(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*Config)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "invalid ASR URL",
			setupFunc: func(cfg *Config) {
				cfg.ASR.URL = "invalid-url"
			},
			wantErr: true,
			errMsg:  "ASR URL",
		},
		{
			name: "invalid TTS URL",
			setupFunc: func(cfg *Config) {
				cfg.TTS.URL = "invalid-url"
			},
			wantErr: true,
			errMsg:  "TTS URL",
		},
		{
			name: "invalid Embedding URL",
			setupFunc: func(cfg *Config) {
				cfg.Embedding.URL = "invalid-url"
			},
			wantErr: true,
			errMsg:  "Embedding URL",
		},
		{
			name: "embedding dimensions required when URL set",
			setupFunc: func(cfg *Config) {
				cfg.Embedding.URL = "http://localhost:11434"
				cfg.Embedding.Dimensions = 0
			},
			wantErr: true,
			errMsg:  "dimensions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.setupFunc(cfg)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("error should contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidate_MCP(t *testing.T) {
	t.Run("requires server name", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MCP.Servers = []MCPServerConfig{{
			Name:      "",
			Transport: "stdio",
			Command:   "node",
		}}
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for missing server name")
		}
		if !strings.Contains(err.Error(), "name is required") {
			t.Errorf("error should mention name requirement, got: %v", err)
		}
	})

	t.Run("validates transport type", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MCP.Servers = []MCPServerConfig{{
			Name:      "test",
			Transport: "invalid",
		}}
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for invalid transport")
		}
		if !strings.Contains(err.Error(), "transport must be") {
			t.Errorf("error should mention transport validation, got: %v", err)
		}
	})

	t.Run("requires command for stdio transport", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MCP.Servers = []MCPServerConfig{{
			Name:      "test",
			Transport: "stdio",
			Command:   "",
		}}
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for missing command in stdio transport")
		}
		if !strings.Contains(err.Error(), "command is required") {
			t.Errorf("error should mention command requirement, got: %v", err)
		}
	})

	t.Run("requires URL for http transport", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MCP.Servers = []MCPServerConfig{{
			Name:      "test",
			Transport: "http",
			URL:       "",
		}}
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for missing URL in http transport")
		}
		if !strings.Contains(err.Error(), "URL is required") {
			t.Errorf("error should mention URL requirement, got: %v", err)
		}
	})

	t.Run("validates http URL format", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MCP.Servers = []MCPServerConfig{{
			Name:      "test",
			Transport: "http",
			URL:       "invalid-url",
		}}
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for invalid MCP server URL")
		}
		if !strings.Contains(err.Error(), "URL must be a valid URL") {
			t.Errorf("error should mention URL validation, got: %v", err)
		}
	})

	t.Run("accepts valid stdio server", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.LiveKit.URL = ""
		cfg.MCP.Servers = []MCPServerConfig{{
			Name:      "test",
			Transport: "stdio",
			Command:   "node",
			Args:      []string{"server.js"},
		}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("unexpected error for valid stdio server: %v", err)
		}
	})

	t.Run("accepts valid http server", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.LiveKit.URL = ""
		cfg.MCP.Servers = []MCPServerConfig{{
			Name:      "test",
			Transport: "http",
			URL:       "http://localhost:3000",
		}}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("unexpected error for valid http server: %v", err)
		}
	})
}

func TestIsLiveKitConfigured(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		apiKey    string
		apiSecret string
		want      bool
	}{
		{"fully configured", "ws://localhost:7880", "key", "secret", true},
		{"missing URL", "", "key", "secret", false},
		{"missing API key", "ws://localhost:7880", "", "secret", false},
		{"missing API secret", "ws://localhost:7880", "key", "", false},
		{"all empty", "", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.LiveKit.URL = tt.url
			cfg.LiveKit.APIKey = tt.apiKey
			cfg.LiveKit.APISecret = tt.apiSecret
			if got := cfg.IsLiveKitConfigured(); got != tt.want {
				t.Errorf("IsLiveKitConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsASRConfigured(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.IsASRConfigured() {
		t.Error("default config should have ASR configured")
	}

	cfg.ASR.URL = ""
	if cfg.IsASRConfigured() {
		t.Error("ASR should not be configured with empty URL")
	}

	cfg.ASR.URL = "http://localhost:8001"
	if !cfg.IsASRConfigured() {
		t.Error("ASR should be configured with valid URL")
	}
}

func TestIsTTSConfigured(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.IsTTSConfigured() {
		t.Error("default config should have TTS configured")
	}

	cfg.TTS.URL = ""
	if cfg.IsTTSConfigured() {
		t.Error("TTS should not be configured with empty URL")
	}

	cfg.TTS.URL = "http://localhost:8001"
	if !cfg.IsTTSConfigured() {
		t.Error("TTS should be configured with valid URL")
	}
}

func TestIsEmbeddingConfigured(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.IsEmbeddingConfigured() {
		t.Error("default config should have Embedding configured")
	}

	cfg.Embedding.URL = ""
	if cfg.IsEmbeddingConfigured() {
		t.Error("Embedding should not be configured with empty URL")
	}

	cfg.Embedding.URL = "http://localhost:11434"
	if !cfg.IsEmbeddingConfigured() {
		t.Error("Embedding should be configured with valid URL")
	}
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"valid http", "http://localhost:8000", true},
		{"valid https", "https://api.example.com", true},
		{"valid ws", "ws://localhost:7880", true},
		{"valid wss", "wss://example.com", true},
		{"valid postgresql", "postgresql://user:pass@localhost/db", true},
		{"missing scheme", "localhost:8000", false},
		{"missing host", "http://", false},
		{"empty string", "", false},
		{"scheme only", "http", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidURL(tt.url); got != tt.want {
				t.Errorf("isValidURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	t.Run("uses ALICIA_CONFIG env var when set", func(t *testing.T) {
		t.Setenv("ALICIA_CONFIG", "/custom/path/config.json")
		path := getConfigPath()
		if path != "/custom/path/config.json" {
			t.Errorf("expected custom path, got %s", path)
		}
	})

	t.Run("defaults to .config/alicia when no env var", func(t *testing.T) {
		path := getConfigPath()
		expectedPath := filepath.Join(homeDir, ".config", "alicia", "config.json")
		if path != expectedPath {
			t.Errorf("expected %s, got %s", expectedPath, path)
		}
	})
}
