# Frontend i18n Guide

> Complete guide to internationalization in the GGID Admin Console using next-intl.

---

## Overview

The GGID Console uses [next-intl](https://next-intl-docs.vercel.app/) for client-side internationalization. Translation files live in `console/src/messages/`.

```
console/src/messages/
  ├── en.json     # English (default)
  └── zh.json     # Chinese
```

---

## Message Key Convention

### Naming Pattern

```
{page}.{section}.{element}
```

Examples:
```json
{
  "common.save": "Save",
  "common.cancel": "Cancel",
  "common.delete": "Delete",
  "users.page.title": "Users",
  "users.page.subtitle": "Manage user accounts",
  "users.table.username": "Username",
  "users.table.email": "Email",
  "users.form.create.title": "Create User",
  "users.form.create.username_label": "Username",
  "settings.sso.title": "Single Sign-On",
  "settings.sso.saml.enabled": "SAML Enabled",
  "audit.page.title": "Audit Log",
  "error.network": "Network error. Please try again.",
  "error.unauthorized": "You are not authorized to perform this action."
}
```

### Categories

| Prefix | Usage |
|--------|-------|
| `common.*` | Shared buttons, labels, statuses (Save, Cancel, Loading) |
| `{page}.*` | Page-specific strings (users, roles, audit, settings) |
| `error.*` | Error messages |
| `nav.*` | Navigation labels |

---

## Configuration

### next-intl Setup

```typescript
// console/src/i18n/request.ts
import { getRequestConfig } from 'next-intl/server';

export default getRequestConfig(async ({ locale }) => ({
  messages: (await import(`../messages/${locale}.json`)).default,
}));
```

### Next.js Config

```typescript
// console/next.config.ts
import createNextIntlPlugin from 'next-intl/plugin';

const withNextIntl = createNextIntlPlugin('./src/i18n/request.ts');

export default withNextIntl({
  // ... other config
});
```

### Middleware

```typescript
// console/src/middleware.ts
import createMiddleware from 'next-intl/middleware';

export default createMiddleware({
  locales: ['en', 'zh'],
  defaultLocale: 'en',
});

export const config = {
  matcher: ['/((?!api|_next|_vercel|.*\\..*).*)'],
};
```

---

## useTranslations Hook

```tsx
'use client';
import { useTranslations } from 'next-intl';

export function UserTable() {
  const t = useTranslations('users');

  return (
    <div>
      <h1>{t('page.title')}</h1>
      <p>{t('page.subtitle')}</p>
      <table>
        <thead>
          <tr>
            <th>{t('table.username')}</th>
            <th>{t('table.email')}</th>
          </tr>
        </thead>
      </table>
    </div>
  );
}
```

### Using Common Keys

```tsx
import { useTranslations } from 'next-intl';

function DeleteButton() {
  const t = useTranslations('common');
  return <button>{t('delete')}</button>;
}
```

### With Parameters

```json
{
  "users.found": "Found {count} users",
  "users.deleted": "User {name} deleted"
}
```

```tsx
const t = useTranslations('users');
return <p>{t('found', { count: 42 })}</p>;
// → "Found 42 users"
```

---

## LanguageSwitcher Component

```tsx
'use client';
import { useLocale } from 'next-intl';
import { useRouter, usePathname } from 'next/navigation';
import { useState } from 'react';

const languages = [
  { code: 'en', label: 'English', flag: '\ud83c\uddfa\ud83c\uddf8' },
  { code: 'zh', label: '中文', flag: '\ud83c\udde8\ud83c\uddf3' },
];

export function LanguageSwitcher() {
  const locale = useLocale();
  const router = useRouter();
  const pathname = usePathname();

  const switchLocale = (newLocale: string) => {
    // next-intl middleware handles locale prefix
    const newPath = pathname.replace(`/${locale}`, `/${newLocale}`);
    router.push(newPath);
  };

  return (
    <select
      value={locale}
      onChange={(e) => switchLocale(e.target.value)}
      className="border rounded px-2 py-1"
    >
      {languages.map((lang) => (
        <option key={lang.code} value={lang.code}>
          {lang.flag} {lang.label}
        </option>
      ))}
    </select>
  );
}
```

### Place in Layout

```tsx
// console/src/app/[locale]/layout.tsx
import { LanguageSwitcher } from '@/components/LanguageSwitcher';

export default function LocaleLayout({ children }) {
  return (
    <html>
      <body>
        <nav>
          {/* ... nav items ... */}
          <LanguageSwitcher />
        </nav>
        {children}
      </body>
    </html>
  );
}
```

---

## Adding a New Language

1. **Create translation file**:
```bash
cp console/src/messages/en.json console/src/messages/fr.json
```

2. **Translate values** in `fr.json`:
```json
{
  "common.save": "Enregistrer",
  "common.cancel": "Annuler",
  "users.page.title": "Utilisateurs"
}
```

3. **Add to middleware locales**:
```typescript
locales: ['en', 'zh', 'fr'], // add 'fr'
```

4. **Add to LanguageSwitcher**:
```typescript
{ code: 'fr', label: 'Français', flag: '\ud83c\uddeb\ud83c\uddf7' },
```

---

## Extraction Strategy (1051 Hardcoded Strings)

The console currently has ~1051 hardcoded English strings. Extraction priority:

| Priority | Pages | Est. Strings |
|----------|-------|-------------|
| P0 | settings/sso (21), settings/oauth-clients (19), settings/api-keys (18) | 58 |
| P1 | settings/tenant-config (17), settings/certificates (15), organizations (15) | 47 |
| P2 | settings/branding (12), security-center (11), settings/mfa (9) | 32 |
| P3 | Remaining pages | ~914 |

### Extraction Process

```bash
# Find hardcoded strings
grep -rn '"[A-Z][a-z]*' console/src/app/**/*.tsx \
  | grep -v node_modules \
  | grep -v '.test.' \
  | wc -l

# Replace:
// Before: <button onClick={save}>Save User</button>
// After:  <button onClick={save}>{t('save_user')}</button>
```

---

*Last updated: 2025-07-11*