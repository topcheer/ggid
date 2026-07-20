# Cross-Board ERP Demo Progress Tracker

> **Goal**: Multi-language ERP demo testing all GGID core features + all SDKs
> **Started**: 2026-07-21
> **Last Updated**: 2026-07-21

## Status Overview

| # | Language | SDK | Dir | Status | Owner |
|---|----------|-----|-----|--------|-------|
| 1 | Go | Go SDK | examples/erp-go/ | 🔲 Not started | researcher |
| 2 | TypeScript/Node | Node SDK | examples/erp-node/ | 🔲 Not started | frontend |
| 3 | React | React SDK | examples/erp-react/ | 🔲 Not started | frontend |
| 4 | Python | Python SDK | examples/erp-python/ | 🔲 Not started | backend |
| 5 | Java | Java SDK | examples/erp-java/ | 🔲 Not started | backend |
| 6 | C# | C# SDK | examples/erp-csharp/ | 🔲 Not started | backend |
| 7 | Ruby | Ruby SDK | examples/erp-ruby/ | 🔲 Not started | TBD |
| 8 | Rust | Rust SDK | examples/erp-rust/ | 🔲 Not started | TBD |

## ERP Modules (7 modules, all CRUD)

| Module | Endpoints | Permission Required |
|--------|-----------|---------------------|
| Auth | login, logout, refresh, verify | none (public) |
| Users | CRUD + role assign | users:read, users:write, users:delete |
| Roles | CRUD + permission tree | roles:read, roles:write |
| Organizations | CRUD + hierarchy | orgs:read, orgs:write |
| Inventory | CRUD products/stock | inventory:read, inventory:write, inventory:delete |
| Orders | CRUD + approval | orders:read, orders:write, orders:approve |
| Audit | View audit log | audit:read |

## Permission Matrix

| Role | Permissions |
|------|------------|
| Viewer | dashboard:read, inventory:read, orders:read |
| Sales | + orders:write |
| Manager | + orders:approve, orders:read:all |
| Admin | admin (bypass) |

## Phase Progress

### Phase 1: Go + Node + React
- [ ] Go ERP backend
- [ ] Node ERP backend
- [ ] React ERP frontend

### Phase 2: Python + Java + C#
- [ ] Python ERP backend
- [ ] Java ERP backend
- [ ] C# ERP backend

### Phase 3: Ruby + Rust
- [ ] Ruby ERP backend
- [ ] Rust ERP backend

## GGID Features Tested

| Feature | How Tested |
|--------|-----------|
| OAuth 2.1 PKCE | Auth module login flow |
| Token verify/refresh | Every API call |
| JWT permissions claim | RequirePermission on all routes |
| RBAC (roles) | Role assignment in Users module |
| Fine-grained permissions | Permission matrix per role |
| User CRUD | Users module |
| Organization hierarchy | Orgs module |
| Audit log | Audit module |
| Multi-tenant | X-Tenant-ID header |
| SDK completeness | Each language uses its SDK |