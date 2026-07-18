# E2E Business Flows Report

**Date:** 2026-07-18 23:15:05 UTC
**URL:** https://ggid.iot2.win

## Results

| Status | Count |
|--------|-------|
| ✅ PASS | 1 |
| ❌ FAIL | 10 |
| **Total** | **11** |

## Steps

| # | Step | Result | Latency |
|---|------|--------|---------|\n✅ PASS | 1. Login (admin) | 263ms\n❌ FAIL | 2. Create User | 85ms | code=401 body={"detail":"invalid or expired token","title":"Unauthenticated","type":"https://ggid.dev/errors/unaut\n❌ FAIL | 3. List Users | 62ms | code=401 total=0\n❌ FAIL | 4. Assign Role | 0ms | no user_id\n❌ FAIL | 5. Check Permission | 61ms | code=401 body={"detail":"invalid or expired token","title":"Unauthenticated","type":"https://g\n❌ FAIL | 6. Create OAuth Client | 62ms | code=401 body={"detail":"invalid or expired token","title":"Unauthenticated","type":"https://ggid.dev/errors/unaut\n❌ FAIL | 7. Client Credentials Token | 0ms | missing client_id or secret\n❌ FAIL | 8. Query Audit Events | 59ms | code=401\n❌ FAIL | 9. Create Webhook | 58ms | code=401 body={"detail":"invalid or expired token","title":"Unauthenticated","type":"https://g\n❌ FAIL | 10. Audit Export | 56ms | code=401\n❌ FAIL | 11. List Sessions | 55ms | code=401

## Conclusion

10 flow(s) failed. See details above.
