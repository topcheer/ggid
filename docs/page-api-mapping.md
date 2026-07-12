# 112 Console Pages Without API — Exact Backend Route Mapping

## 问题说明
这些页面有 UI（useState + 硬编码 mock 数据），但没有 fetch/useApi 调用真实后端 API。需要每页连接到下方指定的后端 endpoint。

## 实施要求
每页必须：
1. 删除 useState 中的硬编码 mock 数据数组
2. 添加 `useEffect(() => { fetch(...) }, [])` 从后端加载真实数据
3. 所有 fetch 加 `headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }`
4. 有 `const [loading, setLoading] = useState(true)` 和 `const [error, setError] = useState<string | null>(null)`
5. loading 时显示 "Loading..."，error 时显示错误信息，空数据时显示 "No data"

## 页面 → API 映射

### Auth Service (35 pages)
| 页面 | API Endpoint | Method |
|------|-------------|--------|
| account-lockout-config | /api/v1/auth/lockout-policy/config | GET/PUT |
| adaptive-authentication | /api/v1/auth/adaptive-auth/config | GET/PUT |
| breach-detection-config | /api/v1/auth/breach-warnings | GET |
| credential-rotation | /api/v1/auth/credentials/rotation/due | GET |
| credential-vault-management | /api/v1/auth/credentials/ | GET/POST |
| device-binding-config | /api/v1/auth/sessions/device-binding-status | GET |
| dpop-config | /api/v1/auth/token-reuse-check | GET |
| geo-fencing | /api/v1/auth/geo-fencing/config | GET/PUT |
| geo-velocity-rules | /api/v1/auth/velocity-rules | GET/PUT |
| hibp-breach-check | /api/v1/auth/password-breach-check | POST |
| host-validation-config | /api/v1/auth/throttle-status | GET |
| impersonation-config | /api/v1/auth/impersonation/config | GET/PUT |
| impersonation-session | /api/v1/auth/impersonate | POST |
| introspection-cache-config | /api/v1/auth/expiry-status | GET |
| ip-reputation-config | /api/v1/auth/tor-vpn/detect | POST |
| jwt-expiry-config | /api/v1/auth/expiry-status | GET |
| ldap-config | /api/v1/auth/adaptive-auth/config | GET/PUT |
| login-security-center | /api/v1/auth/risk/aggregate | GET |
| login-security-policy | /api/v1/auth/password-policy/config | GET/PUT |
| mfa-challenge-config | /api/v1/auth/mfa/challenge-config | GET/PUT |
| mfa-enrollment | /api/v1/auth/mfa/enrollment-stats | GET |
| mfa-enrollment-center | /api/v1/auth/mfa/factors | GET |
| passkey-management | /api/v1/auth/passkeys/status | GET |
| password-history-config | /api/v1/auth/password-history/config | GET/PUT |
| password-policy-center | /api/v1/auth/password-policy | GET |
| risk-engine-config | /api/v1/auth/risk-scoring/config | GET/PUT |
| risk-engine-dashboard | /api/v1/auth/risk/aggregate | GET |
| session-management-config | /api/v1/auth/session-timeout/config | GET/PUT |
| session-revocation-center | /api/v1/auth/sessions/revoke | POST |
| smtp-config | /api/v1/auth/email-template/config | GET/PUT |
| token-binding-config | /api/v1/auth/token-reuse-check | GET |
| token-binding-strategies | /api/v1/auth/sessions/anomaly-score | GET |
| token-management | /api/v1/auth/sessions | GET |
| token-claims | /api/v1/auth/sessions/anomaly-score | GET |
| token-introspection-center | /api/v1/auth/expiry-status | GET |

### Identity Service (18 pages)
| 页面 | API Endpoint | Method |
|------|-------------|--------|
| account-linking-config | /api/v1/identity/account-linking/config | GET/PUT |
| agent-access-review | /api/v1/identity/nhi | GET |
| agent-delegation-graph | /api/v1/identity/nhi/orphans | GET |
| digital-identity-lifecycle | /api/v1/identity/user-lifecycle/stages | GET |
| did-resolver | /api/v1/identity/did | GET |
| deprovisioning-workflow | /api/v1/identity/deprovisioning/config | GET/PUT |
| deprovisioning-workflow-config | /api/v1/identity/deprovisioning/config | GET/PUT |
| identity-correlation-graph | /api/v1/identity/groups/ | GET |
| nhi-inventory | /api/v1/identity/nhi | GET |
| notification-preview | /api/v1/notifications/send | POST |
| notification-provider-config | /api/v1/auth/notification-preferences | GET/PUT |
| notification-templates | /api/v1/auth/email-template/config | GET/PUT |
| onboarding-wizard | /api/v1/identity/joiner-flow | GET |
| org-hierarchy | /api/v1/orgs/tree | GET |
| org-tree-viewer | /api/v1/orgs/tree | GET |
| user-activity-dashboard | /api/v1/users/timeline | GET |
| user-provisioning-center | /api/v1/users/bulk-provision | POST |
| user-provisioning-rules | /api/v1/identity/scim/provisioning-config | GET/PUT |
| verifiable-credentials | /api/v1/identity/vc | GET |

