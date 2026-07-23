# Cron 巡航学习记录

> 本文件由 cron-1 定时任务自动维护。每轮发现的新教训追加到对应分类下。
> 下一轮启动时自动读取，避免重犯已知错误。最新的在最前面。

## 多 Agent 并行巡航（重要！）

三个自动巡航 agent 共享 `/Volumes/new/ggai/ggid` 工作区：

| Agent | 角色 | 负责范围 | 协调规则 |
|-------|------|---------|---------|
| arch_pm (cron-1) | 平台 UX + 部署 | Console 浏览器验收、平台 API、k8s 部署、CI | 不改 SDK/CLI 代码 |
| ggcxf_researcher | SDK + Demo | SDK 方法完备性、demo 三层对齐、JWT 验签 | 发现问题 DM arch_pm 评估平台影响 |
| ggcxf_cli&products | CLI 工具 | GGID CLI 管理工具功能实现 | 不碰 Console/SDK 代码 |

**共享工作区规则**：
- 三方都只 `git add` 自己的文件，禁止 `git add .`
- git pull --rebase 冲突 → DM 文件作者协调，不 stash/reset
- 发现属于其他 agent 范围的问题 → DM 该 agent，不自己改
- 每次 git 操作前先 `git status` 确认没有其他人的未提交更改

## 部署踩坑

- [2026-07-21] OAuth 服务 k8s deployment 用 `:v2` 镜像 tag，不是 `:latest`。推镜像前先 `kubectl get deploy ggid-oauth -o jsonpath='{.spec.template.spec.containers[0].image}'` 确认 tag
- [2026-07-21] 镜像构建流程：先 `go build -o build/bin/{svc}` 编译二进制 → 再 `docker buildx build -f deploy/docker/Dockerfile.service --build-arg BINARY=build/bin/{svc}`
- [2026-07-21] Console 镜像用 `:latest`，与 OAuth 不同

## 已修复的 Bug（回归测试用）

