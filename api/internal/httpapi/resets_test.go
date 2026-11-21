package httpapi

import (
	"context"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/sorin-work/estates/api/internal/billing"
	"github.com/sorin-work/estates/api/internal/mail"
	"github.com/sorin-work/estates/api/internal/store"
	"github.com/sorin-work/estates/api/internal/testdb"
)

type fakeMailer struct{ sent chan mail.Message }

func (f *fakeMailer) Send(_ context.Context, m mail.Message) error {
	f.sent <- m
	return nil
}

// received returns the messages sent so far.
func (f *fakeMailer) received() []mail.Message {
	var out []mail.Message
	for {
		select {
		case m := <-f.sent:
			out = append(out, m)
		default:
			return out
		}
	}
}

type resetFixture struct {
	srv    *Server
	c      *client
	mailer *fakeMailer
	now    time.Time
}

func newResetServer(t *testing.T) *resetFixture {
	t.Helper()
	st := store.New(testdb.New(t))
	f := &resetFixture{mailer: &fakeMailer{sent: make(chan mail.Message, 20)}, now: time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)}
	f.srv = New(Options{
		Store:   st,
		Billing: billing.New(st, quietLogger()),
		Tokens:  testTokens(t),
		Logger:  quietLogger(),
		Now:     func() time.Time { return f.now },
		Mailer:  f.mailer,
		AppURL:  "https://estates.example.com",
	})
	f.c = &client{t: t, handler: f.srv.Handler()}
	var auth authResponse
	f.c.mustStatus(http.StatusCreated, "POST", "/api/auth/register", map[string]string{
		"email": "owner@example.com", "name": "Owner", "password": "old-password",
	}).decode(t, &auth)
	f.c.token = auth.Token
	return f
}

var tokenInLink = regexp.MustCompile(`https://estates\.example\.com/reset-password#token=([A-Za-z0-9_-]+)`)

// forgot asks for a link and returns the token from the email, or "".
func (f *resetFixture) forgot(t *testing.T, email string) string {
	t.Helper()
	anon := &client{t: t, handler: f.c.handler}
	var body struct {
		Message string `json:"message"`
	}
	anon.mustStatus(http.StatusAccepted, "POST", "/api/auth/forgot", map[string]string{"email": email}).decode(t, &body)
	if body.Message != resetSent {
		t.Errorf("message %q", body.Message)
	}
	f.srv.Wait()
	msgs := f.mailer.received()
	if len(msgs) == 0 {
		return ""
	}
	m := tokenInLink.FindStringSubmatch(msgs[0].Body)
	if len(m) != 2 || msgs[0].To != "owner@example.com" {
		t.Fatalf("unexpected email %+v", msgs[0])
	}
	return m[1]
}

func TestPasswordReset(t *testing.T) {
	f := newResetServer(t)
	token := f.forgot(t, " Owner@Example.com ")
	if token == "" {
		t.Fatal("no email sent")
	}

	anon := &client{t: t, handler: f.c.handler}
	var auth authResponse
	anon.mustStatus(http.StatusOK, "POST", "/api/auth/reset", map[string]string{
		"token": token, "new_password": "new-password",
	}).decode(t, &auth)
	if auth.Token == "" || auth.User.Email != "owner@example.com" {
		t.Errorf("reset answer %+v", auth)
	}

	// Every earlier session has ended; the new password works.
	f.c.mustStatus(http.StatusUnauthorized, "GET", "/api/auth/me", nil)
	anon.mustStatus(http.StatusUnauthorized, "POST", "/api/auth/login", login("owner@example.com", "old-password"))
	anon.mustStatus(http.StatusOK, "POST", "/api/auth/login", login("owner@example.com", "new-password"))

	// The link works once.
	res := anon.mustStatus(http.StatusBadRequest, "POST", "/api/auth/reset", map[string]string{
		"token": token, "new_password": "third-password",
	})
	if res.errorCode(t) != "invalid_reset" {
		t.Errorf("code %q", res.errorCode(t))
	}
}

func TestPasswordResetUnknownEmailLooksTheSame(t *testing.T) {
	f := newResetServer(t)
	if token := f.forgot(t, "nobody@example.com"); token != "" {
		t.Error("an email was sent for an unknown address")
	}
}

func TestPasswordResetExpires(t *testing.T) {
	f := newResetServer(t)
	token := f.forgot(t, "owner@example.com")
	f.now = f.now.Add(resetTTL + time.Second)
	anon := &client{t: t, handler: f.c.handler}
	anon.mustStatus(http.StatusBadRequest, "POST", "/api/auth/reset", map[string]string{
		"token": token, "new_password": "new-password",
	})
	anon.mustStatus(http.StatusOK, "POST", "/api/auth/login", login("owner@example.com", "old-password"))
}

