# Email Templates Guide

Email template customization, branding, and configuration for all GGID email types.

---

## Email Types

| Email | Trigger | Template |
|-------|---------|----------|
| Welcome | User registration | `welcome.html` |
| Email Verification | After registration | `verify-email.html` |
| Password Reset | Password reset request | `password-reset.html` |
| Password Changed | After password change | `password-changed.html` |
| MFA Code | Email OTP login | `mfa-code.html` |
| Magic Link | Passwordless login | `magic-link.html` |
| Account Locked | After 5 failed logins | `account-locked.html` |
| Invitation | Admin invites user | `invitation.html` |

---

## Template Format

Each template has HTML and plain-text versions:

```
templates/
  welcome.html
  welcome.txt
  verify-email.html
  verify-email.txt
  ...
```

The Auth service sends multipart emails (text + HTML). Clients that don't support HTML see the plain text version.

---

## SMTP Configuration

```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=noreply@yourcompany.com
SMTP_PASSWORD=your-app-password
SMTP_FROM_EMAIL=noreply@yourcompany.com
SMTP_FROM_NAME=Your Company IAM
```

### Test SMTP

```bash
# Send test email via API
POST /api/v1/settings/smtp/test
{"to": "admin@example.com"}
```

---

## Brand Customization

### Logo

```bash
PUT /api/v1/settings/branding
{
  "logo_url": "https://cdn.example.com/logo.png",
  "primary_color": "#2563eb",
  "login_bg_color": "#1e293b"
}
```

### Email Template Variables

All templates receive these variables:

| Variable | Example | Description |
|----------|---------|-------------|
| `{{.AppName}}` | Acme IAM | From branding settings |
| `{{.LogoURL}}` | https://cdn.../logo.png | From branding settings |
| `{{.PrimaryColor}}` | #2563eb | From branding settings |
| `{{.RecipientName}}` | John Doe | User's display name |
| `{{.RecipientEmail}}` | john@example.com | User's email |
| `{{.ActionURL}}` | https://iam.../verify?token=... | Click-through URL |
| `{{.Code}}` | 123456 | MFA or verification code |
| `{{.ExpiryMinutes}}` | 30 | Token validity |
| `{{.SupportEmail}}` | support@example.com | From settings |
| `{{.Year}}` | 2024 | Current year |

---

## Custom Templates

Override default templates by placing files in the templates directory:

```bash
# Docker volume mount
docker run -v /my-templates:/templates auth-service

# Environment variable
TEMPLATE_DIR=/templates
```

### Example: Custom Welcome Email

```html
<!-- templates/welcome.html -->
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: Arial, sans-serif; background: #f3f4f6; margin: 0; padding: 20px;">
  <div style="max-width: 600px; margin: 0 auto; background: white; border-radius: 8px; overflow: hidden;">
    <!-- Header -->
    <div style="background: {{.PrimaryColor}}; padding: 24px; text-align: center;">
      <img src="{{.LogoURL}}" alt="{{.AppName}}" style="height: 40px;">
    </div>
    
    <!-- Body -->
    <div style="padding: 32px;">
      <h1 style="color: #1f2937; font-size: 24px;">Welcome to {{.AppName}}!</h1>
      <p style="color: #4b5563; font-size: 16px; line-height: 1.6;">
        Hi {{.RecipientName}},
      </p>
      <p style="color: #4b5563; font-size: 16px; line-height: 1.6;">
        Your account has been created. Please verify your email address to get started.
      </p>
      
      <!-- CTA Button -->
      <div style="text-align: center; margin: 32px 0;">
        <a href="{{.ActionURL}}" 
           style="background: {{.PrimaryColor}}; color: white; padding: 12px 32px; text-decoration: none; border-radius: 6px; font-weight: bold;">
          Verify Email
        </a>
      </div>
      
      <p style="color: #6b7280; font-size: 14px;">
        This link expires in {{.ExpiryMinutes}} minutes.
      </p>
    </div>
    
    <!-- Footer -->
    <div style="background: #f9fafb; padding: 20px; text-align: center;">
      <p style="color: #6b7280; font-size: 12px; margin: 0;">
        © {{.Year}} {{.AppName}}. All rights reserved.
      </p>
    </div>
  </div>
</body>
</html>
```

### Plain Text Version

```text
Welcome to {{.AppName}}!

Hi {{.RecipientName}},

Your account has been created. Please verify your email address:
{{.ActionURL}}

This link expires in {{.ExpiryMinutes}} minutes.

© {{.Year}} {{.AppName}}
```

---

## Localization

Templates support multiple languages:

```
templates/
  en-US/
    welcome.html
    password-reset.html
  zh-CN/
    welcome.html
    password-reset.html
  ja-JP/
    welcome.html
    password-reset.html
```

Language is selected based on the user's `locale` field:

```bash
# Set user locale
PATCH /api/v1/users/{user_id}
{"locale": "zh-CN"}
```

---

## MFA Code Email

```html
<div style="text-align: center; padding: 40px;">
  <h2>Your verification code</h2>
  <div style="font-size: 36px; font-weight: bold; letter-spacing: 8px; color: {{.PrimaryColor}};">
    {{.Code}}
  </div>
  <p style="color: #6b7280; margin-top: 16px;">
    This code expires in {{.ExpiryMinutes}} minutes.
  </p>
</div>
```

---

## Password Reset Email

```html
<div style="padding: 32px;">
  <h1>Reset your password</h1>
  <p>We received a request to reset your password. Click the button below to choose a new password.</p>
  <a href="{{.ActionURL}}" style="background: {{.PrimaryColor}}; color: white; padding: 12px 32px; border-radius: 6px;">
    Reset Password
  </a>
  <p style="color: #6b7280; font-size: 14px;">
    If you didn't request this, you can safely ignore this email.
    This link expires in {{.ExpiryMinutes}} minutes.
  </p>
</div>
```

---

## Best Practices

1. **Always include plain text** — Some clients disable HTML
2. **Test across email clients** — Gmail, Outlook, Apple Mail render differently
3. **Use inline CSS** — Most email clients strip `<style>` tags
4. **Keep emails under 100KB** — Large emails may be clipped
5. **Include clear expiry info** — Users should know when links expire
6. **Don't put sensitive data in email** — Use tokens, not passwords
7. **Brand consistently** — Match email design with login page and console
