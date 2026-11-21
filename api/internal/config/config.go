// Package config reads the API configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
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
	// AllowRegistration lets anyone create an account. When false only the
	// first account can be created.
	AllowRegistration bool
	// TrustedProxies are the proxies whose X-Forwarded-For header is
	// believed, as CIDR prefixes.
	TrustedProxies []netip.Prefix
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

		AllowRegistration: true,
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
	if v := os.Getenv("ALLOW_REGISTRATION"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("ALLOW_REGISTRATION: %w", err))
		}
		cfg.AllowRegistration = b
	}
	for _, p := range splitList(os.Getenv("TRUSTED_PROXIES")) {
		prefix, err := parsePrefix(p)
		if err != nil {
			errs = append(errs, fmt.Errorf("TRUSTED_PROXIES: %w", err))
			continue
		}
		cfg.TrustedProxies = append(cfg.TrustedProxies, prefix)
	}
	return cfg, errors.Join(errs...)
}

// parsePrefix reads a CIDR prefix, or a single address as a prefix of one.
func parsePrefix(s string) (netip.Prefix, error) {
	if strings.Contains(s, "/") {
		p, err := netip.ParsePrefix(s)
		return p.Masked(), err
	}
	a, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Prefix{}, err
	}
	return netip.PrefixFrom(a, a.BitLen()), nil
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
