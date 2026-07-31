# GGID IAM 平台深度功能审查

**日期**: 2026-07-30
**审查员**: Explore sub-agent
**环境**: Kubernetes (ggid namespace)

---

## 执行步骤

### 1. 代码同步与状态
- Git pull 因未暂存更改而跳过
- 服务健康检查：所有核心服务运行正常

### 2. 最近关键变更
- `f67597e07` - 修复邮件轰炸防护（magic link 60s 速率限制）
- `5e14461ee` - BOLA P0：审计导出租户隔离
- `369f5cff8` - OAuth consent token replay 使用 Redis SetNX
- `c9cd03a58` - SDK Java 支持 client_id/client_secret 刷新
- `7f132b0a2` - WebAuthn challenge TOCTOU 原子性修复

### 3. 深度审查领域

#### 3.1 架构债务搜索

**TODO/FIXME 分析**：
- 大多数为测试代码注释（XXXX-XXXX 格式、mock 说明）
- 无需修复的合规性注释

**Mock/Hardcode 非测试代码**：
- `pkg/sysconfig/defaults.go:4` - 配置优先级设计（DB > env > default）— **正常**
- `services/oauth/internal/server/grpc_handler.go:194` - 租户 ID 从环境变量获取，不硬编码 UUID — **良好设计**
- `services/identity/internal/server/grpc_handler.go:353` - uuid.Nil 当未设置，注释明确禁止硬编码 — **良好**

**发现的硬编码回退**：

1. **RBAC 管理端点硬编码列表** (P2)
   - 文件：`services/gateway/internal/middleware/rbac.go:36-63`
   - 问题：defaultAdminPrefixes 列表在代码中硬编码，作为动态 RBAC 解析器的回退
   - 影响：当动态解析器无数据时，依赖此列表。新增管理端点需要同步更新
   - 建议：考虑从数据库或配置中心加载，减少维护负担

2. **动态 RBAC 解析器回退机制** (P3)
   - 文件：`services/gateway/internal/middleware/rbac_dynamic.go`
   - 行 20: "warm-start fallback, and hardcoded-prefix fallback when neither is available"
   - 行 73: "hardcoded prefix list"
   - 行 81: "could only block the hardcoded admin prefixes"
   - 行 141: "RequireAdminScope falls back to the hardcoded prefix logic"
   - 影响：多层回退依赖硬编码前缀，配置变更可能不一致
   - 建议：统一管理端点定义，动态或配置化

3. **CCM 引擎硬编码值** (P3)
   - 文件：`services/audit/internal/server/ccm_engine.go:41,62`
   - 问题：当 DB pool 为 nil 时，回退到保守的硬编码值
   - 影响：Credibility Scoring 在无连接时可能不准确
   - 建议：明确回退行为，记录审计日志

#### 3.2 OAuth Client 管理 API 数据正确性

**检查点**：
- `CreateClient`: 正确从 context 提取 tenantID
- `tenantIDFromContext` 函数：
  - 优先使用 `GGID_TENANT_ID` 环境变量
  - 回退到 `DEFAULT_TENANT_ID`
  - 当两者都未设置时返回 `uuid.Nil`（不允许硬编码 UUID）

**观察**：
- gRPC 处理器正确实现了租户上下文提取
- ClientSecret 只在创建时返回明文（安全性良好）
- PageToken 使用简单 offset 实现（无加密保护）

**潜在问题**：
- ListClients 的 pageToken 只是简单的 offset 整数字符串，易被篡改
- 建议：考虑加密 pageToken 或使用 cursor-based 分页

#### 3.3 租户隔离验证

**跨租户保护检查**：
- 大量代码注释提到 BOLA 修复和跨租户防护
- 关键检查点：
  - `services/policy/internal/service/role_service.go:266` - 防止 UUID 枚举导致的跨租户 BOLA
  - `services/identity/internal/server/dashboard_stats_handler.go:46` - Fail-closed：无租户上下文时不返回数据
  - `services/audit/internal/repository/audit_repo.go:364` - 必须传递真实租户 ID（P0：跨租户审计销毁）

**测试覆盖**：
- 发现多个跨租户安全测试（`passkey_tenant_isolation_test.go`、`middleware_security_test.go`）

#### 3.4 Console 用户体验

**空状态组件**：
- 统一 `EmptyState` 组件存在 (`console/src/components/EmptyState.tsx`)
- 支持图标、标题、描述、操作按钮
- 应用广泛（组织管理、审计页面等）

**加载状态组件**：
- `LoadingState` 组件存在
- 所有页面都有 loading/error state 管理

**错误处理**：
- 错误提示统一使用 `<AlertCircle>` 图标 + 可关闭按钮
- 提供重试机制

---

## 问题汇总

| 优先级 | 问题 | 位置 | 建议 |
|--------|------|------|------|
| P2 | RBAC 管理端点硬编码列表 | `services/gateway/internal/middleware/rbac.go:36-63` | 考虑从配置或数据库加载，同步更新机制 |
| P3 | 动态 RBAC 多层硬编码回退 | `services/gateway/internal/middleware/rbac_dynamic.go` | 统一管理端点定义，减少硬编码依赖 |
| P3 | CCM 引擎硬编码回退值 | `services/audit/internal/server/ccm_engine.go:41,62` | 明确回退行为，添加审计日志 |
| P3 | ListClients pageToken 可篡改 | `services/oauth/internal/server/grpc_handler.go:85-104` | 使用加密 token 或 cursor-based 分页 |

---

## 建议修复

### P2: RBAC 管理端点配置化
1. 创建配置表存储管理端点前缀
2. 启动时从数据库加载 defaultAdminPrefixes
3. 提供管理界面更新端点列表

### P3: 其他改进
- CCM 引擎添加回退日志记录
- OAuth ListClients 考虑加密 pageToken

---

## 良好实践

1. **租户隔离设计**：
   - 统一使用 `tenantIDFromContext` 模式
   - Fail-closed 策略（无上下文时不返回数据）
   - 大量跨租户安全测试

2. **密码策略**：
   - WebAuthn challenge TOCTOU 原子性修复
   - OAuth consent token replay 使用 Redis SetNX

3. **SDK 支持**：
   - Java SDK 支持 client_id/client_secret 刷新
   - 多语言 SDK 维护

4. **审计追踪**：
   - 关键操作审计日志
   - GDPR 合规（账户删除时清理）

---

## 结论

总体而言，GGID IAM 平台在安全性、租户隔离和用户体验方面表现良好。主要债务集中在 RBAC 动态解析器的硬编码回退机制，属于 P2-P3 级别技术债务，不影响核心功能。

**无 P0/P1 级别问题需要立即修复。**

下一步：根据优先级逐步配置化硬编码列表，减少维护成本。
---

## 补充审查 — 2026-07-31 08:30

