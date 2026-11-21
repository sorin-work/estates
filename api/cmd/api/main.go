// Command api runs the estates HTTP API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // landlords' time zones work without tzdata installed

	"github.com/sorin-work/estates/api/internal/auth"
	"github.com/sorin-work/estates/api/internal/billing"
	"github.com/sorin-work/estates/api/internal/config"
	"github.com/sorin-work/estates/api/internal/database"
	"github.com/sorin-work/estates/api/internal/httpapi"
	"github.com/sorin-work/estates/api/internal/mail"
	"github.com/sorin-work/estates/api/internal/seed"
	"github.com/sorin-work/estates/api/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	log := newLogger(cfg)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	connectCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	pool, err := database.Connect(connectCtx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool); err != nil {
		return err
	}

	st := store.New(pool)
	bill := billing.New(st, log)

	if cfg.SeedDemo {
		if err := seed.Demo(ctx, st, bill, log); err != nil {
			return fmt.Errorf("seed demo data: %w", err)
		}
	}

	tokens, err := auth.NewTokens(cfg.JWTSecret, cfg.TokenTTL)
	if err != nil {
		return err
	}
	opts := httpapi.Options{
		Store:       st,
		Billing:     bill,
		Tokens:      tokens,
		Logger:      log,
		CORSOrigins: cfg.CORSOrigins,

		CloseRegistration: !cfg.AllowRegistration,
		TrustedProxies:    cfg.TrustedProxies,
	}
	if cfg.MailEnabled() {
		opts.Mailer = mail.SMTP{
			Host: cfg.SMTPHost, Port: cfg.SMTPPort, Username: cfg.SMTPUsername, Password: cfg.SMTPPassword,
			From: cfg.MailFrom,
		}
		opts.AppURL = cfg.AppURL
	}
	api := httpapi.New(opts)

	go bill.Run(ctx, cfg.BillingInterval)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		log.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	err = srv.Shutdown(shutdownCtx)
	// Let emails already being sent finish.
	api.Wait()
	return err
}

func newLogger(cfg config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	if cfg.LogFormat == "text" {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
