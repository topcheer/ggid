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