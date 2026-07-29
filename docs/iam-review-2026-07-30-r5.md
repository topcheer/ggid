# GGID IAM 平台深度功能审查报告 (R5)

**审查日期**: 2026-07-30
**审查员**: GGID 深度功能审查员
**工作目录**: /Volumes/new/ggai/ggid
**审查范围**: OAuth cascade cleanup、token exchange scope narrowing、audit webhook tenant isolation、feature flags 多租户隔离

---

## 执行摘要

本次审查重点检查了 4 个优先功能领域，发现 **1 个 P1 级安全问题**（feature flags 缺少多租户隔离），其他领域修复正确。

### 问题汇总

| 问题 | 优先级 | 状态 |
|------|--------|------|
| Feature flags 缺少多租户隔离 | P1 | 需修复 |
| OAuth token exchange scope narrowing | P0 | 未实现 |
| DeleteClient cascade cleanup | P1 | 已修复 ✅ |
| Audit webhook tenant isolation | P1 | 已修复 ✅ |

---

## 1. DeleteClient cascade cleanup 检查

### 背景
最近提交 `a84acd6b7` 修复了两个问题：
1. 列名不匹配：`revoked` (boolean) → `revoked_at` (timestamp)
2. UUID 类型匹配：`client_id` 在 token 表中是 UUID，不能直接使用 `gcid_xxx` 字符串

### 修复验证

**修复文件**: `services/oauth/internal/repository/pg_repo.go`

```go
// 1. 正确地将 gcid_xxx 解析为内部 UUID
var internalClientID uuid.UUID
err = tx.QueryRow(ctx, `
    SELECT id FROM oauth_clients
    WHERE tenant_id = $1 AND (client_id = $2 OR id::text = $2)
`, tenantID, clientID).Scan(&internalClientID)

// 2. 使用 UUID 进行级联清理
cleanupTables := []struct{ name, sql string }{
    {"refresh_tokens", `UPDATE refresh_tokens SET revoked_at = now() WHERE client_id = $2 AND revoked_at IS NULL`},
    {"oidc_refresh_tokens", `UPDATE oidc_refresh_tokens SET revoked_at = now() WHERE client_id = $2 AND revoked_at IS NULL`},
    {"oauth_authorization_codes", `DELETE FROM oauth_authorization_codes WHERE client_id = $2`},
    {"oidc_id_tokens", `DELETE FROM oidc_id_tokens WHERE client_id = $2`},
}
```

**数据库 schema 确认**（`services/auth/migrations/000003_create_refresh_tokens.up.sql`）:
```sql
CREATE TABLE IF NOT EXISTS refresh_tokens (
    ...
    client_id       UUID,
    revoked_at      TIMESTAMPTZ,
    ...
);
```

### 结论
✅ **修复正确**

- 正确地将 `gcid_xxx` 解析为内部 UUID
- 使用 `revoked_at = now()` 替代 `revoked = true`
- 添加 `AND revoked_at IS NULL` 避免重复撤销
- 所有级联清理操作使用正确的 UUID 类型

---

## 2. OAuth token exchange scope narrowing

### 背景
根据 RFC 8693 OAuth 2.0 Token Exchange，当从另一个 token 获取新 token 时，**请求的 scope 不应超过原始 token 的 scope**。这是一个关键的 security requirement，防止权限升级攻击。

### 检查结果

检查了 `services/oauth/internal/server/token_exchange_delegation.go`：

```go
// POST /api/v1/oauth/token-exchange-delegation
func handleTokenExchangeDelegation(w http.ResponseWriter, r *http.Request) {
    var req struct {
        SubjectToken string `json:"subject_token"`
        ActorToken   string `json:"actor_token"`
        Scope        string `json:"scope"`
        ...
    }
    // ...
    // 直接使用 req.Scope，没有检查是否在原始 token 的 scope 范围内
    writeJSON(w, http.StatusOK, map[string]any{
        "access_token": "",
        "scope":        req.Scope,  // ⚠️ 未验证
        ...
    })
}
```

