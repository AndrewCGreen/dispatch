# Dispatch Documentation

Welcome to the Dispatch documentation. Dispatch is a self-hosted email orchestration service designed for operators running multiple websites who need centralized, reliable email delivery with built-in compliance.

## Table of Contents

### Getting Started
- [Installation](getting-started/installation.md) — How to install and run Dispatch
- [Quick Start](getting-started/quickstart.md) — Send your first email in 5 minutes
- [Configuration](getting-started/configuration.md) — Main config file reference

### Core Concepts
- [Architecture](concepts/architecture.md) — How Dispatch works internally
- [Sites](concepts/sites.md) — Multi-site management
- [Templates](concepts/templates.md) — Email template system
- [Backends](concepts/backends.md) — Pluggable email delivery backends
- [Queue](concepts/queue.md) — Send queue, retries, and delivery lifecycle

### API Reference
- [Authentication](api/authentication.md) — API keys and authorization
- [Send Emails](api/send.md) — Transactional, raw, and batch sending
- [Subscribers](api/subscribers.md) — Subscriber management
- [Suppressions](api/suppressions.md) — Global suppression list
- [Templates API](api/templates.md) — Template listing and preview
- [Messages](api/messages.md) — Message history and delivery status
- [GDPR](api/gdpr.md) — Data export and right to erasure
- [Webhooks](api/webhooks.md) — Event notifications
- [Health](api/health.md) — Service health monitoring
- [Error Codes](api/errors.md) — Complete error code reference

### Compliance
- [GDPR](compliance/gdpr.md) — GDPR compliance features and configuration
- [CAN-SPAM](compliance/can-spam.md) — CAN-SPAM Act compliance
- [Unsubscribe](compliance/unsubscribe.md) — Unsubscribe handling and RFC 8058

### Backends
- [SMTP](backends/smtp.md) — Direct SMTP delivery
- [Resend](backends/resend.md) — Resend API integration
- [SES](backends/ses.md) — Amazon SES integration (planned)
- [Listmonk](backends/listmonk.md) — Listmonk integration (planned)
- [Custom Backends](backends/custom.md) — Building your own backend

### SDKs
- [Python SDK](sdks/python.md) — Python client library
- [Go SDK](sdks/go.md) — Go client library
- [HTTP/cURL](sdks/http.md) — Raw HTTP examples

### Operations
- [Deployment](operations/deployment.md) — Docker, systemd, reverse proxy
- [Monitoring](operations/monitoring.md) — Health checks, logging, alerting
- [Backup & Recovery](operations/backup.md) — Database backup strategies
- [Scaling](operations/scaling.md) — Horizontal scaling and PostgreSQL migration
- [Security](operations/security.md) — Security best practices
- [Troubleshooting](operations/troubleshooting.md) — Common issues and solutions

### Development
- [Contributing](development/contributing.md) — How to contribute
- [Development Setup](development/setup.md) — Local development environment
- [Testing](development/testing.md) — Running and writing tests
- [Release Process](development/releases.md) — Versioning and release workflow
