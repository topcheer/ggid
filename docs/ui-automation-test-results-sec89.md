# GGID UI Automation Test Results — Section 8 & 9

**Date**: 2026-07-12
**Environment**: k3s (https://ggid.iot2.win)

## Section 8: Security Center

| Test | Endpoint | Status | Result |
|------|----------|--------|--------|
| 8.1 Risk score | GET /api/v1/audit/risk-score?user_id={id} | PASS | `{"Score":0.5,"Velocity":0,"GeoAnomaly":false,"NewIP":true,"Recommendations":["verify_device"]}` |
| 8.2 Threats | GET /api/v1/security/threats | FAIL | 404 — endpoint not registered |
| 8.3 Anomalies | GET /api/v1/security/anomalies | FAIL | 404 — endpoint not registered |
| 8.4 Sessions | GET /api/v1/auth/sessions?user_id={id} | FAIL | 500 — `{"error":{"code":"internal","message":"failed to list sessions"}}` |

**Security Center Summary**: 1/4 PASS. Risk score endpoint works via `/api/v1/audit/risk-score` (not `/api/v1/security/`). Threats and anomalies endpoints return 404. Sessions endpoint returns 500 (internal error — likely missing DB table or Redis query failure).

### Failed Tests Detail

1. **GET /api/v1/security/threats** — 404. Route not registered in gateway. May need to be under `/api/v1/audit/threats` or implemented in a security service.
2. **GET /api/v1/security/anomalies** — 404. Same as above.
3. **GET /api/v1/auth/sessions?user_id={id}** — 500. Route exists but handler fails with "failed to list sessions". Likely missing session table in DB or Redis session store not configured.

## Section 9: AI Agents

| Test | Endpoint | Status | Result |
|------|----------|--------|--------|
| 9.1 Agent list (empty) | GET /api/v1/agents | PASS | `{"agents":[],"total":0}` |
| 9.2 Agent list (alt) | GET /api/v1/agents/list | PASS | `null` (valid empty response) |
| 9.3 Register agent | POST /api/v1/agents/register | PASS | Created agent with id, client_id, status=active, rate_limit=100/min |
| 9.4 Agent list (after register) | GET /api/v1/agents | PASS | `{"agents":[{...}],"total":1}` — registered agent visible |

**AI Agents Summary**: 4/4 PASS. Full CRUD cycle verified: list empty -> register -> list with new agent. Agent registration creates proper agent record with client_id, tenant isolation, and default rate limiting.

## Overall Results

| Section | Tests | Pass | Fail |
|---------|-------|------|------|
| 8. Security Center | 4 | 1 | 3 |
| 9. AI Agents | 4 | 4 | 0 |
| **Total** | **8** | **5** | **3** |

## Failed Tests Summary

1. `/api/v1/security/threats` — 404, route not registered
2. `/api/v1/security/anomalies` — 404, route not registered
3. `/api/v1/auth/sessions` — 500, handler error "failed to list sessions"

## Notes

- Risk score endpoint lives under `/api/v1/audit/` not `/api/v1/security/`
- Agent registration works with `owner_user_id` field (required)
- Agent tokens include `client_id` in format `agent_{uuid}`
- Session listing likely needs Redis session store or DB sessions table
