package mail

import (
	"strings"
	"testing"
	"time"
)

func TestBuild(t *testing.T) {
	s := SMTP{From: "Estates <noreply@example.com>"}
	raw, err := s.build(Message{To: "owner@example.com", Subject: "Reset your password", Body: "line one\nline two"},
		time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	for _, want := range []string{
		"From: \"Estates\" <noreply@example.com>\r\n",
		"To: <owner@example.com>\r\n",
		"Subject: Reset your password\r\n",
		"Content-Type: text/plain; charset=utf-8\r\n",
		"\r\n\r\nline one\r\nline two",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
}

func TestBuildRefusesHeaderInjection(t *testing.T) {
	s := SMTP{From: "noreply@example.com"}
	if _, err := s.build(Message{To: "a@example.com", Subject: "Hi\r\nBcc: x@example.com"}, time.Now()); err == nil {
		t.Error("subject with a line break accepted")
	}
	if _, err := s.build(Message{To: "a@example.com\r\nBcc: x@example.com", Subject: "Hi"}, time.Now()); err == nil {
		t.Error("recipient with a line break accepted")
	}
}

func TestSubjectEncoding(t *testing.T) {
	if got := mime("Résumé"); !strings.HasPrefix(got, "=?utf-8?q?R=C3=A9sum=C3=A9") {
		t.Errorf("mime = %q", got)
	}
}
