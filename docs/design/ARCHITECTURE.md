# GGID — 商用 IAM 套件架构设计

> 版本: v0.1 (Draft) | 日期: 2025-01

## 1. 产品定位

面向全球市场的 **统一身份与访问管理平台**，支持 SaaS 云端多租户和私有化部署，覆盖认证、授权、组织管理、SSO、审计全链路。

### 核心原则

- **安全第一** — 密码加密、最小权限、全审计
- **标准驱动** — 遵循 OIDC / SAML / SCIM / OAuth2 标准
- **多租户原生** — 从第一天就支持租户隔离
- **可插拔** — 认证方式、存储后端、通知渠道均可扩展
- **全球合规** — GDPR / SOC2 / 等保三级

---

## 2. 技术栈

| 层级 | 技术选型 | 说明 |
|------|---------|------|
| 后端语言 | **Go 1.23+** | 高性能、并发友好 |
| 微服务框架 | **go-kratos v2** | gRPC 优先、自动 REST 转发、内置服务发现/熔断/限流/链路追踪 |
| LDAP | **go-ldap/ldap v3** | AD/LDAP 目录访问 |
| 迁移工具 | **golang-migrate** | SQL 版本化迁移 |
| API 协议 | **gRPC + Protobuf** (内部), **REST + OpenAPI 3** (外部) | 双协议暴露 |
| 数据库 | **PostgreSQL 16** | 多租户、JSONB 策略存储、RLS |
| 缓存 | **Redis 7** | Session/Token/限流 |
| 消息队列 | **NATS JetStream** | 审计事件流、异步任务 |
| 对象存储 | **MinIO / S3** | 合规导出、附件 |
| 服务发现 | **Consul / K8s DNS** | 私有化 / SaaS 分别适配 |
| 前端 | **React 19 + TypeScript + Next.js 15** | SSR + 嵌入式登录 |
| 前端 UI | **shadcn/ui + Tailwind CSS 4** | 可定制品牌 |
| 基础设施 | **Kubernetes + Helm** | SaaS 和私有化统一编排 |

---

## 3. 微服务拆分

```
                          ┌────────────────────────────────────────┐
                          │           API Gateway (GGID-GW)         │
                          │  REST/GraphQL · Rate Limit · JWT 验证   │
                          └────────────────┬───────────────────────┘
                                           │
          ┌────────────┬────────────┬───────┴───────┬────────────┬────────────┐
          │            │            │               │            │            │
    ┌─────▼─────┐ ┌────▼─────┐ ┌───▼────┐  ┌────────▼───┐ ┌─────▼────┐ ┌────▼─────┐
    │ Identity  │ │   Auth   │ │  OAuth │  │   Policy   │ │   Org    │ │  Audit   │
    │  Service  │ │ Service  │ │  IdP   │  │  Engine    │ │ Service  │ │ Service  │
    └─────┬─────┘ └────┬─────┘ └───┬────┘  └────────┬───┘ └─────┬────┘ └────┬─────┘
          │            │            │               │            │            │
    ┌─────▼─────┐ ┌────▼─────┐     │          ┌────▼───┐  ┌─────▼────┐  ┌────▼─────┐
    │Credential │ │ Session  │     │          │  RBAC  │  │  Tenant  │  │  Event   │
    │  Manager  │ │ Manager  │     │          │  ABAC  │  │  Dept    │  │  Stream  │
    └───────────┘ └──────────┘     │          └────────┘  │  Team    │  └──────────┘
                                   │                      └──────────┘
                              ┌────▼────┐
                              │   SCIM  │ (Provisioning — Phase 2)
                              │  Sync   │
                              └─────────┘
```

### 3.1 各服务职责

