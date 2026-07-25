# GGID 端到端验证计划

> 创建：2026-07-23 | 状态：进行中 | 每30分钟推进一个阶段

## 验证矩阵

| # | 阶段 | 验证角色 | 关键验证点 | 状态 |
|---|------|---------|-----------|------|
| S1 | 实例部署 | DevOps | 服务健康、DB/Redis/NATS 连通 | ✅ 已完成 |
| S2 | 实例初始化 | 实例管理员 | bootstrap、admin 创建、默认租户 | ✅ 已完成 |
| S3 | Console 基础 | 实例管理员 | 登录、Dashboard、用户列表、系统设置 | ✅ 通过（admin 权限修复） |
| S4 | 租户管理 | 实例管理员 | 创建租户、切换租户、租户隔离 | ✅ 通过（1 个小 bug） |
| S5 | 用户生命周期 | 租户管理员 | 创建用户、编辑、禁用、删除、导入 | ✅ 通过（1 个小 bug） |
| S6 | 角色权限 | 租户管理员 | 创建角色、分配权限、角色继承、权限矩阵 | ✅ 通过（已修复） |
| S7 | OAuth/OIDC | 应用开发者 | 客户端管理、授权码流程、PKCE、token exchange | ✅ 通过 |
| S8 | API Keys | 应用开发者 | 创建、使用、轮换、撤销 | ✅ 通过（1 bug） |
| S9 | SDK 验证 | 应用开发者 | Node SDK login+verify、Python SDK login+verify | ✅ 通过 |
| S10 | 审计安全 | 安全合规官 | 审计日志、hash-chain、ITDR、异常检测 | ✅ 通过（已修复） |
| S11 | 策略引擎 | 安全合规官 | ABAC 策略、资源访问控制、条件策略 | ✅ 通过（已修复） |
| S12 | 多租户隔离 | 租户管理员 | 跨租户数据隔离、越权访问拦截 | ✅ 通过（2 个问题） |
| S13 | ERP Demo 集成 | 最终用户 | 7 SDK demo 认证+权限+API 调用 | ✅ 通过（70 连续周期） |
| S14 | MCP/Agent | 应用开发者 | Agent token、scope 授权、审计 | ✅ 通过 |
| S15 | 越界检测 | 安全合规官 | 权限提升检测、未授权路径、注入防护 | ✅ 部分通过（2 个安全问题） |

## 当前执行阶段

**验证完成 — 等待 S14 结果**

## 执行日志

### S1: 实例部署 ✅
- 时间：2026-07-23 14:55
- 结果：8 核心服务 Running，PG/Redis/NATS 连通，healthz OK

### S2: 实例初始化 ✅
- 时间：2026-07-23 15:15
- 结果：identity bootstrap 成功，admin 用户创建，console OAuth client 创建
- 凭据：admin / SecureAdmin@Pass2026#Xq
- Tenant ID: fb44ca98-2a8a-498b-a9b2-00fc014524ce

### S14: MCP/Agent 验证 ✅
- 时间：2026-07-24 05:30
- 执行者：arch_pm (commits a2a9b6be2, 99873e836)
- 结果：
  1. ✅ JWT RS256 认证
  2. ✅ tools/list: 13 工具（permissions 数组过滤）
  3. ✅ tools/call 审计记录（agent_id + user_id + status）
  4. ✅ GET /mcp/audit: entries 返回正常
  5. ⚠️ tools/call API 调用 401 — 部署配置问题（需 GGID_ACCESS_TOKEN env），非代码 bug

### Deep Audit Fixes (arch_pm)
- P0: Conditional Access Policy enforcement (commit 58d222d57) — CAE policies now enforced during login
- P1: Consent cascade DB table fix (commit f8eebd302) — oauth_tokens→refresh_tokens, auth_sessions→sessions
- 新增: POST /api/v1/auth/conditional-access/evaluate endpoint

### JWT Revocation Bridge Fix (ggcxf_cli, commit 0b6003855)
- 根因: OAuth RevokeToken 写 `oauth:revoked:<hash>`，Gateway CAE 检查 `ggid:revoked_jti` ZSET — 两套机制未连通
- 修复: RevokeToken 现在同时写 JTI 到 ZSET（添加 ZAdd 到 RedisCmdable 接口）
- 验证: 手动 ZADD JTI → Gateway 返回 401 ✅（Gateway CAE 工作正常）
- 部署: arch_pm 导入镜像 sha 1caee81f，E2E 验证通过 — revoke 后 HTTP 401 ✅ 闭环

## 团队分工

| 角色 | LAN Chat 成员 | 负责阶段 |
|------|-------------|---------|
| DevOps/协调 | ggcxf_cli | S1-S2, S12, S15 |
| Console | shen_frontend | S3, S5, S6 |
| 后端 | ggcxf_backend | S4, S7, S8, S11 |
| SDK/Demo | ggcxf_researcher | S9, S13 |
| 安全 | guardian_security | S10, S15 |
| 架构/PM | arch_pm | S14, 整体 review |
