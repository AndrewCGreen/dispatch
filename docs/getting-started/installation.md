# Installation

Dispatch can be installed as a single binary, via Docker, or built from source.

---

## System Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU | 1 core | 2 cores |
| RAM | 256 MB | 512 MB |
| Disk | 100 MB + data | 1 GB+ |
| OS | Linux (amd64, arm64) | Ubuntu 22.04+ / Debian 12+ |
| Go (build only) | 1.22+ | Latest stable |

Dispatch is lightweight by design. It runs comfortably on a Raspberry Pi or a $5/month VPS.

---

## Option 1: Docker (Recommended)

The fastest way to get started:

```bash
# Create project directory
mkdir dispatch && cd dispatch

# Download the compose file
curl -O https://raw.githubusercontent.com/dispatch-email/dispatch/main/docker-compose.yml

# Initialize config
docker run --rm -v $(pwd):/etc/dispatch dispatch init

# Start
docker compose up -d
```

The service will be available at `http://localhost:8080`.

### Docker Image Tags

| Tag | Description |
|-----|-------------|
| `latest` | Latest stable release |
| `x.y.z` | Specific version (e.g., `0.1.0`) |
| `edge` | Latest commit on main branch |

---

## Option 2: Pre-built Binary

Download the latest release for your platform:

```bash
# Linux amd64
curl -LO https://github.com/dispatch-email/dispatch/releases/latest/download/dispatch-linux-amd64
chmod +x dispatch-linux-amd64
sudo mv dispatch-linux-amd64 /usr/local/bin/dispatch

# Linux arm64 (Raspberry Pi, ARM servers)
curl -LO https://github.com/dispatch-email/dispatch/releases/latest/download/dispatch-linux-arm64
chmod +x dispatch-linux-arm64
sudo mv dispatch-linux-arm64 /usr/local/bin/dispatch

# macOS (Apple Silicon)
curl -LO https://github.com/dispatch-email/dispatch/releases/latest/download/dispatch-darwin-arm64
chmod +x dispatch-darwin-arm64
sudo mv dispatch-darwin-arm64 /usr/local/bin/dispatch
```

Verify the installation:

```bash
dispatch version
# dispatch 0.1.0-dev
```

---

## Option 3: Build from Source

Requires Go 1.22+ and a C compiler (for SQLite):

```bash
# Clone
git clone https://github.com/dispatch-email/dispatch.git
cd dispatch

# Build
CGO_ENABLED=1 go build -o dispatch .

# Install globally (optional)
sudo mv dispatch /usr/local/bin/
```

### Build Dependencies

- **Go 1.22+** — [install instructions](https://go.dev/doc/install)
- **GCC or musl-dev** — required for SQLite CGO bindings
  - Ubuntu/Debian: `apt install gcc`
  - Alpine: `apk add gcc musl-dev`
  - macOS: Xcode command line tools (usually pre-installed)

---

## Post-Installation

After installing, initialize your configuration:

```bash
# Create config directory and default files
dispatch init
```

This creates:
- `dispatch.yaml` — main configuration file
- `sites/` — directory for site configurations
- `shared/templates/` — shared email template layouts
- `data/` — database storage directory

Next: [Quick Start →](quickstart.md)

---

## Uninstalling

**Binary:**
```bash
sudo rm /usr/local/bin/dispatch
rm -rf /path/to/your/dispatch/config
```

**Docker:**
```bash
docker compose down -v  # -v removes the data volume too
```
