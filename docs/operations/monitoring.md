# Monitoring

Keep Dispatch healthy with logging, health checks, and alerting.

---

## Health Endpoint

The simplest monitoring: poll the health endpoint.

```bash
curl https://mail.yourdomain.com/api/v1/health
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
    "resend": "ok"
  }
}
```

**Alert triggers:**
- `status` is not `"healthy"`
- `queue.failed` is increasing over time
- `queue.pending` grows continuously (workers may be stuck)
- Any backend shows an error

---

## Logs

### Log Format

Dispatch logs in structured JSON by default:

```json
{"time":"2026-03-14T22:50:00Z","level":"INFO","msg":"request","method":"POST","path":"/api/v1/sites/my-site/send","status":202,"duration":"1.2ms","remote":"127.0.0.1:52411"}
{"time":"2026-03-14T22:50:01Z","level":"INFO","msg":"email sent","message_id":"msg_abc123","to":"user@example.com","backend":"resend","backend_id":"re_xxx"}
{"time":"2026-03-14T22:50:05Z","level":"ERROR","msg":"send failed","worker":"worker-2","message_id":"msg_def456","attempt":1,"error":"resend API error 429: rate limited"}
```

### Viewing Logs

**Docker:**
```bash
docker compose logs -f dispatch
docker compose logs --since 1h dispatch
```

**systemd:**
```bash
journalctl -u dispatch -f
journalctl -u dispatch --since "1 hour ago"
journalctl -u dispatch --since "2026-03-14" --until "2026-03-15"
```

### Log Levels

| Level | What's Logged |
|-------|--------------|
| `debug` | Template rendering details, queue poll cycles, all SQL queries |
| `info` | All HTTP requests, emails sent, queue operations (default) |
| `warn` | Retries, degraded backends, slow operations |
| `error` | Send failures, database errors, unhandled exceptions |

---

## Uptime Monitoring

### UptimeRobot (Free)

1. Create a new monitor: **HTTP(S)**
2. URL: `https://mail.yourdomain.com/api/v1/health`
3. Monitoring interval: 5 minutes
4. Alert contacts: your email/SMS

### Healthchecks.io

Add a cron-style check that Dispatch pings:

```bash
# In your cron or a simple script:
curl -fsS --retry 3 https://hc-ping.com/YOUR-UUID > /dev/null
```

Or configure Dispatch to ping it every N minutes (via heartbeat system).

### Simple Shell Monitoring

```bash
#!/bin/bash
# /usr/local/bin/dispatch-monitor.sh

HEALTH=$(curl -sf https://mail.yourdomain.com/api/v1/health)
if [ $? -ne 0 ]; then
    echo "Dispatch health check failed!" | mail -s "ALERT: Dispatch Down" admin@yourdomain.com
    exit 1
fi

STATUS=$(echo $HEALTH | python3 -c "import sys,json; print(json.load(sys.stdin)['status'])")
if [ "$STATUS" != "healthy" ]; then
    echo "Dispatch status: $STATUS" | mail -s "ALERT: Dispatch Degraded" admin@yourdomain.com
fi
```

Add to cron:
```
*/5 * * * * /usr/local/bin/dispatch-monitor.sh
```

---

## Prometheus Metrics (Planned)

Prometheus-compatible metrics endpoint planned for v0.3:

```
GET /metrics

# HELP dispatch_messages_total Total messages by status
# TYPE dispatch_messages_total counter
dispatch_messages_total{status="delivered"} 12453
dispatch_messages_total{status="failed"} 12
dispatch_messages_total{status="suppressed"} 87

# HELP dispatch_queue_depth Current queue depth
# TYPE dispatch_queue_depth gauge
dispatch_queue_depth{state="pending"} 3
dispatch_queue_depth{state="processing"} 1

# HELP dispatch_backend_healthy Backend health status (1=ok, 0=error)
# TYPE dispatch_backend_healthy gauge
dispatch_backend_healthy{backend="resend"} 1
```

---

## Key Metrics to Watch

| Metric | Warning | Critical | Action |
|--------|---------|----------|--------|
| `queue.failed` growing | +5/hour | +50/hour | Check backend, review errors |
| `queue.pending` growing | >100 | >1000 | Increase workers or check backend |
| Response time | >500ms | >2s | Check DB, check system load |
| Backend health | degraded | down | Check credentials and connectivity |
| Disk usage | >70% | >90% | Archive old messages or increase disk |

---

## Alerting Best Practices

1. **Alert on symptoms, not causes** — "emails not being delivered" matters more than "CPU is high"
2. **Alert on queue.failed rate** — the clearest signal something is wrong
3. **Alert on backend health** — catch provider outages early
4. **Don't alert on queue.pending alone** — some pending is normal; alert only if it's growing
5. **Set a dead man's switch** — alert if no emails sent in X hours (may indicate a stuck queue)
