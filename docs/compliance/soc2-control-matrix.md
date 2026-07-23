# SOC 2 Type II — GGID 控制项映射矩阵

> 本文档将 AICPA SOC 2 Trust Service Criteria 映射到 GGID 平台的具体功能和配置。

## Trust Service Categories

| Category | Abbreviation | GGID 覆盖度 |
|----------|-------------|------------|
| Security | CC (Common Criteria) | ✅ 完整 |
| Availability | A | ✅ 完整 |
| Processing Integrity | PI | ⚠️ 部分 |
| Confidentiality | C | ✅ 完整 |
| Privacy | P | ✅ 完整 |

---

## Common Criteria (CC) — Security

| Control ID | 描述 | GGID 功能 | Evidence |
|-----------|------|-----------|----------|
| CC1.1 | 责任与问责制 | RBAC + 角色分配审计 | Console > Roles; Audit Log |
| CC1.2 | 安全职责分离 | ABAC 策略引擎 + 权限矩阵 | Console > Policies > Permission Matrix |
| CC2.1 | 系统组件清单 | SCIM 2.0 Provisioning + 用户生命周期 | Console > Users; SCIM sync log |
| CC2.2 | 内部风险识别 | ITDR 威胁检测 + 异常评分 | Console > Security > ITDR |
| CC3.1 | 风险评估 | 零信任 Posture Score | Console > Security > Posture |
| CC3.2 | 风险管理响应 | Conditional Access (CAE) | Console > Settings > Conditional Access |
| CC4.1 | 监控控制 | 审计 hash-chain + SIEM Metrics | Console > Audit; Monitoring > Alerts |
| CC4.2 | 异常纠正 | Access Reviews + Deprovisioning | Console > Audit > Access Reviews |
| CC5.1 | 逻辑访问安全 | JWT + MFA + WebAuthn/Passkey | Auth Service; MFA Settings |
| CC5.2 | 用户访问授权 | 细粒度 RBAC + ABAC | Console > Roles > Permission Matrix |
| CC5.3 | 凭证管理 | 密码策略 + API Keys + Token Rotation | Settings > Password Policy; API Keys |
| CC6.1 | 网络防护 | NetworkPolicy + Rate Limiting + Geo-fencing | deploy/helm NetworkPolicy; Geo-fence handler |
| CC6.2 | 入侵检测 | ITDR + Anomaly Detection + Threat Feed | Console > Security > ITDR |
| CC6.3 | 恶意软件防护 | AAGUID Allowlist (Passkey) + Session Revocation | Auth > AAGUID handler |
| CC7.1 | 系统监控 | Prometheus metrics + Grafana dashboards | deploy/grafana/ |
| CC7.2 | 异常检测 | 审计 anomaly detection + SIEM correlation | Console > Audit > Anomalies |
| CC7.3 | 安全事件评估 | Emergency Audit + Incident Response | Console > Audit > Emergency |
| CC7.4 | 事件响应 | 审计 tamper protection + alerting | Console > Audit > Tamper Protection |
| CC7.5 | 恢复计划 | DB backup + HA failover | deploy/docs/ha-architecture.md |
| CC8.1 | 变更管理 | API Versioning + Breaking Change CI | Gateway versioning; SDK drift detection |
| CC9.1 | 风险缓解 | Conditional Access + Step-up Auth | Settings > Conditional Access |

## Availability (A)

| Control ID | 描述 | GGID 功能 | Evidence |
|-----------|------|-----------|----------|
| A1.1 | 容量管理 | HPA autoscaling + Resource limits | values-prod.yaml autoscaling |
| A1.2 | 环境保护 | Multi-region HA + DNS failover | deploy/docs/ha-architecture.md |
| A1.3 | 恢复基础设施 | DB replication + backup verification | PG streaming replication |
| A2.1 | 系统恢复 | Helm rollback + Rolling update | deploy/rolling-update.sh |

## Confidentiality (C)

| Control ID | 描述 | GGID 功能 | Evidence |
|-----------|------|-----------|----------|
| C1.1 | 数据分类 | PII Scan + Data Lineage | Console > Audit > PII Scan |
| C1.2 | 加密传输 | TLS + JWT RS256 | Gateway TLS termination |
| C1.3 | 加密存储 | Password hashing (bcrypt + pepper) + Audit hash-chain | Auth Service password handler |

## Privacy (P)

| Control ID | 描述 | GGID 功能 | Evidence |
|-----------|------|-----------|----------|
| P2.1 | 隐私通知 | Brand customization + Terms | Settings > Branding |
| P3.1 | 数据收集 | GDPR Forget + Data Export | Console > Audit > GDPR Forget |
| P4.1 | 数据使用 | Consent + Tenant isolation | Multi-tenant architecture |
| P5.1 | 数据保留 | Audit retention + Token expiry policy | Audit config; Token lifecycle |
| P5.2 | 数据处置 | Deprovisioning workflow | Console > Audit > Deprovisioning |
| P6.1 | 访问权限 | RBAC + Privacy settings | Console > Profile > Privacy |
| P7.1 | 隐私监控 | PII Scan + Data Lineage tracking | Console > Audit > Data Lineage |
| P8.1 | 披露通知 | Compliance Reports + Evidence Chain | Console > Audit > Compliance Reports |

---

## Evidence Collection 自动化建议

| Evidence | 自动化方式 | 频率 |
|----------|-----------|------|
| 审计日志完整性 | Hash-chain tamper check cron | 每日 |
| 权限矩阵快照 | `ggid-cli policies export` | 每周 |
| MFA 覆盖率 | Audit query + Grafana panel | 每月 |
| Access Review 完成 | Access review workflow | 季度 |
| 备份恢复测试 | backup-verify.sh cron job | 每日 |
| Posture Score 趋势 | Grafana dashboard export | 月度 |
