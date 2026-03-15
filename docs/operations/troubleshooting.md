# Troubleshooting

Common issues and how to resolve them.

---

## Diagnosis Tools

### Health Check

```bash
curl http://localhost:8080/api/v1/health | python3 -m json.tool
```

### Doctor Command

```bash
dispatch doctor
```

### Log Inspection

```bash
# Docker
docker compose logs --tail=100 dispatch

# systemd
journalctl -u dispatch --since "1 hour ago" | grep -i error

# Follow live
docker compose logs -f dispatch | grep -E "(ERROR|WARN|failed)"
```

---

## Server Won't Start

### Port Already in Use

```
listen tcp 0.0.0.0:8080: bind: address already in use
```

**Fix:** Something else is on port 8080. Change the port in `dispatch.yaml` or stop the conflicting process:
```bash
sudo lsof -i :8080        # Find what's using the port
sudo kill -9 <PID>        # Kill it
```

### Config Parse Error

```
failed to load config: parsing config: yaml: line 12: mapping values are not allowed here
```

**Fix:** YAML syntax error in `dispatch.yaml`. Validate with:
```bash
python3 -c "import yaml; yaml.safe_load(open('dispatch.yaml'))"
```

### Missing Site Config

```
loading site my-store: reading site config: open sites/my-store/site.yaml: no such file or directory
```

**Fix:** The site directory exists but is missing `site.yaml`. Either create the file or remove the directory.

### Database Lock Error

```
database is locked
```

**Fix:** Another process has the SQLite database open. Check for:
- Multiple Dispatch instances running
- sqlite3 CLI left open
- A backup process holding the file

---

## Emails Not Being Sent

### 1. Check the Queue

```bash
curl http://localhost:8080/api/v1/health | python3 -c "import sys,json; h=json.load(sys.stdin); print(h['queue'])"
```

If `failed` is non-zero, there are permanently failed messages. Check logs for error details.

### 2. Check Backend Health

```bash
curl http://localhost:8080/api/v1/health | python3 -c "import sys,json; h=json.load(sys.stdin); print(h['backends'])"
```

If a backend shows an error, see the [SMTP](../backends/smtp.md) or [Resend](../backends/resend.md) troubleshooting sections.

### 3. Check Suppression

```bash
curl "http://localhost:8080/api/v1/suppressions/check/user@example.com" \
  -H "Authorization: Bearer YOUR_KEY"
```

If suppressed, the email won't be sent. Remove from suppression if appropriate.

### 4. Check Queue Workers

If `pending` is growing but `processing` stays at 0, workers may be stuck. Restart Dispatch:
```bash
sudo systemctl restart dispatch
```

---

## API Errors

### 401 Unauthorized

```json
{"error": "INVALID_KEY", "message": "Invalid API key"}
```

**Fix:**
- Check the key matches what's in `dispatch.yaml` or `site.yaml`
- Ensure the `Authorization: Bearer ...` header is correctly formatted
- For site keys, ensure the key matches the site you're accessing

### 403 Forbidden

```json
{"error": "FORBIDDEN", "message": "No access to this site"}
```

**Fix:** You're using a site-scoped key to access a different site. Use the correct site key or the master key.

### 404 Site Not Found

```json
{"error": "SITE_NOT_FOUND", "message": "site \"my-site\" not found"}
```

**Fix:**
- Check that `sites/my-site/site.yaml` exists
- Check that the `slug` field in `site.yaml` matches `my-site`
- Restart Dispatch (config is loaded at startup)

### 400 Template Error

```json
{"error": "TEMPLATE_ERROR", "message": "template \"welcome\" not found in sites/my-site/templates"}
```

**Fix:**
- Check that `sites/my-site/templates/welcome.html` exists
- Check the `templates_dir` setting in `site.yaml`
- Template slug is the filename without `.html`

---

## Template Problems

### Variables Not Rendering

**Symptom:** `{{.Data.name}}` appears literally in the output

**Fix:** You're likely sending a raw string where Go template syntax is expected. Check the template is `.html` not `.txt` when viewed in HTML context.

### Template Renders Wrong Base

**Symptom:** Site-specific layout not applied, shows shared base layout

**Fix:** Name your site base template `_base.html` (with underscore). Site templates override shared templates when they have the same filename.

---

## Backend Problems

### SMTP: 535 Authentication Failed

```
SMTP send failed: 535 5.7.8 Username and Password not accepted
```

**Fix:**
- Gmail: Use an App Password (your regular password won't work with SMTP)
- Outlook: Enable SMTP AUTH in Exchange admin settings
- Check for typos in username/password

### Resend: 422 Domain Not Verified

```
resend API error 422: The from address ... is not verified
```

**Fix:** Your `from` address must use a domain that's verified in Resend's dashboard. Go to Resend → Domains → verify your domain.

### SMTP: Connection Refused

```
SMTP connection failed: dial tcp mail.example.com:587: connection refused
```

**Fix:**
- Check the hostname and port are correct
- Check your firewall allows outbound TCP on port 587
- Test manually: `telnet mail.example.com 587`

---

## Database Problems

### Database Corruption

```
database disk image is malformed
```

**Fix:**
```bash
# Stop Dispatch
sudo systemctl stop dispatch

# Try to repair
sqlite3 /var/lib/dispatch/dispatch.db "PRAGMA integrity_check;"
sqlite3 /var/lib/dispatch/dispatch.db ".recover" > /tmp/recovered.sql
sqlite3 /var/lib/dispatch/dispatch-new.db < /tmp/recovered.sql

# If recovery successful
mv /var/lib/dispatch/dispatch.db /var/lib/dispatch/dispatch.db.corrupt
mv /var/lib/dispatch/dispatch-new.db /var/lib/dispatch/dispatch.db

# Restart
sudo systemctl start dispatch
```

If recovery fails, restore from backup. See [Backup & Recovery](backup.md).

### Disk Full

```
database or disk is full
```

**Fix:**
```bash
# Check disk usage
df -h /var/lib/dispatch

# Check database size
ls -lh /var/lib/dispatch/dispatch.db

# Archive/delete old message records if safe to do so
sqlite3 /var/lib/dispatch/dispatch.db \
  "DELETE FROM messages WHERE queued_at < date('now', '-90 days');"
sqlite3 /var/lib/dispatch/dispatch.db "VACUUM;"
```

---

## Getting Help

1. **Check the logs first** — most errors are self-explanatory in the logs
2. **Run `dispatch doctor`** — catches the most common configuration issues
3. **Check GitHub Issues** — [github.com/dispatch-email/dispatch/issues](https://github.com/dispatch-email/dispatch/issues)
4. **Open a new issue** — include your Dispatch version, OS, config (with secrets redacted), and the full error message
