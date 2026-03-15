package cmd

import (
	"fmt"
	"os"
	"path/filepath"
)

func runInit() error {
	dirs := []string{
		"data",
		"sites",
		"shared/templates",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		fmt.Printf("  created %s/\n", dir)
	}

	// Write default config
	cfgPath := "dispatch.yaml"
	if _, err := os.Stat(cfgPath); err == nil {
		fmt.Printf("  %s already exists, skipping\n", cfgPath)
	} else {
		if err := os.WriteFile(cfgPath, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}
		fmt.Printf("  created %s\n", cfgPath)
	}

	// Write default base template
	baseTpl := filepath.Join("shared", "templates", "_default-base.html")
	if _, err := os.Stat(baseTpl); err != nil {
		if err := os.WriteFile(baseTpl, []byte(defaultBaseTemplate), 0644); err != nil {
			return fmt.Errorf("failed to write base template: %w", err)
		}
		fmt.Printf("  created %s\n", baseTpl)
	}

	fmt.Println("\n✓ Dispatch initialized. Next steps:")
	fmt.Println("  1. Edit dispatch.yaml with your settings")
	fmt.Println("  2. Run 'dispatch site add <name>' to add a site")
	fmt.Println("  3. Run 'dispatch serve' to start")
	return nil
}

const defaultConfig = `# Dispatch — Email Orchestration Service
# Docs: https://github.com/dispatch-email/dispatch

server:
  host: 0.0.0.0
  port: 8080
  base_url: http://localhost:8080

database:
  driver: sqlite
  dsn: ./data/dispatch.db

auth:
  # Generate with: dispatch keys create --type master
  master_key: ""

queue:
  workers: 4
  retry_max: 3
  retry_backoff: "5m,30m,2h"

compliance:
  gdpr_enabled: true
  consent_logging: true
  suppression_global: true

logging:
  level: info
  format: json
`

const defaultBaseTemplate = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 0; background: #f5f5f5; }
    .container { max-width: 600px; margin: 0 auto; padding: 20px; }
    .content { background: #ffffff; border-radius: 8px; padding: 32px; }
    .footer { text-align: center; padding: 20px; color: #888; font-size: 12px; }
    .button { display: inline-block; padding: 12px 24px; background: #2563eb; color: #ffffff; text-decoration: none; border-radius: 6px; font-weight: 600; }
  </style>
</head>
<body>
  <div class="container">
    <div class="content">
      {{block "content" .}}{{end}}
    </div>
    <div class="footer">
      <p>{{.Site.Name}}</p>
      {{if .UnsubscribeURL}}<p><a href="{{.UnsubscribeURL}}">Unsubscribe</a></p>{{end}}
    </div>
  </div>
</body>
</html>
`
