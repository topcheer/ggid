# Email Template Design

Transactional vs marketing, template inheritance, variable system, responsive HTML/CSS, dark mode, accessibility, localization, testing, and deliverability.

## Template Types

| Type | Trigger | Example |
|------|---------|---------|
| Transactional | User action | Welcome, password reset, MFA code |
| Operational | System event | Security alert, deprovisioning |
| Marketing | Campaign | Product update (separate unsubscribe) |

## Template Inheritance

```
base.html (layout: header, footer, styles)
  ├── welcome.html (content block)
  ├── password-reset.html (content block)
  ├── mfa-code.html (content block)
  └── security-alert.html (content block)
```

```html
<!-- base.html -->
<html>
<head><style>{{styles}}</style></head>
<body>
  <header>{{logo}}</header>
  <main>{{content}}</main>
  <footer>{{footer}}</footer>
</body>
</html>

<!-- welcome.html -->
{{extends "base.html"}}
{{block "content"}}
  <h1>Welcome to GGID, {{user_name}}!</h1>
  <p>Click to activate: <a href="{{action_url}}">Activate Account</a></p>
  <p>Link expires in {{expiry_hours}} hours.</p>
{{end}}
```

## Variable System

| Variable | Source | Example |
|----------|--------|---------|
| `{{user_name}}` | User profile | Jane Doe |
| `{{action_url}}` | Generated per-email | `https://auth.ggid.dev/activate?token=...` |
| `{{expiry}}` | TTL config | 24 hours |
| `{{tenant_name}}` | Tenant config | Acme Corp |
| `{{support_email}}` | Tenant config | support@acme.com |
| `{{logo_url}}` | Tenant branding | `https://acme.com/logo.png` |
| `{{current_year}}` | System | 2025 |

## Responsive HTML/CSS

```html
<style>
  body { font-family: -apple-system, sans-serif; margin: 0; padding: 0; }
  .container { max-width: 600px; margin: 0 auto; padding: 20px; }
  .button { background: #0052CC; color: white; padding: 12px 24px; border-radius: 4px; }
  @media (max-width: 480px) {
    .container { padding: 10px; }
    .button { width: 100%; text-align: center; }
  }
  @media (prefers-color-scheme: dark) {
    body { background: #1a1a1a; color: #fff; }
    .button { background: #2684FF; }
  }
</style>
```

## Localization

```yaml
email_templates:
  welcome:
    en: "Welcome to {{tenant_name}}, {{user_name}}!"
    zh: "欢迎加入 {{tenant_name}}，{{user_name}}！"
    ja: "{{tenant_name}}へようこそ、{{user_name}}さん！"
    ar: "مرحبا {{user_name}} في {{tenant_name}}!"  # RTL
```

### RTL Support

```html
<html lang="ar" dir="rtl">
  <!-- Content flows right-to-left automatically -->
</html>
```

### Date/Time Formatting

```go
func formatDate(t time.Time, locale string) string {
    switch locale {
    case "en": return t.Format("January 2, 2006 at 3:04 PM MST")
    case "zh": return t.Format("2006年1月2日 15:04")
    case "ja": return t.Format("2006年1月2日 15:04")
    }
    return t.Format(time.RFC3339)
}
```

## Accessibility

- Semantic HTML (`<h1>`, `<p>`, `<a>`)
- Alt text on all images: `<img alt="GGID Logo" src="...">`
- Minimum 14px font size
- Color contrast ratio ≥ 4.5:1 (WCAG AA)
- Link text descriptive (not "click here")
- `role="presentation"` on decorative images

## Deliverability

| Record | Purpose |
|--------|---------|
| SPF | Authorize sending IPs |
| DKIM | Sign emails with domain key |
| DMARC | Policy for SPF/DKIM failures |

```dns
ggid.dev.    TXT  "v=spf1 include:_spf.google.com ~all"
default._domainkey.ggid.dev.  TXT  "v=DKIM1; k=rsa; p=..."
_dmarc.ggid.dev.  TXT  "v=DMARC1; p=quarantine; rua=mailto:dmarc@ggid.dev"
```

## Testing

- **Litmus/EmailOnAcid**: Test rendering across 40+ email clients
- **CI check**: Validate HTML, check links, verify variables resolved
- **Preview mode**: Send test email with sample data before production

## Monitoring

| Metric | Alert |
|--------|-------|
| Delivery rate | <95% → check reputation |
| Bounce rate | >2% → clean list |
| Open rate (transactional) | >80% expected |
| Template render errors | Any → fix template |

## See Also

- [Identity Lifecycle Automation](identity-lifecycle-automation.md)
- [Digital Identity Lifecycle](digital-identity-lifecycle.md)
- Notification Templates