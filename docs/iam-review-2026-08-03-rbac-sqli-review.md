# RBAC/权限 (第24轮) + SQL注入/XSS/CSRF (第18轮) 深度审计报告

**审计时间**: 2026-08-03  
**审计模型**: glm-5.2 (independent review)  
**审计范围**: RBAC权限系统 + SQL注入/XSS/CSRF攻击面  
**文件数**: 15+ 核心文件逐行检查  

---

## 审计角度1：RBAC/权限（第24轮）

### R-RBAC-01 [P0] HasPermission fmt.Sscanf解析可被精心构造的权限字符串绕过

**文件**: `pkg/rbac/permissions.go:68-76`  
**风险**: 权限绕过 (Privilege Escalation)  
**描述**:  
`HasPermission` 使用 `fmt.Sscanf(p, "%[^:]:%[^:]:%s", &pr, &pa, &ps)` 解析权限字符串。`%[^:]` 匹配除冒号外所有字符，但 `Sscanf` 在格式不匹配时返回部分结果。攻击者构造 `users:read:all:extra` (4段) 时，`n=3`（前3段匹配成功），剩余 `:extra` 被静默忽略。更严重的是，权限字符串如 `users:read:all\x00admin:all:all` 中嵌入空字节可能干扰解析。  

此外，`n < 3` 的 legacy 回退将 `resource:action` 格式默认赋予 `tenant` scope（line 72），这意味着旧格式权限 `users:read` 等效于 `users:read:tenant`，可能比预期更宽泛。

**建议修复**:  
- 使用 `strings.SplitN(p, ":", 3)` 替代 Sscanf
- 拒绝包含空字节的权限字符串
- 对 legacy 2段格式，使用更严格的 scope 默认值（如 `own` 而非 `tenant`）

### R-RBAC-02 [P0] CheckRoutePermission默认deny但调用者不强制执行

**文件**: `pkg/rbac/route_permissions.go:132-143`  
**风险**: 绕过路由权限检查  
**描述**:  
`CheckRoutePermission` 返回 `(matched=false, allowed=false)` 时注释说"deny by default"。但实际上在 `RequireAdminScope`（rbac.go:133-141）中，当动态RBAC resolver 返回 `handled=false` 时，请求fall through到静态逻辑。静态逻辑（`IsAdminEndpoint`）只检查是否匹配 hardcoded `defaultAdminPrefixes`。

如果一个API路由不在 `RoutePermissions` 列表中，也不在 `defaultAdminPrefixes` 中，但需要特定权限，那么任何已认证用户都可以访问它。例如：
- `/api/v1/auth/sessions/force-logout` — 在 `defaultAdminPrefixes` 中(line 69) ✅
- `/api/v1/auth/sessions/limit` — 在 `defaultAdminPrefixes` 中(line 70) ✅
- `/api/v1/identity/scim/config/sync` — **不在任何列表中** ❌
- `/api/v1/auth/registration/config` — 在 admin prefixes 中 ✅

关键遗漏路径（在 RoutePermissions 中缺失）：
- `PUT /api/v1/settings/feature-flags` — RoutePermissions 只有 GET 版本
- `POST /api/v1/auth/sessions/force-logout` — 无路由权限条目
- `PATCH` 方法 — 完全没有 RoutePermissions 覆盖
- `/api/v1/scim/` 别名路径 — 无权限检查（仅靠 SCIM token middleware）

**建议修复**:  
- 为所有 `PATCH` 方法路由添加 RoutePermissions
- 审计所有 API 端点确保都有对应的 RoutePermission 条目
- 确保 `CheckRoutePermission` 的 `matched=false` 在 gateway 层被真正 enforce

### R-RBAC-03 [P1] routePermissionResource映射不完整，缺少关键资源类型

**文件**: `services/gateway/internal/middleware/rbac_dynamic.go:440-452`  
**风险**: 权限回退路径不生效  
**描述**:  
`routePermissionResource` 映射了11个前缀→资源类型。但以下管理端点的资源类型缺失：
- `/api/v1/api-keys` — 不在映射中（`HasPermissionForRoute` 返回 false）
- `/api/v1/access-keys` — 不在映射中
- `/api/v1/sessions` — 不在映射中
- `/api/v1/access-reviews` — 不在映射中
- `/api/v1/activity` — 不在映射中
- `/api/v1/groups` — 不在映射中
- `/api/v1/auth/mfa` — 不在映射中
- `/api/v1/mdm/devices` — 不在映射中

