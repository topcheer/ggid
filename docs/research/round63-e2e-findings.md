# Round 63 E2E Findings — Role Assign + Provisioning

## Date: 2026-07-16

## P0: User Role Assign — HTTP Route Missing

**Symptom:** POST /api/v1/users/{id}/roles returns "method not allowed"

**Root cause:** AssignRole is implemented as gRPC handler only. No HTTP route registered.
Frontend calls REST API, gets 405.

**Fix needed:** Register HTTP handler in services/identity/internal/server/http.go:
```
POST /api/v1/users/{id}/roles → assign role to user
DELETE /api/v1/users/{id}/roles/{role_id} → remove role
GET /api/v1/users/{id}/roles → list user's roles
```

## P1: Provisioning Service Unhealthy

**Symptom:** healthz/deep reports provisioning as unhealthy (49/50)

**Root cause:** Gateway config routes /api/v1/provisioning → localhost:9090 (OPERATOR_SERVICE_URL)
but no service runs on :9090 in all-in-one Docker.

**Fix options:**
1. Remove the route from gateway config (cleanest)
2. Add provisioning service to Docker (heavyweight)
3. Add health check skip for optional services (compromise)

## Positive Findings
- User Create: working with real DB
- Policy Create: working with real DB  
- OAuth Client Create: working with memory repo fallback
- Webhook Create: working
- MFA Setup: TOTP secret generated
- All frontend pages: 200 OK
- JWT scope: admin scope present

## Lessons
1. **gRPC-only features break REST consumers** — always wire both protocols
2. **Gateway routes for non-existent services** cause health check noise
