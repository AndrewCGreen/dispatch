# Error Codes

All Dispatch API errors follow a consistent JSON format.

---

## Error Response Format

```json
{
  "error": "ERROR_CODE",
  "message": "Human-readable description of what went wrong",
  "code": "ERROR_CODE"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `error` | string | Machine-readable error code |
| `message` | string | Human-readable error description |
| `code` | string | Same as `error` (included for compatibility) |

---

## Error Code Reference

### Authentication Errors (4xx)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `UNAUTHORIZED` | 401 | Missing `Authorization` header |
| `INVALID_KEY` | 401 | API key is invalid, revoked, or expired |
| `FORBIDDEN` | 403 | Key does not have access to the requested site or resource |

### Validation Errors (4xx)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_JSON` | 400 | Request body is not valid JSON |
| `MISSING_FIELD` | 400 | A required field is missing from the request |
| `INVALID_FIELD` | 400 | A field value is invalid (wrong type, format, or out of range) |
| `SITE_NOT_FOUND` | 404 | The requested site slug does not exist in configuration |
| `NOT_FOUND` | 404 | The requested resource (subscriber, message, template, etc.) was not found |

### Conflict Errors (4xx)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `ALREADY_EXISTS` | 409 | Resource already exists (e.g., subscriber with that email) |
| `RECIPIENT_SUPPRESSED` | 409 | The email address is on the global suppression list |
| `CONFIRMATION_REQUIRED` | 400 | A destructive action requires explicit `"confirm": true` |

### Template Errors (4xx)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `TEMPLATE_ERROR` | 400 | Template not found, failed to parse, or failed to render |
| `RENDER_ERROR` | 400 | Template rendering failed (e.g., missing variable, syntax error) |

### Server Errors (5xx)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INTERNAL` | 500 | Unexpected server error (database failure, backend error, etc.) |

---

## HTTP Status Code Summary

| Status | Meaning | When Used |
|--------|---------|-----------|
| 200 | OK | Successful read, update, or delete operations |
| 201 | Created | Successful creation (subscriber, suppression, webhook) |
| 202 | Accepted | Send request accepted and queued for delivery |
| 400 | Bad Request | Invalid input, missing fields, template errors |
| 401 | Unauthorized | Missing or invalid authentication |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource or site not found |
| 409 | Conflict | Duplicate resource or suppressed recipient |
| 500 | Internal Server Error | Unexpected server failure |

---

## Handling Errors in SDKs

### Python

```python
from dispatch import Dispatch, DispatchError, SuppressedError

client = Dispatch(host="...", api_key="...")

try:
    result = client.send(template="welcome", to="user@example.com")
except SuppressedError:
    print("Recipient is suppressed, skipping")
except DispatchError as e:
    print(f"Error {e.code}: {e.message}")
```

### Go

```go
result, err := client.Send(ctx, req)
if err != nil {
    var apiErr *dispatch.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.Code {
        case "RECIPIENT_SUPPRESSED":
            log.Printf("Skipping suppressed recipient: %s", req.To)
        default:
            log.Printf("API error %s: %s", apiErr.Code, apiErr.Message)
        }
    }
}
```
