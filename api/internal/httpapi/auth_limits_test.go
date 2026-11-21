package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"
)

// post sends a JSON body from the given peer address, with an optional
// X-Forwarded-For header.
func post(t *testing.T, h http.Handler, path, remote, forwarded string, body any) response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = remote
	if forwarded != "" {
		req.Header.Set("X-Forwarded-For", forwarded)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return response{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
}

func login(email, password string) map[string]string {
	return map[string]string{"email": email, "password": password}
}

func TestLoginLimitedByEmail(t *testing.T) {
	srv, c := newTestServer(t, newFakeStore())
	clock := newFakeClock()
	srv.limits.loginByEmail.now = clock.now
	c.mustStatus(http.StatusCreated, "POST", "/api/auth/register", map[string]string{
		"email": "owner@example.com", "name": "Owner", "password": "password123",
	})

	for i := range failuresPerEmail {
		// Each failure from a different address: the email is what counts.
		res := post(t, c.handler, "/api/auth/login", "203.0.113."+itoa(int64(i+1))+":1", "", login("Owner@example.com", "wrong"))
		if res.Status != http.StatusUnauthorized {
			t.Fatalf("failure %d: status %d", i+1, res.Status)
		}
	}
	res := post(t, c.handler, "/api/auth/login", "203.0.113.50:1", "", login("owner@example.com", "password123"))
	if res.Status != http.StatusTooManyRequests || res.errorCode(t) != "too_many_attempts" {
		t.Fatalf("right password while limited: %d %s", res.Status, res.Body)
	}
	if got := res.Header.Get("Retry-After"); got != "180" {
		t.Errorf("Retry-After = %q, want 180", got)
	}

	// An unknown email is limited the same way, so the limit does not tell
	// which accounts exist.
	for range failuresPerEmail {
		post(t, c.handler, "/api/auth/login", "203.0.113.60:1", "", login("nobody@example.com", "wrong"))
	}
	if res := post(t, c.handler, "/api/auth/login", "203.0.113.61:1", "", login("nobody@example.com", "wrong")); res.Status != http.StatusTooManyRequests {
		t.Errorf("unknown email: status %d, want 429", res.Status)
	}

	clock.add(emailWindow / failuresPerEmail)
	res = post(t, c.handler, "/api/auth/login", "203.0.113.50:1", "", login("owner@example.com", "password123"))
	if res.Status != http.StatusOK {
		t.Fatalf("after waiting: %d %s", res.Status, res.Body)
	}
}

func TestLoginSuccessClearsFailures(t *testing.T) {
	_, c := newTestServer(t, newFakeStore())
	c.mustStatus(http.StatusCreated, "POST", "/api/auth/register", map[string]string{
		"email": "owner@example.com", "name": "Owner", "password": "password123",
	})
	for range 3 {
		for range failuresPerEmail - 1 {
			c.mustStatus(http.StatusUnauthorized, "POST", "/api/auth/login", login("owner@example.com", "wrong"))
		}
		c.mustStatus(http.StatusOK, "POST", "/api/auth/login", login("owner@example.com", "password123"))
	}
}

func TestLoginLimitedByAddress(t *testing.T) {
	srv, c := newTestServer(t, newFakeStore())
	clock := newFakeClock()
	srv.limits.loginByAddress.now = clock.now
	for i := range loginsPerAddress {
		res := post(t, c.handler, "/api/auth/login", "203.0.113.7:1", "", login("user"+itoa(int64(i))+"@example.com", "x"))
		if res.Status != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status %d", i+1, res.Status)
		}
	}
	res := post(t, c.handler, "/api/auth/login", "203.0.113.7:1", "", login("fresh@example.com", "x"))
	if res.Status != http.StatusTooManyRequests {
		t.Fatalf("status %d, want 429", res.Status)
	}
	if res.Header.Get("Retry-After") != "3" {
		t.Errorf("Retry-After = %q, want 3", res.Header.Get("Retry-After"))
	}
	if res := post(t, c.handler, "/api/auth/login", "203.0.113.8:1", "", login("fresh@example.com", "x")); res.Status != http.StatusUnauthorized {
		t.Errorf("another address: status %d, want 401", res.Status)
	}
	clock.add(3 * time.Second)
	if res := post(t, c.handler, "/api/auth/login", "203.0.113.7:1", "", login("fresh@example.com", "x")); res.Status != http.StatusUnauthorized {
		t.Errorf("after waiting: status %d, want 401", res.Status)
	}
}