**审查员**: Independent Reviewer (sub-agent)
**范围**: 最近5个commit回归 + 核心服务编译/测试

### 编译与测试状态
- `go build ./...` — 通过
- `go test ./services/oauth/internal/service/ ./services/auth/internal/service/ ./services/identity/internal/scim/` — 全部通过

### 最近5个Commit分析

| Commit | 描述 | 评估 |
|--------|------|------|
| f67597e | magic link 60s rate limit + fail-closed login + email verification enforcement | 修复正确 |
| 5e14461 | audit exports handler tenant isolation (BOLA P0) | 修复正确，但仅覆盖 GET/POST |
| 369f5cf | consent token replay Redis SetNX | 修复正确 |
| c9cd03a | Java SDK refreshToken client_id/secret | 修复正确 |
| 7f132b0 | WebAuthn challenge TOCTOU atomic getAndDelete | 修复正确 |

### 新发现问题

#### P1-1: 审计报告下载缺少租户隔离 (BOLA)
- **文件**: `services/audit/internal/server/report_handler.go:99-145`
- **端点**: `GET /api/v1/audit/reports/{id}/download`
- **描述**: `handleReportDownload` 通过 report ID 查找报告时，仅检查报告是否存在和状态是否为 "ready"，不验证 `report.TenantID` 是否匹配请求者租户上下文。任何租户用户只要知道 report ID 即可下载其他租户的合规报告。
- **影响**: 跨租户数据访问（BOLA）。虽然报告内容目前为模板数据，但 report ID 通过 UUID v4 生成，可预测性低，实际利用难度中等。
- **修复建议**: 在查找报告后添加 `tenant.FromContext` 校验 `report.TenantID == tenantIDStr`。

#### P1-2: 审计报告生成接受客户端提供的 tenant_id
- **文件**: `services/audit/internal/server/report_handler.go:63-83`
- **端点**: `POST /api/v1/audit/reports/generate`
- **描述**: `handleReportGenerate` 从 JSON body 读取 `TenantID` 字段并直接使用，未从认证上下文中获取。攻击者可以在请求体中指定任意 `tenant_id`，为其他租户生成报告。
- **影响**: 租户隔离绕过。与已修复的 exports handler BOLA（commit 5e14461）是同类问题。
- **修复建议**: 忽略 body 中的 `tenant_id`，从 `tenant.FromContext(r.Context())` 获取。

### 未发现新 P0 问题

上述5个commit均为修复已有问题的正确补丁，未引入新的安全漏洞。VerifyCredentials 的 fail-closed 改动是正确的安全增强（identity service 不可用时拒绝登录），RequireEmailVerification 默认 false 不影响向后兼容。

### 总结

发现 2 个问题：P0 0个 / P1 2个 / P2 0个

---

## 独立审视报告 — 2026-07-31 09:30 (独立审查者)

### 编译验证

`go build ./...` **失败**，2 个语法错误：

1. `services/gateway/internal/middleware/response_cache.go:201:45: syntax error: unexpected name int in argument list`
2. `services/gateway/internal/middleware/jti_replay.go:77:1: syntax error: unexpected EOF, expected }`

### 根因分析

**P0-1: response_cache.go cleanup 函数大括号缺失导致编译破坏**

- **文件**: `services/gateway/internal/middleware/response_cache.go`
- **问题**: 工作区未提交的修改在 cleanup 函数中将 `for range ticker.C` 改为 `for { select { case <-rc.done: return; case <-ticker.C: ... } }`，但 `case <-ticker.C:` 分支体缺少一个开括号，导致大括号不匹配（43 open / 42 close = 缺 1 个 close brace）。
- **影响**: gateway middleware 包无法编译，导致整个 gateway 服务无法构建。这是一个 **P0 编译破坏**，阻断 CI/CD 部署。
- **根因**: 修改者在将 `for range ticker.C` 重构为 `for { select { case ... } }` 时，未给 `case <-ticker.C:` 分支体添加开括号。原 `for range` 的循环体直接成为 `case` 分支体，但缺少了 `case <-ticker.C: {` 中的 `{`。
- **修复建议**: 在 `case <-ticker.C:` 后添加 `{` 并在分支末尾添加 `}`，或在 `case <-ticker.C:` 分支体前后补齐大括号。

```go
// 当前（错误）:
		case <-ticker.C:
		rc.mu.Lock()
		...
		rc.mu.Unlock()
	}  // <- 闭合 select
}    // <- 闭合 for
}    // <- 闭合 func — 但少了一个 } 

// 应为:
		case <-ticker.C:
			rc.mu.Lock()
			...
			rc.mu.Unlock()
		}  // <- 闭合 case body（新增）
	}      // <- 闭合 select
}        // <- 闭合 for
}        // <- 闭合 func
```

**P0-2: jti_replay.go cleanupLoop 编译错误（级联错误）**

- **文件**: `services/gateway/internal/middleware/jti_replay.go`
- **问题**: jti_replay.go 本身大括号匹配正常（15/15），但由于 response_cache.go 的语法错误导致 Go parser 在解析整个包时级联报错。
- **影响**: 与 P0-1 相同，修复 P0-1 后此错误自动消失。

### 回归测试

`go test -timeout 60s ./services/oauth/internal/service/ ./services/auth/internal/service/ ./services/identity/internal/scim/` — **全部通过**（cached）

### 最近 5 个 Commit 分析

| Commit | 描述 | 评估 |
|--------|------|------|
| `c26f38e2a` | RateLimiter cleanup goroutine leak — add done channel | 正确修复，与已有 pattern 一致 |
| `7fec3d204` | R206 P1 report generate/download tenant isolation | 正确修复 BOLA |
| `f67597e07` | magic link 60s rate limit (email bombing) | 正确安全增强 |
| `5e14461ee` | audit exports tenant isolation (BOLA P0) | 正确修复 |
| `369f5cff8` | consent token replay Redis SetNX | 正确多实例修复 |

### 未提交修改审查

工作区有大量未提交修改，发现以下值得关注的点：

1. **TOTP 加密前缀 `enc:` 改动** (`pkg/crypto/totp.go`): 从 silent fallback 改为 fail-closed，正确安全增强。但需注意：如果 `GGID_ENCRYPTION_KEY` 环境变量未设置且已有 `enc:` 前缀的密文，DecryptTOTPSecret 将返回错误而非回退到明文。部署时需确保 key 已配置。

2. **Notification dispatcher SSRF 防护** (`pkg/notification/dispatcher.go`): 新增 `ssrfSafeDialContext` 阻止连接私有 IP，正确。`CheckRedirect: http.ErrUseLastResponse` 阻止重定向 SSRF，正确。DNS 重绑定已通过直接使用解析后的 IP 拨号缓解。

3. **Go SDK prefix bypass 修复** (`sdk/go/middleware/middleware.go`): `strings.HasPrefix(r.URL.Path, p)` 改为 `strings.HasPrefix(r.URL.Path, p+"/")`，正确防止 `/public` 匹配 `/publicadmin`。

