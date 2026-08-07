package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config is the small, explicit runtime contract for the local MVP.
type Config struct {
	APIKey      string
	DatabaseURL string
	Port        int
	MediaConfig string
	StagingRoot string
}

func Load() (Config, error) {
	c := Config{
		APIKey:      os.Getenv("PRA_API_KEY"),
		DatabaseURL: os.Getenv("PRA_DATABASE_URL"),
		MediaConfig: os.Getenv("PRA_MEDIA_CONFIG_ROOT"),
		StagingRoot: os.Getenv("PRA_STAGING_ROOT"),
		Port:        8093,
	}
	if raw := os.Getenv("PRA_PORT"); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil || port < 1 || port > 65535 {
			return Config{}, fmt.Errorf("PRA_PORT inválido: %q", raw)
		}
		c.Port = port
	}
	if c.DatabaseURL == "" {
		return Config{}, fmt.Errorf("PRA_DATABASE_URL es obligatorio")
	}
	if c.MediaConfig == "" {
		return Config{}, fmt.Errorf("PRA_MEDIA_CONFIG_ROOT es obligatorio")
	}
	if c.StagingRoot == "" {
		return Config{}, fmt.Errorf("PRA_STAGING_ROOT es obligatorio")
	}
	return c, nil
}

