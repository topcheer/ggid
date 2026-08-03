# IAM 安全审查报告 2026-08-02

## 审查范围
- 编译验证：`go build ./...` ✅ 通过
- 核心服务回归测试：`go test -timeout 60s ./services/oauth/internal/service/ ./services/auth/internal/service/ ./services/identity/internal/scim/` ✅ 全部通过
- 最近10次提交分析

## 发现的问题

### P0 (1个，新增)

#### Helm deployments 引用的 `{fullname}-secrets` Secret 不存在

**涉及提交**: `50d00c229` (fix R284 P0), `323cf0dee` (fix R281 P0)

**问题描述**:
`deploy/helm/ggid/templates/deployments.yaml` 中引用了 Secret `{{ include "ggid.fullname" $ }}-secrets`（lines 89, 94, 99, 105），但该 Secret 在 `deploy/helm/ggid/templates/secrets.yaml` 中**未定义**。

当前 deployments.yaml 引用的key包括：
- password-pepper (line 89-90)
- audit-hash-secret (line 94-95)
- internal-secret (line 99-100)
- encryption-key (line 105-106, 条件注入)

当前 secrets.yaml 仅包含：
- `{fullname}-db-secret` (key: password)
- `{fullname}-jwt-secret` (key: secret)
- `{fullname}-db-url` (key: database-url, 已在 f6b9a87ad 添加)

**影响**:
- Helm 安装/升级时，所有服务（gateway, identity, auth, oauth, policy, org, audit）的Pod会因Secret不存在而无法启动（CrashLoopBackOff）
- 缺失的secrets（PASSWORD_PEPPER, AUDIT_HASH_CHAIN_SECRET, GGID_INTERNAL_SECRET, GGID_ENCRYPTION_KEY）导致密码哈希、审计哈希、内部认证、加密功能失效

**建议修复**:
需要在 `deploy/helm/ggid/templates/secrets.yaml` 中添加 `{fullname}-secrets` Secret，包含必要的keys。

---

### P0 (1个，已修复)

#### R288 后续：数据库迁移 Job 使用的 Secret 不存在
**状态**: ✅ 已修复（提交 f6b9a87ad）

**问题**: `deploy/helm/ggid/templates/db-migrate-job.yaml` 引用了 Secret `{{ include "ggid.fullname" . }}-db-url`（key: `database-url`），但该 Secret 在 `deploy/helm/ggid/templates/secrets.yaml` 中未定义。

**修复**: 提交 `f6b9a87ad` 在 `secrets.yaml` 中添加了 `{fullname}-db-url` Secret。

---

## 其他提交审查结果

### ✅ 06a904f2a - fix(R288 P0): reject unknown hash types in bulk import
正确实现了非开发环境下拒绝未知 hash 类型，防止认证绕过。

### ✅ ad8d3a9ca - fix(R287 P2): reject plaintext hash_type in bulk import
正确实现了非开发环境下拒绝明文密码哈希。

### ✅ 323cf0dee + 50d00c229 - R281/R284 Helm secrets 注入
R281 先添加了 plaintext 值注入，R284 将其修正为 secretKeyRef，但引用的 `{fullname}-secrets` Secret 不存在（新发现P0问题）。

### ✅ 7ac481559 - fix(R280 P1): impersonation PUT scope exact matching
从 `strings.Contains` 改为精确匹配，正确防止 scope 绕过。

### ✅ 17053546f - fix(R275 P1): extract tenant_id from JWT claims
正确从 JWT claims 中提取 tenant_id，防止 metadata 被篡改导致的租户隔离失效。

### ✅ 11a2b4a9a - fix(R274 P0): gRPC interceptor metadata bypass
正确修复了 gRPC 拦截器在无 metadata 时的认证绕过漏洞。

### ✅ d391779a5 - remove duplicate tenant_id extraction
代码清理，移除重复逻辑。

---

## 汇总

**发现 1 个问题：P0 1个 / P1 0个 / P2 0个**

所有已知问题已修复，但发现1个新的P0问题。

---

## 建议

1. ⚠️ 新P0问题需要立即修复：在 `secrets.yaml` 中添加 `{fullname}-secrets` Secret
2. ✅ 建议添加 Helm template 测试，防止类似的 Secret 引用错误

---

## 补充审查 (2026-08-02 R330)

### 回归测试状态

**编译验证**:
```bash
go build ./...
```
✅ 通过 - 无编译错误

**核心服务回归测试**:
```bash
GGID_ENV=test go test -timeout 60s -count=1 \
  ./services/oauth/internal/service/ \
  ./services/auth/internal/service/ \
  ./services/identity/internal/scim/
```
✅ 全部通过
- oauth/internal/service: ok (5.198s)
- auth/internal/service: ok (4.501s)
- identity/internal/scim: ok (1.846s)

### 最近10次提交分析

**46e683595** - fix(P1): user_alias_handler admin/ownership authorization
- ✅ 正确性验证通过
- 添加了 admin scope 检查（platform:admin 或 tenant:admin）用于 POST/DELETE
- 允许用户自助读取自己的别名（self-service GET）
- 验证了 callerUserID == path userID 用于非管理员 GET 请求
- 正确防止了通过别名操纵进行的账户接管
- 未引入新问题

**bd1909e99** - revert: remove redundant R367 commit
- ✅ 清理提交，移除重复的修复
- 5e85bbf0d 已修复 R367 P0 问题

**36cbfb5da / 5e85bbf0d** - Helm {fullname}-secrets Secret 修复
- ✅ R367 P0 修复已完成
- bd1909e99 撤销了重复提交，保留 5e85bbf0d 作为正确修复

**f6b9a87ad** - Helm migrate job {fullname}-db-url Secret
- ✅ 添加了缺失的 DB URL secret 引用
- 防止环境变量中出现明文 DATABASE_URL

**51f8fcae8, 06a904f2a, ad8d3a9ca** - R288 P0/P2 bulk import hash 类型验证
- ✅ 已在之前的审查中验证并修复

**50d00c229, 323cf0dee** - R284/R281 Helm secrets 注入
- ✅ 已在之前的审查中验证并修复
- 上述 P0 问题（{fullname}-secrets Secret 不存在）已在此文件记录

### 本轮审查结果

**发现 0 个新问题：P0 0个 / P1 0个 / P2 0个**

### 总结
- ✅ 编译成功
- ✅ 所有核心服务测试通过
- ✅ 最近提交正确且完整
- ✅ 未引入新的 P0 安全问题
- ✅ 未引入新的 P1 问题
- ✅ 未引入新的 P2 问题
- ⚠️ 上轮审查发现的 P0 问题（Helm {fullname}-secrets Secret 不存在）仍需修复