func TestForwardedForOnlyFromTrustedProxy(t *testing.T) {
	limited := func(trusted []netip.Prefix) bool {
		srv := New(Options{Store: newFakeStore(), Tokens: testTokens(t), Logger: quietLogger(), TrustedProxies: trusted})
		h := srv.Handler()
		// Every attempt claims a new client address through the header.
		for i := range loginsPerAddress + 1 {
			res := post(t, h, "/api/auth/login", "172.18.0.3:1", "198.51.100."+itoa(int64(i)), login("a"+itoa(int64(i))+"@example.com", "x"))
			if res.Status == http.StatusTooManyRequests {
				return true
			}
		}
		return false
	}
	if !limited(nil) {
		t.Error("header believed from an untrusted peer")
	}
	if limited([]netip.Prefix{netip.MustParsePrefix("172.16.0.0/12")}) {
		t.Error("clients behind a trusted proxy share one limit")
	}
}

func TestRegistrationClosed(t *testing.T) {
	st := newFakeStore()
	srv := New(Options{Store: st, Tokens: testTokens(t), Logger: quietLogger(), CloseRegistration: true})
	c := &client{t: t, handler: srv.Handler()}

	var cfg struct {
		RegistrationOpen bool `json:"registration_open"`
	}
	c.mustStatus(http.StatusOK, "GET", "/api/auth/config", nil).decode(t, &cfg)
	if !cfg.RegistrationOpen {
		t.Error("the first account should be allowed")
	}
	c.mustStatus(http.StatusCreated, "POST", "/api/auth/register", map[string]string{
		"email": "owner@example.com", "name": "Owner", "password": "password123",
	})

	c.mustStatus(http.StatusOK, "GET", "/api/auth/config", nil).decode(t, &cfg)
	if cfg.RegistrationOpen {
		t.Error("registration still open after the first account")
	}
	res := c.mustStatus(http.StatusForbidden, "POST", "/api/auth/register", map[string]string{
		"email": "stranger@example.com", "name": "Stranger", "password": "password123",
	})
	if res.errorCode(t) != "registration_closed" {
		t.Errorf("code = %q", res.errorCode(t))
	}
	if len(st.users) != 1 {
		t.Errorf("%d users, want 1", len(st.users))
	}
	c.mustStatus(http.StatusOK, "POST", "/api/auth/login", login("owner@example.com", "password123"))
}

func TestRegistrationOpenByDefault(t *testing.T) {
	_, c := newTestServer(t, newFakeStore())
	for _, email := range []string{"a@example.com", "b@example.com"} {
		c.mustStatus(http.StatusCreated, "POST", "/api/auth/register", map[string]string{
			"email": email, "name": "Someone", "password": "password123",
		})
	}
	var cfg struct {
		RegistrationOpen bool `json:"registration_open"`
	}
	c.mustStatus(http.StatusOK, "GET", "/api/auth/config", nil).decode(t, &cfg)
	if !cfg.RegistrationOpen {
		t.Error("registration should be open")
	}
}

func TestRegisterLimitedByAddress(t *testing.T) {
	_, c := newTestServer(t, newFakeStore())
	for i := range signupsPerAddress {
		c.mustStatus(http.StatusCreated, "POST", "/api/auth/register", map[string]string{
			"email": "u" + itoa(int64(i)) + "@example.com", "name": "Someone", "password": "password123",
		})
	}
	res := c.mustStatus(http.StatusTooManyRequests, "POST", "/api/auth/register", map[string]string{
		"email": "one-more@example.com", "name": "Someone", "password": "password123",
	})
	if res.Header.Get("Retry-After") == "" {
		t.Error("no Retry-After")
	}
}