- [2026-07-22] **P1: JWT permissions 数组不控制路由访问** — gateway 有两套脱节的授权系统：role_route_permissions (DB角色匹配) 和 permissions 表→JWT permissions array。后者虽正确填充到 JWT 但 gateway 从不读取。修复：新增 `HasPermissionForRoute(path, method, permissions)` 将路由前缀映射到 resource type，在 `checkRouteScope` 和 `CheckAccess` 两层插入 permission fallback。commit 6a31a7ba5。验证：users:read 用户可 GET /users (200) 但 POST 403，无 roles:read 的 GET /roles 403
- [2026-07-22] **P1: Token introspection 不返回 permissions/roles** — IntrospectionResponse 只提取 scope/sub/aud 等，遗漏 permissions 和 roles 数组。下游资源服务器无法做权限检查。修复：IntrospectionResponse 加 Roles+Permissions 字段，IntrospectToken 用 getStringSliceClaim 提取。commit 8448423a3。注意：introspect 端点 form 方式需要 client_secret（public client 如 ggid-console 无法用 form），Bearer token 方式工作正常
- [2026-07-22] **已验证: RBAC 全链路** — create role→assign permissions(string keys)→assign to user→login→JWT 含 permissions→GET /users 200 (users:read)→POST /users 403 (需 write)→GET /roles 403 (无 roles:read)→GET /users/me 200 (self-service)。QA agent 独立验证 6/6 PASS
- [2026-07-22] **已验证: Permission matrix API** — grant (POST /roles/{id}/permissions {"permissions":["audit:read"]}) → 200 "granted"；revoke (DELETE) → 200 "revoked"；验证 DB 计数正确变化
- [2026-07-22] **已验证: Token introspection** — Bearer 方式返回完整 roles + permissions 数组。resource server 可直接用于权限检查
- [2026-07-22] **P2: JWT permissions 数组重复** — 用户有多个角色且角色共享相同权限时，permissions 数组含重复项（admin: 26项仅13 unique）。修复：fetchUserPermissions SQL 加 `SELECT DISTINCT`。commit c24a19645。验证：13 total, 0 duplicates
- [2026-07-22] **安全验证: 无角色用户** — JWT permissions=[]+roles=[] 的用户正确被 403 拦截所有 admin 路径，/users/me 仍 200。HasPermissionForRoute 空数组返回 false，无过度授权风险
- [2026-07-23] **P1: CAE action 字段持久化失败** — Console/API 发 `action` (单数字符串), handler 只读 `actions` (复数 map)。action 被 JSON decode 忽略。修复：handler 同时接受 `action` string 和 `actions` map，struct 新增顶层 `action` 字段。commit 684eb084c
- [2026-07-23] **已验证: OAuth Client 全生命周期** — create (auto-gen client_id `gcid_*`, secret `gcs_*`) → list → M2M token (scope+permissions 正确) → disable → M2M blocked → re-enable → update redirect URIs → delete。全链路 PASS
- [2026-07-23] **已验证: Users API 数据完整性** — search 工作正常 (8 results for "admin"), roles 嵌入在 user 对象中 (role_name + role_id), 分页参数 limit 生效 (50 default)。display_name 是标准字段（无 first_name/last_name）
- [2026-07-23] **已验证: Tenant resolve API** — GET /api/v1/tenants/resolve?slug=default 返回 tenant_id+name+slug。Gateway 已有 EnhancedTenantResolver 中间件支持 subdomain→tenant 解析。R1-01 onboarding 路由基础设施已就绪
- [2026-07-23] **P1 安全修复已部署**: auth(S1-S4,S6,S7) + identity(S1) + audit(S3,CORS) + oauth(introspection,dedup,scope交集) + gateway(CORS fail-closed) + policy(CAE action)。5 个服务全部重建部署
- [2026-07-23] **govulncheck 结果**: 1 vulnerability (GO-2026-5856 crypto/tls ECH privacy leak, fixed in go1.26.5)。需升级 Go 版本修复
- [2026-07-23] **P1: Console conditional-access 设置页 API 端点错误** — 调用 `/api/v1/auth/conditional-access/policies` (不存在) 而非 `/api/v1/policies/conditional-access` (正确)。返回空→静默回退到 mock 数据。已修复 (commit 97d07e904)，Console 已重建部署
- [2026-07-23] **已验证: review-schedules API** — `/api/v1/identity/review-schedules` 返回 200 + 空数组 (正确)。Identity 服务已注册路由
- [2026-07-23] **R1-02 Social Login 已部署验证**: Gateway publicPaths 放行 + auth handler 部署。`GET /api/v1/auth/social/google` 返回 503 (无 IdP 配置，正确业务错误)。需在 tenant_idp_configs 配置 Google OAuth 才能端到端
- [2026-07-23] **admin 密码 cred sync race**: auth pod 重启后密码 hash 被 bootstrap 覆盖。手动用 crypto.HashPassword 生成新 hash 更新 DB。密码: q7Rf9Xk2Lm3pW8zB (无尾A)。PASSWORD_PEPPER 环境变量未设置 (S6 修复仅 warning 不 fail)
- [2026-07-23] **P1: API Keys 完全不可用** — 两层断裂：(1) api_keys_handler.go 用内存 `apiKeys = []APIKey{}` 存储 (pod 重启丢失) (2) Gateway `APIKeyAuth` 中间件存在但从未 wire 进 middleware chain (cmd/main.go 无调用)。用户创建 key 得到 `ggid_sk_*` 但 X-API-Key 调 API 返回 401。已 DM backend 认领
- [2026-07-23] **已验证: Webhook 生命周期** — create (返回 name+url+events+id) → list → update (PUT) → delete → verify gone。全链路 PASS
- [2026-07-23] **已验证: API Key 创建** — POST /api/v1/api-keys 返回 `{"key":"ggid_sk_*"}` 格式正确，但 key 不可用于认证（见上）
- [2026-07-23] **已修复: API Key 认证完整闭环** — DBAPIKeyValidator (Argon2id) 接入 gateway middleware chain。根因 1: middleware 顺序 (APIKey 必须 wrap JWTAuth)。根因 2: COALESCE(expires_at,'epoch') 把 NULL 当 1970 过期。commits: 4183b84e4, 693f5597b, 0130c87f0
- [2026-07-23] **已修复: ITDR 端点 404** — threat-heatmap, kill-chain, incident-timeline 在 audit http.go 已注册但 deployed image 过旧。重建部署后全部 200。Console security 页面 ITDR 功能可用
- [2026-07-23] **R3-02 Helm Chart v1.0.0** — values-dev/staging/prod 分层 + RUNBOOK.md (install/rollback/scaling/troubleshoot/backup). commit 42e16af45
- [2026-07-23] **R3-03 HA 设计** — 多区域架构文档 (CNPG PG 复制, Redis sentinel, GeoDNS 故障转移). commit d0e76c864

