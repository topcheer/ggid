# Branding & Customization Guide

> Per-tenant branding: logo, colors, custom domain, email templates, login page.

---

## Per-Tenant Branding

Each tenant can have completely independent branding:

```bash
curl -X PUT http://localhost:8080/api/v1/tenants/$TENANT_ID/branding \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "logo_url": "https://acme.com/logo.png",
    "favicon_url": "https://acme.com/favicon.ico",
    "primary_color": "#FF5733",
    "secondary_color": "#33C4FF",
    "login_bg_color": "#1a1a2e",
    "login_text_color": "#ffffff",
    "button_color": "#FF5733"
  }'
```

---

## Custom Domain

Point a custom domain to GGID for per-tenant hosted login:

```bash
# Configure custom domain
curl -X POST http://localhost:8080/api/v1/tenants/$TENANT_ID/domains \
  -d '{"domain": "login.acme.com", "tls": true}'

# DNS: CNAME login.acme.com → ggid.example.com
```

GGID serves the hosted login page at `https://login.acme.com/login` with Acme's branding.

---

## Email Templates

Customize email templates per tenant:

```bash
curl -X PUT http://localhost:8080/api/v1/tenants/$TENANT_ID/email-templates \
  -d '{
    "welcome": {
      "subject": "Welcome to Acme!",
      "body_html": "<h1>Welcome {{username}}</h1><p>Click <a href=\"{{verify_url}}\">here</a> to verify.</p>"
    },
    "password_reset": {
      "subject": "Reset your Acme password",
      "body_html": "<p>Reset link: {{reset_url}}</p>"
    }
  }'
```

Template variables: `{{username}}`, `{{email}}`, `{{verify_url}}`, `{{reset_url}}`, `{{tenant_name}}`.

---

## Login Page Customization

### Hosted Login

GGID's hosted login page at `/login` auto-applies tenant branding:
- Logo, colors, background from branding config
- Tenant resolved from domain or `X-Tenant-ID` header

### Embedded Login

Use GGID SDK to build your own login page:

```jsx
import { GGIDProvider, LoginForm } from '@ggid/react';

<GGIDProvider domain="ggid.example.com" tenantId="acme">
  <LoginForm
    logoUrl="https://acme.com/logo.png"
    primaryColor="#FF5733"
  />
</GGIDProvider>
```

---

## CSS Overrides

For Admin Console customization:

```bash
# Upload custom CSS
curl -X POST http://localhost:8080/api/v1/tenants/$TENANT_ID/custom-css \
  -d '{"css": ".btn-primary { border-radius: 8px; }"}'
```

---

*See: [Tenant Onboarding](tenant-onboarding.md) | [Multi-Tenant Guide](multi-tenant-guide.md)*

*Last updated: 2025-07-11*
