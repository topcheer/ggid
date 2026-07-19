# E2E Business Flows Report

**Date:** 2026-07-19 00:44:42 UTC
**URL:** https://ggid.iot2.win

## Results

| Status | Count |
|--------|-------|
| ✅ PASS | 10 |
| ❌ FAIL | 1 |
| **Total** | **11** |

## Steps

| # | Step | Result | Latency |
|---|------|--------|---------|\n✅ PASS | 1. Login (admin) | 234ms\n✅ PASS | 2. Create User | 324ms\n✅ PASS | 3. List Users (total=408) | 102ms\n❌ FAIL | 4. Assign Role | 69ms | code=403 body={"error":{"code":"permission_denied","message":"admin role required to assign roles"}}\n✅ PASS | 5. Check Permission | 70ms\n✅ PASS | 6. Create OAuth Client (id=gcid_XbpawgosgKXoHyp) | 122ms\n✅ PASS | 7. Client Credentials Token | 122ms\n✅ PASS | 8. Query Audit Events (count=9) | 71ms\n✅ PASS | 9. Create Webhook | 84ms\n✅ PASS | 10. Audit Export | 60ms\n✅ PASS | 11. List Sessions | 63ms

## Conclusion

1 flow(s) failed. See details above.
