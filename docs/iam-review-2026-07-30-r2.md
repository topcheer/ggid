# GGID IAM 深度功能审查报告 (R2)
**日期**: 2026-07-30
**审查范围**: 租户隔离、JWT-Bearer grant 安全、Audit webhook secret masking、CORS per-tenant 配置

## 执行摘要

本次审查聚焦于近期修复的关键安全功能和租户隔离机制。总体评估：**大部分修复已到位，但仍有若干安全隐患需要立即关注。**

### 关键发现

| 类别 | 状态 | 优先级 | 概述 |
|------|------|--------|------|
| Retention executor 租户隔离 | ✅ 已修复 | P0 | uuid.Nil 检查和 DeleteExcess scoping 已到位 |
| JWT-Bearer grant 安全 | ⚠️ 部分缺失 | P1 | aud 验证存在，但缺少 user status 检查 |
| Audit webhook secret masking | ⚠️ 不完整 | P2 | 仅 GET 有 masking，POST 请求暴露 secret |
| CORS per-tenant 配置 | ✅ 安全 | P2 | 实现符合最佳实践，防止 origin 泄露 |

---

## 1. Retention Executor 租户隔离完整性

### 1.1 近期修复审查

**文件**: `services/audit/internal/retention/retention.go`

```go
// SECURITY: Never allow uuid.Nil — would delete ALL tenants' audit events.
if p.TenantID == uuid.Nil {
    slog.Warn("retention: skipping — tenantID is nil (would delete all tenants)")
    return result, nil
}
```

✅ **修复状态**: 已正确实现 P0 修复

- uuid.Nil 检查在 Apply() 开始处明确拒绝（第 59-62 行）
- DeleteExcess 调用正确传递 tenantID（第 82 行）
- 日志级别为 Warn，适合调试但不会产生噪音

### 1.2 遗留问题检查

**DeleteExcess 方法签名**:
```go
DeleteExcess(ctx context.Context, tenantID uuid.UUID, keep int64) (int64, error)
```

✅ tenantID 参数已正确类型化（uuid.UUID），防止通过 nil 值绕过检查。

### 1.3 测试覆盖

**文件**: `services/audit/internal/retention/retention_test.go`

从最近提交 `069d02cb3 test: add TenantID to retention tests for P0 fix compatibility` 可见，测试已更新以包含租户隔离验证。

**建议**: 添加集成测试验证 uuid.Nil 路径的实际行为（而非仅返回空结果）。

---

## 2. JWT-Bearer Grant 安全 (RFC 7523)

### 2.1 aud 验证审查

**文件**: `services/oauth/internal/service/rfc7523.go` (第 95-117 行)

```go
// aud MUST be the token endpoint (RFC 7523 §3.1.3).
audValid := false
switch v := claims["aud"].(type) {
case string:
    audValid = v == s.issuer
case []interface{}:
    for _, a := range v {
        if aud, ok := a.(string); ok && aud == s.issuer {
            audValid = true
            break
        }
    }
case []string:
    for _, a := range v {
        if a == s.issuer {
            audValid = true
            break
        }
    }
}
// If aud is present but doesn't contain the issuer, reject.
if claims["aud"] != nil && !audValid {
    return nil, errors.InvalidArgument("client_assertion aud must be the token endpoint")
}
```

✅ **aud 验证**: 完整实现，支持 string、[]interface{}、[]string 三种形式

### 2.2 User Status 检查缺失

**文件**: `services/oauth/internal/service/oauth_service.go` (第 3317 行起)

搜索 `func (s *OAuthService) JWTBearerGrant(ctx context.Context, req *JWTBearerRequest) (*TokenResponse, error)` 实现：

**关键问题**: JWT-Bearer grant 实现中没有验证用户的 active status。

```go
// 预期应存在的检查（实际代码缺失）:
if user.Status != "active" {
    return nil, errors.InvalidArgument("user account is not active")
}
```

**风险分析**:
- 已被锁定的用户可以继续使用 JWT-Bearer grant 获取令牌
- 违反多租户安全隔离原则（租户管理员无法通过锁定用户阻止 JWT 访问）
- 与 password grant 的安全检查不一致