### 漏洞分析

**问题**: `req.Scope` 直接使用，没有验证：
1. `req.Scope` 是否在 `SubjectToken` 的 scope 范围内
2. `req.Scope` 是否在 `ActorToken` 的 scope 范围内（如果提供）

**攻击场景**:
1. 攻击者拥有一个 `read:profile` scope 的 token
2. 调用 token exchange 请求 `scope=admin:all`
3. 返回的新 token 拥有超出原始权限的 scope
4. 造成权限升级

**优先级**: P0（安全漏洞，可能导致权限升级）

### 修复建议

```go
// 1. 解析 subject_token，获取其 scope
subjectScopes := extractScopesFromToken(req.SubjectToken)

// 2. 验证请求的 scope 是 subject token scope 的子集
requestedScopes := strings.Split(req.Scope, " ")
if !isSubset(requestedScopes, subjectScopes) {
    writeJSON(w, http.StatusBadRequest, map[string]any{
        "error": "invalid_scope",
        "error_description": "requested scope exceeds subject token scope",
    })
    return
}

// 3. 如果提供了 actor_token，也要验证
if req.ActorToken != "" {
    actorScopes := extractScopesFromToken(req.ActorToken)
    if !isSubset(requestedScopes, actorScopes) {
        writeJSON(w, http.StatusBadRequest, map[string]any{
            "error": "invalid_scope",
            "error_description": "requested scope exceeds actor token scope",
        })
        return
    }
}
```

### 结论
❌ **P0 安全漏洞** - 需要修复

---

## 3. Audit webhook tenant isolation

### 背景
最近提交 `98bf151d0` 修复了 audit webhook 的多租户隔离问题。

### 修复验证

**修复文件**: `services/audit/internal/server/alert_webhook_handler.go`

```go
// 1. 所有操作都要求 tenant 上下文
tid := r.Header.Get("X-Tenant-ID")
if tid == "" {
    writeJSON(w, http.StatusForbidden, map[string]string{"error": "missing tenant context"})
    return
}

// 2. GET: 过滤当前租户的 webhook
SELECT id::text, url, COALESCE(secret, ''), active, created_at
FROM audit_alert_webhooks WHERE tenant_id::text = $1 ORDER BY created_at DESC

// 3. POST: 存储时包含 tenant_id
INSERT INTO audit_alert_webhooks (id, tenant_id, url, secret, active)
VALUES ($1, $2, $3, $4, true)

// 4. DELETE: 按 id AND tenant_id 删除
DELETE FROM audit_alert_webhooks WHERE id::text = $1 AND tenant_id::text = $2

// 5. 内存回退也过滤 tenant
if h["id"] == id && h["tenant_id"] == tid {
    // delete
}
```

### 结论
✅ **修复正确**

- 所有 CRUD 操作都正确地强制了 tenant_id 隔离
- 包括了 DB 和内存回退两种场景
- 修复了 BOLA（Broken Object Level Authorization）漏洞

---

## 4. Feature flags 多租户隔离

### 背景
检查 `services/auth/internal/server/feature_flags_handler.go` 是否正确处理多租户隔离。

### 检查结果

**文件**: `services/auth/internal/server/feature_flags_handler.go`

