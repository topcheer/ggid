# GGID Branding & Customization Guide

This guide covers configuring per-tenant branding — logos, colors, CSS, email templates, and custom domains.

## Overview

GGID supports per-tenant branding through the Identity service's branding API. Each tenant can customize the visual identity of the GGID Admin Console and authentication flows.

## Branding Configuration

### API

```bash
# Get current branding
curl https://api.ggid.example.com/api/v1/branding \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# Update branding
curl -X PUT https://api.ggid.example.com/api/v1/branding \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "logo_url": "https://cdn.example.com/logo.svg",
    "primary_color": "#0066CC",
    "secondary_color": "#003D7A",
    "accent_color": "#FF6B35",
    "company_name": "Acme Corp",
    "support_email": "support@acme.com",
    "custom_css": ".btn-primary { border-radius: 8px; }",
    "login_page_title": "Sign in to Acme",
    "login_page_subtitle": "Access your Acme workspace"
  }'
```

### Configuration Fields

| Field | Type | Description |
|-------|------|-------------|
| `logo_url` | URL | Logo image (SVG/PNG, max 1MB) |
| `primary_color` | Hex | Primary brand color (buttons, links) |
| `secondary_color` | Hex | Secondary color (hover states) |
| `accent_color` | Hex | Accent color (highlights) |
| `company_name` | String | Display name in console header |
| `support_email` | Email | Shown in error/help states |
| `custom_css` | String | Custom CSS overrides (sanitized) |
| `login_page_title` | String | Login page heading |
| `login_page_subtitle` | String | Login page subtext |
| `favicon_url` | URL | Custom favicon |

## Color System

GGID uses CSS custom properties for theming:

```css
:root {
  --ggid-primary: #0066CC;    /* Set by branding API */
  --ggid-secondary: #003D7A;
  --ggid-accent: #FF6B35;
  --ggid-bg: #F9FAFB;
  --ggid-surface: #FFFFFF;
  --ggid-text: #1F2937;
  --ggid-text-muted: #6B7280;
  --ggid-border: #E5E7EB;
  --ggid-success: #10B981;
  --ggid-error: #EF4444;
  --ggid-warning: #F59E0B;
}
```

## Custom CSS

Custom CSS is sanitized server-side. Allowed:
- Color properties
- Border radius
- Padding/margin
- Font-family
- Box-shadow

Blocked: `position: fixed`, `display: none` on critical elements, `javascript:` URLs, `@import`.

## Email Templates

GGID sends transactional emails with per-tenant branding:

| Email Type | Trigger | Variables |
|-----------|---------|----------|
| Welcome | User registration | `{{name}}`, `{{login_url}}` |
| Password reset | Reset requested | `{{name}}`, `{{reset_url}}` |
| MFA enrollment | MFA device added | `{{name}}`, `{{device_type}}` |
| Account locked | Lockout triggered | `{{name}}`, `{{unlock_url}}` |
| Access granted | Role assigned | `{{name}}`, `{{role}}` |

### Template Structure

```html
<!-- Wrapped in branding layout -->
<div style="font-family: {{font_family}}; color: {{text_color}};">
  <img src="{{logo_url}}" alt="{{company_name}}" style="max-height: 40px;"/>
  <h1 style="color: {{primary_color}};">{{title}}</h1>
  <p>Hi {{name}},</p>
  <p>{{body}}</p>
  <a href="{{action_url}}" 
     style="background: {{primary_color}}; color: white; padding: 12px 24px; border-radius: 4px; text-decoration: none;">
    {{action_text}}
  </a>
</div>
```

## Custom Domains

### DNS Configuration

```
# CNAME to GGID gateway
auth.example.com  CNAME  api.ggid.example.com
```

### TLS Certificate

GGID automatically provisions TLS certificates for custom domains via Let's Encrypt / cert-manager:

```bash
# Register custom domain
curl -X POST https://api.ggid.example.com/api/v1/domains \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"domain": "auth.example.com"}'

# Check provisioning status
curl https://api.ggid.example.com/api/v1/domains/auth.example.com \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

## Branding via Console

Navigate to **Settings → Branding** in the Admin Console to configure branding interactively with a live preview.

## See Also

- [Console Admin Guide](console-admin-guide.md)
- [Multi-Tenant Guide](multi-tenant-guide.md)
- [Tenant Onboarding](tenant-onboarding.md)