| 服务 | 职责 | 关键实体 |
|------|------|---------|
| **Identity Service** | 用户身份生命周期：注册、个人资料、状态管理 | User, Profile, IdentityLink |
| **Auth Service** | 认证核心：密码验证、MFA、Passkey、Session、Token 签发 | Credential, Session, MFADevice, Token |
| **OAuth/OIDC IdP** | OAuth2 授权服务器 + OIDC Provider + SAML IdP | Client, AuthorizationCode, IDToken, SAMLAssertion |
| **Policy Engine** | RBAC/ABAC 策略评估、权限检查 | Role, Permission, Policy, Resource |
| **Org Service** | 租户、组织架构、部门、团队、成员关系 | Tenant, Organization, Department, Team, Membership |
| **Audit Service** | 全审计日志采集、存储、查询、合规报告 | AuditEvent, ComplianceReport |

### 3.2 Phase 2 服务

| 服务 | 职责 |
|------|------|
| **Provisioning Service** | SCIM 2.0 端点、LDAP/AD 同步、HR 系统集成、JIT |
| **Risk Engine** | 异常登录检测、IP 信誉、设备指纹、风控规则 |
| **Notification Service** | Email/SMS/Webhook 通知（可对接第三方） |

---

## 4. 核心数据模型

### 4.1 Identity Service

```
users
├── id              UUID PK
├── tenant_id       UUID (多租户隔离)
├── username        VARCHAR(64) UNIQUE PER TENANT
├── email           VARCHAR(255) UNIQUE PER TENANT
├── phone           VARCHAR(20)
├── status          ENUM(active, locked, disabled, deleted)
├── email_verified  BOOLEAN
├── phone_verified  BOOLEAN
├── primary_email_id  UUID
├── avatar_url      VARCHAR(500)
├── locale          VARCHAR(10) DEFAULT 'en'
├── timezone        VARCHAR(50)
├── last_login_at   TIMESTAMPTZ
├── last_login_ip   INET
├── created_at      TIMESTAMPTZ
├── updated_at      TIMESTAMPTZ
└── deleted_at      TIMESTAMPTZ (软删除)

user_emails (多邮箱)
├── id              UUID PK
├── user_id         UUID FK
├── email           VARCHAR(255)
├── is_primary      BOOLEAN
└── verified_at     TIMESTAMPTZ

user_external_identities (第三方身份关联)
├── id              UUID PK
├── user_id         UUID FK
├── provider        VARCHAR(50)  -- google, github, wechat, saml:xxx
├── external_id     VARCHAR(255) -- 第三方唯一ID
├── metadata        JSONB
└── linked_at       TIMESTAMPTZ
```

### 4.2 Auth Service

```
credentials (凭证)
├── id              UUID PK
├── user_id         UUID FK
├── type            ENUM(password, passkey, totp, sms, email_code)
├── identifier      VARCHAR(255)  -- 对于 passkey 是 credential_id
├── secret          BYTEA         -- 加密的密码 hash / TOTP secret
├── metadata        JSONB         -- passkey 的公钥、AAGUID 等
├── enabled         BOOLEAN
├── failed_attempts INT DEFAULT 0
├── locked_until    TIMESTAMPTZ
├── created_at      TIMESTAMPTZ
└── last_used_at    TIMESTAMPTZ

sessions
├── id              UUID PK
├── user_id         UUID FK
├── tenant_id       UUID
├── token_hash      VARCHAR(128)  -- session token 的 hash
├── device_info     JSONB
├── ip_address      INET
├── user_agent      TEXT
├── expires_at      TIMESTAMPTZ
├── revoked_at      TIMESTAMPTZ
├── created_at      TIMESTAMPTZ
└── metadata        JSONB         -- MFA 已验证、当前 auth context 等

mfa_devices
├── id              UUID PK
├── user_id         UUID FK
├── type            ENUM(totp, sms, email, passkey, backup_code)
├── name            VARCHAR(100)
├── secret          BYTEA         -- 加密存储
├── metadata        JSONB
├── verified        BOOLEAN
├── created_at      TIMESTAMPTZ
└── last_used_at    TIMESTAMPTZ

refresh_tokens
├── id              UUID PK
├── user_id         UUID FK
├── session_id      UUID FK
├── client_id       UUID FK
├── token_hash      VARCHAR(128)
├── scope           TEXT[]
├── expires_at      TIMESTAMPTZ
├── rotated_from    UUID          -- token 轮转链
├── revoked_at      TIMESTAMPTZ
└── created_at      TIMESTAMPTZ
```

