package mail

import (
	"strings"
	"testing"
	"time"
)

func TestMessageValidate(t *testing.T) {
	tests := []struct {
		name    string
		msg     *Message
		wantErr bool
	}{
		{
			name: "valid message",
			msg: &Message{
				To:      []string{"user@example.com"},
				Subject: "Test",
				Body:    "Hello",
			},
			wantErr: false,
		},
		{
			name: "no recipients",
			msg: &Message{
				Subject: "Test",
				Body:    "Hello",
			},
			wantErr: true,
		},
		{
			name: "no subject",
			msg: &Message{
				To:   []string{"user@example.com"},
				Body: "Hello",
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			msg: &Message{
				To:      []string{"not-an-email"},
				Subject: "Test",
				Body:    "Hello",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"file.txt", "file.txt"},
		{"../../etc/passwd", "passwd"},
		{"path/to/file.txt", "file.txt"},
		{"file\x00name.txt", "filename.txt"},
		{"file\r\nname.txt", "filename.txt"},
		{"", "attachment"},
		{".", "attachment"},
		{"..", "attachment"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTemplateManager(t *testing.T) {
	tm := NewTemplateManager()

	err := tm.AddTextTemplate("welcome", "Hello {{.Name}}, welcome to {{.Service}}!")
	if err != nil {
		t.Fatalf("failed to add text template: %v", err)
	}

	err = tm.AddHTMLTemplate("welcome_html", "<p>Hello <b>{{.Name}}</b>, welcome to {{.Service}}!</p>")
	if err != nil {
		t.Fatalf("failed to add HTML template: %v", err)
	}

	data := struct {
		Name    string
		Service string
	}{
		Name:    "Alice",
		Service: "Scion",
	}

	text, err := tm.RenderText("welcome", data)
	if err != nil {
		t.Fatalf("failed to render text: %v", err)
	}
	if text != "Hello Alice, welcome to Scion!" {
		t.Errorf("unexpected text: %s", text)
	}

	html, err := tm.RenderHTML("welcome_html", data)
	if err != nil {
		t.Fatalf("failed to render HTML: %v", err)
	}
	if !strings.Contains(html, "<b>Alice</b>") {
		t.Errorf("unexpected HTML: %s", html)
	}
}

func TestTemplateXSSEscaping(t *testing.T) {
	tm := NewTemplateManager()
	_ = tm.AddHTMLTemplate("test", "<div>{{.Input}}</div>")

	data := struct {
		Input string
	}{
		Input: `<script>alert("xss")</script>`,
	}

	html, err := tm.RenderHTML("test", data)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	// html/template should escape the script tag.
	if strings.Contains(html, "<script>") {
		t.Error("HTML template did not escape XSS payload")
	}
}

func TestOptionsDefaults(t *testing.T) {
	o := Defaults()
	if o.Port != 587 {
		t.Errorf("expected default port 587, got %d", o.Port)
	}
	if o.Timeout != 10*time.Second {
		t.Errorf("expected default timeout 10s, got %v", o.Timeout)
	}
	if o.QueueSize != 100 {
		t.Errorf("expected default queue size 100, got %d", o.QueueSize)
	}
	if !o.UseSTARTTLS {
		t.Error("expected STARTTLS enabled by default")
	}
}

func TestOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		wantErr bool
	}{
		{"valid", Options{Host: "smtp.example.com", Port: 587, From: "from@example.com", Timeout: 10 * time.Second, QueueSize: 100}, false},
		{"no host", Options{Port: 587, From: "from@example.com", Timeout: 10 * time.Second, QueueSize: 100}, true},
		{"invalid port", Options{Host: "smtp.example.com", Port: 0, From: "from@example.com", Timeout: 10 * time.Second, QueueSize: 100}, true},
		{"no from", Options{Host: "smtp.example.com", Port: 587, Timeout: 10 * time.Second, QueueSize: 100}, true},
		{"negative timeout", Options{Host: "smtp.example.com", Port: 587, From: "from@example.com", Timeout: -1, QueueSize: 100}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildRawMessage(t *testing.T) {
	msg := &Message{
		To:      []string{"user@example.com"},
		Subject: "Test Subject",
		Body:    "Hello World",
	}
	opts := Defaults()
	opts.From = "from@example.com"

	raw, err := buildRawMessage(msg, opts)
	if err != nil {
		t.Fatalf("buildRawMessage failed: %v", err)
	}

	s := string(raw)
	if !strings.Contains(s, "From: from@example.com") {
		t.Error("missing From header")
	}
	if !strings.Contains(s, "To: user@example.com") {
		t.Error("missing To header")
	}
	if !strings.Contains(s, "Subject: Test Subject") {
		t.Error("missing Subject header")
	}
	if !strings.Contains(s, "Hello World") {
		t.Error("missing body")
	}
}

func TestBuildRawMessageWithHTML(t *testing.T) {
	msg := &Message{
		To:      []string{"user@example.com"},
		Subject: "HTML Test",
		Body:    "Plain text",
		HTML:    "<p>HTML content</p>",
	}
	opts := Defaults()
	opts.From = "from@example.com"

	raw, err := buildRawMessage(msg, opts)
	if err != nil {
		t.Fatalf("buildRawMessage failed: %v", err)
	}

	s := string(raw)
	if !strings.Contains(s, "multipart/alternative") {
		t.Error("missing multipart/alternative content type")
	}
	if !strings.Contains(s, "Plain text") {
		t.Error("missing plain text body")
	}
	if !strings.Contains(s, "<p>HTML content</p>") {
		t.Error("missing HTML body")
	}
}

func TestBuildRawMessageWithAttachment(t *testing.T) {
	msg := &Message{
		To:      []string{"user@example.com"},
		Subject: "Attachment Test",
		Body:    "See attachment",
		Attachments: []Attachment{
			{Filename: "test.txt", Content: []byte("file content")},
		},
	}
	opts := Defaults()
	opts.From = "from@example.com"

	raw, err := buildRawMessage(msg, opts)
	if err != nil {
		t.Fatalf("buildRawMessage failed: %v", err)
	}

	s := string(raw)
	if !strings.Contains(s, "multipart/mixed") {
		t.Error("missing multipart/mixed content type")
	}
	if !strings.Contains(s, "test.txt") {
		t.Error("missing attachment filename")
	}
}
