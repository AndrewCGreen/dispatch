# Scaling

Dispatch is designed for self-hosted deployments but can scale horizontally when needed.

---

## When to Scale

Most self-hosted deployments will **never need to scale** Dispatch horizontally. A single instance on modest hardware handles:

- **SQLite:** ~10,000–50,000 emails/day (single node)
- **PostgreSQL:** 100,000+ emails/day (single node)
- **Horizontal:** Unlimited with PostgreSQL and multiple nodes

Before scaling out, try scaling up:
- More workers (`queue.workers: 8–16`)
- Faster backend (Resend → SES)
- Better disk I/O (SSD for SQLite)

---

## Vertical Scaling

The simplest optimization — increase resources on the existing node:

```yaml
# dispatch.yaml
queue:
  workers: 8       # Default: 4. Increase for more concurrent sends.
```

Worker count guidelines:
- 1–4 workers: < 10,000 emails/day
- 4–8 workers: 10,000–50,000 emails/day
- 8–16 workers: 50,000–200,000 emails/day

---

## Migrating from SQLite to PostgreSQL

SQLite is the limiting factor for very high volumes. Migrate to PostgreSQL:

### 1. Set Up PostgreSQL

```bash
# Install PostgreSQL
sudo apt install postgresql postgresql-contrib

# Create database and user
sudo -u postgres psql <<EOF
CREATE USER dispatch WITH PASSWORD 'strong_password';
CREATE DATABASE dispatch OWNER dispatch;
EOF
```

### 2. Export Existing Data

```bash
# Dump SQLite to SQL
sqlite3 /var/lib/dispatch/dispatch.db .dump > /tmp/dispatch-dump.sql
```

### 3. Convert and Import

SQLite and PostgreSQL SQL syntax differ slightly. Use `pgloader`:

```bash
# Install pgloader
sudo apt install pgloader

# Convert SQLite to PostgreSQL
pgloader sqlite:///var/lib/dispatch/dispatch.db postgresql://dispatch:password@localhost/dispatch
```

### 4. Update Config

```yaml
# dispatch.yaml
database:
  driver: postgres
  dsn: "postgres://dispatch:password@localhost:5432/dispatch?sslmode=disable"
```

### 5. Run Migrations

```bash
dispatch migrate
```

### 6. Restart

```bash
sudo systemctl restart dispatch
```

---

## Horizontal Scaling (Multiple Nodes)

Run multiple Dispatch instances pointing at the same PostgreSQL database:

```
                    ┌─────────────────┐
                    │   Load Balancer │
                    │  (Nginx/HAProxy)│
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
     ┌────────▼────┐ ┌───────▼─────┐ ┌─────▼───────┐
     │  Dispatch 1 │ │  Dispatch 2 │ │  Dispatch 3 │
     │  (API only) │ │  (API only) │ │  (API+Queue)│
     └─────────────┘ └─────────────┘ └─────────────┘
              │              │              │
              └──────────────┼──────────────┘
                             │
                    ┌────────▼────────┐
                    │   PostgreSQL    │
                    └─────────────────┘
```

### Configuration

All nodes point to the same PostgreSQL instance:

```yaml
# dispatch.yaml (same on all nodes)
database:
  driver: postgres
  dsn: "postgres://dispatch:password@db-host:5432/dispatch"

queue:
  workers: 4   # Per-node worker count
```

### Queue Worker Design

The queue uses database-level locking, so multiple nodes can safely process the queue simultaneously without duplicate sends. Each worker:
1. Claims items by setting `locked_by = 'worker-hostname-N'`
2. Processes them exclusively
3. Releases/completes the lock

### Load Balancer Configuration (Nginx)

```nginx
upstream dispatch {
    least_conn;
    server 10.0.0.1:8080;
    server 10.0.0.2:8080;
    server 10.0.0.3:8080;
}

server {
    location / {
        proxy_pass http://dispatch;
    }
}
```

### Health Check for Load Balancer

```nginx
upstream dispatch {
    server 10.0.0.1:8080;
    server 10.0.0.2:8080;
    
    # Remove nodes that fail health checks
    check interval=10s fails=3 passes=2;
    check_http_send "GET /api/v1/health HTTP/1.0\r\n\r\n";
    check_http_expect_alive http_200;
}
```

---

## Backend Rate Limits

Email providers impose their own rate limits. Dispatch's queue respects these through retry logic, but you should configure workers to stay within limits:

| Provider | Rate Limit | Recommended Workers |
|----------|-----------|---------------------|
| Resend (free) | 100/day | 1 |
| Resend (Pro) | 50,000/month | 4 |
| SES | 14/second (default) | 4–8 |
| SES (increased) | Negotiated | Scale freely |
| SMTP | Provider-dependent | 1–2 |