func TestPasswordResetEndsOtherLinks(t *testing.T) {
	f := newResetServer(t)
	first := f.forgot(t, "owner@example.com")
	second := f.forgot(t, "owner@example.com")
	anon := &client{t: t, handler: f.c.handler}
	anon.mustStatus(http.StatusOK, "POST", "/api/auth/reset", map[string]string{"token": second, "new_password": "new-password"})
	anon.mustStatus(http.StatusBadRequest, "POST", "/api/auth/reset", map[string]string{"token": first, "new_password": "other-password"})
}

func TestPasswordResetRefusesBadInput(t *testing.T) {
	f := newResetServer(t)
	anon := &client{t: t, handler: f.c.handler}
	for _, token := range []string{"", "short", "not base64 !!", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"} {
		res := anon.mustStatus(http.StatusBadRequest, "POST", "/api/auth/reset", map[string]string{"token": token, "new_password": "new-password"})
		if res.errorCode(t) != "invalid_reset" {
			t.Errorf("token %q: code %q", token, res.errorCode(t))
		}
	}
	token := f.forgot(t, "owner@example.com")
	res := anon.mustStatus(http.StatusUnprocessableEntity, "POST", "/api/auth/reset", map[string]string{"token": token, "new_password": "short"})
	if res.errorFields(t)["new_password"] == "" {
		t.Error("a short password was not refused")
	}
	anon.mustStatus(http.StatusUnprocessableEntity, "POST", "/api/auth/forgot", map[string]string{"email": ""})
}

func TestPasswordResetLimits(t *testing.T) {
	f := newResetServer(t)
	const attacker, owner = "203.0.113.66:1", "198.51.100.7:1"
	// One address and email: a few requests an hour, then 429.
	for range forgotsPerPair {
		res := post(t, f.c.handler, "/api/auth/forgot", attacker, "", map[string]string{"email": "owner@example.com"})
		if res.Status != http.StatusAccepted {
			t.Fatalf("status %d", res.Status)
		}
	}
	res := post(t, f.c.handler, "/api/auth/forgot", attacker, "", map[string]string{"email": "owner@example.com"})
	if res.Status != http.StatusTooManyRequests || res.Header.Get("Retry-After") == "" {
		t.Fatalf("over the pair limit: %d %v", res.Status, res.Header)
	}
	f.srv.Wait()
	if n := len(f.mailer.received()); n != forgotsPerPair {
		t.Fatalf("%d emails, want %d", n, forgotsPerPair)
	}

	// Someone else spending their budget does not stop the owner's request.
	res = post(t, f.c.handler, "/api/auth/forgot", owner, "", map[string]string{"email": "owner@example.com"})
	if res.Status != http.StatusAccepted {
		t.Fatalf("owner: status %d", res.Status)
	}
	f.srv.Wait()
	if n := len(f.mailer.received()); n != 1 {
		t.Fatalf("owner got %d emails, want 1", n)
	}
}

func TestPasswordResetMailCapPerAccount(t *testing.T) {
	f := newResetServer(t)
	// Many addresses together get at most resetMailsPerEmail links a day
	// sent to one inbox. Past that the answer is the same, without email.
	sent := 0
	for i := range resetMailsPerEmail + 2 {
		addr := "203.0.113." + itoa(int64(i+1)) + ":1"
		res := post(t, f.c.handler, "/api/auth/forgot", addr, "", map[string]string{"email": "owner@example.com"})
		if res.Status != http.StatusAccepted {
			t.Fatalf("request %d: status %d", i+1, res.Status)
		}
		f.srv.Wait()
		sent += len(f.mailer.received())
	}
	if sent != resetMailsPerEmail {
		t.Errorf("%d emails, want %d", sent, resetMailsPerEmail)
	}
}

func TestPasswordResetOffWithoutMail(t *testing.T) {
	_, c := newTestServer(t, newFakeStore())
	var cfg struct {
		PasswordReset bool `json:"password_reset"`
	}
	c.mustStatus(http.StatusOK, "GET", "/api/auth/config", nil).decode(t, &cfg)
	if cfg.PasswordReset {
		t.Error("password reset offered without a mail server")
	}
	res := c.mustStatus(http.StatusNotFound, "POST", "/api/auth/forgot", map[string]string{"email": "owner@example.com"})
	if res.errorCode(t) != "not_enabled" {
		t.Errorf("code %q", res.errorCode(t))
	}
	c.mustStatus(http.StatusNotFound, "POST", "/api/auth/reset", map[string]string{"token": "x", "new_password": "new-password"})
}
