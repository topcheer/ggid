# Internationalization (i18n) Setup Guide

> How to add new languages to GGID, extract hardcoded strings, and wire translations into services.

---

## Table of Contents

1. [Overview](#overview)
2. [Current State](#current-state)
3. [i18n Architecture](#i18n-architecture)
4. [Adding a New Language](#adding-a-new-language)
5. [Extracting Strings](#extracting-strings)
6. [Translation Workflow](#translation-translation-workflow)
7. [Wiring into Services](#wiring-into-services)

---

## Overview

GGID uses `pkg/i18n/` for server-side translations. The translator wraps a message bundle that loads JSON translation files.

---

## Current State

- **Supported languages**: English (`en`), Chinese (`zh-CN`)
- **Translation coverage**: ~60% of user-facing strings
- **Hardcoded strings**: ~937 strings still embedded in Go code (need extraction)
- **Console (frontend)**: Uses `next-intl` with separate translation files in `console/src/messages/`

---

## i18n Architecture

```
┌────────────────────────────────────────────┐
│                 Service Layer                │
│  ┌──────────────────────────────────────┐  │
│  │  pkg/i18n/translator.go               │  │
│  │  • Translator struct                  │  │
│  │  • Translate(lang, key, params)       │  │
│  │  • Loads JSON bundles                │  │
│  └──────────┬───────────────────────────┘  │
│             │                               │
│  ┌──────────▼───────────────────────────┐  │
│  │  translations/                        │  │
│  │  ├── en.json (English)               │  │
│  │  └── zh-CN.json (Chinese)            │  │
│  └──────────────────────────────────────┘  │
└────────────────────────────────────────────┘
```

---

## Adding a New Language

### 1. Create Translation File

```bash
cp translations/en.json translations/ja-JP.json
```

### 2. Translate Values

```json
// translations/ja-JP.json
{
  "auth.login_failed": "ログインに失敗しました",
  "auth.account_locked": "アカウントがロックされました。しばらくしてから再試行してください",
  "auth.mfa_required": "多要素認証が必要です",
  "user.not_found": "ユーザーが見つかりません",
  "error.unauthorized": "認証が必要です",
  "error.forbidden": "アクセス権限がありません"
}
```

### 3. Register the Language

```go
// In service initialization
translator := i18n.NewTranslator(
    i18n.WithBundle("translations/en.json"),      // default
    i18n.WithBundle("translations/zh-CN.json"),    // Chinese
    i18n.WithBundle("translations/ja-JP.json"),    // Japanese (new)
    i18n.WithDefaultLanguage("en"),
)
```

### 4. Use in Handlers

```go
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
    lang := r.Header.Get("Accept-Language") // "ja-JP"
    msg := h.translator.Translate(lang, "auth.login_failed")
    json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
```

---

## Extracting Strings

### Current State: ~937 Hardcoded Strings

Many error messages and user-facing strings are still hardcoded:

```go
// Before (hardcoded)
return fmt.Errorf("user not found")

// After (i18n)
return fmt.Errorf(t.Translate(lang, "user.not_found"))
```

### Extraction Process

1. **Find hardcoded strings**:
```bash
# Find fmt.Errorf with string literals
grep -rn 'fmt\.Errorf(' --include='*.go' | grep -v _test | grep -v vendor | wc -l

# Find json error strings
grep -rn '"error":' --include='*.go' | grep -v _test | grep -v vendor | wc -l
```

2. **Add to en.json**:
```json
{
  "user.not_found": "User not found",
  "user.already_exists": "User already exists",
  "auth.invalid_credentials": "Invalid username or password"
}
```

3. **Replace in code**:
```go
// Replace hardcoded strings with translator calls
return errors.New(t.Translate(lang, "user.not_found"))
```

---

## Translation Workflow

```
Developer writes code
  ↓
Add string key to translations/en.json
  ↓
Use translator.Translate(lang, key) in code
  ↓
Translator creates translations/<lang>.json
  ↓
Translator fills in translated values
  ↓
Review → Merge → Deploy
```

### Translation File Format

```json
{
  "auth.login_failed": "Invalid username or password",
  "auth.account_locked": "Account locked. Try again later.",
  "user.created": "User created successfully",
  "user.not_found": "User not found",
  "role.insufficient_permissions": "Insufficient permissions for this action",
  "tenant.isolation_error": "Cross-tenant access denied"
}
```

### Naming Convention

``n<domain>.<message>
``

Examples:
- `auth.login_failed`
- `user.not_found`
- `role.created`
- `audit.query_too_broad`

---

## Wiring into Services

### Gateway

```go
// services/gateway/cmd/main.go
translator := i18n.NewTranslator(...)
gateway := NewGateway(
    gateway.WithI18n(translator),
)
```

### Auth Service

```go
// services/auth/cmd/main.go
translator := i18n.NewTranslator(...)
authService := service.New(
    service.WithTranslator(translator),
)
```

### Accept-Language Header

Clients specify their preferred language:

```bash
curl -H "Accept-Language: zh-CN" \
     -H "Authorization: Bearer $JWT" \
     http://localhost:8080/api/v1/users
```

The service resolves the language from:
1. `Accept-Language` header (highest priority)
2. User profile `locale` field
3. Tenant default language
4. Fallback: `en`

---

## Console (Frontend)

The admin console uses `next-intl` with separate translation files:

```
console/src/messages/
  ├── en.json
  └── zh-CN.json
```

```tsx
// Using translations in React
import { useTranslations } from 'next-intl';

function LoginPage() {
    const t = useTranslations('auth');
    return <p>{t('login_failed')}</p>;
}
```

---

*Last updated: 2025-07-11*