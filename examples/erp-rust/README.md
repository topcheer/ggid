# GGID Cross-Board ERP Demo — Rust

ERP demo using GGID Rust SDK via axum.

## Setup

```bash
cd examples/erp-rust
cargo run
# Server starts on :9092
```

## Modules
1. Inventory — CRUD (inventory:read/write/delete)
2. Orders — CRUD + approval (orders:read/write/approve + row-level filter)
3. Audit — View audit log
4. Dashboard — Summary metrics

## Permission Matrix
| Role | Permissions |
|------|------------|
| Viewer | dashboard:read, inventory:read, orders:read |
| Sales | + orders:write |
| Manager | + orders:approve, orders:read:all |
| Admin | admin (bypass) |
