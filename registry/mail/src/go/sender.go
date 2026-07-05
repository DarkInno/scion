package mail

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"strings"
	"sync"
	"time"
)

// Sender sends email messages via SMTP.
type Sender struct {
	opts   Options
	queue  chan *Message
	done   chan struct{}
	mu     sync.Mutex
	closed bool
}

// NewSender creates a new SMTP sender with the given options.
func NewSender(opts Options) (*Sender, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	s := &Sender{
		opts:  opts,
		queue: make(chan *Message, opts.QueueSize),
		done:  make(chan struct{}),
	}
	go s.processQueue()
	return s, nil
}

// Send sends a message synchronously.
func (s *Sender) Send(msg *Message) error {
	if s.IsClosed() {
		return errors.New("sender is closed")
	}
	if err := msg.Validate(); err != nil {
		return err
	}
	return s.sendSMTP(msg)
}

// SendAsync enqueues a message for asynchronous sending.
// Returns an error if the queue is full.
func (s *Sender) SendAsync(msg *Message) error {
	if s.IsClosed() {
		return errors.New("sender is closed")
	}
	if err := msg.Validate(); err != nil {
		return err
	}
	select {
	case s.queue <- msg:
		return nil
	default:
		return errors.New("send queue is full")
	}
}

// IsClosed returns true if the sender has been closed.
func (s *Sender) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// Close shuts down the sender, flushing any pending messages.
// After closing, Send and SendAsync return an error.
func (s *Sender) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.mu.Unlock()
	close(s.queue)
	<-s.done
}

// processQueue processes messages from the async queue.
func (s *Sender) processQueue() {
	defer close(s.done)
	for msg := range s.queue {
		if err := s.sendSMTP(msg); err != nil {
			// In production, log this error. We don't log the password.
			_ = err
		}
	}
}

