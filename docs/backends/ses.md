# Amazon SES Backend

> 🚧 **Status:** Planned for v0.2. This document describes the intended implementation.

Amazon Simple Email Service (SES) is the most cost-effective option for high-volume sending at ~$0.10 per 1,000 emails.

---

## Why SES?

- **Cheapest at scale** — $0.10/1,000 emails after the free tier
- **Free tier** — 62,000 emails/month when sending from EC2 (3,000 otherwise)
- **Excellent deliverability** — AWS manages IP reputation
- **Flexible** — supports SMTP relay and HTTP API
- **Built-in bounce/complaint handling** — SNS notifications

---

## Planned Configuration

```yaml
backend: ses
backend_config:
  region: us-east-1                    # AWS region
  access_key_id: "${AWS_ACCESS_KEY_ID}"
  secret_access_key: "${AWS_SECRET_ACCESS_KEY}"
  # OR use IAM role (recommended for EC2/ECS)
  use_instance_role: "true"
```

---

## Prerequisites

### 1. AWS Account

Create an AWS account at [aws.amazon.com](https://aws.amazon.com).

### 2. Verify Your Sending Domain

In the AWS Console → SES → Verified Identities:
1. Click **Create Identity** → **Domain**
2. Enter your domain
3. Add the provided DNS records
4. Wait for verification

### 3. Request Production Access

New SES accounts start in **sandbox mode**, which restricts sending to verified addresses only. Request production access in the SES console. AWS typically approves within 24 hours.

### 4. Create IAM Credentials

Option A: IAM User (for development):
1. IAM → Users → Create User
2. Attach policy: `AmazonSESFullAccess` (or a more restrictive custom policy)
3. Create access key

Option B: IAM Role (for production on EC2/ECS — recommended):
1. Create an IAM role with `AmazonSESSendingAccess` policy
2. Attach the role to your EC2 instance or ECS task
3. Use `use_instance_role: "true"` in config

---

## Minimal IAM Policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ses:SendEmail",
        "ses:SendRawEmail"
      ],
      "Resource": "*"
    }
  ]
}
```

---

## Bounce and Complaint Handling (Planned)

SES fires SNS notifications for bounces and complaints. Dispatch will handle these via a webhook endpoint, automatically adding bounced/complained addresses to the suppression list.

1. SES → Configuration Sets → Create Configuration Set
2. Add Event Destination → SNS
3. Events: Bounces, Complaints
4. Create SNS Topic → Subscribe with HTTP endpoint: `https://mail.yourdomain.com/webhooks/ses`

---

## Cost Estimate

| Volume | Monthly Cost |
|--------|-------------|
| 10,000 emails | $1.00 |
| 100,000 emails | $10.00 |
| 1,000,000 emails | $100.00 |

Plus SES data transfer costs (typically negligible).
