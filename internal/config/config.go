package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds all configuration for Alicia
type Config struct {
	LLM       LLMConfig       `json:"llm"`
	LiveKit   LiveKitConfig   `json:"livekit"`
	ASR       ASRConfig       `json:"asr"`
	TTS       TTSConfig       `json:"tts"`
	Embedding EmbeddingConfig `json:"embedding"`
	Database  DatabaseConfig  `json:"database"`
	Server    ServerConfig    `json:"server"`
	MCP       MCPConfig       `json:"mcp"`
}

// LLMConfig holds LLM API configuration (vLLM/LiteLLM)
type LLMConfig struct {
	URL         string  `json:"url"`
	APIKey      string  `json:"api_key"`
	Model       string  `json:"model"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

// LiveKitConfig holds LiveKit server configuration
type LiveKitConfig struct {
	URL           string `json:"url"`             // WebSocket URL (e.g., wss://localhost:7880)
	APIKey        string `json:"api_key"`         // LiveKit API key
	APISecret     string `json:"api_secret"`      // LiveKit API secret
	WorkerCount   int    `json:"worker_count"`    // Number of worker goroutines for event processing (default: 10)
	WorkQueueSize int    `json:"work_queue_size"` // Size of the buffered work queue (default: 100)
}

// ASRConfig holds Automatic Speech Recognition configuration (Whisper via speaches)
type ASRConfig struct {
	URL    string `json:"url"`
	APIKey string `json:"api_key"`
	Model  string `json:"model"` // e.g., "whisper-large-v3"
}

// TTSConfig holds Text-to-Speech configuration (Kokoro via speaches)
type TTSConfig struct {
	URL    string `json:"url"`
	APIKey string `json:"api_key"`
	Model  string `json:"model"` // e.g., "kokoro"
	Voice  string `json:"voice"` // e.g., "af_sarah"
}

// EmbeddingConfig holds embedding API configuration
type EmbeddingConfig struct {
	URL        string `json:"url"`
	APIKey     string `json:"api_key"`
	Model      string `json:"model"`      // e.g., "text-embedding-3-small"
	Dimensions int    `json:"dimensions"` // e.g., 1536 for text-embedding-3-small
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	// Path is used for SQLite (CLI mode)
	Path string `json:"path"`
	// PostgreSQL connection (server mode)
	PostgresURL string `json:"postgres_url"`
}

// ServerConfig holds API server configuration
type ServerConfig struct {
	Host        string   `json:"host"`
	Port        int      `json:"port"`
	StaticDir   string   `json:"static_dir"`   // Path to frontend static files
	CORSOrigins []string `json:"cors_origins"` // Allowed CORS origins
}

// MCPConfig holds MCP (Model Context Protocol) server configurations
type MCPConfig struct {
	Servers []MCPServerConfig `json:"servers"`
}

// MCPServerConfig represents a single MCP server configuration
type MCPServerConfig struct {
	Name           string   `json:"name"`
	Transport      string   `json:"transport"` // "stdio" or "http"
	Command        string   `json:"command,omitempty"`
	Args           []string `json:"args,omitempty"`
	Env            []string `json:"env,omitempty"`
	URL            string   `json:"url,omitempty"`
	APIKey         string   `json:"api_key,omitempty"`
	AutoReconnect  bool     `json:"auto_reconnect"`
	ReconnectDelay int      `json:"reconnect_delay,omitempty"` // in seconds
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".alicia")

	return &Config{
		LLM: LLMConfig{
			URL:         "http://localhost:8000/v1",
			APIKey:      "",
			Model:       "Qwen/Qwen3-8B-AWQ",
			MaxTokens:   4096,
			Temperature: 0.7,
		},
		LiveKit: LiveKitConfig{
			URL:           "ws://localhost:7880",
			APIKey:        "",
			APISecret:     "",
			WorkerCount:   10,
			WorkQueueSize: 100,
		},
		ASR: ASRConfig{
			URL:    "http://localhost:8001/v1",
			APIKey: "",
			Model:  "whisper-large-v3",
		},
		TTS: TTSConfig{
			URL:    "http://localhost:8001/v1",
			APIKey: "",
			Model:  "kokoro",
			Voice:  "af_sarah",
		},
		Embedding: EmbeddingConfig{
			URL:        "http://localhost:11434/v1",
			APIKey:     "",
			Model:      "text-embedding-3-small",
			Dimensions: 1536,
		},
		Database: DatabaseConfig{
			Path:        filepath.Join(dataDir, "alicia.db"),
			PostgresURL: "",
		},
		Server: ServerConfig{
			Host:        "0.0.0.0",
			Port:        8080,
			StaticDir:   "",                                // Empty by default, can be set via env
			CORSOrigins: []string{"http://localhost:3000"}, // Default development origin
		},
		MCP: MCPConfig{
			Servers: []MCPServerConfig{}, // No MCP servers by default
		},
	}
}

// envString loads a string environment variable into the target pointer if set
func envString(key string, target *string) {
	if v := os.Getenv(key); v != "" {
		*target = v
	}
}

// envInt loads an integer environment variable into the target pointer if set and valid
func envInt(key string, target *int) {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			*target = i
		}
	}
}

// envFloat loads a float64 environment variable into the target pointer if set and valid
func envFloat(key string, target *float64) {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			*target = f
		}
	}
}

// envStringSlice loads a comma-separated environment variable into a string slice
func envStringSlice(key string, target *[]string) {
	if v := os.Getenv(key); v != "" {
		parts := strings.Split(v, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			*target = result
		}
	}
}

// Load loads configuration from environment variables and config file
func Load() (*Config, error) {
	cfg := DefaultConfig()

	configPath := getConfigPath()
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse config file %s: %v\n", configPath, err)
		}
	}

	// Load LLM configuration from environment
	envString("ALICIA_LLM_URL", &cfg.LLM.URL)
	envString("ALICIA_LLM_API_KEY", &cfg.LLM.APIKey)
	envString("ALICIA_LLM_MODEL", &cfg.LLM.Model)
	envInt("ALICIA_LLM_MAX_TOKENS", &cfg.LLM.MaxTokens)
	envFloat("ALICIA_LLM_TEMPERATURE", &cfg.LLM.Temperature)

	// Load LiveKit configuration from environment
	envString("ALICIA_LIVEKIT_URL", &cfg.LiveKit.URL)
	envString("ALICIA_LIVEKIT_API_KEY", &cfg.LiveKit.APIKey)
	envString("ALICIA_LIVEKIT_API_SECRET", &cfg.LiveKit.APISecret)
	envInt("ALICIA_LIVEKIT_WORKER_COUNT", &cfg.LiveKit.WorkerCount)
	envInt("ALICIA_LIVEKIT_WORK_QUEUE_SIZE", &cfg.LiveKit.WorkQueueSize)

	// Load ASR configuration from environment
	envString("ALICIA_ASR_URL", &cfg.ASR.URL)
	envString("ALICIA_ASR_API_KEY", &cfg.ASR.APIKey)
	envString("ALICIA_ASR_MODEL", &cfg.ASR.Model)

	// Load TTS configuration from environment
	envString("ALICIA_TTS_URL", &cfg.TTS.URL)
	envString("ALICIA_TTS_API_KEY", &cfg.TTS.APIKey)
	envString("ALICIA_TTS_MODEL", &cfg.TTS.Model)
	envString("ALICIA_TTS_VOICE", &cfg.TTS.Voice)

	// Load Embedding configuration from environment
	envString("ALICIA_EMBEDDING_URL", &cfg.Embedding.URL)
	envString("ALICIA_EMBEDDING_API_KEY", &cfg.Embedding.APIKey)
	envString("ALICIA_EMBEDDING_MODEL", &cfg.Embedding.Model)
	envInt("ALICIA_EMBEDDING_DIMENSIONS", &cfg.Embedding.Dimensions)

	// Load Database configuration from environment
	envString("ALICIA_DB_PATH", &cfg.Database.Path)
	envString("ALICIA_POSTGRES_URL", &cfg.Database.PostgresURL)

	// Load Server configuration from environment
	envString("ALICIA_STATIC_DIR", &cfg.Server.StaticDir)
	envString("ALICIA_SERVER_HOST", &cfg.Server.Host)
	envInt("ALICIA_SERVER_PORT", &cfg.Server.Port)
	envStringSlice("ALICIA_CORS_ORIGINS", &cfg.Server.CORSOrigins)

	// Load MCP configuration from environment
	// MCP servers are primarily configured via config file, but can be augmented via env
	if mcpServersJSON := os.Getenv("ALICIA_MCP_SERVERS"); mcpServersJSON != "" {
		var envServers []MCPServerConfig
		if err := json.Unmarshal([]byte(mcpServersJSON), &envServers); err == nil {
			cfg.MCP.Servers = append(cfg.MCP.Servers, envServers...)
		}
	}

	dataDir := filepath.Dir(cfg.Database.Path)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// IsLiveKitConfigured returns true if LiveKit is properly configured
func (c *Config) IsLiveKitConfigured() bool {
	return c.LiveKit.URL != "" && c.LiveKit.APIKey != "" && c.LiveKit.APISecret != ""
}

// IsASRConfigured returns true if ASR (speech recognition) is configured
func (c *Config) IsASRConfigured() bool {
	return c.ASR.URL != ""
}

// IsTTSConfigured returns true if TTS (text-to-speech) is configured
func (c *Config) IsTTSConfigured() bool {
	return c.TTS.URL != ""
}

// IsEmbeddingConfigured returns true if embedding service is configured
func (c *Config) IsEmbeddingConfigured() bool {
	return c.Embedding.URL != ""
}

// isValidURL validates that a URL has proper format
func isValidURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// Validate checks that the configuration has valid values
func (c *Config) Validate() error {
	var errs []string

	// Server validation
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errs = append(errs, "server port must be between 1 and 65535")
	}

	// LLM validation
	if c.LLM.Temperature < 0 || c.LLM.Temperature > 2 {
		errs = append(errs, "LLM temperature must be between 0 and 2")
	}
	if c.LLM.MaxTokens < 1 {
		errs = append(errs, "LLM max_tokens must be positive")
	}
	if c.LLM.URL == "" {
		errs = append(errs, "LLM URL is required")
	} else if !isValidURL(c.LLM.URL) {
		errs = append(errs, "LLM URL must be a valid URL")
	}

	// Database validation
	if c.Database.PostgresURL == "" && c.Database.Path == "" {
		errs = append(errs, "either PostgreSQL URL or database path is required")
	}
	if c.Database.PostgresURL != "" && !isValidURL(c.Database.PostgresURL) {
		errs = append(errs, "PostgreSQL URL must be a valid URL")
	}

	// LiveKit validation (if enabled)
	if c.LiveKit.URL != "" {
		if !isValidURL(c.LiveKit.URL) {
			errs = append(errs, "LiveKit URL must be a valid URL")
		}
		if c.LiveKit.APIKey == "" || c.LiveKit.APISecret == "" {
			errs = append(errs, "LiveKit API key and secret are required when URL is set")
		}
	}
	if c.LiveKit.WorkerCount < 1 {
		errs = append(errs, "LiveKit worker count must be at least 1")
	}
	if c.LiveKit.WorkQueueSize < 1 {
		errs = append(errs, "LiveKit work queue size must be at least 1")
	}

	// ASR validation (optional but validate if set)
	if c.ASR.URL != "" && !isValidURL(c.ASR.URL) {
		errs = append(errs, "ASR URL must be a valid URL")
	}

	// TTS validation (optional but validate if set)
	if c.TTS.URL != "" && !isValidURL(c.TTS.URL) {
		errs = append(errs, "TTS URL must be a valid URL")
	}

	// Embedding validation (optional but validate if set)
	if c.Embedding.URL != "" {
		if !isValidURL(c.Embedding.URL) {
			errs = append(errs, "Embedding URL must be a valid URL")
		}
		if c.Embedding.Dimensions < 1 {
			errs = append(errs, "Embedding dimensions must be positive when URL is set")
		}
	}

	// MCP validation
	for i, server := range c.MCP.Servers {
		if server.Name == "" {
			errs = append(errs, fmt.Sprintf("MCP server %d: name is required", i))
		}
		if server.Transport != "stdio" && server.Transport != "http" {
			errs = append(errs, fmt.Sprintf("MCP server %s: transport must be 'stdio' or 'http'", server.Name))
		}
		if server.Transport == "stdio" && server.Command == "" {
			errs = append(errs, fmt.Sprintf("MCP server %s: command is required for stdio transport", server.Name))
		}
		if server.Transport == "http" && server.URL == "" {
			errs = append(errs, fmt.Sprintf("MCP server %s: URL is required for http transport", server.Name))
		}
		if server.Transport == "http" && server.URL != "" && !isValidURL(server.URL) {
			errs = append(errs, fmt.Sprintf("MCP server %s: URL must be a valid URL", server.Name))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// getConfigPath returns the path to the config file
func getConfigPath() string {
	if path := os.Getenv("ALICIA_CONFIG"); path != "" {
		return path
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "config.json"
	}

	// Check ~/.config/alicia/config.json first
	configDir := filepath.Join(homeDir, ".config", "alicia")
	configPath := filepath.Join(configDir, "config.json")
	if _, err := os.Stat(configPath); err == nil {
		return configPath
	}

	// Check ~/.alicia/config.json
	altPath := filepath.Join(homeDir, ".alicia", "config.json")
	if _, err := os.Stat(altPath); err == nil {
		return altPath
	}

	return configPath
}
