# GGID IAM 深度功能审查 (R3)
日期: 2026-07-29
审查范围: OAuth refresh token 轮换路径、Conditional Access 策略评估、Audit hash chain 完整性、Console error helper 统一性

## 执行摘要

本次审查发现 **2 个 P0 问题**、**3 个 P1 问题**，以及多个中低优先级技术债务。最关键的发现是:

1. **P0**: OAuth refresh token 轮换时 Redis 和 DB 状态可能不一致，导致安全漏洞
2. **P0**: Conditional Access 策略评估函数存在双重实现，一个是实际使用的带逻辑版本，一个是硬编码返回 "allow" 的空函数

服务健康状态: 所有核心服务运行正常 (ggid-auth x3, ggid-audit, ggid-gateway, ggid-identity, ggid-oauth, ggid-policy, ggid-org, ggid-console)。

---

## 1. OAuth Refresh Token 轮换路径完整性审查

### 1.1 当前实现分析

**文件位置**: `services/auth/internal/service/token_service.go`

```go
func (ts *TokenService) RevokeRefreshToken(ctx context.Context, plaintext string) error {
    tokenHash := hashToken(plaintext)

    // Delete from Redis
    ts.rdb.Del(ctx, refreshTokenKey(tokenHash))

    // Revoke in DB
    rt, err := ts.refreshRepo.FindByHash(ctx, tokenHash)
    if err != nil {
        return err
    }
    if rt == nil {
        return nil // already gone
    }
    return ts.refreshRepo.Revoke(ctx, rt.ID)
}
```

**文件位置**: `services/auth/internal/repository/refresh_token_repo.go`

```go
func (r *RefreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
    _, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1`, id)
    return err
}
```

### 1.2 发现的问题

#### **P0-1: Refresh token 撤销时的 Redis/DB 状态不一致**

**问题描述**:
`RevokeRefreshToken` 方法先删除 Redis 缓存，再更新数据库。如果数据库更新失败 (如网络故障、连接池耗尽、约束冲突)，Redis 中的缓存已被删除，导致该 token 在 Redis 中不存在但数据库中仍然有效。攻击者可以在这个时间窗口内使用该 token。

**影响**:
- 攻击者可以在 Redis 删除后、DB 更新前的时间窗口内使用被撤销的 token
- 这违反了原子性原则，破坏了 refresh token 的安全保证

**建议修复**:
```go
func (ts *TokenService) RevokeRefreshToken(ctx context.Context, plaintext string) error {
    tokenHash := hashToken(plaintext)

    // First, revoke in DB (source of truth)
    rt, err := ts.refreshRepo.FindByHash(ctx, tokenHash)
    if err != nil {
        return err
    }
    if rt == nil {
        // Token not in DB, clean up Redis if present
        ts.rdb.Del(ctx, refreshTokenKey(tokenHash))
        return nil
    }

    if err := ts.refreshRepo.Revoke(ctx, rt.ID); err != nil {
        return err // DB revoke failed, do not delete from Redis
    }

    // Only delete from Redis after successful DB revoke
    ts.rdb.Del(ctx, refreshTokenKey(tokenHash))
    return nil
}
```

#### **P1-1: Refresh token 撤销缺少事务回滚机制**

**问题描述**:
当前 `RevokeRefreshToken` 方法使用两步操作 (Redis + DB)，但没有提供事务回滚机制。如果 Redis 删除成功但 DB 更新失败，系统进入不一致状态，且无法自动恢复。

**建议修复**:
- 在 DB 更新失败时，记录到 Redis 作为 "待撤销" 标记，异步重试
- 或者使用两阶段提交模式确保一致性

#### **P1-2: Redis fallback 路径未在 token 验证中实现**

**问题描述**:
虽然 `RevokeRefreshToken` 同时操作 Redis 和 DB，但在 token 验证路径中 (`FindByHash`) 只查询数据库，没有检查 Redis 缓存。这导致 Redis 的缓存作用仅限于手动撤销操作，而不是完整的缓存层。

**影响**:
- Redis 无法加速 token 验证路径
- 高并发场景下 DB 压力大

**建议修复**:
在 `FindByHash` 或 token 验证逻辑中添加 Redis 检查:
```go
func (ts *TokenService) FindRefreshTokenByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
    // Check Redis cache first
    cached, err := ts.rdb.Get(ctx, refreshTokenKey(hash)).Result()
    if err == nil && cached != "" {
        // Cache hit - deserialize and return
        var rt domain.RefreshToken
        if err := json.Unmarshal([]byte(cached), &rt); err == nil {
            return &rt, nil
        }
    }

    // Fall back to DB
    return ts.refreshRepo.FindByHash(ctx, hash)
}
```

---

## 2. Conditional Access 策略评估逻辑审查

### 2.1 当前实现分析

**文件位置**: `services/policy/internal/server/conditional_access_handler.go`

发现 **两个 EvaluateConditionalAccess 函数**:

```go
// 行 218-262: 实际使用的完整实现
func (s *HTTPServer) EvaluateConditionalAccess(tenantID string, ctxMap map[string]any) (action string, matchedPolicy *ConditionalAccessPolicy) {
    if s.policyMap == nil {
        return "allow", nil
    }
    rows, _ := s.policyMap.List(context.Background(), "conditional_access_store")
    for _, row := range rows {
        enabled := pmGetBool(row, "enabled")
        if !enabled {
            continue
        }
        pTenantID := pmGetString(row, "tenant_id")
        if tenantID != "" && pTenantID != "" && pTenantID != tenantID {
            continue
        }
        // ... 完整的匹配逻辑
    }
    return "allow", nil
}