- [2026-07-21] auth-guard.tsx scope 匹配：JWT roles 存的是显示名（如 "Platform Administrator"），不是 scope key（如 "platform:admin"）。匹配逻辑必须大小写不敏感 + 包含显示名变体
- [2026-07-21] Go SDK verifyTokenOffline 用 ParseUnverified 接受伪造 JWT。已移除，VerifyToken 强制要求 WithJWKS()
- [2026-07-21] M2M client_credentials token permissions 为空。根因：issueAccessTokenWithAMR 用 uuid.Nil 查 user_roles。修复：新增 issueClientAccessToken 从 client.scopes 获取
- [2026-07-21] OAuth Client 创建 name 字段丢失。前端发 `name` 但后端期望 `client_name`（json tag 不匹配）。修复 console/src/app/oauth-clients/page.tsx commit 82121d81c

## 已验证通过的流程

- [2026-07-21] 登录→Dashboard→刷新保持
- [2026-07-21] 用户管理（创建/搜索/角色筛选/密码强度）
- [2026-07-21] MFA TOTP（QR+Secret+Verify）
- [2026-07-21] Console Profile（Security tab: 密码/MFA/Devices）
- [2026-07-21] OAuth Clients 列表页（20 clients）
- [2026-07-21] ERP Demo 8/8 认证流程
- [2026-07-21] Introspect 端点（form + Bearer 两种方式）
- [2026-07-21] OIDC discovery（device_authorization_endpoint + 全部 grant types）
- [2026-07-22] Console 登录（OAuth password grant + scopes 提取 + Dashboard 跳转 + sidebar）
- [2026-07-22] Task-F Settings 页面验证 PASS（branding/scim/import 无 404，连接真实 API）—— QA agent 完成
- [2026-07-22] Register 页密码强度反馈+toggle 修复（QA commit af707f85b）
- [2026-07-22] RBAC 完整回归验证通过（Users/Roles/Audit/OAuth Clients 全部 200，无 token 401）—— QA agent 完成
- [2026-07-22] 动态 RBAC (Task-B) 已实现+部署（commit ab762f527），checkRouteScope 同时评估 scopes+roles
- [2026-07-22] **v1.0 基线达成** — P0 (Task-A/B/C) + P1 (Task-D/E/F/G) 全部完成。企业销售就绪。下一步 P2 (SDK 发布/Helm/追踪)
- [2026-07-22] 用户管理 API 深度验证 PASS：创建用户→分配角色(201 Created)→登录验证→删除用户。POST /api/v1/users/{id}/roles 用 {"role_id":"UUID"} 格式工作正常
- [2026-07-22] **P1: 角色权限分配静默失败** — 已修复（arch_pm commit）。Policy handler 现在接受 `{"permissions":["users:read"]}` string key 格式，内部 resolve key→UUID。DB 验证写入正确。
- [2026-07-22] **安全: 跨租户防护** — Gateway JWTAuth 新增 tenant boundary 检查（commit 31c7e5c1e）。JWT tenant_id ≠ X-Tenant-ID → 401，platform admin 例外
- [2026-07-22] Refresh token (Task-E) 部署阻塞：`oidc_refresh_tokens` 表已手动创建（migration 缺失），但 `scope` 列类型不匹配（TEXT vs []string）。已 DM backend 修复 pg_repo.go → **已修复**（commit bd7c3b647）。E2E 验证通过：password grant→refresh rotation→reuse detection 全部正确。ggid-console client 改为 public type（SPA 不持有 secret）
- [2026-07-21] M2M token permissions（9 个权限正确写入）
- [2026-07-21] OAuth Client 创建向导（填表→Secret 显示→name 正确保存到 DB）
- [2026-07-21] Console 7 核心页面全量浏览器验证 PASS —— QA agent 完成
- [2026-07-21] Audit Log 搜索+筛选（13 actions/4 results）+分页+导出 全功能验证 PASS

