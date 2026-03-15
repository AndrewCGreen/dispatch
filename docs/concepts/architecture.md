# Architecture

This document explains how Dispatch works internally — the request lifecycle, component responsibilities, and design decisions.

---

## High-Level Overview

```
┌──────────────────────────────────────────────────────────────┐
│                        Dispatch Service                       │
│                                                              │
│  ┌─────────┐    ┌──────────────┐    ┌────────────────────┐  │
│  │  HTTP    │    │  Orchestrator│    │  Template Engine   │  │
│  │  Server  │───►│              │───►│                    │  │
│  │  (chi)   │    │  - Validate  │    │  - Load from disk  │  │
│  └─────────┘    │  - Authorize │    │  - Render Go tpl   │  │
│                  │  - Route     │    │  - Subject extract │  │
│  ┌─────────┐    │  - Suppress  │    │  - Auto plain text │  │
│  │  Auth    │───►│  - Enqueue   │    └────────────────────┘  │
│  │  Middle- │    └──────┬───────┘                            │
│  │  ware    │           │                                    │
│  └─────────┘           ▼                                    │
│                  ┌──────────────┐    ┌────────────────────┐  │
│                  │  Send Queue  │    │  Backend Router    │  │
│                  │              │───►│                    │  │
│                  │  - Workers   │    │  ┌──────┐ ┌─────┐ │  │
│                  │  - Retry     │    │  │ SMTP │ │Rsnd │ │  │
│                  │  - Backoff   │    │  └──────┘ └─────┘ │  │
│                  │  - Lock mgmt│    │  ┌──────┐ ┌─────┐ │  │
│                  └──────────────┘    │  │ SES  │ │Lmnk │ │  │
│                                      │  └──────┘ └─────┘ │  │
│  ┌─────────────────────────────┐    └────────────────────┘  │
│  │  Data Store (SQLite/PG)     │                            │
│  │  - Subscribers              │                            │
│  │  - Messages + Queue         │                            │
│  │  - Suppressions             │                            │
│  │  - Consent log              │                            │
│  │  - API keys                 │                            │
│  │  - Webhooks                 │                            │
│  └─────────────────────────────┘                            │
└──────────────────────────────────────────────────────────────┘
```

---

## Request Lifecycle: Sending an Email

Here's what happens when your application calls `POST /api/v1/sites/my-site/send`:

### 1. Authentication (middleware.go)

```
Request → Extract Bearer token → Check against master key
                                → Check against site keys
                                → Check against DB keys (hashed)
                                → Reject if no match (401)
```

The auth middleware determines the key type (`master`, `site`, `readonly`) and the associated site (if site-scoped). This context is passed to the handler.

### 2. Validation (handlers_send.go)

The handler:
- Extracts the site slug from the URL path
- Verifies the authenticated key has access to this site
- Loads the site configuration from the in-memory config
- Parses and validates the JSON request body
- Checks required fields (`to`, `template`)

### 3. Suppression Check (store.go)

Before any send, the recipient email is checked against the global suppression list:

```sql
SELECT COUNT(*) FROM suppressions WHERE email = ?
```

If suppressed, the request is rejected immediately with a `409 Conflict` and error code `RECIPIENT_SUPPRESSED`. This prevents accidentally emailing someone who unsubscribed, bounced, or was GDPR-forgotten.

### 4. Template Rendering (template/engine.go)

The template engine:
1. Locates the template file(s) on disk (single file or directory)
2. Loads shared base templates (`shared/templates/_*.html`)
3. Loads site-level base templates (`sites/{site}/templates/_*.html`)
4. Extracts the subject line from the template comment
5. Executes the Go template with the provided data
6. Auto-generates a plain text version by stripping HTML tags
7. Returns the rendered subject, HTML, and text

### 5. Message Creation & Enqueueing (store.go)

Within a single database transaction:
1. A `messages` row is inserted with status `queued`
2. A `send_queue` row is inserted referencing the message