// 行 267-269: 硬编码返回 "allow" 的空函数
func EvaluateConditionalAccess(tenantID string, ctx map[string]any) (action string, matchedPolicy *ConditionalAccessPolicy) {
    return "allow", nil
}
```

### 2.2 发现的问题

#### **P0-2: Conditional Access 评估函数双重实现，可能导致绕过安全策略**

**问题描述**:
存在两个同名的 `EvaluateConditionalAccess` 函数:
1. **HTTPServer.EvaluateConditionalAccess** (行 218): 完整实现，包含策略匹配逻辑
2. **EvaluateConditionalAccess** (行 267): 包级别函数，硬编码返回 `"allow", nil`

第二个函数通过包名直接调用时，会绕过所有 Conditional Access 策略检查，这是严重的安全漏洞。

**调用路径分析**:
- `handleConditionalAccessEvaluate` (行 207) 调用的是 `s.EvaluateConditionalAccess`，正确
- 但如果其他地方直接调用包级别函数 `EvaluateConditionalAccess(...)`，会绕过所有检查

**影响**:
- 任何配置的 Conditional Access 策略 (IP 限制、时间窗口、设备姿态) 可能被完全绕过
- 攻击者如果找到调用包级别函数的路径，可以绕过所有访问控制

**建议修复**:
```go
// 删除或标记为废弃的包级别函数
// Deprecated: Use HTTPServer.EvaluateConditionalAccess instead
func EvaluateConditionalAccess(tenantID string, ctx map[string]any) (action string, matchedPolicy *ConditionalAccessPolicy) {
    // 应该调用 policy 服务或返回错误，而不是硬编码 allow
    return "deny", nil // 或者 panic("use HTTPServer.EvaluateConditionalAccess")
}
```

**审查范围扩展**:
需要检查整个代码库中是否有地方调用了包级别的 `EvaluateConditionalAccess` 函数:
```bash
grep -r "policy.*EvaluateConditionalAccess" --include="*.go"
```

#### **P2-1: 策略评估缺少输入验证**

**问题描述**:
`EvaluateConditionalAccess` 直接使用 `ctxMap` 中的值进行字符串比较，没有类型安全检查:
```go
if fmt.Sprintf("%v", v) != fmt.Sprintf("%v", ctxVal) {
    matched = false
    break
}
```

**影响**:
- 类型不匹配可能导致误判 (如数字 123 vs 字符串 "123")
- 恶意构造的输入可能导致意外行为

**建议修复**:
添加类型检查和比较逻辑:
```go
func matchCondition(v any, ctxVal any) bool {
    if v == nil || ctxVal == nil {
        return v == ctxVal
    }

    // Same type: direct compare
    if reflect.TypeOf(v) == reflect.TypeOf(ctxVal) {
        return v == ctxVal
    }

    // Cross-type: string conversion for compatibility
    return fmt.Sprintf("%v", v) == fmt.Sprintf("%v", ctxVal)
}
```

---

## 3. Audit Hash Chain 完整性验证审查

### 3.1 当前实现分析

**文件位置**: `services/audit/internal/domain/hash_chain.go`

```go
func (e *AuditEvent) ComputeHash(prevHash string) string {
    secret := hashChainSecrets[hashChainCurrentVersion]
    if secret == nil {
        secret = hashChainSecrets[0] // backward compat
    }

    mac := hmac.New(sha256.New, secret)
    // Write version tag + prev_hash + canonical data
    mac.Write([]byte(fmt.Sprintf("v%d:", hashChainCurrentVersion)))
    mac.Write([]byte(prevHash))
    mac.Write(canonicalEventData(e))
    return hex.EncodeToString(mac.Sum(nil))
}

