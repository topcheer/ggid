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

---

## Font Configuration

Customize the typography of the Admin Console and hosted login pages to align
with your brand guidelines.

### Available Font Settings

| Setting | Description | Default |
|---------|-------------|---------|
| `font_family` | Primary font family CSS declaration | `'Inter', -apple-system, sans-serif` |
| `font_url` | Google Fonts or self-hosted CSS URL | Inter from Google Fonts |
| `heading_font` | Font for headings (h1-h6) | Same as `font_family` |
| `mono_font` | Monospace font for code blocks | `'JetBrains Mono', monospace` |
| `base_font_size` | Root font size | `16px` |
| `heading_weight` | Font weight for headings (300-900) | `600` |
| `body_weight` | Font weight for body text (300-900) | `400` |

### Configure via API

```bash
PUT /api/v1/settings/branding/fonts
Authorization: Bearer <admin-token>
Content-Type: application/json

{
  "font_family": "'Poppins', 'Helvetica Neue', sans-serif",
  "font_url": "https://fonts.googleapis.com/css2?family=Poppins:wght@400;500;600;700&display=swap",
  "heading_font": "'Poppins', sans-serif",
  "heading_weight": 600,
  "body_weight": 400
}
```

Changes take effect immediately. The Console dynamically injects the `<link>`
tag for the font URL and updates CSS variables.

### Google Fonts Example

```json
{
  "font_family": "'Nunito Sans', -apple-system, BlinkMacSystemFont, sans-serif",
  "font_url": "https://fonts.googleapis.com/css2?family=Nunito+Sans:wght@400;600;700;800&display=swap",
  "heading_font": "'Nunito Sans', sans-serif",
  "heading_weight": 800
}
```

### Self-Hosted Fonts

For air-gapped or compliance-restricted deployments, self-host font files:

```bash
# Upload font files
curl -X POST $API/api/v1/assets/fonts \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -F "regular=@/path/to/brand-regular.woff2" \
  -F "bold=@/path/to/brand-bold.woff2" \
  -F "name=my-brand"

# Reference self-hosted font
curl -X PUT $API/api/v1/settings/branding/fonts \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "font_family": "MyBrand, sans-serif",
    "font_url": "/api/v1/assets/fonts/my-brand.css"
  }'
```

The self-hosted CSS is automatically generated:

```css
@font-face {
  font-family: 'MyBrand';
  src: url('/api/v1/assets/fonts/brand-regular.woff2') format('woff2');
  font-weight: 400;
  font-display: swap;
}
@font-face {
  font-family: 'MyBrand';
  src: url('/api/v1/assets/fonts/brand-bold.woff2') format('woff2');
  font-weight: 700;
  font-display: swap;
}
```

### Recommended Font Pairings

| Heading Font | Body Font | Style |
|--------------|-----------|-------|
| Poppins | Inter | Modern, friendly |
| Montserrat | Open Sans | Corporate, clean |
| Playfair Display | Source Sans Pro | Elegant, premium |
| Space Grotesk | Inter | Tech, startup |
| IBM Plex Sans | IBM Plex Sans | Enterprise, trustworthy |
| Noto Sans | Noto Sans | International, neutral |

---

## White-Label Login Widget

Embed a fully branded login experience in your own application using the GGID
login widget. The widget handles authentication flows (password, social, MFA)
while respecting your brand settings.

### Embedding the Widget

```html
<!-- Add the GGID widget script -->
<script src="https://iam.yourcompany.com/widget/ggid-login.js"></script>

<!-- Mount point -->
<div id="ggid-login"></div>

<script>
  GGIDLogin.mount('#ggid-login', {
    tenantId: '00000000-0000-0000-0000-000000000001',

    // Brand overrides (optional, defaults from tenant settings)
    primaryColor: '#7c3aed',
    logoUrl: 'https://cdn.acme.com/logo.svg',
    platformName: 'Acme Portal',

    // Login configuration
    defaultMethod: 'password',         // 'password' | 'magic-link' | 'sso'
    socialProviders: ['google', 'github'],

    // Callbacks
    onSuccess: function(token) {
      console.log('JWT:', token.access_token);
      window.location.href = '/dashboard';
    },
    onError: function(err) {
      console.error('Login failed:', err.message);
    },

    // Styling
    containerClass: 'acme-login-card',
    width: '400px',
    borderRadius: '12px',
  });
</script>
```