### 4.3 OAuth/OIDC IdP

```
oauth_clients (应用注册)
├── id              UUID PK
├── tenant_id       UUID
├── client_id       VARCHAR(64) UNIQUE
├── client_secret   BYTEA         -- 加密
├── name            VARCHAR(100)
├── type            ENUM(confidential, public)
├── grant_types     TEXT[]        -- authorization_code, client_credentials, refresh_token...
├── response_types  TEXT[]        -- code, token, id_token
├── redirect_uris   TEXT[]
├── scopes          TEXT[]
├── token_endpoint_auth_method  VARCHAR(50)
├── metadata        JSONB         -- logo, terms_url, privacy_url
├── enabled         BOOLEAN
├── created_at      TIMESTAMPTZ
└── updated_at      TIMESTAMPTZ

oauth_authorization_codes
├── id              UUID PK
├── code_hash       VARCHAR(128)
├── client_id       UUID FK
├── user_id         UUID FK
├── redirect_uri    TEXT
├── scope           TEXT[]
├── code_challenge  VARCHAR(256)  -- PKCE
├── code_challenge_method  VARCHAR(10)
├── nonce           VARCHAR(128)
├── expires_at      TIMESTAMPTZ
├── used            BOOLEAN DEFAULT FALSE
└── created_at      TIMESTAMPTZ

oidc_id_tokens (审计用，实际 JWT 无状态)
├── id              UUID PK
├── jti             VARCHAR(128) UNIQUE
├── user_id         UUID
├── client_id       UUID
├── scope           TEXT[]
├── claims          JSONB
├── expires_at      TIMESTAMPTZ
└── issued_at       TIMESTAMPTZ

saml_connections
├── id              UUID PK
├── tenant_id       UUID
├── name            VARCHAR(100)
├── entity_id       VARCHAR(255)
├── metadata_xml    TEXT          -- IdP metadata
├── sso_url         VARCHAR(500)
├── slo_url         VARCHAR(500)
├── x509_cert       TEXT
├── direction       ENUM(inbound, outbound)  -- 作为SP(入站) 或 IdP(出站)
├── name_id_format  VARCHAR(100)
├── attribute_mapping  JSONB
├── enabled         BOOLEAN
└── created_at      TIMESTAMPTZ
```

### 4.4 Policy Engine

```
roles
├── id              UUID PK
├── tenant_id       UUID
├── key             VARCHAR(64)   -- e.g. "admin", "viewer"
├── name            VARCHAR(100)
├── description     TEXT
├── system_role     BOOLEAN       -- 系统内置角色不可删
├── parent_role_id  UUID          -- 角色继承
├── created_at      TIMESTAMPTZ
└── updated_at      TIMESTAMPTZ

permissions
├── id              UUID PK
├── tenant_id       UUID
├── key             VARCHAR(128)  -- e.g. "iam:users:read"
├── name            VARCHAR(100)
├── resource_type   VARCHAR(50)   -- users, organizations, clients...
├── action          VARCHAR(50)   -- read, write, delete, admin
├── description     TEXT
└── system_perm     BOOLEAN

role_permissions
├── role_id         UUID FK
├── permission_id   UUID FK
└── conditions      JSONB         -- 可选条件 (ABAC)

user_roles
├── user_id         UUID FK
├── role_id         UUID FK
├── scope_type      ENUM(global, organization, department, team, resource)
├── scope_id        UUID          -- 关联的具体资源
├── granted_by      UUID
├── expires_at      TIMESTAMPTZ   -- 可选：限时角色
├── created_at      TIMESTAMPTZ
└── PRIMARY KEY (user_id, role_id, scope_type, scope_id)

policies (ABAC 策略 — 类 AWS IAM)
├── id              UUID PK
├── tenant_id       UUID
├── name            VARCHAR(100)
├── description     TEXT
├── effect          ENUM(allow, deny)
├── actions         TEXT[]        -- "iam:users:*"
├── resources       TEXT[]        -- "arn:ggid:iam::tenant-id:user/*"
├── conditions      JSONB         -- {"IpAddress": {"aws:SourceIp": "10.0.0.0/8"}}
├── priority        INT           -- deny 优先
└── created_at      TIMESTAMPTZ

policy_attachments
├── policy_id       UUID FK
├── principal_type  ENUM(user, role, group)
└── principal_id    UUID
```

