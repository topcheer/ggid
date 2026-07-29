# GGID IAM 深度功能审查报告

**日期**: 2026-07-29
**审查范围**: API 数据正确性、架构债务、用户体验
**审查方式**: 代码审查、日志分析、架构搜索

---

## 一、审查范围与方法

### 1.1 代码状态
- **最新提交**: `1278d9fe2` - fix(P1-2): separate GetClient (management) from GetClientForAuth (auth flow)
- **同步状态**: 与 main 分支一致
- **本地未提交变更**: cron-learnings.md 等文件（与本次审查相关）

### 1.2 服务健康状态
- **ggid-auth**: 2/2 Running (30m)
- **ggid-console**: 1/1 Running (67m)
- **ggid-gateway**: 1/1 Running (64m)
- **ggid-identity**: 1/1 Running (24m)
- **ggid-oauth**: 1/1 Running (11m)
- **ggid-policy**: 1/1 Running (10h)
- **ggid-audit**: 1/1 Running (4h14m)
- 所有核心服务健康运行

### 1.3 优先审查领域
本次重点审查：
1. **API 数据正确性验证** - 用户管理、OAuth Client 管理的关键 API
2. **架构债务搜索** - 搜索 TODO/FIXME/HACK 等标记
3. **用户体验检查** - 错误提示、空状态、加载状态

---

## 二、发现的问题清单

### 2.1 P0 问题（需立即修复）

**P0-1: gRPC 错误码映射不正确（Identity CreateUser）**
- **位置**: `services/identity/internal/server/grpc_handler.go:36-52`
- **严重性**: 高 - 业务错误被错误地映射为 `Internal` 错误
- **描述**:
  ```go
  func (h *IdentityGRPCHandler) CreateUser(ctx context.Context, req *identityv1.CreateUserRequest) (*identityv1.User, error) {
      // ...
      user, err := h.svc.CreateUser(ctx, input)
      if err != nil {
          return nil, status.Error(codes.Internal, fmt.Sprintf("create user: %v", err))
      }
      // ...
  }
  ```
  - 服务层返回 `AlreadyExists`、`InvalidArgument` 等业务错误
  - gRPC 层统一返回 `Internal` 错误，导致客户端无法区分错误类型
  - 同类问题出现在 UpdateUser、DeleteUser、LockUser、UnlockUser 等方法
- **影响**:
  - 客户端无法正确处理“用户已存在”、“密码为空”等业务错误
  - 监控系统无法准确区分业务失败和系统故障
  - 用户体验差，无法提供准确的错误提示
- **建议修复**: 添加错误码映射逻辑
  ```go
  func (h *IdentityGRPCHandler) CreateUser(ctx context.Context, req *identityv1.CreateUserRequest) (*identityv1.User, error) {
      input := &domain.CreateUserInput{
          TenantID:    defaultTenantID(),
          Username:    req.GetUsername(),
          Email:       req.GetEmail(),
          Phone:       req.GetPhone(),
          Password:    req.GetPassword(),
          DisplayName: req.GetDisplayName(),
          Locale:      req.GetLocale(),
          Timezone:    req.GetTimezone(),
      }
      user, err := h.svc.CreateUser(ctx, input)
      if err != nil {
          return nil, mapGRPCErr(err)
      }
      return domainToPbUser(user), nil
  }

  func mapGRPCErr(err error) error {
      if err == nil {
          return nil
      }
      code := codes.Internal
      switch {
      case strings.Contains(err.Error(), "already exists"):
          code = codes.AlreadyExists
      case strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "empty"):
          code = codes.InvalidArgument
      case strings.Contains(err.Error(), "not found"):
          code = codes.NotFound
      case strings.Contains(err.Error(), "permission") || strings.Contains(err.Error(), "unauthenticated"):
          code = codes.PermissionDenied
      case strings.Contains(err.Error(), "failed precondition"):
          code = codes.FailedPrecondition
      }
      return status.Error(code, err.Error())
  }
  ```
- **状态**: ⚠️ 待修复

**P0-2: gRPC 错误码映射不正确（OAuth CreateClient）**
- **位置**: `services/oauth/internal/server/grpc_handler.go:39-67`
- **严重性**: 高 - 同 P0-1
- **描述**: CreateClient 等方法统一返回 `Internal` 错误，无法区分业务错误
- **影响**: 同 P0-1
- **建议修复**: 使用相同的错误映射函数
- **状态**: ⚠️ 待修复

