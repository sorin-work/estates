// Package config reads the API configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr        string
	DatabaseURL string
	JWTSecret   string
	TokenTTL    time.Duration
	LogLevel    slog.Level
	LogFormat   string
	// CORSOrigins lists browser origins allowed to call the API directly.
	// Not needed when the web app proxies /api.
	CORSOrigins []string
	SeedDemo    bool
	// BillingInterval is how often rent charges and late fees are generated.
	BillingInterval time.Duration
}

// Load reads configuration from the environment.
func Load() (Config, error) {
	cfg := Config{
		Addr:            getenv("ADDR", ":4200"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		LogFormat:       getenv("LOG_FORMAT", "json"),
		CORSOrigins:     splitList(os.Getenv("CORS_ORIGINS")),
		TokenTTL:        7 * 24 * time.Hour,
		BillingInterval: time.Hour,
	}

	var errs []error
	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if len(cfg.JWTSecret) < 32 {
		errs = append(errs, errors.New("JWT_SECRET must be at least 32 characters"))
	}
	if err := cfg.LogLevel.UnmarshalText([]byte(getenv("LOG_LEVEL", "info"))); err != nil {
		errs = append(errs, fmt.Errorf("LOG_LEVEL: %w", err))
	}
	if v := os.Getenv("TOKEN_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("TOKEN_TTL: %w", err))
		}
		cfg.TokenTTL = d
	}
	if v := os.Getenv("BILLING_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			errs = append(errs, fmt.Errorf("BILLING_INTERVAL: invalid duration %q", v))
		}
		cfg.BillingInterval = d
	}
	if v := os.Getenv("SEED_DEMO"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("SEED_DEMO: %w", err))
		}
		cfg.SeedDemo = b
	}
	return cfg, errors.Join(errs...)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