### 4.5 Org Service

```
tenants
├── id              UUID PK
├── name            VARCHAR(100)
├── slug            VARCHAR(50) UNIQUE  -- subdomain slug
├── plan            ENUM(free, pro, enterprise)
├── status          ENUM(active, suspended, deleted)
├── settings        JSONB          -- 功能开关、密码策略、品牌配置
├── max_users       INT
├── created_at      TIMESTAMPTZ
└── updated_at      TIMESTAMPTZ

organizations
├── id              UUID PK
├── tenant_id       UUID
├── parent_id       UUID           -- 组织树
├── name            VARCHAR(200)
├── path            LTREE          -- 物化路径，高效查询子树
├── metadata        JSONB
├── created_at      TIMESTAMPTZ
└── updated_at      TIMESTAMPTZ

departments
├── id              UUID PK
├── org_id          UUID FK
├── parent_id       UUID           -- 部门树
├── name            VARCHAR(200)
├── path            LTREE
├── manager_id      UUID FK -> users
├── metadata        JSONB
└── created_at      TIMESTAMPTZ

teams
├── id              UUID PK
├── org_id          UUID FK
├── name            VARCHAR(100)
├── description     TEXT
├── created_by      UUID
└── created_at      TIMESTAMPTZ

memberships
├── id              UUID PK
├── user_id         UUID FK
├── tenant_id       UUID
├── org_id          UUID FK
├── dept_id         UUID FK (nullable)
├── team_id         UUID FK (nullable)
├── title           VARCHAR(100)   -- 职位
├── status          ENUM(active, invited, removed)
├── joined_at       TIMESTAMPTZ
└── metadata        JSONB
```

### 4.6 Audit Service

```
audit_events
├── id              UUID PK
├── tenant_id       UUID
├── actor_type      ENUM(user, api_key, system, anonymous)
├── actor_id        UUID
├── actor_name      VARCHAR(200)   -- 冗余存储，便于查询
├── action          VARCHAR(100)   -- "user.login", "role.assign"...
├── resource_type   VARCHAR(50)
├── resource_id     UUID
├── resource_name   VARCHAR(200)
├── result          ENUM(success, failure, denied)
├── ip_address      INET
├── user_agent      TEXT
├── request_id      VARCHAR(64)    -- 链路追踪
├── metadata        JSONB          -- 详细变更内容 (before/after)
├── created_at      TIMESTAMPTZ
└── (按月分区 partition by range(created_at))
```

---

## 5. 多租户架构策略

### 三级隔离架构（从第一天支持）

| 隔离级别 | 适用场景 | 实现 | 租户配置 |
|---------|---------|------|----------|
| **Level 1: 共享 DB + RLS** | 默认 SaaS 租户、Free/Pro Plan | 所有租户共享数据库，PostgreSQL Row Level Security 行级隔离 | `isolation_level = 'shared'` |
| **Level 2: 独立 Schema** | Enterprise SaaS 租户、合规要求高 | 每个租户独立 PostgreSQL Schema，同 DB 实例 | `isolation_level = 'schema'` |
| **Level 3: 独立 DB** | 大客户私有化、金融/政府合规 | 每个租户独立数据库实例 | `isolation_level = 'database'` |

