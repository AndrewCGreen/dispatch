package cmd

import (
	"fmt"
	"os"
	"path/filepath"
)

func runSite() error {
	if len(os.Args) < 3 {
		fmt.Println("Usage: dispatch site <add|list|remove> [options]")
		return nil
	}

	switch os.Args[2] {
	case "add":
		if len(os.Args) < 4 {
			return fmt.Errorf("usage: dispatch site add <slug>")
		}
		return siteAdd(os.Args[3])
	case "list":
		return siteList()
	case "remove":
		if len(os.Args) < 4 {
			return fmt.Errorf("usage: dispatch site remove <slug>")
		}
		return siteRemove(os.Args[3])
	default:
		return fmt.Errorf("unknown site command: %s", os.Args[2])
	}
}

func siteAdd(slug string) error {
	siteDir := filepath.Join("sites", slug)
	tplDir := filepath.Join(siteDir, "templates")

	if _, err := os.Stat(siteDir); err == nil {
		return fmt.Errorf("site %q already exists", slug)
	}

	if err := os.MkdirAll(tplDir, 0755); err != nil {
		return fmt.Errorf("failed to create site directory: %w", err)
	}

	siteConfig := fmt.Sprintf(`slug: %s
name: %s
from: hello@example.com
from_name: "%s"
reply_to: ""

backend: smtp
backend_config:
  host: localhost
  port: 587
  username: ""
  password: ""

templates_dir: ./templates

double_optin: true
optin_template: confirm-subscription
welcome_template: welcome

api_key: ""

tags: []
`, slug, slug, slug)

	if err := os.WriteFile(filepath.Join(siteDir, "site.yaml"), []byte(siteConfig), 0644); err != nil {
		return fmt.Errorf("failed to write site config: %w", err)
	}

	// Write a sample welcome template
	welcomeTpl := `{{/* subject: Welcome to {{.Site.Name}}! */}}

{{template "_default-base.html" .}}

{{define "content"}}
<h1>Welcome, {{.Data.name}}!</h1>
<p>Thanks for signing up. We're glad to have you.</p>
{{if .Data.login_url}}
<p><a href="{{.Data.login_url}}" class="button">Get Started</a></p>
{{end}}
{{end}}
`

	if err := os.WriteFile(filepath.Join(tplDir, "welcome.html"), []byte(welcomeTpl), 0644); err != nil {
		return fmt.Errorf("failed to write template: %w", err)
	}

	fmt.Printf("✓ Site %q created at %s\n", slug, siteDir)
	fmt.Println("  Next: edit site.yaml with your sender and backend settings")
	return nil
}

func siteList() error {
	entries, err := os.ReadDir("sites")
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No sites directory found. Run 'dispatch init' first.")
			return nil
		}
		return err
	}

	if len(entries) == 0 {
		fmt.Println("No sites configured. Run 'dispatch site add <slug>' to create one.")
		return nil
	}

	fmt.Println("Sites:")
	for _, e := range entries {
		if e.IsDir() {
			fmt.Printf("  • %s\n", e.Name())
		}
	}
	return nil
}

func siteRemove(slug string) error {
	siteDir := filepath.Join("sites", slug)
	if _, err := os.Stat(siteDir); os.IsNotExist(err) {
		return fmt.Errorf("site %q not found", slug)
	}

	fmt.Printf("Remove site %q and all its templates? [y/N] ", slug)
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "y" && confirm != "Y" {
		fmt.Println("Cancelled.")
		return nil
	}

	if err := os.RemoveAll(siteDir); err != nil {
		return fmt.Errorf("failed to remove site: %w", err)
	}
	fmt.Printf("✓ Site %q removed\n", slug)
	return nil
}