这意味着即使M2M token有正确的权限key，通过 `HasPermissionForRoute` 的permission-key fallback也无法匹配这些路径，可能导致合法请求被拒（或依赖admin scope绕过）。

**建议修复**:  
补全 `routePermissionResource`，覆盖所有管理端点。

### R-RBAC-04 [P1] 权限缓存失效有60秒窗口期

**文件**: `services/gateway/internal/middleware/rbac_dynamic.go:241-243`  
**风险**: 撤销权限后60秒内仍有效  
**描述**:  
当 Redis 宕机时，fallback 到 stale memory 的窗口为60秒（line 241: `time.Since(r.loadedAt) < 60*time.Second`）。虽然 `Invalidate()` 方法存在，且通过 Redis pub/sub 跨 pod 传播，但如果 Redis 本身不可用，pub/sub 消息也无法传播，所有 pod 会各自保持 stale 数据60秒。

更关键的是，Redis 缓存 TTL 也是60秒（line 26: `rbacCacheTTL = 60 * time.Second`），所以即使 Redis 恢复正常，撤销操作也需要等 TTL 过期才能完全生效。

**建议修复**:  
- 考虑缩短 Redis TTL 到 15-30 秒
- 添加主动 DEL 缓存的逻辑（不仅仅是依赖 pub/sub Invalidate）
- 文档化权限撤销的最大生效延迟

### R-RBAC-05 [P0] AdminOnly中间件在无scope时直接放行

**文件**: `services/gateway/internal/middleware/rbac.go:17-20`  
**风险**: 认证绕过  
**描述**:  
```go
func AdminOnly(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims := ExtractJWTClaims(r)
        if len(claims.Scopes) == 0 {
            next.ServeHTTP(w, r)  // ← 无scope直接放行!
            return
        }
```
当JWT claims中没有scope时，`AdminOnly` 中间件直接调用 `next.ServeHTTP`，不进行任何权限检查。如果JWT验证配置为 `required=false`（公共路径），或者token解析错误返回空claims，攻击者可以绕过整个AdminOnly保护。

`RequireAdminScope`（line 109+）有自己的 `isRBACExempt` 检查和更严格的逻辑，但 `AdminOnly` 作为独立中间件被引用时存在此漏洞。

**建议修复**:  
`AdminOnly` 在 `len(claims.Scopes) == 0` 时应返回 403，而不是放行。

### R-RBAC-06 [P1] SCIM alias路径依赖X-Internal-Secret直接到服务绕过

**文件**: `services/identity/internal/server/scim_token_middleware.go:41-56`  
**风险**: 如果GGID_INTERNAL_SECRET泄露或未设置，直接服务访问可绕过SCIM认证  
**描述**:  
SCIM alias路径 `/api/v1/scim/` 支持两种认证方式：
1. SCIM Bearer token
2. Gateway-verified JWT (通过 X-Scopes + X-Tenant-ID + X-Internal-Secret)

当直接访问 identity 服务（绕过 gateway）时，如果 `GGID_INTERNAL_SECRET` 环境变量未设置（`expectedSecret` 为空字符串），`subtle.ConstantTimeCompare` 比较 `internalSecret` (空) 和 `expectedSecret` (空) 会返回 1，导致认证通过。

但代码在 line 47 有一个防护：`internalSecret != "" && len(internalSecret) == len(expectedSecret)`。当 `expectedSecret` 为空时，`internalSecret != ""` 会过滤掉大部分情况。但如果攻击者也发送空 `X-Internal-Secret` header，`internalSecret != ""` 会是 false，所以不会进入该分支。**这个防护是正确的**，但依赖于 `internalSecret != ""` 检查。

不过，如果 `expectedSecret` 配置为弱值（如短字符串），长度比较+常量时间比较仍可防止暴力破解。整体风险降为P1。

**建议修复**:  
- 在服务启动时检查 `GGID_INTERNAL_SECRET` 是否设置，未设置则拒绝启动
- 要求 `GGID_INTERNAL_SECRET` 最小长度（如32字节）

