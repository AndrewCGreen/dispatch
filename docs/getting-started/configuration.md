# Configuration Reference

Dispatch uses YAML configuration files. The main config (`dispatch.yaml`) controls the service, while each site has its own `site.yaml`.

---

## Main Configuration: `dispatch.yaml`

### Server

```yaml
server:
  host: 0.0.0.0          # Listen address
  port: 8080              # Listen port
  base_url: https://mail.yourdomain.com  # Public URL (used in unsubscribe links, etc.)
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `host` | string | `0.0.0.0` | IP address to bind to. Use `127.0.0.1` if behind a reverse proxy. |
| `port` | int | `8080` | TCP port to listen on. |
| `base_url` | string | `http://localhost:8080` | The public-facing URL of your Dispatch instance. Used to generate unsubscribe links, view-in-browser URLs, and webhook callbacks. **Must be set correctly in production.** |

### Database

```yaml
database:
  driver: sqlite          # sqlite | postgres
  dsn: ./data/dispatch.db # Connection string
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `driver` | string | `sqlite` | Database driver. `sqlite` for SQLite, `postgres` for PostgreSQL. |
| `dsn` | string | `./data/dispatch.db` | Data Source Name. For SQLite, this is a file path. For PostgreSQL, use a connection string like `postgres://user:pass@host:5432/dispatch?sslmode=require`. |

#### SQLite Notes
- SQLite is the default and requires zero additional infrastructure.
- WAL mode is enabled automatically for better concurrent read performance.
- Suitable for most self-hosted deployments (up to ~100k emails/day).
- Database file is created automatically on first run.

#### PostgreSQL Notes
- Recommended for high-volume deployments or when you need horizontal scaling.
- Requires PostgreSQL 14+.
- Run `dispatch migrate` after switching drivers.
- Connection string supports all standard libpq parameters.

### Authentication

```yaml
auth:
  master_key: "dsp_master_xxxxxxxxxxxxx"
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `master_key` | string | `""` (empty) | Master API key with full access to all sites and admin endpoints. **Required for production.** Generate a secure random string. |

> ⚠️ If `master_key` is empty, admin endpoints (key management, cross-site operations) will be inaccessible. Site-specific keys (in `site.yaml`) still work for their respective sites.

### Queue

```yaml
queue:
  workers: 4
  retry_max: 3
  retry_backoff: "5m,30m,2h"
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `workers` | int | `4` | Number of concurrent queue workers processing the send queue. Each worker claims and processes messages independently. |
| `retry_max` | int | `3` | Maximum number of send attempts before a message is marked as permanently failed. |
| `retry_backoff` | string | `"5m,30m,2h"` | Comma-separated retry delays. First retry after 5 minutes, second after 30 minutes, third after 2 hours. Supports Go duration format: `s` (seconds), `m` (minutes), `h` (hours). |

#### Queue Behavior

1. When a send request arrives, the message is inserted into the `messages` table with status `queued` and added to the `send_queue` table.
2. Workers poll the `send_queue` every second for unlocked items where `next_retry <= now`.
3. A worker locks the item (sets `locked_by` and `locked_at`), processes it, then either completes or fails it.
4. Stale locks (older than 5 minutes) are automatically released, allowing another worker to retry.
5. After all retries are exhausted, the message status is set to `failed` with the last error recorded.

### Compliance

```yaml
compliance:
  gdpr_enabled: true
  consent_logging: true
  suppression_global: true
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `gdpr_enabled` | bool | `true` | Enable GDPR endpoints (`/api/v1/gdpr/export`, `/api/v1/gdpr/forget`). When disabled, these endpoints return 404. |
| `consent_logging` | bool | `true` | Log all consent-related actions (subscribe, unsubscribe, export, forget) to the `consent_log` table with timestamps, IP addresses, and source information. **Recommended for GDPR compliance.** |
| `suppression_global` | bool | `true` | When true, the suppression list is global — an email suppressed for one site is suppressed for all sites. When false, suppressions are per-site (not yet implemented). |

### Logging

```yaml
logging:
  level: info
  format: json
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `level` | string | `info` | Log level. Options: `debug`, `info`, `warn`, `error`. Use `debug` during development for verbose output including template rendering details and queue operations. |
| `format` | string | `json` | Log output format. `json` for structured logging (recommended for production), `text` for human-readable output. |

---

## Environment Variable Interpolation

All config files support environment variable expansion using `${VAR_NAME}` syntax:

```yaml
backend_config:
  api_key: "${RESEND_API_KEY}"
  password: "${SMTP_PASSWORD}"
```

This allows you to keep secrets out of config files. Set environment variables in your shell, `.env` file, or Docker Compose environment section.

**Precedence:** If the environment variable is not set, the literal string `${VAR_NAME}` is used (which will likely cause an error, alerting you to the missing variable).

---

## Site Configuration: `sites/<slug>/site.yaml`

Each site lives in its own directory under `sites/` and has a `site.yaml` configuration file.

### Full Reference