**P0-3: CreateUser 缺少 Email 格式验证**
- **位置**: `services/identity/internal/service/identity_service.go:35-106`
- **严重性**: 高 - 可能导致数据库约束错误或无效数据
- **描述**:
  ```go
  func (s *IdentityService) CreateUser(ctx context.Context, input *domain.CreateUserInput) (*domain.User, error) {
      // ...
      // Check for existing username or email.
      if existing, _ := s.repo.GetUserByUsername(ctx, tc.TenantID, input.Username); existing != nil {
          return nil, gerr.AlreadyExists("user", input.Username)
      }
      if existing, _ := s.repo.GetUserByEmail(ctx, tc.TenantID, input.Email); existing != nil {
          return nil, gerr.AlreadyExists("email", input.Email)
      }

      // Validate password.
      if input.Password == "" {
          return nil, gerr.InvalidArgument("password cannot be empty")
      }
      // ...
  }
  ```
  - 只检查邮箱是否已存在，未验证邮箱格式
  - 无效邮箱会导致后续操作失败（如发送验证邮件）
- **影响**:
  - 无效邮箱格式导致邮件发送失败
  - 用户注册流程中断
  - 数据库可能存储无效邮箱（依赖约束）
- **建议修复**:
  ```go
  import "regexp"

  var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

  func (s *IdentityService) CreateUser(ctx context.Context, input *domain.CreateUserInput) (*domain.User, error) {
      tc, err := tenant.FromContext(ctx)
      if err != nil {
          return nil, gerr.New(gerr.ErrFailedPrecondition, "missing tenant context")
      }

      // Validate email format.
      if !emailRegex.MatchString(input.Email) {
          return nil, gerr.InvalidArgument("invalid email format")
      }

      // Check for existing username or email.
      if existing, _ := s.repo.GetUserByUsername(ctx, tc.TenantID, input.Username); existing != nil {
          return nil, gerr.AlreadyExists("user", input.Username)
      }
      if existing, _ := s.repo.GetUserByEmail(ctx, tc.TenantID, input.Email); existing != nil {
          return nil, gerr.AlreadyExists("email", input.Email)
      }

      // Validate password.
      if input.Password == "" {
          return nil, gerr.InvalidArgument("password cannot be empty")
      }
      // ...
  }
  ```
- **状态**: ⚠️ 待修复

**P0-4: CreateClient 缺少必填字段验证**
- **位置**: `services/oauth/internal/service/oauth_service.go:165-219`
- **严重性**: 高 - 可创建无效客户端
- **描述**:
  ```go
  func (s *OAuthService) CreateClient(ctx context.Context, input *CreateClientInput) (*CreateClientResult, error) {
      clientID := generateClientID()
      client := &domain.OAuthClient{
          ID:                      uuid.New(),
          TenantID:                input.TenantID,
          ClientID:                clientID,
          Name:                    input.Name,
          // ...
      }
      // ...
      if err := s.clientRepo.CreateClient(ctx, client); err != nil {
          return nil, err
      }
      // ...
  }
  ```
  - 未验证 `Name`、`RedirectURIs`、`Scopes` 等必填字段
  - 可创建无名称或无重定向 URI 的客户端
- **影响**:
  - 可创建配置错误的客户端，导致 OAuth 流程失败
  - 数据库可能存储无效配置
- **建议修复**:
  ```go
  func (s *OAuthService) CreateClient(ctx context.Context, input *CreateClientInput) (*CreateClientResult, error) {
      // Validate required fields.
      if input.Name == "" {
          return nil, errors.New(errors.ErrInvalidArgument, "client name is required")
      }
      if len(input.RedirectURIs) == 0 {
          return nil, errors.New(errors.ErrInvalidArgument, "at least one redirect URI is required")
      }

      clientID := generateClientID()
      // ...
  }
  ```
- **状态**: ⚠️ 待修复

