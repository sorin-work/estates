// Package mail sends plain text email through an SMTP server.
package mail

import (
	"context"
	"crypto/tls"
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

// SMTP sends through one server. It upgrades the connection with STARTTLS
// and refuses to send without it, unless AllowPlaintext is set (for a mail
// catcher on the same machine). With a username it logs in.
type SMTP struct {
	Host           string
	Port           int
	Username       string
	Password       string
	From           string
	AllowPlaintext bool
}

// dialTimeout bounds connecting when the context allows longer.
const dialTimeout = 10 * time.Second

// ErrNoTLS means the server does not offer STARTTLS and plain text is not
// allowed.
var ErrNoTLS = errors.New("the mail server does not offer STARTTLS; set SMTP_ALLOW_PLAINTEXT=true to send without it")

// Send delivers m. The message is built here from fixed headers, so a
// subject or address cannot add headers of its own. The whole exchange
// stops when ctx ends, so a server that never answers cannot hold a
// connection open.
func (s SMTP) Send(ctx context.Context, m Message) error {
	raw, err := s.build(m, time.Now())
	if err != nil {
		return err
	}
	from, _ := mail.ParseAddress(s.From)
	to, _ := mail.ParseAddress(m.To)

	d := net.Dialer{Timeout: dialTimeout}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(s.Host, fmt.Sprint(s.Port)))
	if err != nil {
		return err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.SetDeadline(time.Now()) })
	defer stop()

	c, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		conn.Close()
		return err
	}
	defer c.Close()
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	} else if !s.AllowPlaintext {
		return ErrNoTLS
	}
	if s.Username != "" {
		if err := c.Auth(plainAuth{username: s.Username, password: s.Password}); err != nil {
			return err
		}
	}
	if err := c.Mail(from.Address); err != nil {
		return err
	}
	if err := c.Rcpt(to.Address); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(raw); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// plainAuth is PLAIN authentication. Unlike smtp.PlainAuth it does not
// check for TLS itself: Send has already required TLS unless plain text was
// allowed on purpose.
type plainAuth struct{ username, password string }

func (a plainAuth) Start(*smtp.ServerInfo) (string, []byte, error) {
	return "PLAIN", []byte("\x00" + a.username + "\x00" + a.password), nil
}

func (a plainAuth) Next(_ []byte, more bool) ([]byte, error) {
	if more {
		return nil, errors.New("unexpected server challenge")
	}
	return nil, nil
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
