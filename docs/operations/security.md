# Security

Security best practices for running Dispatch in production.

---

## API Key Management

### Generating Strong Keys

API keys should be long, random, and unique:

```bash
# Generate a strong key (Linux/macOS)
openssl rand -hex 32

# Add the prefix manually
echo "dsp_master_$(openssl rand -hex 32)"
echo "dsp_site_$(openssl rand -hex 32)"
```

### Key Storage

- **Never commit keys to version control** — use environment variables or secrets management
- **Use `.env` files locally**, add `.env` to `.gitignore`
- **Use secrets management in production** — AWS Secrets Manager, HashiCorp Vault, GitHub Secrets, etc.
- **Rotate keys periodically** — create new key, update all consumers, revoke old key

### Principle of Least Privilege

- Give each application the most **restrictive key** that meets its needs
- Web application sending emails → **site-scoped key**
- Admin dashboard reading stats → **read-only key**
- GDPR processing script → **master key** (kept offline, used manually)

---

## Network Security

### Bind to Localhost Only

Dispatch should never be exposed directly to the internet. Always use a reverse proxy:

```yaml
# dispatch.yaml
server:
  host: 127.0.0.1   # ← Not 0.0.0.0
  port: 8080
```

Or in Docker:
```yaml
ports:
  - "127.0.0.1:8080:8080"   # ← Not "8080:8080"
```

### Firewall Rules

```bash
# Block direct access to Dispatch port
sudo ufw deny 8080/tcp

# Only allow HTTP and HTTPS (handled by reverse proxy)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
```

### TLS Configuration

Terminate TLS at the reverse proxy layer. Use modern TLS settings in Nginx/Caddy:

```nginx
ssl_protocols TLSv1.2 TLSv1.3;
ssl_prefer_server_ciphers off;
ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:...;
add_header Strict-Transport-Security "max-age=63072000" always;
```

---

## Database Security

### SQLite File Permissions

```bash
# Only the dispatch user should read/write the database
chmod 600 /var/lib/dispatch/dispatch.db
chown dispatch:dispatch /var/lib/dispatch/dispatch.db
```

### PostgreSQL

```sql
-- Create a dedicated database user with minimal permissions
CREATE USER dispatch_app WITH PASSWORD 'strong_password_here';
CREATE DATABASE dispatch OWNER dispatch_app;
GRANT CONNECT ON DATABASE dispatch TO dispatch_app;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO dispatch_app;
```

### Encryption at Rest

For highly sensitive deployments, use filesystem-level encryption:
- **Linux:** LUKS (`cryptsetup`)
- **Cloud:** EBS encryption (AWS), disk encryption (GCP/Azure)

---

## Secret Management

### Environment Variables (Basic)

```bash
# /etc/dispatch/env (chmod 600)
DISPATCH_MASTER_KEY=dsp_master_xxx
RESEND_API_KEY=re_xxx
```

### Docker Secrets

```yaml
# docker-compose.yml
services:
  dispatch:
    secrets:
      - dispatch_master_key
      - resend_api_key
    environment:
      - DISPATCH_MASTER_KEY_FILE=/run/secrets/dispatch_master_key

secrets:
  dispatch_master_key:
    external: true
  resend_api_key:
    external: true
```

### HashiCorp Vault

```bash
# Store secrets
vault kv put secret/dispatch \
  master_key="dsp_master_xxx" \
  resend_api_key="re_xxx"

# Retrieve in startup script
export DISPATCH_MASTER_KEY=$(vault kv get -field=master_key secret/dispatch)
```

---

## Securing the Unsubscribe Endpoint

The unsubscribe endpoint is public (no auth required) by design. To prevent abuse:

1. **Token-based** — tokens are short-lived and tied to a specific subscriber
2. **One-time use** — tokens invalidate after use (planned for v0.2)
3. **Rate limiting** — limit requests per IP (configure in Nginx/Caddy)

```nginx
# Rate limit unsubscribe endpoint
limit_req_zone $binary_remote_addr zone=unsubscribe:10m rate=10r/m;

location /unsubscribe/ {
    limit_req zone=unsubscribe burst=5 nodelay;
    proxy_pass http://127.0.0.1:8080;
}
```

---

## Security Headers

Add these headers in your reverse proxy:

```nginx
add_header Strict-Transport-Security "max-age=63072000; includeSubDomains" always;
add_header X-Content-Type-Options "nosniff" always;
add_header X-Frame-Options "DENY" always;
add_header X-XSS-Protection "1; mode=block" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
```

---

## Audit Trail

Dispatch maintains an audit trail of compliance-relevant actions in `consent_log`:

```sql
SELECT * FROM consent_log ORDER BY created_at DESC LIMIT 20;
```

Review this regularly for unexpected patterns (e.g., mass unsubscribes, unusual IPs).

---

## Security Checklist

- [ ] API keys are strong random strings (not guessable)
- [ ] Master key stored securely, not in version control
- [ ] Site API keys use principle of least privilege
- [ ] Dispatch binds to `127.0.0.1`, not `0.0.0.0`
- [ ] Reverse proxy configured with HTTPS only
- [ ] HSTS header enabled
- [ ] Port 8080 blocked by firewall
- [ ] Database file permissions are 600
- [ ] Secrets managed via environment variables or secrets manager
- [ ] Backups are encrypted
- [ ] Logs reviewed regularly for anomalies
- [ ] Keys are rotated periodically
