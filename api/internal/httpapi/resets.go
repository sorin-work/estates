package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/sorin-work/estates/api/internal/auth"
	"github.com/sorin-work/estates/api/internal/mail"
	"github.com/sorin-work/estates/api/internal/store"
)

type resetStore interface {
	CreatePasswordReset(ctx context.Context, userID string, tokenHash []byte, expiresAt time.Time) error
	ResetPassword(ctx context.Context, tokenHash []byte, passwordHash string, now time.Time) (store.User, error)
}

// resetTTL is how long a reset link works.
const resetTTL = 30 * time.Minute

// resetSent is the answer to every reset request, whether or not the email
// has an account.
const resetSent = "If an account exists for that email, a link to reset its password is on its way."

func (s *Server) resetEnabled() bool { return s.mailer != nil && s.appURL != "" }

func errResetOff() error {
	return &apiError{Status: http.StatusNotFound, Code: "not_enabled",
		Message: "password reset is not set up on this server"}
}

func errResetInvalid() error {
	return &apiError{Status: http.StatusBadRequest, Code: "invalid_reset",
		Message: "this link is invalid or has expired, ask for a new one"}
}

// tokenHash is what is stored for a reset token.
func tokenHash(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// forgotPassword starts a reset. It answers the same way whether or not
// the email has an account, and creates and sends the link after it has
// answered, so the time it takes does not tell either.
func (s *Server) forgotPassword(w http.ResponseWriter, r *http.Request) error {
	if !s.resetEnabled() {
		return errResetOff()
	}
	addr := clientAddr(r, s.trustedProxies)
	if ok, wait := s.limits.forgotByAddress.take(limitKey(addr)); !ok {
		return errTooManyAttempts(w, wait)
	}
	var in struct {
		Email string `json:"email"`
	}
	if err := decodeLimit(w, r, &in, authBodyBytes); err != nil {
		return err
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	var v validator
	v.required(email, "email")
	v.maxLen(email, "email", maxEmailLength)
	if err := v.err(); err != nil {
		return err
	}
	if ok, wait := s.limits.forgotByPair.take(limitKey(email, addr)); !ok {
		return errTooManyAttempts(w, wait)
	}

	u, err := s.store.UserByEmail(r.Context(), email)
	switch {
	case errors.Is(err, store.ErrNotFound):
	case err != nil:
		return err
	default:
		// However many addresses ask, an inbox gets a few links an hour.
		if ok, _ := s.limits.resetMailsByEmail.take(limitKey(email)); ok {
			s.background(func(ctx context.Context) { s.sendReset(ctx, u) })
		} else {
			s.log.Warn("password reset mail limited", "user_id", u.ID)
		}
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"message": resetSent})
	return nil
}

// sendReset creates a reset token for u and emails the link. Failures are
// logged without the token.
func (s *Server) sendReset(ctx context.Context, u store.User) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		s.log.Error("reset token", "err", err)
		return
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	if err := s.store.CreatePasswordReset(ctx, u.ID, tokenHash(token), s.now().Add(resetTTL)); err != nil {
		s.log.Error("store reset token", "user_id", u.ID, "err", err)
		return
	}
	// The token goes after # so it never reaches a server log or a
	// Referer header.
	link := s.appURL + "/reset-password#token=" + token
	body := "Someone asked to reset the password of your Estates account, " + u.Email + ".\n\n" +
		"Open this link within 30 minutes to choose a new password:\n\n" + link + "\n\n" +
		"The link works once. If you did not ask for this, ignore this email: your password stays as it is.\n"
	if err := s.mailer.Send(ctx, mail.Message{To: u.Email, Subject: "Reset your Estates password", Body: body}); err != nil {
		s.log.Error("send reset email", "user_id", u.ID, "err", err)
		return
	}
	s.log.Info("password reset email sent", "user_id", u.ID)
}

// resetPassword sets a new password with a token from the email. Every
// session of the account ends, and this one is signed in.
func (s *Server) resetPassword(w http.ResponseWriter, r *http.Request) error {
	if !s.resetEnabled() {
		return errResetOff()
	}
	if ok, wait := s.limits.resetByAddress.take(limitKey(clientAddr(r, s.trustedProxies))); !ok {
		return errTooManyAttempts(w, wait)
	}
	var in struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := decodeLimit(w, r, &in, authBodyBytes); err != nil {
		return err
	}
	if raw, err := base64.RawURLEncoding.DecodeString(in.Token); err != nil || len(raw) != 32 {
		return errResetInvalid()
	}
	var v validator
	if msg := auth.ValidatePassword(in.NewPassword); msg != "" {
		v.check(false, "new_password", msg)
	}
	if err := v.err(); err != nil {
		return err
	}
	hash, err := auth.HashPassword(in.NewPassword)
	if err != nil {
		return err
	}
	u, err := s.store.ResetPassword(r.Context(), tokenHash(in.Token), hash, s.now())
	if errors.Is(err, store.ErrResetInvalid) {
		return errResetInvalid()
	}
	if err != nil {
		return err
	}
	s.log.Info("password reset", "user_id", u.ID)
	return s.issue(w, http.StatusOK, u)
}

// background runs fn after the response, with its own deadline, and lets
// Wait (for tests and shutdown) know when it is done.
func (s *Server) background(fn func(ctx context.Context)) {
	s.jobs.Add(1)
	go func() {
		defer s.jobs.Done()
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		fn(ctx)
	}()
}

// Wait blocks until work started after a response has finished.
func (s *Server) Wait() { s.jobs.Wait() }
