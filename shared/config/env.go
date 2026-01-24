// Package config provides environment variable helpers used across services.
package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// MustEnv fatally exits if the env var is not set.
func MustEnv(key string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	log.Fatalf("Required env var %s not set", key)
	return ""
}

func GetEnvWithFallback(primary, fallback, defaultValue string) string {
	if value := os.Getenv(primary); value != "" {
		return value
	}
	if value := os.Getenv(fallback); value != "" {
		return value
	}
	return defaultValue
}

func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if result, err := strconv.Atoi(value); err == nil {
			return result
		}
	}
	return defaultValue
}

func GetEnvIntWithFallback(primary, fallback string, defaultValue int) int {
	for _, key := range []string{primary, fallback} {
		if value := os.Getenv(key); value != "" {
			if i, err := strconv.Atoi(value); err == nil {
				return i
			}
		}
	}
	return defaultValue
}

func GetEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if result, err := strconv.ParseFloat(value, 64); err == nil {
			return result
		}
	}
	return defaultValue
}

func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func GetEnvBoolWithFallback(primary, fallback string, defaultValue bool) bool {
	for _, key := range []string{primary, fallback} {
		if value := os.Getenv(key); value != "" {
			if b, err := strconv.ParseBool(value); err == nil {
				return b
			}
		}
	}
	return defaultValue
}

func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

// GetEnvSlice parses a comma-separated env var into a string slice.
func GetEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}

func GetEnvSliceWithFallback(primary, fallback string, defaultValue []string) []string {
	for _, key := range []string{primary, fallback} {
		if value := os.Getenv(key); value != "" {
			parts := strings.Split(value, ",")
			result := make([]string, 0, len(parts))
			for _, p := range parts {
				if trimmed := strings.TrimSpace(p); trimmed != "" {
					result = append(result, trimmed)
				}
			}
			if len(result) > 0 {
				return result
			}
		}
	}
	return defaultValue
}
