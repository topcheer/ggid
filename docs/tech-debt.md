# GGID Technical Debt & Improvements

记录开发过程中发现的技术债。每位 teammate 发现问题就记录在这里。

## Active

- Router coverage 48.8% — inflated by ~300 lines HTML template strings. Real Go logic coverage ~70%.
- OAuth service coverage 47.7% — needs more handler-level tests.
- Org service coverage 65.8% — needs PUT/DELETE/member handler tests.
- Social package coverage 30.6% — needs mock HTTP server tests for connectors.

## Resolved

- [x] Policy service coverage: 54.6% → 94.7%
- [x] Gateway tenant forwarding: header-only → query param + body injection
- [x] Register duplicate email: 500 → 409 Conflict
- [x] Audit NULL columns: pgx v5 scan fix
- [x] Login IPv6: net.SplitHostPort fix for [::1]
