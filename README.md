# Dispatch

**Self-hosted email orchestration for multi-site operators.**

One service. Many sites. Pluggable backends. GDPR built in.

---

## What It Does

Dispatch sits between your web applications and email delivery. Instead of configuring email for every site, you configure it once:

```
Your Sites → Dispatch API (with auto footer) → SMTP / Resend / SES / Listmonk
```

- **Multi-site** — one instance serves all your projects
- **Pluggable backends** — bring your own SMTP, or use Resend, SES, etc.
- **File-based templates** — version-controlled, per-site
- **GDPR compliant** — consent logging, unsubscribe, data export/erasure
- **SQLite by default** — zero external dependencies to start

## Quick Start

```bash
# 1. Initialize
dispatch init

# 2. Add a site
dispatch site add my-site

# 3. Edit config
vim dispatch.yaml
vim sites/my-site/site.yaml

# 4. Start
dispatch serve
```

Or with Docker:

```bash
docker compose up -d
```

## Send an Email

```bash
curl -X POST http://localhost:8080/api/v1/sites/my-site/send \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "user@example.com",
    "template": "welcome",
    "data": { "name": "Alex" }
  }'
```

## Project Structure

```
dispatch/
├── dispatch.yaml            # Main config
├── sites/
│   └── my-site/
│       ├── site.yaml        # Site config (sender, backend, etc.)
│       └── templates/
│           └── welcome.html # Email template
├── shared/
│   └── templates/           # Shared base templates
└── data/
    └── dispatch.db          # SQLite database
```

## Backends

| Backend | Status | Notes |
|---------|--------|-------|
| SMTP | ✅ Ready | Any SMTP server |
| Resend | ✅ Ready | resend.com API |
| SES | 🚧 Planned | AWS SES |
| Listmonk | 🚧 Planned | Self-hosted |

## API Reference

See [SPEC.md](SPEC.md) for the full API specification.

### Send with Pre-Rendered HTML

Send emails with your own HTML templates - Dispatch wraps them in a responsive layout with automatic unsubscribe footer:

```bash
curl -X POST http://localhost:8080/api/v1/sites/my-site/send/raw \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "user@example.com",
    "subject": "Your Daily Reminder",
    "html": "<h1>Time to play!</h1><p>Your daily reminder...</p>",
    "data": { "name": "PlayerOne" }
  }'
```

**Features:**
- Services send raw HTML - no template migration needed
- Dispatch automatically wraps emails in responsive layout
- Configurable unsubscribe footer added to all emails (per-site)
- Plain text version auto-generated with footer included

**Configure Footer in `sites/my-site/site.yaml`:**
```yaml
unsubscribe_footer:
  enabled: true
  text: "To stop receiving emails from {site_name}, click here: {unsubscribe_url}"
  # Available variables: {site_name}, {unsubscribe_url}, {recipient_email}
```

### Core Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/sites/{site}/send` | Send transactional email |
| POST | `/api/v1/sites/{site}/send/batch` | Batch send |
| POST | `/api/v1/sites/{site}/subscribers` | Add subscriber |
| GET | `/api/v1/sites/{site}/subscribers` | List subscribers |
| POST | `/api/v1/gdpr/export` | Export user data |
| POST | `/api/v1/gdpr/forget` | Delete user data |
| GET | `/api/v1/health` | Service health |

## License

TBD — considering AGPL-3.0

---

*Built for self-hosters who run multiple sites and want email that just works.*
