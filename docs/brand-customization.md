# GGID Brand Customization Guide

How to customize the GGID Admin Console and login pages to match your brand.

---

## Table of Contents

- [Console Branding](#console-branding)
- [Login Page Branding](#login-page-branding)
- [Email Templates](#email-templates)
- [Custom Domain](#custom-domain)
- [Programmatic Configuration](#programmatic-configuration)

---

## Console Branding

### Via Admin Console (Settings > Branding)

Navigate to **Settings > Branding** in the Admin Console to configure:

| Setting | Description | Default |
|---------|-------------|---------|
| Platform Name | Displayed in header and browser title | GGID |
| Logo URL | Logo image (SVG or PNG, max 2MB) | GGID logo |
| Primary Color | Hex color for buttons, links, active states | `#2563EB` (blue) |
| Favicon URL | Browser tab icon | GGID favicon |
| Login Background | Background image or color for login page | `#F8FAFC` |
| Custom CSS | Additional CSS overrides (max 10KB) | _(empty)_ |
| Support Email | Shown in error messages and footer | `support@example.com` |
| Documentation URL | Help link in the console | GGID docs URL |
| Footer Text | Custom text in console footer | _(empty)_ |

### Color Palette

GGID uses a Tailwind CSS-based design system. Customize the primary color:

| Color Role | CSS Variable | Usage |
|------------|-------------|-------|
| Primary | `--color-primary` | Buttons, active nav, links |
| Primary Hover | `--color-primary-hover` | Button hover state |
| Primary Light | `--color-primary-light` | Background accents |
| Success | `--color-success` | Active status, confirmations |
| Warning | `--color-warning` | Pending status, warnings |
| Danger | `--color-danger` | Delete buttons, errors |
| Dark | `--color-dark` | Sidebar, header background |
| Light | `--color-light` | Page background |

### Recommended Color Schemes

**Corporate Blue:**
```json
{
  "primary_color": "#1e40af",
  "primary_hover_color": "#1e3a8a",
  "primary_light_color": "#dbeafe",
  "dark_color": "#1e293b"
}
```

**Tech Green:**
```json
{
  "primary_color": "#059669",
  "primary_hover_color": "#047857",
  "primary_light_color": "#d1fae5",
  "dark_color": "#064e3b"
}
```

**Brand Purple:**
```json
{
  "primary_color": "#7c3aed",
  "primary_hover_color": "#6d28d9",
  "primary_light_color": "#ede9fe",
  "dark_color": "#2e1065"
}
```

### Custom CSS

Override any style with custom CSS:

```css
/* Rounded buttons */
.btn-primary {
  border-radius: 24px;
  font-weight: 600;
}

/* Custom sidebar gradient */
.sidebar {
  background: linear-gradient(180deg, #1a1a2e 0%, #16213e 100%);
}

/* Custom font */
body {
  font-family: 'Inter', -apple-system, sans-serif;
}

/* Hide specific elements */
.feature-flag-banner {
  display: none;
}
```

### Logo Requirements

| Property | Recommendation |
|----------|----------------|
| Format | SVG (preferred) or PNG with transparency |
| Dimensions | 200×40px (header), 32×32px (favicon) |
| Max size | 2MB |
| Background | Transparent (logo on dark and light backgrounds) |
| Hosting | Serve from your CDN or GGID Console `/public/` |

---

## Login Page Branding

### Hosted Login Pages

GGID provides hosted login pages at:
- `/login` — Username/password login
- `/register` — User registration
- `/forgot-password` — Password reset request

These pages respect the branding settings configured in the Console.

### Login Page Layout

```
┌─────────────────────────────────────────────┐
│                                             │
│              [Your Logo Here]                │
│          Your Platform Name                  │
│                                             │
│     ┌─────────────────────────────────┐     │
│     │  Username                        │     │
│     ├─────────────────────────────────┤     │
│     │  Password                        │     │
│     ├─────────────────────────────────┤     │
│     │        [ Sign In ]               │     │
│     └─────────────────────────────────┘     │
│                                             │
│     or continue with                       │
│     [Google] [GitHub] [Microsoft]          │
│                                             │
│     Forgot password? · Need help?          │
│                                             │
│   © 2024 Your Company. All rights reserved. │
└─────────────────────────────────────────────┘
```

### Background Customization

```json
{
  "login_background_type": "gradient",
  "login_background_value": "linear-gradient(135deg, #667eea 0%, #764ba2 100%)"
}
```

Options:
- `gradient` — CSS gradient string
- `image` — URL to background image (recommended: 1920×1080, < 500KB)
- `color` — Solid hex color

### Social Login Button Display

Control which social providers appear on the login page:

```json
{
  "social_providers": ["google", "github", "microsoft"],
  "social_button_order": ["google", "github", "microsoft"]
}
```

---

## Email Templates

Customize the emails sent by GGID (verification, password reset, MFA codes).

### Available Templates

| Template | Trigger | Subject (default) |
|----------|---------|-------------------|
| `email_verification` | User registers | Verify Your Email |
| `password_reset` | Forgot password | Reset Your Password |
| `magic_link` | Passwordless login | Your Magic Link |
| `mfa_code` | MFA email OTP | Your Verification Code |
| `welcome` | Post-registration | Welcome to {Platform Name} |
| `password_changed` | Password changed | Your Password Was Changed |
| `account_locked` | Account locked | Your Account Has Been Locked |

### Template Variables

All templates support these variables:

| Variable | Description |
|----------|-------------|
| `{{.PlatformName}}` | Platform name from branding settings |
| `{{.SupportEmail}}` | Support email from settings |
| `{{.RecipientName}}` | User's display name |
| `{{.RecipientEmail}}` | User's email address |
| `{{.ActionURL}}` | Verification/reset/magic link URL |
| `{{.Code}}` | OTP code (for MFA emails) |
| `{{.TenantName}}` | Tenant display name |
| `{{.ExpiryHours}}` | Link/code expiry in hours |

### Custom Template Example

```html
<!DOCTYPE html>
<html>
<head>
  <style>
    body { font-family: 'Inter', Arial, sans-serif; }
    .header { background: {{.PrimaryColor}}; padding: 20px; }
    .content { padding: 30px; }
    .btn { background: {{.PrimaryColor}}; color: white; padding: 12px 24px; }
  </style>
</head>
<body>
  <div class="header">
    <img src="{{.LogoURL}}" alt="{{.PlatformName}}" />
  </div>
  <div class="content">
    <h1>Verify Your Email</h1>
    <p>Hi {{.RecipientName}},</p>
    <p>Please verify your email address to activate your account:</p>
    <a href="{{.ActionURL}}" class="btn">Verify Email</a>
    <p>This link expires in {{.ExpiryHours}} hours.</p>
  </div>
</body>
</html>
```

### Configure via API

```bash
PUT /api/v1/settings/email-templates
Authorization: Bearer <admin-token>
Content-Type: application/json

{
  "email_verification": {
    "subject": "Confirm your {{.PlatformName}} account",
    "html_body": "<your-custom-html>",
    "text_body": "Visit {{.ActionURL}} to verify your email."
  }
}
```

---

## Custom Domain

### Point Your Domain to GGID

```nginx
# nginx reverse proxy
server {
    listen 443 ssl http2;
    server_name iam.yourcompany.com;

    ssl_certificate /etc/ssl/iam.yourcompany.com.pem;
    ssl_certificate_key /etc/ssl/iam.yourcompany.com.key;

    location / {
        proxy_pass http://ggid-gateway:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

### Configure GGID to Use Custom Domain

```bash
# Set environment variable
GATEWAY_DOMAIN_SUFFIX=.iam.yourcompany.com

# Update OAuth redirect URIs
curl -X PUT "$GW/api/v1/oauth/clients/{client_id}" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "redirect_uris": ["https://app.yourcompany.com/callback",
                       "https://iam.yourcompany.com/oauth/callback"]
  }'
```

---

## Programmatic Configuration

### Get Current Branding

```bash
GET /api/v1/settings/branding
Authorization: Bearer <admin-token>

Response:
{
  "platform_name": "Acme IAM",
  "logo_url": "https://cdn.acme.com/logo.svg",
  "primary_color": "#7c3aed",
  "primary_hover_color": "#6d28d9",
  "favicon_url": "https://cdn.acme.com/favicon.ico",
  "login_background_type": "gradient",
  "login_background_value": "linear-gradient(135deg, #667eea, #764ba2)",
  "custom_css": "/* ... */",
  "support_email": "iam@acme.com",
  "documentation_url": "https://docs.acme.com/iam",
  "footer_text": "© 2024 Acme Corp"
}
```

### Update Branding

```bash
PUT /api/v1/settings/branding
Authorization: Bearer <admin-token>
Content-Type: application/json

{
  "platform_name": "Acme IAM",
  "logo_url": "https://cdn.acme.com/logo.svg",
  "primary_color": "#7c3aed"
}
```

Changes take effect immediately — the Console and login pages re-fetch branding
on each page load.

---

## Internationalization (i18n)

GGID supports multi-language UI with runtime language switching.

### Supported Languages

| Code | Language | Status |
|------|----------|--------|
| `en` | English | Default |
| `zh-CN` | Simplified Chinese | Full |
| `zh-TW` | Traditional Chinese | Full |
| `ja` | Japanese | Full |
| `ko` | Korean | Full |
| `es` | Spanish | Full |
| `de` | German | Full |
| `fr` | French | Full |
| `pt-BR` | Portuguese (Brazil) | Full |

### Configure Default Language

```bash
curl -X PUT $API/api/v1/settings/i18n \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "default_locale": "zh-CN",
    "fallback_locale": "en",
    "available_locales": ["en", "zh-CN", "ja"]
  }'
```

### Login Page Language Detection

GGID detects the user's preferred language in this order:

1. `?lang=zh-CN` query parameter
2. `Accept-Language` header
3. User's `preferred_locale` field (if logged in)
4. Tenant's `default_locale` setting
5. System fallback (`en`)

### Translation File Format

```json
// locales/zh-CN.json
{
  "login.title": "登录",
  "login.username": "用户名",
  "login.password": "密码",
  "login.submit": "登录",
  "login.forgot_password": "忘记密码？",
  "login.mfa_code": "请输入验证码",
  "register.title": "注册",
  "register.email": "邮箱地址",
  "error.invalid_credentials": "用户名或密码错误",
  "error.account_locked": "账户已被锁定，请15分钟后重试"
}
```

### Custom Translations

```bash
# Add custom translations for a tenant
curl -X PUT $API/api/v1/tenants/$TENANT_ID/i18n/zh-CN \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "login.title": "员工登录",
    "login.welcome": "欢迎使用 Acme 内部系统"
  }'
```

### RTL (Right-to-Left) Support

For Arabic and Hebrew:

```bash
curl -X PUT $API/api/v1/settings/i18n \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "available_locales": ["en", "ar", "he"],
    "rtl_locales": ["ar", "he"]
  }'
```

The Console automatically switches to RTL layout for configured locales.
