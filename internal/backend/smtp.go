package backend

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

func init() {
	Register("smtp", NewSMTP)
}

// SMTP implements the Backend interface for direct SMTP delivery.
type SMTP struct {
	host     string
	port     string
	username string
	password string
	tls      bool
}

// NewSMTP creates a new SMTP backend.
func NewSMTP(cfg map[string]string) (Backend, error) {
	host := cfg["host"]
	if host == "" {
		host = "localhost"
	}
	port := cfg["port"]
	if port == "" {
		port = "587"
	}

	return &SMTP{
		host:     host,
		port:     port,
		username: cfg["username"],
		password: cfg["password"],
		tls:      cfg["tls"] == "true",
	}, nil
}

func (s *SMTP) Name() string { return "smtp" }

func (s *SMTP) Send(ctx context.Context, msg *OutgoingMessage) (*SendResult, error) {
	addr := net.JoinHostPort(s.host, s.port)

	// Build the email
	var body strings.Builder
	body.WriteString(fmt.Sprintf("From: %s <%s>\r\n", msg.FromName, msg.From))
	body.WriteString(fmt.Sprintf("To: %s\r\n", msg.To))
	body.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	body.WriteString("MIME-Version: 1.0\r\n")

	// Custom headers
	for k, v := range msg.Headers {
		body.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}

	body.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	body.WriteString("\r\n")
	body.WriteString(msg.HTML)

	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	var err error
	if s.tls {
		err = s.sendTLS(addr, auth, msg.From, msg.To, body.String())
	} else {
		err = smtp.SendMail(addr, auth, msg.From, []string{msg.To}, []byte(body.String()))
	}

	if err != nil {
		return nil, fmt.Errorf("SMTP send failed: %w", err)
	}

	return &SendResult{
		BackendID: msg.MessageID,
		Status:    "sent",
	}, nil
}

func (s *SMTP) sendTLS(addr string, auth smtp.Auth, from, to, body string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: s.host})
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return err
	}
	defer client.Quit()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}

	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}

	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(body))
	if err != nil {
		return err
	}
	return w.Close()
}

func (s *SMTP) BatchSend(ctx context.Context, msgs []*OutgoingMessage) (*BatchResult, error) {
	result := &BatchResult{}
	for i, msg := range msgs {
		if _, err := s.Send(ctx, msg); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, BatchError{Index: i, Email: msg.To, Error: err})
		} else {
			result.Succeeded++
		}
	}
	return result, nil
}

func (s *SMTP) Health(ctx context.Context) error {
	addr := net.JoinHostPort(s.host, s.port)
	conn, err := net.DialTimeout("tcp", addr, 5*1e9)
	if err != nil {
		return fmt.Errorf("SMTP connection failed: %w", err)
	}
	conn.Close()
	return nil
}

func (s *SMTP) MaxBatchSize() int { return 0 }
