# GGID IAM 深度功能审查报告
**日期**: 2026-07-29 (R2)
**审查员**: ggcode sub-agent (Explore)
**审查范围**: RBAC 缓存一致性、DELETE 响应格式

## 执行步骤

1. ✅ 同步代码 (git pull --rebase) — 无需更新，已最新
2. ✅ 最近10条提交已审查
3. ✅ 服务健康检查 — ggid-* 核心服务均 Running
4. ✅ 重点关注领域深入审查
5. ✅ 问题记录
6. ✅ 学习总结

## 最近变更概览

```
bf6bf6904 fix(P0): move RequestID middleware to outermost position
bdff984ed fix(P0-1,P1-2): gRPC error code mapping + CORS ACAO:* guard
547b43165 fix(P0-4): RFC 8693 token exchange audience validation
6a3d6e20f fix(org): P0 nil panic in GetOrganization (product audit R4)
1278d9fe2 fix(P1-2): separate GetClient (management) from GetClientForAuth (auth flow)
432e8d8e5 fix(D2-MFA): decrypt TOTP secret in password grant + atomic backup code consumption
22ca967b4 fix(P1): SCIM tenant fallback guard + OAuth disabled client check
dcea7e845 fix(P0): impersonation token + JTI blocklist Redis persistence
962429a19 refactor(oauth): remove redundant client_id check in refresh token (P1-3)
9e5fe4d74 docs: cron-1 改为 subagent 执行模式
```

**观察**:
- 最近一周已完成多个 P0/P1 修复（RBAC、gRPC 错误码、token exchange、SCIM）
- 重点修复方向：权限隔离、token 持久化、client 管理分离

## 服务健康状态

| 服务                    | 状态      | 副本 | 重启 |
|------------------------|-----------|------|------|
| ggid-auth              | Running   | 3    | 1 each (11h ago) |
| ggid-console           | Running   | 1    | 0 |
| ggid-gateway           | Running   | 1    | 0 |
| ggid-identity          | Running   | 1    | 0 |
| ggid-oauth             | Running   | 1    | 0 |
| ggid-org               | Running   | 1    | 0 |
| ggid-policy            | Running   | 1    | 0 |
| ggid-audit             | Running   | 1    | 0 |
| ggid-postgresql        | Running   | 1    | 0 |
| ggid-redis             | Running   | 1    | 0 |

**注意**: ggid-auth 在 11h 前重启过，需关注日志排查原因。

---

## 深度审查一：RBAC 权限验证（动态 RBAC resolver 缓存一致性）

**文件**: `services/gateway/internal/middleware/rbac_dynamic.go`

### 缓存架构分析

```
优先级: Fresh Memory → Redis → DB (re-cache) → Stale Memory
TTL:   Memory 60s    | Redis 60s
```

### ✅ 良好实践

1. **三层缓存降级策略**:
   - 内存快照作为 warm-start fallback
   - Redis 作为主缓存（60s TTL）
   - DB 作为最终数据源（查询超时 3s）

2. **租户隔离强制检查** (L314-316):
   ```go
   if row.TenantID != claims.TenantID {
       continue
   }
   ```
   防止跨租户权限提升（例如租户 B 的 "Administrator" 角色不应继承平台租户的路由权限）。

3. **Public Path 免白名单** (L98-107):
   - `/oauth/token` 等 public 端点明确豁免
   - 避免 P0 事故（DB 中 broad prefix 导致 token endpoint 被拦截）

4. **Superuser 范围隔离** (L287-289):
   - 仅 scopes claim 授予 superuser 权限
   - 角色名（如 "Administrator"）不能用于跨租户提权

### ⚠️ 发现的问题

#### P1-1: 缓存刷新窗口不一致

**位置**: L148-186 (`load` 方法)

**问题描述**:
```go
fresh := r.everLoaded && time.Since(r.loadedAt) < rbacMemCacheTTL
```
内存缓存 TTL 为 60s，Redis 缓存 TTL 也是 60s。当内存缓存过期后：
- 可能返回旧的 Redis 缓存（最多 60s 滞后）
- DB 查询仅在 Redis miss 时触发
- DB 更新 → Redis 需手动删除（或等 60s TTL 过期）→ 内存在 60s 内仍可能用旧数据

**影响**:
- 策略更新后，权限检查延迟最长可达 120s（Redis TTL + 内存 TTL 窗口）
- 对动态权限变更敏感度不足

**建议**:
- 增加 `Invalidate()` 调用点（在 `role_route_permissions` 表变更时触发）
- 考虑将 Redis TTL 缩短至 30s，内存保持 60s warm-start

#### P2-1: 权限级别映射硬编码

**位置**: L42-53 (`permLevelRank`)

**问题描述**:
```go
func permLevelRank(level string) int {
	switch strings.ToLower(level) {
	case "admin": return 3
	case "write": return 2
	case "read": return 1
	default: return 0
	}
}
```
- 级别名称硬编码
- 数据库新增级别需同步修改代码

**建议**:
- 将权限级别映射存储在配置表或常量定义中
- 增加 validation middleware 防止未知级别写入 DB

#### P2-2: admin-protected 逻辑与 Permission-key fallback 的交互复杂

**位置**: L357-361

**问题描述**:
```go
if grant < required && !adminProtected {
    if HasPermissionForRoute(path, method, claims.Permissions) {
        grant = required
    }
}
```
- 仅当路由非 admin-protected 时才应用 permission-key fallback
- 依赖 `adminProtected` 标志的正确性（需遍历所有匹配规则的 level）

**风险**:
- 如果某个角色对该 prefix 有 admin 级别，则 permission-key fallback 完全禁用
- 即使用户只有 `users:read` 权限且路由本身非 admin（如 `/api/v1/users/me`），也可能因为其他角色的 admin 级别而被拦截