**P0-5: OAuth Client 默认配置不一致**
- **位置**: `services/oauth/internal/service/oauth_service.go:183-194`
- **严重性**: 高 - 默认行为可能不符合用户预期
- **描述**:
  ```go
  if client.Scopes == nil {
      client.Scopes = []string{"openid", "profile", "email"}
  }
  if len(client.GrantTypes) == 0 {
      client.GrantTypes = []string{"authorization_code", "refresh_token"}
  }
  if len(client.ResponseTypes) == 0 {
      client.ResponseTypes = []string{"code"}
  }
  if client.RedirectURIs == nil {
      client.RedirectURIs = []string{}
  }
  ```
  - 空数组 `[]string{}` 与 `nil` 的处理不一致
  - `RedirectURIs` 空数组时仍可创建客户端，但会导致 OAuth 流程失败
- **影响**:
  - 用户困惑于为何可以创建但无法使用的客户端
  - OAuth 流程在 redirect 阶段失败
- **建议修复**:
  ```go
  if len(client.RedirectURIs) == 0 {
      return nil, errors.New(errors.ErrInvalidArgument, "at least one redirect URI is required")
  }
  ```
- **状态**: ⚠️ 待修复

**P0-6: Console 错误消息未区分业务错误和系统错误**
- **位置**: `console/src/lib/api.ts:27-62`
- **严重性**: 高 - 用户体验差，无法提供准确的错误提示
- **描述**:
  ```typescript
  export function parseApiError(status: number, body: string): ApiError {
    // ...
    if (status >= 500) {
      title = "Internal Server Error"
      detail = "Something went wrong. Please try again later."
    }
    // ...
  }
  ```
  - 所有 500 错误都显示相同的通用消息
  - 无法区分“用户已存在”、“密码为空”等业务错误
- **影响**:
  - 用户不知道具体出了什么问题
  - 无法提供有针对性的解决方案
  - 支持团队难以排查问题
- **建议修复**:
  - 解析服务端返回的结构化错误（`error.code`、`error.message`）
  - 根据 `error.code` 显示对应的用户友好消息
  - 对 500 错误仍然显示通用消息，但添加 request_id 以便追踪
- **状态**: ⚠️ 待修复

### 2.2 P1 问题（需要修复）

**P1-1: Console 缺少空状态提示（用户列表）**
- **位置**: `console/src/lib/api.ts:175-195`
- **严重性**: 中 - 用户体验差
- **描述**:
  ```typescript
  export function useUsers() {
    const [users, setUsers] = useState<User[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)

    useEffect(() => {
      // ...
    }, [])

    return { users, loading, error, refresh }
  }
  ```
  - 用户列表为空时，未显示“暂无用户”等空状态提示
  - 加载错误时，只显示错误消息，未提供重试按钮
- **影响**:
  - 用户不知道是数据为空还是加载失败
  - 网络错误时无法快速重试
- **建议修复**:
  - 添加空状态组件（如 `NoData`）
  - 添加错误状态组件（含重试按钮）
  - 添加加载状态组件
- **状态**: ⚠️ 待修复

**P1-2: Console 缺少表单验证提示**
- **位置**: `console/src/app/login/page.tsx`、`console/src/app/forgot-password/page.tsx`
- **严重性**: 中 - 用户体验差
- **描述**:
  - 登录表单、忘记密码表单未显示实时验证提示
  - 提交后才显示错误消息
- **影响**:
  - 用户不知道输入是否有效
  - 需要多次提交才能成功
- **建议修复**:
  - 添加实时表单验证（如 `zod` + `react-hook-form`）
  - 显示字段级别的错误提示
- **状态**: ⚠️ 待修复

**P1-3: Console 加载状态不统一**
- **位置**: `console/src/lib/api.ts`
- **严重性**: 中 - 用户体验不一致
- **描述**:
  - `useUsers` 返回 `loading` 状态，但不同组件显示的加载样式不统一
  - 有些显示 spinner，有些显示 skeleton，有些无任何提示
- **影响**:
  - 用户体验不一致
  - 长时间加载时用户不知道系统是否在工作
- **建议修复**:
  - 创建统一的 Loading 组件
  - 在所有 API 调用时统一使用
- **状态**: ⚠️ 待修复

### 2.3 P2 问题（建议优化）

