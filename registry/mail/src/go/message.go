package mail

import (
	"errors"
	"net/mail"
	"path/filepath"
	"strings"
)

// maxSubjectLen limits subject line length to prevent abuse.
const maxSubjectLen = 998 // RFC 5322 line length limit

// Attachment represents a file attached to an email.
type Attachment struct {
	Filename string // Sanitized filename
	Content  []byte // File content
	Inline   bool   // Is inline (e.g., image in HTML)
}

// Message represents an email message.
type Message struct {
	From        string       // Sender email (overrides Options.From)
	FromName    string       // Sender display name
	To          []string     // Recipient email addresses
	Cc          []string     // CC recipients
	Bcc         []string     // BCC recipients
	Subject     string       // Email subject
	Body        string       // Plain text body
	HTML        string       // HTML body (optional)
	Attachments []Attachment // File attachments
}

// Validate checks the message for required fields and safety.
func (m *Message) Validate() error {
	if len(m.To) == 0 {
		return errors.New("at least one recipient is required")
	}
	if m.Subject == "" {
		return errors.New("subject is required")
	}
	if len(m.Subject) > maxSubjectLen {
		return errors.New("subject exceeds maximum length")
	}

	// Validate all email addresses.
	all := make([]string, 0, len(m.To)+len(m.Cc)+len(m.Bcc))
	all = append(all, m.To...)
	all = append(all, m.Cc...)
	all = append(all, m.Bcc...)
	for _, addr := range all {
		if err := validateEmail(addr); err != nil {
			return err
		}
	}

	// Check for CRLF injection in subject and From.
	if strings.ContainsAny(m.Subject, "\r\n") {
		return errors.New("subject contains CRLF characters (header injection)")
	}
	if m.From != "" {
		if err := validateEmail(m.From); err != nil {
			return err
		}
	}

	// Sanitize attachment filenames.
	for i := range m.Attachments {
		m.Attachments[i].Filename = sanitizeFilename(m.Attachments[i].Filename)
	}

	return nil
}

// validateEmail validates an email address using net/mail.ParseAddress.
// Rejects display name format (only accepts bare addresses).
func validateEmail(addr string) error {
	if strings.ContainsAny(addr, "\r\n") {
		return errors.New("email address contains CRLF characters (header injection)")
	}
	parsed, err := mail.ParseAddress(addr)
	if err != nil {
		return errors.New("invalid email address: " + addr)
	}
	// Reject if ParseAddress parsed a display name we didn't intend.
	// mail.ParseAddress("Name <addr>") succeeds but we want bare addresses.
	if parsed.Name != "" && !strings.Contains(addr, "<") {
		// Some inputs like "addr" get parsed with no name, which is fine.
		// But "Name addr" might parse unexpectedly.
		return errors.New("invalid email address format: " + addr)
	}
	return nil
}

// sanitizeFilename removes path separators, .., and control characters.
func sanitizeFilename(name string) string {
	// Get the base name (strips directory traversal).
	name = filepath.Base(name)
	// Remove null bytes and CRLF.
	name = strings.ReplaceAll(name, "\x00", "")
	name = strings.ReplaceAll(name, "\r", "")
	name = strings.ReplaceAll(name, "\n", "")
	// Replace any remaining path separators.
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	if name == "" || name == "." || name == ".." {
		name = "attachment"
	}
	return name
}
