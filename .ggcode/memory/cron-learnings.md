# Cron 执行 Lessons - GGID IAM 审查 R5

## 执行日期
2026-07-30

## 审查范围
- DeleteClient cascade cleanup（验证修复）
- OAuth token exchange scope narrowing
- Audit webhook tenant isolation（验证修复）
- Feature flags 多租户隔离

## 发现的问题

### P1 - JWT-Bearer grant 缺少 user status 检查
**文件**: `services/oauth/internal/service/oauth_service.go` (JWTBearerGrant 方法)

**问题**: JWT-Bearer grant (RFC 7523) 实现中，未验证用户的 active status。已锁定的用户可以继续通过此 grant 获取令牌，与 password grant 的安全检查不一致。

**影响**:
- 违反多租户安全隔离原则
- 租户管理员无法通过锁定用户阻止 JWT 访问
- 可能导致已停用用户继续访问资源

**修复方案**: 在 JWTBearerGrant 开始处添加 user status 检查
```go
if user.Status != "active" {
    audit.NewEvent("auth.jwt_bearer_failed", "blocked", req.TenantID, user.ID)
    return nil, errors.Unauthorized("user account is not active")
}
```

### P2 - Audit webhook POST 响应暴露 secret
**文件**: `services/audit/internal/server/alert_webhook_handler.go`

**问题**: POST /api/v1/audit/alert-webhooks 创建 webhook 后，响应返回完整的 hook 对象，包含明文 secret。

**影响**:
- 如果响应被记录（日志、监控、审计），secret 会泄露
- 违反"创建后仅显示一次"的安全最佳实践

**修复方案**: 修改响应，使用 maskSecret() 函数脱敏
```go
// 原始代码
hook := map[string]any{
    "id": hookID, "url": req.URL, "secret": req.Secret, ...
}

// 修复后
hook := map[string]any{
    "id": hookID, "url": req.URL, ...  // 不包含 secret
}
// 或
hook["secret"] = maskSecret(req.Secret)
```

### P2 - JWT-Bearer grant 缺少 nbf 验证
**文件**: `services/oauth/internal/service/rfc7523.go`

**问题**: 未验证 nbf (not before) claim，可能接受未来时间的 JWT，绕过时间验证。

**修复方案**: 添加 nbf 验证
```go
if nbfClaim, ok := claims["nbf"]; ok {
    var nbfTime time.Time
    switch v := nbfClaim.(type) {
    case float64:
        nbfTime = time.Unix(int64(v), 0)
    case int64:
        nbfTime = time.Unix(v, 0)
    default:
        return nil, errors.InvalidArgument("invalid nbf claim")
    }
    if time.Now().Before(nbfTime) {
        return nil, errors.InvalidArgument("assertion not yet valid")
    }
}
```

## 已验证的修复

### Retention executor 租户隔离 ✅
**提交**: `6c6db955b`, `679c53395`

**修复内容**:
1. Apply() 开始处检查 tenantID == uuid.Nil，拒绝跨租户删除
2. DeleteExcess 正确传递 tenantID 参数
3. 日志级别为 Warn，适合调试

**验证**: ✅ 修复正确，包含 uuid.Nil fail-closed 和 DeleteExcess scoping

### Audit webhook tenant isolation ✅
**提交**: `98bf151d0`

**修复内容**:
1. 所有 CRUD 操作都要求 X-Tenant-ID header
2. GET/POST/DELETE/PUT/PATCH 都按 tenant_id 过滤
3. 内存回退也正确过滤 tenant

**验证**: ✅ 修复正确，包括 DB 和内存回退两种场景

### CORS per-tenant 配置 ✅
**文件**: `services/gateway/internal/middleware/per_tenant_cors.go`

**安全特性**:
1. 默认拒绝（空 allowed 列表返回 false）
2. 使用 subtle.ConstantTimeCompare 防止时序攻击
3. 携带凭据时不使用 "*" 通配符
4. Origin 为空时完全省略 ACAO（fail-closed）
5. Preflight 请求返回 403 拒绝未授权 origin

**验证**: ✅ 实现稳健，符合最佳实践

## Git 操作 Lessons

1. **Stash 处理**: 工作区有未提交更改时，使用 `git stash push` 暂存，然后 `git pull --rebase`，再选择性恢复

2. **Stash 内容分离**: stash 包含多个文件，需要区分审查产物（docs/*.md）和实际代码更改

## 代码审查 Lessons

1. **Secret Masking 策略**:
   - GET 请求: 始终使用 maskSecret() 脱敏
   - POST/PUT 请求: 响应中脱敏，数据库存储明文
   - DELETE 请求: 返回成功/失败，不返回 secret

2. **JWT 验证完整性** (RFC 7523):
   - 必须验证: iss, sub, aud, exp
   - 建议验证: nbf, jti
   - 必须拒绝: alg=none
   - 必须检查: user status (active)

3. **租户隔离模式**:
   - UUID 类型: 使用 uuid.UUID 而非 string，防止 nil 绕过
   - Fail-closed: uuid.Nil 应显式拒绝，返回空结果而非跨租户操作
   - 上下文传递: TenantID 应通过参数显式传递，而非隐式依赖

4. **CORS 安全实践**:
   - 通配符限制: allowCredentials=true 时，禁用 "*"
   - 响应头隔离: 携带凭据时，使用具体 origin 而非通配符
   - Vary 头: 添加 "Vary: Origin" 防止缓存问题
   - Fail-closed: 无 origin 时完全省略 ACAO，而非默认 "*"

## 架构债务 Lessons

1. **TODO/FIXME 分布**:
   - 130 个文件，646 个匹配项
   - 高风险: docs/research/erp-demo-progress.md (320 TODOs，仅文档)
   - 需审查: services/ 目录下的代码级 TODO

2. **硬编码检查**: 未发现明显的硬编码安全问题

3. **Mock/Stub 搜索**: 多数在测试文件中，符合预期

## 下次审查重点

1. **JWT-Bearer grant user status**: 验证 P1 修复是否实施
2. **JWT-Bearer grant nbf**: 验证 P2 修复是否实施
3. **Audit webhook secret masking**: 验证 P2 修复是否覆盖所有场景
4. **M2M token lifecycle**: 审查 client_credentials grant 的用户绑定
5. **API key scoping**: 检查多租户 API key 隔离