# E2E Business Flows Report

**Date:** 2026-07-19 00:29:37 UTC
**URL:** https://ggid.iot2.win

## Results

| Status | Count |
|--------|-------|
| ✅ PASS | 10 |
| ❌ FAIL | 1 |
| **Total** | **11** |

## Steps

| # | Step | Result | Latency |
|---|------|--------|---------|\n✅ PASS | 1. Login (admin) | 160ms\n✅ PASS | 2. Create User | 321ms\n✅ PASS | 3. List Users (total=405) | 57ms\n❌ FAIL | 4. Assign Role | 55ms | code=403 body={"error":{"code":"permission_denied","message":"admin role required to assign roles"}}\n✅ PASS | 5. Check Permission | 56ms\n✅ PASS | 6. Create OAuth Client (id=gcid_0-UM8_MQsy5xkcS) | 117ms\n✅ PASS | 7. Client Credentials Token | 80ms\n✅ PASS | 8. Query Audit Events (count=0) | 159ms\n✅ PASS | 9. Create Webhook | 67ms\n✅ PASS | 10. Audit Export | 69ms\n✅ PASS | 11. List Sessions | 65ms

## Conclusion

1 flow(s) failed. See details above.
