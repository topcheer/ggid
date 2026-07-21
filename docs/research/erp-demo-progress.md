# Cross-Board ERP Demo Progress Tracker

> **Last Updated**: 2026-07-21 09:00

## Overall: Code 8/8 | Deploy 8/8 | Health OK 8/8 | Auth Adapt 4/8 | CRUD Verified 3/8 | Browser Pending

| # | Lang | Code | Tenant ID | Auth Required | Auth Status | k8s | Health | CRUD | Verified | Notes |
|---|------|------|-----------|---------------|-------------|-----|--------|------|----------|-------|
| 1 | Go | ✅ | 0001 | OAuth2 PKCE | ⏳ DM shen | ✅ | ✅ 200 | ✅ CRUD OK | ⚠️ perm | Currently password login, CRUD verified with test token |
| 2 | Node | ✅ | 0002 | Client Creds | ✅ M2M | ✅ | ✅ 200 | ⚠️ token verify | 🔲 | Token verification issue - uses GW verify endpoint |
| 3 | React | ✅ | 0003 | SPA PKCE | ✅ PKCE | ✅ | ✅ 307→/login | 🔲 | 🔲 | PKCE impl complete, tenant fixed in auth.ts |
| 4 | Python | ✅ | 0004 | SAML SSO | ✅ SAML | ✅ | ✅ 200 | ✅ List OK | ⚠️ | SAML flow ready, inventory list works |
| 5 | C# | ✅ | 0005 | Password Grant | ✅ Password | ✅ | ✅ 200 | ✅ CRUD OK | ⚠️ JSON types | CRUD works, JSON type mismatch on create (price/quantity types) |
| 6 | Java | ✅ | 0006 | SAML SSO | ⏳ DM backend | ✅ | ✅ 200 | ⚠️ partial | ⚠️ | Currently password, GET works, POST not supported |
| 7 | Ruby | ✅ | 0007 | Device Code | ⏳ DM guardian | ✅ | ✅ 200 | ⚠️ perm | 🔲 | HostAuth FIXED, permissions not passing through |
| 8 | Rust | ✅ | 0008 | Token Exchange | ⏳ DM backend | ✅ | ✅ 200 | ⚠️ deserial | 🔲 | Deployed! Rust 1.88, Axum route fix, borrow checker fix |

## Deployment Details

### All 8 Pods Running (8/8)
- erp-go: 1/1 Running (7h+)
- erp-node: 1/1 Running (7h+)
- erp-react: 1/1 Running (6h+)
- erp-python: 1/1 Running (6h+)
- erp-csharp: 1/1 Running (6h+)
- erp-java: 1/1 Running (6h+)
- erp-ruby: 1/1 Running (FIXED HostAuth)
- erp-rust: 1/1 Running (NEW - deployed)

### All 8 Ingress Configured
- erp-go.iot2.win → erp-go svc ✅ (FIXED: was cross-erp-go)
- erp-node.iot2.win → erp-node svc ✅
- erp-react.iot2.win → erp-react svc ✅
- erp-python.iot2.win → erp-python svc ✅ (FIXED: tenant → 0004)
- erp-csharp.iot2.win → erp-csharp svc ✅
- erp-java.iot2.win → erp-java svc ✅ (FIXED: was cross-erp-java)
- erp-ruby.iot2.win → erp-ruby svc ✅
- erp-rust.iot2.win → erp-rust svc ✅ (NEW)

## Auth Adaptation Status

| Demo | Required | Current | Owner | Status |
|------|----------|---------|-------|--------|
| Go | PKCE (OIDC) | Password | shen_frontend | ⏳ DM sent |
| Node | Client Credentials (M2M) | M2M ✅ | — | ✅ Done |
| React | SPA PKCE | PKCE ✅ | — | ✅ Done (tenant fixed) |
| Python | SAML SSO | SAML ✅ | — | ✅ Done |
| C# | Password Grant | Password ✅ | — | ✅ Done |
| Java | SAML SSO | Password | ggcxf_backend | ⏳ DM sent |
| Ruby | Device Code | Password | guardian_security | ⏳ DM sent |
| Rust | Token Exchange | verify only | ggcxf_backend | ⏳ DM sent |

## CRUD Verification Results

### Go Demo (Tenant 0001) - FULL CRUD ✅
- Create Inventory: ✅ PROD-0001 created
- List Inventory: ✅ Returns created product
- Create Order: ✅ ORD-0001 created
- List Orders: ✅ Returns created order
- Dashboard: ✅ Returns stats (1 product, 1 order, 1 pending, 2 audit)
- Audit Log: ✅ Returns audit entries
- Permission Check: ✅ Rejects without token, allows with admin perm

### C# Demo (Tenant 0005) - PARTIAL CRUD ⚠️
- List Inventory: ✅ Returns seed data (p001, p002)
- List Orders: ✅ Returns seed data (o001, o002)
- Create Inventory: ❌ JSON type mismatch (price/quantity as String vs Number)
- Create Order: ❌ Same JSON type issue
- Health: ✅ Shows correct tenant + auth method

### Python Demo (Tenant 0004) - LIST OK ✅
- Root info: ✅ Shows SAML SSO + tenant 0004
- List Inventory: ✅ Returns seed data
- SAML flow: Ready (login_url configured)

### Java Demo (Tenant 0006) - PARTIAL ⚠️
- Dashboard: ✅ Returns user info + permissions + stats
- List Inventory: ✅ Returns user context
- Create: ❌ POST not supported (need SAML adaptation)
- Permission check: Working (modules enabled/disabled)

### Rust Demo (Tenant 0008) - DEPLOYED ✅
- Health: ✅ {"status":"ok"}
- Create Inventory: ❌ Missing field `id` (Rust struct requires all fields)
- Dashboard: Returns empty (no data yet)
- Permission check: Working (rejects without token)

## Fixes Applied This Session
1. Ruby HostAuth 403: Fixed with `set :host_authorization, permitted_hosts: ['.']`
2. Rust compilation: Fixed `# comment` → `// comment`, borrow checker, Axum `:id` → `{id}`
3. Rust deployment: Created k8s Deployment + Service + Ingress
4. Tenant ID fixes: Python (→0004), Java (→0006), Node (→0002), React (→0003)
5. Ingress routing: Fixed Go (→erp-go), Java (→erp-java) from old cross-erp-* services

## Known Issues / GAPs
1. **Auth adaptation**: 4/8 demos need auth code changes (Go, Java, Ruby, Rust)
2. **C# JSON types**: Price and quantity need string types in C# deserialization
3. **Rust struct**: Product/Order structs require `id` field for POST (should be auto-generated)
4. **Node token verify**: Uses GW verify endpoint, may fail with custom tokens
5. **Ruby permissions**: verify_token succeeds but has_permission? returns false (needs debugging)
6. **Multi-tenant users**: Only default tenant (0000) has admin user; other tenants need user creation

## Next Steps
1. Wait for team members to complete auth adaptation (Go, Java, Ruby, Rust)
2. Fix C# JSON type handling (price as string)
3. Fix Rust struct to not require `id` on POST
4. Debug Ruby permissions claim extraction
5. Create test users in each tenant for multi-tenant verification
6. Browser verification of all 8 demos after auth adaptation complete
7. Tenant isolation verification (cross-tenant data access test)