This ensures atomicity — either both records exist or neither does.

```sql
BEGIN;
INSERT INTO messages (id, site, to_email, template, subject, status, backend, ...) VALUES (...);
INSERT INTO send_queue (id, message_id, site) VALUES (...);
COMMIT;
```

### 6. Response

The API immediately returns `202 Accepted` with the message ID and queued timestamp. The actual delivery happens asynchronously.

### 7. Queue Processing (queue/queue.go)

Queue workers run in background goroutines:

```
Every 1 second:
  1. Query for unlocked items where next_retry <= now
  2. Lock them (SET locked_by, locked_at)
  3. For each item:
     a. Load the message from DB
     b. Re-check suppression (in case it changed)
     c. Get the backend for this site
     d. Build OutgoingMessage struct
     e. Call backend.Send()
     f. On success: delete from queue, update message status to "delivered"
     g. On failure: increment attempts, schedule retry or mark as "failed"
```

### 8. Backend Delivery (backend/*.go)

The backend receives a fully rendered `OutgoingMessage` and delivers it:

- **SMTP:** Connects to the SMTP server, authenticates, sends the email
- **Resend:** Makes an HTTP POST to `https://api.resend.com/emails`
- **SES:** Calls the AWS SES API

The backend returns a `SendResult` with the provider's message ID.

---

## Component Details

### Config Loader (`internal/config`)

**Responsibility:** Read and parse YAML config files, discover sites, expand environment variables.

**Key behaviors:**
- Loads `dispatch.yaml` at startup
- Scans the `sites/` directory for subdirectories containing `site.yaml`
- Expands `${ENV_VAR}` patterns in all config values
- Sets sensible defaults for missing fields
- Builds a `map[string]*SiteConfig` for fast site lookup

**Thread safety:** Config is read-only after loading. Restart required for changes.

### Data Store (`internal/store`)

**Responsibility:** All database operations — CRUD for subscribers, messages, suppressions, consent, queue management.

**Key behaviors:**
- Initializes the database connection and runs migrations
- SQLite: Enables WAL mode, foreign keys, and busy timeout
- All operations accept a `context.Context` for cancellation
- Queue operations use transactions for atomic lock/unlock/complete/fail
- Generates UUIDs for all primary keys

**Schema management:** The schema is embedded as a SQL string constant and applied with `CREATE TABLE IF NOT EXISTS`, making migrations idempotent.

### Backend System (`internal/backend`)

**Responsibility:** Abstract email delivery behind a common interface.

**Key components:**
- `Backend` interface — defines what every backend must implement
- `Registry` — global map of backend name → factory function
- `Router` — maps site slugs to backend instances
- Individual backends register themselves via `init()` functions

**Adding a new backend:**
1. Create a new file in `internal/backend/`
2. Implement the `Backend` interface
3. Call `Register("name", factory)` in `init()`
4. The backend is automatically available in config

### Template Engine (`internal/template`)

**Responsibility:** Load, parse, and render email templates.

**Key behaviors:**
- Supports single-file templates (`welcome.html`) and directory-based templates (`welcome/body.html` + `subject.txt` + `body.txt`)
- Extracts subject from Go template comments: `{{/* subject: Your Subject */}}`
- Loads base templates (prefixed with `_`) for layout inheritance
- Auto-generates plain text from HTML when no `.txt` file exists
- All template data is available under `.Data`, `.Site`, `.Subscriber`, etc.

### Send Queue (`internal/queue`)

**Responsibility:** Reliable asynchronous email delivery with retries.

**Key behaviors:**
- Configurable number of worker goroutines
- Workers poll every 1 second for available items
- Distributed locking via `locked_by` and `locked_at` columns
- Stale lock recovery (locks older than 5 minutes are released)
- Exponential backoff with configurable delays
- Messages transition through: `queued` → `sending` → `delivered` | `failed`

### HTTP Server (`internal/server`)

**Responsibility:** HTTP routing, middleware, request handling.

