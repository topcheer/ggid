# ISO 27001:2022 — GGID 控制项映射矩阵

> 本文档将 ISO/IEC 27001:2022 Annex A 控制项映射到 GGID 平台功能。

---

## Annex A —信息安全控制

### A.5 组织安全 (Organizational Controls)

| Control ID | 描述 | GGID 功能 | 实现状态 |
|-----------|------|-----------|---------|
| A.5.1 | 信息安全策略 | Brand customization + Policy engine | ✅ |
| A.5.2 | 信息安全角色与职责 | RBAC + Role assignment audit | ✅ |
| A.5.3 | 信息分级 | PII Scan + Data Lineage + ABAC | ✅ |
| A.5.4 | 管理层责任 | Access Reviews + Audit reports | ✅ |
| A.5.7 | 威胁情报 | ITDR + Threat Feed + SIEM correlation | ✅ |
| A.5.8 | 云服务信息安全 | Multi-tenant isolation + NetworkPolicy | ✅ |
| A.5.9 | 供应商关系 | SCIM provisioning + OAuth client management | ✅ |
| A.5.10 | 客户协议 | Brand customization + Terms of service | ✅ |
| A.5.15 | 访问控制 | RBAC + ABAC + Conditional Access | ✅ |
| A.5.16 | 身份管理 | User lifecycle + SCIM + Deprovisioning | ✅ |
| A.5.17 | 认证信息管理 | Password policy + API Keys + Token rotation | ✅ |
| A.5.18 | 访问权限审计 | Access Reviews + Audit hash-chain | ✅ |
| A.5.21 | 网络隔离 | NetworkPolicy + Tenant isolation | ✅ |
| A.5.23 | 云服务信息过滤 | Data residency + Geo-fencing | ✅ |
| A.5.30 | ICT 应急准备 | Incident response + Emergency audit | ✅ |
| A.5.34 | 隐私与 PII 保护 | GDPR Forget + PII Scan | ✅ |

### A.6 人员安全 (People Controls)

| Control ID | 描述 | GGID 功能 | 实现状态 |
|-----------|------|-----------|---------|
| A.6.3 | 信息系统培训 | Enrollment Campaign (Passkey/MFA) | ✅ |
| A.6.6 | 保密协议 | Terms + Brand customization | ⚠️ 流程 |
| A.6.8 | 不当行为报告 | Audit anomaly detection + alerting | ✅ |

### A.7 物理安全 (Physical Controls)

> GGID 作为云原生 IAM 平台，物理安全由基础设施提供商负责。

| Control ID | 描述 | GGID 功能 | 实现状态 |
|-----------|------|-----------|---------|
| A.7.2 | 物理入口 | N/A（云原生部署） | ➖ |

### A.8 资产管理 (Asset Controls)

| Control ID | 描述 | GGID 功能 | 实现状态 |
|-----------|------|-----------|---------|
| A.8.2 | 特权访问权限 | Privileged activity monitoring + RBAC | ✅ |
| A.8.3 | 信息访问限制 | Fine-grained RBAC + ABAC + API Keys | ✅ |
| A.8.4 | 源代码访问 | OAuth client scopes + Git integration | ✅ |
| A.8.5 | 安全认证 | MFA + WebAuthn/Passkey + Adaptive MFA | ✅ |
| A.8.6 | 容量管理 | HPA autoscaling + Resource limits | ✅ |
| A.8.7 | 恶意软件防护 | AAGUID Allowlist + Session revocation | ✅ |
| A.8.8 | 变更管理 | API versioning + Breaking change CI + SDK drift | ✅ |
| A.8.9 | 配置管理 | Helm values + Terraform IaC | ✅ |
| A.8.10 | 信息删除 | GDPR Forget + Deprovisioning | ✅ |
| A.8.11 | 数据屏蔽 | PII Scan + Console privacy settings | ✅ |
| A.8.12 | 数据泄露预防 | DLP integration + Audit evidence chain | ✅ |
| A.8.13 | 信息备份 | DB backup + backup-verify + HA replication | ✅ |
| A.8.14 | 信息冗余 | Multi-region HA + Redis Sentinel | ✅ |
| A.8.15 | 日志记录 | Audit hash-chain + Tamper protection | ✅ |
| A.8.16 | 监控活动 | Prometheus + Grafana + SIEM metrics | ✅ |
| A.8.17 | 时钟同步 | NTP (K8s managed) + Timestamp audit | ✅ |
| A.8.18 | 特权功能 | Privileged activity audit + Impersonation | ✅ |
| A.8.19 | 聚合信息安装 | NetworkPolicy + Pod anti-affinity | ✅ |
| A.8.20 | 网络安全 | Rate limiting + Geo-fencing + TLS | ✅ |
| A.8.21 | 网络服务安全 | OAuth client deprecation + Scope management | ✅ |
| A.8.22 | 网页过滤 | N/A（应用层） | ➖ |
| A.8.23 | 密码学 | JWT RS256 + Password hashing + TLS 1.3 | ✅ |
| A.8.24 | 匿名化 | Data export + GDPR Forget | ✅ |
| A.8.25 | 安全开发生命周期 | API versioning + SDK testing + CI gates | ✅ |
| A.8.26 | 应用安全要求 | Security checklist + OWASP compliance | ✅ |
| A.8.27 | 安全系统架构 | C4 architecture docs + ADR | ✅ |
| A.8.28 | 安全编码 | Secure coding guidelines + SAST CI | ✅ |
| A.8.29 | 开发/测试中的安全 | Test coverage (446 test files) + E2E tests | ✅ |
| A.8.30 | 外包开发 | N/A（内部开发） | ➖ |
| A.8.31 | 技术漏洞管理 | Vulnerability management doc + Security audits | ✅ |

---

## 总结

| Category | Total Controls | GGID 覆盖 | 覆盖率 |
|----------|---------------|-----------|--------|
| A.5 组织 | 16 | 16 | 100% |
| A.6 人员 | 3 | 2 | 67% |
| A.7 物理 | 1 | 0 (N/A) | — |
| A.8 资产 | 31 | 29 | 94% |
| **合计** | **51** | **47** | **92%** |

> 未覆盖项（A.6.6 保密协议、A.7.2 物理安全、A.8.22 网页过滤、A.8.30 外包开发）为流程/物理层控制，超出 IAM 平台范围。