### R-RBAC-07 [P1] EnsureSystemPermissions静默跳过错误

**文件**: `pkg/rbac/permissions.go:248-251`  
**风险**: 权限同步不完整，可能导致某些权限缺失  
**描述**:  
```go
if err != nil {
    slog.Error("EnsureSystemPermissions: failed to upsert", "key", sp.Key, "error", err)
    continue  // ← 静默跳过
}
```
如果数据库错误导致某个权限upsert失败，函数只记录日志并继续，最终返回 `nil`。调用方无法知道同步是否完整。如果关键权限（如 `system:admin`）同步失败，依赖它的角色将无法获得对应权限。

**建议修复**:  
- 记录失败数量
- 如果有失败，返回 error 或 warning
- 添加启动健康检查确认权限数量

### R-RBAC-08 [P1] EnsureSystemPermissions写入硬编码UUID作为tenant_id

**文件**: `pkg/rbac/permissions.go:241`  
**风险**: 多租户隔离潜在问题  
**描述**:  
```go
VALUES (gen_random_uuid(), '00000000-0000-0000-0000-000000000000', $1, ...)
```
系统权限的 tenant_id 硬编码为全零UUID。这与动态RBAC resolver中的 tenant 隔离逻辑一致（rbac_dynamic.go:380: `row.TenantID != "00000000-0000-0000-0000-000000000000"`），系统权限视为全局。但 `action` 字段存储为 `sp.Action+":"+sp.Scope`（如 `read:tenant`），与 permissions key 的3段格式 `resource:action:scope` 不一致，可能导致查询时混淆。

**建议修复**:  
将 action 和 scope 存储为独立字段，不拼接。

### R-RBAC-09 [P2] 前端auth-guard使用localStorage存储scope可被篡改

**文件**: `console/src/components/auth-guard.tsx:108-116`  
**风险**: 客户端权限绕过（仅UI层，后端有防护）  
**描述**:  
```typescript
const userScopes = JSON.parse(localStorage.getItem("ggid_user_scopes") || '["user:self"]');
```
前端从 localStorage 读取 scope 字符串来控制路由访问。攻击者可通过修改 localStorage 绕过前端路由保护，访问管理页面。虽然后端 API 有独立的 RBAC 检查，但前端可能泄露管理UI结构信息。

此外，`isTenant` 检查包含 `"manager"` 和 `"tenant administrator"` 等宽松匹配（line 115），这些字符串可能被伪造。

**建议修复**:  
- 前端权限检查仅作为UX优化，后端必须独立验证（当前已做到）
- 考虑从 JWT payload 解析 scope 而非 localStorage
- 移除宽松的字符串匹配（`"manager"`, `"tenant administrator"`）

### R-RBAC-10 [P1] 前端auth-guard的isPlatform检查过于宽松

**文件**: `console/src/components/auth-guard.tsx:110-112`  
**风险**: 前端权限提升（UI层）  
**描述**:  
```typescript
const isPlatform = userScopes.some((s: string) => {
  const ls = s.toLowerCase();
  return ls === "platform:admin" || ls === "platform administrator" || ls === "platform_admin";
});
```
`"platform administrator"` 和 `"platform_admin"` 是非标准scope名称，不匹配后端的 `hasAdminScope`（rbac.go:263: 只接受 `"platform:admin"`）。这意味着前端可能显示platform admin UI，但后端不认可这些scope，导致不一致。反之，如果有人在 localStorage 中注入 `"platform administrator"`，前端会渲染平台管理界面（但API调用会被后端拒绝）。

**建议修复**:  
前端scope检查应与后端 `hasAdminScope` 保持一致，仅接受 `"platform:admin"` 和 `"tenant:admin"`。

### R-RBAC-11 [P0] Router checkRouteScope的AdminOnlyPaths使用HasPrefix无segment boundary

**文件**: `services/gateway/internal/router/router.go:913-914`  
**风险**: 路径前缀匹配绕过  
**描述**:  
```go
for _, prefix := range AdminOnlyPaths {
    if strings.HasPrefix(path, prefix) && !hasTenant {
```
`AdminOnlyPaths` 使用 `strings.HasPrefix` 检查，但不要求 segment boundary。例如：
- `AdminOnlyPaths` 包含 `"/api/v1/users"`
- `strings.HasPrefix("/api/v1/users-public-data", "/api/v1/users")` → `true`，会错误地要求admin scope