**Key behaviors:**
- Uses [chi](https://github.com/go-chi/chi) for routing (lightweight, idiomatic Go)
- Middleware stack: RequestID → RealIP → Recoverer → Timeout → Logging → Auth
- Handlers are organized by domain (send, subscribers, suppression, etc.)
- All responses are JSON with consistent error format
- Health endpoint is unauthenticated (for load balancers and monitoring)

---

## Design Decisions

### Why Go?

- **Single binary distribution** — no runtime dependencies, easy to deploy
- **Low memory footprint** — runs on a Raspberry Pi with room to spare
- **Excellent concurrency** — goroutines for queue workers, HTTP handling
- **Fast startup** — sub-second cold start
- **Strong standard library** — `net/smtp`, `html/template`, `database/sql` are solid

### Why SQLite by Default?

- **Zero external dependencies** — no separate database server needed
- **Excellent for single-node deployments** — handles thousands of writes/second with WAL mode
- **File-based** — easy to backup (just copy the file), easy to inspect
- **Perfectly adequate** — most self-hosted deployments won't exceed SQLite's capabilities
- **PostgreSQL available** — when you need horizontal scaling or concurrent writers

### Why File-Based Templates?

- **Version control friendly** — templates live in git alongside config
- **No admin UI needed** — edit in your IDE/editor of choice
- **Easy to review** — `git diff` shows exactly what changed
- **Portable** — copy the directory to move between environments
- **Safe** — no risk of template injection via API

### Why No Admin UI (v1)?

- **Reduce attack surface** — fewer endpoints to secure
- **Faster to build** — focus on the core service
- **CLI and API are sufficient** — for the target audience (developers/self-hosters)
- **Planned for later** — optional web UI in v1.0

### Why Not Use an External Queue (Redis, RabbitMQ)?

- **Simplicity** — one less service to deploy and maintain
- **Adequate performance** — SQLite handles thousands of queue operations per second
- **Atomicity** — message creation and enqueueing in one transaction
- **Recoverable** — queue state persists across restarts automatically
- **Future option** — Redis/NATS adapter can be added later if needed

---

## Data Flow Diagrams

### Transactional Send

```
Client                  Dispatch                    Backend
  │                       │                           │
  │  POST /send           │                           │
  │──────────────────────►│                           │
  │                       │  Auth check               │
  │                       │  Suppression check        │
  │                       │  Render template          │
  │                       │  Insert message + queue   │
  │  202 Accepted         │                           │
  │◄──────────────────────│                           │
  │                       │                           │
  │                       │  [Queue worker picks up]  │
  │                       │  Build OutgoingMessage    │
  │                       │──────────────────────────►│
  │                       │                           │  Deliver
  │                       │  SendResult               │
  │                       │◄──────────────────────────│
  │                       │  Update message: delivered│
  │                       │                           │
```

### Subscriber Lifecycle

```
Client                  Dispatch                    Recipient
  │                       │                           │
  │  POST /subscribers    │                           │
  │──────────────────────►│                           │
  │                       │  Create (status: pending) │
  │                       │  Log consent              │
  │                       │  Send confirmation email  │
  │  201 Created          │──────────────────────────►│
  │◄──────────────────────│                           │
  │                       │                           │
  │                       │      [User clicks link]   │
  │                       │◄──────────────────────────│
  │                       │  Update: active           │
  │                       │  Send welcome email       │
  │                       │──────────────────────────►│
  │                       │                           │
```

### GDPR Forget

```
Client                  Dispatch
  │                       │
  │  POST /gdpr/forget    │
  │──────────────────────►│
  │                       │  For each site:
  │                       │    Delete subscriber
  │                       │  Add to suppression (permanent)
  │                       │  Log consent: "forget"
  │  200 OK               │
  │◄──────────────────────│
  │                       │
  │  [Future send to      │
  │   this email]         │
  │──────────────────────►│
  │  409 Suppressed       │  ← Blocked by suppression
  │◄──────────────────────│
```
