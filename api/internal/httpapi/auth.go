package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/sorin-work/estates/api/internal/auth"
	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

type userStore interface {
	CreateUser(ctx context.Context, email, name, passwordHash string) (store.User, error)
	CreateFirstUser(ctx context.Context, email, name, passwordHash string) (store.User, error)
	HasUsers(ctx context.Context) (bool, error)
	UserByEmail(ctx context.Context, email string) (store.User, error)
	UserByID(ctx context.Context, id string) (store.User, error)
	UpdateUser(ctx context.Context, id string, in store.UserSettings) error
	SetPassword(ctx context.Context, id, passwordHash string) (int, error)
}

type authResponse struct {
	Token     string     `json:"token"`
	ExpiresAt time.Time  `json:"expires_at"`
	User      store.User `json:"user"`
}

func (s *Server) issue(w http.ResponseWriter, status int, u store.User) error {
	token, exp, err := s.tokens.Issue(u.ID, u.Email, u.TokenVersion)
	if err != nil {
		return err
	}
	writeJSON(w, status, authResponse{Token: token, ExpiresAt: exp, User: u})
	return nil
}

// registrationOpen reports whether a new account may be created. With
// registration closed the first account can still be made, so a fresh
// install is not locked out.
func (s *Server) registrationOpen(ctx context.Context) (bool, error) {
	if !s.closeRegistration {
		return true, nil
	}
	exists, err := s.store.HasUsers(ctx)
	return !exists, err
}

// authConfig tells the sign-in pages what they may offer.
func (s *Server) authConfig(w http.ResponseWriter, r *http.Request) error {
	open, err := s.registrationOpen(r.Context())
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]bool{"registration_open": open})
	return nil
}

func errRegistrationClosed() error {
	return &apiError{Status: http.StatusForbidden, Code: "registration_closed",
		Message: "this server does not take new accounts"}
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) error {
	if ok, wait := s.limits.signupByAddress.take(limitKey(clientAddr(r, s.trustedProxies))); !ok {
		return errTooManyAttempts(w, wait)
	}
	open, err := s.registrationOpen(r.Context())
	if err != nil {
		return err
	}
	if !open {
		return errRegistrationClosed()
	}

	var in struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := decodeLimit(w, r, &in, authBodyBytes); err != nil {
		return err
	}
	trim(&in.Email, &in.Name)
	in.Email = strings.ToLower(in.Email)

	var v validator
	v.required(in.Email, "email")
	v.email(in.Email, "email")
	v.maxLen(in.Email, "email", maxEmailLength)
	v.required(in.Name, "name")
	v.maxLen(in.Name, "name", 100)
	if msg := auth.ValidatePassword(in.Password); msg != "" {
		v.check(false, "password", msg)
	}
	if err := v.err(); err != nil {
		return err
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return err
	}
	create := s.store.CreateUser
	if s.closeRegistration {
		// Two sign-ups racing on a fresh install must not both get in.
		create = s.store.CreateFirstUser
	}
	u, err := create(r.Context(), in.Email, in.Name, hash)
	if errors.Is(err, store.ErrNotFirstUser) {
		return errRegistrationClosed()
	}
	if err != nil {
		return err
	}
	s.log.Info("user registered", "user_id", u.ID)
	return s.issue(w, http.StatusCreated, u)
}

// dummyHash is compared against when the email is unknown, so a login for a
// missing account takes as long as one with a wrong password.
var dummyHash, _ = auth.HashPassword("not-a-real-password")

// maxEmailLength is the longest email address that can exist.
const maxEmailLength = 254

// authBodyBytes is plenty for an email, a name and a password.
const authBodyBytes = 4 << 10

// login is limited by address, counting every attempt, and by email and
// address together, counting wrong passwords until one is right. Someone
// guessing from one address therefore never stops the owner signing in from
// another. Wrong passwords for an email from all addresses are counted too;
// past a point, an address new to that email starts with one attempt. The
// limits work the same for emails with no account, so they do not reveal
// which exist.
func (s *Server) login(w http.ResponseWriter, r *http.Request) error {
	addr := clientAddr(r, s.trustedProxies)
	if ok, wait := s.limits.loginByAddress.take(limitKey(addr)); !ok {
		s.log.Warn("login limited", "by", "address", "addr", addr)
		return errTooManyAttempts(w, wait)
	}
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeLimit(w, r, &in, authBodyBytes); err != nil {
		return err
	}
	invalid := &apiError{Status: http.StatusUnauthorized, Code: "invalid_credentials", Message: "email or password is incorrect"}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if len(email) > maxEmailLength {
		auth.CheckPassword(dummyHash, in.Password)
		return invalid
	}

	emailKey, pairKey := limitKey(email), limitKey(email, addr)
	start := failuresPerPair
	if !s.limits.failuresByEmail.has(emailKey) {
		start = 1
	}
	if ok, wait := s.limits.loginByPair.takeFrom(pairKey, start); !ok {
		s.log.Warn("login limited", "by", "email", "addr", addr)
		return errTooManyAttempts(w, wait)
	}
	failed := func() error {
		s.limits.failuresByEmail.take(emailKey)
		return invalid
	}

	u, err := s.store.UserByEmail(r.Context(), email)
	if errors.Is(err, store.ErrNotFound) {
		auth.CheckPassword(dummyHash, in.Password)
		return failed()
	}
	if err != nil {
		return err
	}
	if !auth.CheckPassword(u.PasswordHash, in.Password) {
		return failed()
	}
	s.limits.loginByPair.reset(pairKey)
	return s.issue(w, http.StatusOK, u)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) error {
	u, err := s.store.UserByID(r.Context(), userID(r))
	if errors.Is(err, store.ErrNotFound) {
		return &apiError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "account no longer exists"}
	}
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, u)
	return nil
}

// updateMe changes the signed in user's name and display currency.
func (s *Server) updateMe(w http.ResponseWriter, r *http.Request) error {
	var in store.UserSettings
	if err := decode(w, r, &in); err != nil {
		return err
	}
	trim(&in.Name, &in.Currency)
	in.Currency = strings.ToUpper(in.Currency)
	if in.Currency == "" {
		in.Currency = "USD"
	}
	if in.DepositReturnDays == 0 {
		in.DepositReturnDays = domain.DefaultDepositReturnDays
	}

	var v validator
	v.required(in.Name, "name")
	v.maxLen(in.Name, "name", 100)
	v.check(domain.OneOf(in.Currency, domain.Currencies), "currency", "is not a supported currency")
	v.check(in.DepositReturnDays >= 1 && in.DepositReturnDays <= 90, "deposit_return_days", "must be between 1 and 90")
	if err := v.err(); err != nil {
		return err
	}
	if err := s.store.UpdateUser(r.Context(), userID(r), in); err != nil {
		return err
	}
	return s.me(w, r)
}

// changePassword needs the current password, so a stolen token alone cannot
// lock the owner out of their account. Every other session is signed out;
// this one gets a new token.
func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	u, err := s.store.UserByID(r.Context(), userID(r))
	if err != nil {
		return err
	}

	var v validator
	v.check(auth.CheckPassword(u.PasswordHash, in.CurrentPassword), "current_password", "is incorrect")
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
	if u.TokenVersion, err = s.store.SetPassword(r.Context(), u.ID, hash); err != nil {
		return err
	}
	s.log.Info("password changed", "user_id", u.ID)
	return s.issue(w, http.StatusOK, u)
}