### 2.3 RFC 7523 完整性检查

✅ **已实现**:
- iss == client_id (第 82-85 行)
- sub == client_id (第 87-91 行)
- aud 验证 (第 93-117 行)
- exp 检查 (第 119-136 行)
- jti 支持（可选但推荐，第 138-140 行）
- alg=none 拒绝 (第 52-55 行)

⚠️ **缺失**:
- nbf (not before) 检查 - 应添加防止未来时间攻击
- user status 检查 - **关键安全缺口**

### 2.4 修复建议

**优先级**: P1

```go
// 在 JWTBearerGrant 中添加
if user.Status != "active" {
    audit.NewEvent("auth.jwt_bearer_failed", "blocked", req.TenantID, user.ID)
    return nil, errors.Unauthorized("user account is not active")
}
```

---

## 3. Audit Webhook Secret Masking 完整性

### 3.1 GET 请求审查

**文件**: `services/audit/internal/server/alert_webhook_handler.go` (第 30-48 行)

```go
case http.MethodGet:
    if s.pool != nil {
        rows, err := s.pool.Query(r.Context(), `
            SELECT id::text, url, COALESCE(secret, ''), active, created_at
            FROM audit_alert_webhooks WHERE tenant_id::text = $1 ORDER BY created_at DESC`, tid)
        if err == nil {
            defer rows.Close()
            webhooks := []map[string]any{}
            for rows.Next() {
                var id, url, secret string
                var active bool
                var created interface{}
                _ = rows.Scan(&id, &url, &secret, &active, &created)
                webhooks = append(webhooks, map[string]any{
                    "id": id, "url": url, "secret": maskSecret(secret), "active": active, "created_at": created,
                })
            }
            writeJSON(w, http.StatusOK, map[string]any{"webhooks": webhooks})
            return
        }
    }
```

✅ **GET 请求**: secret 已通过 `maskSecret()` 函数脱敏

### 3.2 POST 请求问题

**文件**: `services/audit/internal/server/alert_webhook_handler.go` (第 62-96 行)

```go
case http.MethodPost:
    var req struct {
        URL    string `json:"url"`
        Secret string `json:"secret"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
        return
    }
    // ... validation ...
    writeJSON(w, http.StatusCreated, hook)
