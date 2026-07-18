# E2E Business Flows Report

**Date:** 2026-07-18 19:31:39 UTC
**URL:** https://ggid.iot2.win

## Results

| Status | Count |
|--------|-------|
| ✅ PASS | 10 |
| ❌ FAIL | 1 |
| **Total** | **11** |

## Steps

| # | Step | Result | Latency |
|---|------|--------|---------|\n✅ PASS | 1. Login (admin) | 177ms\n✅ PASS | 2. Create User | 1106ms\n✅ PASS | 3. List Users (total=402) | 74ms\n✅ PASS | 4. Assign Role (viewer) | 78ms\n✅ PASS | 5. Check Permission | 96ms\n✅ PASS | 6. Create OAuth Client (id=gcid_NSUXT3dFsviJbeJ) | 350ms\n❌ FAIL | 7. Client Credentials Token | 350ms | code=201 body={"Client":{"ID":"222cd614-3ea1-460e-8b2a-5f5a9a39a312","TenantID":"00000000-0000\n✅ PASS | 8. Query Audit Events (count=0) | 90ms\n✅ PASS | 9. Create Webhook | 101ms\n✅ PASS | 10. Audit Export | 67ms\n✅ PASS | 11. List Sessions | 67ms

## Conclusion

1 flow(s) failed. See details above.
