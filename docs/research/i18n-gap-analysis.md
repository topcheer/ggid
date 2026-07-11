# Internationalization (i18n) Gap Analysis for GGID IAM

> **Date**: 2025-07-11
> **Author**: Competitive Analysis Team
> **Scope**: GGID i18n maturity vs Auth0, Keycloak, Casdoor, Ory
> **Conclusion**: GGID has a functional but minimal i18n foundation. Significant gaps exist in language coverage, translatability of Go error messages, email template localization, RTL support, and ICU/pluralization support. This document provides a prioritized roadmap to reach competitive parity.

---

## Table of Contents

1. [GGID Current i18n Status](#1-ggid-current-i18n-status)
2. [Competitor i18n Comparison](#2-competitor-i18n-comparison)
3. [What Needs Translation in IAM](#3-what-needs-translation-in-iam)
4. [Language Priority Matrix](#4-language-priority-matrix)
5. [Machine Translation Strategy](#5-machine-translation-strategy)
6. [RTL (Right-to-Left) Support](#6-rtl-right-to-left-support)
7. [Legal/Compliance Localization](#7-legalcompliance-localization)
8. [Technical Implementation Review](#8-technical-implementation-review)
9. [Gap Analysis & Recommendations](#9-gap-analysis--recommendations)

---

## 1. GGID Current i18n Status

### 1.1 Console Frontend (Next.js)

The admin console uses JSON message files in `console/messages/`:

| File | Locale | Keys | Coverage |
|------|--------|------|----------|
| `en.json` | English | 4 namespaces (nav, common, dashboard, settings), ~75 keys | Base reference |
| `zh.json` | Chinese (Simplified) | Mirrors en.json, ~75 keys | Full console coverage |

**Languages supported in console: 2** (en, zh)

Key observations:
- The console locale files cover only navigation labels, common UI buttons, dashboard widgets, and settings labels.
- Page-level content (form labels, error messages, help text, tooltips, empty states) is **not** translated — these strings are hardcoded in JSX.
- The console does **not** use `next-intl` or any i18n framework. A grep for `useTranslations`, `getTranslations`, `next-intl`, `NextIntlClientProvider`, `setRequestLocale`, and `getLocale` returned **zero matches**. The message files appear to be manually consumed or not yet wired into the rendering pipeline.
- This means the console is effectively **English-only** in practice, with `zh.json` serving as a prepared-but-not-yet-active translation file.

### 1.2 Go Backend i18n Package (`pkg/i18n/`)

The `pkg/i18n/` package provides a `Translator` struct for server-side hosted pages and email rendering.

**Locale files in `pkg/i18n/locales/`:**

| File | Locale | Keys | Use Case |
|------|--------|------|----------|
| `en.json` | English | 44 keys (login, register, forgot, reset, email subjects) | Hosted login pages |
| `zh-CN.json` | Chinese | 39 keys (same coverage minus email subjects) | Hosted login pages |
| `ja.json` | Japanese | 23 keys (login, register, common — missing forgot/reset/email) | Hosted login pages |

**Languages supported in backend i18n: 3** (en, zh-CN, ja)

Key observations:
- The backend i18n package is designed for hosted authentication pages (login, register, forgot password, reset password, MFA).
- **Email subjects** are only translated in `en.json` (`email.welcome_subject`, `email.reset_subject`, `email.verify_subject`, `email.mfa_subject`). They are missing from `zh-CN.json` and `ja.json`.
- **Japanese locale is incomplete**: it lacks `forgot.*`, `reset.*`, `register.error.*`, `common.error`, `common.required`, `common.optional`, and all `email.*` keys.

### 1.3 Total Supported Languages Across GGID

| Component | Languages |
|-----------|-----------|
| Console (`console/messages/`) | en, zh |
| Backend hosted pages (`pkg/i18n/locales/`) | en, zh-CN, ja |
| **Union** | **en, zh/zh-CN, ja** (3 languages) |

### 1.4 Go Error Messages — Translatability Audit

GGID's Go services contain **hundreds of hardcoded English error messages** that are never passed through the i18n translator.

- `services/auth/` alone contains **246 instances** of `fmt.Errorf`, `errors.New`, or `http.Error` with hardcoded English strings across 33 files.
- `pkg/` packages similarly use `fmt.Errorf` with English text throughout.
- These messages are returned directly in JSON API responses and are visible to end users via API error bodies and HTTP error responses.
- **No Go error message in any GGID service is translatable** — there is no error-code-to-translation-key mapping layer.

Examples of hardcoded, non-translatable errors:
```go
// services/auth/internal/service/auth_service.go (40 fmt.Errorf/errors.New calls)
fmt.Errorf("invalid credentials")
fmt.Errorf("user not found")
fmt.Errorf("rate limit exceeded")
fmt.Errorf("account locked")
```

These strings appear verbatim in API JSON responses like `{"error": "invalid credentials"}`.

### 1.5 Email Templates — English-Only

The email template system (`pkg/email/templates.go`) generates HTML and plain-text emails with **100% hardcoded English content**. Four templates exist:

| Template | Status |
|----------|--------|
| `PasswordResetHTML` / `PasswordResetText` | English only |
| `EmailVerificationHTML` | English only |
| `WelcomeHTML` | English only |
| `MFACodeHTML` | English only |

Despite `en.json` having `email.welcome_subject` etc., the **body content** of all emails is hardcoded. The translation keys for email subjects are never referenced in `pkg/email/templates.go`.

---

## 2. Competitor i18n Comparison

### 2.1 Auth0

| Aspect | Details |
|--------|---------|
| **Languages** | 10+ for Universal Login |
| **Supported locales** | en, es, pt, fr, de, it, ja, nl, zh |
| **Error messages** | Fully localized in all 10 languages |
| **Email templates** | Localized subject + body via Liquid templates |
| **Hosted pages** | Language auto-detected from `Accept-Language` header, user override cookie |
| **Customization** | Tenants can override any translation string via dashboard |
| **API errors** | Error codes (not strings) in API responses, with localized descriptions in documentation |
| **Maturity** | Production-grade, maintained by Auth0 team |

### 2.2 Keycloak

| Aspect | Details |
|--------|---------|
| **Languages** | 20+ via theme properties files |
| **Supported locales** | en, de, es, fr, it, ja, nl, pt, ru, zh, ar, ca, cs, da, fi, hu, ko, no, pl, sk, sv, tr, uk, and more |
| **Translation model** | Community-contributed via GitHub PRs to `messages_*.properties` files |
| **Error messages** | Fully localized, properties-file based |
| **Email templates** | Localized via FreeMarker templates with theme message bundles |
| **Admin console** | Fully localized (separate message bundles from login pages) |
| **RTL support** | Yes, via theme CSS (Arabic and Hebrew themes exist) |
| **Maturity** | 10+ years, translations contributed and reviewed by the community |

### 2.3 Casdoor

| Aspect | Details |
|--------|---------|
| **Languages** | 20+ including zh, en, fr, de, ja, ko, ru, ar, he |
| **Implementation** | i18next JSON files per locale |
| **Admin console** | Fully localized |
| **Error messages** | Localized via i18next with fallback |
| **RTL** | Supported (ar, he) |
| **Maturity** | Active development, community translations |

### 2.4 Ory

| Aspect | Details |
|--------|---------|
| **Languages** | 5 (en, de, fr, zh, pt) |
| **Quality** | Machine-translated baseline; human review ongoing |
| **Error messages** | Self-service UI messages localized via JSON files |
| **Email templates** | Customizable per locale via courier templates |
| **RTL** | Limited |
| **Maturity** | Newer i18n effort, still maturing |

### 2.5 Comparison Matrix

| Feature | Auth0 | Keycloak | Casdoor | Ory | **GGID** |
|---------|-------|----------|---------|-----|----------|
| Total languages | 10 | 20+ | 20+ | 5 | **3** |
| Error message localization | Full | Full | Full | Partial | **None** |
| Email template localization | Full | Full | Full | Partial | **None** |
| Admin console localization | Full | Full | Full | N/A (API-first) | **Prepared only** |
| RTL support | No | Yes | Yes | Limited | **No** |
| ICU MessageFormat | No | No | Yes (i18next) | No | **No** |
| Pluralization | Custom | Custom | Yes | No | **No** |
| Community translations | No | Yes | Yes | No | **No** |
| API error codes (not strings) | Yes | Mixed | Mixed | Yes | **No** |

### 2.6 GGID Competitive Position

GGID ranks **last** among comparable IAM solutions in i18n maturity. With only 3 languages (and 2 of those incomplete), no error message localization, no email localization, and no RTL support, GGID is significantly behind even Ory (the least mature competitor) in language coverage and translatability.

---

## 3. What Needs Translation in IAM

### 3.1 Translation Surface Area

An IAM system has an unusually large translation surface because text appears in many contexts:

| Surface | Description | Priority | Current GGID Status |
|---------|-------------|----------|---------------------|
| **Login/Signup UI** | Username/password fields, submit buttons, social login buttons, forgot password link | P0 | en + zh-CN + ja (partial) |
| **Error messages (UI)** | "Invalid credentials", "Account locked", "Rate limited", "Code expired" | P0 | en + zh-CN + ja (partial) |
| **Error messages (API)** | JSON `error` field in API responses | P0 | English only, hardcoded |
| **Email templates** | Welcome, password reset, email verification, MFA code, password changed, account locked | P1 | English only, hardcoded bodies |
| **Email subjects** | Subject lines for all notification emails | P1 | en only; missing from zh-CN/ja |
| **Admin console** | Navigation, forms, tables, dashboards, settings pages | P1 | en + zh (prepared, not wired) |
| **Consent screens** | OAuth/OIDC consent: scopes, permissions, grant text | P1 | Not implemented |
| **MFA prompts** | TOTP enrollment, backup codes, WebAuthn instructions, SMS code entry | P1 | en only |
| **Audit log descriptions** | Human-readable event descriptions ("User created", "Role assigned", "Policy updated") | P2 | English only |
| **API error codes** | Stable machine-readable error codes with localized descriptions | P2 | Not implemented |
| **Password policy messages** | "Password must be at least 8 characters", "Must contain uppercase" | P2 | English only |
| **Legal/compliance text** | Privacy policy, terms of service, cookie notice, DPA | P2 | Not implemented |
| **Webhook payloads** | Human-readable descriptions in webhook event bodies | P3 | English only |
| **Documentation** | Integration guides, API docs, admin manual | P3 | English only |

### 3.2 Priority Tiers

**Tier P0 (Immediate — User-Facing Auth Flows)**:
Everything a user sees during login, registration, password reset, and MFA. This is the minimum viable i18n surface. Users who cannot read English cannot complete authentication.

**Tier P1 (Short-term — Admin Experience + Notifications)**:
Admin console localization and email template localization. Required for non-English organizations to administer GGID effectively.

**Tier P2 (Medium-term — Compliance + Developer Experience)**:
API error codes, audit log descriptions, password policy messages, and legal text localization. Important for enterprise and EU compliance.

**Tier P3 (Long-term — Documentation + Nice-to-Have)**:
Full documentation translation and webhook description localization.

---

## 4. Language Priority Matrix

### 4.1 Ranking Criteria

| Criterion | Weight | Description |
|-----------|--------|-------------|
| Market size | 30% | Number of potential users/developers speaking the language |
| Regulatory requirement | 25% | EU requires all 24 official languages for consumer products; other jurisdictions have similar mandates |
| Developer demand | 20% | Languages frequently requested by open-source community and enterprise prospects |
| Competitive parity | 15% | Matching competitors' language coverage to avoid feature-gap objections |
| Community contribution potential | 10% | Likelihood of community translators maintaining the locale |

### 4.2 Tier 1 — Must-Have (Ship First)

| Language | Code | Speakers (M) | Key Markets | Rationale |
|----------|------|-------------|-------------|-----------|
| English | en | 1,500+ | Global | Default language, already complete |
| Chinese (Simplified) | zh-CN | 1,100+ | China, Singapore | Largest internet population; GGID already has zh-CN locale |
| Spanish | es | 550+ | Latin America, Spain | 2nd most spoken native language globally |
| Portuguese | pt | 260+ | Brazil, Portugal | Large developer community; Brazil growing IAM market |
| French | fr | 300+ | France, Canada, Africa | EU official language; large African francophone tech market |
| German | de | 130+ | Germany, Austria, Switzerland | Strong enterprise market; data sovereignty requirements |
| Japanese | ja | 125+ | Japan | GGID already has ja locale; high enterprise demand |

### 4.3 Tier 2 — Important (Ship Within 6 Months)

| Language | Code | Speakers (M) | Rationale |
|----------|------|-------------|-----------|
| Korean | ko | 80+ | Strong enterprise IAM market; competitive parity |
| Russian | ru | 260+ | Large developer community; CIS market |
| Arabic | ar | 420+ | RTL required; large market; competitive parity with Casdoor |
| Italian | it | 70+ | EU official language; enterprise market |
| Dutch | nl | 30+ | EU official language; competitive parity with Auth0 |
| Hindi | hi | 600+ | India's growing developer market |

### 4.4 Tier 3 — Community-Driven (As Available)

| Language | Code | Speakers (M) | Rationale |
|----------|------|-------------|-----------|
| Turkish | tr | 80+ | Keycloak supports it |
| Polish | pl | 50+ | EU; Keycloak/Casdoor support |
| Swedish | sv | 10+ | EU Nordic enterprise |
| Czech | cs | 13+ | EU; Keycloak supports |
| Vietnamese | vi | 95+ | Growing developer community |
| Thai | th | 70+ | Southeast Asia |
| Indonesian | id | 270+ | Large population, growing tech |
| Hebrew | he | 9+ | RTL required; enterprise |
| Ukrainian | uk | 40+ | EU-adjacent; Keycloak supports |
| Finnish | fi | 5+ | EU Nordic |
| Danish | da | 6+ | EU Nordic |
| Norwegian | no | 5+ | EU Nordic |
| Catalan | ca | 10+ | Regional EU |
| Hungarian | hu | 13+ | EU |
| Slovak | sk | 5+ | EU |

### 4.5 EU Compliance Consideration

The EU has **24 official languages**. For consumer-facing products sold in the EU, providing information (including terms of service and privacy notices) in the user's language is a legal requirement under various directives (e.g., Consumer Rights Directive, GDPR transparency obligations).

Of the 24 EU official languages, GGID currently covers **2** (en, zh — and zh is not an EU language, so effectively just **1**: en).

The 24 EU languages: bg, hr, cs, da, nl, en, et, fi, fr, de, el, hu, ga, it, lv, lt, mt, pl, pt, ro, sk, sl, es, sv.

---

## 5. Machine Translation Strategy

### 5.1 Initial Translation Pipeline

For reaching 20+ languages quickly, machine translation (MT) is the pragmatic starting point.

```
English source strings
        │
        ▼
   DeepL API (preferred — higher quality for security/legal terms)
        │
        ▼
   Machine-translated locale files (es.json, fr.json, de.json, ...)
        │
        ▼
   Automated validation:
   - JSON structure matches en.json
   - All format verbs (%s, %d) preserved
   - No empty values
        │
        ▼
   Human review (native speakers for Tier 1 languages)
        │
        ▼
   Committed to repository
```

### 5.2 DeepL vs Google Translate

| Aspect | DeepL | Google Translate |
|--------|-------|-----------------|
| Quality (Indo-European) | Superior nuance | Good |
| Quality (CJK) | Good (zh, ja) | Good |
| Security/legal terminology | Better context awareness | More literal translations |
| API cost | €4.99/month (500K chars) | $20/month (1M chars) |
| Data privacy | EU servers, GDPR compliant | US servers, data used for training |
| Recommendation | **Primary for IAM** (privacy + quality) | Fallback/secondary |

### 5.3 Quality Risks for Security Terminology

Machine translation poses specific risks for IAM systems where precise terminology matters:

| English Term | MT Risk | Correct Translation (ja) | Common MT Error (ja) |
|-------------|---------|-------------------------|---------------------|
| Authentication | Low | 認証 | Correct |
| Authorization | **High** — often conflated with authn | 認可 (authorization) | 認証 (authentication) |
| Credentials | **High** | 資格情報 | 証明書 (certificate) |
| Two-factor authentication | Medium | 二要素認証 | 二段階認証 (acceptable variant) |
| Rate limit | **High** | レート制限 | 速度制限 (less standard) |
| Session | Low | セッション | Correct |
| Consent | Medium | 同意 | 許可 (permission — different legal meaning) |
| Data processing | Medium | データ処理 | Correct |
| Scope (OAuth) | **High** | スコープ | 範囲 (range — wrong meaning) |
| Token | Low | トークン | Correct |
| WebAuthn | Low | WebAuthn | Webオーセンティケーション (should NOT translate) |

**Mitigation**: Maintain a terminology glossary that is pre-translated by humans and injected as DeepL glossary entries.

### 5.4 Human Review Process

1. **Crowdin or Weblate** as the translation management platform
2. **Glossary** of IAM-specific terms pre-translated and locked
3. **Tier 1 languages** (7): Human-reviewed by paid native speakers
4. **Tier 2 languages** (6): Community-reviewed by 2+ contributors
5. **Tier 3 languages**: Machine-translated with a visible "beta" badge

### 5.5 Community Translation Model

Following Keycloak's proven model:

1. **Translation files** are JSON in the repository (`pkg/i18n/locales/`, `console/messages/`)
2. **CONTRIBUTING.md** includes a section on "How to add a new locale"
3. **Translation completeness check** in CI: a script verifies that all locale files have the same set of keys as `en.json`
4. **Community maintainers** are assigned per language (like Keycloak's language maintainers)
5. **Translation freeze** before releases: no new keys added after RC, giving translators time to catch up

### 5.6 Go Code: MT Fallback Pattern

```go
// TranslateWithMTFallback returns the best available translation.
// If no human translation exists, it returns the English string.
// A production system could call an MT API for on-the-fly translation
// of missing keys, but this is NOT recommended for security-critical text.
func (t *Translator) TranslateWithMTFallback(locale, key string, params ...interface{}) string {
    msg := t.Translate(locale, key, params...)
    if msg == key {
        // Key not found in any locale — return English as safe fallback
        msg = t.Translate(t.defaultLocale, key, params...)
    }
    return msg
}
```

**Recommendation**: Do NOT use on-the-fly MT for IAM error messages. Pre-translate everything and ship static locale files. On-the-fly MT introduces latency, API dependency, and quality unpredictability for security-critical text.

---

## 6. RTL (Right-to-Left) Support

### 6.1 Languages Requiring RTL

| Language | Code | Speakers | Script |
|----------|------|----------|--------|
| Arabic | ar | 420M | Arabic (RTL) |
| Hebrew | he | 9M | Hebrew (RTL) |
| Persian/Farsi | fa | 110M | Arabic (RTL) |
| Urdu | ur | 230M | Arabic (RTL) |
| Pashto | ps | 50M | Arabic (RTL) |

### 6.2 CSS Adjustments for RTL

Modern CSS supports logical properties that automatically adapt to text direction:

```css
/* Instead of physical properties */
.login-form {
  padding-left: 20px;    /* LTR only */
  margin-right: 16px;    /* LTR only */
}

/* Use logical properties (direction-aware) */
.login-form {
  padding-inline-start: 20px;   /* RTL aware */
  margin-inline-end: 16px;      /* RTL aware */
}
```

Additional RTL adjustments:
- `direction: rtl` on `<html>` or container
- `text-align: start` instead of `left`/`right`
- Flip icons that imply direction (arrows, chevrons)
- Flexbox `flex-direction` handles RTL automatically with `dir` attribute
- Use CSS `:dir(rtl)` pseudo-class for direction-specific overrides

### 6.3 Console UI Changes

1. **Set `dir` attribute** based on locale: `<html dir="rtl" lang="ar">`
2. **Tailwind CSS** (if used) supports RTL via `rtl:` and `ltr:` variant modifiers
3. **Navigation sidebar** should flip to the right side in RTL
4. **Tables** should mirror column order
5. **Forms** should right-align labels in RTL
6. **Icons**: directional icons (back arrow, next chevron) must be mirrored

### 6.4 Login Page RTL

The hosted login page must support RTL. Current implementation (`pkg/i18n/locales/`) has locale files but the HTML rendering needs direction detection.

```go
// DirectionForLocale returns the text direction for a locale.
func DirectionForLocale(locale string) string {
    rtlLocales := map[string]bool{
        "ar": true, "he": true, "fa": true, "ur": true, "ps": true,
    }
    // Normalize: "ar-SA" → "ar"
    lang := strings.ToLower(strings.SplitN(locale, "-", 2)[0])
    if rtlLocales[lang] {
        return "rtl"
    }
    return "ltr"
}
```

The hosted login page HTML template should:
```html
<html lang="{{.Locale}}" dir="{{.Direction}}">
```

### 6.5 Current GGID RTL Status

**No RTL support exists in GGID.** The `pkg/i18n/translator.go` has no direction detection. The console CSS uses physical properties in some places. Adding RTL is a medium-effort task but should be planned before adding Arabic/Hebrew locales.

---

## 7. Legal/Compliance Localization

### 7.1 GDPR Requirements

Under GDPR Article 12, information provided to data subjects must be:
- "Concise, transparent, intelligible and in an **easily accessible form**, using **clear and plain language**"
- For a child, "in language that the child can easily understand"

This means consent screens, privacy notices, and data subject rights information should be provided in the user's language, especially for consumer-facing (B2C) IAM deployments.

### 7.2 Documents Requiring Localization

| Document | Trigger | Languages Required |
|----------|---------|-------------------|
| **Privacy Policy** | User registration, data collection | All languages offered to users |
| **Terms of Service** | Account creation | All languages offered to users |
| **Cookie Notice** | EU users (ePrivacy Directive) | At least EU official languages |
| **Consent Text (OAuth)** | OAuth/OIDC consent flow | User's preferred language |
| **Data Processing Agreement (DPA)** | Enterprise contracts | Negotiated per contract (usually en) |
| **Data Subject Rights Notice** | GDPR Articles 13-15 | User's language |

### 7.3 Consent Screen Localization

OAuth/OIDC consent screens must clearly state what data is being shared and with whom. This text varies by scope:

```json
{
  "consent.scope.openid": "Access your basic profile information",
  "consent.scope.email": "Read your email address",
  "consent.scope.profile": "Access your profile (name, avatar)",
  "consent.scope.offline_access": "Access your data even when you're offline",
  "consent.grant_button": "Authorize",
  "consent.deny_button": "Deny",
  "consent.app_requesting": "{app} is requesting the following permissions:",
  "consent.review": "Review permissions"
}
```

### 7.4 Jurisdiction-Specific Requirements

| Jurisdiction | Requirement | Impact |
|-------------|-------------|--------|
| **EU (GDPR)** | All user-facing text in user's language | Must localize consent, privacy policy, DSR notices |
| **Brazil (LGPD)** | Portuguese for Brazilian users | pt-BR locale needed |
| **California (CCPA)** | "Notice at Collection" in English or Spanish | en, es locales |
| **Quebec (Bill 96)** | French mandatory for consumer-facing services | fr-CA locale |
| **China (PIPL)** | Chinese mandatory for user-facing services | zh-CN locale |
| **Russia** | Russian required for consumer services targeting RU market | ru locale |

### 7.5 GGID Current Status

**No legal/compliance localization exists.** There are no privacy policy templates, no consent screen text, no localized terms of service. These would need to be added as part of the i18n effort and should be **human-translated and legally reviewed** — never machine-translated for legal documents.

---

## 8. Technical Implementation Review

### 8.1 Current Architecture

GGID's i18n system consists of:

1. **`pkg/i18n/translator.go`** — Go translator for server-side rendering
2. **`pkg/i18n/locales/*.json`** — Flat JSON locale files for hosted auth pages
3. **`console/messages/*.json`** — Next.js console locale files (not wired to next-intl)

### 8.2 Translator API Review

```go
type Translator struct {
    mu            sync.RWMutex
    translations  map[string]map[string]string  // locale → key → string
    defaultLocale string
}
```

**Strengths:**
- Thread-safe (`sync.RWMutex`)
- Clean fallback chain: requested locale → default locale → key itself
- `Accept-Language` header parsing via `ResolveLocale()`
- `fmt.Sprintf`-style parameter substitution
- `LoadDirectory()` auto-discovers locale files

**Weaknesses:**

| Issue | Impact | Severity |
|-------|--------|----------|
| **No ICU MessageFormat** | Cannot express gender, complex plurals, nested messages | High |
| **No pluralization** | "1 item" vs "5 items" impossible; must use workarounds | High |
| **No number formatting** | `1,000` (en) vs `1.000` (de) vs `1 000` (fr) not handled | Medium |
| **No date formatting** | `MM/DD/YYYY` (en-US) vs `DD/MM/YYYY` (en-GB) not handled | Medium |
| **`fmt.Sprintf` only** | Positional arguments only; reordering for different languages is impossible (e.g., "Welcome %s" → "%sさん、ようこそ" reorders the argument) | High |
| **No message context** | Cannot disambiguate "Sign" (verb) vs "Sign" (noun) in same locale | Medium |
| **Case sensitivity** | `Translate("EN", ...)` doesn't match `"en"` — should normalize | Low |
| **`ResolveLocale` ignores q-values** | Returns first language regardless of quality preference; doesn't match against supported locales | Medium |
| **Not wired to any service** | `NewTranslator` only called in tests; no production service uses it | Critical |
| **Flat key namespace** | No hierarchical structure; all keys in one flat map | Low |

### 8.3 ICU MessageFormat Gap

ICU MessageFormat is the industry standard for internationalized message formatting. It handles:

**Pluralization:**
```
{count, plural,
  =0 {No messages}
  =1 {One message}
  other {# messages}
}
```

**Gender:**
```
{gender, select,
  male {He}
  female {She}
  other {They}
}
```

**Nested messages:**
```
{count, plural,
  =1 {{name} has one notification}
  other {{name} has # notifications}
}
```

**Positional reordering:**
```
// English
"Welcome, {name}!"
// Japanese (name first)
"{name}さん、ようこそ！"
```

GGID's current `fmt.Sprintf` approach cannot handle any of these patterns. The Go library `github.com/nicksnyder/go-i18n/v2` or `github.com/gotnospirit/messageformat` could provide ICU support.

### 8.4 Pluralization Gap

Current GGID approach would require pre-generating strings for each plural case:

```go
// Current (no pluralization support):
tr.Translate(locale, "messages.count", count)  // "You have %d messages"
// Always uses "messages" — grammatically incorrect for count=1 in English
```

With ICU:
```json
{
  "messages.count": "{count, plural, =1 {You have one message} other {You have # messages}}"
}
```

### 8.5 Number/Date Formatting Gap

GGID has no locale-aware number or date formatting. All dates and numbers are formatted with Go defaults. For the console, this means:

- User counts: `1,234` always uses comma separator (wrong for de, fr, es, pt)
- Dates: always Go default format, ignoring locale conventions

For full parity, GGID should use `golang.org/x/text` package for:
- `message.Printer` for locale-aware formatting
- `number.NewFormat` for number formatting
- `time.Format` with locale-aware layouts

### 8.6 API Error Code Architecture

Currently, GGID API responses contain raw English error strings:

```json
{"error": "invalid credentials"}
```

The recommended architecture separates machine-readable codes from human-readable messages:

```json
{
  "error": {
    "code": "AUTH_INVALID_CREDENTIALS",
    "message": "Invalid username or password",
    "message_key": "login.error.invalid"
  }
}
```

The `code` field is stable for programmatic handling. The `message_key` allows clients to look up localized text. The `message` field provides a sensible English fallback.

---

## 9. Gap Analysis & Recommendations

### 9.1 Gap Summary

| Gap | Current State | Target State | Priority |
|-----|---------------|-------------|----------|
| Console i18n not wired | en/zh JSON exist but next-intl not configured | next-intl with automatic locale detection | **P0** |
| Go error messages not translatable | 246+ hardcoded strings in auth service alone | Error code → translation key mapping | **P0** |
| Email templates English-only | 4 templates, all hardcoded English | Locale-aware templates with subject + body | **P1** |
| Only 3 locales | en, zh-CN, ja | 7 Tier 1 + 6 Tier 2 = 13 locales | **P1** |
| Incomplete locale files | ja missing 21 keys, zh-CN missing email keys | All locales 100% key coverage | **P1** |
| No ICU MessageFormat | `fmt.Sprintf` only | go-i18n with ICU support | **P1** |
| No pluralization | None | ICU plural rules | **P1** |
| No RTL support | None | Full RTL for ar, he | **P2** |
| No consent screen i18n | Consent not implemented | Localized consent with scope descriptions | **P2** |
| No legal text localization | None | Privacy policy, ToS per locale | **P2** |
| No number/date formatting | Go defaults | `golang.org/x/text` formatting | **P2** |
| No community translation process | None | Weblate/Crowdin integration | **P3** |

### 9.2 Prioritized Action Items

#### Phase 1: Foundation (2-3 weeks, ~60 hours)

| # | Task | Effort | Owner |
|---|------|--------|-------|
| 1.1 | Wire `next-intl` into console with automatic locale detection from `Accept-Language` | 8h | Frontend |
| 1.2 | Extract all hardcoded UI strings from console pages into `console/messages/` | 12h | Frontend |
| 1.3 | Wire `pkg/i18n.Translator` into auth service HTTP handlers for hosted pages | 6h | Backend |
| 1.4 | Define IAM error code taxonomy (e.g., `AUTH_INVALID_CREDENTIALS`, `AUTH_ACCOUNT_LOCKED`) | 8h | Backend |
| 1.5 | Add error code to translation key mapping in API JSON responses | 8h | Backend |
| 1.6 | Complete `ja.json` and `zh-CN.json` with all missing keys | 4h | i18n |
| 1.7 | Add CI check for locale key completeness (all locales must match en.json keys) | 4h | DevOps |
| 1.8 | Upgrade `ResolveLocale` to match against supported locales and respect q-values | 4h | Backend |
| 1.9 | Add locale case normalization (lowercase, normalize separators) | 2h | Backend |

#### Phase 2: Language Expansion (3-4 weeks, ~80 hours)

| # | Task | Effort | Owner |
|---|------|--------|-------|
| 2.1 | Create IAM terminology glossary (50-100 terms, human-translated to Tier 1 languages) | 12h | i18n |
| 2.2 | Machine-translate Tier 1 locales (es, pt, fr, de) via DeepL with glossary | 8h | i18n |
| 2.3 | Human review of Tier 1 MT output by native speakers | 16h | Contractors |
| 2.4 | Migrate from `fmt.Sprintf` to `go-i18n/v2` with ICU MessageFormat | 16h | Backend |
| 2.5 | Localize email templates (subject + body) with locale parameter | 12h | Backend |
| 2.6 | Add locale-aware number formatting via `golang.org/x/text` | 8h | Backend |
| 2.7 | Add pluralization support for count-based messages | 8h | Backend |

#### Phase 3: Completeness (4-6 weeks, ~100 hours)

| # | Task | Effort | Owner |
|---|------|--------|-------|
| 3.1 | Machine-translate Tier 2 locales (ko, ru, it, nl, hi) | 8h | i18n |
| 3.2 | Implement consent screen with localized scope descriptions | 12h | Backend + Frontend |
| 3.3 | Add RTL support to console (CSS logical properties, dir attribute) | 16h | Frontend |
| 3.4 | Add RTL support to hosted login pages | 8h | Backend |
| 3.5 | Add Arabic (ar) locale with full RTL testing | 8h | i18n + QA |
| 3.6 | Localize password policy messages | 8h | Backend |
| 3.7 | Localize audit log event descriptions | 12h | Backend |
| 3.8 | Set up Weblate or Crowdin for community translation management | 16h | DevOps |
| 3.9 | Write "How to add a new locale" guide in CONTRIBUTING.md | 4h | Docs |
| 3.10 | Legal text templates (privacy policy, ToS) in en + zh + fr + de | 8h | Legal + i18n |

### 9.3 Effort Summary

| Phase | Duration | Effort | Languages Added |
|-------|----------|--------|-----------------|
| Phase 1: Foundation | 2-3 weeks | ~60h | 0 (fix existing 3) |
| Phase 2: Expansion | 3-4 weeks | ~80h | +4 (es, pt, fr, de) |
| Phase 3: Completeness | 4-6 weeks | ~100h | +5 (ko, ru, it, nl, ar/hi) |
| **Total** | **9-13 weeks** | **~240h** | **12 total** |

### 9.4 Success Metrics

| Metric | Current | Target (6 months) | Target (12 months) |
|--------|---------|-------------------|--------------------|
| Supported locales | 3 | 7 | 12 |
| Console key coverage | ~75 keys | ~300 keys | ~500 keys |
| Backend error messages translatable | 0% | 80% | 100% |
| Email templates localized | 0% | 100% | 100% |
| RTL languages | 0 | 0 | 2 (ar, he) |
| ICU MessageFormat | No | Yes | Yes |
| Locale completeness (CI check) | Manual | Automated | Automated |
| Community translation platform | None | Weblate setup | 3+ active community translators |

### 9.5 Key Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Security terminology mistranslation | Users misunderstand auth prompts → security incidents | Human review of all Tier 1 translations; locked glossary |
| Incomplete locale files shipped | Users see English fallback mid-page → poor UX | CI check enforcing key completeness before merge |
| ICU migration breaks existing format strings | Runtime panics in production | Gradual migration; maintain backward compat during transition |
| Community translations introduce errors | Poor quality translations damage trust | Require 2+ approvals for community-sourced translations |
| Legal text MT liability | Incorrect legal text → regulatory penalties | Legal text is human-translated and legally reviewed only |

### 9.6 Recommendation Summary

GGID's i18n infrastructure has a **solid but incomplete foundation**. The `Translator` struct in `pkg/i18n/` is well-designed with thread safety and fallback chains, but it lacks ICU MessageFormat, pluralization, and number/date formatting. More critically, it is **not wired into any production code path** — neither the auth service nor the console currently use it for rendering.

The highest-priority work is:
1. **Wire next-intl into the console** (Phase 1.1-1.2) — makes the existing zh translation functional
2. **Wire the Translator into auth service HTTP handlers** (Phase 1.3) — makes hosted pages locale-aware
3. **Introduce error codes** (Phase 1.4-1.5) — enables API consumers to localize client-side
4. **Migrate to go-i18n/v2 with ICU** (Phase 2.4) — future-proofs the translation system
5. **Expand to 7 Tier 1 languages** (Phase 2.2-2.3) — reaches competitive parity with Auth0

With ~240 hours of focused effort across 9-13 weeks, GGID can reach **12 languages** with ICU support, localized emails, and RTL — matching Casdoor and exceeding Ory in i18n maturity.

---

*End of document.*
