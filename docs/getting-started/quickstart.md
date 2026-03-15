# Quick Start

Get from zero to sending your first email in under 5 minutes.

---

## Prerequisites

- Dispatch installed ([Installation Guide](installation.md))
- An SMTP server or Resend account for delivery

---

## Step 1: Initialize

```bash
mkdir my-dispatch && cd my-dispatch
dispatch init
```

This creates your project structure:

```
my-dispatch/
├── dispatch.yaml
├── data/
├── sites/
└── shared/
    └── templates/
        └── _default-base.html
```

---

## Step 2: Configure

Edit `dispatch.yaml` with your settings:

```yaml
server:
  host: 0.0.0.0
  port: 8080
  base_url: http://localhost:8080

database:
  driver: sqlite
  dsn: ./data/dispatch.db

auth:
  master_key: "dsp_master_change_me_in_production"

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
```

> ⚠️ **Important:** Change the `master_key` to something unique and secret before deploying.

---

## Step 3: Add a Site

```bash
dispatch site add my-site
```

This creates `sites/my-site/` with a config file and a sample template.

Edit `sites/my-site/site.yaml`:

```yaml
slug: my-site
name: My Website
from: hello@mysite.com
from_name: "My Website"
reply_to: support@mysite.com

# Option A: Direct SMTP
backend: smtp
backend_config:
  host: smtp.gmail.com
  port: "587"
  username: "your-email@gmail.com"
  password: "your-app-password"

# Option B: Resend (comment out SMTP above, uncomment below)
# backend: resend
# backend_config:
#   api_key: "${RESEND_API_KEY}"

templates_dir: ./templates
double_optin: false
api_key: "dsp_site_my_site_secret_key"
```

---

## Step 4: Create a Template

Edit `sites/my-site/templates/welcome.html`:

```html
{{/* subject: Welcome to {{.Site.Name}}, {{.Data.name}}! */}}

<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    body { font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; }
    .button { display: inline-block; padding: 12px 24px; background: #2563eb; color: #fff; text-decoration: none; border-radius: 6px; }
  </style>
</head>
<body>
  <h1>Welcome, {{.Data.name}}!</h1>
  <p>Thanks for joining {{.Site.Name}}. We're glad to have you.</p>

  {{if .Data.login_url}}
  <p><a href="{{.Data.login_url}}" class="button">Get Started</a></p>
  {{end}}

  <hr>
  <p style="color: #888; font-size: 12px;">
    {{.Site.Name}}<br>
    {{if .UnsubscribeURL}}<a href="{{.UnsubscribeURL}}">Unsubscribe</a>{{end}}
  </p>
</body>
</html>
```

---

## Step 5: Start the Server

```bash
dispatch serve
```

You should see:

```json
{"level":"INFO","msg":"dispatch starting","addr":"0.0.0.0:8080","version":"0.1.0-dev"}
{"level":"INFO","msg":"queue started","workers":4}
```

---

## Step 6: Send Your First Email

```bash
curl -X POST http://localhost:8080/api/v1/sites/my-site/send \
  -H "Authorization: Bearer dsp_site_my_site_secret_key" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "you@example.com",
    "template": "welcome",
    "data": {
      "name": "Alex",
      "login_url": "https://mysite.com/dashboard"
    }
  }'
```

**Expected response (202 Accepted):**

```json
{
  "id": "msg_a1b2c3d4e5f6",
  "status": "queued",
  "to": "you@example.com",
  "template": "welcome",
  "site": "my-site",
  "queued_at": "2026-03-14T23:50:00Z"
}
```

---

## Step 7: Verify

Check if the email was delivered:

```bash
# Check message status
curl http://localhost:8080/api/v1/messages/msg_a1b2c3d4e5f6/status \
  -H "Authorization: Bearer dsp_master_change_me_in_production"
```

Check service health:

```bash
curl http://localhost:8080/api/v1/health
```

---

## What's Next?

- [Configuration Reference](configuration.md) — All config options explained
- [Templates Guide](../concepts/templates.md) — Advanced template features
- [Adding More Sites](../concepts/sites.md) — Multi-site setup
- [Deployment Guide](../operations/deployment.md) — Production deployment
- [Python SDK](../sdks/python.md) — Integrate with your Python apps
