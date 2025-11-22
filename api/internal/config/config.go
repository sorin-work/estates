// Package config reads the API configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
)

type Config struct {
	Addr        string
	DatabaseURL string
	LogLevel    slog.Level
	LogFormat   string
}

// Load reads configuration from the environment.
func Load() (Config, error) {
	cfg := Config{
		Addr:        getenv("ADDR", ":4200"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		LogFormat:   getenv("LOG_FORMAT", "json"),
	}

	var errs []error
	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if err := cfg.LogLevel.UnmarshalText([]byte(getenv("LOG_LEVEL", "info"))); err != nil {
		errs = append(errs, fmt.Errorf("LOG_LEVEL: %w", err))
	}
	return cfg, errors.Join(errs...)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