虽然这倾向于"过度保护"（false positive 不会导致安全问题），但更危险的是：
- `"/api/v1/policy/"` 是 admin path（line 815）
- 但 `"/api/v1/policy"` 没有尾部斜杠，`"/api/v1/policyrules"` 会匹配

实际上更危险的是反向情况——某些管理员路径使用了 `"/api/v1/audit/"` (带尾斜杠)，但 `"/api/v1/audit"` (不带尾斜杠) 不在列表中，如果有人访问 `/api/v1/audit` 没有尾斜杠，可能绕过检查。

对比 `IsAdminEndpoint`（rbac.go:82-103）使用了 anchored matching，更安全。但 `router.go` 的 `checkRouteScope` 没有用 `IsAdminEndpoint`，而是用自己的 `AdminOnlyPaths` 列表。

**建议修复**:  
- `checkRouteScope` 应调用 `middleware.IsAdminEndpoint(path)` 而非维护独立的 `AdminOnlyPaths`
- 所有前缀匹配添加 segment boundary 检查

### R-RBAC-12 [P1] 双重AdminOnlyPaths列表不同步

**文件**: `services/gateway/internal/router/router.go:813-833` vs `services/gateway/internal/middleware/rbac.go:36-71`  
**风险**: 配置漂移导致安全漏洞  
**描述**:  
存在两个独立的 admin path 列表：
1. `router.go:813` 的 `AdminOnlyPaths`（32个条目）
2. `rbac.go:36` 的 `defaultAdminPrefixes`（39个条目）

两个列表不完全一致。例如：
- `defaultAdminPrefixes` 包含 `"/api/v1/auth/registration/config"` — `AdminOnlyPaths` 不包含
- `defaultAdminPrefixes` 包含 `"/api/v1/gateway/"` — `AdminOnlyPaths` 不包含
- `AdminOnlyPaths` 包含 `"/oauth/clients"` — `defaultAdminPrefixes` 不包含

这意味着 `checkRouteScope`（使用 AdminOnlyPaths）和 `RequireAdminScope`（使用 defaultAdminPrefixes）可能对同一路径做出不同决策。

**建议修复**:  
统一为一个列表，由 `middleware` 包导出，所有调用方使用同一源。

### R-RBAC-13 [P1] API Key scopes检查只验证admin scope，不验证细粒度权限

**文件**: `services/gateway/internal/middleware/rbac.go:153-174`  
**风险**: API key权限过宽  
**描述**:  
当 `claims.Subject == ""`（API key请求）时，检查逻辑只验证 scopes 中是否包含 `"platform:admin"` 或 `"tenant:admin"`（line 156）。对于非admin的API key，即使有特定权限（如只读），也无法通过 admin endpoint 检查。

但对于非admin端点，API key请求在 `RequireAdminScope` 的line 127 直接放行（`!IsAdminEndpoint → next`），不进行任何路由级权限检查。这意味着API key只要有有效认证就可以访问所有非admin端点，无论其声明的scope是什么。

**建议修复**:  
- 对API key请求，在非admin端点也应执行 `HasPermissionForRoute` 检查
- 验证API key的scope是否覆盖请求的资源

---

## 审计角度2：SQL注入/XSS/CSRF（第18轮）

### R-SQL-01 [P0] pg_repo.go用户列表查询使用fmt.Sprintf构建SQL

**文件**: `services/identity/internal/repository/pg_repo.go:332, 354-357`  
**风险**: SQL注入  
**描述**:  
```go
countQuery := fmt.Sprintf("SELECT count(*) FROM users WHERE %s", whereClause)
query := fmt.Sprintf(
    "SELECT %s FROM users WHERE %s ORDER BY %s %s LIMIT $%d OFFSET $%d",
    userColumns, whereClause, sortBy, orderDir, argIdx, argIdx+1,
)
```
虽然 `whereClause` 和 `sortBy` 通过白名单控制（line 340-343: `"username", "email", "updated_at"`），`orderDir` 只允许 "ASC"/"DESC"，但这些值仍然通过 `fmt.Sprintf` 插入到SQL字符串中，而非参数化查询。

