# Penetration Testing Guide

This guide covers test scope, test categories, OWASP Testing Guide alignment, tools, methodology, GGID-specific test cases, and remediation SLA.

## Test Scope

### In-Scope Components

| Component | Scope | Priority |
|---|---|---|
| API Gateway | All routes, auth, rate limiting | P0 |
| Auth Service | Login, MFA, token issuance, password reset | P0 |
| OAuth Service | Authorization flows, token exchange, introspection | P0 |
| Identity Service | User CRUD, SCIM, profile management | P1 |
| Policy Service | RBAC/ABAC evaluation, role assignment | P0 |
| Audit Service | Audit query, export, hash chain | P1 |
| Admin Console | All pages, forms, API calls | P1 |

### Out-of-Scope

- Infrastructure DoS (load testing is separate)
- Social engineering of employees
- Physical security
- Third-party libraries (covered by SCA)

## Test Categories

### Auth Bypass

| Test | Description | OWASP Reference |
|---|---|---|
| JWT signature bypass | Modify JWT without key | OTG-SESS-002 |
| Token expiration bypass | Use expired token | OTG-AUTHN-007 |
| MFA bypass | Skip MFA verification | OTG-AUTHN-006 |
| Password reset abuse | Reset for other users | OTG-AUTHN-009 |
| Anonymous access | Access protected endpoints without auth | OTG-AUTHZ-001 |

### IDOR (Insecure Direct Object Reference)

| Test | Description |
|---|---|
| Cross-user access | User A accesses User B's resources |
| Cross-tenant access | Tenant A accesses Tenant B's data |
| UUID enumeration | Guess predictable resource IDs |
| API parameter manipulation | Change user_id, tenant_id in requests |

### SSRF

| Test | Description |
|---|---|
| Internal URL via webhook | Submit webhook URL pointing to internal services |
| Metadata endpoint | http://169.254.169.254 (cloud metadata) |
| DNS rebinding | URL resolves to internal IP after validation |
| Port scanning via SSRF | Use SSRF to map internal network |

### Injection

| Test | Description |
|---|---|
| SQL injection | Input fields, query parameters, headers |
| NoSQL injection | JSON body parameters |
| Command injection | Template fields, filename inputs |
| LDAP injection | Search filters, user lookup |
| XML injection | SAML responses, XML payloads |
| Template injection | Email templates, notification templates |

### XSS (Cross-Site Scripting)

| Test | Description |
|---|---|
| Reflected XSS | Input reflected in response |
| Stored XSS | Malicious input stored and rendered |
| DOM XSS | Client-side JS sinks |
| Content-type confusion | JSON endpoint rendering HTML |

### CSRF

| Test | Description |
|---|---|
| Stateless CSRF | No CSRF token on state-changing requests |
| Same-site cookie bypass | Cookie scope too broad |
| JSON CSRF | POST with text/plain content-type |

### Session Fixation

| Test | Description |
|---|---|
| Session ID in URL | Session token passed via URL parameter |
| Session doesn't rotate on login | Same session ID before and after auth |
| Session doesn't rotate on MFA | Same session after step-up |

### Token Replay

| Test | Description |
|---|---|
| Authorization code replay | Use same code twice |
| Refresh token reuse | Use rotated refresh token again |
| DPoP proof replay | Replay DPoP proof with different request |

## OWASP Testing Guide v4.2 Alignment

### Mapped Test Categories

