# Send Queue

The send queue is Dispatch's internal mechanism for reliable, asynchronous email delivery with automatic retries.

---

## Overview

When you send an email through the API, it's not delivered immediately. Instead, it's placed in a queue and processed by background workers. This design provides:

- **Reliability** — emails survive server restarts
- **Rate limiting** — prevent overwhelming your backend
- **Retry logic** — automatic retries with exponential backoff
- **Monitoring** — track delivery status of every message

---

## How It Works

### Message Lifecycle

```
API Request → Message Created (queued) → Queue Item Created
                                              │
                                    Worker picks up item
                                              │
                                    ┌─────────▼──────────┐
                                    │  Check suppression  │
                                    │  Load message data  │
                                    │  Get backend        │
                                    │  Send email         │
                                    └─────────┬──────────┘
                                              │
                              ┌───────────────┼───────────────┐
                              │               │               │
                         Success          Temp Fail       Max Retries
                              │               │               │
                    Status: delivered    Schedule retry   Status: failed
                    Remove from queue    Increment count  Remove from queue
                                        Unlock item      Record error
```

### Queue States

| Status | Description |
|--------|-------------|
| **Queued** | Message created, waiting for a worker to pick it up |
| **Locked** | A worker has claimed the item and is processing it |
| **Completed** | Email delivered successfully (item removed from queue) |
| **Failed** | All retries exhausted (item removed, message marked failed) |
| **Stale Lock** | Worker crashed or timed out (lock auto-released after 5 min) |

---

## Configuration

```yaml
queue:
  workers: 4
  retry_max: 3
  retry_backoff: "5m,30m,2h"
```

### Workers

The number of concurrent goroutines processing the queue. Each worker independently polls for items, locks them, and processes them.

| Volume | Recommended Workers |
|--------|-------------------|
| < 100 emails/day | 1–2 |
| 100–1,000 emails/day | 2–4 |
| 1,000–10,000 emails/day | 4–8 |
| > 10,000 emails/day | 8–16 (consider PostgreSQL) |

Workers are lightweight goroutines — even 16 workers use minimal RAM.

### Retry Strategy

When a send fails (backend error, network timeout, rate limit), Dispatch retries with exponential backoff:

```yaml
retry_max: 3                   # 3 total attempts
retry_backoff: "5m,30m,2h"    # Wait 5min, then 30min, then 2h
```

**Retry timeline example:**

```
Attempt 1: Immediate (on first pick-up)
  └─ Fails → schedule retry in 5 minutes

Attempt 2: +5 minutes
  └─ Fails → schedule retry in 30 minutes

Attempt 3: +30 minutes
  └─ Fails → MAX RETRIES REACHED
     └─ Mark message as "failed"
     └─ Record error message
     └─ Remove from queue
```

### Backoff Format

Comma-separated Go duration strings:

```
"5m,30m,2h"          # 5 minutes, 30 minutes, 2 hours
"30s,2m,10m,1h"      # 30 seconds, 2 minutes, 10 minutes, 1 hour
"1m,5m,15m,1h,6h"    # 5 stages of increasing delay
```

If there are more retries than backoff stages, the last duration is reused.

---

## Locking Mechanism

Workers use database-level locking to prevent duplicate sends:

```sql
-- Worker claims items
UPDATE send_queue
SET locked_by = 'worker-2', locked_at = '2026-03-14T22:50:00Z'
WHERE id IN (
  SELECT id FROM send_queue
  WHERE (locked_by IS NULL OR locked_at < ?)  -- unlocked or stale
  AND next_retry <= ?                           -- ready for retry
  ORDER BY next_retry ASC
  LIMIT 10
)
```

### Stale Lock Recovery

If a worker crashes while processing an item, the lock becomes "stale." Stale locks are automatically recovered:

- Lock timeout: **5 minutes**
- Any worker can claim an item with a stale lock
- This prevents items from being permanently stuck

---

## Monitoring

### Health Endpoint

```bash
curl http://localhost:8080/api/v1/health
```

```json
{
  "queue": {
    "pending": 12,     // Items waiting to be processed
    "processing": 3,   // Items currently being processed by workers
    "failed": 0        // Total failed messages (across all time)
  }
}
```

### Message Status

Check individual message delivery status:

```bash
curl http://localhost:8080/api/v1/messages/msg_abc123/status \
  -H "Authorization: Bearer YOUR_KEY"
```

```json
{
  "id": "msg_abc123",
  "status": "delivered"
}
```

### Logs

Queue activity is logged at the `info` level:

```json
{"level":"INFO","msg":"email sent","message_id":"msg_abc123","to":"user@example.com","backend":"resend","backend_id":"re_xyz"}
{"level":"ERROR","msg":"send failed","worker":"worker-1","message_id":"msg_def456","attempt":2,"error":"resend API error 429: rate limit exceeded"}
```

---

## Suppression Enforcement

The queue performs a suppression check before every send — even if the API already checked when the message was created. This catches race conditions:

1. User sends email to `alice@example.com` → passes suppression check → queued
2. Alice unsubscribes (added to suppression list)
3. Queue worker picks up the message → checks suppression → **blocked**

The message is silently completed (removed from queue) without sending.

---

## Database Schema

```sql
-- The send queue table
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

-- Index for efficient worker polling
CREATE INDEX idx_send_queue_next ON send_queue(next_retry);
```

---

## Failure Modes

### Backend Down

If the backend is completely unreachable, all sends will fail and be retried. After `retry_max` attempts, messages are marked as permanently failed. Monitor the health endpoint to detect backend issues early.

### Database Full

If the SQLite database runs out of disk space, new messages can't be created. The queue workers will continue processing existing items. Monitor disk usage.

### Worker Overload

If messages are arriving faster than workers can process them, the queue will grow. Increase `workers` in config or scale the backend. The `pending` count in the health endpoint is the key metric to watch.

### Duplicate Sends

Dispatch uses database-level locking to prevent duplicate sends. However, if a worker sends an email and then crashes before marking it complete, the item will be retried (potentially sending a duplicate). This is rare and acceptable for most use cases. If exact-once delivery is critical, implement idempotency keys in your backend.