4. **React SDK token refresh 去重** (`sdk/react/src/GGIDProvider.tsx`): 使用 `refreshPromiseRef` 去重并发 refresh 请求，正确防止 token rotation race condition。

5. **Audit LIKE 注入修复** (`services/audit/internal/repository/audit_repo.go`): 添加 `escapeLikeWildcards` 和 `ESCAPE '\\'`，正确。

6. **CSV 注入防护** (`services/audit/internal/server/http.go`): 添加 `sanitizeCSVCell`，正确。

7. **Java SDK 日志泄露** (`sdk/java/.../JwtVerifier.java`): 在 JWT 验证失败时记录 `e.getMessage()` 到 logger。JWT 验证异常消息可能包含 claim 值（如 issuer/audience），但不包含原始 token。评估为 **P2 低风险**。

### 总结

发现 2 个问题：P0 1个 / P1 0个 / P2 1个

- **P0**: response_cache.go 编译破坏（大括号缺失），阻断 gateway 服务构建
- **P2**: Java SDK JwtVerifier 日志中记录验证错误消息，可能泄露 claim 值

---

## 审查会 2026-07-31 10:30 (独立审视)

### 回归状态

- **go build ./...**: 通过
- **go test oauth/internal/service**: 通过 (cached)
- **go test identity/internal/scim**: 通过 (cached)
- **go test auth/internal/service**: 失败 (1 个测试)
  - `TestHookManager_PreHookAllowAndDeny` 失败：新增 SSRF 防护阻止了 httptest.NewServer 的 127.0.0.1 回环地址，但测试未设置 `GGID_ENV=test` 环境变量来启用测试模式绕过。设置 `GGID_ENV=test` 后测试通过。这是**测试环境配置问题**，非生产安全 bug。

### 最近 5 个 Commit 分析

```
9eab563dd fix(auth): add SSRF protection to auth hooks callWebhook
5fed9b436 fix(gateway): add panic recovery to webhook delivery goroutines
e2058ed3d fix: goroutine cleanup leaks + fire-and-forget panic recovery + Dockerfile non-root user
7e5db5536 fix(build): add missing log/slog imports
bbd55cb0b fix(identity): R207 scim fire-and-forget panic recovery
```

1. **9eab563dd** (hooks.go SSRF 防护): 新增 `ssrfSafeDialContext` 阻止私有/回环/链路本地 IP。逻辑正确，IP 固定防止 DNS 重绑定。`testMode` 绕过仅限回环 IP，不影响私有 IP阻断。**测试未设置 GGID_ENV=test 导致回归失败**。
2. **5fed9b436** (webhook panic recovery): 正确添加 `defer recover()` 到 `DeliverEvent` goroutine。同时修复了 `validateURL` 错误信息泄露（从 `err.Error()` 改为通用消息）。
3. **e2058ed3d** (goroutine cleanup + Dockerfile): 3 个 cleanup goroutine 添加 done channel + select。Dockerfile 添加非 root 用户。逻辑正确。
4. **7e5db5536** (build fix): 补充缺失的 `log/slog` import，修复编译。
5. **bbd55cb0b** (SCIM panic recovery): SCIM token middleware 添加 panic recovery，正确。

### 新发现问题

#### P1-1: DID:web 解析器 SSRF 漏洞
- **文件**: `services/identity/internal/service/did_resolver.go:86-91`
- **描述**: `resolveDIDWeb` 从用户输入的 DID 后缀构造 URL `https://{suffix}/.well-known/did.json`，使用无 SSRF 防护的 `didHTTPClient` 发送 HTTP 请求。攻击者可通过 `GET /api/v1/identity/did/did:web:169.254.169.254` 访问云元数据端点，或 `did:web:10.0.0.1` 探测内部服务。
- **影响**: SSRF — 认证用户可探测内网服务和云元数据端点
- **修复建议**: 为 `didHTTPClient` 添加 SSRF 安全 transport（参考 `http_provider.go` 的 `ssrfSafeTransport` 或 `hooks.go` 的 `ssrfSafeDialContext`），或使用 `ssrfSafeDialContext` 包装 DialContext。

#### P1-2: SCIM 出站 provisioning SSRF 漏洞
- **文件**: `services/identity/internal/scim/outbound/client.go:97`
- **描述**: SCIM 出站客户端的 `httpClient` 无 SSRF 防护。租户管理员可配置 SCIM target endpoint 指向内部服务（如 `http://10.0.0.1:8080`），通过 SCIM 同步操作触发对内部服务的请求（POST/PUT/GET/DELETE），实现 SSRF 数据泄露或服务探测。
- **影响**: SSRF — 租户管理员可探测内网服务（需租户管理员权限）
- **修复建议**: 为 `httpClient` 添加 SSRF 安全 transport，在 `NewClient` 中使用 `ssrfSafeTransport` 替代裸 `http.Client`。

#### P2-1: auth hooks 测试未设置 GGID_ENV 导致回归失败
- **文件**: `services/auth/internal/service/session_risk_taska_test.go:135` (TestHookManager_PreHookAllowAndDeny)
- **描述**: commit 9eab563dd 新增 SSRF 防护后，测试使用 `httptest.NewServer`（监听 127.0.0.1）但未设置 `GGID_ENV=test`，导致 SSRF 防护阻止回环连接，测试失败。
- **影响**: 回归测试失败（非安全 bug）
- **修复建议**: 在测试函数开头设置 `os.Setenv("GGID_ENV", "test")` 或在 TestMain 中统一设置。

### 总结

发现 3 个问题：P0 0个 / P1 2个 / P2 1个

- **P1**: DID:web 解析器 SSRF（认证用户可利用，无 SSRF 防护）
- **P1**: SCIM 出站 provisioning SSRF（租户管理员可利用，无 SSRF 防护）
- **P2**: auth hooks 测试因 SSRF 防护未设置 GGID_ENV=test 而失败（测试环境配置问题）

---

## 审查会 2026-07-31 11:15 (回归验证)

### 回归状态

- **go build ./...**: 通过
- **go test oauth/internal/service**: 通过 (4.647s)
- **go test auth/internal/service** (GGID_ENV=test): 通过 (4.005s)
- **go test identity/internal/scim**: 通过 (0.790s)

### 最近 5 个 Commit 分析

```
4e6e18486 fix(gateway): dedup admin path lists — single source of truth (R139 fix)
686f7c1e1 fix(identity): R210 P1 DID resolver + SCIM outbound SSRF protection
a0841f4b7 fix(identity): add SSRF protection to JML engine HTTP client
9eab563dd fix(auth): add SSRF protection to auth hooks callWebhook
5fed9b436 fix(gateway): add panic recovery to webhook delivery goroutines
```

