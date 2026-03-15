# Python SDK

The Dispatch Python SDK provides a clean, idiomatic interface for sending emails, managing subscribers, and handling compliance from Python applications.

---

## Installation

```bash
pip install dispatch-email
```

Or with Poetry:

```bash
poetry add dispatch-email
```

---

## Quick Start

```python
from dispatch import Dispatch

# Initialize the client
client = Dispatch(
    host="https://mail.yourdomain.com",
    api_key="dsp_site_xxxxxxxxxxxxx"
)

# Send an email
result = client.send(
    template="welcome",
    to="user@example.com",
    data={"name": "Alex", "login_url": "https://myapp.com/login"}
)

print(result.message_id)   # msg_a1b2c3d4e5f6
print(result.status)       # queued
```

---

## Configuration

### Basic Setup

```python
from dispatch import Dispatch

client = Dispatch(
    host="https://mail.yourdomain.com",  # Your Dispatch instance URL
    api_key="dsp_site_xxxxx",            # API key
    timeout=30,                          # Request timeout in seconds (default: 30)
    retries=3,                           # HTTP retry attempts (default: 3)
)
```

### From Environment Variables

```python
import os
from dispatch import Dispatch

client = Dispatch.from_env()
# Reads DISPATCH_HOST and DISPATCH_API_KEY from environment
```

```bash
export DISPATCH_HOST=https://mail.yourdomain.com
export DISPATCH_API_KEY=dsp_site_xxxxx
```

### Site-Scoped Client

When your app only interacts with one site, bind the client to it:

```python
client = Dispatch(host="...", api_key="dsp_site_xxx")
# All operations are scoped to the site in the API key
```

---

## Sending Email

### Template Send

```python
result = client.send(
    template="welcome",
    to="user@example.com",
    data={
        "name": "Alex",
        "login_url": "https://myapp.com/login",
        "plan": "pro"
    },
    tags=["onboarding"],
    metadata={"user_id": "usr_123"}
)

print(result.message_id)    # msg_a1b2c3d4e5f6
print(result.status)        # queued
print(result.queued_at)     # datetime object
```

### Raw Send (No Template)

```python
result = client.send_raw(
    to="user@example.com",
    subject="Your order has shipped!",
    html="<h1>Great news!</h1><p>Your order <strong>{{.Data.order_id}}</strong> has shipped.</p>",
    text="Great news! Your order {{.Data.order_id}} has shipped.",
    data={"order_id": "ORD-1234"}
)
```

### Batch Send

```python
results = client.batch_send(
    template="weekly-digest",
    recipients=[
        {"to": "alice@example.com", "data": {"name": "Alice", "items": 5}},
        {"to": "bob@example.com",   "data": {"name": "Bob",   "items": 12}},
        {"to": "carol@example.com", "data": {"name": "Carol", "items": 0}},
    ],
    tags=["digest", "weekly"]
)

print(results.total)       # 3
print(results.queued)      # 2
print(results.suppressed)  # 1
```

---

## Tracking Delivery

```python
# Check message status
msg = client.messages.get("msg_a1b2c3d4e5f6")
print(msg.status)         # delivered
print(msg.delivered_at)   # datetime

# Quick status check
status = client.messages.status("msg_a1b2c3d4e5f6")
print(status)  # "delivered"
```

---

## Subscribers

```python
# Add a subscriber
sub = client.subscribers.add(
    email="user@example.com",
    name="Alex Johnson",
    attributes={"plan": "pro", "signed_up": "2026-03-14"},
    lists=["newsletter", "product-updates"],
    consent={
        "source": "signup-form",
        "ip": request.remote_addr,      # Pass real IP from your web framework
        "url": "https://mysite.com/signup"
    }
)

print(sub["status"])  # active or pending (if double opt-in)

# Get a subscriber
sub = client.subscribers.get("user@example.com")

# Update attributes
client.subscribers.update(
    email="user@example.com",
    attributes={"plan": "enterprise"},
    lists=["newsletter", "enterprise-features"]
)

# Remove a subscriber
client.subscribers.remove("user@example.com")

# List all subscribers (paginated)
page = client.subscribers.list(page=1, per_page=50, status="active")
for sub in page.subscribers:
    print(sub["email"])
```

