// Package mail sends plain text email through an SMTP server.
package mail

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

// Message is one plain text email to one address.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Sender delivers messages.
type Sender interface {
	Send(ctx context.Context, m Message) error
}

// SMTP sends through one server. With a username it logs in, which
// net/smtp only does over TLS (STARTTLS) or to localhost.
type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// Send delivers m. The message is built here from fixed headers, so a
// subject or address cannot add headers of its own.
func (s SMTP) Send(ctx context.Context, m Message) error {
	raw, err := s.build(m, time.Now())
	if err != nil {
		return err
	}
	var auth smtp.Auth
	if s.Username != "" {
		auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
	}
	from, _ := mail.ParseAddress(s.From)
	addr := net.JoinHostPort(s.Host, fmt.Sprint(s.Port))

	done := make(chan error, 1)
	go func() { done <- smtp.SendMail(addr, auth, from.Address, []string{m.To}, raw) }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s SMTP) build(m Message, now time.Time) ([]byte, error) {
	from, err := mail.ParseAddress(s.From)
	if err != nil {
		return nil, fmt.Errorf("MAIL_FROM: %w", err)
	}
	to, err := mail.ParseAddress(m.To)
	if err != nil {
		return nil, fmt.Errorf("recipient: %w", err)
	}
	if strings.ContainsAny(m.Subject, "\r\n") {
		return nil, errors.New("subject has a line break")
	}
	var b strings.Builder
	b.WriteString("From: " + from.String() + "\r\n")
	b.WriteString("To: " + to.String() + "\r\n")
	b.WriteString("Subject: " + mime(m.Subject) + "\r\n")
	b.WriteString("Date: " + now.Format(time.RFC1123Z) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(strings.ReplaceAll(strings.ReplaceAll(m.Body, "\r\n", "\n"), "\n", "\r\n"))
	return []byte(b.String()), nil
}

// mime encodes a subject with non-ASCII characters.
func mime(s string) string {
	for _, r := range s {
		if r > 126 {
			return "=?utf-8?q?" + qEncode(s) + "?="
		}
	}
	return s
}

func qEncode(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		switch {
		case c == ' ':
			b.WriteByte('_')
		case c >= '0' && c <= '9', c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "=%02X", c)
		}
	}
	return b.String()
}
