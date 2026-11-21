package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/estates")
	t.Setenv("JWT_SECRET", strings.Repeat("x", 32))

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":4200" || cfg.LogLevel != slog.LevelInfo || cfg.TokenTTL != 7*24*time.Hour {
		t.Errorf("unexpected defaults %+v", cfg)
	}
	if cfg.SeedDemo || cfg.BillingInterval != time.Hour {
		t.Errorf("unexpected defaults %+v", cfg)
	}
	if !cfg.AllowRegistration || len(cfg.TrustedProxies) != 0 {
		t.Errorf("registration should be open and no proxy trusted: %+v", cfg)
	}
}

func TestLoadSignInSettings(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/estates")
	t.Setenv("JWT_SECRET", strings.Repeat("x", 32))
	t.Setenv("ALLOW_REGISTRATION", "false")
	t.Setenv("TRUSTED_PROXIES", "172.16.0.0/12, 10.0.0.7, fd00::/8")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AllowRegistration {
		t.Error("registration should be closed")
	}
	var got []string
	for _, p := range cfg.TrustedProxies {
		got = append(got, p.String())
	}
	if strings.Join(got, " ") != "172.16.0.0/12 10.0.0.7/32 fd00::/8" {
		t.Errorf("trusted proxies %v", got)
	}

	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8, not-an-address")
	t.Setenv("ALLOW_REGISTRATION", "sometimes")
	_, err = Load()
	if err == nil || !strings.Contains(err.Error(), "TRUSTED_PROXIES") || !strings.Contains(err.Error(), "ALLOW_REGISTRATION") {
		t.Errorf("err = %v", err)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/estates")
	t.Setenv("JWT_SECRET", strings.Repeat("x", 32))
	t.Setenv("ADDR", ":9000")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("SEED_DEMO", "true")
	t.Setenv("BILLING_INTERVAL", "15m")
	t.Setenv("CORS_ORIGINS", "http://localhost:5173, http://localhost:5175")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":9000" || cfg.LogLevel != slog.LevelDebug || !cfg.SeedDemo || cfg.BillingInterval != 15*time.Minute {
		t.Errorf("unexpected %+v", cfg)
	}
	if len(cfg.CORSOrigins) != 2 || cfg.CORSOrigins[1] != "http://localhost:5175" {
		t.Errorf("cors origins %v", cfg.CORSOrigins)
	}
}

func TestLoadReportsAllProblems(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "short")
	t.Setenv("SEED_DEMO", "maybe")

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"DATABASE_URL", "JWT_SECRET", "SEED_DEMO"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %s", err, want)
		}
	}
}