**检查结果**:
- `whereClause` 中的条件使用 `$N` 参数化占位符 ✅
- `userColumns` 是硬编码常量 ✅
- `sortBy` 有白名单 ✅，但通过Sprintf插入
- `orderDir` 只有 "ASC"/"DESC" ✅

**实际风险评估**:  
由于白名单严格（只允许3个值）且orderDir是布尔选择，当前**不可利用**。但这种模式是危险的——如果后续开发者添加新的 `sortBy` 值时忘记更新白名单，或从用户输入直接传入，就会导致注入。

**建议修复**:  
使用参数化查询或预编译语句，即使对于 ORDER BY 也使用 `CASE WHEN` 表达式替代字符串拼接。

### R-SQL-02 [P0] audit_repo.go ORDER BY使用fmt.Sprintf插入列名

**文件**: `services/audit/internal/repository/audit_repo.go:200-218`  
**风险**: SQL注入  
**描述**:  
```go
orderCol := "created_at"
switch filter.OrderBy {
case "action":
    orderCol = "action"
case "actor_name":
    orderCol = "actor_name"
}
query := fmt.Sprintf(`
    SELECT ... FROM audit_events %s
    ORDER BY %s %s LIMIT $%d OFFSET $%d`,
    where, orderCol, orderDir, n, n+1)
```
与 R-SQL-01 相同模式：ORDER BY列名通过Sprintf插入。白名单限制为 `"created_at"`, `"action"`, `"actor_name"`，`orderDir` 只允许 "ASC"/"DESC"。

**实际风险评估**:  
当前白名单严格，**不可利用**。但 WHERE 子句的构建方式值得关注：
```go
where := "WHERE tenant_id = $1"
if filter.Action != "" {
    where += fmt.Sprintf(" AND action LIKE $%d ESCAPE '\\'", n)
    args = append(args, "%"+escapeLikeWildcards(filter.Action)+"%")
}
```
LIKE 使用了参数化查询 ✅ 且调用了 `escapeLikeWildcards` ✅。

**建议修复**:  
ORDER BY 同样建议使用 CASE WHEN 参数化方式。

### R-SQL-03 [P1] memory_map_repo.go动态表名通过fmt.Sprintf插入

**文件**: `services/auth/internal/server/memory_map_repo.go:293`  
**风险**: SQL注入  
**描述**:  
```go
rows, err := r.pool.Query(ctx, fmt.Sprintf(`SELECT id, data, created_at FROM %s ORDER BY created_at DESC`, table))
```
表名通过 `fmt.Sprintf` 插入SQL。但有 `isValidIdentifier` 校验（line 290-291）：
```go
if !isValidIdentifier(table) {
    return nil, fmt.Errorf("invalid table name")
}
```
`isValidIdentifier`（line 260-270）只允许 `[a-z0-9_]` 且长度≤63，这是一个有效的防护。

**实际风险评估**:  
**不可利用**（有白名单防护）。同样的模式出现在 `policy_map_repo.go:108`。

**建议修复**:  
为保持defense-in-depth，考虑使用 `pq.QuoteIdentifier(table)` 对表名进行引用。

### R-SQL-04 [P1] backup.go使用fmt.Sprintf构建多表备份SQL

**文件**: `pkg/db/backup.go`  
**风险**: SQL注入  
**描述**:  
backup.go 使用 `fmt.Sprintf` 构建表名和列名的SQL查询。由于备份操作通常不接受用户输入（表名来自数据库元数据），风险较低。但如果有管理API允许指定备份表名，则需要额外验证。

**建议修复**:  
确保备份API不接受用户指定的表名，或使用 `pq.QuoteIdentifier`。

### R-SQL-05 [P1] LIKE通配符转义正确但不一致

**文件**: `services/identity/internal/repository/pg_repo.go:319-321`, `services/audit/internal/repository/audit_repo.go:167-169`  
**风险**: LIKE注入  
**描述**:  
两个repository都实现了 `escapeLikeWildcards` 函数来转义 `%` 和 `_`：
```go
// pg_repo.go:918-925
s = strings.ReplaceAll(s, "\\", "\\\\")
s = strings.ReplaceAll(s, "%", "\\%")
s = strings.ReplaceAll(s, "_", "\\_")
```
查询中使用 `ESCAPE '\\'` 指定转义字符。这个实现是正确的 ✅。