### Level 1 — 共享 DB + RLS (默认)

```sql
-- 每张业务表包含 tenant_id
-- PostgreSQL Row Level Security 强制隔离
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON users
  FOR ALL
  USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

### Level 2 — 独立 Schema

```sql
-- 每个租户一个 schema: tenant_{uuid}
CREATE SCHEMA tenant_a1b2c3d4;
-- 连接时动态设置 search_path
SET search_path TO tenant_a1b2c3d4, public;
```

### Level 3 — 独立 DB

```go
// 根据租户配置获取对应的 DB 连接池
db := tenantDBManager.GetDB(tenantID)
// 每个 tenant 有独立的 *sql.DB 连接池
```

### 租户路由管理器 (TenantResolver)

```go
type TenantResolver interface {
    // 根据 tenantID 返回对应的隔离级别和数据库连接
    Resolve(ctx context.Context, tenantID uuid.UUID) (*TenantContext, error)
}

type TenantContext struct {
    TenantID        uuid.UUID
    IsolationLevel  IsolationLevel // shared | schema | database
    DB              *sql.DB         // 对应的数据库连接
    SchemaName      string          // schema 级别时的 schema 名
    Settings        json.RawMessage // 租户自定义配置
}
```

**部署模式适配：**
- **SaaS 模式** — 默认 Level 1，Enterprise 租户自动升级到 Level 2/3
- **私有化模式** — 默认 Level 3 (单租户独立 DB)，简化部署

---

## 5.1 认证后端抽象层 (Multi-Backend Auth)

用户认证支持多种后端来源，通过统一的 `AuthProvider` 接口抽象：

```go
type AuthProvider interface {
    // 认证凭据校验
    Authenticate(ctx context.Context, creds Credentials) (*AuthResult, error)
    // 该 provider 是否启用
    Type() ProviderType
    Name() string
}

type ProviderType string

const (
    ProviderLocal    ProviderType = "local"     // 本地用户（密码存储在 DB）
    ProviderLDAP     ProviderType = "ldap"     // LDAP/Active Directory
    ProviderOIDC     ProviderType = "oidc"     // 外部 OIDC Provider
    ProviderSAML     ProviderType = "saml"     // 外部 SAML IdP
    ProviderOAuth2   ProviderType = "oauth2"   // 社交登录 (Google/GitHub/微信等)
)

type AuthResult struct {
    UserID      uuid.UUID
    ExternalID  string           // 外部 IDP 中的用户标识
    Provider    ProviderType
    Attributes  map[string]any   // 从外部 IdP 同步的属性
    MustLink    bool             // 是否需要关联本地账户
    NewUser     bool             // 是否首次登录（需要 JIT 创建）
}
```

### LDAP/AD 后端

```go
type LDAPProvider struct {
    cfg      LDAPConfig
    connPool *ldap.ConnPool
}

type LDAPConfig struct {
    ServerURL     string         // ldap://dc01.corp.local:389
    BindDN        string         // 管理员 Bind DN
    BindPassword  string         // (加密存储)
    BaseDN        string         // dc=corp,dc=local
    UserFilter    string         // (&(objectClass=user)(sAMAccountName=%s))
    GroupFilter   string         // (member=%s)
    GroupBaseDN   string
    SyncInterval  time.Duration  // 全量同步间隔
    RealTimeSync  bool           // 是否启用实时同步
    AutoProvision bool           // JIT: LDAP 用户首次登录自动创建本地账户
}
```

### 认证流程

```
用户提交凭据 (username + password)
  │
  ├── 1. AuthProviderChain 按 tenant 配置的优先级遍历
  │
  ├── 2. LocalProvider.Authenticate()
  │      └── 本地 DB 密码验证 (Argon2id)
  │
  ├── 3. LDAPProvider.Authenticate() (如果启用)
  │      └── LDAP Bind 验证 + 属性查询
  │          └── AutoProvision: 首次登录自动创建本地 user 记录
  │
  ├── 4. 匹配成功 → 签发 Session + JWT
  │
  └── 5. 全部失败 → 返回认证错误（不泄露具体失败原因）
