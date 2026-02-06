package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Server
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration

	// TLS-API
	TLSAPIUrl   string
	TLSAPIToken string

	// Provider API Keys
	JeviAPIKey    string
	N4SAPIKey     string
	RoolinkAPIKey string

	// Cache
	CacheEnabled bool

	// Debug
	Debug bool
}

func Load() (*Config, error) {
	cfg := &Config{
		// Defaults
		Port:            getEnvInt("SERVER_PORT", 9999),
		ReadTimeout:     getEnvDuration("SERVER_READ_TIMEOUT", 10*time.Second),
		WriteTimeout:    getEnvDuration("SERVER_WRITE_TIMEOUT", 60*time.Second),
		ShutdownTimeout: getEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 15*time.Second),

		TLSAPIUrl:   getEnv("TLS_API_URL", "http://localhost:8080"),
		TLSAPIToken: getEnv("TLS_API_TOKEN", ""),

		// Provider API Keys (sem defaults - devem ser configurados)
		JeviAPIKey:    getEnv("JEVI_API_KEY", ""),
		N4SAPIKey:     getEnv("N4S_API_KEY", ""),
		RoolinkAPIKey: getEnv("ROOLINK_API_KEY", ""),

		CacheEnabled: getEnvBool("REQS_PROVIDER_CACHE_ENABLE", true),
		Debug:        getEnvBool("DEBUG", false),
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if v := os.Getenv(key); v != "" {
		return v == "1" || v == "true" || v == "yes"
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultValue
}