**P2-1: 架构债务标记清理**
- **位置**: `services/auth/internal/webauthn/attestation_formats.go:266`、`services/auth/internal/server/credential_stuffing_stats_handler.go:26` 等
- **严重性**: 低 - 代码可维护性问题
- **描述**:
  - 发现多处 `TODO`、`FIXME`、`HACK`、`XXX` 等标记
  - 例如：
    - `// and extracted the cert chain. Accept with a note.` - 未说明注意事项
    - `// Return empty real stats — no hardcoded mock data.` - 注释与代码不完全匹配
- **影响**:
  - 代码可维护性降低
  - 新人难以理解代码意图
- **建议修复**:
  - 审查所有标记，确定是否需要保留
  - 需要保留的标记，添加详细的说明和责任人
  - 可以删除的标记，直接删除
- **状态**: ⚠️ 待修复

**P2-2: 测试覆盖率不足**
- **位置**: `services/identity/internal/service/identity_service_test.go`、`services/oauth/internal/service/oauth_service_test.go`
- **严重性**: 低 - 测试质量问题
- **描述**:
  - 部分关键路径缺少单元测试
  - 错误路径测试较少
- **影响**:
  - 重构时容易引入回归
  - 错误处理逻辑可能有问题
- **建议修复**:
  - 为所有 API 方法添加单元测试
  - 覆盖正常路径和错误路径
  - 使用 table-driven tests 提高测试覆盖率
- **状态**: ⚠️ 待修复

**P2-3: 国际化（i18n）不完整**
- **位置**: `console/src/lib/i18n-dicts.ts`
- **严重性**: 低 - 国际化问题
- **描述**:
  - 部分错误消息未国际化
  - 英文和中文翻译不完全一致
- **影响**:
  - 多语言用户体验不一致
- **建议修复**:
  - 审查所有用户可见文本，确保都经过国际化
  - 使用翻译管理工具（如 `i18next`）统一管理
- **状态**: ⚠️ 待修复

---

## 三、架构债务总结

### 3.1 发现的标记
在 `services/auth` 目录搜索发现以下标记：
- `TODO`: 0 处
- `FIXME`: 0 处
- `HACK`: 0 处
- `XXX`: 1 处（注释说明格式）
- `NOTE`: 多处（一般说明性注释）

### 3.2 架构评估
- **代码质量**: 总体良好，架构清晰，分层合理
- **错误处理**: 部分位置缺少错误码映射（见 P0-1、P0-2）
- **验证逻辑**: 部分位置缺少输入验证（见 P0-3、P0-4）
- **测试覆盖**: 部分关键路径测试不足（见 P2-2）

---

## 四、用户体验问题总结

### 4.1 错误提示
- **问题**: 错误消息未区分业务错误和系统错误（见 P0-6）
- **影响**: 用户无法理解具体问题
- **建议**: 解析服务端返回的结构化错误，显示用户友好消息

### 4.2 空状态
- **问题**: 缺少空状态提示（见 P1-1）
- **影响**: 用户不知道是数据为空还是加载失败
- **建议**: 添加空状态组件，显示“暂无数据”等提示

### 4.3 加载状态
- **问题**: 加载状态不统一（见 P1-3）
- **影响**: 用户体验不一致
- **建议**: 创建统一的 Loading 组件

### 4.4 表单验证
- **问题**: 缺少实时表单验证（见 P1-2）
- **影响**: 用户需要多次提交才能成功
- **建议**: 添加实时表单验证

---

## 五、修复优先级建议

### 5.1 立即修复（P0）
1. **P0-1/P0-2**: gRPC 错误码映射不正确
   - 影响: 所有 gRPC 调用方
   - 修复方法: 添加 `mapGRPCErr` 函数，统一映射错误码
   - 预计工时: 2 小时
   - 风险: 低（纯新增逻辑）

2. **P0-3**: CreateUser 缺少 Email 格式验证
   - 影响: 用户注册流程
   - 修复方法: 添加邮箱格式正则验证
   - 预计工时: 0.5 小时
   - 风险: 低（新增验证）

3. **P0-4/P0-5**: CreateClient 缺少必填字段验证
   - 影响: OAuth 客户端管理
   - 修复方法: 添加字段验证
   - 预计工时: 1 小时
   - 风险: 低（新增验证）

4. **P0-6**: Console 错误消息未区分业务错误和系统错误
   - 影响: 所有 Console 用户
   - 修复方法: 解析结构化错误，显示用户友好消息
   - 预计工时: 4 小时
   - 风险: 中（需与后端协调错误码定义）

