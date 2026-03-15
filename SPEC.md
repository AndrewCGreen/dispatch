# Dispatch — API & Architecture Spec

> Self-hosted email orchestration for multi-site operators.
> Working name: **Dispatch** (check availability before launch)

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Configuration](#configuration)
4. [API Design](#api-design)
5. [Backend Plugin System](#backend-plugin-system)
6. [Template Engine](#template-engine)
7. [Subscriber Management](#subscriber-management)
8. [GDPR & Compliance](#gdpr--compliance)
9. [Webhooks & Events](#webhooks--events)
10. [Authentication](#authentication)
11. [Data Models](#data-models)
12. [Deployment](#deployment)
13. [SDK Interface](#sdk-interface)

---

## Overview

Dispatch is a lightweight, self-hosted service that sits between your web applications and email delivery backends. It provides:

- **Multi-site management** — one instance serves all your projects
- **Pluggable backends** — SMTP, Resend, SES, Listmonk, Mailgun, etc.
- **File-based templates** — version-controlled, per-site
- **Built-in compliance** — GDPR, CAN-SPAM, unsubscribe handling
- **Simple REST API** — any language, any stack

### What It Is Not

- Not a mail server (no IMAP/POP3)
- Not a marketing platform (no drag-and-drop editors)
- Not a mail client (no inbox)

It's the **routing and orchestration layer** — the missing piece between "I want to send an email" and the dozen things you need to handle.

---

## Architecture

```
                    ┌──────────────────────────────────────────┐
                    │              Dispatch Service             │
                    │                                          │
  Python SDK ──────►│  ┌──────────┐    ┌───────────────────┐  │
  Go SDK ──────────►│  │ REST API │───►│  Orchestrator     │  │
  curl / any ──────►│  └──────────┘    │                   │  │
                    │                  │  - Route to site   │  │
                    │  ┌──────────┐    │  - Render template │  │
                    │  │  Config  │───►│  - Check suppress  │  │
                    │  │  Loader  │    │  - Enqueue send    │  │
                    │  └──────────┘    └────────┬──────────┘  │
                    │                           │              │
                    │  ┌────────────────────────▼───────────┐  │
                    │  │         Backend Router             │  │
                    │  │                                    │  │
                    │  │  ┌───────┐ ┌──────┐ ┌──────────┐  │  │
                    │  │  │ SMTP  │ │Resend│ │ Listmonk │  │  │
                    │  │  └───────┘ └──────┘ └──────────┘  │  │
                    │  │  ┌───────┐ ┌──────┐ ┌──────────┐  │  │
                    │  │  │  SES  │ │Mailgn│ │ Postal   │  │  │
                    │  │  └───────┘ └──────┘ └──────────┘  │  │
                    │  └───────────────────────────────────┘  │
                    │                                          │
                    │  ┌───────────────────────────────────┐  │
                    │  │         Data Store                 │  │
                    │  │  SQLite (default) / PostgreSQL     │  │
                    │  │  - Subscribers                     │  │
                    │  │  - Suppression list                │  │
                    │  │  - Send log / audit trail          │  │
                    │  │  - Consent records                 │  │
                    │  └───────────────────────────────────┘  │
                    └──────────────────────────────────────────┘
```

### Key Design Decisions

1. **Go single binary** — easy to distribute, low resource usage, ideal for self-hosting on a Pi or $5 VPS
2. **SQLite by default** — zero external dependencies to start; PostgreSQL supported for scale
3. **File-based config + templates** — git-friendly, no admin UI required (optional web UI later)
4. **Queue built-in** — internal send queue with retry logic, no external queue dependency
5. **Stateless API** — all state lives in DB + config files; horizontally scalable if needed

---

## Configuration

### Main Config: `dispatch.yaml`

```yaml
server:
  host: 0.0.0.0
  port: 8080
  base_url: https://mail.yourdomain.com

database:
  driver: sqlite          # sqlite | postgres
  dsn: ./data/dispatch.db # or postgres connection string

auth:
  master_key: "dsp_xxxxxxxxxxxxx"   # admin access
  # Per-site keys defined in site configs

queue:
  workers: 4
  retry_max: 3
  retry_backoff: "5m,30m,2h"

compliance:
  unsubscribe_url: "{{base_url}}/unsubscribe/{{token}}"
  gdpr_enabled: true
  consent_logging: true
  suppression_global: true        # suppressed email = suppressed everywhere

logging:
  level: info
  format: json
```

### Site Config: `sites/<site-slug>/site.yaml`

```yaml
slug: maple-game
name: Maple Game
from: hello@maplegame.com
from_name: Maple Game Team
reply_to: support@maplegame.com

backend: resend              # which backend to route through
backend_config:
  api_key: "${RESEND_API_KEY}"  # env var interpolation supported

templates_dir: ./templates   # relative to this site dir

# Subscriber settings
double_optin: true
optin_template: confirm-subscription
welcome_template: welcome    # sent after confirmation (optional)

# Site-specific API key (scoped to this site only)
api_key: "dsp_site_xxxxxxxxxxxxx"

# Optional metadata
tags:
  - gaming
  - indie
```

### Directory Structure

```
dispatch/
├── dispatch.yaml            # main config
├── data/
│   └── dispatch.db          # SQLite database
├── sites/
│   ├── maple-game/
│   │   ├── site.yaml
│   │   └── templates/
│   │       ├── _base.html          # shared layout for this site
│   │       ├── welcome.html
│   │       ├── password-reset.html
│   │       └── weekly-digest.html
│   └── my-saas/
│       ├── site.yaml
│       └── templates/
│           ├── _base.html
│           ├── trial-ending.html
│           └── invoice.html
├── shared/
│   └── templates/           # shared templates available to all sites
│       └── _default-base.html
└── backends/                # backend plugin configs (if needed)
```

---

## API Design

Base: `POST /api/v1/...`
Auth: `Authorization: Bearer <api_key>`
Content-Type: `application/json`

---

### Send Email

The primary endpoint. Sends a transactional email immediately (or queues it).

```
POST /api/v1/sites/{site}/send
```

**Request:**
```json
{
  "to": "user@example.com",
  "template": "welcome",
  "data": {
    "name": "Alex",
    "login_url": "https://maplegame.com/login"
  },
  "tags": ["onboarding"],
  "metadata": {
    "user_id": "usr_123",
    "signup_source": "landing-page"
  }
}
```

**Response (202 Accepted):**
```json
{
  "id": "msg_a1b2c3d4e5",
  "status": "queued",
  "to": "user@example.com",
  "template": "welcome",
  "site": "maple-game",
  "queued_at": "2026-03-14T22:50:00Z"
}
```

**Errors:**
```json
{
  "error": "suppressed",
  "message": "Recipient is on the suppression list",
  "code": "RECIPIENT_SUPPRESSED"
}
```

---

### Send Raw (No Template)

For one-off sends where you provide the content directly.

```
POST /api/v1/sites/{site}/send/raw
```

```json
{
  "to": "user@example.com",
  "subject": "Your order has shipped",
  "html": "<h1>Shipped!</h1><p>Tracking: {{tracking_id}}</p>",
  "text": "Shipped! Tracking: {{tracking_id}}",
  "data": {
    "tracking_id": "1Z999AA10123456784"
  }
}
```

---

### Batch Send

Send to multiple recipients with per-recipient data.

```
POST /api/v1/sites/{site}/send/batch
```

```json
{
  "template": "weekly-digest",
  "recipients": [
    {
      "to": "alice@example.com",
      "data": { "name": "Alice", "items_count": 5 }
    },
    {
      "to": "bob@example.com",
      "data": { "name": "Bob", "items_count": 12 }
    }
  ],
  "tags": ["digest", "weekly"]
}
```

**Response (202):**
```json
{
  "batch_id": "batch_x1y2z3",
  "total": 2,
  "queued": 2,
  "suppressed": 0
}
```

---

### Subscribers

```
POST   /api/v1/sites/{site}/subscribers           # Add/subscribe
GET    /api/v1/sites/{site}/subscribers           # List (paginated)
GET    /api/v1/sites/{site}/subscribers/{email}   # Get one
PUT    /api/v1/sites/{site}/subscribers/{email}   # Update attributes
DELETE /api/v1/sites/{site}/subscribers/{email}   # Unsubscribe + remove
```

**Subscribe:**
```json
{
  "email": "user@example.com",
  "name": "Alex",
  "attributes": {
    "plan": "pro",
    "signed_up": "2026-03-14"
  },
  "lists": ["newsletter", "product-updates"],
  "consent": {
    "source": "signup-form",
    "ip": "203.0.113.42",
    "url": "https://maplegame.com/signup"
  }
}
```

**Response (201):**
```json
{
  "email": "user@example.com",
  "status": "pending",
  "confirm_sent": true,
  "message": "Double opt-in confirmation sent"
}
```

**List response:**
```json
{
  "subscribers": [...],
  "total": 1432,
  "page": 1,
  "per_page": 50,
  "has_more": true
}
```

---

### Suppression List

Global suppression — works across all sites.

```
GET    /api/v1/suppressions                      # List
POST   /api/v1/suppressions                      # Add manually
DELETE /api/v1/suppressions/{email}              # Remove
GET    /api/v1/suppressions/check/{email}        # Quick check
```

**Add to suppression:**
```json
{
  "email": "user@example.com",
  "reason": "manual",
  "note": "Requested removal via support ticket #4521"
}
```

Reasons: `unsubscribe`, `bounce`, `complaint`, `manual`, `gdpr`

---

### Templates

```
GET    /api/v1/sites/{site}/templates              # List available
GET    /api/v1/sites/{site}/templates/{slug}        # Get template source
POST   /api/v1/sites/{site}/templates/{slug}/render # Preview with data
```

**Preview:**
```json
{
  "data": {
    "name": "Alex",
    "login_url": "https://example.com"
  }
}
```

**Response:**
```json
{
  "subject": "Welcome, Alex!",
  "html": "<h1>Welcome, Alex!</h1>...",
  "text": "Welcome, Alex! ..."
}
```

> Templates are managed as files on disk. The API is read-only by design —
> edit templates in your editor/IDE/git, not through the API.

---

### GDPR Endpoints

```
POST /api/v1/gdpr/export
POST /api/v1/gdpr/forget
```

**Export (data subject access request):**
```json
{
  "email": "user@example.com"
}
```

**Response:**
```json
{
  "email": "user@example.com",
  "sites": {
    "maple-game": {
      "subscribed": true,
      "lists": ["newsletter"],
      "attributes": { "plan": "pro" },
      "consent": {
        "source": "signup-form",
        "recorded_at": "2026-01-15T10:00:00Z",
        "ip": "203.0.113.42"
      },
      "messages_sent": 24,
      "last_sent": "2026-03-12T08:00:00Z"
    },
    "my-saas": {
      "subscribed": false,
      "unsubscribed_at": "2026-02-01T14:30:00Z"
    }
  }
}
```

**Forget (right to erasure):**
```json
{
  "email": "user@example.com",
  "confirm": true
}
```

This deletes all subscriber data, adds to suppression (to prevent re-addition), and logs the erasure event (required for compliance audit trail).

---

### Messages / Audit Log

```
GET /api/v1/sites/{site}/messages                 # List sent messages
GET /api/v1/sites/{site}/messages/{id}            # Get message detail
GET /api/v1/messages/{id}/status                  # Delivery status
```

**Message detail:**
```json
{
  "id": "msg_a1b2c3d4e5",
  "site": "maple-game",
  "to": "user@example.com",
  "template": "welcome",
  "status": "delivered",
  "backend": "resend",
  "backend_id": "re_xxxxx",
  "queued_at": "2026-03-14T22:50:00Z",
  "sent_at": "2026-03-14T22:50:01Z",
  "delivered_at": "2026-03-14T22:50:03Z",
  "tags": ["onboarding"],
  "metadata": { "user_id": "usr_123" }
}
```

Statuses: `queued` → `sending` → `delivered` | `bounced` | `failed` | `suppressed`

---

### Health

```
GET /api/v1/health
```

```json
{
  "status": "healthy",
  "version": "0.1.0",
  "uptime": "4d 12h 30m",
  "database": "ok",
  "queue": {
    "pending": 3,
    "processing": 1,
    "failed": 0
  },
  "backends": {
    "resend": "ok",
    "smtp-fallback": "ok"
  }
}
```

---

## Backend Plugin System

Each backend implements the `Backend` interface:

```go
type Backend interface {
    // Name returns the backend identifier (e.g., "resend", "ses", "smtp")
    Name() string

    // Send delivers a single rendered email
    Send(ctx context.Context, msg *OutgoingMessage) (*SendResult, error)

    // BatchSend delivers multiple emails (optional optimization)
    // Default implementation falls back to sequential Send calls
    BatchSend(ctx context.Context, msgs []*OutgoingMessage) (*BatchResult, error)

    // Health checks backend connectivity
    Health(ctx context.Context) error

    // MaxBatchSize returns the maximum batch size (0 = no batch support)
    MaxBatchSize() int
}

type OutgoingMessage struct {
    MessageID  string
    From       string
    FromName   string
    ReplyTo    string
    To         string
    Subject    string
    HTML       string
    Text       string
    Headers    map[string]string    // custom headers (List-Unsubscribe, etc.)
    Tags       []string
    Metadata   map[string]string
}

type SendResult struct {
    BackendID  string               // provider's message ID
    Status     string               // "sent", "queued" (at provider level)
}

type BatchResult struct {
    Succeeded  int
    Failed     int
    Errors     []BatchError
}
```

### Built-in Backends (v1)

| Backend | Type | Notes |
|---------|------|-------|
| `smtp` | Direct SMTP | Any SMTP server, TLS/STARTTLS |
| `resend` | API | resend.com — generous free tier |
| `ses` | API | AWS SES — cheapest at scale |
| `listmonk` | API | For users already running Listmonk |

### Adding a Custom Backend

```go
package myprovider

import "github.com/yourname/dispatch/backend"

type MyProvider struct {
    apiKey string
}

func New(config map[string]string) (backend.Backend, error) {
    return &MyProvider{apiKey: config["api_key"]}, nil
}

func (m *MyProvider) Name() string { return "myprovider" }

func (m *MyProvider) Send(ctx context.Context, msg *backend.OutgoingMessage) (*backend.SendResult, error) {
    // your sending logic here
}

// Register in init() or via plugin config
func init() {
    backend.Register("myprovider", New)
}
```

---

## Template Engine

### Template Format

Each template is a directory or a single HTML file:

```
templates/
├── welcome.html            # simple: single file
├── password-reset/         # complex: directory
│   ├── subject.txt         # "Reset your password, {{.Name}}"
│   ├── body.html           # HTML version
│   └── body.txt            # plain text version (auto-generated if missing)
└── _base.html              # layout: {{block "content" .}}{{end}}
```

### Simple Template (single file)

```html
{{/* subject: Welcome to {{.SiteName}}, {{.Name}}! */}}

{{template "_base.html" .}}

{{define "content"}}
<h1>Welcome, {{.Name}}!</h1>
<p>Thanks for signing up. Get started here:</p>
<a href="{{.LoginURL}}" class="button">Log In</a>
{{end}}
```

### Template Variables

Every template receives:

| Variable | Description |
|----------|-------------|
| `.Data.*` | Custom data passed in the send request |
| `.Site.Name` | Site display name |
| `.Site.URL` | Site base URL |
| `.Subscriber.Email` | Recipient email |
| `.Subscriber.Name` | Recipient name |
| `.Subscriber.Attributes.*` | Custom subscriber attributes |
| `.UnsubscribeURL` | One-click unsubscribe link |
| `.PreferencesURL` | Email preferences page |
| `.ViewInBrowserURL` | Web version link |

### Auto-Generated Content

- **Plain text version**: Auto-generated from HTML if `body.txt` is not provided
- **List-Unsubscribe header**: Automatically added to all non-transactional emails
- **Tracking pixel**: Optional, disabled by default (privacy-first)

---

## Subscriber Management

### Lists

Each site can have multiple lists. Subscribers can belong to multiple lists.

```yaml
# In site.yaml
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
    unsubscribable: false    # can't unsubscribe from password resets
```

### Subscriber Lifecycle

```
subscribe request
       │
       ▼
  ┌──────────┐     ┌───────────┐     ┌────────┐
  │  pending  │────►│ confirmed │────►│ active │
  └──────────┘     └───────────┘     └────────┘
  (double optin      (clicked           │
   email sent)        confirm link)     │
                                        ▼
                                  ┌──────────────┐
                                  │ unsubscribed  │
                                  └──────────────┘
                                        │
                                        ▼ (GDPR forget)
                                  ┌──────────────┐
                                  │   deleted +   │
                                  │  suppressed   │
                                  └──────────────┘
```

---

## GDPR & Compliance

### Automatic Behaviors

1. **Consent recording** — every subscribe action logs: source, timestamp, IP, page URL
2. **Unsubscribe link** — injected into all campaign/list emails (not transactional)
3. **List-Unsubscribe header** — RFC 8058 one-click unsubscribe header added automatically
4. **Suppression enforcement** — suppressed emails are checked before every send; sends silently blocked
5. **Data export** — single API call returns all data for an email across all sites
6. **Right to erasure** — single API call deletes all data + adds to permanent suppression
7. **Audit trail** — all compliance-relevant actions logged with timestamps

### Unsubscribe Flow

```
User clicks unsubscribe link
         │
         ▼
┌─────────────────────┐
│  /unsubscribe/{tok} │  ← public page, no auth required
│                     │
│  "You've been       │
│   unsubscribed."    │
│                     │
│  [Manage prefs]     │  ← optional: choose which lists
│  [Resubscribe]      │  ← with confirmation
└─────────────────────┘
         │
         ▼
  Subscriber status → unsubscribed
  Suppression entry created
  Webhook fired: "subscriber.unsubscribed"
```

---

## Webhooks & Events

Dispatch emits events that your applications can subscribe to.

### Webhook Registration

```
POST /api/v1/sites/{site}/webhooks
```

```json
{
  "url": "https://maplegame.com/hooks/email",
  "events": ["message.delivered", "message.bounced", "subscriber.unsubscribed"],
  "secret": "whsec_xxxxx"
}
```

### Event Types

| Event | Fired When |
|-------|-----------|
| `message.queued` | Email enters the send queue |
| `message.sent` | Email handed to backend |
| `message.delivered` | Backend confirms delivery |
| `message.bounced` | Hard/soft bounce received |
| `message.complained` | Spam complaint received |
| `message.failed` | All retries exhausted |
| `subscriber.created` | New subscriber added |
| `subscriber.confirmed` | Double opt-in confirmed |
| `subscriber.unsubscribed` | Unsubscribe processed |
| `subscriber.deleted` | GDPR erasure completed |

### Webhook Payload

```json
{
  "event": "message.delivered",
  "timestamp": "2026-03-14T22:50:03Z",
  "site": "maple-game",
  "data": {
    "message_id": "msg_a1b2c3d4e5",
    "to": "user@example.com",
    "template": "welcome",
    "backend": "resend",
    "backend_id": "re_xxxxx"
  }
}
```

Signed with HMAC-SHA256 in `X-Dispatch-Signature` header.

---

## Authentication

### API Keys

Three levels:

| Key Type | Prefix | Scope |
|----------|--------|-------|
| Master | `dsp_master_` | Full access to all sites and admin endpoints |
| Site | `dsp_site_` | Scoped to a single site |
| Read-only | `dsp_read_` | Read access only (templates, subscribers, logs) |

### Auth Header

```
Authorization: Bearer dsp_site_xxxxxxxxxxxxx
```

### Key Management

```
POST   /api/v1/auth/keys           # Create key (master only)
GET    /api/v1/auth/keys           # List keys (master only)
DELETE /api/v1/auth/keys/{id}      # Revoke key (master only)
```

Or via CLI:
```bash
dispatch keys create --site maple-game --type site
dispatch keys create --type master
dispatch keys list
dispatch keys revoke dsp_site_xxxxx
```

---

## Data Models

### Database Schema (SQLite/PostgreSQL)

```sql
-- Subscribers (per-site)
CREATE TABLE subscribers (
    id          TEXT PRIMARY KEY,
    site        TEXT NOT NULL,
    email       TEXT NOT NULL,
    name        TEXT,
    status      TEXT NOT NULL DEFAULT 'pending',  -- pending, active, unsubscribed
    attributes  JSON,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    confirmed_at TIMESTAMP,
    unsubscribed_at TIMESTAMP,
    UNIQUE(site, email)
);

-- Subscriber list memberships
CREATE TABLE subscriber_lists (
    subscriber_id TEXT NOT NULL REFERENCES subscribers(id),
    list_slug     TEXT NOT NULL,
    subscribed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (subscriber_id, list_slug)
);

-- Consent records (append-only)
CREATE TABLE consent_log (
    id          TEXT PRIMARY KEY,
    email       TEXT NOT NULL,
    site        TEXT NOT NULL,
    action      TEXT NOT NULL,    -- subscribe, unsubscribe, export, forget
    source      TEXT,             -- signup-form, api, import
    ip          TEXT,
    url         TEXT,
    note        TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Global suppression list
CREATE TABLE suppressions (
    email       TEXT PRIMARY KEY,
    reason      TEXT NOT NULL,    -- unsubscribe, bounce, complaint, manual, gdpr
    note        TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Message log
CREATE TABLE messages (
    id          TEXT PRIMARY KEY,
    site        TEXT NOT NULL,
    to_email    TEXT NOT NULL,
    template    TEXT,
    subject     TEXT,
    status      TEXT NOT NULL DEFAULT 'queued',
    backend     TEXT,
    backend_id  TEXT,
    tags        JSON,
    metadata    JSON,
    error       TEXT,
    queued_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sent_at     TIMESTAMP,
    delivered_at TIMESTAMP,
    bounced_at  TIMESTAMP,
    failed_at   TIMESTAMP
);

-- Send queue
CREATE TABLE send_queue (
    id          TEXT PRIMARY KEY,
    message_id  TEXT NOT NULL REFERENCES messages(id),
    site        TEXT NOT NULL,
    attempts    INTEGER NOT NULL DEFAULT 0,
    next_retry  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    locked_by   TEXT,
    locked_at   TIMESTAMP,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Webhook registrations
CREATE TABLE webhooks (
    id          TEXT PRIMARY KEY,
    site        TEXT NOT NULL,
    url         TEXT NOT NULL,
    events      JSON NOT NULL,
    secret      TEXT NOT NULL,
    active      BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- API keys
CREATE TABLE api_keys (
    id          TEXT PRIMARY KEY,
    key_hash    TEXT NOT NULL UNIQUE,  -- bcrypt hash, never store raw
    key_prefix  TEXT NOT NULL,         -- first 8 chars for identification
    type        TEXT NOT NULL,         -- master, site, readonly
    site        TEXT,                  -- NULL for master keys
    name        TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used   TIMESTAMP,
    revoked_at  TIMESTAMP
);

CREATE INDEX idx_subscribers_site_email ON subscribers(site, email);
CREATE INDEX idx_messages_site ON messages(site, queued_at DESC);
CREATE INDEX idx_messages_status ON messages(status);
CREATE INDEX idx_send_queue_next ON send_queue(next_retry) WHERE locked_by IS NULL;
CREATE INDEX idx_suppressions_email ON suppressions(email);
```

---

## Deployment

### Docker Compose (Primary)

```yaml
version: "3.8"
services:
  dispatch:
    image: ghcr.io/yourname/dispatch:latest
    ports:
      - "8080:8080"
    volumes:
      - ./dispatch.yaml:/etc/dispatch/dispatch.yaml
      - ./sites:/etc/dispatch/sites
      - ./data:/var/lib/dispatch
    environment:
      - RESEND_API_KEY=${RESEND_API_KEY}
      - AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID}
      - AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY}
    restart: unless-stopped
```

### Single Binary

```bash
# Download
curl -LO https://github.com/yourname/dispatch/releases/latest/download/dispatch-linux-amd64
chmod +x dispatch-linux-amd64

# Init config
./dispatch init

# Run
./dispatch serve
```

### CLI Commands

```bash
dispatch serve                          # Start the server
dispatch init                           # Generate default config
dispatch site add <slug>                # Scaffold a new site config
dispatch site list                      # List configured sites
dispatch keys create --site <slug>      # Generate API key
dispatch keys list                      # List API keys
dispatch template preview <site> <tpl>  # Preview a template locally
dispatch send <site> <tpl> <email>      # Send a test email
dispatch queue status                   # Show queue stats
dispatch migrate                        # Run database migrations
dispatch doctor                         # Check config, backends, DNS
```

---

## SDK Interface

### Python SDK

```python
from dispatch import Dispatch

# Initialize
client = Dispatch(
    host="https://mail.yourdomain.com",
    api_key="dsp_site_xxxxx"
)

# Send transactional email
result = client.send(
    template="welcome",
    to="user@example.com",
    data={"name": "Alex", "login_url": "https://..."}
)
print(result.message_id)  # msg_a1b2c3d4e5

# Batch send
results = client.batch_send(
    template="weekly-digest",
    recipients=[
        {"to": "alice@example.com", "data": {"name": "Alice"}},
        {"to": "bob@example.com", "data": {"name": "Bob"}},
    ]
)

# Manage subscribers
client.subscribers.add(
    email="user@example.com",
    name="Alex",
    lists=["newsletter"],
    consent={"source": "signup-form", "ip": "203.0.113.42"}
)

client.subscribers.remove("user@example.com")

# Check suppression
is_suppressed = client.suppressions.check("user@example.com")

# GDPR
export = client.gdpr.export("user@example.com")
client.gdpr.forget("user@example.com")
```

### Go SDK

```go
package main

import "github.com/yourname/dispatch-go"

func main() {
    client := dispatch.New("https://mail.yourdomain.com", "dsp_site_xxxxx")

    // Send
    result, err := client.Send(ctx, &dispatch.SendRequest{
        Template: "welcome",
        To:       "user@example.com",
        Data:     map[string]any{"name": "Alex"},
    })

    // Subscribe
    err = client.Subscribers.Add(ctx, &dispatch.Subscriber{
        Email: "user@example.com",
        Name:  "Alex",
        Lists: []string{"newsletter"},
    })
}
```

---

## Open Questions

- [ ] **Name**: "Dispatch" — need to check GitHub/PyPI/Go module availability
- [ ] **Admin UI**: Build a minimal web UI, or stay CLI/API only for v1?
- [ ] **Open tracking**: Include pixel tracking as opt-in, or leave it out entirely? (Privacy considerations)
- [ ] **Rate limiting**: Per-site rate limits? Per-backend?
- [ ] **Multi-tenant auth**: Should sites be fully isolated (separate DBs) or shared?
- [ ] **License**: MIT? Apache 2.0? AGPL? (AGPL common for self-hosted to prevent SaaS-ification)

---

## Version Roadmap

### v0.1 — MVP
- Core service with SMTP backend
- Single-file templates
- SQLite storage
- Basic API (send, subscribers, suppress)
- CLI (serve, init, send)
- Docker Compose deployment
- Python SDK

### v0.2 — Backends + Polish
- Resend, SES backends
- Go SDK
- Batch sending
- Webhook events
- Queue retry logic
- `dispatch doctor` diagnostics

### v0.3 — Compliance + Scale
- Full GDPR endpoints
- Consent logging
- PostgreSQL support
- Double opt-in flow
- Unsubscribe page customization

### v1.0 — Production Ready
- Listmonk backend
- Admin web UI (optional)
- Horizontal scaling docs
- Security audit
- Comprehensive docs site

---

*Last updated: 2026-03-14*