```

### 外部 IdP 登录流程 (OAuth2/OIDC/SAML)

```
用户点击 "使用 XX 登录" 按钮
  │
  ├── 1. 重定向到外部 IdP 授权页面
  ├── 2. IdP 回调到 GGID
  ├── 3. 获取用户信息 (OIDC UserInfo / SAML Assertion)
  ├── 4. 查找 user_external_identities 表
  │      ├── 已关联 → 直接登录
  │      └── 未关联 → 创建新账户或要求关联已有账户
  └── 5. 签发 Session + JWT
```

---

## 6. API 设计

### 6.1 REST API 规范

```
基础路径: https://{domain}/api/v1

认证方式:
  - Bearer Token (JWT Access Token)
  - API Key (X-API-Key header, 仅限服务端调用)

错误格式 (RFC 7807 Problem Details):
  {
    "type": "https://ggid.dev/errors/permission-denied",
    "title": "Permission Denied",
    "status": 403,
    "detail": "You don't have permission to perform this action",
    "instance": "/api/v1/users/123"
  }
```

### 6.2 核心 API 端点

```
# Identity
POST   /api/v1/users                 # 创建用户
GET    /api/v1/users                  # 列表（分页、搜索、过滤）
GET    /api/v1/users/:id              # 获取详情
PATCH  /api/v1/users/:id              # 更新
DELETE /api/v1/users/:id              # 删除（软删除）
POST   /api/v1/users/:id/lock         # 锁定
POST   /api/v1/users/:id/unlock       # 解锁
POST   /api/v1/users/:id/reset-password  # 重置密码

# Auth
POST   /api/v1/auth/register          # 注册
POST   /api/v1/auth/login             # 登录（用户名/密码）
POST   /api/v1/auth/mfa/verify        # MFA 验证
POST   /api/v1/auth/mfa/setup         # 配置 MFA
POST   /api/v1/auth/logout            # 登出
POST   /api/v1/auth/refresh           # 刷新 Token
POST   /api/v1/auth/password/forgot   # 忘记密码
POST   /api/v1/auth/password/reset    # 重置密码
GET    /api/v1/auth/sessions          # 活跃会话列表
DELETE /api/v1/auth/sessions/:id      # 撤销会话

# OAuth/OIDC (标准化端点)
GET    /.well-known/openid-configuration   # OIDC Discovery
GET    /oauth/authorize                    # 授权端点
POST   /oauth/token                        # 令牌端点
GET    /oauth/userinfo                     # UserInfo 端点
GET    /oauth/jwks                         # JWKS 端点
POST   /oauth/revoke                       # 撤销令牌
POST   /oauth/introspect                   # 令牌内省

# SAML
GET    /saml/metadata                      # SP Metadata
POST   /saml/acs                           # Assertion Consumer Service
GET    /saml/sso                           # 发起 SSO
GET    /saml/slo                           # Single Logout

# Policy / RBAC
POST   /api/v1/roles                       # 创建角色
GET    /api/v1/roles                       # 角色列表
POST   /api/v1/permissions                 # 创建权限
POST   /api/v1/users/:id/roles             # 分配角色
DELETE /api/v1/users/:id/roles/:roleId     # 移除角色
POST   /api/v1/policies/evaluate           # 权限评估
POST   /api/v1/policies/check              # 权限检查（返回 boolean）

# Organization
POST   /api/v1/tenants                     # 创建租户
GET    /api/v1/organizations               # 组织树
POST   /api/v1/organizations               # 创建组织
POST   /api/v1/departments                 # 创建部门
POST   /api/v1/teams                       # 创建团队
POST   /api/v1/invitations                 # 邀请成员
POST   /api/v1/invitations/:token/accept   # 接受邀请