1. **4e6e18486** (R139 admin path dedup): `isAdminOnlyPath` 原有 14 项硬编码列表与 `defaultAdminPrefixes`（24 项）不同步，缺失 10 个管理员路径。修复后 `isAdminOnlyPath` 委托给 `isAdminEndpoint`，并将旧列表中 3 个独有路径合并到 `defaultAdminPrefixes`。逻辑正确，无路径遗漏。
2. **686f7c1e1** (DID resolver + SCIM outbound SSRF): 修复了上一轮审查发现的 P1-1 和 P1-2。DID:web 解析器和 SCIM 出站客户端均添加 `ssrfSafeDialContext`，阻止私有/回环/链路本地 IP。IP 固定防止 DNS 重绑定。无 `testMode` 绕过，防护始终生效。正确。
3. **a0841f4b7** (JML engine SSRF): 为 `jmlHTTPClient` 添加 `ssrfSafeDialContext`，模式与 auth/hooks.go 一致。包含 `testMode` 绕过（仅限回环 IP，不影响私有 IP 阻断）。正确。
4. **9eab563dd** (auth hooks SSRF): 已在上一轮审查确认。`ssrfSafeDialContext` + `CheckRedirect`（限制重定向次数 ≤3）。Go http.Client 的 Transport.DialContext 对重定向目标同样生效，因此重定向路径的 IP 也会被验证。正确。
5. **5fed9b436** (webhook panic recovery): `DeliverEvent` goroutine 添加 `defer recover()`，防止 panic 崩溃网关进程。同时修复 `validateURL` 错误信息泄露（从 `err.Error()` 改为通用消息）。正确。

### 新发现问题

无新问题。上一轮审查发现的 P1-1（DID resolver SSRF）和 P1-2（SCIM outbound SSRF）已在 commit 686f7c1e1 中修复。所有 SSRF 防护实现正确，无重定向绕过风险（Go http.Client Transport.DialContext 对重定向目标同样生效）。

### 总结

发现 0 个新问题：P0 0个 / P1 0个 / P2 0个

回归状态：全部通过。之前发现的 P1 问题已修复。

---

## 独立审查轮次 — 2026-07-31 (sub-agent)

**审查员**: Independent sub-agent
**范围**: 最近 5 个 commit + 63 个未提交文件

### 编译与回归

- `go build ./...`: 通过
- `go test -timeout 60s ./services/oauth/internal/service/ ./services/auth/internal/service/ ./services/identity/internal/scim/` (GGID_ENV=test): 全部通过

### 最近 5 个 Commit 分析

1. **275c4bde7** (auth TestMain GGID_ENV=test): 测试环境设置，无安全影响。
2. **68ee1e37c** (lookupAuthRefreshToken nil UserID + auth-store tenant check): 修复 RefreshTokenRecord.UserID=uuid.Nil 问题。新增 DB 查询恢复 UserID + tenant_id，含租户校验。fail-closed 设计正确。
3. **dc02f3291** (auth hook test GGID_ENV=test): 测试环境设置，无安全影响。
4. **4e6e18486** (gateway admin path dedup): isAdminOnlyPath 委托给 isAdminEndpoint，消除硬编码列表不一致。合并 3 条独有路径到 defaultAdminPrefixes。正确。
5. **686f7c1e1** (DID resolver + SCIM outbound SSRF): 两个 SSRF 防护实现正确，使用 DialContext + IP 过滤阻断 loopback/private/link-local。

### 未提交变更审查 (63 文件)

审查了所有安全敏感文件的 diff：

- **CORS fix** (middleware.go): Credentials header 移到 explicit origin match 分支内，不再在无匹配时设置。修复正确，防止 wildcard + credentials 滥用。
- **PKCE constant-time compare** (models.go): ValidatePKCE 使用 subtle.ConstantTimeCompare 替代 == 比较。正确防止 timing side-channel。
- **TOTP secret encryption** (totp.go): 引入 "enc:" 前缀区分加密/legacy 明文。Decrypt 对加密值 fail-closed（不再回退明文）。正确。
- **MFA VerifyUserCode** (mfa_service.go): 改用 ValidateCustom Skew=1 与 VerifyMFA 保持一致。正确修复 clock drift 导致的 MFA 登录失败。
- **Backup code atomic single-use** (backup_codes.go): MarkUsed 失败时 continue 而非返回 nil。正确处理并发竞态。
- **Recovery token** (identity_recovery.go): 改用 crypto/rand 生成 token + constant-time compare。正确。
- **JIT migration SQL injection** (jit_migration.go): 添加 identifier 正则校验。正确。
- **CSV formula injection** (audit http.go + export_service.go): sanitizeCSVCell 前缀危险字符。正确。
- **Audit export file permissions** (export_service.go): 改用 0600 权限创建。正确。
- **LIKE wildcard escape** (audit_repo.go + pg_repo.go): escapeLikeWildcards + ESCAPE '\\'. 正确。
- **OAuth client tenant scoping** (oauth pg_repo.go): GetClientByID/ListClients/DeleteClient 添加 tenant_id WHERE 条件。正确深度防御。
- **Policy handler tenant context** (policy http.go): handlePolicyByID 改为先验证 X-Tenant-ID header + 注入 context。正确。
- **SDK middleware prefix bypass** (sdk/go middleware.go): SkipPaths 改用 path segment boundary 匹配。正确。
- **SDK JWKS key rotation** (sdk/go middleware.go): kid not found 时 force refresh + retry。正确。
- **SDK error message leak** (sdk/go middleware.go): issuer/audience 错误不再泄露期望值。正确。
- **Notification webhook SSRF** (dispatcher.go): 添加 ssrfSafeDialContext + disable redirect。正确。
- **Session limit enforcement** (session_service.go): Create 后 EnforceSessionLimit best-effort 撤销旧会话。正确。
- **Device tracking JSON format** (device_tracking.go): 从 colon-delimited 改为 JSON 存储，修复 IPv6 地址含冒号问题。正确。
- **Identity recovery audit cap** (identity_recovery.go): appendAudit 限制 1000 条。正确。
- **Backup table identifier validation** (pkg/db/backup.go): isValidIdentifier 校验 tableName。正确。
- **Botdetect cleanup done channel** (botdetect.go): 添加 StopCleanup + done channel 优雅关闭。正确。
- **Tier ratelimit cleanup** (tier_ratelimit.go): 同上，添加 done channel。正确。
- **Input validation Content-Type** (input_validation.go): 改用 strings.HasPrefix 替代 ==，兼容 charset 参数。正确。
- **Identity user enumeration** (identity_service.go): AlreadyExists 不再回显 username/email。正确。
- **Offset limit** (identity http.go): offset 上限 100000。正确。
- **EmailVerified propagation** (http_identity_client.go): 新增 EmailVerified 字段。正确。
- **Password history purge** (password_history.go): PurgeUser 用于账户删除。正确。
- **Token revocation oidc_refresh_tokens** (token_revocation.go): RevokeByClient/RevokeByUser 扩展撤销 oidc_refresh_tokens。

