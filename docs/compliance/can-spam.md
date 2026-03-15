# CAN-SPAM Compliance

The CAN-SPAM Act is a US federal law that sets rules for commercial email. This document explains how Dispatch helps you comply.

---

## CAN-SPAM Requirements & How Dispatch Helps

### 1. Don't Use False or Misleading Header Information

Your "From," "To," "Reply-To," and routing information must be accurate.

**Dispatch:** The `from`, `from_name`, and `reply_to` fields in `site.yaml` are used directly. Set them accurately.

### 2. Don't Use Deceptive Subject Lines

The subject line must accurately reflect the content of the message.

**Dispatch:** Subject lines are defined in your templates. You control them — keep them honest.

### 3. Identify the Message as an Ad

If your email is an advertisement, you must disclose that.

**Dispatch:** Add a small disclosure to your marketing templates:
```html
<p style="font-size:11px;color:#888;">This is a promotional email from {{.Site.Name}}.</p>
```

### 4. Tell Recipients Where You're Located

Include your valid physical postal address.

**Dispatch:** Add your address to the footer of your base template:
```html
<p style="font-size:11px;color:#888;">
  {{.Site.Name}}<br>
  123 Main St, Suite 100<br>
  San Francisco, CA 94105
</p>
```

### 5. Tell Recipients How to Opt Out

Every commercial email must include a clear unsubscribe mechanism.

**Dispatch:** 
- `{{.UnsubscribeURL}}` — include this in every marketing/newsletter template
- List-Unsubscribe header added automatically (one-click unsubscribe for supported clients)

### 6. Honor Opt-Out Requests Promptly

You must process unsubscribes within 10 business days.

**Dispatch:** Unsubscribes are processed **immediately** — the recipient is added to the suppression list and will not receive further emails.

### 7. Monitor What Others Are Doing on Your Behalf

Even if you hire another company to handle your email marketing, you're responsible.

**Dispatch:** Since you self-host Dispatch, you have full control and visibility into your email operations.

---

## Transactional vs. Commercial Email

CAN-SPAM primarily applies to **commercial** email (marketing, promotions, newsletters). **Transactional** emails (order confirmations, password resets, shipping notifications) are largely exempt but must still:
- Be accurate in their headers
- Not be primarily commercial in nature

In Dispatch, mark transactional lists as `unsubscribable: false` so they don't include misleading unsubscribe options:

```yaml
lists:
  - slug: transactional
    name: Transactional
    unsubscribable: false
```
