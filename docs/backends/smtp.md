# SMTP Backend

The SMTP backend delivers email directly to any SMTP server — your own mail server, a hosting provider's relay, or a transactional email provider that exposes SMTP access.

---

## Configuration

```yaml
# sites/my-site/site.yaml
backend: smtp
backend_config:
  host: smtp.example.com       # SMTP server hostname (required)
  port: "587"                  # Port number as string (required)
  username: "user@example.com" # Auth username (optional)
  password: "${SMTP_PASSWORD}" # Auth password (optional)
  tls: "false"                 # Implicit TLS on port 465 (optional, default: false)
```

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `host` | ✅ | `localhost` | SMTP server hostname or IP |
| `port` | No | `587` | TCP port |
| `username` | No | `""` | SMTP AUTH username (leave empty for unauthenticated) |
| `password` | No | `""` | SMTP AUTH password |
| `tls` | No | `false` | Set to `"true"` to use implicit TLS (SMTPS) on port 465 |

---

## Common SMTP Providers

### Gmail (App Password)

```yaml
backend: smtp
backend_config:
  host: smtp.gmail.com
  port: "587"
  username: "your-email@gmail.com"
  password: "${GMAIL_APP_PASSWORD}"
```

> **Note:** Gmail requires an [App Password](https://support.google.com/accounts/answer/185833) — not your regular password. Enable 2FA first, then generate an App Password.
> **Limitation:** Gmail has strict daily sending limits (500/day for regular accounts).

### Outlook / Microsoft 365

```yaml
backend: smtp
backend_config:
  host: smtp.office365.com
  port: "587"
  username: "your-email@yourdomain.com"
  password: "${OUTLOOK_PASSWORD}"
```

### Fastmail

```yaml
backend: smtp
backend_config:
  host: smtp.fastmail.com
  port: "587"
  username: "your-email@fastmail.com"
  password: "${FASTMAIL_APP_PASSWORD}"
```

### Self-Hosted Postfix

```yaml
backend: smtp
backend_config:
  host: mail.yourdomain.com
  port: "587"
  username: "dispatch@yourdomain.com"
  password: "${POSTFIX_PASSWORD}"
```

### No Auth (Local Relay)

If Dispatch and your SMTP relay are on the same trusted network:

```yaml
backend: smtp
backend_config:
  host: 127.0.0.1
  port: "25"
```

---

## Port Reference

| Port | Protocol | Use Case |
|------|----------|----------|
| 25 | SMTP | Server-to-server (often blocked by ISPs) |
| 465 | SMTPS | Implicit TLS (legacy, but widely supported) |
| 587 | SMTP+STARTTLS | Submission — **recommended** for client-to-server |
| 2525 | SMTP+STARTTLS | Alternative submission (some providers) |

---

## Deliverability Considerations

Using SMTP directly means deliverability depends on your IP reputation and DNS configuration. For best results:

### DNS Records to Configure

**SPF** — authorizes your server to send email for your domain:
```
TXT v=spf1 ip4:YOUR_SERVER_IP include:_spf.google.com ~all
```

**DKIM** — cryptographically signs outgoing emails (requires mail server support):
- Configure in your mail server (Postfix + OpenDKIM, or your provider's dashboard)

**DMARC** — policy for handling authentication failures:
```
TXT _dmarc.yourdomain.com → v=DMARC1; p=quarantine; rua=mailto:dmarc@yourdomain.com
```

### IP Warmup

If you're sending from a new IP address, start slowly to build reputation:
- Week 1: < 1,000 emails/day
- Week 2: < 5,000 emails/day
- Week 3+: Scale up gradually

### When to Use a Relay Instead

Self-hosted SMTP is challenging for deliverability. Consider using Resend or SES as the backend if:
- You don't control your outbound IP (shared hosting)
- You're seeing emails land in spam
- You're not able to set up DKIM

---

## Health Check

The SMTP backend checks health by opening a TCP connection to the configured host and port:

```json
{
  "backends": {
    "smtp": "ok"
  }
}
```

If the connection fails:
```json
{
  "backends": {
    "smtp": "error: SMTP connection failed: dial tcp smtp.example.com:587: connection refused"
  }
}
```

---

## Troubleshooting

### Authentication Failed
```
SMTP send failed: 535 5.7.8 Username and Password not accepted
```
- Check username and password
- For Gmail: ensure you're using an App Password, not your account password
- For Outlook: ensure "SMTP AUTH" is enabled in your Exchange settings

### Connection Refused
```
SMTP connection failed: dial tcp: connection refused
```
- Verify the host and port are correct
- Check that your server's firewall allows outbound connections on that port
- ISPs often block port 25 — use 587 instead

### TLS Certificate Error
```
SMTP send failed: tls: failed to verify certificate
```
- The SMTP server's SSL certificate may be self-signed or expired
- For development: this is acceptable; for production, fix the certificate

### Timeout
```
SMTP connection failed: i/o timeout
```
- Network route issue between Dispatch and the SMTP server
- Check firewall rules, security groups (if cloud-hosted)
