# GDPR Compliance

Dispatch provides built-in features to help you comply with the EU General Data Protection Regulation. This document explains what Dispatch handles, what you need to handle in your application, and best practices.

---

## What Dispatch Handles

### 1. Consent Recording

Every subscriber action is logged in an immutable consent log:

```
Action: subscribe
Email: user@example.com
Site: my-store
Source: signup-form
IP: 203.0.113.42
URL: https://mystore.com/signup
Timestamp: 2026-03-14T10:00:00Z
```

This provides evidence of:
- **When** consent was given
- **How** consent was given (source, URL)
- **Where** the person was (IP address)

### 2. Double Opt-In

When `double_optin: true` is set for a site, new subscribers must confirm their email address before receiving emails:

```
Subscribe request → Confirmation email sent → User clicks link → Active
```

This provides the strongest form of consent evidence.

### 3. Unsubscribe Mechanism

- **One-click unsubscribe link** included in all list/campaign emails
- **List-Unsubscribe header** (RFC 8058) added automatically
- **Public unsubscribe page** — no login required
- **Immediate suppression** — no more emails after unsubscribe

### 4. Data Subject Access Request (DSAR)

The `/api/v1/gdpr/export` endpoint returns all data for an email address across all sites. This supports GDPR Article 15 (Right of Access).

### 5. Right to Erasure

The `/api/v1/gdpr/forget` endpoint:
- Deletes all subscriber records across all sites
- Adds the email to the permanent suppression list
- Logs the erasure action for audit purposes
- Blocks any future sends to that address

### 6. Global Suppression

Suppressed emails are checked before every send, ensuring no email is ever sent to someone who has opted out or been forgotten.

---

## What You Must Handle in Your Application

Dispatch manages email-related data. Your application likely holds additional personal data that GDPR applies to:

### User Accounts
- Delete user accounts when processing forget requests
- Export all user data (not just email data) for access requests

### Logs and Analytics
- Review application logs for personal data
- Anonymize or delete analytics data containing email addresses

### Third-Party Services
- If you share email addresses with other services (analytics, CRM, etc.), ensure those are also updated during GDPR requests

### Privacy Policy
- Document what data you collect and why
- Explain how users can request export or deletion
- Specify your lawful basis for sending emails

### Consent UI
- Your signup forms should clearly explain what emails the user will receive
- Provide granular consent options (separate checkboxes for newsletter, marketing, etc.)
- Don't pre-check consent boxes

---

## Data Retention

### What Dispatch Retains

| Data | Retention | Reason |
|------|-----------|--------|
| Subscriber records | Until unsubscribe or forget | Active subscriber management |
| Consent log | Indefinite | Legal requirement — proof of consent |
| Message log | Configurable (default: indefinite) | Audit trail, debugging |
| Suppression list | Indefinite | Prevent re-sending to opted-out addresses |

### Consent Log Preservation

Consent records are **never deleted**, even during a GDPR forget request. GDPR Article 7(1) requires data controllers to demonstrate that consent was obtained. The consent log is your proof. It records:

- That consent was given (subscribe event)
- That consent was withdrawn (unsubscribe event)
- That erasure was performed (forget event)

---

## GDPR Compliance Checklist

### Configuration

- [ ] `compliance.gdpr_enabled: true` (default)
- [ ] `compliance.consent_logging: true` (default)
- [ ] `compliance.suppression_global: true` (default)
- [ ] `double_optin: true` for all marketing/newsletter lists

### Application Integration

- [ ] Pass `consent` information when creating subscribers (source, IP, URL)
- [ ] Implement a "Delete My Data" flow that calls both your app's deletion AND `/api/v1/gdpr/forget`
- [ ] Implement a "Export My Data" flow that calls both your app's export AND `/api/v1/gdpr/export`
- [ ] Respond to GDPR requests within 30 days (legal requirement)

### Templates

- [ ] All list/campaign emails include an unsubscribe link (`{{.UnsubscribeURL}}`)
- [ ] Transactional emails do not include misleading unsubscribe links (password resets should not have unsubscribe)

### Operations

- [ ] Regular review of suppression list
- [ ] Monitor consent log for anomalies
- [ ] Document your data processing activities

---

## Lawful Basis for Sending Email

GDPR requires a lawful basis for processing personal data. For email:

| Email Type | Typical Lawful Basis |
|-----------|---------------------|
| Newsletters | Consent (Article 6(1)(a)) — explicit opt-in required |
| Marketing | Consent — must be freely given, specific, informed, unambiguous |
| Transactional (order confirmation, password reset) | Legitimate interest or Contract performance (Article 6(1)(b/f)) |
| Service notices (ToS changes, security alerts) | Legitimate interest (Article 6(1)(f)) |

---

## International Considerations

### CAN-SPAM (US)

In addition to GDPR, if you send to US recipients, you must comply with CAN-SPAM. Dispatch helps by:
- Including a visible unsubscribe mechanism
- Supporting accurate "From" addressing
- See [CAN-SPAM Compliance](can-spam.md) for details

### CASL (Canada)

Canadian Anti-Spam Legislation requires express or implied consent. Dispatch's double opt-in feature satisfies the express consent requirement.

### PECR (UK)

The UK's Privacy and Electronic Communications Regulations are similar to GDPR for email marketing. Dispatch's consent logging and unsubscribe mechanisms apply.