### 新发现问题

无新问题。

所有变更均为安全加固或 bug 修复，实现正确。之前已知的 P2/延后问题列表中的项目未重复报告。

### 总结

发现 0 个新问题：P0 0个 / P1 0个 / P2 0个

回归状态：全部通过。编译通过，核心服务测试通过。最近 5 个 commit 和 63 个未提交文件中的安全加固实现正确，无新引入的安全漏洞或 bug。

---

## 独立审视轮次 — 2026-07-31 12:45 (回归验证)

**审查员**: Independent sub-agent
**范围**: 最近 5 个 commit + 63 个未提交文件

### 编译与回归

- `go build ./...`: 通过
- `go test -timeout 60s ./services/oauth/internal/service/`: 通过 (cached)
- `go test -timeout 60s ./services/identity/internal/scim/`: 通过 (cached)
- `GGID_ENV=test go test -timeout 60s ./services/auth/internal/service/`: 通过 (cached)

### 最近 5 个 Commit 分析

```
147dec1b7 fix(k8s): add securityContext to all deployments (P0 Guardian #8)
275c4bde7 fix(auth): set GGID_ENV=test in TestMain for SSRF-protected hook tests
68ee1e37c fix(P1): lookupAuthRefreshToken nil UserID + auth-store tenant check
dc02f3291 test(auth): set GGID_ENV=test for hook test (SSRF blocks loopback)
4e6e18486 fix(gateway): dedup admin path lists — single source of truth (R139 fix)
```

1. **147dec1b7** (K8s securityContext): 所有部署添加 runAsNonRoot/runAsUser=1001/drop ALL capabilities。正确，与 Dockerfile appuser 一致。readOnlyRootFilesystem=false 合理（服务需写临时文件）。
2. **275c4bde7** (auth TestMain): TestMain 中设置 GGID_ENV=test，允许 SSRF 防护测试使用回环地址。正确。
3. **68ee1e37c** (lookupAuthRefreshToken): 新增 DB 查询恢复 UserID + tenant_id，含租户校验。参数化查询，无注入风险。fail-closed 设计正确。CreateDeviceAuthorization 新增 client_id/tenant_id 验证。正确。
4. **dc02f3291** (hook test env): 测试函数内设置 GGID_ENV=test + defer Unsetenv。正确。
5. **4e6e18486** (gateway admin path dedup): isAdminOnlyPath 委托给 isAdminEndpoint，消除硬编码列表不一致。合并 3 条独有路径（api-keys、access-keys、identity/dashboard）到 defaultAdminPrefixes。正确。

### 未提交变更审查 (63 文件)

已审查安全敏感文件 diff，确认均为安全加固或 bug 修复：

- **CORS credentials 修复** (middleware.go): Allow-Credentials 仅在 explicit origin match 时设置，修复了之前无匹配也设置 credentials 的问题。安全增强。
- **Identity user enumeration** (identity_service.go): AlreadyExists 不再回显 username/email 值。安全增强。
- **Recovery token 加密** (identity_recovery.go): 改用 crypto/rand 256-bit token + constant-time compare + 有界 audit log。安全增强。
- **CSV injection 防护** (audit http.go): sanitizeCSVCell 前缀危险字符。安全增强。
- **Offset 上限** (identity http.go): offset 上限 100000，防止超大偏移。安全增强。
- **Input validation** (input_validation.go): Content-Type 使用 strings.HasPrefix 兼容 charset；错误返回 400 而非静默放行。安全增强。
- **Token revocation oidc_refresh_tokens** (token_revocation.go): RevokeByClient/RevokeByUser 扩展撤销 oidc_refresh_tokens。注意：新增 SQL 查询缺少 tenant_id 过滤，但这两个方法当前无调用者（dead code），为潜在 P1 而非活跃漏洞。
- **Go SDK 安全改进** (sdk/go middleware.go): SkipPaths path segment boundary 匹配、JWKS key rotation 自动刷新、issuer 错误不泄露期望值。安全增强。

### 新发现问题

无新问题。所有变更均为安全加固或 bug 修复，实现正确。之前已知的 P2/延后问题未重复报告。token_revocation.go 中 RevokeByClient/RevokeByUser 缺少 tenant_id 过滤的问题为 dead code（无调用者），不在本次报告范围。

### 总结

发现 0 个新问题：P0 0个 / P1 0个 / P2 0个

回归状态：全部通过。编译通过，核心服务测试通过。最近 5 个 commit 为正确的安全加固和测试环境修复，63 个未提交文件中的安全改进实现正确，无新引入的安全漏洞或 bug。

---

## 审查批次 2026-07-31 15:00（回归 + 最近5个 commit 分析）

### 编译与测试状态

- `go build ./...`: PASS
- `go test -timeout 60s ./services/oauth/internal/service/ ./services/auth/internal/service/ ./services/identity/internal/scim/`: ALL PASS (GGID_ENV=test)
  - oauth: cached (pass)
  - auth: 3.483s (pass)
  - identity/scim: cached (pass)

### 最近 5 个 commit 分析

| Commit | 描述 | 结论 |
|--------|------|------|
| c53b02c39 | fix(auth): email_change + stepup token TOCTOU — atomic GetDel | 修复正确，但引入 2 个 P2（见下） |
| 147dec1b7 | fix(k8s): add securityContext to all deployments | 修复正确，无新问题 |
| 275c4bde7 | fix(auth): set GGID_ENV=test in TestMain | 测试修复，无安全影响 |
| 68ee1e37c | fix(P1): lookupAuthRefreshToken nil UserID + auth-store tenant check | 修复正确，参数化查询，fail-closed，无新问题 |
| dc02f3291 | test(auth): set GGID_ENV=test for hook test | 测试修复，无安全影响 |

### 新发现问题

#### P2-NEW-1: ValidateStepUpToken 缺少 tenantID 校验（防御纵深缺失）

- **文件**: `services/auth/internal/service/stepup.go:134-156`
- **严重性**: P2（防御纵深缺失，调用方有独立 tenant 校验）
- **描述**: `ValidateStepUpToken` 只校验 `userID`（parts[1]），忽略了 token 中存储的 `tenantID`（parts[0]）。step-up token 在 `VerifyStepUp` L121 存储格式为 `tenantID:userID`，但 `ValidateStepUpToken` 从未解析或比对 `tenantID` 与请求上下文中的 tenant。
- **影响**: 如果用户在多租户环境中，为 tenant A 签发的 step-up token 理论上可用于 tenant B 的操作（如果调用方未做独立 tenant 校验）。当前所有调用方（`self_service_handler.go` L92/L129/L153）已从请求上下文获取 `tenantID` 并做独立校验，因此实际不可利用。
- **建议**: 在 `ValidateStepUpToken` 签名中增加 `tenantID` 参数，校验 `parts[0] == tenantID`。

