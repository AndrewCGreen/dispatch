# Deployment

Production deployment guide for Dispatch, covering Docker, systemd, and reverse proxy configuration.

---

## Option 1: Docker Compose (Recommended)

The simplest production setup — everything in one file.

### docker-compose.yml

```yaml
version: "3.8"

services:
  dispatch:
    image: ghcr.io/dispatch-email/dispatch:latest
    restart: unless-stopped
    ports:
      - "127.0.0.1:8080:8080"    # Bind to localhost only (behind reverse proxy)
    volumes:
      - ./dispatch.yaml:/etc/dispatch/dispatch.yaml:ro
      - ./sites:/etc/dispatch/sites:ro
      - ./shared:/etc/dispatch/shared:ro
      - dispatch-data:/var/lib/dispatch
    environment:
      - DISPATCH_MASTER_KEY=${DISPATCH_MASTER_KEY}
      - RESEND_API_KEY=${RESEND_API_KEY}
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/api/v1/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s

volumes:
  dispatch-data:
```

### .env (keep out of git)

```
DISPATCH_MASTER_KEY=dsp_master_your_secret_key_here
RESEND_API_KEY=re_xxxxxxxxxxxxxxxxxx
```

### Start / Stop / Update

```bash
# Start
docker compose up -d

# View logs
docker compose logs -f dispatch

# Stop
docker compose down

# Update to latest version
docker compose pull && docker compose up -d

# Restart
docker compose restart dispatch
```

---

## Option 2: systemd (Binary)

For running the binary directly on Linux without Docker.

### 1. Install the Binary

```bash
curl -LO https://github.com/dispatch-email/dispatch/releases/latest/download/dispatch-linux-amd64
chmod +x dispatch-linux-amd64
sudo mv dispatch-linux-amd64 /usr/local/bin/dispatch
```

### 2. Create Service User

```bash
sudo useradd --system --shell /bin/false --home /var/lib/dispatch dispatch
sudo mkdir -p /var/lib/dispatch /etc/dispatch
sudo chown dispatch:dispatch /var/lib/dispatch
```

### 3. Install Config Files

```bash
sudo mkdir -p /etc/dispatch/sites /etc/dispatch/shared/templates
sudo cp dispatch.yaml /etc/dispatch/
sudo cp -r sites/ /etc/dispatch/
sudo cp -r shared/ /etc/dispatch/
sudo chown -R dispatch:dispatch /etc/dispatch
```

### 4. Create Environment File

```bash
sudo tee /etc/dispatch/env > /dev/null <<EOF
DISPATCH_MASTER_KEY=dsp_master_your_secret_key_here
RESEND_API_KEY=re_xxxxxxxxxxxxxxxxxx
EOF
sudo chmod 600 /etc/dispatch/env
sudo chown dispatch:dispatch /etc/dispatch/env
```

### 5. Create systemd Service

```bash
sudo tee /etc/systemd/system/dispatch.service > /dev/null <<EOF
[Unit]
Description=Dispatch Email Service
After=network.target
Documentation=https://github.com/dispatch-email/dispatch

[Service]
Type=simple
User=dispatch
Group=dispatch
WorkingDirectory=/etc/dispatch
EnvironmentFile=/etc/dispatch/env
ExecStart=/usr/local/bin/dispatch serve /etc/dispatch/dispatch.yaml
Restart=on-failure
RestartSec=5s

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=/var/lib/dispatch

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=dispatch

[Install]
WantedBy=multi-user.target
EOF
```

### 6. Enable and Start

```bash
sudo systemctl daemon-reload
sudo systemctl enable dispatch
sudo systemctl start dispatch

# Check status
sudo systemctl status dispatch

# View logs
sudo journalctl -u dispatch -f
```

### 7. Update

```bash
# Download new binary
curl -LO https://github.com/dispatch-email/dispatch/releases/latest/download/dispatch-linux-amd64
chmod +x dispatch-linux-amd64
sudo mv dispatch-linux-amd64 /usr/local/bin/dispatch

# Restart service
sudo systemctl restart dispatch
```

---

## Reverse Proxy

Always run Dispatch behind a reverse proxy in production — it handles TLS, compression, and acts as a security buffer.

### Caddy (Recommended — Auto TLS)

```caddyfile
# /etc/caddy/Caddyfile

mail.yourdomain.com {
    reverse_proxy localhost:8080

    # Optional: compress responses
    encode gzip

    # Optional: access logging
    log {
        output file /var/log/caddy/dispatch.log
    }
}
```

Start Caddy: `sudo systemctl start caddy`

Caddy automatically obtains and renews Let's Encrypt certificates. No configuration needed.

### Nginx

```nginx
# /etc/nginx/sites-available/dispatch

server {
    listen 80;
    server_name mail.yourdomain.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name mail.yourdomain.com;

    # SSL (manage with certbot or similar)
    ssl_certificate /etc/letsencrypt/live/mail.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/mail.yourdomain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers off;

    # Security headers
    add_header Strict-Transport-Security "max-age=63072000" always;
    add_header X-Content-Type-Options nosniff;
    add_header X-Frame-Options DENY;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Timeouts
        proxy_connect_timeout 30s;
        proxy_read_timeout 30s;
    }
}
```

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/dispatch /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx

# Get SSL certificate
sudo certbot --nginx -d mail.yourdomain.com
```

---

## Raspberry Pi Deployment

Dispatch runs well on a Raspberry Pi (arm64). Use the arm64 binary:

```bash
curl -LO https://github.com/dispatch-email/dispatch/releases/latest/download/dispatch-linux-arm64
chmod +x dispatch-linux-arm64
sudo mv dispatch-linux-arm64 /usr/local/bin/dispatch
```

Then follow the systemd instructions above. A Pi 4 with 2GB RAM can comfortably handle thousands of emails per day.

**Resource usage (typical):**
- RAM: ~15–30 MB
- CPU: < 1% at idle
- Disk: ~5 MB per 10,000 messages (SQLite)

---

## Firewall

Open only the ports you need:

```bash
# UFW (Ubuntu/Debian)
sudo ufw allow 22/tcp      # SSH
sudo ufw allow 80/tcp      # HTTP (for redirect)
sudo ufw allow 443/tcp     # HTTPS
sudo ufw enable

# Block direct access to Dispatch port (it's behind Nginx/Caddy)
# Port 8080 should NOT be open to the internet
```

---

## Deployment Checklist

Before going live:

- [ ] `master_key` is set to a strong random value (not the default)
- [ ] Site API keys are unique and strong
- [ ] HTTPS configured via reverse proxy
- [ ] Port 8080 is NOT exposed to the internet
- [ ] Sending domain is verified with your backend (Resend/SES)
- [ ] DNS records configured: SPF, DKIM, DMARC
- [ ] `dispatch doctor` passes all checks
- [ ] Database backup scheduled (see [Backup Guide](backup.md))
- [ ] Monitoring configured (see [Monitoring Guide](monitoring.md))
- [ ] Test send works end-to-end
