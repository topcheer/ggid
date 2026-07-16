# 后端服务内部认证设计（纵深防御）

> 状态: Proposed | 作者: IAMExpert | 日期: 2026-07-17 | 来源: arch Round 4 审计 P0 #1b
> 关联: 【IAM审计报告】2026-07-17 02:55 — all-in-one 端口暴露导致 SCIM 未授权访问

## 1. 问题

GGID 后端服务（identity/auth/oauth/policy/org/audit）**自身无任何认证**，完全信任"上游一定是 gateway"。一旦后端端口暴露（all-in-one Dockerfile 曾 EXPOSE 全部端口，已修）或集群内横向移动，攻击者可直连后端绕过 gateway 的全部安全层（JWT、租户绑定、限流、bot 检测）。

防御原则：**不信任网络拓扑，每个服务独立验证调用者身份**（零信任"never trust, always verify"在服务间调用的应用）。

## 2. 方案：HMAC 签名内部头（比共享密钥头更安全）

### 2.1 为什么不选静态共享密钥头

| 方案 | 风险 |
|------|------|
| 静态 `X-Internal-Auth: <shared-secret>` | 密钥泄露即永久失效；日志/抓包可见；无法防重放 |
| **HMAC 签名头（选择）** | 密钥不出现在请求中；带时间戳防重放；可按服务区分 |

### 2.2 设计

**签名算法**：`HMAC-SHA256(secret, service_name + "|" + timestamp_unix + "|" + request_id)`

**请求头（gateway Director 注入）**：
```
X-Internal-Service: gateway
X-Internal-Timestamp: 1784217600
X-Internal-Signature: hex(HMAC-SHA256(secret, "gateway|1784217600|<request_id>"))
```

**后端验证中间件**（`pkg/middleware/internal_auth.go`，所有服务共用）：
```go
func InternalAuth(secret []byte, opts ...Option) func(http.Handler) http.Handler
```
验证逻辑：
1. 三头齐全 → 否则 403（除非在白名单路径）
2. timestamp 与服务器时间差 ≤ 120s（防重放，容忍时钟漂移）
3. 重算 HMAC 比对（constant-time）
4. 失败 → 403 `internal auth failed`，写 warn 日志（含 remote addr）

**白名单**（无需内部认证的路径）：
- `/healthz`、`/metrics`（监控探活，K8s liveness/readiness 必须可达）
- identity: `GET /api/v1/system/initialized`、`POST /api/v1/system/bootstrap`、`GET /api/v1/tenants/resolve`（未初始化阶段的公开端点，与 gateway publicPaths 对齐）

### 2.3 密钥管理

- 环境变量 `GGID_INTERNAL_SECRET`（所有服务 + gateway 同一值）
- **启动校验**：生产模式（`GGID_ENV=production`）下未设置 → 服务拒绝启动并给出明确错误；dev 模式缺省值 `dev-internal-secret` 并打 warn
- 轮换：支持双密钥窗口 `GGID_INTERNAL_SECRET` + `GGID_INTERNAL_SECRET_PREV`（验证时两个都试，轮换期 24h）
- 密钥生成：`openssl rand -hex 32`，文档写入 deploy/PRODUCTION.md

### 2.4 Gateway 注入点

`services/gateway/internal/router/router.go` 两处 proxy Director（line ~160, ~839 模式），在注入 X-Request-ID 处同步注入三个内部头。HMAC 输入中的 request_id 复用已注入的 X-Request-ID 值。

### 2.5 服务间接入清单

| 服务 | 文件 | 改动 |
|------|------|------|
| pkg/middleware | internal_auth.go（新增） | 共享中间件 + 测试 |
| gateway | router/router.go | Director 注入签名头 ×2 处 |
| identity | internal/server/http.go | mux 包 InternalAuth |
| auth | internal/server/http.go | 同上 |
| oauth | internal/server/server.go | 同上 |
| policy | internal/server/http.go | 同上 |
| org | internal/server/http.go | 同上 |
| audit | internal/server/http.go | 同上 |
| all-in-one | supervisord.conf / Dockerfile | ENV GGID_INTERNAL_SECRET 生成 |

**注意服务间直接调用**：auth→identity（HTTPIdentityClient）、oauth→identity 等内部客户端也必须在出站请求加签名头。梳理清单（审计发现）：auth 服务的 identity client（IDENTITY_SERVICE_URL）是唯一直连点，其余均经 gateway。该 client 复用 pkg/middleware 提供的 `SignInternalRequest(req)` helper。