### Widget Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `tenantId` | string | _(required)_ | Tenant UUID for multi-tenant |
| `primaryColor` | string | From branding settings | Hex color |
| `logoUrl` | string | From branding settings | Logo image URL |
| `platformName` | string | From branding settings | Display name |
| `defaultMethod` | string | `password` | Default auth method shown |
| `socialProviders` | string[] | From branding settings | Social login buttons |
| `showRememberMe` | boolean | `true` | "Remember me" checkbox |
| `showForgotPassword` | boolean | `true` | Forgot password link |
| `containerClass` | string | — | Custom CSS class for container |
| `width` | string | `400px` | Widget width |
| `borderRadius` | string | `8px` | Border radius for inputs/buttons |

### React Component

```tsx
import { GGIDLoginWidget } from '@ggid/react-widget';

function LoginPage() {
  return (
    <GGIDLoginWidget
      tenantId="00000000-0000-0000-0000-000000000001"
      primaryColor="#7c3aed"
      logoUrl="/logo.svg"
      platformName="Acme Portal"
      socialProviders={['google', 'github']}
      onSuccess={(token) => {
        localStorage.setItem('token', token.access_token);
        router.push('/dashboard');
      }}
      onError={(err) => setLoginError(err.message)}
    />
  );
}
```

### Widget Theming via CSS

Override widget styles with CSS custom properties:

```css
:root {
  --ggid-primary: #7c3aed;
  --ggid-primary-hover: #6d28d9;
  --ggid-bg: #f8fafc;
  --ggid-card-bg: #ffffff;
  --ggid-text: #1e293b;
  --ggid-text-muted: #64748b;
  --ggid-border: #e2e8f0;
  --ggid-radius: 12px;
  --ggid-font: 'Poppins', sans-serif;
  --ggid-input-height: 48px;
}

#ggid-login {
  max-width: 400px;
  margin: 0 auto;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.1);
}
```

### Headless Mode

For full control over the UI, use the headless SDK (no rendered widget):

```typescript
import { GGIDAuth } from '@ggid/sdk';

const auth = new GGIDAuth({
  baseUrl: 'https://iam.yourcompany.com',
  tenantId: '00000000-0000-0000-0000-000000000001',
});

// Custom login form
const result = await auth.login({
  username: emailInput.value,
  password: passwordInput.value,
});

if (result.success) {
  // Build your own UI, navigate, etc.
  localStorage.setItem('jwt', result.access_token);
} else {
  showCustomError(result.error.message);
}
```

### Embedded Login Security

- The widget script is served with `integrity` hashes (SRI) for tamper detection

---

## Custom Domain Mapping

Map custom domains (e.g., `login.acme.com`) to your GGID tenant for a fully
white-labeled experience. Users see your brand in the URL bar, not GGID.

### Architecture

```
User browses to login.acme.com
        │
        ▼
  DNS CNAME → ggid.example.com
        │
        ▼
  GGID Gateway (checks Host header)
        │
        ├── Matches login.acme.com → Tenant: acme
        ├── Loads acme branding (logo, colors, fonts)
        └── Serves hosted login page with acme theme
```

### Configure Custom Domain

```bash
# Step 1: Register the custom domain in GGID
curl -X POST $API/api/v1/settings/custom-domain \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d '{
        "domain": "login.acme.com",
        "type": "login_page",
        "ssl_mode": "managed"
    }'

# Response includes DNS verification records
# {
#   "domain": "login.acme.com",
#   "verification_status": "pending",
#   "dns_records": [
#     { "type": "CNAME", "name": "login.acme.com", "value": "ggid.example.com" },
#     { "type": "TXT", "name": "_ggid.login.acme.com", "value": "ggid-verify=abc123..." }
#   ]
# }
```

### Step 2: Configure DNS

Add the DNS records returned by the API to your DNS provider:

