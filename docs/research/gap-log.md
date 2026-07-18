## GAP Log 2026-07-18T10:10:07Z

| Component | Status | Detail |
|-----------|--------|--------|
| NHI lifecycle | ❌ 内存 map | make(map[string]*NHIIdentity), 83行, 重启丢数据 |
| NHI risk engine | ❌ 内存 | 无 pool/pgx 引用 |
| Conditional access | ✅ DB-backed | pgxpool.Pool |
| CAE evaluations | ✅ DB-backed | pgxpool.Pool |
| Privilege creep | ✅ DB-backed | pcRepo with pgx |
| Delegation | ✅ DB-backed | pgxpool.Pool |
| Password strength | ✅ 纯计算 | 无需 DB |

## Additional GAPs found $(date -u +%Y-%m-%dT%H:%M:%SZ)
| SoD violations | ❌ 内存 store | sodViolationStore, 重启丢失 |
| SoD rules | ❌ 包级变量 | sodRules = []SoDRule{}, 硬编码, 重启丢失自定义 |
| OpenAPI coverage | ⚠️ 5% | 103/857 端点有 swag 注解 |

## Cloud-Native Integration GAP $(date -u +%Y-%m-%dT%H:%M:%SZ)

### 现有部署资产（已发现但未充分测试/文档化）
| 资产 | 状态 | 问题 |
|------|------|------|
| deploy/helm/ggid/ | 存在(9 templates, 229行 values) | 未测试 helm install |
| deploy/terraform/ | 存在(325行) | 未测试 terraform plan |
| deploy/operator/ | 存在(完整 K8s operator) | 未文档化 |
| deploy/openapi.yaml | 存在(974行) | 覆盖率未验证 |
| docker-compose.yml | 刚创建 | 未测试 |
| docker-compose.prod.yaml | 存在 | 未测试 |

### 云原生集成 GAP
| 集成 | 状态 | 优先级 |
|------|------|--------|
| AWS EKS 一键部署 | ❌ 缺少 | P1 |
| Azure AKS 一键部署 | ❌ 缺少 | P1 |
| GCP GKE 一键部署 | ❌ 缺少 | P1 |
| Workload Identity Federation | ❌ 缺少 | P2 |
| Cloud IAM 桥接 (AWS IAM/Azure AD/GCP IAM) | ⚠️ 部分(SAML) | P1 |
| Market place listing | ❌ 不适用(开源) | — |
