# Go SDK

The Dispatch Go SDK provides an idiomatic Go client for the Dispatch API.

---

## Installation

```bash
go get github.com/dispatch-email/dispatch-go
```

---

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/dispatch-email/dispatch-go"
)

func main() {
    client := dispatch.New(
        "https://mail.yourdomain.com",
        "dsp_site_xxxxxxxxxxxxx",
    )

    result, err := client.Send(context.Background(), &dispatch.SendRequest{
        Template: "welcome",
        To:       "user@example.com",
        Data: map[string]any{
            "name":      "Alex",
            "login_url": "https://myapp.com/login",
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Queued: %s\n", result.ID)
}
```

---

## Configuration

```go
// Basic
client := dispatch.New(host, apiKey)

// With options
client := dispatch.NewWithOptions(host, apiKey, dispatch.Options{
    Timeout: 30 * time.Second,
    Retries: 3,
})

// From environment variables
client := dispatch.NewFromEnv()
// Reads DISPATCH_HOST and DISPATCH_API_KEY
```

---

## Sending Email

### Template Send

```go
result, err := client.Send(ctx, &dispatch.SendRequest{
    Template: "welcome",
    To:       "user@example.com",
    Data: map[string]any{
        "name":      "Alex",
        "login_url": "https://myapp.com/login",
    },
    Tags:     []string{"onboarding"},
    Metadata: map[string]string{"user_id": "usr_123"},
})
```

### Raw Send

```go
result, err := client.SendRaw(ctx, &dispatch.SendRawRequest{
    To:      "user@example.com",
    Subject: "Your order has shipped!",
    HTML:    "<h1>Shipped!</h1><p>Tracking: {{.Data.tracking_id}}</p>",
    Data:    map[string]any{"tracking_id": "1Z999AA10123456784"},
})
```

### Batch Send

```go
result, err := client.BatchSend(ctx, &dispatch.BatchSendRequest{
    Template: "weekly-digest",
    Recipients: []dispatch.Recipient{
        {To: "alice@example.com", Data: map[string]any{"name": "Alice", "count": 5}},
        {To: "bob@example.com",   Data: map[string]any{"name": "Bob",   "count": 12}},
    },
    Tags: []string{"digest"},
})

fmt.Printf("Queued: %d, Suppressed: %d\n", result.Queued, result.Suppressed)
```

---

## Error Handling

```go
import (
    "errors"
    "github.com/dispatch-email/dispatch-go"
)

result, err := client.Send(ctx, req)
if err != nil {
    var apiErr *dispatch.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.Code {
        case "RECIPIENT_SUPPRESSED":
            log.Printf("Skipping suppressed recipient: %s", req.To)
            return nil
        case "TEMPLATE_ERROR":
            log.Printf("Template error: %s", apiErr.Message)
            return err
        default:
            log.Printf("API error %s: %s", apiErr.Code, apiErr.Message)
            return err
        }
    }
    // Network or other non-API error
    return fmt.Errorf("dispatch request failed: %w", err)
}
```

---

## Subscribers

```go
// Add subscriber
err := client.Subscribers.Add(ctx, &dispatch.Subscriber{
    Email: "user@example.com",
    Name:  "Alex Johnson",
    Attributes: map[string]any{"plan": "pro"},
    Lists: []string{"newsletter"},
    Consent: &dispatch.Consent{
        Source: "signup-form",
        IP:     "203.0.113.42",
        URL:    "https://mysite.com/signup",
    },
})

// Get subscriber
sub, err := client.Subscribers.Get(ctx, "user@example.com")

// Remove subscriber
err = client.Subscribers.Remove(ctx, "user@example.com")
```

---

## Suppression List

```go
// Check suppression
suppressed, err := client.Suppressions.Check(ctx, "user@example.com")
if suppressed {
    log.Printf("Email is suppressed")
}

// Add to suppression
err = client.Suppressions.Add(ctx, "user@example.com", dispatch.ReasonManual, "Support ticket #4521")

// Remove from suppression
err = client.Suppressions.Remove(ctx, "user@example.com")
```

---

## GDPR

```go
// Export data
export, err := client.GDPR.Export(ctx, "user@example.com")

// Forget
err = client.GDPR.Forget(ctx, "user@example.com")
```

---

## Health Check

```go
health, err := client.Health(ctx)
fmt.Printf("Status: %s\n", health.Status)
fmt.Printf("Pending: %d\n", health.Queue.Pending)
```

---

## Context and Cancellation

All SDK methods accept a `context.Context`, enabling cancellation and deadline propagation:

```go
// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

result, err := client.Send(ctx, req)

// With cancellation
ctx, cancel := context.WithCancel(context.Background())
go func() {
    <-done
    cancel()
}()
result, err := client.Send(ctx, req)
```

---

## SDK Source

The Go SDK is in a separate repository: [github.com/dispatch-email/dispatch-go](https://github.com/dispatch-email/dispatch-go)