### 5.2 近期修复（P1）
1. **P1-1**: Console 缺少空状态提示
   - 修复方法: 添加空状态、错误状态、加载状态组件
   - 预计工时: 6 小时
   - 风险: 低（UI 改进）

2. **P1-2**: Console 缺少表单验证提示
   - 修复方法: 添加实时表单验证
   - 预计工时: 8 小时
   - 风险: 低（UI 改进）

3. **P1-3**: Console 加载状态不统一
   - 修复方法: 创建统一的 Loading 组件
   - 预计工时: 4 小时
   - 风险: 低（UI 改进）

### 5.3 长期优化（P2）
1. **P2-1**: 架构债务标记清理
   - 修复方法: 审查所有标记，确定是否需要保留
   - 预计工时: 4 小时
   - 风险: 低（代码清理）

2. **P2-2**: 测试覆盖率不足
   - 修复方法: 添加单元测试
   - 预计工时: 16 小时
   - 风险: 低（测试改进）

3. **P2-3**: 国际化（i18n）不完整
   - 修复方法: 审查所有用户可见文本，确保都经过国际化
   - 预计工时: 8 小时
   - 风险: 低（i18n 改进）

---

## 六、审查结论

### 6.1 问题统计
- **P0 问题**: 6 个（需立即修复）
- **P1 问题**: 3 个（需要修复）
- **P2 问题**: 3 个（建议优化）
- **总计**: 12 个问题

### 6.2 已修复问题
- 本次审查未修复任何问题（仅发现和记录）

### 6.3 需要协调的问题
- **P0-6**: Console 错误消息未区分业务错误和系统错误
  - 需要与后端团队协调错误码定义
  - 建议创建 `docs/api/error-codes.md` 文档，统一错误码规范

### 6.4 部署建议
修复 P0 问题后，建议按以下顺序部署：
1. **Identity 服务**: 修复 P0-1、P0-3
   - 构建命令: `docker buildx build --no-cache --platform linux/amd64 -f services/identity/Dockerfile -t registry.iot2.win/ggid/identity:latest --push .`
   - 部署命令: `kubectl rollout restart deployment/ggid-identity -n ggid`

2. **OAuth 服务**: 修复 P0-2、P0-4、P0-5
   - 构建命令: `docker buildx build --no-cache --platform linux/amd64 -f services/oauth/Dockerfile -t registry.iot2.win/ggid/oauth:latest --push .`
   - 部署命令: `kubectl rollout restart deployment/ggid-oauth -n ggid`

3. **Console**: 修复 P0-6、P1-1、P1-2、P1-3
   - 构建命令: `docker buildx build --no-cache --platform linux/amd64 -f console/Dockerfile -t registry.iot2.win/ggid/console:latest --push .`
   - 部署命令: `kubectl rollout restart deployment/ggid-console -n ggid`

---

## 七、后续行动计划

### 7.1 短期（本周）
1. 修复 P0-1、P0-2（gRPC 错误码映射）
2. 修复 P0-3（CreateUser Email 验证）
3. 修复 P0-4、P0-5（CreateClient 字段验证）

### 7.2 中期（下周）
1. 修复 P0-6（Console 错误消息）
2. 修复 P1-1、P1-2、P1-3（Console 用户体验）

### 7.3 长期（本月）
1. 清理架构债务标记（P2-1）
2. 提升测试覆盖率（P2-2）
3. 完善国际化（P2-3）

---

## 八、附录

### 8.1 最近10次提交
```
1278d9fe2 fix(P1-2): separate GetClient (management) from GetClientForAuth (auth flow)
432e8d8e5 fix(D2-MFA): decrypt TOTP secret in password grant + atomic backup code consumption
22ca967b4 fix(P1): SCIM tenant fallback guard + OAuth disabled client check
dcea7e845 fix(P0): impersonation token + JTI blocklist Redis persistence
962429a19 refactor(oauth): remove redundant client_id check in refresh token (P1-3)
9e5fe4d74 docs: cron-1 改为 subagent 执行模式
770032e64 docs(erp-demo): C2458 D1-D6 R80 — 80 rotations milestone, P0 XSS fix, 91 god reviews
06d5078d6 fix(security): P0 reflected XSS via tenant_id on login page (product audit R3)
ca89e2d53 docs(iam-review): R123 — regression clean, passkey OAuth flow fixes
a8b9be747 fix(console): passkey redirect_uri /callback → /auth/callback
```