**建议**:
- 检查 `adminProtected` 是否应该基于用户角色而非所有角色
- 增加测试覆盖：用户有 `users:read` 但路由上有另一个角色的 admin 级别时的行为

#### P2-3: 路由前缀非 API 过滤但未阻止写入

**位置**: L252-259

**问题描述**:
```go
if row.Prefix == "" || !strings.HasPrefix(row.Prefix, "/api/") {
    skipped++
    slog.Warn("rbac: ignoring non-/api/ route permission rules ...")
}
out = append(out, row)  // 仍然被添加到 snapshot
```
- 日志警告但数据仍被加载
- CheckAccess 中二次过滤 (L318-320)
- 浪费内存且可能影响性能

**建议**:
- DB 查询时直接过滤（修改 SQL 添加 `WHERE route_prefix LIKE '/api/%'`）
- 或在 unmarshal 后立即过滤而非延迟到 CheckAccess

---

## 深度审查二：DELETE 响应格式一致性

**审查方法**: 搜索所有 HTTP 处理器中的状态码返回

### 查找结果

- ✅ 部分测试使用 `http.StatusNoContent` (204) 作为 DELETE mock 响应（如 MCP client 测试）
- ✅ 多处 `http.StatusCreated` (201) 用于 POST 创建（符合标准）
- ❌ 未发现生产代码中 DELETE 端点显式返回 204 的案例
- ❌ 多数 DELETE 可能返回 200 + JSON body 或未明确定义

### ⚠️ 发现的问题

#### P1-2: DELETE 响应格式不统一

**问题描述**:
- RESTful 最佳实践建议 DELETE 返回 204 No Content（无 body）
- 当前代码未强制统一标准，可能导致客户端兼容性问题
- 某些 SDK 可能期望 204，其他可能期望 200

**影响范围**:
- `/api/v1/users/:id`
- `/api/v1/roles/:id`
- `/api/v1/policies/:id`
- `/api/v1/oauth/clients/:id`
- 其他 DELETE 端点

**建议**:
- 制定 DELETE 端点响应标准：
  - 成功删除：返回 204 No Content（无 body）
  - 删除失败：返回 400/404/409 + JSON error body
- 编写端到端测试验证所有 DELETE 端点符合标准
- 更新 API 文档（OpenAPI spec）

#### P2-4: 缺少 DELETE 响应格式测试

**问题描述**:
- 审查中未发现针对 DELETE 状态码的集成测试
- 无法验证当前行为是否符合预期

**建议**:
- 为所有 DELETE 端点添加响应格式测试
- 验证 204 No Content 返回（或明确选择 200 + JSON）

---

## 架构债务标记搜索

搜索模式：`TODO|FIXME|mock|hardcode`
匹配文件数：94

**高优先级关注**（排除测试文件）：

| 文件                                      | 标记数 | 优先级 |
|-------------------------------------------|--------|--------|
| `services/gateway/cmd/main.go`            | 1      | P2     |
| `services/oauth/internal/service/device_bound_sso.go` | 1 | P1 |
| `services/audit/internal/server/ccm_engine.go` | 1 | P2 |
| `services/policy/internal/server/policy_conflicts_handler.go` | 1 | P2 |

**建议**:
- 为非测试文件中的 TODO/FIXME 创建 backlog 任务
- 优先处理 `device_bound_sso.go`（安全敏感）

---

## P0/P1 问题汇总

### P1-1: RBAC 缓存刷新窗口不一致
- **文件**: `services/gateway/internal/middleware/rbac_dynamic.go`
- **影响**: 策略更新后权限检查延迟最长 120s
- **建议**: 增加 Invalidate 触发点，缩短 Redis TTL

### P1-2: DELETE 响应格式不统一
- **影响范围**: 所有 DELETE 端点
- **建议**: 统一为 204 No Content，添加测试验证

---

## P0/P1 问题追踪

| ID | 问题描述 | 文件 | 优先级 | 状态 |
|----|----------|------|--------|------|
| P1-1 | RBAC 缓存刷新窗口不一致 | `services/gateway/internal/middleware/rbac_dynamic.go` | P1 | 📝 待修复 |
| P1-2 | DELETE 响应格式不统一 | 多个服务 | P1 | 📝 待修复 |

---

## 下一步行动

### 高优先级
1. [ ] 实现 RBAC 缓存主动失效机制（通过 Kafka/Redis pub/sub 或 DB 触发器）
2. [ ] 定义 DELETE 端点响应标准并更新所有处理器
3. [ ] 为 device_bound_sso.go 的 TODO 制定修复计划

### 中优先级
4. [ ] 重构 `permLevelRank` 使用配置表
5. [ ] 优化 RBAC 非前缀过滤（DB 层过滤）
6. [ ] 审查 admin-protected 与 permission-key fallback 的交互逻辑

### 低优先级
7. [ ] 清理架构债务标记（94 个 TODO/FIXME/mock/hardcode）
8. [ ] 统一 API 错误响应格式（与 DELETE 问题一起处理）

---

## 审查结论

**总体评估**: ✅ 良好

- ✅ RBAC 多层缓存架构设计合理
- ✅ 租户隔离和 public path 免白名单有明确保护
- ✅ 最近 P0/P1 修复覆盖关键安全风险
- ⚠️ 缓存刷新机制有 120s 延迟窗口，需优化
- ⚠️ DELETE 响应格式未统一，存在兼容性风险

**建议下次审查**:
- OAuth token endpoint RFC 6749 合规性（近期有 token exchange 修复）
- 分层配置体系（App→Tenant→Instance fallback）

---

**审查完成时间**: 2026-07-29
**审查耗时**: ~10 分钟
**状态**: 已完成