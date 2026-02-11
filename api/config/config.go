package config

import (
	iconfig "github.com/longregen/alicia/shared/config"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	LiveKit   LiveKitConfig
	Headscale HeadscaleConfig
	Langfuse  LangfuseConfig
	Otel      OtelConfig
}

type OtelConfig struct {
	Endpoint    string
	Environment string
}

type LangfuseConfig struct {
	Host      string
	PublicKey string
	SecretKey string
}

type ServerConfig struct {
	Host             string
	Port             int
	AllowedOrigins   []string
	AllowEmptyOrigin bool
	AgentSecret      string
	RequireAuth      bool
}

type DatabaseConfig struct {
	URL string
}

type LiveKitConfig struct {
	URL       string
	APIKey    string
	APISecret string
}

type HeadscaleConfig struct {
	URL        string
	PreAuthKey string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host:             iconfig.GetEnvWithFallback("ALICIA_SERVER_HOST", "HOST", "0.0.0.0"),
			Port:             iconfig.GetEnvIntWithFallback("ALICIA_SERVER_PORT", "PORT", 8080),
			AllowedOrigins:   iconfig.GetEnvSliceWithFallback("ALICIA_ALLOWED_ORIGINS", "ALLOWED_ORIGINS", []string{"*"}),
			AllowEmptyOrigin: iconfig.GetEnvBoolWithFallback("ALICIA_ALLOW_EMPTY_ORIGIN", "ALLOW_EMPTY_ORIGIN", false),
			AgentSecret:      iconfig.GetEnvWithFallback("ALICIA_AGENT_SECRET", "AGENT_SECRET", ""),
			RequireAuth:      iconfig.GetEnvBoolWithFallback("ALICIA_REQUIRE_AUTH", "REQUIRE_AUTH", false),
		},
		Database: DatabaseConfig{
			URL: iconfig.GetEnvWithFallback("ALICIA_POSTGRES_URL", "DATABASE_URL", "postgres://localhost:5432/alicia?sslmode=disable"),
		},
		LiveKit: LiveKitConfig{
			URL:       iconfig.GetEnvWithFallback("ALICIA_LIVEKIT_URL", "LIVEKIT_URL", ""),
			APIKey:    iconfig.GetEnvWithFallback("ALICIA_LIVEKIT_API_KEY", "LIVEKIT_API_KEY", ""),
			APISecret: iconfig.GetEnvWithFallback("ALICIA_LIVEKIT_API_SECRET", "LIVEKIT_API_SECRET", ""),
		},
		Headscale: HeadscaleConfig{
			URL:        iconfig.GetEnvWithFallback("ALICIA_HEADSCALE_URL", "HEADSCALE_URL", ""),
			PreAuthKey: iconfig.GetEnvWithFallback("ALICIA_HEADSCALE_PREAUTH_KEY", "HEADSCALE_PREAUTH_KEY", ""),
		},
		Langfuse: LangfuseConfig{
			Host:      iconfig.GetEnvWithFallback("ALICIA_LANGFUSE_HOST", "LANGFUSE_HOST", ""),
			PublicKey: iconfig.GetEnvWithFallback("ALICIA_LANGFUSE_PUBLIC_KEY", "LANGFUSE_PUBLIC_KEY", ""),
			SecretKey: iconfig.GetEnvWithFallback("ALICIA_LANGFUSE_SECRET_KEY", "LANGFUSE_SECRET_KEY", ""),
		},
		Otel: OtelConfig{
			Endpoint:    iconfig.GetEnvWithFallback("ALICIA_OTEL_ENDPOINT", "OTEL_EXPORTER_OTLP_ENDPOINT", ""),
			Environment: iconfig.GetEnvWithFallback("ALICIA_ENVIRONMENT", "ENVIRONMENT", "development"),
		},
	}
}

func (c *Config) IsLangfuseConfigured() bool {
	return c.Langfuse.Host != "" && c.Langfuse.PublicKey != "" && c.Langfuse.SecretKey != ""
}

func (c *Config) IsLiveKitConfigured() bool {
	return c.LiveKit.URL != "" && c.LiveKit.APIKey != "" && c.LiveKit.APISecret != ""
}

func (c *Config) IsHeadscaleConfigured() bool {
	return c.Headscale.URL != "" && c.Headscale.PreAuthKey != ""
}