### Policy Service (22 pages)
| 页面 | API Endpoint | Method |
|------|-------------|--------|
| abac-policy-editor | /api/v1/policies/abac/groups | GET/POST |
| access-request-center | /api/v1/policies/access-requests | GET |
| access-review-center | /api/v1/policies/access-reviews/campaigns | GET |
| condition-builder | /api/v1/policy/abac/condition-config | GET/PUT |
| delegation-management | /api/v1/policies/delegations | GET |
| delegation-validator | /api/v1/policy/delegation/validate | POST |
| event-correlation-rules | /api/v1/audit/correlation/rules | GET/PUT |
| feature-flags-config | /api/v1/policy/feature-flags | GET/PUT |
| permission-inheritance-config | /api/v1/policies/inheritance | GET |
| permission-tree | /api/v1/policies/permissions/tree | GET |
| policy-simulation-center | /api/v1/policies/simulate | POST |
| rbac-matrix | /api/v1/policies/sod/matrix | GET |
| role-templates-config | /api/v1/policies/role-templates | GET |
| scope-management | /api/v1/policies/abac/export | GET |
| scope-resolver-config | /api/v1/policies/attribute-mapping | GET/PUT |
| sod-conflict-detection | /api/v1/policies/sod/check | POST |
| sod-rules-config | /api/v1/policies/sod/rules | GET/PUT |
| bulk-operations | /api/v1/policies/bundles | GET/POST |

### Audit Service (18 pages)
| 页面 | API Endpoint | Method |
|------|-------------|--------|
| alert-webhook-config | /api/v1/audit/alert-webhooks | GET/POST |
| audit-export-center | /api/v1/audit/export | GET |
| audit-log-viewer | /api/v1/audit/events | GET |
| compliance-reports | /api/v1/audit/compliance-report | GET |
| data-export-center | /api/v1/audit/exports/schedule | GET/POST |
| hash-chain-status | /api/v1/audit/hash-chain/config | GET |
| hash-chain-verification | /api/v1/audit/verify-integrity | POST |
| sbom-center | /api/v1/audit/sbom | GET |
| security-dashboard | /api/v1/audit/security-posture | GET |
| siem-forwarder-config | /api/v1/audit/siem/forwarder-config | GET/PUT |
| siem-forwarder-dashboard | /api/v1/audit/siem/metrics | GET |
| siem-integration | /api/v1/audit/siem/health | GET |
| user-activity-dashboard | /api/v1/audit/aggregations | GET |
| webhook-delivery-monitor | /api/v1/audit/webhooks/delivery-status | GET |
| webhook-subscription-config | /api/v1/audit/alerts/config | GET/PUT |
| webhook-subscriptions | /api/v1/audit/webhooks | GET |

### OAuth Service (8 pages)
| 页面 | API Endpoint | Method |
|------|-------------|--------|
| client-lifecycle | /api/v1/oauth/clients | GET |
| client-onboarding | /api/v1/oauth/clients | POST |
| client-secret-rotation | /api/v1/oauth/clients | GET |
| dynamic-client-registration | /api/v1/oauth/clients | POST |
| oauth-client-registry | /api/v1/oauth/clients | GET |
| oauth-clients-config | /api/v1/oauth/clients | GET |
| par-config-management | /api/v1/oauth/par | GET |
| consent-management-center | /api/v1/oauth/consent | GET |

### Gateway/Infra (11 pages) — no direct API, use gateway health
| 页面 | API Endpoint | Method |
|------|-------------|--------|
| api-gateway-routes | /api/v1/audit/metrics | GET |
| api-health-monitor | /healthz | GET |
| api-key-management | /api/v1/auth/credentials/ | GET |
| api-keys-config | /api/v1/auth/credentials/ | GET |
| api-versioning-config | /api/v1/audit/metrics | GET |
| api-versioning-strategy | /api/v1/audit/metrics | GET |
| auto-scaling-config | /healthz | GET |
| body-size-limit-config | /healthz | GET |
| circuit-breaker-config | /healthz | GET |
| circuit-breaker-dashboard | /api/v1/audit/metrics | GET |
| k8s-deployment | /healthz | GET |
| k8s-deployment-management | /healthz | GET |
| request-id-tracking | /api/v1/audit/events | GET |
| retry-policy-config | /healthz | GET |
| service-dependency-graph | /healthz | GET |
| encryption-config | /healthz | GET |
| data-masking-config | /api/v1/audit/pii-scan | POST |