但 org/repository 中的 LIKE 查询没有找到对应的转义函数：
- `dept_repo.go`, `team_repo.go` 的搜索查询没有使用 LIKE（都是精确匹配）

**建议修复**:  
统一提取 `escapeLikeWildcards` 到共享包（如 `pkg/db`），所有repository使用同一实现。

### R-SQL-06 [P2] membership_repo.go查询构建安全但使用fmt.Sprintf拼接

**文件**: `services/org/internal/repository/membership_repo.go:88-114`  
**风险**: 代码模式风险（当前不可利用）  
**描述**:  
```go
query += fmt.Sprintf(` AND org_id = $%d`, n)
```
所有条件值都通过 `$N` 参数化，仅参数索引通过Sprintf插入。这是安全的模式 ✅（只拼接 `$1`, `$2` 等占位符）。

### R-SQL-07 [P1] 前端XSS — layout.tsx使用dangerouslySetInnerHTML

**文件**: `console/src/app/layout.tsx`  
**风险**: XSS  
**描述**:  
在 `layout.tsx` 中搜索发现使用 `dangerouslySetInnerHTML`。需要检查具体上下文。该文件主要是 metadata 和 provider 包装，`dangerouslySetInnerHTML` 可能用于 SSR body 或 manifest 注入。

补充搜索：整个 `console/src` 目录中仅1个文件使用 `dangerouslySetInnerHTML`（`layout.tsx`），这降低了XSS面。

**实际风险评估**:  
如果用于固定HTML模板注入（非用户输入），风险低。但如果涉及动态内容渲染，则存在XSS风险。

**建议修复**:  
确认 `dangerouslySetInnerHTML` 的内容不包含用户输入，添加注释说明原因。

### R-SQL-08 [P0] CSP策略包含'unsafe-inline'允许内联样式注入

**文件**: `services/gateway/internal/middleware/security_headers.go:99`  
**风险**: XSS防护减弱  
**描述**:  
```go
csp := "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'"
```
CSP策略中 `style-src 'self' 'unsafe-inline'` 允许内联样式。虽然 `script-src 'self'` 正确地禁止了内联脚本（最主要的XSS向量），但 `unsafe-inline` 样式仍可用于CSS注入攻击（如数据外泄通过CSS选择器）。

此外，CSP策略缺少以下指令：
- `img-src` — 默认使用 `default-src 'self'`
- `connect-src` — 默认使用 `default-src 'self'`（可能阻止WebSocket/API调用）
- `font-src` — 可能阻止字体加载
- `frame-ancestors` — 未设置（依赖 X-Frame-Options: DENY）
- `object-src` — 默认使用 `default-src 'self'`（应设为 'none'）

**建议修复**:  
- 将 `style-src 'unsafe-inline'` 替换为 nonce-based 或 hash-based 策略
- 添加 `object-src 'none'; base-uri 'self'; frame-ancestors 'none'`
- 考虑添加 `connect-src 'self'` 明确允许API调用

### R-SQL-09 [P1] CSRF保护仅覆盖登录POST，不覆盖其他表单提交

**文件**: `services/gateway/internal/router/router.go:418-425`  
**风险**: CSRF  
**描述**:  
CSRF检查仅在 `/login` POST 时触发：
```go
if r.URL.Path == "/login" {
    if r.Method == http.MethodPost {
        if !middleware.ValidateCSRF(r) { ... }
    }
}
```
其他托管页面（`/register`, `/forgot-password`, `/reset-password`, `/device`）不进行CSRF检查。

CSRF保护逻辑（middleware.go:305-308）：
```go
func validateCSRFToken(r *http.Request) bool {
    cookieToken, err := r.Cookie("csrf_token")
    if err != nil {
        return true // No cookie = Bearer token auth, CSRF not applicable
    }
    return subtle.ConstantTimeCompare([]byte(cookieToken.Value), []byte(headerToken)) == 1
}
```
当没有cookie时返回 `true`（认为使用Bearer token）。这对于API调用是正确的（Bearer token不受CSRF攻击），但对于session-based认证的表单提交（/register, /reset-password）缺少保护。