```
# CNAME: point custom domain to GGID
login.acme.com.  CNAME  ggid.example.com.

# TXT: verify domain ownership
_ggid.login.acme.com.  TXT  "ggid-verify=abc123def456"
```

### Step 3: Verify Domain

```bash
# Check verification status
curl $API/api/v1/settings/custom-domain/login.acme.com \
    -H "Authorization: Bearer $ADMIN_TOKEN"

# Or trigger manual verification
curl -X POST $API/api/v1/settings/custom-domain/login.acme.com/verify \
    -H "Authorization: Bearer $ADMIN_TOKEN"

# Expected: { "status": "verified", "ssl_status": "active" }
```

### TLS/SSL for Custom Domains

GGID supports three SSL modes for custom domains:

| Mode | Description | Use Case |
|------|-------------|----------|
| `managed` | GGID provisions and renews certs automatically via Let's Encrypt | Recommended |
| `custom` | You provide a certificate (upload PEM) | Enterprise with private CA |
| `disabled` | No TLS (use external load balancer for TLS) | Behind nginx/Caddy |

#### Managed SSL (Automatic)

```bash
# GGID automatically provisions Let's Encrypt cert
# Just set ssl_mode=managed — no manual steps needed
# Certificate renews automatically 30 days before expiry
```

#### Custom SSL (Bring Your Own Certificate)

```bash
# Upload certificate
curl -X POST $API/api/v1/settings/custom-domain/login.acme.com/ssl \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -F "certificate=@/path/to/fullchain.pem" \
    -F "private_key=@/path/to/privkey.pem"

# Certificate must be valid for login.acme.com
# GGID will use this certificate for TLS termination
```

### Multiple Custom Domains

A single tenant can have multiple custom domains:

```bash
# Login page on one domain
POST /api/v1/settings/custom-domain
{ "domain": "login.acme.com", "type": "login_page" }

# Console on another domain
POST /api/v1/settings/custom-domain
{ "domain": "admin.acme.com", "type": "console" }

# OAuth callback on a third
POST /api/v1/settings/custom-domain
{ "domain": "auth.acme.com", "type": "oauth" }
```

### Branding Per Domain

Each custom domain can have different branding:

```bash
# Set branding for login.acme.com
curl -X PUT $API/api/v1/settings/branding \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -H "X-Domain: login.acme.com" \
    -d '{
        "logo_url": "https://cdn.acme.com/login-logo.svg",
        "primary_color": "#7c3aed",
        "platform_name": "Acme Portal"
    }'

# Set different branding for admin.acme.com
curl -X PUT $API/api/v1/settings/branding \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -H "X-Domain: admin.acme.com" \
    -d '{
        "logo_url": "https://cdn.acme.com/admin-logo.svg",
        "primary_color": "#1e40af",
        "platform_name": "Acme Admin"
    }'
```

### OAuth Redirect URIs with Custom Domains

When using custom domains, update OAuth client redirect URIs:

```bash
# Register OAuth client with custom domain callback
curl -X PUT $API/api/v1/oauth/clients/$CLIENT_ID \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d '{
        "redirect_uris": [
            "https://app.acme.com/callback",
            "https://login.acme.com/callback"
        ]
    }'
```

### Cookie Domain Configuration

For cross-subdomain authentication (e.g., `login.acme.com` and `app.acme.com`):

```bash
# Set cookie domain to parent domain
curl -X PUT $API/api/v1/settings/cookies \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d '{
        "domain": ".acme.com",
        "same_site": "lax"
    }'
```

This allows session cookies to be shared across all `*.acme.com` subdomains.

### Custom Domain Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| Domain stuck in `pending` | DNS not propagated | Wait 5-15 min, run `dig login.acme.com` |
| SSL provisioning failed | DNS not verified first | Verify domain ownership TXT record |
| 404 on custom domain | Domain not registered in GGID | Add via `POST /settings/custom-domain` |
| Cookie not shared | Cookie domain mismatch | Set cookie domain to `.acme.com` |
| Redirect loop | TLS mode conflict | Use `managed` SSL or configure LB correctly |
