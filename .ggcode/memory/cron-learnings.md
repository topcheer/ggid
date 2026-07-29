## 2026-07-28 arch_pm 深度审查

### P0 发现并修复: JWT Audience 不匹配导致全站 401
- **根因**: OAuth 签发的 JWT aud=ggid-console (client_id)，但 gateway 配置 GATEWAY_JWT_AUDIENCE=gcid-console
- **影响**: 除 /dashboard/stats (publicPaths 豁免) 外所有受保护端点返回 401
- **修复**: kubectl set env GATEWAY_JWT_AUDIENCE=ggid-console + rollout restart
- **可能引入时间**: P0-2 JWT kid key rotation 修复后

### P1 发现并修复: Permissions DB level 全部为 tenant
- **根因**: EnsureSystemPermissions 的 UPSERT 新增了 level 字段，但之前部署的旧版本已用默认值 'tenant' 插入。ON CONFLICT DO UPDATE 现在包含 level，但需要新版本部署才会生效
- **修复**: 手动 UPDATE permissions SET level='instance' WHERE key LIKE 'tenants:%' OR key LIKE 'system:%' (13 行) + 重建部署 policy 服务
- **验证**: API 返回 79 tenant + 13 instance，分布正确

### Admin 密码重置
- auth pod 6 次重启后 bootstrap 覆盖了密码 hash
- 新密码: Admin@2026Reset#9
- 重置方法: kubectl exec auth pod → printenv PASSWORD_PEPPER → go run hash_pw with SetPepper() → UPDATE credentials SET secret
- 注意: pkgcrypto.HashPassword(password) 只接收 1 个参数，pepper 通过 SetPepper() 全局设置

### 权限 API 端点确认
- 正确端点: /api/v1/permissions (不是 /api/v1/policies/permissions)
- Console roles 页面正确调用 /api/v1/permissions 和 /api/v1/roles/{id}/permissions
- /api/v1/policies/permissions 被错误路由到 /api/v1/policies/{id} handler

---

## 2026-07-29 深度功能审查 (R2)

### 审查范围
1. RBAC 动态 resolver 缓存一致性
2. DELETE 响应格式一致性
3. 架构债务标记搜索 (TODO/FIXME/mock/hardcode)

### P1-1: RBAC 缓存刷新窗口不一致
- **根因**: Memory TTL (60s) + Redis TTL (60s) 叠加，策略更新后最长 120s 延迟
- **影响**: 权限变更（如角色调整、路由权限修改）无法即时生效
- **建议**: 增加 Invalidate() 触发点（通过 DB 触发器或应用层事件），考虑缩短 Redis TTL 至 30s
- **参考文件**: `services/gateway/internal/middleware/rbac_dynamic.go` L148-186

### P1-2: DELETE 响应格式不统一
- **根因**: 未强制统一 DELETE 端点响应标准，部分可能返回 200 + JSON，部分未定义
- **影响**: SDK 兼容性风险，REST 最佳实践（204 No Content）未落实
- **建议**: 统一为 204 No Content，添加 DELETE 端点响应格式测试，更新 OpenAPI spec
- **影响范围**: `/api/v1/users/:id`, `/api/v1/roles/:id`, `/api/v1/policies/:id` 等

### 良好实践
- ✅ RBAC 三层缓存降级（Memory → Redis → DB）设计合理
- ✅ 租户隔离强制检查防止跨租户提权
- ✅ Public Path 免白名单避免 P0 事故 (/oauth/token 拦截)
- ✅ Superuser 范围隔离（仅 scopes claim）

### 架构债务
- 搜索到 94 个 TODO/FIXME/mock/hardcode 标记
- 优先关注: `services/oauth/internal/service/device_bound_sso.go` (安全敏感)

### 下次审查建议
- OAuth token endpoint RFC 6749 合规性
- 分层配置体系 (App→Tenant→Instance fallback)

---

## 2026-07-29 深度功能审查 (R3)

### 审查范围
1. OAuth refresh token 轮换路径完整性 (auth Redis fallback 路径)
2. Conditional Access 策略评估逻辑
3. Audit hash chain 完整性验证
4. Console error helper 统一性

### P0-1: Refresh token 撤销时 Redis 和 DB 状态可能不一致
- **根因**: RevokeRefreshToken 先删除 Redis，再更新数据库。如果数据库更新失败，Redis 缓存已删除但 DB 中 token 仍然有效
- **影响**: 攻击者可以在 Redis 删除后、DB 更新前的时间窗口内使用被撤销的 token
- **修复方案**: 先执行 DB 撤销 (源数据)，成功后再删除 Redis 缓存。如果 DB 撤销失败，保留 Redis 缓存并返回错误
- **参考文件**: `services/auth/internal/service/token_service.go` L148-163

### P0-2: Conditional Access 策略评估函数双重实现
- **根因**: 存在两个同名的 EvaluateConditionalAccess 函数：
  - HTTPServer.EvaluateConditionalAccess (L218): 完整实现，包含策略匹配逻辑
  - EvaluateConditionalAccess (L267): 包级别函数，硬编码返回 "allow", nil
- **影响**: 如果其他地方调用包级别函数，会完全绕过所有 Conditional Access 策略检查
- **修复方案**: 删除或废弃包级别函数，添加 Deprecated 注解或返回错误
- **参考文件**: `services/policy/internal/server/conditional_access_handler.go` L267-269

### P1-1: Refresh token 撤销缺少事务回滚机制
- **根因**: 两步操作 (Redis + DB) 没有事务回滚机制，状态不一致时无法自动恢复
- **影响**: Redis 删除成功但 DB 更新失败后，系统进入不一致状态
- **修复方案**: 在 DB 更新失败时，记录到 Redis 作为 "待撤销" 标记，异步重试

### P1-2: Redis fallback 路径未在 token 验证中实现
- **根因**: FindByHash 只查询数据库，不检查 Redis 缓存
- **影响**: Redis 无法加速 token 验证路径，高并发场景下 DB 压力大
- **修复方案**: 在 FindByHash 中先检查 Redis，再 fallback 到 DB

### P1-3: Conditional Access 上下文数据未经验证直接使用
- **根因**: EvaluateConditionalAccess 直接使用 ctxMap 中的值进行字符串比较，没有类型安全检查
- **影响**: 类型不匹配可能导致误判，超大输入可能导致拒绝服务
- **修复方案**: 添加输入大小限制和类型验证

### 良好实践
- ✅ Audit hash chain 的 FOR UPDATE 锁正确防止并发写入导致的链断裂
- ✅ Conditional Access 的 tenant 隔离强制从 header 获取 tenant_id，防止欺骗
- ✅ Token hash 计算 SHA256 哈希存储 token，不存储明文

### 架构债务
- P2-1: Hash chain 秘钥轮转未持久化版本号，无法确定旧事件使用的 secret version
- P2-2: Canonical JSON 字段顺序依赖 Go 默认序列化，存在风险
- P3-1: Console error helper 错误消息提取逻辑不一致

### 最近变更相关
- 最近的修复包括 "atomic refresh token consumption" 和 "revoke nil-tenant fix"
- 这些修复可能与 P0-1 问题相关，需要确认是否已经解决部分问题