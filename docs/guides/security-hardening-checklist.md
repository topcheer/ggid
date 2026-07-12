# Security Hardening Checklist

CIS benchmarks, network policies, pod security, RBAC least privilege, image scanning, admission controllers, and audit policy.

## Checklist

### Network Security

| # | Control | Pass Criteria | Status |
|---|---------|---------------|--------|
| 1 | All ingress via gateway | NetworkPolicy: only gateway → services | ✅ |
| 2 | mTLS between services | Service mesh or app-level mTLS | ✅ |
| 3 | Egress restricted | NetworkPolicy denies all egress except required | ✅ |
| 4 | No public DB access | DB in private subnet, no public IP | ✅ |
| 5 | DNS over TLS | CoreDNS configured with TLS upstream | ⚠️ |

### Pod Security

| # | Control | Pass Criteria | Status |
|---|---------|---------------|--------|
| 6 | Run as non-root | `runAsNonRoot: true` in all pods | ✅ |
| 7 | Read-only root filesystem | `readOnlyRootFilesystem: true` | ✅ |
| 8 | Drop all capabilities | `drop: ["ALL"]` in securityContext | ✅ |
| 9 | No privileged pods | `privileged: false` | ✅ |
| 10 | Resource limits set | CPU + memory limits on all containers | ✅ |
| 11 | seccomp profile | `RuntimeDefault` seccomp | ✅ |

### RBAC Least Privilege

| # | Control | Pass Criteria | Status |
|---|---------|---------------|--------|
| 12 | No cluster-admin for services | Each service uses dedicated ServiceAccount | ✅ |
| 13 | Minimal RBAC roles | Only required verbs/resources | ✅ |
| 14 | No default service account | `automountServiceAccountToken: false` | ✅ |
| 15 | Human access via OIDC | kubectl auth via GGID OIDC provider | ✅ |

### Image Security

| # | Control | Pass Criteria | Status |
|---|---------|---------------|--------|
| 16 | Image scanning (Trivy) | No CRITICAL CVEs | ✅ |
| 17 | Base image distroless | No shell in production images | ✅ |
| 18 | Image pull policy Always | `imagePullPolicy: Always` | ✅ |
| 19 | Private registry only | Images from `registry.ggid.dev` | ✅ |
| 20 | Signature verification | Cosign signatures verified | ⚠️ |

### Admission Control

| # | Control | Pass Criteria | Status |
|---|---------|---------------|--------|
| 21 | OPA Gatekeeper | Enforces pod security standards | ✅ |
| 22 | Image policy webhook | Rejects unsigned/unscanned images | ⚠️ |
| 23 | Resource quota enforced | Per-namespace CPU/memory limits | ✅ |
| 24 | Namespace isolation | Each tenant in own namespace | ✅ |

### Data Security

| # | Control | Pass Criteria | Status |
|---|---------|---------------|--------|
| 25 | Encryption at rest | DB TDE + volume encryption | ✅ |
| 26 | Encryption in transit | TLS 1.3 everywhere | ✅ |
| 27 | Secrets in Vault | No secrets in env vars or configmaps | ✅ |
| 28 | RLS enabled | All tenant tables have RLS | ✅ |

### Audit

| # | Control | Pass Criteria | Status |
|---|---------|---------------|--------|
| 29 | Kubernetes audit logging | API server audit enabled | ✅ |
| 30 | App audit with hash chain | Tamper-evident audit log | ✅ |
| 31 | 7-year retention | Audit data retained per compliance | ✅ |
| 32 | SIEM forwarding | Events sent to Splunk/ELK | ✅ |

### Secrets Management

| # | Control | Pass Criteria | Status |
|---|---------|---------------|--------|
| 33 | Vault for all secrets | No hardcoded secrets | ✅ |
| 34 | Auto-rotation (90 days) | DB/API keys rotated | ✅ |
| 35 | External Secrets Operator | Syncs Vault → K8s secrets | ✅ |

## Summary

| Category | Total | Pass | Warning | Gap |
|----------|-------|------|---------|-----|
| Network | 5 | 4 | 1 | 0 |
| Pod Security | 6 | 6 | 0 | 0 |
| RBAC | 4 | 4 | 0 | 0 |
| Image | 5 | 4 | 1 | 0 |
| Admission | 4 | 3 | 1 | 0 |
| Data | 4 | 4 | 0 | 0 |
| Audit | 4 | 4 | 0 | 0 |
| Secrets | 3 | 3 | 0 | 0 |
| **Total** | **35** | **32** | **3** | **0** |

## Open Items

| # | Item | Priority |
|---|------|----------|
| 5 | Configure CoreDNS DNS-over-TLS | P3 |
| 20 | Implement Cosign image signing | P2 |
| 22 | Deploy image policy webhook | P2 |

## See Also

- [Tenant Isolation Architecture](tenant-isolation-architecture.md)
- [Database Security](database-security.md)
- [Secret Sprawl Prevention](secret-sprawl-prevention.md)
- [Compliance Framework Mapping](compliance-framework-mapping.md)
