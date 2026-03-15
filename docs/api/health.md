# Health Endpoint

Monitor Dispatch service health, including database connectivity, queue status, and backend availability.

---

## Health Check

```
GET /api/v1/health
```

**No authentication required** — designed for load balancers and monitoring systems.

### Response (200 OK)

```json
{
  "status": "healthy",
  "version": "0.1.0",
  "uptime": "4d 12h 30m 15s",
  "database": "ok",
  "queue": {
    "pending": 3,
    "processing": 1,
    "failed": 0
  },
  "backends": {
    "resend": "ok",
    "smtp": "ok"
  }
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `status` | string | Overall status: `healthy` or `degraded` |
| `version` | string | Dispatch version |
| `uptime` | string | Time since server started |
| `database` | string | Database status: `ok` or error description |
| `queue.pending` | int | Messages waiting to be processed |
| `queue.processing` | int | Messages currently being processed by workers |
| `queue.failed` | int | Total permanently failed messages |
| `backends` | object | Map of backend name → status (`ok` or error) |

### Interpreting Health Status

| Scenario | `status` | Action |
|----------|----------|--------|
| Everything working | `healthy` | None needed |
| Backend unreachable | `degraded` | Check backend connectivity, credentials |
| Database error | `degraded` | Check disk space (SQLite) or DB connectivity |
| High pending count | `healthy` (but concerning) | Consider increasing workers |
| High failed count | `healthy` | Review failed messages, check backend logs |

---

## Using with Monitoring

### Load Balancer Health Check

Point your load balancer at `/api/v1/health` and check for HTTP 200.

### Prometheus / Grafana

Parse the JSON response to extract metrics:
- `dispatch_queue_pending` — gauge
- `dispatch_queue_processing` — gauge
- `dispatch_queue_failed` — counter
- `dispatch_backend_healthy` — boolean per backend

### Uptime Monitoring (UptimeRobot, Healthchecks.io)

```
URL: https://mail.yourdomain.com/api/v1/health
Method: GET
Expected Status: 200
Check Interval: 60 seconds
```

---

## Doctor Command

For a more detailed diagnostic, use the CLI:

```bash
dispatch doctor
```

Checks:
- Config file syntax and required fields
- Database connectivity and migration status
- Site config validity
- Template file existence
- Backend connectivity
- DNS records (SPF, DKIM, DMARC) for sending domains