| OWASP WSTG | Category | GGID Tests |
|---|---|---|
| WSTG-INFO-01 | Information Gathering | DNS, search engine, metadata |
| WSTG-CONFIG-02 | Application Platform Config | Default creds, admin panels |
| WSTG-IDM-01 | User Enumeration | Registration, login, reset |
| WSTG-AUTHN-01 | Credentials Transport | HTTPS enforcement |
| WSTG-AUTHN-03 | Default Credentials | No default accounts |
| WSTG-AUTHN-04 | Lockout Mechanism | Rate limiting, account lockout |
| WSTG-AUTHN-07 | Weak Lock Out | Login throttle |
| WSTG-AUTHZ-01 | Path Traversal | File access |
| WSTG-AUTHZ-02 | Bypass Auth Schema | Direct page access |
| WSTG-AUTHZ-04 | IDOR | Resource access |
| WSTG-SESS-01 | Session Management | Session timeout, rotation |
| WSTG-SESS-02 | Cookies Attributes | httpOnly, secure, sameSite |
| WSTG-SESS-07 | Session Timeout | Idle timeout enforcement |
| WSTG-INPV-01 | Reflected XSS | Input reflection |
| WSTG-INPV-02 | Stored XSS | Persistent XSS |
| WSTG-INPV-05 | SQL Injection | Database injection |
| WSTG-INPV-11 | SQL Injection (REST) | API parameters |
| WSTG-CRYP-01 | Weak SSL/TLS | Protocol versions, ciphers |
| WSTG-CRYP-04 | Weak Crypto | Algorithm choices |

## Tools

### Recommended Toolset

| Tool | Purpose | Type |
|---|---|---|
| Burp Suite | Manual testing, interception | Commercial/Free |
| OWASP ZAP | Automated scanning, proxy | Free |
| nuclei | Template-based vulnerability scanner | Free |
| gobuster | Directory/file enumeration | Free |
| sqlmap | SQL injection detection | Free |
| ffuf | Fuzzing, parameter discovery | Free |
| JWT Tool | JWT analysis and manipulation | Free |
| SAML Raider | SAML testing (Burp extension) | Free |

### nuclei Templates

```bash
# Run nuclei with GGID-specific templates
nuclei -u https://auth.ggid.example.com -t cves/ -t exposures/ -t misconfiguration/

# Custom GGID templates
nuclei -u https://staging.ggid.example.com -t ggcid-templates/
```

### Custom Templates

```yaml
# ggcid-templates/jwt-none-alg.yaml
id: jwt-none-algorithm
info:
  name: JWT None Algorithm Test
  severity: high
http:
  - method: GET
    path:
      - "{{BaseURL}}/api/v1/users"
    headers:
      Authorization: "Bearer eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiJhZG1pbiJ9."
    matchers:
      - type: status
        status:
          - 200
```

## Test Methodology

### 1. Reconnaissance

```bash
# Subdomain enumeration
amass enum -d ggid.example.com
subfinder -d ggid.example.com

# Port scanning
nmap -sV -p 1-65535 auth.ggid.example.com

# Technology fingerprinting
whatweb https://auth.ggid.example.com
wappalyzer analyze https://auth.ggid.example.com
```

### 2. Enumeration

```bash
# Directory enumeration
gobuster dir -u https://auth.ggid.example.com -w /usr/share/wordlists/dirb/common.txt

# API endpoint discovery
ffuf -u https://api.ggid.example.com/api/v1/FUZZ -w api-endpoints.txt

# User enumeration via login
for user in admin root test user; do
  curl -X POST /api/v1/auth/login -d "{\"username\":\"$user\",\"password\":\"invalid\"}"
done
# Check if error messages differ for valid vs invalid users
```

### 3. Exploitation

```bash
# SQL injection
sqlmap -u "https://api.ggid.example.com/api/v1/users?id=1" --cookie="session=..."

# JWT manipulation
python3 jwt_tool.py eyJ... -T  # Tamper
python3 jwt_tool.py eyJ... -X k  # Key confusion
python3 jwt_tool.py eyJ... -X n  # None algorithm

# SSRF testing
curl -X POST /api/v1/webhooks -d '{"url":"http://169.254.169.254/latest/meta-data/"}'

# IDOR testing
TOKEN_USER_A=$(login user_a)
curl -H "Authorization: Bearer $TOKEN_USER_A" /api/v1/users/$USER_B_ID
```

### 4. Report

### Report Structure

