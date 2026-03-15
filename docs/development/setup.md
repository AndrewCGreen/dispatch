# Development Setup

Get a local Dispatch development environment running in minutes.

---

## Prerequisites

- **Go 1.22+** — [install](https://go.dev/doc/install)
- **GCC** — for SQLite CGO (`apt install gcc` / `brew install gcc`)
- **Git**
- Optional: **Docker** for running a local SMTP server

---

## Clone and Build

```bash
git clone https://github.com/dispatch-email/dispatch.git
cd dispatch

# Download dependencies
go mod download

# Build
CGO_ENABLED=1 go build -o dispatch .

# Verify
./dispatch version
```

---

## Local Development Setup

### 1. Initialize Config

```bash
./dispatch init
```

### 2. Add a Test Site

```bash
./dispatch site add dev-site
```

### 3. Configure SMTP (Local)

For local development, use [Mailpit](https://mailpit.axllent.org/) — a local SMTP server with a web UI that catches all outgoing email:

```bash
# Install Mailpit
go install github.com/axllent/mailpit@latest

# Run Mailpit (catches all email, web UI at http://localhost:8025)
mailpit &
```

Configure `sites/dev-site/site.yaml`:

```yaml
slug: dev-site
name: Dev Site
from: dev@localhost
from_name: "Dev Site"
backend: smtp
backend_config:
  host: localhost
  port: "1025"   # Mailpit SMTP port
templates_dir: ./templates
api_key: "dsp_site_dev_key"
```

Configure `dispatch.yaml`:

```yaml
server:
  host: 0.0.0.0
  port: 8080
  base_url: http://localhost:8080
database:
  driver: sqlite
  dsn: ./data/dispatch.db
auth:
  master_key: "dsp_master_dev_key"
logging:
  level: debug   # ← Use debug in development
  format: text   # ← Easier to read than JSON
```

### 4. Run the Server

```bash
./dispatch serve
```

You should see:
```
INFO  dispatch starting addr=0.0.0.0:8080 version=0.1.0-dev
INFO  queue started workers=4
```

### 5. Send a Test Email

```bash
curl -X POST http://localhost:8080/api/v1/sites/dev-site/send \
  -H "Authorization: Bearer dsp_site_dev_key" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "test@example.com",
    "template": "welcome",
    "data": {"name": "Developer"}
  }'
```

Check Mailpit at http://localhost:8025 to see the email.

---

## Development Workflow

### Live Reload

Use [Air](https://github.com/air-verse/air) for automatic rebuilds on file changes:

```bash
# Install Air
go install github.com/air-verse/air@latest

# Run with Air
air
```

Create `.air.toml`:
```toml
[build]
  cmd = "CGO_ENABLED=1 go build -o ./tmp/dispatch ."
  bin = "./tmp/dispatch"
  args_bin = ["serve"]
  include_ext = ["go", "yaml"]
  exclude_dir = ["data", "sites", "shared", "docs"]
```

### Testing Templates

The template preview API is useful during template development:

```bash
# Preview without sending
curl -X POST http://localhost:8080/api/v1/sites/dev-site/templates/welcome/render \
  -H "Authorization: Bearer dsp_site_dev_key" \
  -H "Content-Type: application/json" \
  -d '{"data": {"name": "Alex"}}'
```

---

## Project Structure

```
dispatch/
├── main.go                   # Entry point
├── cmd/                      # CLI commands (serve, init, site, keys, etc.)
├── internal/
│   ├── config/               # Config loading and site discovery
│   ├── models/               # Shared data types and API types
│   ├── store/                # Database layer (SQLite/PostgreSQL)
│   ├── backend/              # Email delivery backends
│   ├── template/             # Template engine
│   ├── queue/                # Send queue and workers
│   └── server/               # HTTP server, middleware, handlers
├── docs/                     # Documentation (this directory)
├── tests/                    # Integration tests
├── Dockerfile
└── docker-compose.yml
```

---

## Code Style

Follow standard Go conventions:

```bash
# Format code
gofmt -w .

# Lint
go vet ./...

# golangci-lint (if installed)
golangci-lint run
```

Key style points:
- Use `context.Context` as the first parameter for all functions that do I/O
- Return errors; don't panic in library code
- Wrap errors with context: `fmt.Errorf("loading config: %w", err)`
- Write tests for all new functionality
- Use `slog` for logging (not `log` or `fmt.Println`)

---

## Debugging

### Enable Debug Logging

```yaml
logging:
  level: debug
  format: text
```

Debug logging includes:
- Template rendering steps
- Queue worker poll cycles
- Database queries (planned)
- Backend request/response details

### SQLite Browser

Inspect the database directly:

```bash
# CLI
sqlite3 data/dispatch.db

# GUI: DB Browser for SQLite (recommended)
# https://sqlitebrowser.org/
```

Useful queries:

```sql
-- Check queue depth
SELECT site, COUNT(*), MIN(next_retry), MAX(attempts)
FROM send_queue GROUP BY site;

-- Recent messages
SELECT id, to_email, template, status, queued_at
FROM messages ORDER BY queued_at DESC LIMIT 20;

-- Recent consent log
SELECT email, site, action, source, created_at
FROM consent_log ORDER BY created_at DESC LIMIT 20;

-- Suppression list
SELECT * FROM suppressions ORDER BY created_at DESC;
```