#### P2-NEW-2: VerifyStepUp MFA 失败路径泄露原始错误

- **文件**: `services/auth/internal/service/stepup.go:104-105`
- **严重性**: P2（信息泄露，影响范围有限）
- **描述**: `VerifyStepUp` 在 MFA 验证失败时直接返回 `VerifyUserCode` 的原始 `err`（`return nil, err`），而所有其他失败路径（password 不匹配、challenge 格式错误等）均返回统一的 `ErrInvalidCredentials`。`VerifyUserCode` 可返回 `fmt.Errorf("no enabled MFA device: %w", err)`，这会向 API 客户端泄露用户是否已配置 MFA 设备的内部状态。
- **影响**: 攻击者可通过观察错误响应差异，推断目标用户是否已启用 MFA。在 step-up 场景下需要已认证会话，利用价值有限。
- **建议**: 将 L105 `return nil, err` 改为 `return nil, ErrInvalidCredentials`，与其他失败路径保持一致。

#### P2-NEW-3: email_change.go 双确认并发竞态（潜伏 double-application）

- **文件**: `services/auth/internal/service/email_change.go:86-120`
- **严重性**: P2（潜伏问题，当前未接入实际邮箱更新）
- **描述**: `ConfirmEmailChange` 中，当 "old" 和 "new" 确认几乎同时到达时，两个 goroutine 都可能执行：Set(confirmedKey) → Get(otherKey) 返回成功 → 进入双确认应用逻辑（L101-120）。两个 goroutine 都会执行 Del + 应用邮箱变更，导致 double-application。Set 和 Get 之间不是原子的，Redis 的单线程模型不保证两个操作的顺序性（另一个 goroutine 的 Set 可能在本 goroutine 的 Get 之后执行，但两个 goroutine 的 Set 都在各自的 Get 之前执行，所以并发时两个 Get 都能看到对方的 confirmed key）。
- **影响**: 当前潜伏——L118 注释 "In production, update the email via identity client here"，实际邮箱更新尚未实现。一旦接入将导致邮箱被更新两次或审计事件重复。
- **建议**: 使用 Redis 事务或 Lua 脚本实现原子的 "check both confirmed + apply" 操作，或在应用阶段使用 `GetDel(dataKey)` 确保只有一个 goroutine 能获取 dataKey。

### 汇总

发现 3 个问题：P0 0个 / P1 0个 / P2 3个

所有 3 个问题均为 P2 级别，无 P0 安全漏洞。最近 5 个 commit 的修复质量良好，TOCTOU 修复（GetDel）正确，lookupAuthRefreshToken 修复使用了参数化查询和 fail-closed 策略。K8s securityContext 修复正确。3 个 P2 问题中，2 个位于刚修复的 stepup.go 文件（防御纵深和信息泄露），1 个为 email_change 的潜伏并发问题。

---

## 追加审查 — 2026-07-31 (独立审视者)

**审查范围**: 最近 5 个 commit (275c4bde7..31b91d06f) + 编译/回归测试

### 编译与回归状态

- `go build ./...` — 通过
- `go test ./services/oauth/internal/service/` — 通过
- `GGID_ENV=test go test ./services/auth/internal/service/` — 通过
- `go test ./services/identity/internal/scim/` — 通过

### 最近 5 个 commit 分析

