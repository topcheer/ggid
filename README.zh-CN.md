# GGID — 生产级身份与访问管理平台

> Apache 2.0 开源 IAM 平台。Go 1.25 微服务 + Next.js 15 管理控制台。

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8.svg)](https://golang.org)
[![Coverage](https://img.shields.io/badge/coverage-85%25-brightgreen.svg)]()

---

## GGID 是什么？

GGID 是一个开源的身份与访问管理（IAM）平台，提供认证、授权、多租户、审计等完整功能。

### 核心特性

- **认证** — 密码（Argon2id）、MFA（TOTP + WebAuthn/Passkey + 邮箱验证码）、LDAP/AD
- **社交登录** — Google、GitHub、Microsoft、Discord、LinkedIn、Slack、GitLab
- **企业 SSO** — OAuth2/OIDC、SAML 2.0
- **授权** — RBAC + ABAC 混合策略引擎
- **多租户** — PostgreSQL 行级安全（RLS）
- **API 网关** — JWT 验证、速率限制、熔断器、CORS
- **审计** — NATS JetStream 事件管道
- **管理控制台** — Next.js 15 + Tailwind CSS
- **SDK** — Go / Node.js / Java / Python
- **SCIM 2.0** — 标准用户配置协议

---

## 快速开始

```bash
# 克隆
git clone https://github.com/ggid/ggid.git
cd ggid

# 启动
cd deploy && docker compose up -d
sleep 30

# 注册用户
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"alice","email":"alice@example.com","password":"W3lcome-2025!"}'

# 登录获取 JWT
JWT=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"alice","password":"W3lcome-2025!"}' | jq -r .access_token)

# 使用 JWT
curl http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"

# 打开控制台
open http://localhost:3000
```

---

## 架构

```
┌──────────────┐    ┌──────────────────────────────────────────────┐
│  管理控制台   │    │              API 网关 (:8080)                 │
│  (Next.js)   │───▶│  JWT 验证 · 路由 · 速率限制                    │
└──────────────┘    └──────┬──────┬──────┬──────┬──────┬──────┬──────┘
                           │      │      │      │      │      │
                    ┌──────▼──┐┌──▼───┐┌─▼────┐│┌─────▼──┐┌─▼────┐
                    │Identity ││ Auth ││OAuth ││ Policy  ││ Audit│
                    │ (:8081) ││(:9001)││(:9005)││ (:8070) ││(:8072)│
                    └─────────┘└──────┘└──────┘└─────────┘└──────┘
                                         ┌──────────┐
                                         │ 组织服务  │
                                         │ (:8071)  │
                                         └──────────┘
                    ┌───────────────────────────────────────────────┐
                    │ PostgreSQL 16  ·  Redis 7  ·  NATS  ·  LDAP  │
                    └───────────────────────────────────────────────┘
```

### 微服务

| 服务 | 端口 | 职责 |
|------|------|------|
| Gateway | 8080 | API 网关，JWT 验证，路由 |
| Identity | 8081 | 用户 CRUD，SCIM 2.0 |
| Auth | 9001 | 登录，MFA，LDAP，WebAuthn |
| OAuth | 9005 | OAuth 2.1/OIDC |
| Policy | 8070 | RBAC + ABAC 策略引擎 |
| Org | 8071 | 组织架构树 |
| Audit | 8072 | 审计事件查询 |

---

## 部署

| 方式 | 适用场景 | 文档 |
|------|---------|------|
| Docker Compose | 开发、小团队 | [部署指南](docs/deploy/docker.md) |
| Kubernetes / Helm | 生产环境 | [K8s 部署](docs/deploy/kubernetes.md) |
| K3s | 轻量级 K8s | [K3s 部署](docs/deploy/k3s.md) |
| 裸金属 | 虚拟机、本地 | [裸金属部署](docs/deploy/bare-metal.md) |

---

## 开发

```bash
# 构建所有服务
make build

# 运行测试
make test

# 运行覆盖率
make coverage
```

---

## 竞品对比

| 特性 | GGID | Auth0 | Keycloak |
|------|------|-------|----------|
| 许可证 | Apache 2.0 | 专有 | Apache 2.0 |
| 自托管 | 是 | 否 | 是 |
| 多租户 | RLS | 内置 | Realms |
| 语言 | Go | Node.js | Java |
| 镜像大小 | 18-35 MB | N/A | 600MB+ |

---

## 文档

- [快速开始](docs/quickstart/5-minute-jwt.md)
- [开发者指南](docs/quickstart/developer-onboarding.md)
- [API 参考](docs/api-reference.md)
- [部署指南](docs/deploy/docker.md)
- [架构概览](docs/architecture/overview.md)

完整文档列表见 [README.md](README.md)（英文）。

---

## 许可证

Apache License 2.0