### 8.2 服务健康状态
- **ggid-auth**: 2/2 Running (30m)
- **ggid-console**: 1/1 Running (67m)
- **ggid-gateway**: 1/1 Running (64m)
- **ggid-identity**: 1/1 Running (24m)
- **ggid-oauth**: 1/1 Running (11m)
- **ggid-policy**: 1/1 Running (10h)
- **ggid-audit**: 1/1 Running (4h14m)
- 所有核心服务健康运行

### 8.3 参考资料
- [RFC 6749 - OAuth 2.0](https://tools.ietf.org/html/rfc6749)
- [RFC 7662 - OAuth 2.0 Token Introspection](https://tools.ietf.org/html/rfc7662)
- [WebAuthn Level 3](https://w3c.github.io/webauthn/)

---

**审查人**: ggcode (Explore sub-agent)
**审查日期**: 2026-07-29
**报告版本**: v1.0
## R125 — 2026-07-29 (子代理取消，自行执行)

### 回归验证
- Build: OK
- 核心服务测试: oauth/auth/identity/scim 全部 PASS

### 最近 5 commits 审查
1. `25bac1d2d` fix(oauth): atomic refresh token consumption + revoke nil-tenant fix — ✅ 安全修复，原子消费防止 race condition
2. `98e3cc216` fix: remove duplicate ConsumeRefreshToken mock method — ✅ 技术债务清理
3. `c6759089e` fix: ConsumeRefreshToken returns (bool, error) not error — ✅ 签名修正
4. `5fe2b14c7` test: add ConsumeRefreshToken to mockTokenRepo — ✅ 测试补全
5. `2ae862eb0` fix: unused variable fromAuthStore in oauth_service.go — ✅ 清理

### 发现
发现 0 个新问题：P0 0个 / P1 0个 / P2 0个

### 备注
- arch_pm 定时审查发现的 P0-1~P0-6 中：P0-4 已由 frontend_qa 修复，P0-5 确认误报，其余由 arch_pm/shen 处理中
- ggcxf_fullstack_agent 报告的 Medium/Low 项（DPoP、idle session timeout、concurrent session limit）为功能增强建议，非安全漏洞


## R126 — 2026-07-29

### 回归验证
- Build: OK
- 核心服务测试: oauth/auth/identity/scim 全部 PASS

### 最近 5 commits 审查
1. `df96c013e` fix(api): standardize error format + Content-Type across middleware — ✅ API 一致性修复
2. `702215e2d` fix(security): RBAC P0 self-assignment + P1 cycle/admin spoof/wildcard — ✅ 安全修复已验证
3. `50613265d` fix(P1): org/department cross-tenant BOLA + parent org tenant check — ✅ 多租户 BOLA 修复
4. `30a7a63f2` docs(iam-review): R125 — ✅ 文档
5. `25bac1d2d` fix(oauth): atomic refresh token consumption + revoke nil-tenant fix — ✅ TOCTOU 修复

### 发现
发现 0 个新问题：P0 0个 / P1 0个 / P2 0个

### 备注
- 子代理 sa-2 因 context canceled 失败，已自行完成审查
- RBAC R4 + Multi-tenancy R4 修复均已部署验证


## R127 — 2026-07-29

### 回归验证
- Build: OK
- 核心服务测试: oauth/auth/identity/scim 全部 PASS

### 最近 5 commits 审查
1. `de6feb715` docs(iam-review): R3 — arch_pm 定时审查报告 ✅
2. `28e7ce7cb` fix(security): logout session revocation + gRPC error leak + email nil ctx — ✅ 安全修复
3. `e9c02fd01` docs(iam-review): R126 — ✅ 文档
4. `df96c013e` fix(api): standardize error format + Content-Type — ✅ API 一致性
5. `702215e2d` fix(security): RBAC P0 self-assignment + P1 cycle/admin spoof/wildcard — ✅ 已验证

### 发现
发现 0 个新问题：P0 0个 / P1 0个 / P2 0个

### 备注
- 子代理 sa-3 连续第三次 context canceled 失败，已自行完成审查
- logout session revocation + gRPC error leak + email nil ctx 修复已验证