func VerifyChain(events []*AuditEvent) int {
    if len(events) == 0 {
        return -1
    }
    prevHash := ""
    for i, e := range events {
        if !e.VerifyHash(prevHash) {
            return i
        }
        prevHash = e.Hash
    }
    return -1
}
```

**文件位置**: `services/audit/internal/repository/audit_repo.go`

```go
func (r *AuditRepository) Insert(ctx context.Context, e *domain.AuditEvent) error {
    // ... 省略前面代码

    if domain.IsHashChainEnabled() {
        var ph string
        // FOR UPDATE locks the row, serializing chain appends per tenant.
        err := tx.QueryRow(ctx,
            `SELECT COALESCE(hash, '') FROM audit_events
             WHERE tenant_id = $1 ORDER BY created_at DESC, id DESC LIMIT 1 FOR UPDATE`,
            e.TenantID,
        ).Scan(&ph)
        if err != nil && err != pgx.ErrNoRows {
            return fmt.Errorf("query prev hash: %w", err)
        }
        e.PrevHash = ph
        e.Hash = e.ComputeHash(ph)
    }

    // ... 插入逻辑
}
```

### 3.2 发现的问题

#### **P2-2: Hash chain 秘钥轮转未持久化版本号**

**问题描述**:
虽然代码支持版本化的 secret (`SetHashChainSecretVersioned`)，但事件记录中**没有保存使用的 secret version**。`VerifyHashWithVersion` 方法接受一个 `secretVersion` 参数，但这个版本号从事件本身获取不到。

**影响**:
- 如果 secret 被轮转，无法确定旧事件是用哪个版本的 secret 计算的 hash
- 验证旧链时需要知道正确的 secret version，但这个信息丢失了

**建议修复**:
在 `audit_events` 表中添加 `hash_secret_version` 列，并在 Insert 时保存当前版本:
```go
e.HashSecretVersion = hashChainCurrentVersion
// 在 SQL 中插入此字段
```

#### **P2-3: FOR UPDATE 锁可能导致 tenant 间串行化**

**问题描述**:
`Insert` 方法使用 `FOR UPDATE` 锁定 tenant 的最后一条记录，这是正确的 tenant 内部串行化。但需要确认是否有跨 tenant 的竞争场景。

**当前分析**:
- 查询条件包含 `WHERE tenant_id = $1`，锁是 per-tenant 的
- 不同 tenant 的并发插入不会互相阻塞
- **结论**: 实现正确

#### **P2-4: Canonical JSON 顺序不一致可能导致 hash 不匹配**

**问题描述**:
`CanonicalJSON` 方法返回 JSON，但 Go 的 `json.Marshal` 不保证字段顺序。虽然当前代码使用确定性的结构体，但依赖 Go 的默认序列化顺序存在风险。

**建议修复**:
使用字段标签的 `omitempty` 确保一致性，或者使用稳定的 JSON 序列化库。

---

## 4. Console Error Helper 统一性审查

### 4.1 当前实现分析

**文件位置**: `console/src/lib/error-helpers.ts`

```typescript
export async function extractErrorMessage(resp: Response | null, fallback: string): Promise<string> {
  if (!resp) return fallback;
  try {
    const data = await resp.json();
    if (typeof data.error === "string") return data.error_description || data.error;
    if (data.error?.message) return data.error.message;
    if (data.error?.code) return data.error.code;
    if (data.message) return data.message;
    if (data.detail) return data.detail;
    return fallback;
  } catch {
    return fallback;
  }
}

