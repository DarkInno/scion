package mail

import (
	"bytes"
	"errors"
	"fmt"
	htmpl "html/template"
	"strings"
	ttmpl "text/template"
)

// TemplateManager manages email templates (both text and HTML).
type TemplateManager struct {
	textTemplates *ttmpl.Template
	htmlTemplates *htmpl.Template
}

// NewTemplateManager creates a new template manager.
func NewTemplateManager() *TemplateManager {
	return &TemplateManager{
		textTemplates: ttmpl.New("text"),
		htmlTemplates: htmpl.New("html"),
	}
}

// AddTextTemplate adds or replaces a text template.
func (tm *TemplateManager) AddTextTemplate(name, content string) error {
	_, err := tm.textTemplates.New(name).Parse(content)
	return err
}

// AddHTMLTemplate adds or replaces an HTML template.
// HTML templates use html/template for automatic XSS escaping.
func (tm *TemplateManager) AddHTMLTemplate(name, content string) error {
	_, err := tm.htmlTemplates.New(name).Parse(content)
	return err
}

// RenderText renders a text template with the given data.
func (tm *TemplateManager) RenderText(name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := tm.textTemplates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("text template %s: %w", name, err)
	}
	return buf.String(), nil
}

// RenderHTML renders an HTML template with the given data.
// Uses html/template for automatic XSS prevention.
func (tm *TemplateManager) RenderHTML(name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := tm.htmlTemplates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("HTML template %s: %w", name, err)
	}
	return buf.String(), nil
}

// RenderMessage creates a Message from templates.
func (tm *TemplateManager) RenderMessage(textTemplate, htmlTemplate string, data interface{}, to []string, subject string) (*Message, error) {
	if textTemplate == "" && htmlTemplate == "" {
		return nil, errors.New("at least one template name is required")
	}

	msg := &Message{
		To:      to,
		Subject: subject,
	}

	if textTemplate != "" {
		body, err := tm.RenderText(textTemplate, data)
		if err != nil {
			return nil, err
		}
		msg.Body = body
	}

	if htmlTemplate != "" {
		html, err := tm.RenderHTML(htmlTemplate, data)
		if err != nil {
			return nil, err
		}
		msg.HTML = html
	}

	// Check for CRLF injection in rendered output.
	if strings.ContainsAny(msg.Body, "\r\n") {
		msg.Body = strings.ReplaceAll(msg.Body, "\r", "")
		msg.Body = strings.ReplaceAll(msg.Body, "\n", "\r\n")
	}
	if strings.Contains(msg.Subject, "\r\n") {
		return nil, errors.New("template injected CRLF into subject")
	}

	return msg, nil
}
