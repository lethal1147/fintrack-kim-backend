package emailclient

import (
	"context"
	"crypto/tls"
	"fmt"

	gomail "gopkg.in/gomail.v2"
)

// Sender is the interface for sending email.
type Sender interface {
	Send(ctx context.Context, to, subject, html string) error
}

// New returns an SMTPClient when host is non-empty; otherwise a logSender that
// prints the email to stdout (dev/local mode — no SMTP config required).
func New(host string, port int, username, password, from string) Sender {
	if host == "" {
		return &logSender{from: from}
	}
	return &SMTPClient{host: host, port: port, username: username, password: password, from: from}
}

// ─── SMTP client ──────────────────────────────────────────────────────────────

type SMTPClient struct {
	host, username, password, from string
	port                           int
}

func (c *SMTPClient) Send(_ context.Context, to, subject, html string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", c.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", html)

	d := gomail.NewDialer(c.host, c.port, c.username, c.password)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: false, ServerName: c.host}
	return d.DialAndSend(m)
}

// ─── Log sender (dev mode) ────────────────────────────────────────────────────

type logSender struct{ from string }

func (s *logSender) Send(_ context.Context, to, subject, html string) error {
	fmt.Printf("[emailclient] from=%s to=%s subject=%q body=%s\n", s.from, to, subject, html)
	return nil
}
