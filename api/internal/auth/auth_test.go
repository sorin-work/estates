package auth

import (
	"strings"
	"testing"
	"time"
)

func TestPasswordHashing(t *testing.T) {
	hash, err := HashPassword("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "correct horse" || !strings.HasPrefix(hash, "$2") {
		t.Fatalf("not a bcrypt hash: %q", hash)
	}
	if !CheckPassword(hash, "correct horse") {
		t.Error("right password rejected")
	}
	if CheckPassword(hash, "wrong horse") {
		t.Error("wrong password accepted")
	}
}

func TestValidatePassword(t *testing.T) {
	if ValidatePassword("short") == "" {
		t.Error("short password accepted")
	}
	if ValidatePassword(strings.Repeat("a", 73)) == "" {
		t.Error("73 byte password accepted")
	}
	if msg := ValidatePassword("long enough"); msg != "" {
		t.Errorf("good password rejected: %s", msg)
	}
}

const secret = "test-secret-that-is-at-least-32-chars"

func TestTokenRoundTrip(t *testing.T) {
	tokens, err := NewTokens(secret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tok, exp, err := tokens.Issue("user-1", "a@example.com", 3)
	if err != nil {
		t.Fatal(err)
	}
	if time.Until(exp) < 59*time.Minute {
		t.Errorf("expiry too soon: %v", exp)
	}
	id, ver, err := tokens.Verify(tok)
	if err != nil || id != "user-1" || ver != 3 {
		t.Fatalf("Verify = %q, %d, %v", id, ver, err)
	}
}

func TestTokenRejections(t *testing.T) {
	tokens, _ := NewTokens(secret, time.Hour)
	tok, _, _ := tokens.Issue("user-1", "a@example.com", 3)

	other, _ := NewTokens("another-secret-that-is-also-32-chars!", time.Hour)
	if _, _, err := other.Verify(tok); err == nil {
		t.Error("token signed with another key accepted")
	}

	if _, _, err := tokens.Verify(tok + "x"); err == nil {
		t.Error("tampered token accepted")
	}
	if _, _, err := tokens.Verify(""); err == nil {
		t.Error("empty token accepted")
	}

	tokens.now = func() time.Time { return time.Now().Add(2 * time.Hour) }
	if _, _, err := tokens.Verify(tok); err == nil {
		t.Error("expired token accepted")
	}
}

func TestNewTokensRequiresLongSecret(t *testing.T) {
	if _, err := NewTokens("short", time.Hour); err == nil {
		t.Error("short secret accepted")
	}
}
