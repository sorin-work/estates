package mail

import (
	"bufio"
	"context"
	"errors"
	"net"
	"strconv"
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

// fakeServer answers SMTP without STARTTLS and records the message data.
func fakeServer(t *testing.T) (host string, port int, data chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	data = make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		say := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }
		say("220 fake ESMTP")
		var body strings.Builder
		inData := false
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			if inData {
				if line == ".\r\n" {
					inData = false
					data <- body.String()
					say("250 queued")
					continue
				}
				body.WriteString(line)
				continue
			}
			switch cmd := strings.ToUpper(strings.TrimSpace(line)); {
			case strings.HasPrefix(cmd, "EHLO"):
				say("250-fake")
				say("250 8BITMIME")
			case strings.HasPrefix(cmd, "DATA"):
				inData = true
				say("354 go ahead")
			case strings.HasPrefix(cmd, "QUIT"):
				say("221 bye")
				return
			default:
				say("250 ok")
			}
		}
	}()
	addr := ln.Addr().(*net.TCPAddr)
	return "127.0.0.1", addr.Port, data
}

func TestSendRefusesPlaintext(t *testing.T) {
	host, port, _ := fakeServer(t)
	s := SMTP{Host: host, Port: port, From: "noreply@example.com"}
	err := s.Send(context.Background(), Message{To: "owner@example.com", Subject: "Hi", Body: "hello"})
	if !errors.Is(err, ErrNoTLS) {
		t.Fatalf("err = %v, want ErrNoTLS", err)
	}
}

func TestSendPlaintextWhenAllowed(t *testing.T) {
	host, port, data := fakeServer(t)
	s := SMTP{Host: host, Port: port, From: "noreply@example.com", AllowPlaintext: true}
	if err := s.Send(context.Background(), Message{To: "owner@example.com", Subject: "Hi", Body: "hello"}); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-data:
		if !strings.Contains(got, "Subject: Hi") || !strings.Contains(got, "hello") {
			t.Errorf("message %q", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no message")
	}
}

// A server that accepts and never answers must not hold Send past its
// context.
func TestSendGivesUpOnSilentServer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		conn, err := ln.Accept()
		if err == nil {
			defer conn.Close()
			time.Sleep(5 * time.Second)
		}
	}()
	port, _ := strconv.Atoi(strings.TrimPrefix(ln.Addr().String(), "127.0.0.1:"))
	s := SMTP{Host: "127.0.0.1", Port: port, From: "noreply@example.com"}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := s.Send(ctx, Message{To: "owner@example.com", Subject: "Hi", Body: "hello"}); err == nil {
		t.Fatal("Send succeeded against a silent server")
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("Send took %v", d)
	}
}
