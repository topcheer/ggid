# cron-1 8 维度后端深度审查规范

> 更新时间: 2026-07-28
> 定时: `17,47 * * * *` (每30分钟)
> 覆盖: 10 服务 / 110 路由前缀 / 270 DB 表
> 执行方式: 主代理 spawn_agent 创建一次性 subagent 完成审查+修复+commit，主代理 wait_agent 收结果、核验 make test、DM arch_pm 部署

## 维度轮换表（每次取下一个）

### D1 身份认证全链路
注册(self-register/admin-create/B2B/invite) → 登录(password/passkey/OTP/social/SAML) → 密码策略(min/max/complexity/pepper一致性) → 密码重置 → 凭证存储(argon2id) → 软删除 → 凭证残留检查

### D2 MFA与无密码认证
TOTP绑定+加密 → SMS/email OTP发送+验证+暴力破解防护 → WebAuthn/passkey注册+登录+归属验证 → backup codes → require_mfa策略 → auth_ticket创建+验证+跨租户校验 → pepper一致性

### D3 OAuth/OIDC协议安全
client注册 → authorize → PKCE(S256-only) → token签发(iss/aud/exp/jti) → refresh rotation → revocation → introspection → client_credentials → auth_ticket exchange → grant类型校验 → redirect URI验证

### D4 RBAC与权限引擎
中间件链(JWTAuth→CAE→RequireAdminScope→proxy) → adminOnlyPaths覆盖 → 动态RBAC(role_route_permissions) → 权限粒度 → impersonation token → scope过滤(不可升级) → policy service决策

### D5 审计链与合规
hash chain覆盖率 → tamper-check端点 → WORM触发器 → 敏感操作审计覆盖 → SIEM转发 → access review → ITDR检测 → repair-chain工具安全

### D6 多租户隔离与数据安全
RLS策略状态 → BYPASSRLS → WHERE tenant_id覆盖 → 跨租户泄露检测 → SCIM /Me隔离 → org service隔离 → conditional_access_store隔离

### D7 会话与令牌生命周期
session创建 → 超时 → 清理goroutine → refresh token rotation → orphan cleanup → token撤销 → CAE jti blocklist → concurrent session limit

### D8 安全配置与加固
加密密钥(RSA/TOTP/pepper) → secrets管理 → TLS → CORS → security headers → rate limiting → 输入验证/SQL injection → MCP service(RFC 9728) → 硬编码凭据扫描

## 执行规则
- 用 kubectl run 创建临时 pod 调用 API
- 每个检查必须查实际 DB 数据 (psql)
- 发现FAIL: 修复→make test→commit→DM arch_pm
- 最近3个代码commit审查 nil-pointer / error handling / 边界
- 不重复上一轮, 查 git log --since="1 hour ago"
- **每个维度至少检查3个子项**

## 输出格式
检查维度名 + PASS/FAIL + 证据(DB查询结果/API响应)