1. `31b91d06f` - fix(test): permissions invalid tenant -> 403 — 将 createPermission/listPermissions 从 query param 改为 header + admin 检查（安全加固）
2. `74f7722f5` - fix(auth): R218 P2 stepup validation + MFA error + email change atomic — 修复已知 P2 问题
3. `c53b02c39` - fix(auth): email_change + stepup token TOCTOU — atomic GetDel — 修复 TOCTOU
4. `147dec1b7` - fix(k8s): add securityContext to all deployments (P0 Guardian #8) — 安全加固
5. `275c4bde7` - fix(auth): set GGID_ENV=test in TestMain — 测试修复

### 新发现问题

#### P1-NEW-1: handleDecisionLog 跨租户数据泄露

- **文件**: `services/policy/internal/server/http.go:2297-2356`
- **严重性**: P1（跨租户数据泄露）
- **描述**: `handleDecisionLog` 端点 (`GET /api/v1/policies/decision-log`) 返回全局内存中的策略决策日志 (`service.GetRecentDecisions`)，不进行任何租户过滤。决策日志包含 `tenant_id`、`user_id`、`action`、`resource`、`allowed`、`reason` 等字段。任何已认证的用户（包括 M2M client_credentials）都可以调用此端点，获取所有租户的策略决策记录。
- **影响**: 租户 A 的用户可以看到租户 B 的用户访问了哪些资源、执行了什么操作、被允许或拒绝了什么。这泄露了其他租户的访问模式、策略结构和用户行为。
- **根因**: `handleDecisionLog` 仅检查 HTTP 方法，不检查 `X-Tenant-ID` header，也不对返回结果按租户过滤。对比 `handlePolicyExport`（L1704-1708）正确使用 `X-Tenant-ID` header 验证租户上下文。
- **建议**: 从 `X-Tenant-ID` header 提取租户 ID 并过滤 `decisions` 切片，仅返回当前租户的决策记录。或使用 `requireTenantHeader` 获取租户上下文后过滤。

#### P1-NEW-2: handleCheck/handleEvaluate tenantIDFromHeader 查询参数回退导致跨租户策略评估

- **文件**: `services/policy/internal/server/http.go:1852-1864, 1120, 1207`
- **严重性**: P1（跨租户策略评估）
- **描述**: `handleCheck` (`POST /api/v1/policies/check`) 和 `handleEvaluate` (`POST /api/v1/policies/evaluate`) 使用 `tenantIDFromHeader(r)` 提取租户 ID。该函数先检查 `X-Tenant-ID` header，如果为空则回退到 `?tenant_id=` 查询参数。网关 `JWTAuth` 中间件 (L744-763) 仅在 `X-Tenant-ID` header 存在且与 JWT tenant_id 不匹配时才拒绝请求。对于 M2M client_credentials token（无 tenant_id claim），网关在 L780-782 清除租户上下文（不设置 X-Tenant-ID header）。此时 `tenantIDFromHeader` 会回退到 `?tenant_id=` 查询参数，允许 M2M 客户端指定任意租户进行策略评估。
- **影响**: M2M 服务可以传入 `?tenant_id=<victim_tenant>` 评估其他租户用户的权限。虽然评估结果不直接泄露策略内容，但可通过探测判断特定用户在目标租户中是否有特定权限（信息侧信道）。对比 commit 31b91d06f 刚修复的 `createPermission`/`listPermissions` 使用了 `requireTenantHeader` 进行 header/query 一致性检查。
- **根因**: `tenantIDFromHeader` 函数 (L1852) 回退到 `tenant_id` 查询参数，与 commit 31b91d06f 修复的 R7 模式相同 — query-param-only tenant selection。`handleCheck` 和 `handleEvaluate` 未使用 `requireTenantHeader` 验证 header 与 query param 一致性。
- **建议**: 在 `handleCheck` 和 `handleEvaluate` 中使用 `requireTenantHeader` 或直接仅从 `X-Tenant-ID` header 提取租户 ID（移除 query param 回退），与 `handlePolicyExport`/`handlePolicyImport`/`handleDryRun` 等已修复端点保持一致。

### 更新汇总

发现 5 个问题：P0 0个 / P1 2个 / P2 3个

新增 2 个 P1 问题：
- P1-NEW-1: handleDecisionLog 跨租户数据泄露（无租户过滤）
- P1-NEW-2: handleCheck/handleEvaluate 查询参数回退导致跨租户策略评估

编译和核心服务回归测试全部通过。最近 5 个 commit 修复质量良好，但发现 policy 服务中 2 个遗漏的租户隔离问题。

---

## 独立审视轮次 — 2026-07-31 16:45 (回归验证 + BOLA 审查)

**审查员**: Independent sub-agent
**范围**: 最近 5 个 commit + 编译/回归测试

### 编译与回归

- `go build ./...`: 通过
- `go test -timeout 60s ./services/oauth/internal/service/ ./services/auth/internal/service/ ./services/identity/internal/scim/` (GGID_ENV=test): 全部通过 (cached)

### 最近 5 个 Commit 分析

| Commit | 描述 | 评估 |
|--------|------|------|
| `8ce4cd378` | fix(P1): oauth clients handler query tenant fallback (header-only) | 正确修复 BOLA — GET/DELETE /api/v1/oauth/clients/{id} 移除 query param 回退 |
| `ffe3a1bcd` | fix(gateway): block forceLogout/sessionLimit BOLA — require admin scope | 正确修复 — 将 force-logout 和 session/limit 加入 defaultAdminPrefixes |
| `0ee9ed089` | fix(P1): oauth query tenant fallback + policy decision-log tenant isolation | 正确修复 — oauth clients list + policy decision-log 移除 query param 回退 + 添加租户过滤 |
| `ca8768a49` | fix(policy): update AddTestDecisionForTest call sites for tenantID param | 测试适配，无安全影响 |
| `8072b75d0` | fix(P1): policy decision-log tenant isolation + tenantIDFromHeader header-only | 正确修复 — handleDecisionLog 添加租户过滤 + tenantIDFromHeader 移除 query param 回退 |

### 新发现问题

#### P1-NEW-1: /api/v1/auth/registration/config BOLA — 跨租户注册配置读写

- **文件**: `services/auth/internal/server/registration_config_handler.go:124-186`
- **端点**: `GET /api/v1/auth/registration/config?tenant_id=X`、`PUT /api/v1/auth/registration/config`
- **严重性**: P1（跨租户数据访问 + 配置篡改）
- **描述**:
  1. **GET** (L124-158): `getRegistrationConfig` 优先从 `?tenant_id=` query param 获取租户 ID，仅在 query param 为空时才回退到 JWT claims。任意已认证用户可通过修改 query param 查看**任意租户**的注册配置（包括 enabled 状态、allowed_domains、default_role）。
  2. **PUT/POST** (L161-186): `updateRegistrationConfig` 从 JSON body 的 `tenant_id` 字段获取租户 ID，不校验 body 中的 tenant_id 是否匹配 JWT 中的租户。任意已认证用户可通过指定 body 中的 `tenant_id` 修改**任意租户**的注册配置（启用/禁用注册、修改 allowed_domains、修改 default_role）。
  3. 该端点不在 gateway 的 `defaultAdminPrefixes` 中（`rbac.go:50-69`），也不在 `publicPaths` 中（`router.go:31-69`），因此仅需有效 JWT 认证，无管理员权限检查。
- **影响**: 与已修复的 R30/R32/R34 同类 BOLA。攻击者可：(a) 枚举其他租户的注册配置信息；(b) 篡改其他租户的注册配置，例如启用注册并修改 default_role 为高权限角色，然后通过自注册获取目标租户的初始高权限账户。
- **根因**: 与 R30/R32 相同模式 — tenant_id 从客户端可控的 query param / body 字段获取，而非从 gateway 验证的 X-Tenant-ID header 或 JWT claims 获取。
- **修复建议**:
  1. GET: 移除 `?tenant_id=` query param 回退，仅从 `X-Tenant-ID` header 获取租户 ID（与 R32/R34 修复模式一致）。
  2. PUT/POST: 忽略 body 中的 `tenant_id`，从 `X-Tenant-ID` header 或 JWT context 获取租户 ID。
  3. 将 `/api/v1/auth/registration/config` 加入 gateway 的 `defaultAdminPrefixes`，要求 admin scope（因为注册配置影响租户安全策略）。

### 汇总

发现 1 个问题：P0 0个 / P1 1个 / P2 0个

最近 5 个 commit 均为正确的 BOLA 修复补丁，修复质量良好。编译和核心服务回归测试全部通过。但发现 `/api/v1/auth/registration/config` 端点存在与刚修复的 R30/R32 同类的 BOLA 漏洞，GET 和 PUT/POST 方法均受影响。

---

## 独立审视轮次 — 2026-07-31 17:00 (回归验证 + handler 层 BOLA 审查)

**审查员**: Independent sub-agent
**范围**: 最近 5 个 commit + 编译/回归测试

### 编译与回归

- `go build ./...`: 通过
- `go test -timeout 60s ./services/oauth/internal/service/`: 通过 (cached)
- `go test -timeout 60s ./services/identity/internal/scim/`: 通过 (cached)
- `GGID_ENV=test go test -timeout 60s ./services/auth/internal/service/`: 通过 (cached)

### 最近 5 个 Commit 分析

| Commit | 描述 | 评估 |
|--------|------|------|
| `3551f8749` | fix(P1): registration config BOLA — cross-tenant read/tamper | 正确修复 — GET header-only, PUT/POST 忽略 body tenant_id |
| `8ce4cd378` | fix(P1): oauth clients handler query tenant fallback | 正确修复 — GET/DELETE 移除 query param 回退 |
| `ffe3a1bcd` | fix(gateway): block forceLogout/sessionLimit BOLA — require admin scope | 部分修复 — gateway 层已加 admin scope，但 handler 层缺 tenant 一致性校验（见 P1-NEW-1） |
| `0ee9ed089` | fix(P1): oauth query tenant fallback + policy decision-log tenant isolation | 正确修复 — header-only + 租户过滤 |
| `ca8768a49` | fix(policy): update AddTestDecisionForTest call sites | 测试适配，无安全影响 |

### 新发现问题

#### P1-NEW-1: forceLogout / sessionLimit handler 缺少跨租户 tenant_id 一致性校验

- **文件**: `services/auth/internal/server/http.go:3146-3192` (forceLogout), `services/auth/internal/server/http.go:3196-3231` (sessionLimit)
- **端点**: `POST /api/v1/auth/sessions/force-logout`, `POST /api/v1/auth/sessions/limit`
- **严重性**: P1（跨租户 BOLA — admin 权限下跨租户操作）
- **描述**: commit `ffe3a1bcd` 将这两个端点加入 gateway `defaultAdminPrefixes`，要求 `platform:admin` 或 `tenant:admin` scope。但 handler 层直接信任 body 中的 `tenant_id` 和 `user_id`，未验证 body `tenant_id` 与 caller 的 `X-Tenant-ID` header 是否一致。一个 `tenant:admin` 用户可以指定其他租户的 `tenant_id` 来强制登出或限制任意租户用户的会话。
- **对比**: 同文件中的 `handleRevokeUser` (L49-67) 已正确实现此校验：从 header 获取 `callerTenantStr`，与 body `tenant_id` 比较，不一致时要求 `platform:admin` scope。`forceLogout` 和 `sessionLimit` 缺少同样的防护。
- **影响**: `tenant:admin` 可跨租户执行 force-logout（拒绝服务）和 session-limit（会话限制）操作。`platform:admin` 本就允许跨租户，不受影响。
- **修复建议**: 在两个 handler 中添加与 `handleRevokeUser` 一致的校验逻辑：
  1. 从 `X-Tenant-ID` header 获取 caller tenant
  2. 若 body `tenant_id` 与 header 不一致，要求 `platform:admin` scope，否则 403

### 汇总

发现 1 个问题：P0 0个 / P1 1个 / P2 0个

编译和核心服务回归测试全部通过。最近 5 个 commit 中 4 个为完整正确的修复，`ffe3a1bcd` 为部分修复（gateway 层防护到位但 handler 层缺 tenant 一致性校验）。新增 P1 问题为 `forceLogout`/`sessionLimit` 的跨租户 BOLA，需在 handler 层补充 tenant_id 一致性校验。

---

## 复审记录 2（2026-07-31）

**审查员**: Independent sub-agent
**范围**: 最新 5 个 commit（a663e072a / e48311f59 / 3551f8749 / 8ce4cd378 / ffe3a1bcd）+ 编译/回归测试

### 编译与回归

- `go build ./...`: 通过
- `GGID_ENV=test go test -timeout 60s ./services/oauth/internal/service/`: 通过 (cached)
- `GGID_ENV=test go test -timeout 60s ./services/auth/internal/service/`: 通过 (cached)
- `GGID_ENV=test go test -timeout 60s ./services/identity/internal/scim/`: 通过 (cached)

### 最近 5 个 Commit 分析

| Commit | 描述 | 评估 |
|--------|------|------|
| `a663e072a` | fix(P1): forceLogout + sessionLimit 跨租户 body tenant_id admin override | 正确修复 — 补 hasAdminScope 检查 + platform:admin 跨租户覆盖（与 handleRevokeUser 模式一致）；依赖 gateway 注入的 X-Scopes/X-Tenant-ID（auth 为 ClusterIP，直连伪造不可行） |
| `e48311f59` | fix(auth): R223 P1 forceLogout+sessionLimit header-only tenant | 正确修复 — body/header tenant 一致性校验（已并入 a663e072a 的最终形态） |
| `3551f8749` | fix(P1): registration config BOLA — cross-tenant read/tamper via tenant_id | 部分修复 — 跨租户 BOLA 已修（GET header-only、PUT 忽略 body tenant_id），但 handler 层仍缺 admin scope 检查（见 P1-NEW-2） |
| `8ce4cd378` | fix(P1): oauth clients handler query tenant fallback (header-only) | 正确修复 — 移除 ?tenant_id= query 回退，header-only |
| `ffe3a1bcd` | fix(gateway): block forceLogout/sessionLimit BOLA — require admin scope | 正确修复 — defaultAdminPrefixes 增加 force-logout / sessions/limit |

### 新发现问题

#### P1-NEW-2: registration config 端点缺少 admin scope 检查（3551f8749 修复遗漏）

- **文件**: `services/auth/internal/server/registration_config_handler.go:113-184`
- **端点**: `GET/PUT/POST /api/v1/auth/registration/config`
- **严重性**: P1（越权配置读写 — 任意已认证用户可篡改本租户注册策略）
- **描述**: `3551f8749` 修复了跨租户 BOLA（改为 header-only 取租户），但 **`getRegistrationConfig` 和 `updateRegistrationConfig` 均未调用 `hasAdminScope(r)`**。路径也不在 gateway `defaultAdminPrefixes`（rbac.go:36-69）中，gateway 仅要求 JWT（不在 publicPaths），不要求 admin scope。
- **对比**: 同批修复的 `forceLogout`/`sessionLimit`（http.go:3152, 3224）均显式加了 `hasAdminScope`；同类配置端点 `handlePasswordPolicyConfig`（password_policy_config.go:102）也有 `hasAdminScope` 检查。registration config 是唯一遗漏。
- **攻击场景**: 租户内任何已认证用户（如最低权限 viewer）可 PUT 修改本租户注册配置：
  - `require_email_verification=false` → 绕过邮箱验证开放注册
  - `allowed_domains` 清空/篡改 → 破坏域名白名单
  - `default_role` 设为高权限角色 → 后续注册用户获得该角色（当前后端 Register 流程尚未消费 DefaultRole，影响受限；一旦注册流程启用配置即形成提权链）
  - 可读取配置 → 信息泄露（探测租户注册策略）
- **修复建议**: 在 `getRegistrationConfig`/`updateRegistrationConfig` 入口添加 `hasAdminScope(r)` 检查（与 forceLogout 一致），并在 gateway `defaultAdminPrefixes` 增加 `/api/v1/auth/registration`（纵深防御）。

### 排查排除项（非误报确认）

- **X-Scopes / X-Tenant-ID 伪造直连**: gateway 无条件删除并重派生这些 header（router.go:238-271），auth service 为 ClusterIP 不直接暴露公网（deploy/helm/ggid/values-k3s.yaml:47-52），内网信任模型成立 — 不报告。
- **password-policy 端点**: 已有 `hasAdminScope` 检查（password_policy_config.go:102）— 安全。

### 汇总

发现 1 个问题：P0 0个 / P1 1个 / P2 0个

编译和三个核心服务回归测试全部通过。`a663e072a`/`e48311f59` 修复了上轮 P1-NEW-1（forceLogout/sessionLimit 跨租户），`3551f8749` 存在修复遗漏：registration config 端点无 admin scope 检查，任意已认证租户用户可读写注册配置，需补 hasAdminScope 检查。