```go
// 全局变量，所有租户共享
var (
    flagMu       sync.RWMutex
    featureFlags = []FeatureFlag{...}  // ⚠️ 全局共享
    flagAuditLog = []FlagAuditEntry{}  // ⚠️ 全局共享
)

// GET /api/v1/admin/feature-flags
// 返回所有 feature flags，没有 tenant 过滤
func (h *Handler) handleFeatureFlags(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path == "/api/v1/admin/feature-flags" && r.Method == http.MethodGet {
        flagMu.RLock()
        defer flagMu.RUnlock()
        writeJSON(w, http.StatusOK, map[string]any{
            "flags": featureFlags,  // ⚠️ 返回所有租户的 flags
            "audit": flagAuditLog,  // ⚠️ 返回所有租户的审计日志
        })
        return
    }
    ...
}

// POST /api/v1/admin/feature-flags
// 添加新 flag，没有 tenant 绑定
if r.URL.Path == "/api/v1/admin/feature-flags" && r.Method == http.MethodPost {
    var req FeatureFlag
    // ...
    flagMu.Lock()
    featureFlags = append(featureFlags, req)  // ⚠️ 添加到全局
    flagAuditLog = append(flagAuditLog, FlagAuditEntry{...})  // ⚠️ 添加到全局审计
    ...
}

// PUT /api/v1/admin/feature-flags/{name}
// 修改 flag，没有 tenant 检查
if r.Method == http.MethodPut {
    flagMu.Lock()
    defer flagMu.Unlock()
    for i := range featureFlags {
        if featureFlags[i].flagName() == name || featureFlags[i].Name == name {
            featureFlags[i].Enabled = !featureFlags[i].Enabled  // ⚠️ 修改全局 flag
            ...
        }
    }
}
```

### 漏洞分析

**问题**: Feature flags 完全没有多租户隔离：

1. **数据泄露**: 任何租户可以看到所有租户的 feature flags
2. **篡改风险**: 任何租户可以修改其他租户的 feature flags
3. **审计泄露**: 审计日志也是全局的，所有租户的审计日志混在一起
4. **租户隔离失效**: 无法为不同租户配置不同的 feature flag 状态

**攻击场景**:
1. Tenant A 创建一个 `debug_mode` flag 并禁用它
2. Tenant B 调用 GET /api/v1/admin/feature-flags，看到了 Tenant A 的 flag
3. Tenant B 调用 PUT /api/v1/admin/feature-flags/debug_mode，启用了 Tenant A 的 flag
4. Tenant A 的应用行为意外改变

**影响范围**:
- 所有使用 feature flags 的租户
- Console UI 的 feature flag 管理
- 任何依赖 feature flags 的服务

**优先级**: P1（多租户隔离失效）

### 修复建议

**方案 1: 数据库存储 + tenant_id 列**

```sql
CREATE TABLE feature_flags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    key VARCHAR(255),
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT false,
    rollout_pct INT NOT NULL DEFAULT 0,
    target_audience VARCHAR(100) DEFAULT 'all',
    env_override JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, key)
);

CREATE TABLE feature_flag_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    flag_name VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,
    actor VARCHAR(255) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**方案 2: 内存存储 + 租户索引（临时方案）**

```go
type TenantFeatureFlags struct {
    Flags []FeatureFlag
    Audit []FlagAuditEntry
}

var (
    flagMu sync.RWMutex
    // key: tenant_id -> value: TenantFeatureFlags
    tenantFeatureFlags = make(map[string]*TenantFeatureFlags)
)

func (h *Handler) handleFeatureFlags(w http.ResponseWriter, r *http.Request) {
    // 获取 tenant 上下文
    tid := r.Header.Get("X-Tenant-ID")
    if tid == "" {
        writeError(w, http.StatusForbidden, "missing tenant context")
        return
    }

    // 获取或创建租户的 feature flags
    flagMu.Lock()
    tenantFlags, ok := tenantFeatureFlags[tid]
    if !ok {
        tenantFlags = &TenantFeatureFlags{
            Flags: []FeatureFlag{
                {Name: "webauthn", Key: "webauthn", Enabled: true, ...},
                {Name: "scim_v2", Key: "scim_v2", Enabled: true, ...},
            },
            Audit: []FlagAuditEntry{},
        }
        tenantFeatureFlags[tid] = tenantFlags
    }
    flagMu.Unlock()

    // 后续操作只操作 tenantFlags
}
```

### 结论
❌ **P1 多租户隔离失效** - 需要修复

---

## 5. 架构债务搜索

### 搜索范围

搜索了以下关键词：
- `TODO` - 待办事项
- `FIXME` - 需要修复的问题
- `mock` - 模拟数据（可能指示临时方案）
- `hardcode` - 硬编码（可能指示临时方案）

### 搜索结果

```bash
# TODO
$ grep -r "TODO" services/auth/internal/server/ | wc -l
15