```
1. Executive Summary
2. Methodology
3. Findings (by severity)
   - Title
   - Severity (Critical/High/Medium/Low)
   - Description
   - Affected component
   - Steps to reproduce
   - Proof of concept
   - Impact
   - Remediation
   - References
4. Appendix (full test log)
```

### 5. Retest

After remediation, retest all findings:
- Verify fix works
- Check for regressions
- Test edge cases
- Update report with retest status

## GGID-Specific Test Cases

### Auth Service

| # | Test | Expected Result |
|---|---|---|
| 1 | Login with expired token | 401 Unauthorized |
| 2 | Login without MFA when required | 403 MFA required |
| 3 | Password reset for other user | 403 Forbidden |
| 4 | Brute force login (100 attempts) | 429 Rate limited |
| 5 | Login with breached password | 400 Breached password |
| 6 | Token with wrong issuer | 401 Invalid issuer |
| 7 | Token with wrong audience | 401 Invalid audience |
| 8 | Refresh token reuse | 401 + family revoked |

### OAuth Service

| # | Test | Expected Result |
|---|---|---|
| 1 | Auth code without PKCE | 400 PKCE required |
| 2 | Auth code with "plain" method | 400 S256 required |
| 3 | Redirect URI with wildcard | 400 URI mismatch |
| 4 | Redirect URI with HTTP (not HTTPS) | 400 HTTPS required |
| 5 | Auth code replay | 400 Code already used |
| 6 | Implicit grant attempt | 400 Grant deprecated |
| 7 | Client credentials with user scope | 403 Scope not allowed |
| 8 | Token introspection without auth | 401 Unauthorized |

### Identity Service

| # | Test | Expected Result |
|---|---|---|
| 1 | Cross-tenant user access | 404 Not Found |
| 2 | SCIM request without auth | 401 Unauthorized |
| 3 | User enumeration via registration | Generic error (no user info) |
| 4 | PII in error messages | No PII in errors |
| 5 | Bulk user export without permission | 403 Forbidden |

### Policy Service

| # | Test | Expected Result |
|---|---|---|
| 1 | Access restricted resource without role | 403 Forbidden |
| 2 | Policy bypass via parameter manipulation | 403 Forbidden |
| 3 | Role escalation via API | 403 Forbidden |
| 4 | ABAC attribute spoofing | 403 Forbidden |
| 5 | Policy dry-run with malicious input | Input validated |

### Gateway

| # | Test | Expected Result |
|---|---|---|
| 1 | X-Tenant-ID spoofing | JWT claim takes priority |
| 2 | Rate limit bypass via header manipulation | Rate limit enforced |
| 3 | Unregistered route access | 404 Not Found |
| 4 | Large payload (>1MB) | 413 Payload too large |
| 5 | SQL injection in query params | Input sanitized |

## Remediation SLA

| Severity | SLA | Process |
|---|---|---|
| Critical | 24 hours | Immediate patch, hotfix deploy |
| High | 7 days | Fix in next release |
| Medium | 30 days | Fix in sprint |
| Low | 90 days | Fix in backlog |

### Remediation Tracking

```yaml
pentest:
  tracking:
    tool: "jira"
    project: "SEC"
    sla:
      critical: 24h
      high: 7d
      medium: 30d
      low: 90d
    notify_on_overdue: true
    require_retest: true
    retest_by: "security-team"
```

## Best Practices

1. **Test staging, not production** — Use staging environment for active testing
2. **Get written authorization** — Obtain scope approval before testing
3. **Test after major changes** — Pentest after architecture changes
4. **Use both automated and manual** — Automated for coverage, manual for depth
5. **Test real attack paths** — Don't just test individual vulnerabilities
6. **Document everything** — Full reproduction steps for each finding
7. **Prioritize by risk** — Fix critical findings first
8. **Retest after fixes** — Verify remediation actually works
9. **Share findings with dev team** — Use as learning opportunities
10. **Schedule regularly** — Quarterly pentest, annual red team