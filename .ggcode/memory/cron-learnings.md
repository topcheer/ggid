# Cron 执行 Lessons - GGID IAM 审查 R5

## 执行日期
2026-07-30

## 审查范围
- DeleteClient cascade cleanup（验证修复）
- OAuth token exchange scope narrowing
- Audit webhook tenant isolation（验证修复）
- Feature flags 多租户隔离

## 发现的问题

### P0 - OAuth token exchange scope narrowing
**文件**: `services/oauth/internal/server/token_exchange_delegation.go`

**问题**: 根据 RFC 8693，token exchange 应该验证请求的 scope 不超过原始 token 的 scope，但当前实现直接使用请求的 scope，没有验证。

**修复方案**:
1. 从 subject_token 和 actor_token 提取 scopes
2. 验证请求的 scope 是原始 scopes 的子集
3. 如果超范围，返回 400 error with `invalid_scope`

### P1 - Feature flags 多租户隔离
**文件**: `services/auth/internal/server/feature_flags_handler.go`

**问题**: Feature flags 使用全局变量存储，所有租户共享，没有任何 tenant 隔离：
- GET 返回所有租户的 flags
- POST 添加到全局存储
- PUT 修改全局 flag
- 审计日志也是全局的

**修复方案**:
- 方案 1: 数据库存储，添加 tenant_id 列
- 方案 2: 内存存储，使用 `map[tenantID]*TenantFeatureFlags`
- 所有操作都检查 X-Tenant-ID header
- 审计日志也按 tenant 分离

## 已验证的修复

### DeleteClient cascade cleanup ✅
**提交**: `a84acd6b7`

**修复内容**:
1. 将 `gcid_xxx` 解析为内部 UUID
2. 使用 `revoked_at = now()` 替代 `revoked = true`
3. 添加 `AND revoked_at IS NULL` 避免重复撤销

**验证**: ✅ 修复正确，所有级联清理操作使用正确的 UUID 类型

### Audit webhook tenant isolation ✅
**提交**: `98bf151d0`

**修复内容**:
1. 所有 CRUD 操作都要求 X-Tenant-ID header
2. GET/POST/DELETE/PUT/PATCH 都按 tenant_id 过滤
3. 内存回退也正确过滤 tenant

**验证**: ✅ 修复正确，包括 DB 和内存回退两种场景

## Git 操作 Lessons

1. **Stash 处理**: 工作区有未提交更改时，使用 `git stash push` 暂存，然后 `git pull --rebase`，再 `git stash pop`（选择性恢复）

2. **Stash 内容审查**: stash 包含多个文件（cron-learnings.md、iam-review-2026-07-29.md、scripts/auto-review/main.go 等），都是之前的审查产物，需要选择性恢复

## Kubectl 操作 Lessons

1. **Pod 状态检查**: `kubectl get pods -n ggid` 显示所有服务健康运行，Ready 状态正常

2. **Restart 注意**: ggid-auth 有 3 个副本，每个都有 1 次重启（约 62s 前），可能是配置更新导致的正常重启

## 代码审查 Lessons

1. **Schema 验证**: 修复 cascade cleanup 时，需要确认数据库 schema 中的列名类型（`revoked_at` 是 TIMESTAMPTZ，不是 boolean）

2. **UUID 类型匹配**: OAuth token 表中 `client_id` 是 UUID 类型，不能直接使用 `gcid_xxx` 字符串，需要先解析为内部 UUID

3. **多租户隔离模式**: 正确的模式是：
   - DB 存储时包含 `tenant_id` 列
   - 所有查询都添加 `WHERE tenant_id = $1`
   - HTTP handler 从 `X-Tenant-ID` header 获取租户上下文
   - 如果 header 缺失，返回 403 Forbidden

4. **Scope narrowing**: OAuth 2.0 Token Exchange (RFC 8693) 要求验证请求的 scope 不超过原始 token 的 scope，这是防止权限升级的关键安全检查

## 架构债务 Lessons

1. **TODO/FIXME 搜索**: 当前项目中没有 `FIXME` 标记，`TODO` 主要是文档注释，不是技术债务

2. **Hardcode 检查**: 没有发现 `hardcode` 标记，代码中没有明显的硬编码问题

## 下次审查重点

1. **Scope narrowing 实现**: 检查 token exchange 是否正确实现 scope narrowing
2. **Feature flags 修复**: 验证多租户隔离修复是否完整
3. **其他 cascade cleanup**: 检查其他服务（如 user delete）是否也需要类似的级联清理
4. **RLS 策略**: 检查数据库 Row Level Security 是否覆盖所有敏感表