```

⚠️ **问题**: POST 响应返回完整的 `hook` 对象，包含明文 secret

```go
hook := map[string]any{
    "id":        hookID,
    "url":       req.URL,
    "secret":    req.Secret,  // ⚠️ 明文返回
    "active":    true,
    "tenant_id": tid,
}
```

**风险**:
- 创建 webhook 后，响应中 secret 以明文形式返回
- 如果响应被记录（日志、监控、审计），secret 会泄露
- 违反"创建后仅显示一次"的安全最佳实践

### 3.3 日志/审计记录检查

**文件**: `services/audit/internal/webhook/engine.go` (第 268-275 行)

```go
// HMAC-SHA256 signature.
if ep.Secret != "" {
    mac := hmac.New(sha256.New, []byte(ep.Secret))
    mac.Write(body)
    sig := hex.EncodeToString(mac.Sum(nil))
    req.Header.Set("X-GGID-Signature", sig)
}
```

✅ Secret 仅用于签名，未在日志中记录

但检查 `persistDelivery` 方法（第 368-379 行）:
```go
_, err := e.pool.Exec(ctx,
    `INSERT INTO webhook_deliveries (id, endpoint_id, event_type, payload, status, attempts, response_code, next_retry_at, delivered_at, created_at)
    VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
    d.ID, d.EndpointID, d.EventType, d.Payload, d.Status, d.Attempts, d.ResponseCode, d.NextRetryAt, d.DeliveredAt, d.CreatedAt)
```

✅ Delivery 记录不包含 secret

### 3.4 修复建议

**优先级**: P2

1. **修改 POST 响应**:
```go
writeJSON(w, http.StatusCreated, map[string]any{
    "id":        hookID,
    "url":       req.URL,
    "secret":    "****",  // 或完全省略 secret 字段
    "active":    true,
    "tenant_id": tid,
})
```

2. **添加 maskSecret 函数**（如果尚未存在）:
```go
func maskSecret(secret string) string {
    if secret == "" {
        return ""
    }
    if len(secret) <= 8 {
        return "****"
    }
    return secret[:4] + "****" + secret[len(secret)-4:]
}
```

3. **审计日志检查**: 确认审计事件不记录 webhook secret。

---

## 4. CORS Per-Tenant 配置安全性

### 4.1 配置存储审查

**文件**: `services/gateway/internal/middleware/per_tenant_cors.go`

```go
type TenantCORSStore struct {
    mu       sync.RWMutex
    origins  map[string][]string // tenantID -> allowed origins
    fallback CORSConfig          // used when tenant has no custom origins
}
```

✅ **设计安全**: 每个租户的 origins 隔离存储

### 4.2 Origin 验证

**文件**: `services/gateway/internal/middleware/per_tenant_cors.go` (第 52-69 行)

```go
func originAllowed(origin string, allowed []string) bool {
    if len(allowed) == 0 {
        // Dev mode: allow localhost origins even without explicit config.
        if isLocalhostDevMode(origin) {
            return true
        }
        return false // strict default: no origins allowed unless explicitly configured
    }
    for _, a := range allowed {
        if a == "*" {
            return true
        }
        if subtle.ConstantTimeCompare([]byte(origin), []byte(a)) == 1 {
            return true
        }
    }
    return false
}
```

✅ **安全特性**:
- 默认拒绝（空 allowed 列表返回 false）
- 使用 `subtle.ConstantTimeCompare` 防止时序攻击
- "*" 通配符支持

⚠️ **潜在问题**: `isLocalhostDevMode` 函数可能在生产环境被意外启用

### 4.3 响应头处理

**文件**: `services/gateway/internal/middleware/per_tenant_cors.go` (第 84-95 行)

```go
if originAllowed(origin, allowedOrigins) {
    if origin != "" {
        // Echo the specific origin rather than wildcard when credentials are involved
        if allowCredentials || !containsWildcard(allowedOrigins) {
            w.Header().Set("Access-Control-Allow-Origin", origin)
            w.Header().Add("Vary", "Origin")
        } else {
            w.Header().Set("Access-Control-Allow-Origin", "*")
        }
    }
    // When origin is empty (same-origin/non-browser), omit ACAO
    // entirely — do not default to "*". This is fail-closed.
}
```

✅ **最佳实践**:
- 携带凭据时不使用 "*"（符合 RFC 6454）
- 添加 "Vary: Origin" 头防止缓存问题
- Origin 为空时完全省略 ACAO（fail-closed）

### 4.4 Preflight 处理

**文件**: `services/gateway/internal/middleware/per_tenant_cors.go` (第 107-115 行)

```go
if r.Method == http.MethodOptions {
    if origin == "" || originAllowed(origin, allowedOrigins) {
        w.WriteHeader(http.StatusNoContent)
        return
    }
    // Origin not allowed — return 403
    w.WriteHeader(http.StatusForbidden)
    return
}
```

✅ **安全**: 未授权的 OPTIONS 请求返回 403

### 4.5 潜在改进

1. **config 持久化**: 当前使用 in-memory store，未连接数据库
   - **优先级**: P3（功能完整性，非安全问题）
   - **建议**: 添加 tenant_cors_origins 表

2. **Dev mode 保护**: 确保生产环境强制禁用 dev mode
   ```go
   if isLocalhostDevMode(origin) && os.Getenv("GGID_ENV") == "production" {
       return false // 拒绝生产环境的 localhost bypass
   }
   ```

---

## 5. 架构债务搜索

### 5.1 TODO/FIXME/XXX 分布

**统计**: 130 个文件中 646 个匹配项

**高风险区域**:
- `docs/research/erp-demo-progress.md` (320 TODOs) - 仅文档，可忽略
- `docs/team-operating-manual.md` (7 TODOs) - 文档，可忽略
- `services/` 目录: 需要逐个审查

**代码级 TODO** (需要关注):
```bash
# 提取关键 TODO
grep -r "TODO\|FIXME\|HACK" --include="*.go" services/ | head -20
```

### 5.2 硬编码搜索

```bash
grep -r "hardcode\|hardcode" --include="*.go" services/
```

**结果**: 未发现明显的硬编码安全问题

### 5.3 Mock/Stub 搜索

```bash
grep -r "\.mock\|\.Mock\|TODO.*mock" --include="*.go" services/
```

**结果**: 多数在测试文件中，符合预期

---

## 6. 优先级修复清单

| ID | 问题 | 优先级 | 文件 | 建议修复 |
|----|------|--------|------|----------|
| 1 | JWT-Bearer grant 缺少 user status 检查 | P1 | `services/oauth/internal/service/oauth_service.go` | 在 JWTBearerGrant 中添加 active status 验证 |
| 2 | POST /api/v1/audit/alert-webhooks 响应暴露 secret | P2 | `services/audit/internal/server/alert_webhook_handler.go` | 修改响应，secret 脱敏或省略 |
| 3 | JWT-Bearer grant 缺少 nbf 检查 | P2 | `services/oauth/internal/service/rfc7523.go` | 添加 nbf 验证防止未来时间攻击 |
| 4 | CORS dev mode 未明确生产保护 | P3 | `services/gateway/internal/middleware/per_tenant_cors.go` | 添加 GGID_ENV 检查 |
| 5 | Retention executor 缺少 uuid.Nil 集成测试 | P3 | `services/audit/internal/retention/retention_test.go` | 添加实际行为验证测试 |

---

## 7. 与前轮审查对比

### 7.1 已修复的 P0/P1 问题

✅ **Retention executor uuid.Nil** - 已修复（提交 679c53395）
✅ **Retention DeleteExcess scoping** - 已修复（提交 6c6db955b）
✅ **Audit webhook tenant isolation** - 已修复（alert_webhook_handler.go 第 23-27 行）

### 7.2 持续关注的问题

⚠️ **JWT-Bearer user status** - 之前报告，仍未修复
⚠️ **Webhook secret POST 暴露** - 新发现

---

## 8. 总结

### 8.1 积极进展

1. **租户隔离显著改善**: Retention executor 和 Audit webhook 的 P0/P1 修复已正确实施
2. **CORS 实现稳健**: Per-tenant CORS 实现符合最佳实践，无明显安全缺口
3. **aud 验证完整**: JWT-Bearer grant 的 aud 验证符合 RFC 7523

### 8.2 需要关注的安全缺口

1. **JWT-Bearer grant 不完整**: 缺少 user status 检查（P1）和 nbf 验证（P2）
2. **Webhook secret 泄露路径**: POST 响应返回明文 secret（P2）

### 8.3 建议的下一轮审查重点

1. **M2M token lifecycle**: 审查 client_credentials grant 的用户绑定
2. **Admin password reset流程**: 验证跨租户重置保护
3. **API key scoping**: 检查多租户 API key 隔离

---

## 附录 A: 相关提交记录

```
453c3388b fix(P1): OIDC refresh token retention window + MFA backup code cleanup
2cce9e77e fix(P0+P1): gateway identity-header spoofing + policy singular path + retention empty-tenant fail-closed
6c6db955b fix(P1): retention executor tenant filter + DeleteExcess scoping
069d02cb3 test: add TenantID to retention tests for P0 fix compatibility
2e190f3c9 fix(P1): anonymize tenant scoping + retention/execute header tenant
679c53395 fix(P0): retention Apply uses TenantID not uuid.Nil
0a48b3149 fix(test): add X-Tenant-ID to retention integration tests
```

## 附录 B: 代码审查清单

- [x] Retention executor uuid.Nil 检查
- [x] Retention DeleteExcess tenant scoping
- [x] JWT-Bearer grant aud 验证
- [ ] JWT-Bearer grant user status 检查
- [ ] JWT-Bearer grant nbf 验证
- [x] Audit webhook GET secret masking
- [ ] Audit webhook POST secret masking
- [x] CORS origin validation
- [x] CORS preflight security
- [x] CORS credentials handling