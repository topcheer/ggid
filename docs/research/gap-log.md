## GAP Log 2026-07-18T10:10:07Z

| Component | Status | Detail |
|-----------|--------|--------|
| NHI lifecycle | ❌ 内存 map | make(map[string]*NHIIdentity), 83行, 重启丢数据 |
| NHI risk engine | ❌ 内存 | 无 pool/pgx 引用 |
| Conditional access | ✅ DB-backed | pgxpool.Pool |
| CAE evaluations | ✅ DB-backed | pgxpool.Pool |
| Privilege creep | ✅ DB-backed | pcRepo with pgx |
| Delegation | ✅ DB-backed | pgxpool.Pool |
| Password strength | ✅ 纯计算 | 无需 DB |
