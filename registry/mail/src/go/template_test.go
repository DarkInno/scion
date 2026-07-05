package mail

import (
	"strings"
	"testing"
)

func TestTemplateRenderMessageEscapesHTML(t *testing.T) {
	tm := NewTemplateManager()
	if err := tm.AddTextTemplate("text", "Hello {{.Name}}"); err != nil {
		t.Fatalf("AddTextTemplate: %v", err)
	}
	if err := tm.AddHTMLTemplate("welcome_html", "<p>{{.Name}}</p>"); err != nil {
		t.Fatalf("AddHTMLTemplate: %v", err)
	}
	msg, err := tm.RenderMessage("text", "welcome_html", map[string]string{"Name": "<Ada>"}, []string{"ada@example.com"}, "Hi")
	if err != nil {
		t.Fatalf("RenderMessage: %v", err)
	}
	if msg.Body != "Hello <Ada>" {
		t.Fatalf("text body = %q", msg.Body)
	}
	if strings.Contains(msg.HTML, "<Ada>") || !strings.Contains(msg.HTML, "&lt;Ada&gt;") {
		t.Fatalf("HTML was not escaped: %q", msg.HTML)
	}
	if _, err := tm.RenderMessage("", "", nil, nil, "Hi"); err == nil {
		t.Fatal("expected missing template error")
	}
}