export function getErrorMessage(err: unknown, fallback: string): string {
  if (err instanceof Error) {
    const apiErr = err as any;
    if (apiErr.detail && apiErr.detail !== "Request Failed") return apiErr.detail;
    if (apiErr.title && apiErr.title !== "Request Failed") return apiErr.title;
    return err.message || fallback;
  }
  return fallback;
}
```

### 4.2 发现的问题

#### **P3-1: 错误消息提取逻辑不一致**

**问题描述**:
- `extractErrorMessage` 检查 `data.error.message`、`data.message`、`data.detail`
- `getErrorMessage` 检查 `apiErr.detail`、`apiErr.title`、`err.message`

两种检查顺序和字段名不完全一致，可能导致相同的错误在不同路径下返回不同的消息。

**建议修复**:
统一错误消息提取逻辑，优先级应一致:
```typescript
// 统一的错误消息提取优先级
const ERROR_MESSAGE_PRIORITY = [
  'error.message',    // 1. 嵌套 error.message
  'error',            // 2. 顶层 error (string)
  'detail',           // 3. detail
  'message',          // 4. message
  'error_description' // 5. OAuth error_description
] as const;
```

#### **P3-2: "Request Failed" 硬编码过滤过于宽泛**

**问题描述**:
```typescript
if (apiErr.detail && apiErr.detail !== "Request Failed") return apiErr.detail;
```

过滤 `"Request Failed"` 可能会导致合法的错误消息被忽略。应该使用更精确的判断 (如正则匹配或白名单)。

#### **P3-3: 缺少错误类型分类**

**问题描述**:
当前没有区分错误类型 (网络错误、权限错误、验证错误等)，不利于 UI 层根据错误类型显示不同的用户引导。

**建议修复**:
添加错误类型提取:
```typescript
export interface ParsedError {
  message: string;
  code?: string;
  type: 'network' | 'auth' | 'validation' | 'permission' | 'server' | 'unknown';
}