### 2.6 测试

- 中间件单测：有效签名/无头/过期时间戳/错误签名/白名单路径/prev 密钥窗口
- gateway 注入测试：Director 后请求含三头且签名可验证
- 回归：all-in-one 启动后 curl 直连 :8081/scim/v2/Users（带任意 X-Tenant-ID）→ 403；经 gateway → 正常

## 3. 工作量

~1 天（中间件 0.5d + 各服务接入 0.5d）。无外部依赖新增。

---

## SCIM Bearer Token 设计规格

> 状态: Proposed | 关联: 审计 P1 — SCIM 对真实 IdP 不可用

### 1. 问题

Okta/Entra/Google Workspace 的 SCIM provisioning 使用**长期 bearer token**（管理员在 IdP 控制台配置），而非用户 JWT。GGID 现状：/scim/v2 走 gateway 用户 JWT → IdP 无法对接，SCIM 主场景（IdP 驱动的用户/组 provisioning）不可用。

### 2. 设计

#### 2.1 令牌模型

新表 `scim_tokens`（migration 013）：
```sql
CREATE TABLE scim_tokens (
    id          UUID PRIMARY KEY,
    tenant_id   UUID NOT NULL,
    name        TEXT NOT NULL,              -- "Okta Production"
    token_hash  TEXT NOT NULL,              -- Argon2id，明文仅创建时返回一次
    scopes      TEXT[] NOT NULL DEFAULT '{scim}',
    expires_at  TIMESTAMPTZ,                -- NULL = 永不过期（可配）
    last_used_at TIMESTAMPTZ,
    revoked_at  TIMESTAMPTZ,
    created_by  UUID NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- RLS 同其他表
```

令牌格式：`ggid_scim_<base64url(32 bytes)>`（前缀便于识别与泄露扫描）。

#### 2.2 认证流程

1. 管理员在 console 创建 token（POST /api/v1/identity/scim/tokens，需 JWT + admin）
2. IdP 侧配置：SCIM Base URL = `https://<ggid-gateway>/scim/v2`，HTTP Header Authorization = `Bearer ggid_scim_...`
3. 请求链路：
   - gateway 识别 `Authorization: Bearer ggid_scim_*` → 不走 JWT 验证，透传
   - identity 服务 SCIM mux 前置 scimTokenAuth 中间件：查 token_hash 验证（Argon2id）、检查 revoked/expires、写 last_used_at、注入 tenant context（token 绑定租户，**忽略 X-Tenant-ID 头** — 防跨租户）
4. 内部认证中间件顺序：healthz 白名单 → scimTokenAuth（仅 /scim/v2 路径）→ InternalAuth（其余路径）

**关键点**：SCIM token 认证的请求跳过 InternalAuth（因为不来自 gateway），但 tenant 来自 token 本身，比 gateway 注入更严格。

#### 2.3 管理 API（identity 服务）

| Method | Path | 说明 |
|--------|------|------|
| POST | /api/v1/identity/scim/tokens | 创建（JWT admin，返回一次明文） |
| GET | /api/v1/identity/scim/tokens | 列表（不含 hash，含 last_used_at） |
| DELETE | /api/v1/identity/scim/tokens/{id} | 吊销（软删，revoked_at） |
| POST | /api/v1/identity/scim/tokens/{id}/rotate | 轮换（新明文一次返回，旧 token 24h 宽限） |

console 页面：settings/scim-tokens（创建/列表/吊销/复制一次明文引导 + IdP 配置指引：Okta/Entra 分步截图位）。

#### 2.4 安全要求

- 验证用 Argon2id（复用 pkg/crypto HashPassword/VerifyPassword）
- 全量请求写 audit 事件（scim.provision.create/update/delete，actor=token name）
- 速率限制：gateway 对 /scim/v2 独立限流桶（防 IdP 配置错误打爆）
- token 泄露应急：DELETE 立即吊销（软删即时生效，无缓存）

#### 2.5 测试

- token 生命周期：创建→验证→轮换→吊销→过期
- 跨租户拒绝：tenant A 的 token + X-Tenant-ID: tenant B → 操作落在 tenant A（忽略头）
- IdP 兼容性 smoke：Okta SCIM 2.0 标准 payload（User/Group CRUD + PATCH）
- 回归：内部认证中间件与 scimTokenAuth 顺序正确（healthz 通、scim token 通、无认证 403）

### 3. 工作量

~1.5 天（migration+repo 0.5d + 中间件+管理 API 0.5d + console 页面前端协同 0.5d）。
