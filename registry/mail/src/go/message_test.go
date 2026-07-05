package mail

import (
	"strings"
	"testing"
)

func TestMessageValidateAndSanitizeAttachment(t *testing.T) {
	msg := &Message{
		To:      []string{"user@example.com"},
		Subject: "Report",
		Body:    "hello",
		Attachments: []Attachment{
			{Filename: "../report\r\n.txt", Content: []byte("x")},
		},
	}
	if err := msg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if strings.ContainsAny(msg.Attachments[0].Filename, "\r\n/\\") {
		t.Fatalf("attachment filename not sanitized: %q", msg.Attachments[0].Filename)
	}
}

func TestMessageRejectsHeaderInjection(t *testing.T) {
	msg := &Message{To: []string{"user@example.com"}, Subject: "hi\r\nBcc: bad@example.com"}
	if err := msg.Validate(); err == nil {
		t.Fatal("expected CRLF subject rejection")
	}
	msg = &Message{To: []string{"bad\r\n@example.com"}, Subject: "hi"}
	if err := msg.Validate(); err == nil {
		t.Fatal("expected CRLF recipient rejection")
	}
}
