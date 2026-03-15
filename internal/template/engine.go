// Package template renders email templates from the filesystem using Go's
// html/template engine.
//
// Templates can be:
//   - Single-file: sites/{site}/templates/welcome.html
//   - Directory-based: sites/{site}/templates/welcome/{subject.txt, body.html, body.txt}
//
// The subject line is extracted from a Go template comment in single-file templates:
//   {{/* subject: Your subject here */}}
//
// Base templates (files prefixed with _) provide shared layouts that other
// templates can inherit via Go's template block/define mechanism.
// Site-specific base templates override shared ones.
package template

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

// TemplateData holds all variables available to templates.
type TemplateData struct {
	Data           map[string]any
	Site           SiteInfo
	Subscriber     SubscriberInfo
	UnsubscribeURL string
	PreferencesURL string
}

// SiteInfo is injected into every template render.
type SiteInfo struct {
	Name string
	URL  string
}

// SubscriberInfo is injected when a subscriber is known.
type SubscriberInfo struct {
	Email      string
	Name       string
	Attributes map[string]any
}

// RenderResult holds the rendered email parts.
type RenderResult struct {
	Subject string
	HTML    string
	Text    string
}

// Engine renders email templates from the filesystem.
type Engine struct {
	sharedDir string // path to shared/templates
}

// New creates a new template engine.
func New(sharedDir string) *Engine {
	return &Engine{sharedDir: sharedDir}
}

// Render renders a template by slug from the given site templates directory.
func (e *Engine) Render(siteTemplatesDir, slug string, data *TemplateData) (*RenderResult, error) {
	// Check if template is a single file or a directory
	singleFile := filepath.Join(siteTemplatesDir, slug+".html")
	dirPath := filepath.Join(siteTemplatesDir, slug)

	var subjectTpl, htmlContent string

	if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
		// Directory-based template
		htmlBytes, err := os.ReadFile(filepath.Join(dirPath, "body.html"))
		if err != nil {
			return nil, fmt.Errorf("reading template body: %w", err)
		}
		htmlContent = string(htmlBytes)

		subjectBytes, err := os.ReadFile(filepath.Join(dirPath, "subject.txt"))
		if err == nil {
			subjectTpl = strings.TrimSpace(string(subjectBytes))
		}
	} else if _, err := os.Stat(singleFile); err == nil {
		// Single-file template
		content, err := os.ReadFile(singleFile)
		if err != nil {
			return nil, fmt.Errorf("reading template: %w", err)
		}
		htmlContent = string(content)

		// Extract subject from comment: {{/* subject: ... */}}
		subjectTpl = extractSubject(htmlContent)
	} else {
		return nil, fmt.Errorf("template %q not found in %s", slug, siteTemplatesDir)
	}

	// Parse and execute
	result := &RenderResult{}

	// Render subject
	if subjectTpl != "" {
		rendered, err := renderString(subjectTpl, data)
		if err != nil {
			return nil, fmt.Errorf("rendering subject: %w", err)
		}
		result.Subject = rendered
	}

	// Parse HTML template with shared templates
	tmpl := template.New(slug + ".html")

	// Load shared base templates
	sharedPattern := filepath.Join(e.sharedDir, "*.html")
	sharedFiles, _ := filepath.Glob(sharedPattern)
	if len(sharedFiles) > 0 {
		if _, err := tmpl.ParseFiles(sharedFiles...); err != nil {
			return nil, fmt.Errorf("parsing shared templates: %w", err)
		}
	}

	// Load site-level base templates
	siteBasePattern := filepath.Join(siteTemplatesDir, "_*.html")
	siteBaseFiles, _ := filepath.Glob(siteBasePattern)
	if len(siteBaseFiles) > 0 {
		if _, err := tmpl.ParseFiles(siteBaseFiles...); err != nil {
			return nil, fmt.Errorf("parsing site base templates: %w", err)
		}
	}

	// Parse the actual template content
	if _, err := tmpl.Parse(htmlContent); err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}
	result.HTML = buf.String()

	// Auto-generate plain text from HTML (basic strip)
	result.Text = stripHTML(result.HTML)

	// Try to load explicit text version
	textFile := filepath.Join(siteTemplatesDir, slug, "body.txt")
	if textBytes, err := os.ReadFile(textFile); err == nil {
		rendered, err := renderString(string(textBytes), data)
		if err == nil {
			result.Text = rendered
		}
	}

	return result, nil
}

// RenderRaw renders inline HTML/text content with data substitution.
func (e *Engine) RenderRaw(subject, html, text string, data *TemplateData) (*RenderResult, error) {
	result := &RenderResult{}

	var err error
	result.Subject, err = renderString(subject, data)
	if err != nil {
		return nil, fmt.Errorf("rendering subject: %w", err)
	}

	result.HTML, err = renderString(html, data)
	if err != nil {
		return nil, fmt.Errorf("rendering HTML: %w", err)
	}

	if text != "" {
		result.Text, err = renderString(text, data)
		if err != nil {
			return nil, fmt.Errorf("rendering text: %w", err)
		}
	} else {
		result.Text = stripHTML(result.HTML)
	}

	return result, nil
}

// extractSubject pulls the subject from a Go template comment.
// Format: {{/* subject: Your Subject Here */}}
func extractSubject(content string) string {
	const prefix = "{{/* subject:"
	const suffix = "*/}}"

	idx := strings.Index(content, prefix)
	if idx == -1 {
		return ""
	}

	start := idx + len(prefix)
	end := strings.Index(content[start:], suffix)
	if end == -1 {
		return ""
	}

	return strings.TrimSpace(content[start : start+end])
}

// renderString renders a Go template string with data.
func renderString(tplStr string, data *TemplateData) (string, error) {
	tmpl, err := template.New("str").Parse(tplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// stripHTML removes HTML tags for a basic plain text version.
func stripHTML(html string) string {
	// Simple tag stripping — not production-grade, but good enough for auto-generated text.
	var result strings.Builder
	inTag := false
	for _, r := range html {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			result.WriteRune(r)
		}
	}

	// Clean up whitespace
	lines := strings.Split(result.String(), "\n")
	var cleaned []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return strings.Join(cleaned, "\n")
}
