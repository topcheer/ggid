# GGID Team Kanban

> 共享任务看板。所有团队成员自主认领、更新、完成。
> **规则**：手头任务完成 → 从 TODO 取下一项 → 改状态为 in_progress + 填 assignee → 开始做。不需要等 arch 分配。

## 状态说明
- `TODO` — 待认领，任何对应角色成员可取
- `IN_PROGRESS` — 有人在做（填 assignee）
- `REVIEW` — 完成但待验证
- `DONE` — 已验证完成
- `BLOCKED` — 被阻塞（填原因）

## 文件归属规则
- backend → services/
- frontend → console/src/ + sdk/react/src/
- IAMExpert → pkg/crypto/ + 安全审计 + 设计文档
- docs/techwriter → docs/
- arch → pkg/, sdk/, deploy/, cmd/, test/

## === BACKEND (services/) ===

### IN_PROGRESS
| ID | Task | Assignee | Started | Notes |
|----|------|----------|---------|-------|
| B-06 | RFC 8693 Token Exchange 标准 grant | backend | 07-17 | docs/research/token-exchange-standard-grant-gap.md |

### TODO
| ID | Task | Priority | Scope | Acceptance |
|----|------|----------|-------|------------|
| B-06 | RFC 8693 Token Exchange | P1 | services/oauth/ | docs/research/token-exchange-standard-grant-gap.md |

### DONE
| ID | Task | Assignee | Commit |
|----|------|----------|--------|
| B-01 | CAE Phase 1-3 (jti 黑名单 + session revoke + ITDR 联动) | backend | 699c66e7 |
| B-02 | 内部认证 HMAC 中间件 | backend | e52a3528 |
| B-03 | 内部认证 gateway Director 注入 | backend | db2a5a9a |
| B-04 | WebAuthn 持久化 + valid-ids 改 DB | backend | 4a5bff9c |
| B-08 | break-glass 内存数组迁 DB | backend | 88a85dfd |
| B-09 | IGA revoke 写 audit 事件 | backend | 008762a4 |
| B-10 | ITDR Phase 4：3 条新规则 | backend | f1c85214 |
| B-11 | ITDR Redis StateStore | backend | 709a94c4 |
| B-12 | SCIM bearer token 实现 | backend | bb2fbf94 |
| B-13 | ZT posture 真实聚合端点 | backend | d7ad68d1 |
| B-14 | CIEM cross-analysis 端点 | backend | c6847200 |
| B-15 | IGA GenAI 回收建议端点 | backend | 76aee44c |
| B-05 | 内部认证：6 服务 mux 包裹 InternalAuth | backend | f3580d58 |
| B-07 | PAM JIT Zero Standing 实现 | backend | 0a083fff |

## === FRONTEND (console/src/) ===

### IN_PROGRESS
| ID | Task | Assignee | Started |
|----|------|----------|---------|
### TODO
| ID | Task | Priority | Scope | Acceptance |
|----|------|----------|-------|------------|
| F-13 | ITDR Detections 实时仪表盘 | P2 | console/src/app/security/itdr-dashboard/ | SSE/streaming 接 /api/v1/audit/itdr/detections，实时刷新检测列表 |
| F-14 | KMS 配置页面对接后端真实端点 | P2 | console/src/app/settings/kms-config/ | 当前页面已有，验证 PUT/POST/test 端点返回真实数据 |

### DONE
| ID | Task | Assignee | Commit |
|----|------|----------|--------|
| F-01 | ZT 姿态评分页面真实化 | frontend | 5a2a9728 |
| F-02 | CIEM 权限使用分析页面 | frontend | 593c2c7e |
| F-03 | SDK 假数据清理 153 hooks (Batch 1-5) | frontend | fc559c27+4 |
| F-04 | IGA GenAI 辅助审查 UI | frontend | 6270900f |
| F-05 | Onboarding Wizard 4 步引导 | frontend | already-existed |
| F-06 | Passkey 删除接 signalCredentialRemoved | frontend | 12db7caa |
| F-07 | Profile 改名接 signalCurrentUserDetails | frontend | 12db7caa |
| F-08 | CAE Session Revocation 管理页面 | frontend | 1d8b3da3 |
| F-09 | PAM JIT 权限提升请求页面 | frontend | 19c72dbe |
| F-10 | Break-glass 紧急访问页面 | frontend | 19c72dbe |
| F-11 | ITDR Rules 管理页面 | frontend | 19c72dbe |
| F-12 | Console OAuth PKCE 端到端验证 | frontend | verified-no-change |
| F-05 | Onboarding Wizard 4 步引导 | frontend | verified |
| F-06 | Passkey 删除接 signalCredentialRemoved | frontend | 12db7caa |
| F-07 | Profile 改名接 signalCurrentUserDetails | frontend | 12db7caa |

## === IAMEXPERT (pkg/crypto/ + 审计 + 设计) ===

### IN_PROGRESS
| ID | Task | Assignee | Started |
|----|------|----------|---------|
| I-03 | ~~PAM JIT E2E 验收~~ → DONE | IAMExpert | B-07 verified ✓ |

### TODO
| ID | Task | Priority | Acceptance |
|----|------|----------|------------|
| I-04 | ~~UEBA per-user baselines 设计文档~~ → DONE IAMExpert | P2 | docs/architecture/ueba-design.md，30天滑窗+3σ+冷启动 |
| I-05 | ~~零信任统一 PDP 设计文档~~ → DONE IAMExpert | P2 | ABAC DSL + $device.trusted/$itdr.critical/$session.risk |
| I-06 | ~~内部认证 6 服务 mux 完成后 E2E 验收~~ → DONE | P1 | 6/6 wrapped ✓ build 53pkg 0FAIL |

### DONE
| ID | Task | Assignee | Output |
|----|------|----------|--------|
| I-01 | 国密 SM2/SM3/SM4 全链路 | IAMExpert | f5659baf + 7cea65ab |
| I-02 | CAE 设计 + E2E 验收 | IAMExpert | docs/architecture/cae-design.md |

## === DOCS (docs/) ===

### DONE
| ID | Task | Assignee | Commit | Notes |
|----|------|----------|--------|-------|
| D-01 | API Reference 补充 ITDR/CAE/PAM 新端点 | docs&devops | fec4ffd2 | Added 7 new endpoint sections to api-reference.md: ITDR detections/stats/rules, CAE session revoke/posture, security-posture, anomaly-detection, break-glass activate/history, threat-intel feed |
| D-02 | 部署指南更新（端口安全 + CAE Redis 依赖） | docs&devops | fec4ffd2 | Updated all-in-one-deployment.md: removed backend port exposure (P0 fix), added security note + CAE Redis dependency section |

## === ARCH (pkg/, sdk/, deploy/) ===

### TODO
| ID | Task | Priority | Acceptance |
|----|------|----------|------------|
| A-01 | all-in-one Docker 重建（含全部最新代码） | P1 | 所有新功能在 Docker 中可用 |
| A-02 | k3s 镜像全量推送 | P1 | 所有服务最新镜像 |
| A-03 | K3s 部署安全加固：设置 PASSWORD_PEPPER + INTERNAL_AUTH_SECRET + AUDIT_HASH_CHAIN_SECRET env | P1 | kubectl set env 3 secrets, verify auth pod restarts cleanly, login still works |
