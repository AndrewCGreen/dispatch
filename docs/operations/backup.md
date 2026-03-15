# Backup & Recovery

Dispatch's state is stored in a single database file (SQLite) or a PostgreSQL database. Backups are straightforward.

---

## What to Back Up

| Data | Location | Criticality |
|------|----------|-------------|
| Database | `data/dispatch.db` (SQLite) | 🔴 Critical |
| Config files | `dispatch.yaml`, `sites/` | 🟡 Important |
| Templates | `sites/*/templates/` | 🟡 Important |
| Shared templates | `shared/` | 🟡 Important |

**Config and templates** should be in version control (git). The database is the only runtime state.

---

## SQLite Backup

### Method 1: File Copy (Simple)

SQLite's WAL mode allows safe hot backups — just copy the file while Dispatch is running:

```bash
# Simple copy (safe with WAL mode)
cp /var/lib/dispatch/dispatch.db /backups/dispatch-$(date +%Y%m%d-%H%M%S).db

# Or with compression
sqlite3 /var/lib/dispatch/dispatch.db ".backup /tmp/dispatch.db" && \
  gzip -c /tmp/dispatch.db > /backups/dispatch-$(date +%Y%m%d-%H%M%S).db.gz && \
  rm /tmp/dispatch.db
```

### Method 2: SQLite Dump

Creates a portable SQL dump:

```bash
sqlite3 /var/lib/dispatch/dispatch.db .dump | gzip > /backups/dispatch-$(date +%Y%m%d).sql.gz
```

### Automated Daily Backups

```bash
# /usr/local/bin/dispatch-backup.sh
#!/bin/bash

BACKUP_DIR="/backups/dispatch"
DB_PATH="/var/lib/dispatch/dispatch.db"
RETAIN_DAYS=30

mkdir -p "$BACKUP_DIR"

# Create backup
BACKUP_FILE="$BACKUP_DIR/dispatch-$(date +%Y%m%d-%H%M%S).db.gz"
sqlite3 "$DB_PATH" ".backup /tmp/dispatch-backup.db" && \
  gzip -c /tmp/dispatch-backup.db > "$BACKUP_FILE" && \
  rm /tmp/dispatch-backup.db

echo "Backup created: $BACKUP_FILE"

# Remove old backups
find "$BACKUP_DIR" -name "*.db.gz" -mtime +$RETAIN_DAYS -delete
echo "Removed backups older than $RETAIN_DAYS days"
```

Add to cron:
```
0 2 * * * /usr/local/bin/dispatch-backup.sh >> /var/log/dispatch-backup.log 2>&1
```

### Remote Backup (rclone)

Sync backups to S3, B2, or any cloud storage:

```bash
# Install rclone and configure a remote (e.g., "s3")
rclone copy /backups/dispatch s3:your-bucket/dispatch-backups/ \
  --include "*.gz" \
  --min-age 1m

# One-liner: backup and sync
sqlite3 /var/lib/dispatch/dispatch.db ".backup /tmp/dispatch.db" && \
  gzip /tmp/dispatch.db && \
  rclone move /tmp/dispatch.db.gz s3:your-bucket/dispatch-backups/dispatch-$(date +%Y%m%d).db.gz
```

---

## PostgreSQL Backup

If you're using PostgreSQL:

```bash
# pg_dump
pg_dump dispatch | gzip > /backups/dispatch-$(date +%Y%m%d).sql.gz

# Restore
gunzip -c /backups/dispatch-20260314.sql.gz | psql dispatch

# Continuous archiving with WAL (for point-in-time recovery)
# Configure in postgresql.conf:
# archive_mode = on
# archive_command = 'rclone copy %p s3:your-bucket/pg-wal/%f'
```

---

## Recovery

### Restore SQLite from Backup

```bash
# Stop Dispatch
sudo systemctl stop dispatch
# or
docker compose stop dispatch

# Restore backup
gunzip -c /backups/dispatch-20260314-020000.db.gz > /var/lib/dispatch/dispatch.db

# Fix permissions
sudo chown dispatch:dispatch /var/lib/dispatch/dispatch.db

# Start Dispatch
sudo systemctl start dispatch
# or
docker compose start dispatch
```

### Verify Restore

```bash
# Check that the database is healthy
sqlite3 /var/lib/dispatch/dispatch.db "PRAGMA integrity_check;"
# Expected output: ok

# Check record counts
sqlite3 /var/lib/dispatch/dispatch.db "SELECT COUNT(*) FROM subscribers;"
sqlite3 /var/lib/dispatch/dispatch.db "SELECT COUNT(*) FROM messages;"
```

---

## Backup Testing

**Test your backups regularly.** A backup you haven't tested is not a backup.

```bash
# Monthly: restore to a test location and verify
gunzip -c /backups/dispatch-latest.db.gz > /tmp/dispatch-test.db
sqlite3 /tmp/dispatch-test.db "PRAGMA integrity_check;"
sqlite3 /tmp/dispatch-test.db "SELECT COUNT(*) FROM subscribers;"
# Should return same counts as production
rm /tmp/dispatch-test.db
```

---

## Backup Retention Strategy

| Backup Type | Frequency | Retention |
|-------------|-----------|-----------|
| Hourly snapshot | Every hour | 24 backups (1 day) |
| Daily backup | Every day at 2AM | 30 days |
| Weekly backup | Every Sunday | 3 months |
| Monthly backup | 1st of month | 1 year |

Implement with a shell script that copies different backups to different directories with different retention policies.
