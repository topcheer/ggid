# Notification Management

Notification system: email/SMS/push channels, template management, per-event
routing, user preferences, suppression lists, and delivery logs.

---

## Table of Contents

- [Channels](#channels)
- [Event Routing](#event-routing)
- [Template Management](#template-management)
- [User Preferences](#user-preferences)
- [Suppression Lists](#suppression-lists)
- [Delivery Logs](#delivery-logs)

---

## Channels

| Channel | Provider | Use Case |
|---------|----------|----------|
| Email | SMTP / SES / SendGrid | Welcome, password reset, MFA codes |
| SMS | Twilio / AWS SNS | MFA codes, urgent alerts |
| Push | FCM / APNs | Mobile app notifications |
| Webhook | HTTP POST | Integration with external systems |
| In-App | WebSocket / SSE | Real-time console notifications |

### Channel Configuration

```yaml
notification:
  email:
    provider: smtp              # smtp, ses, sendgrid
    smtp:
      host: smtp.example.com
      port: 587
      username: noreply@example.com
      password: "${SMTP_PASSWORD}"
      from: noreply@example.com
      tls: true
  sms:
    provider: twilio            # twilio, sns
    twilio:
      account_sid: "${TWILIO_SID}"
      auth_token: "${TWILIO_TOKEN}"
      from: "+15551234567"
  push:
    provider: fcm
    fcm:
      server_key: "${FCM_KEY}"
```

---

## Event Routing

Route events to specific channels:

```yaml
notification:
  routing:
    user.created:
      channels: [email]
      template: welcome_email
    user.password.reset:
      channels: [email]
      template: password_reset
      priority: high
    user.mfa.triggered:
      channels: [sms, email]
      template: mfa_code
      priority: urgent
    user.account_locked:
      channels: [email]
      template: account_locked
      priority: high
    security.token_reuse:
      channels: [email, webhook]
      template: security_alert
      priority: critical
```

### Priority Levels

| Priority | Behavior |
|----------|----------|
| `low` | Queued, sent within 5 min |
| `normal` | Sent within 30 seconds |
| `high` | Sent immediately, retry on failure |
| `urgent` | Sent immediately, multi-channel, escalate |
| `critical` | Sent to all configured channels, admin alerted |

---

## Template Management

### Template Variables

Templates support Go template syntax with context variables:

| Variable | Description |
|---------|-------------|
| `{{.User.Username}}` | Username |
| `{{.User.Email}}` | Email address |
| `{{.User.DisplayName}}` | Display name |
| `{{.Tenant.Name}}` | Tenant name |
| `{{.Action.URL}}` | Action link (reset, verify) |
| `{{.Code}}` | MFA code |
| `{{.Expires}}` | Expiration time |

### Create Template

```bash
curl -X POST .../admin/notifications/templates \
  -d '{
    "name": "welcome_email",
    "channel": "email",
    "subject": "Welcome to {{.Tenant.Name}}",
    "body": "Hi {{.User.DisplayName}},\n\nWelcome! Your account is ready.\n\n{{.Action.URL}}",
    "format": "text"
  }'
```

### HTML Email Template

```bash
curl -X POST .../admin/notifications/templates \
  -d '{
    "name": "password_reset",
    "channel": "email",
    "subject": "Reset your password",
    "body": "<html>...<a href=\"{{.Action.URL}}\">Reset Password</a>...</html>",
    "format": "html"
  }'
```

### Preview Template

```bash
curl -X POST .../admin/notifications/templates/welcome_email/preview \
  -d '{ "user_id": "test-user-id" }'
```

---

## User Preferences

Users can configure their notification preferences:

```bash
curl -X PATCH .../me/notification-preferences \
  -H "Authorization: Bearer $USER_TOKEN" \
  -d '{
    "email": {
      "user.updated": false,
      "security_alert": true,
      "password_reset": true
    },
    "sms": {
      "mfa_code": true,
      "security_alert": true
    }
  }'
```

### Default Preferences

```yaml
notification:
  defaults:
    email:
      all: true                # All email notifications on by default
      security_alert: true     # Cannot be disabled (forced)
    sms:
      all: false               # SMS off by default
      mfa_code: true           # MFA codes always sent
```

### Forced Notifications

Some notifications cannot be disabled by users:
```yaml
notification:
  forced:
    - security_alert
    - password_reset
    - account_locked
```

---

## Suppression Lists

### Global Suppression

```bash
curl -X POST .../admin/notifications/suppress \
  -d '{ "email": "bounced@example.com", "reason": "permanent_bounce" }'
```

### Auto-Suppression Rules

| Trigger | Action |
|---------|--------|
| Hard bounce | Auto-suppress email |
| Spam complaint | Auto-suppress email |
| 5 consecutive SMS failures | Auto-suppress SMS |

---

## Delivery Logs

### Query Delivery Status

```bash
curl ".../admin/notifications/deliveries?event=user.created&limit=50" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```json
{
  "deliveries": [
    {
      "id": "dlv-uuid",
      "event": "user.created",
      "channel": "email",
      "recipient": "jane@example.com",
      "status": "delivered",
      "sent_at": "2024-01-15T10:00:00Z",
      "delivered_at": "2024-01-15T10:00:02Z"
    },
    {
      "id": "dlv-uuid-2",
      "event": "user.mfa.triggered",
      "channel": "sms",
      "recipient": "+15551234567",
      "status": "failed",
      "error": "carrier_rejected",
      "sent_at": "2024-01-15T10:01:00Z"
    }
  ]
}
```

### Delivery Status Values

| Status | Description |
|--------|-------------|
| `queued` | Waiting to be sent |
| `sent` | Handed to provider (SMTP/Twilio) |
| `delivered` | Provider confirmed delivery |
| `failed` | Delivery failed (see error) |
| `suppressed` | Suppressed (bounce/spam) |

### Retry Policy

| Attempt | Delay |
|---------|-------|
| 1 | Immediate |
| 2 | 5 minutes |
| 3 | 30 minutes |
| 4 | 2 hours (final) |
