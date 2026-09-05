package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/sorin-work/estates/api/internal/auth"
)

const testSecret = "httpapi-test-secret-at-least-32-chars"

func testTokens(t *testing.T) *auth.Tokens {
	t.Helper()
	tokens, err := auth.NewTokens(testSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return tokens
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// client sends JSON requests to a handler and decodes the responses.
type client struct {
	t       *testing.T
	handler http.Handler
	token   string
}

type response struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r response) decode(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %s: %v", r.Body, err)
	}
}

// errorCode returns error.code from an error response.
func (r response) errorCode(t *testing.T) string {
	t.Helper()
	var body struct {
		Error apiError `json:"error"`
	}
	r.decode(t, &body)
	return body.Error.Code
}

func (r response) errorFields(t *testing.T) map[string]string {
	t.Helper()
	var body struct {
		Error apiError `json:"error"`
	}
	r.decode(t, &body)
	return body.Error.Fields
}

func (c *client) do(method, path string, body any) response {
	c.t.Helper()
	var rd io.Reader
	if body != nil {
		if s, ok := body.(string); ok {
			rd = bytes.NewBufferString(s)
		} else {
			b, err := json.Marshal(body)
			if err != nil {
				c.t.Fatal(err)
			}
			rd = bytes.NewReader(b)
		}
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	rec := httptest.NewRecorder()
	c.handler.ServeHTTP(rec, req)
	return response{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
}

func (c *client) mustStatus(want int, method, path string, body any) response {
	c.t.Helper()
	res := c.do(method, path, body)
	if res.Status != want {
		c.t.Fatalf("%s %s: status %d, want %d: %s", method, path, res.Status, want, res.Body)
	}
	return res
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