# Audit
GET    /api/v1/audit/events                # 查询审计日志
GET    /api/v1/audit/export                # 导出审计报告
```

---

## 7. 安全设计

### 7.1 密码安全
- 算法: **Argon2id** (memory=64MB, iterations=3, parallelism=2)
- 密码不落库明文，仅存 hash
- 传输层强制 TLS 1.3
- 密码策略可配置（长度、复杂度、历史、过期）

### 7.2 Token 安全
- Access Token: **JWT**, RS256 签名, 短期 (15min)
- Refresh Token: 不透明随机 token, 存 Redis, 可轮转、可撤销
- PKCE 强制要求 (public client)
- Token 撤销: JWKS + Redis 黑名单 (refresh token)

### 7.3 加密
- 静态加密: PostgreSQL TDE / 磁盘级加密
- 应用层加密: 敏感字段 (TOTP secret, client_secret) 使用 AES-256-GCM
- 密钥管理: KMS (SaaS) / Vault (私有化)

### 7.4 速率限制
- 登录: 5次/分钟/IP, 10次/小时/账户
- 注册: 3次/小时/IP
- API: 按租户配额 (1000 req/min default)
- MFA: 5次/5分钟/账户

---

## 8. 项目结构

```
ggid/
├── docs/
│   └── design/
│       └── ARCHITECTURE.md          # 本文档
├── api/                             # Protobuf 定义 & OpenAPI Spec
│   ├── proto/
│   │   ├── identity/v1/
│   │   ├── auth/v1/
│   │   ├── oauth/v1/
│   │   ├── policy/v1/
│   │   ├── org/v1/
│   │   └── audit/v1/
│   └── openapi/
│       └── v1/
├── pkg/                             # 共享库
│   ├── crypto/                      # 加密工具 (Argon2id, AES, JWT)
│   ├── tenant/                      # 多租户上下文
│   ├── audit/                       # 审计事件发布器
│   ├── errors/                      # 统一错误类型
│   ├── pagination/                  # 分页工具
│   └── validator/                   # 输入校验
├── services/                        # 微服务
│   ├── gateway/                     # API 网关
│   │   ├── cmd/
│   │   ├── internal/
│   │   │   ├── handler/
│   │   │   ├── middleware/
│   │   │   └── config/
│   │   └── Dockerfile
│   ├── identity/                    # 身份服务
│   │   ├── cmd/
│   │   ├── internal/
│   │   │   ├── domain/              # 领域模型
│   │   │   ├── repository/          # 数据访问
│   │   │   ├── service/             # 业务逻辑
│   │   │   ├── handler/             # gRPC/REST handler
│   │   │   └── event/              # 事件发布
│   │   ├── migrations/              # DB 迁移
│   │   └── Dockerfile
│   ├── auth/                        # 认证服务
│   ├── oauth/                       # OAuth/OIDC 服务
│   ├── policy/                      # 策略引擎
│   ├── org/                         # 组织服务
│   └── audit/                       # 审计服务
├── console/                         # 管理控制台 (React/Next.js)
│   ├── src/
│   │   ├── app/                     # Next.js App Router
│   │   ├── components/
│   │   ├── lib/
│   │   └── hooks/
│   ├── package.json
│   └── Dockerfile
├── portal/                          # 用户自助门户 (React/Next.js)
├── sdk/                             # 多语言 SDK
│   ├── go/
│   ├── python/
│   └── typescript/
├── deploy/
│   ├── docker-compose.yaml          # 本地开发
│   ├── helm/                        # K8s 部署
│   └── terraform/                   # 基础设施
├── Makefile
└── README.md
```

---

## 9. 开发阶段规划

### Phase 1 — 基础框架 + 用户认证 + 多 Backend (4-6 周)
- [ ] 项目脚手架 (Go monorepo + go-kratos 微服务骨架)
- [ ] 共享库 (crypto, tenant context, errors, multi-tenant resolver)
- [ ] Identity Service — 用户 CRUD、注册、邮箱验证
- [ ] Auth Service — 密码登录、Session、JWT 签发
- [ ] Auth Provider 抽象层 — LocalProvider + LDAPProvider
- [ ] LDAP/AD 集成 — Bind 认证、属性同步、JIT 自动建用户
- [ ] API Gateway — 路由、JWT 验证中间件
- [ ] PostgreSQL 迁移框架 + 多级隔离 RLS
- [ ] SDK 骨架 — Go / Node.js / Java 基础客户端

### Phase 2 — RBAC + 组织管理 (3-4 周)
- [ ] Policy Engine — 角色、权限、RBAC 评估
- [ ] Org Service — 租户、组织树、部门、团队
- [ ] 成员管理 — 邀请、加入、角色分配

### Phase 3 — MFA + OAuth/OIDC (4-5 周)
- [ ] MFA — TOTP、Backup Code
- [ ] OAuth2 授权服务器 — Authorization Code + PKCE
- [ ] OIDC Provider — ID Token、UserInfo、Discovery
- [ ] 社交登录 — Google、GitHub、微信

### Phase 4 — 审计 + 管理控制台 (3-4 周)
- [ ] Audit Service — 事件采集、存储、查询
- [ ] Console — 用户管理、角色管理、组织管理 UI
- [ ] Portal — 用户自助门户

### Phase 5 — 企业特性 (4-6 周)
- [ ] SAML 2.0 (IdP + SP)
- [ ] Passkey / WebAuthn
- [ ] SCIM 2.0 端点
- [ ] 合规报告导出
- [ ] LDAP/AD 同步

### Phase 6 — 生产化 (持续)
- [ ] Helm Chart 完善
- [ ] 可观测性 (OpenTelemetry, Prometheus, Grafana)
- [ ] 性能测试 + 优化
- [ ] 安全审计 + 渗透测试

---

## 10. 开源策略

**全功能开源，Apache 2.0 协议** — 无社区版/企业版功能切割。

| 方面 | 策略 |
|------|------|
| 核心认证 + RBAC | 开源 |
| 企业 SSO (SAML)、SCIM、风控 | 开源 |
| 多租户管理、审计 | 开源 |
| 商业模式 | 提供托管 SaaS 服务 + 企业技术支持 SLA，而非功能锁定 |

竞品参考: Auth0, Okta, Keycloak, Casdoor, Logto, Authentik

---

## 11. SDK (多语言)

提供三种主流语言 SDK，覆盖后端服务集成需求：

| SDK | 语言 | 适用场景 | 核心功能 |
|-----|------|---------|----------|
| **ggid-go** | Go | GGID 自身服务、Go 后端集成 | gRPC 客户端 + REST 客户端 + 中间件 |
| **ggid-node** | Node.js / TypeScript | Node 后端、Next.js 中间件 | REST 客户端 + Express/Hono 中间件 + JWT 验证 |
| **ggid-java** | Java | Spring Boot / 企业 Java 后端 | REST 客户端 + Spring Security 集成 + Servlet Filter |

### SDK 统一接口约定

```go
// 所有 SDK 遵循一致的接口模式 (以 Go 为例)
type GGIDClient interface {
    // 验证 Access Token
    VerifyToken(ctx context.Context, token string) (*UserInfo, error)
    // 获取用户信息
    GetUser(ctx context.Context, userID string) (*User, error)
    // 权限检查
    CheckPermission(ctx context.Context, userID, resource, action string) (bool, error)
    // 刷新 Token
    RefreshToken(ctx context.Context, refreshToken string) (*TokenSet, error)
    // 创建/更新/删除用户 (服务端 API Key 认证)
    CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error)
    ListUsers(ctx context.Context, opts *ListOptions) (*PageResult[User], error)
}
```