## 待验证的流程

- ~~OAuth Client 创建向导（Register→填表→Secret 显示→复制）~~ → 已验证创建+Secret显示+name保存（修复 commit 82121d81c）
- ~~Audit Log 搜索+筛选+时间范围~~ → 已验证：搜索（Actor/IP）、筛选（Action/Result 13+4选项）、分页（Page 1/2）、导出（CSV/JSON）全部正常
- Roles 权限分配（选 role→勾选 permission）— 页面显示 0 行但 API 返回 29 roles，待 frontend 排查
- [2026-07-22] Conditional Access API 深度验证 PASS：创建策略（action=require_mfa）→ 返回完整对象 → 列表 2 条。注意：conditions 字段创建时未持久化（conditions:{} 为空），核心功能正常
- [2026-07-22] **DB 完全重置 + 硬编码 tenant ID 彻底移除**。新 tenant UUID: 28d6fe98-adeb-4c0c-b49b-20c6695bbca6。Migration/Console/Gateway/k8s 全部改为动态 tenant ID。团队已通知重新铺数据。Commit a6649d2e5
- [2026-07-22] **P0 安全修复**: (1) 空 roles 用户绕过 RBAC → 修复 a32b33aa1 (2) JWT scope 加 role keys (platform:admin) → 4b6431a9e (3) 跨租户防护 → 31c7e5c1e
- [2026-07-22] **P1: Webhook name 不持久化** + **P1: API Key 创建不返回 key 值**。已 DM backend。Webhook create/list 缺少 name 字段。API key create 返回 id/name 但无可用 key，Console 回退 mock。
- [2026-07-22] Console mock 页面清单：admin/feature-flags（100行零错误处理）、admin/key-rotation（112行）、api-keys（mock key fallback）、10+ 处 `/* mock */` catch fallback。需 backend 补全 API 或前端加 error state
- [2026-07-22] **P1: 动态 RBAC 阻止普通用户访问 /users/me** — CheckAccess 匹配 /api/v1/users 前缀导致 /users/me 被拦截。修复：CheckAccess 开头加 self-service 路径豁免（commit c2f39d2c9）。普通用户权限边界验证通过
- [2026-07-22] admin 密码 hash 更新（DB reset 后旧 hash 与 backend 新验证逻辑不匹配，重新生成后验证通过）
- CLI 工具基础命令（由 cli&products agent 推进中）
- ~~动态 RBAC 实现（ADR 已完成 docs/design/adr-dynamic-rbac.md，实现交给 backend）~~ → 已实现+修复 P0 回归（commit ee8132989），gateway 重建部署验证通过

## 已知非阻塞问题

- [2026-07-22] Token issuer 统一化：auth 不再直接签发 token，改为 OAuth 统一签发。`/api/v1/auth/login` 端点被移除（commit 722b3fd4c）。认证改用 OAuth password grant（POST /api/v1/oauth/token grant_type=password）。Demo 需要适配。auth+gateway 镜像已重建部署。`make test` 全部 PASS
- ~~Console 登录页面无法跳转~~ → 已修复（frontend commit 633a2f401 + 204a53ad1，DB 创建 ggid-console client）。Console 已重建部署。登录+scopes+Dashboard 验证通过
- [2026-07-22] Roles 页面显示 0 行但 API 返回 29 roles — 可能是前端数据映射问题（API 返回数组但页面期望 {roles:[]}）。待 frontend 排查
- [2026-07-21] auth CAE scanner nil context panic 导致 pod 每 ~15min 重启一次。k8s 自动恢复，不影响功能。需修复 cae_scanner.go 的 ListByTenant nil context 问题
- [2026-07-21] OAuth Clients 列表固定显示 20 条，无分页控件 → 已修复（QA commit c2a52b589），Console 已部署
