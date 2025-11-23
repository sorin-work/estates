package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sorin-work/estates/api/internal/auth"
	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

// fakeStore implements the parts of Store these tests use. Calling anything
// else panics through the nil embedded interface, which shows up as a 500.
type fakeStore struct {
	Store
	users   map[string]store.User
	pingErr error
}

func newFakeStore() *fakeStore {
	return &fakeStore{users: map[string]store.User{}}
}

func (f *fakeStore) Ping(context.Context) error { return f.pingErr }

func (f *fakeStore) CreateUser(_ context.Context, email, name, hash string) (store.User, error) {
	for _, u := range f.users {
		if strings.EqualFold(u.Email, email) {
			return store.User{}, &store.ConflictError{Msg: "an account with this email already exists"}
		}
	}
	u := store.User{ID: fmt.Sprintf("00000000-0000-0000-0000-%012d", len(f.users)+1), Email: email, Name: name, PasswordHash: hash, CreatedAt: time.Now()}
	f.users[u.ID] = u
	return u, nil
}

func (f *fakeStore) UserByEmail(_ context.Context, email string) (store.User, error) {
	for _, u := range f.users {
		if strings.EqualFold(u.Email, email) {
			return u, nil
		}
	}
	return store.User{}, store.ErrNotFound
}

func (f *fakeStore) UserByID(_ context.Context, id string) (store.User, error) {
	u, ok := f.users[id]
	if !ok {
		return store.User{}, store.ErrNotFound
	}
	return u, nil
}

func newTestServer(t *testing.T, st Store) (*Server, *client) {
	t.Helper()
	srv := New(Options{
		Store:  st,
		Tokens: testTokens(t),
		Logger: quietLogger(),
		Today:  func() domain.Date { return domain.MustParseDate("2026-03-10") },
	})
	return srv, &client{t: t, handler: srv.Handler()}
}

func TestHealth(t *testing.T) {
	st := newFakeStore()
	_, c := newTestServer(t, st)
	c.mustStatus(http.StatusOK, "GET", "/healthz", nil)

	st.pingErr = errors.New("connection refused")
	c.mustStatus(http.StatusServiceUnavailable, "GET", "/healthz", nil)
}

func TestRegisterLoginAndMe(t *testing.T) {
	_, c := newTestServer(t, newFakeStore())

	res := c.mustStatus(http.StatusCreated, "POST", "/api/auth/register", map[string]string{
		"email": "  Landlord@Example.com ", "name": "Lee", "password": "s3cret-pass",
	})
	var reg authResponse
	res.decode(t, &reg)
	if reg.Token == "" || reg.User.Email != "landlord@example.com" {
		t.Fatalf("register response %+v", reg)
	}
	if strings.Contains(string(res.Body), "password") {
		t.Error("response leaks the password hash")
	}

	res = c.mustStatus(http.StatusConflict, "POST", "/api/auth/register", map[string]string{
		"email": "landlord@example.com", "name": "Lee", "password": "s3cret-pass",
	})
	if res.errorCode(t) != "conflict" {
		t.Errorf("duplicate email: %s", res.Body)
	}

	res = c.mustStatus(http.StatusUnauthorized, "POST", "/api/auth/login", map[string]string{
		"email": "landlord@example.com", "password": "wrong-password",
	})
	if res.errorCode(t) != "invalid_credentials" {
		t.Errorf("wrong password: %s", res.Body)
	}
	c.mustStatus(http.StatusUnauthorized, "POST", "/api/auth/login", map[string]string{
		"email": "nobody@example.com", "password": "s3cret-pass",
	})

	res = c.mustStatus(http.StatusOK, "POST", "/api/auth/login", map[string]string{
		"email": "LANDLORD@example.com", "password": "s3cret-pass",
	})
	var login authResponse
	res.decode(t, &login)

	c.mustStatus(http.StatusUnauthorized, "GET", "/api/auth/me", nil)
	c.token = login.Token
	res = c.mustStatus(http.StatusOK, "GET", "/api/auth/me", nil)
	var me store.User
	res.decode(t, &me)
	if me.Name != "Lee" {
		t.Errorf("me %+v", me)
	}
}

func TestRegisterValidation(t *testing.T) {
	_, c := newTestServer(t, newFakeStore())
	res := c.mustStatus(http.StatusUnprocessableEntity, "POST", "/api/auth/register", map[string]string{
		"email": "not-an-email", "password": "short",
	})
	fields := res.errorFields(t)
	for _, f := range []string{"email", "name", "password"} {
		if fields[f] == "" {
			t.Errorf("no error for %s: %v", f, fields)
		}
	}

	res = c.mustStatus(http.StatusBadRequest, "POST", "/api/auth/register", `{"email": `)
	if res.errorCode(t) != "bad_request" {
		t.Errorf("malformed json: %s", res.Body)
	}
	c.mustStatus(http.StatusBadRequest, "POST", "/api/auth/register", "")
}

func TestRequireAuthRejectsBadTokens(t *testing.T) {
	srv, _ := newTestServer(t, newFakeStore())
	h := srv.Handler()

	other, _ := auth.NewTokens("a-completely-different-secret-32-chars", time.Hour)
	foreign, _, _ := other.Issue("someone", "x@example.com")

	for _, header := range []string{"", "Bearer", "Bearer ", "Bearer nonsense", "Basic dXNlcjpwYXNz", "Bearer " + foreign} {
		req := httptest.NewRequest("GET", "/api/auth/me", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%q: status %d", header, rec.Code)
		}
	}
}