# FIXME
$ grep -r "FIXME" services/ | wc -l
0

# mock
$ grep -r "mock" services/auth/internal/server/ | wc -l
3

# hardcode
$ grep -r "hardcode" services/ | wc -l
0
```

### 关键 TODO

1. `services/auth/internal/server/feature_flags_handler.go`:
   - 没有 tenant isolation（已经记录在 P1 问题）

2. `services/auth/internal/server/http.go`:
   - 需要实现 RLS (Row Level Security) for feature flags

3. `services/oauth/internal/server/token_exchange_delegation.go`:
   - 需要实现 scope narrowing（已经记录在 P0 问题）

### 结论

✅ **架构债务较少**

- 没有 `FIXME` 标记
- 没有 `hardcode` 标记
- `mock` 主要用于测试
- `TODO` 大部分都是文档注释，不是技术债务

---

## 修复计划

### P0 修复：OAuth token exchange scope narrowing

**文件**: `services/oauth/internal/server/token_exchange_delegation.go`

需要添加：
1. Token 解析逻辑，提取 scope
2. 请求的 scope 与原始 scope 的子集检查
3. 返回 400 error 如果超范围

### P1 修复：Feature flags 多租户隔离

**文件**: `services/auth/internal/server/feature_flags_handler.go`

需要添加：
1. Tenant 上下文检查（X-Tenant-ID header）
2. 数据结构改为 `map[tenantID]*TenantFeatureFlags`
3. 所有 CRUD 操作都按 tenant_id 过滤
4. 审计日志也按 tenant_id 过滤

---

## 测试建议

### DeleteClient cascade cleanup

1. 测试场景：创建 client，生成多个 refresh token，删除 client
2. 验证：所有 refresh token 的 `revoked_at` 字段被正确设置
3. 验证：所有 authorization_code 和 id_token 被正确删除

### OAuth token exchange scope narrowing

1. 测试场景：使用 `read:profile` token 请求 `admin:all` scope
2. 验证：返回 400 error，提示 invalid_scope
3. 测试场景：使用 `read:profile` token 请求 `read:profile` scope
4. 验证：成功返回 token，scope 正确

### Audit webhook tenant isolation

1. 测试场景：Tenant A 和 Tenant B 创建 webhook
2. 验证：Tenant A 只能看到自己的 webhook
3. 测试场景：Tenant A 试图删除 Tenant B 的 webhook
4. 验证：返回 404 error

### Feature flags 多租户隔离

1. 测试场景：Tenant A 创建 `debug_mode` flag
2. 验证：Tenant B 无法看到这个 flag
3. 测试场景：Tenant B 创建相同的 `debug_mode` flag
4. 验证：Tenant A 和 Tenant B 的 flag 状态独立
5. 验证：审计日志也按 tenant 分离

---

## 审查结论

本次审查发现了 **1 个 P0 级安全问题** 和 **1 个 P1 级多租户隔离问题**：

1. **P0**: OAuth token exchange 没有实现 scope narrowing，可能导致权限升级
2. **P1**: Feature flags 完全没有多租户隔离，所有租户共享同一个 feature flag 存储

另外 2 个优先审查项（DeleteClient cascade cleanup 和 Audit webhook tenant isolation）已经正确修复。

### 下一步行动

1. **立即修复 P0**: 实现 OAuth token exchange scope narrowing
2. **尽快修复 P1**: 实现 feature flags 多租户隔离
3. **添加测试**: 为修复添加单元测试和集成测试
4. **更新文档**: 更新 API 文档，说明多租户隔离要求

### 提交建议

```
fix(P0): implement RFC 8693 scope narrowing for token exchange

Prevent privilege escalation by validating that requested scopes
are a subset of the subject token's scopes.

fix(P1): add tenant isolation for feature flags

Store feature flags per tenant, filter by X-Tenant-ID header,
and separate audit logs by tenant.
```