---

## Suppression List

```python
# Check if suppressed
if client.suppressions.is_suppressed("user@example.com"):
    print("This email is suppressed")

# Add to suppression
client.suppressions.add(
    email="user@example.com",
    reason="manual",
    note="Requested via support ticket #4521"
)

# Remove from suppression
client.suppressions.remove("user@example.com")

# List all suppressions
page = client.suppressions.list()
for sup in page.data:
    print(f"{sup['email']}: {sup['reason']}")
```

---

## GDPR

```python
# Export all data for an email address
export = client.gdpr.export("user@example.com")
# Returns dict with all subscriber data across all sites

# Forget (right to erasure)
client.gdpr.forget("user@example.com")
# Deletes all data and adds to permanent suppression
```

---

## Error Handling

```python
from dispatch import (
    Dispatch,
    DispatchError,
    SuppressedError,
    TemplateError,
    NotFoundError,
    AuthError,
)

try:
    result = client.send(template="welcome", to="user@example.com")

except SuppressedError:
    # Recipient is on the suppression list — don't retry
    logger.info(f"Skipping suppressed recipient")

except TemplateError as e:
    # Template not found or render error
    logger.error(f"Template error: {e}")

except NotFoundError as e:
    # Site or resource not found
    logger.error(f"Not found: {e}")

except AuthError as e:
    # Invalid or missing API key
    logger.error(f"Auth error: {e}")

except DispatchError as e:
    # Any other Dispatch API error
    logger.error(f"API error {e.code}: {e.message}")
```

---

## Django Integration

```python
# settings.py
DISPATCH_HOST = os.environ.get("DISPATCH_HOST")
DISPATCH_API_KEY = os.environ.get("DISPATCH_API_KEY")

# email_service.py
from django.conf import settings
from dispatch import Dispatch

_client = None

def get_dispatch():
    global _client
    if _client is None:
        _client = Dispatch(
            host=settings.DISPATCH_HOST,
            api_key=settings.DISPATCH_API_KEY
        )
    return _client

# views.py
from .email_service import get_dispatch

def signup(request):
    # ... create user ...
    get_dispatch().send(
        template="welcome",
        to=user.email,
        data={"name": user.get_full_name()},
        metadata={"user_id": str(user.pk)}
    )
```

---

## FastAPI Integration

```python
from fastapi import FastAPI, Depends
from dispatch import Dispatch
import os

app = FastAPI()

def get_dispatch():
    return Dispatch(
        host=os.environ["DISPATCH_HOST"],
        api_key=os.environ["DISPATCH_API_KEY"]
    )

@app.post("/users/signup")
async def signup(
    user: UserCreate,
    dispatch: Dispatch = Depends(get_dispatch)
):
    # ... create user ...
    dispatch.send(
        template="welcome",
        to=user.email,
        data={"name": user.name}
    )
    return {"status": "ok"}
```

---

## Health Check

```python
health = client.health()
print(health["status"])           # healthy
print(health["queue"]["pending"]) # 3
print(health["backends"])         # {"resend": "ok"}
```

---

## Type Hints

The SDK is fully typed. If you use a type checker (mypy, pyright):

```python
from dispatch import Dispatch
from dispatch.types import SendResult, BatchSendResult, Subscriber

client: Dispatch = Dispatch(host="...", api_key="...")
result: SendResult = client.send(template="welcome", to="user@example.com")
```

---

## SDK Source

The Python SDK is in a separate repository: [github.com/dispatch-email/dispatch-python](https://github.com/dispatch-email/dispatch-python)

Contributions welcome — see the [Contributing Guide](../development/contributing.md).