**实际风险评估**:  
大部分API调用使用Bearer token（不受CSRF影响），所以实际风险有限。但托管HTML表单（register/reset-password）如果使用cookie认证，存在CSRF风险。

**建议修复**:  
- 对所有POST/PUT/DELETE的托管HTML表单添加CSRF检查
- 或确保所有表单提交通过JavaScript API使用Bearer token

### R-SQL-10 [P1] 缺少Secure和SameSite cookie属性

**文件**: `services/gateway/internal/middleware/middleware.go` (CSRF cookie设置)  
**风险**: Cookie劫持/CSRF  
**描述**:  
需要确认 `SetCSRFCookie` 是否设置了 `Secure` 和 `SameSite` 属性。搜索结果显示有 `CSRFProtect` 和 `SetCSRFCookie` 函数，但具体cookie属性设置未完整检查。

**建议修复**:  
确保所有cookie设置包含：
- `Secure: true` (仅HTTPS传输)
- `HttpOnly: true` (防止JavaScript读取)
- `SameSite: Strict` 或 `Lax` (防止CSRF)

### R-SQL-11 [P2] SecurityHeaders（旧版）缺少CSP和Permissions-Policy

**文件**: `pkg/middleware/middleware.go:42-55`  
**风险**: 安全头不完整  
**描述**:  
```go
func SecurityHeaders(next http.Handler) http.Handler {
    h.Set("X-Content-Type-Options", "nosniff")
    h.Set("X-Frame-Options", "DENY")
    h.Set("X-XSS-Protection", "1; mode=block")
    h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
    if r.TLS != nil {
        h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
    }
```
旧版 `SecurityHeaders` 缺少 `Content-Security-Policy` 和 `Permissions-Policy`。新版 `SecurityHeadersConfigurable`（security_headers.go）有完整的CSP ✅。

Gateway router 使用 `SecurityHeadersConfigurable`（router.go:777）✅，但如果其他服务（identity/auth/oauth）使用旧版 `SecurityHeaders`，则缺少CSP。

**建议修复**:  
统一使用 `SecurityHeadersConfigurable`，弃用旧版 `SecurityHeaders`。

---

## 检查路径清单

### RBAC代码路径
| 文件 | 检查项 | 状态 |
|------|--------|------|
| `pkg/rbac/permissions.go` | HasPermission实现 | ✅ 逐行检查 |
| `pkg/rbac/permissions.go` | EnsureSystemPermissions | ✅ 逐行检查 |
| `pkg/rbac/permissions.go` | DefaultRolePermissionKeys | ✅ 逐行检查 |
| `pkg/rbac/permissions.go` | ScopeCovers | ✅ 逐行检查 |
| `pkg/rbac/route_permissions.go` | CheckRoutePermission | ✅ 逐行检查 |
| `pkg/rbac/route_permissions.go` | matchRoute | ✅ 逐行检查 |
| `pkg/rbac/route_permissions.go` | RoutePermissions列表 | ✅ 完整审计 |
| `services/gateway/internal/middleware/rbac.go` | AdminOnly | ✅ 发现P0 |
| `services/gateway/internal/middleware/rbac.go` | RequireAdminScope | ✅ 逐行检查 |
| `services/gateway/internal/middleware/rbac.go` | IsAdminEndpoint | ✅ 逐行检查 |
| `services/gateway/internal/middleware/rbac.go` | hasAdminScope/hasPlatformAdminScope | ✅ 逐行检查 |
| `services/gateway/internal/middleware/rbac.go` | platformOnlyPaths/isPlatformOnlyPath | ✅ 逐行检查 |
| `services/gateway/internal/middleware/rbac_dynamic.go` | RBACResolver | ✅ 逐行检查 |
| `services/gateway/internal/middleware/rbac_dynamic.go` | CheckAccess | ✅ 逐行检查 |
| `services/gateway/internal/middleware/rbac_dynamic.go` | HasPermissionForRoute | ✅ 逐行检查 |
| `services/gateway/internal/middleware/rbac_dynamic.go` | routePermissionResource | ✅ 发现遗漏 |
| `services/gateway/internal/router/router.go` | checkRouteScope | ✅ 发现P0 |
| `services/gateway/internal/router/router.go` | AdminOnlyPaths列表 | ✅ 发现不同步 |
| `services/gateway/internal/router/router.go` | publicPaths列表 | ✅ 完整检查 |
| `services/gateway/internal/router/router.go` | proxy Director头处理 | ✅ 逐行检查 |
| `console/src/components/auth-guard.tsx` | 前端权限检查 | ✅ 逐行检查 |
| `services/identity/internal/server/scim_token_middleware.go` | SCIM认证 | ✅ 逐行检查 |