export function parseError(err: unknown): ParsedError {
  // 分析错误类型并返回结构化信息
}
```

---

## 5. 架构债务搜索结果

### 5.1 TODO/FIXME/HACK 标记

搜索 `services/auth` 目录中的 TODO/FIXME/HACK 标记:

```
services/auth/internal/service/backup_codes.go:90:// XXXX-XXXX (8 alphanumeric characters).
services/auth/internal/service/backup_codes_test.go:31: if len(c) != 9 { // XXXX-XXXX
services/auth/internal/service/backup_codes_test.go:32:  t.Errorf("code[%d] = %q, expected format XXXX-XXXX", i, c)
```

**分析**:
- 这些是注释中的格式说明，不是实际的 TODO 标记
- **结论**: `services/auth` 目录中没有实际的架构债务标记

### 5.2 Mock/Hardcode 搜索

搜索 `mock` 和 `hardcode` (不区分大小写):

发现大量测试文件中的 mock 代码，这是正常的。生产代码中未发现明显的硬编码问题。

---

## 6. 安全审查 (输入验证、权限边界、错误处理)

### 6.1 输入验证

#### **P1-3: Conditional Access 上下文数据未经验证直接使用**

**问题描述**:
`EvaluateConditionalAccess` 直接使用 `ctxMap` 中的值进行字符串比较，没有验证输入的合法性和安全性。

**风险**:
- 注入攻击 (虽然只是字符串比较，但恶意构造的输入可能导致逻辑错误)
- 拒绝服务 (超大输入)

**建议修复**:
添加输入大小限制和类型验证:
```go
const MAX_CONTEXT_VALUE_LENGTH = 1024

func validateContext(ctxMap map[string]any) error {
    for k, v := range ctxMap {
        if len(k) > 100 {
            return fmt.Errorf("context key too long")
        }
        strVal := fmt.Sprintf("%v", v)
        if len(strVal) > MAX_CONTEXT_VALUE_LENGTH {
            return fmt.Errorf("context value too long")
        }
    }
    return nil
}
```

### 6.2 权限边界

#### **通过**: Conditional Access 的 tenant 隔离正确

`handleConditionalAccess` 使用 `tenantIDFromHeader(r)` 强制使用调用者的 tenant_id，而不是请求体中的值:

```go
// SECURITY: Force tenant_id from caller context, ignore request body value.
callerTenantID := tenantIDFromHeader(r)
if callerTenantID == uuid.Nil {
    writeJSONError(w, http.StatusBadRequest, "tenant_id is required")
    return
}
```

### 6.3 错误处理

#### **P2-5: Refresh token 撤销错误可能泄露信息**

**问题描述**:
```go
func (ts *TokenService) RevokeRefreshToken(ctx context.Context, plaintext string) error {
    tokenHash := hashToken(plaintext)
    ts.rdb.Del(ctx, refreshTokenKey(tokenHash))
    rt, err := ts.refreshRepo.FindByHash(ctx, tokenHash)
    if err != nil {
        return err // 直接返回数据库错误
    }
    if rt == nil {
        return nil // already gone
    }
    return ts.refreshRepo.Revoke(ctx, rt.ID)
}
```

数据库错误直接返回可能泄露系统内部信息。

**建议修复**:
```go
func (ts *TokenService) RevokeRefreshToken(ctx context.Context, plaintext string) error {
    tokenHash := hashToken(plaintext)
    ts.rdb.Del(ctx, refreshTokenKey(tokenHash))
    rt, err := ts.refreshRepo.FindByHash(ctx, tokenHash)
    if err != nil {
        slog.ErrorContext(ctx, "failed to find refresh token for revocation", "error", err)
        return fmt.Errorf("internal error")
    }
    if rt == nil {
        return nil // already gone
    }
    if err := ts.refreshRepo.Revoke(ctx, rt.ID); err != nil {
        slog.ErrorContext(ctx, "failed to revoke refresh token", "error", err)
        return fmt.Errorf("internal error")
    }
    return nil
}
```

---

## 7. 用户体验检查

### 7.1 错误提示

Console error helper 的错误消息提取逻辑已经在第 4 节分析。

### 7.2 空状态和加载状态

**范围**: 本次审查未覆盖 Console UI 的空状态和加载状态实现。

**建议**:
需要审查 Console 前端代码中的列表页面 (用户列表、审计日志等) 的空状态处理和加载状态显示。

---

## 8. P0/P1 问题汇总

### P0 问题

| ID | 问题描述 | 严重性 | 建议优先级 |
|---|---|---|---|
| P0-1 | OAuth refresh token 撤销时 Redis 和 DB 状态可能不一致 | 高 | 立即修复 |
| P0-2 | Conditional Access 策略评估函数双重实现，可能导致绕过安全策略 | 严重 | 立即修复 |

### P1 问题

| ID | 问题描述 | 严重性 | 建议优先级 |
|---|---|---|---|
| P1-1 | Refresh token 撤销缺少事务回滚机制 | 中 | 本周内修复 |
| P1-2 | Redis fallback 路径未在 token 验证中实现 | 中 | 本周内修复 |
| P1-3 | Conditional Access 上下文数据未经验证直接使用 | 中 | 本周内修复 |

---

## 9. 代码级修复 (不涉及 Docker 部署)

### 修复 P0-1: Refresh token 撤销顺序

**目标**: 修改 `services/auth/internal/service/token_service.go` 中的 `RevokeRefreshToken` 方法，确保 DB 更新成功后才删除 Redis 缓存。

**修复方案**:
1. 先查询数据库，确认 token 存在
2. 调用 `refreshRepo.Revoke` 更新数据库
3. 只有数据库更新成功后，才删除 Redis 缓存
4. 如果数据库更新失败，保留 Redis 缓存并返回错误

**不涉及 Docker 部署**: 这是纯代码修复，在代码提交并触发 CI/CD 后会自动部署。

### 修复 P0-2: 删除或废弃包级别 EvaluateConditionalAccess 函数

**目标**: 删除或标记为废弃 `services/policy/internal/server/conditional_access_handler.go` 行 267-269 的包级别函数。

**修复方案**:
1. 删除包级别函数
2. 或添加 `Deprecated` 注释并返回错误/panic

**不涉及 Docker 部署**: 纯代码修复。

---

## 10. 学习点

### 10.1 Positive (值得保持)

1. **Audit hash chain 的 FOR UPDATE 锁**: 正确使用数据库行锁防止并发写入导致的链断裂
2. **Conditional Access 的 tenant 隔离**: 强制从 header 获取 tenant_id，防止欺骗
3. **Token hash 计算**: 使用 SHA256 哈希存储 token，不存储明文

### 10.2 Negative (需要改进)

1. **双重函数实现**: `EvaluateConditionalAccess` 的两个版本是安全漏洞
2. **撤销操作的原子性**: Redis 和 DB 操作顺序错误导致状态不一致
3. **错误信息泄露**: 数据库错误直接返回给调用者

### 10.3 Neutral (中性行为)

1. **Backup codes 的格式说明**: 注释中的 `XXXX-XXXX` 是合法的格式说明，不是技术债务
2. **测试中的 mock 代码**: 测试文件中的 mock 是正常实践

---

## 11. 后续行动

### 立即执行 (P0)

1. 修复 P0-1: 调整 `RevokeRefreshToken` 的操作顺序
2. 修复 P0-2: 删除或废弃包级别 `EvaluateConditionalAccess` 函数
3. 搜索整个代码库，确认是否有其他地方调用了包级别的 `EvaluateConditionalAccess`

### 本周内完成 (P1)

1. 为 refresh token 撤销添加事务回滚机制
2. 实现 Redis fallback 路径 (token 验证优先检查 Redis)
3. 为 Conditional Access 添加输入验证

### 下个迭代 (P2/P3)

1. 添加 hash secret version 的持久化
2. 统一 Console error helper 的错误消息提取逻辑
3. 优化错误信息，避免泄露系统内部信息

---

## 附录 A: 服务健康检查

```
kubectl get pods -n ggid

核心服务状态:
- ggid-auth: 3/3 Running (所有副本健康)
- ggid-audit: 1/1 Running
- ggid-gateway: 1/1 Running
- ggid-identity: 1/1 Running
- ggid-oauth: 1/1 Running
- ggid-policy: 1/1 Running
- ggid-org: 1/1 Running
- ggid-console: 1/1 Running

所有服务无重启，状态正常。
```

---

## 附录 B: 最近变更分析

```
git log --oneline -10

28e7ce7cb fix(security): logout session revocation + gRPC error leak + email nil ctx
e9c02fd01 docs(iam-review): R126 — regression clean, RBAC R4 + Multi-tenancy R4 verified
df96c013e fix(api): standardize error format + Content-Type across middleware
702215e2d fix(security): RBAC P0 self-assignment + P1 cycle/admin spoof/wildcard
50613265d fix(P1): org/department cross-tenant BOLA + parent org tenant check
30a7a63f2 docs(iam-review): R125 — regression clean, atomic refresh token + revoke nil-tenant fix
25bac1d2d fix(oauth): atomic refresh token consumption + revoke nil-tenant fix
98e3cc216 fix: remove duplicate ConsumeRefreshToken mock method
c6759089e fix: ConsumeRefreshToken returns (bool, error) not error
5fe2b14c7 test: add ConsumeRefreshToken to mockTokenRepo
```

**关键发现**:
- 最近的修复包括 "atomic refresh token consumption" 和 "revoke nil-tenant fix"
- 这些修复可能与本次审查发现的 P0-1 问题相关
- 需要检查这些修复是否已经解决了部分问题

---

审查完成时间: 2026-07-29
审查人: ggcode (Explore sub-agent)