package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config is the explicit runtime contract for Media Orchestrator.
type Config struct {
	APIKey      string
	DatabaseURL string
	Host        string
	Port        int
	MediaConfig string
	StagingRoot string
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" { return value }
	return fallback
}

func Load() (Config, error) {
	c := Config{
		APIKey:      os.Getenv("MEDIA_OPS_API_KEY"),
		DatabaseURL: os.Getenv("MEDIA_OPS_DATABASE_URL"),
		Host:        envOr("MEDIA_OPS_HOST", "127.0.0.1"),
		MediaConfig: os.Getenv("MEDIA_OPS_CONFIG_ROOT"),
		StagingRoot: os.Getenv("MEDIA_OPS_STAGING_ROOT"),
		Port:        8100,
	}
	if raw := os.Getenv("MEDIA_OPS_PORT"); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil || port < 1 || port > 65535 {
			return Config{}, fmt.Errorf("MEDIA_OPS_PORT inválido: %q", raw)
		}
		c.Port = port
	}
	if c.DatabaseURL == "" {
		return Config{}, fmt.Errorf("MEDIA_OPS_DATABASE_URL es obligatorio")
	}
	if c.MediaConfig == "" {
		return Config{}, fmt.Errorf("MEDIA_OPS_CONFIG_ROOT es obligatorio")
	}
	if c.StagingRoot == "" {
		return Config{}, fmt.Errorf("MEDIA_OPS_STAGING_ROOT es obligatorio")
	}
	return c, nil
}