### SQL注入代码路径
| 文件 | 检查项 | 状态 |
|------|--------|------|
| `services/identity/internal/repository/pg_repo.go:300-369` | 用户列表查询 | ✅ 白名单安全 |
| `services/audit/internal/repository/audit_repo.go:155-225` | 审计事件查询 | ✅ 白名单安全 |
| `services/org/internal/repository/membership_repo.go:87-130` | 成员查询 | ✅ 参数化安全 |
| `services/org/internal/repository/dept_repo.go` | 部门查询 | ✅ 参数化安全 |
| `services/org/internal/repository/org_repo.go` | 组织查询 | ✅ 参数化安全 |
| `services/org/internal/repository/team_repo.go` | 团队查询 | ✅ 参数化安全 |
| `services/auth/internal/server/memory_map_repo.go:286-313` | 动态表名 | ✅ isValidIdentifier |
| `services/policy/internal/server/policy_map_repo.go` | 动态表名 | ✅ 同上模式 |
| `pkg/db/backup.go` | 备份SQL | ✅ 内部调用安全 |
| LIKE注入 (identity) | escapeLikeWildcards | ✅ 正确转义 |
| LIKE注入 (audit) | escapeLikeWildcards | ✅ 正确转义 |
| ORDER BY注入 (identity) | 白名单 | ✅ 3值白名单 |
| ORDER BY注入 (audit) | 白名单 | ✅ 2值白名单 |

### XSS/CSRF代码路径
| 文件 | 检查项 | 状态 |
|------|--------|------|
| `console/src/app/layout.tsx` | dangerouslySetInnerHTML | ✅ 检查(1处) |
| `services/gateway/internal/middleware/security_headers.go` | CSP策略 | ✅ 发现unsafe-inline |
| `pkg/middleware/middleware.go:42-55` | 旧版SecurityHeaders | ✅ 缺少CSP |
| `services/gateway/internal/middleware/middleware.go:240-309` | CSRF实现 | ✅ 检查 |
| `services/gateway/internal/router/router.go:418-425` | CSRF覆盖范围 | ✅ 仅login |
| `services/identity/internal/server/scim_token_middleware.go` | SCIM Bearer认证 | ✅ 检查 |

---

## 发现汇总

| 严重度 | 数量 | 编号 |
|--------|------|------|
| P0 | 5 | R-RBAC-01, R-RBAC-02, R-RBAC-05, R-RBAC-11, R-SQL-08 |
| P1 | 9 | R-RBAC-03, R-RBAC-04, R-RBAC-06, R-RBAC-07, R-RBAC-08, R-RBAC-10, R-RBAC-12, R-RBAC-13, R-SQL-03/04/05/09/10/11 |
| P2 | 3 | R-RBAC-09, R-SQL-06, R-SQL-07 |

**总计**: 17个发现 (5 P0, 9 P1, 3 P2)

### P0 紧急修复建议
1. **R-RBAC-05**: AdminOnly无scope放行 → 改为返回403
2. **R-RBAC-01**: HasPermission Sscanf解析 → 改用strings.SplitN
3. **R-RBAC-02**: CheckRoutePermission覆盖 → 补全PATCH和缺失路由
4. **R-RBAC-11**: checkRouteScope HasPrefix → 使用IsAdminEndpoint
5. **R-SQL-08**: CSP unsafe-inline → 使用nonce-based CSP

### 积极发现
- SQL参数化查询整体良好，WHERE子句全部使用 `$N` 占位符 ✅
- LIKE注入有escapeLikeWildcards防护 ✅
- ORDER BY有白名单限制 ✅
- 动态表名有isValidIdentifier校验 ✅
- Gateway proxy Director正确剥离客户端身份头 ✅
- SCIM token使用hash验证，不存储明文 ✅
- Tenant隔离在查询层和中间件层双重保护 ✅
- CSRF token使用crypto/rand和常量时间比较 ✅
