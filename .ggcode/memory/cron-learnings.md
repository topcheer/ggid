# Cron 任务执行学习记录

## 2026-07-30 深度功能审查

### 执行环境
- 工作目录: /Volumes/new/ggai/ggid
- K8S 集群: ggid namespace
- 所有服务状态: 健康（ggid-auth 3/3, ggid-gateway 1/1, ggid-identity 1/1, ggid-oauth 1/1, ggid-policy 1/1, ggid-org 1/1, ggid-console 1/1）

### 审查领域
1. 架构债务扫描
2. RBAC 权限验证
3. 分层配置体系
4. Passkey/WebAuthn 集成
5. OAuth Client 管理

### 关键发现

#### P0: Console 登录页面硬编码 Tenant ID
- **文件**: `console/src/app/login/page.tsx:74,80`
- **问题**: tenant UUID "fb44ca98-2a8a-498b-a9b2-00fc014524ce" 硬编码
- **风险**: 多租户隔离绕过、tenant UUID 变更需修改代码
- **建议**: 从环境变量 `GGID_PLATFORM_TENANT_ID` 读取

#### P1: Gateway RBAC 硬编码路径列表
- **文件**: `services/gateway/internal/middleware/rbac.go:36-63`
- **问题**: 27 个管理员端点路径硬编码在 `defaultAdminPrefixes`
- **风险**: 新增端点需修改代码，运维负担重
- **建议**: 优先使用 Policy 服务的动态 RBAC 解析

#### P1: RBAC 路径重写时机问题
- **文件**: `services/gateway/internal/middleware/rbac.go:58-61`
- **问题**: RBAC 检查在路径重写之前，需同时维护两处路径列表
- **建议**: 在 RBAC 检查前先执行路径标准化

#### P1: 分层配置缺少文档
- **文件**: `services/auth/internal/service/hierarchical_*.go`
- **问题**: 配置合并规则（app → tenant → instance → env）未文档化
- **建议**: 添加 ADR 文档和配置调试端点

### 代码质量评估
- ✅ 无 TODO/FIXME 注释
- ✅ 测试覆盖充分
- ✅ 错误处理完善
- ✅ 安全特性到位

### 最近提交分析
最近 10 次提交专注于：
- P0/P1 安全修复（TenantResolver 循环、Session IDOR、JWT alg confusion）
- 并发安全（map panic、goroutine leak）
- CI 对齐（Go 1.26, golangci-lint v2）

### 红线合规检查
- ✅ 未执行 git stash/checkout/reset
- ✅ 未修改其他 agent 代码
- ⚠️ 发现未提交变更：`.ggcode/memory/cron-learnings.md`, `docs/research/iam-security-audit-2026-07-23.md`, `sdk/node`, `sdk/python`

### 下一步行动
1. 立即修复 Console 硬编码 tenant ID（需要环境变量配置）
2. 添加 RBAC 动态解析监控告警
3. 完善分层配置文档

### 部署命令参考
```bash
docker buildx build --no-cache --platform linux/amd64 -f console/Dockerfile -t registry.iot2.win/ggid/console:latest --push .
kubectl rollout restart deployment/ggid-console -n ggid
```

### 报告输出
`docs/iam-review-2026-07-30.md`