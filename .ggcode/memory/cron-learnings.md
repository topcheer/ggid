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