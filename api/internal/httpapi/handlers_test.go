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
	users       map[string]store.User
	pingErr     error
	maintenance map[string]store.MaintenanceRequest
	statusSet   []string
	properties  []store.PropertyInput
}

func newFakeStore() *fakeStore {
	return &fakeStore{users: map[string]store.User{}, maintenance: map[string]store.MaintenanceRequest{}}
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

func (f *fakeStore) CreateProperty(_ context.Context, _ string, in store.PropertyInput) (string, error) {
	f.properties = append(f.properties, in)
	return "11111111-1111-1111-1111-111111111111", nil
}

func (f *fakeStore) GetProperty(_ context.Context, _, id string, _ domain.Date) (store.Property, error) {
	if id != "11111111-1111-1111-1111-111111111111" || len(f.properties) == 0 {
		return store.Property{}, store.ErrNotFound
	}
	in := f.properties[len(f.properties)-1]
	return store.Property{ID: id, Name: in.Name, Kind: in.Kind, AddressLine1: in.AddressLine1, City: in.City}, nil
}

func (f *fakeStore) GetMaintenance(_ context.Context, _, id string) (store.MaintenanceRequest, error) {
	m, ok := f.maintenance[id]
	if !ok {
		return m, store.ErrNotFound
	}
	return m, nil
}

func (f *fakeStore) SetMaintenanceStatus(_ context.Context, _, id, status string, completedOn *domain.Date) error {
	m := f.maintenance[id]
	m.Status = status
	m.CompletedOn = completedOn
	f.maintenance[id] = m
	f.statusSet = append(f.statusSet, status)
	return nil
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

func authed(t *testing.T, st Store) (*Server, *client) {
	t.Helper()
	srv, c := newTestServer(t, st)
	tok, _, err := srv.tokens.Issue("00000000-0000-0000-0000-000000000001", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	c.token = tok
	return srv, c
}

func TestCreatePropertyValidatesAndTrims(t *testing.T) {
	st := newFakeStore()
	_, c := authed(t, st)

	res := c.mustStatus(http.StatusUnprocessableEntity, "POST", "/api/properties", map[string]string{"name": "  ", "kind": "castle"})
	fields := res.errorFields(t)
	if fields["name"] == "" || fields["kind"] == "" || fields["address_line1"] == "" || fields["city"] == "" {
		t.Errorf("fields %v", fields)
	}

	res = c.mustStatus(http.StatusCreated, "POST", "/api/properties", map[string]string{
		"name": " Oak House ", "address_line1": "5 Oak St", "city": "Bend",
	})
	var p store.Property
	res.decode(t, &p)
	if p.Name != "Oak House" || p.Kind != "single_family" {
		t.Errorf("property %+v", p)
	}
}

func TestInvalidIDsAreNotFound(t *testing.T) {
	_, c := authed(t, newFakeStore())
	for _, path := range []string{"/api/properties/123", "/api/leases/abc", "/api/maintenance/1/status", "/api/tenants/%27%20OR%201=1"} {
		method := "GET"
		var body any
		if strings.HasSuffix(path, "/status") {
			method, body = "POST", map[string]string{"status": "open"}
		}
		res := c.do(method, path, body)
		if res.Status != http.StatusNotFound {
			t.Errorf("%s: status %d", path, res.Status)
		}
	}
	res := c.mustStatus(http.StatusNotFound, "GET", "/api/nope", nil)
	if res.errorCode(t) != "not_found" {
		t.Errorf("unknown route: %s", res.Body)
	}
}

func TestMaintenanceStatusWorkflow(t *testing.T) {
	st := newFakeStore()
	id := "22222222-2222-2222-2222-222222222222"
	st.maintenance[id] = store.MaintenanceRequest{ID: id, Title: "Broken heater", Status: domain.MaintenanceOpen}
	_, c := authed(t, st)
	path := "/api/maintenance/" + id + "/status"

	res := c.mustStatus(http.StatusOK, "POST", path, map[string]string{"status": "completed"})
	var m maintenanceView
	res.decode(t, &m)
	if m.Status != "completed" || m.CompletedOn == nil || m.CompletedOn.String() != "2026-03-10" {
		t.Errorf("completed %+v", m)
	}
	if len(m.NextStatuses) != 1 || m.NextStatuses[0] != "open" {
		t.Errorf("next statuses %v", m.NextStatuses)
	}

	res = c.mustStatus(http.StatusUnprocessableEntity, "POST", path, map[string]string{"status": "in_progress"})
	if res.errorCode(t) != "invalid_transition" {
		t.Errorf("completed -> in_progress: %s", res.Body)
	}

	c.mustStatus(http.StatusUnprocessableEntity, "POST", path, map[string]string{"status": "done"})

	res = c.mustStatus(http.StatusOK, "POST", path, map[string]string{"status": "open"})
	res.decode(t, &m)
	if m.CompletedOn != nil {
		t.Error("reopening should clear the completion date")
	}
	if len(st.statusSet) != 2 {
		t.Errorf("status writes %v", st.statusSet)
	}
}

func TestPanicsBecome500(t *testing.T) {
	// fakeStore does not implement ListTenants, so the handler panics.
	_, c := authed(t, newFakeStore())
	res := c.mustStatus(http.StatusInternalServerError, "GET", "/api/tenants", nil)
	if res.errorCode(t) != "internal" {
		t.Errorf("body %s", res.Body)
	}
}