// sendSMTP connects to the SMTP server and sends the message.
func (s *Sender) sendSMTP(msg *Message) error {
	// Build the raw email content.
	raw, err := buildRawMessage(msg, s.opts)
	if err != nil {
		return fmt.Errorf("build message: %w", err)
	}

	addr := net.JoinHostPort(s.opts.Host, fmt.Sprintf("%d", s.opts.Port))

	// Connect with timeout.
	conn, err := net.DialTimeout("tcp", addr, s.opts.Timeout)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", addr, err)
	}
	defer conn.Close()

	// Apply deadline for the entire SMTP conversation.
	// Use 3x the dial timeout to allow for auth + data transfer.
	conversationTimeout := s.opts.Timeout * 3
	if conversationTimeout < 30*time.Second {
		conversationTimeout = 30 * time.Second
	}
	if err := conn.SetDeadline(time.Now().Add(conversationTimeout)); err != nil {
		return fmt.Errorf("set deadline: %w", err)
	}

	var client *smtp.Client

	if s.opts.UseTLS {
		// Implicit TLS (port 465).
		tlsConn := tls.Client(conn, &tls.Config{ServerName: s.opts.Host})
		if err := tlsConn.Handshake(); err != nil {
			return fmt.Errorf("TLS handshake: %w", err)
		}
		client, err = smtp.NewClient(tlsConn, s.opts.Host)
		if err != nil {
			return fmt.Errorf("SMTP client: %w", err)
		}
	} else {
		// Plain connection, optionally upgrade with STARTTLS.
		client, err = smtp.NewClient(conn, s.opts.Host)
		if err != nil {
			return fmt.Errorf("SMTP client: %w", err)
		}
	}

	defer client.Close()

	// STARTTLS upgrade.
	if s.opts.UseSTARTTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: s.opts.Host}); err != nil {
				return fmt.Errorf("STARTTLS: %w", err)
			}
		}
	}

	// Authenticate.
	if s.opts.Username != "" {
		auth := smtp.PlainAuth("", s.opts.Username, s.opts.Password, s.opts.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth: %w", err)
		}
	}

	// Set sender.
	from := msg.From
	if from == "" {
		from = s.opts.From
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}

	// Add recipients.
	for _, to := range msg.To {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("RCPT TO %s: %w", to, err)
		}
	}
	for _, cc := range msg.Cc {
		if err := client.Rcpt(cc); err != nil {
			return fmt.Errorf("RCPT TO (cc) %s: %w", cc, err)
		}
	}
	for _, bcc := range msg.Bcc {
		if err := client.Rcpt(bcc); err != nil {
			return fmt.Errorf("RCPT TO (bcc) %s: %w", bcc, err)
		}
	}

	// Send body.
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	if _, err := w.Write(raw); err != nil {
		return fmt.Errorf("write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}

	return client.Quit()
}

// buildRawMessage constructs the raw RFC 5322 email bytes.
func buildRawMessage(msg *Message, opts Options) ([]byte, error) {
	var buf bytes.Buffer

	// From header.
	from := msg.From
	if from == "" {
		from = opts.From
	}
	fromName := msg.FromName
	if fromName == "" {
		fromName = opts.FromName
	}
	if fromName != "" {
		fmt.Fprintf(&buf, "From: %s <%s>\r\n", escapeHeader(fromName), from)
	} else {
		fmt.Fprintf(&buf, "From: %s\r\n", from)
	}

	// To header.
	fmt.Fprintf(&buf, "To: %s\r\n", strings.Join(msg.To, ", "))

	// Cc header.
	if len(msg.Cc) > 0 {
		fmt.Fprintf(&buf, "Cc: %s\r\n", strings.Join(msg.Cc, ", "))
	}

	// Subject header.
	fmt.Fprintf(&buf, "Subject: %s\r\n", escapeHeader(msg.Subject))

	// Date header.
	fmt.Fprintf(&buf, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))

	// MIME version.
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")

	hasAttachments := len(msg.Attachments) > 0
	hasHTML := msg.HTML != ""

	if hasAttachments {
		// Multipart message with attachments.
		boundary := generateBoundary()
		fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%s\r\n\r\n", boundary)

		// Text/HTML body part.
		if hasHTML {
			altBoundary := generateBoundary()
			fmt.Fprintf(&buf, "--%s\r\n", boundary)
			fmt.Fprintf(&buf, "Content-Type: multipart/alternative; boundary=%s\r\n\r\n", altBoundary)
			fmt.Fprintf(&buf, "--%s\r\n", altBoundary)
			fmt.Fprintf(&buf, "Content-Type: text/plain; charset=UTF-8\r\n")
			fmt.Fprintf(&buf, "Content-Transfer-Encoding: 8bit\r\n\r\n")
			buf.WriteString(msg.Body)
			buf.WriteString("\r\n")
			fmt.Fprintf(&buf, "--%s\r\n", altBoundary)
			fmt.Fprintf(&buf, "Content-Type: text/html; charset=UTF-8\r\n")
			fmt.Fprintf(&buf, "Content-Transfer-Encoding: 8bit\r\n\r\n")
			buf.WriteString(msg.HTML)
			buf.WriteString("\r\n")
			fmt.Fprintf(&buf, "--%s--\r\n", altBoundary)
		} else {
			fmt.Fprintf(&buf, "--%s\r\n", boundary)
			fmt.Fprintf(&buf, "Content-Type: text/plain; charset=UTF-8\r\n")
			fmt.Fprintf(&buf, "Content-Transfer-Encoding: 8bit\r\n\r\n")
			buf.WriteString(msg.Body)
			buf.WriteString("\r\n")
		}

		// Attachments.
		for _, att := range msg.Attachments {
			fmt.Fprintf(&buf, "--%s\r\n", boundary)
			disp := "attachment"
			if att.Inline {
				disp = "inline"
			}
			fmt.Fprintf(&buf, "Content-Disposition: %s; filename=\"%s\"\r\n", disp, escapeHeader(att.Filename))
			fmt.Fprintf(&buf, "Content-Type: application/octet-stream\r\n")
			fmt.Fprintf(&buf, "Content-Transfer-Encoding: base64\r\n\r\n")
			writeBase64(&buf, att.Content)
			buf.WriteString("\r\n")
		}

		fmt.Fprintf(&buf, "--%s--\r\n", boundary)
	} else if hasHTML {
		// Multipart alternative (text + HTML).
		boundary := generateBoundary()
		fmt.Fprintf(&buf, "Content-Type: multipart/alternative; boundary=%s\r\n\r\n", boundary)
		fmt.Fprintf(&buf, "--%s\r\n", boundary)
		fmt.Fprintf(&buf, "Content-Type: text/plain; charset=UTF-8\r\n")
		fmt.Fprintf(&buf, "Content-Transfer-Encoding: 8bit\r\n\r\n")
		buf.WriteString(msg.Body)
		buf.WriteString("\r\n")
		fmt.Fprintf(&buf, "--%s\r\n", boundary)
		fmt.Fprintf(&buf, "Content-Type: text/html; charset=UTF-8\r\n")
		fmt.Fprintf(&buf, "Content-Transfer-Encoding: 8bit\r\n\r\n")
		buf.WriteString(msg.HTML)
		buf.WriteString("\r\n")
		fmt.Fprintf(&buf, "--%s--\r\n", boundary)
	} else {
		// Plain text only.
		fmt.Fprintf(&buf, "Content-Type: text/plain; charset=UTF-8\r\n")
		fmt.Fprintf(&buf, "Content-Transfer-Encoding: 8bit\r\n\r\n")
		buf.WriteString(msg.Body)
	}

	return buf.Bytes(), nil
}

// escapeHeader removes CRLF from header values to prevent injection.
func escapeHeader(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return s
}

// generateBoundary generates a unique MIME boundary.
func generateBoundary() string {
	return fmt.Sprintf("----=_Part_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}

// writeBase64 writes base64-encoded content with line wrapping (76 chars per line).
func writeBase64(w io.Writer, data []byte) {
	const lineLen = 76
	encoded := base64Encode(data)
	for i := 0; i < len(encoded); i += lineLen {
		end := i + lineLen
		if end > len(encoded) {
			end = len(encoded)
		}
		w.Write([]byte(encoded[i:end]))
		w.Write([]byte("\r\n"))
	}
}

// base64Encode encodes data to standard base64.
func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
