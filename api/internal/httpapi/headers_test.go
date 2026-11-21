package httpapi

import (
	"net/http"
	"testing"
)

func checkSecurityHeaders(t *testing.T, what string, res response) {
	t.Helper()
	want := map[string]string{
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "no-referrer",
		"Cache-Control":           "no-store",
	}
	for k, v := range want {
		if got := res.Header.Get(k); got != v {
			t.Errorf("%s: %s = %q, want %q", what, k, got, v)
		}
	}
}

func TestSecurityHeaders(t *testing.T) {
	_, c := newTestServer(t, newFakeStore())
	checkSecurityHeaders(t, "health", c.mustStatus(http.StatusOK, "GET", "/healthz", nil))
	checkSecurityHeaders(t, "unknown endpoint", c.mustStatus(http.StatusNotFound, "GET", "/api/nothing-here", nil))
	checkSecurityHeaders(t, "no token", c.mustStatus(http.StatusUnauthorized, "GET", "/api/auth/me", nil))
	res := c.mustStatus(http.StatusCreated, "POST", "/api/auth/register", map[string]string{
		"email": "owner@example.com", "name": "Owner", "password": "password123",
	})
	checkSecurityHeaders(t, "register", res)
	var auth authResponse
	res.decode(t, &auth)
	c.token = auth.Token
	// fakeStore has no tenants, so this handler panics and answers 500.
	res = c.mustStatus(http.StatusInternalServerError, "GET", "/api/tenants", nil)
	checkSecurityHeaders(t, "panic", res)
}

func TestSecurityHeadersOnExports(t *testing.T) {
	c := newDBServer(t, "2026-03-10")
	res := c.mustStatus(http.StatusOK, "GET", "/api/rent/balances.csv", nil)
	checkSecurityHeaders(t, "csv", res)
	checkSecurityHeaders(t, "dashboard", c.mustStatus(http.StatusOK, "GET", "/api/dashboard", nil))
}
