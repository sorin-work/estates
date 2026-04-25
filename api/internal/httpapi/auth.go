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
	UserByEmail(ctx context.Context, email string) (store.User, error)
	UserByID(ctx context.Context, id string) (store.User, error)
	UpdateUser(ctx context.Context, id string, in store.UserSettings) error
	SetPassword(ctx context.Context, id, passwordHash string) error
}

type authResponse struct {
	Token     string     `json:"token"`
	ExpiresAt time.Time  `json:"expires_at"`
	User      store.User `json:"user"`
}

func (s *Server) issue(w http.ResponseWriter, status int, u store.User) error {
	token, exp, err := s.tokens.Issue(u.ID, u.Email)
	if err != nil {
		return err
	}
	writeJSON(w, status, authResponse{Token: token, ExpiresAt: exp, User: u})
	return nil
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	trim(&in.Email, &in.Name)
	in.Email = strings.ToLower(in.Email)

	var v validator
	v.required(in.Email, "email")
	v.email(in.Email, "email")
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
	u, err := s.store.CreateUser(r.Context(), in.Email, in.Name, hash)
	if err != nil {
		return err
	}
	s.log.Info("user registered", "user_id", u.ID)
	return s.issue(w, http.StatusCreated, u)
}

// dummyHash is compared against when the email is unknown, so a login for a
// missing account takes as long as one with a wrong password.
var dummyHash, _ = auth.HashPassword("not-a-real-password")

func (s *Server) login(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	invalid := &apiError{Status: http.StatusUnauthorized, Code: "invalid_credentials", Message: "email or password is incorrect"}

	u, err := s.store.UserByEmail(r.Context(), strings.TrimSpace(in.Email))
	if errors.Is(err, store.ErrNotFound) {
		auth.CheckPassword(dummyHash, in.Password)
		return invalid
	}
	if err != nil {
		return err
	}
	if !auth.CheckPassword(u.PasswordHash, in.Password) {
		return invalid
	}
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

	var v validator
	v.required(in.Name, "name")
	v.maxLen(in.Name, "name", 100)
	v.check(domain.OneOf(in.Currency, domain.Currencies), "currency", "is not a supported currency")
	if err := v.err(); err != nil {
		return err
	}
	if err := s.store.UpdateUser(r.Context(), userID(r), in); err != nil {
		return err
	}
	return s.me(w, r)
}

// changePassword needs the current password, so a stolen token alone cannot
// lock the owner out of their account.
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
	if err := s.store.SetPassword(r.Context(), u.ID, hash); err != nil {
		return err
	}
	s.log.Info("password changed", "user_id", u.ID)
	w.WriteHeader(http.StatusNoContent)
	return nil
}
