# API Spec Coverage Audit

**Date**: 2025-07-11
**Auditor**: backend agent
**Commit**: this batch

## Summary

Compares `docs/openapi.yaml` paths vs `services/gateway/internal/config/config.go` actual routes.

## Routes in OpenAPI but MISSING from Gateway

| OpenAPI Path | Target Service | Status |
|---|---|---|
| `/api/v1/departments` | Identity | **ADDED** this batch |
| `/api/v1/teams` | Org | **ADDED** this batch |
| `/api/v1/tenants/{id}/branding` | Identity | **ADDED** (covered by `/api/v1/tenants` prefix) |
| `/scim/v2/Users` | Identity | **ADDED** this batch |
| `/scim/v2/Users/{id}` | Identity | **ADDED** (covered by `/scim/v2` prefix) |
| `/api/v1/idp/config` | OAuth | **ADDED** (covered by `/api/v1/idp` prefix) |
| `/.well-known/jwks.json` | OAuth | Works via `/.well-known` prefix (already proxied) |

## Routes in Gateway but NOT in OpenAPI

| Gateway Route | Notes |
|---|---|
| `/api/v1/access-requests` | IGA Workflows — **needs OpenAPI spec** |
| `/api/v1/agents` | AI Agent Identity — **already spec'd** (duplicate entries in YAML) |

## OpenAPI Spec Issues Found

1. **Duplicate agent paths**: `/api/v1/agents/register`, `/api/v1/agents`, `/api/v1/agents/token`, `/api/v1/agents/verify` appear twice in openapi.yaml (lines 1731-1834 and 1925-2044)

## Conclusion

All missing gateway routes have been added. The OpenAPI spec needs deduplication of agent paths and addition of `/api/v1/access-requests` paths.