```yaml
# --- Identity ---
slug: my-site                    # Unique identifier (used in API paths)
name: My Website                 # Human-readable name (used in templates)
from: hello@mysite.com           # Sender email address
from_name: "My Website Team"     # Sender display name
reply_to: support@mysite.com     # Reply-to address (optional)

# --- Backend ---
backend: resend                  # Backend to use for this site
backend_config:                  # Backend-specific settings
  api_key: "${RESEND_API_KEY}"

# --- Templates ---
templates_dir: ./templates       # Path to templates (relative to site dir)

# --- Subscriber Settings ---
double_optin: true               # Require email confirmation before activating
optin_template: confirm-subscription  # Template for confirmation email
welcome_template: welcome        # Template sent after confirmation (optional)

# --- Authentication ---
api_key: "dsp_site_xxxxx"        # API key scoped to this site

# --- Lists ---
lists:
  - slug: newsletter
    name: Weekly Newsletter
    double_optin: true
  - slug: product-updates
    name: Product Updates
    double_optin: false
  - slug: transactional
    name: Transactional
    double_optin: false
    unsubscribable: false         # Can't unsubscribe from transactional emails

# --- Metadata ---
tags:                            # Optional tags for organization
  - gaming
  - indie
```

### Field Reference

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `slug` | string | ✅ | — | Unique site identifier. Used in API paths (`/api/v1/sites/{slug}/...`). Must be URL-safe (lowercase letters, numbers, hyphens). |
| `name` | string | No | Same as slug | Human-readable site name. Available in templates as `{{.Site.Name}}`. |
| `from` | string | ✅ | — | Sender email address. Must be a verified address with your email backend. |
| `from_name` | string | No | `""` | Display name shown alongside the sender email. |
| `reply_to` | string | No | `""` | Reply-to address. If empty, replies go to the `from` address. |
| `backend` | string | ✅ | — | Backend identifier (`smtp`, `resend`, `ses`, `listmonk`, or a custom registered backend). |
| `backend_config` | map | No | `{}` | Key-value pairs passed to the backend factory. See individual backend docs for required fields. |
| `templates_dir` | string | No | `./templates` | Path to the site's templates directory, relative to the site directory. |
| `double_optin` | bool | No | `false` | When true, new subscribers receive a confirmation email and start in `pending` status until they click the confirmation link. |
| `optin_template` | string | No | `""` | Template slug used for the double opt-in confirmation email. |
| `welcome_template` | string | No | `""` | Template slug sent after a subscriber confirms (only with double opt-in). |
| `api_key` | string | No | `""` | Site-scoped API key. This key can only access endpoints for this specific site. |
| `lists` | array | No | `[]` | Named mailing lists for this site. Subscribers can belong to multiple lists. |
| `tags` | array | No | `[]` | Organizational tags (not used by Dispatch internally, but available for your tooling). |

### List Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `slug` | string | — | List identifier |
| `name` | string | — | Human-readable list name |
| `double_optin` | bool | `false` | Override site-level double opt-in for this list |
| `unsubscribable` | bool | `true` | Whether recipients can unsubscribe from this list. Set to `false` for transactional lists (password resets, etc.). |

---

## Directory Structure

The complete Dispatch file structure:

```
your-project/
├── dispatch.yaml                    # Main configuration
├── data/
│   └── dispatch.db                  # SQLite database (auto-created)
├── sites/
│   ├── site-one/
│   │   ├── site.yaml                # Site configuration
│   │   └── templates/
│   │       ├── _base.html           # Site-specific base layout (optional)
│   │       ├── welcome.html         # Single-file template
│   │       ├── password-reset.html
│   │       └── weekly-digest/       # Directory-based template
│   │           ├── subject.txt
│   │           ├── body.html
│   │           └── body.txt         # Plain text version (optional)
│   └── site-two/
│       ├── site.yaml
│       └── templates/
│           └── ...
└── shared/
    └── templates/
        └── _default-base.html       # Shared base layout for all sites
```

### File Naming Conventions

- **Site directories:** lowercase, hyphenated (`my-cool-site`)
- **Template files:** lowercase, hyphenated (`password-reset.html`)
- **Base templates:** prefixed with underscore (`_base.html`) — excluded from template listings
- **Config files:** always `site.yaml` and `dispatch.yaml`

---

## Validating Configuration

Use `dispatch doctor` to check your configuration:

```bash
dispatch doctor
```

This checks:
- ✅ Config file exists and parses correctly
- ✅ Database is accessible and migrations are current
- ✅ Each site config is valid (required fields present)
- ✅ Referenced templates exist
- ✅ Backend connectivity (SMTP, API endpoints)
- ✅ DNS records (SPF, DKIM, DMARC) for sending domains

---

## Configuration Tips

### Minimal Production Config

```yaml
server:
  host: 127.0.0.1       # Behind reverse proxy
  port: 8080
  base_url: https://mail.yourdomain.com

database:
  driver: sqlite
  dsn: /var/lib/dispatch/dispatch.db

auth:
  master_key: "${DISPATCH_MASTER_KEY}"

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

### Multiple Sites Sharing a Backend

If all your sites use the same email provider, you only need to configure the backend once — Dispatch deduplicates backend instances by name:

```yaml
# sites/site-a/site.yaml
backend: resend
backend_config:
  api_key: "${RESEND_API_KEY}"

# sites/site-b/site.yaml
backend: resend
backend_config:
  api_key: "${RESEND_API_KEY}"
```

Both sites share the same Resend backend instance internally.

### Hot Reloading

Configuration is loaded at startup. To apply changes:

```bash
# Restart the service
dispatch serve   # if running in foreground

# Or with Docker
docker compose restart

# Or with systemd
sudo systemctl restart dispatch
```

Hot reload (without restart) is planned for a future release.
