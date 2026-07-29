# GGID IAM 系统审视报告 — 2026-07-29

## 回归状态

| 检查项 | 结果 |
|--------|------|
| `go build ./...` | PASS |
| `go test ./services/oauth/internal/service/` | PASS (5.063s) |
| `go test ./services/auth/internal/service/` | PASS (4.124s) |
| `go test ./services/identity/internal/scim/` | PASS (cached) |

## 最近 5 个 Commit 分析

| Commit | 描述 | 类型 |
|--------|------|------|
| `8ecc4b1c1` | OAuth client delete cascade — revoke tokens + cleanup codes | P1 fix |
| `a5533b352` | dummy hash format + device approve/consent cascade user_id BOLA | P0+P1 fix |
| `545d993b0` | add GetJSON to auditMemoryMapRepo2 (missing method) | bugfix |
| `527f7058b` | API key cache bypass + login timing attack + status leak | P0+P1 fix |
| `18b1e0b29` | remove user_id from consent_url (UUID leak prevention) | P1 fix |

## 发现的问题

### P1-1: DeleteClient cascade cleanup 因类型不匹配而静默失效

**严重级别**: P1 (安全)

**文件**: `services/oauth/internal/repository/pg_repo.go` L243-255

**描述**:

`DeleteClient` 接收 `clientID string` 参数（gcid_xxx 格式），在 cascade cleanup 中使用 `WHERE client_id = $2` 匹配 `refresh_tokens`、`oauth_authorization_codes`、`oidc_id_tokens` 表的 `client_id` 列。但这些表的 `client_id` 列是 UUID 类型，引用的是 `oauth_clients.id`（内部 UUID），不是 `oauth_clients.client_id`（gcid_xxx 字符串）。

```go
// 当前代码 — clientID 是 gcid_xxx 字符串，但 client_id 列是 UUID
cleanupTables := []struct{ name, sql string }{
    {"refresh_tokens", `UPDATE refresh_tokens SET revoked = true WHERE client_id = $2`},
    {"oidc_refresh_tokens", `UPDATE oidc_refresh_tokens SET revoked = true WHERE client_id = $2`},
    {"oauth_authorization_codes", `DELETE FROM oauth_authorization_codes WHERE client_id = $2`},
    {"oidc_id_tokens", `DELETE FROM oidc_id_tokens WHERE client_id = $2`},
}
for _, c := range cleanupTables {
    if _, err := tx.Exec(ctx, c.sql, tenantID, clientID); err != nil {
        // Best-effort: log but don't fail the delete
        log.Printf("DeleteClient: cascade cleanup failed table=%s err=%v", c.name, err)
    }
}
```

**证据**:
- `refresh_tokens.client_id` 是 UUID 类型 (migration L425)
- `oauth_authorization_codes.client_id` 是 UUID NOT NULL (migration L637)
- `oidc_id_tokens.client_id` 是 UUID NOT NULL (migration L667)
- `domain.RefreshToken.ClientID` 是 `*uuid.UUID` 类型 (token.go L15)
- 调用方传入 `result.Client.ClientID`（gcid_xxx 格式字符串）

**影响**:
- 删除 OAuth client 后，关联的 refresh tokens、auth codes、ID tokens 未被撤销/删除
- 被删除 client 的 refresh tokens 可在过期前继续使用（可能长达数周）
- `best-effort` 错误处理意味着即使 SQL 失败也不会阻止删除，用户以为 cascade 生效了但实际没有

**修复建议**:

在 cascade cleanup 之前，先查询 `oauth_clients` 表获取内部 UUID id，然后用 UUID 执行 cascade cleanup：

```go
// 先查出内部 UUID
var internalID uuid.UUID
err = tx.QueryRow(ctx, `SELECT id FROM oauth_clients WHERE tenant_id = $1 AND (client_id = $2 OR id::text = $2)`, tenantID, clientID).Scan(&internalID)
if err != nil {
    return ggiderrors.Wrap(ggiderrors.ErrNotFound, "client not found", err)
}

// 用 internalID 做 cascade cleanup
cleanupTables := []struct{ name, sql string }{
    {"refresh_tokens", `UPDATE refresh_tokens SET revoked = true WHERE client_id = $2`},
    {"oidc_refresh_tokens", `UPDATE oidc_refresh_tokens SET revoked = true WHERE client_id = $2`},
    {"oauth_authorization_codes", `DELETE FROM oauth_authorization_codes WHERE client_id = $2`},
    {"oidc_id_tokens", `DELETE FROM oidc_id_tokens WHERE client_id = $2`},
}
for _, c := range cleanupTables {
    if _, err := tx.Exec(ctx, c.sql, tenantID, internalID); err != nil {
        return ggiderrors.Wrap(ggiderrors.ErrInternal, "cascade cleanup failed: "+c.name, err)
    }
}
```

同时建议将 `best-effort` 改为事务内失败即回滚，确保 cascade 的原子性。

---

**总结: 发现 1 个问题：P0 0个 / P1 1个 / P2 0个**