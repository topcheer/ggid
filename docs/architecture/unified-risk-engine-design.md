# Unified Risk Engine (URE) Design

> 状态: Proposed | 作者: IAMExpert | 日期: 2026-07-18 | 来源: arch 指派 + KB-137..142 研究
> 关联: ITDR 引擎 (B-37b), CAE risk engine, Threat Intel Hub (B-37), Device Posture, UEBA baseline

## 1. 当前状态审计：3 个独立 RiskEngine

GGID 目前有 **3 个独立的、互不通信的风险评分引擎**，各自维护评分逻辑、信号源、和阈值。此外 auth 服务中有 2 个风险消费者（session timeout + hijack check）。

### 1.1 Audit RiskEngine (services/audit/internal/service/risk_engine.go)

| 属性 | 值 |
|------|-----|
| 评分范围 | 0.0 – 1.0 (float) |
| 信号数 | 6 (velocity, device_known, new_ip, geo_anomaly, threat_intel_hit, threat_severity) |
| 存储 | **内存 map** (velocityStore, knownDevices, knownIPs, knownLocations) |
| 消费者 | ITDR Engine callback → session revoke |
| 问题 | 重启数据丢失；无法跨服务查询；threat_intel 是唯一外部信号 |

### 1.2 Policy RiskScore Handler (services/policy/internal/server/risk_score_handler.go)

| 属性 | 值 |
|------|-----|
| 评分范围 | 0 – 100 (int) |
| 信号数 | 5 (login_velocity, geo_anomaly, device_trust, ip_reputation, time_pattern) |
| 存储 | **无持久化** — 每次请求重新计算 |
| 消费者 | ABAC 条件 (`env.risk_score < 0.7`), access decision |
| 问题 | hardcoded 因子（`velocity * 15` 硬编码权重）；不查询 audit/identity 的信号；IP reputation 无数据源 |

### 1.3 Identity RiskProfile Handler (services/identity/internal/server/risk_profile_handler.go)

| 属性 | 值 |
|------|-----|
| 评分范围 | 0 – 100 (int) |
| 信号数 | 5 (privileged_access, stale_password, no_mfa, dormant, exposed_credentials) |
| 存储 | **内存 map** (`data: make(map[string]*userRiskProfile)`) |
| 消费者 | Console 风险页面 |
| 问题 | 所有因子 hardcoded 假数据（"User has admin-level role" 写死）；不查询真实权限/MFA/登录历史；内存 map 违反 checklist |

### 1.4 Auth Risk Consumers

- `session_timeout_handler.go` (49 行): 接收 `?risk_score=X` query param → 返回动态 session timeout
- `hijack_check_handler.go` (56 行): 硬编码 RiskScore struct，返回静态 JSON

### 1.5 覆盖率分析（26 个风险组件）

| 组件类别 | 已有 | 缺失 | 覆盖率 |
|----------|------|------|--------|
| Device | device_known, device_trust | device_posture_score, managed_status, jailbreak_root, attestation | 2/6 (33%) |
| Geo/Anomaly | geo_anomaly, impossible_travel | vpn_proxy, asn_mismatch, new_country_velocity | 2/5 (40%) |
| Network (Threat) | threat_intel_hit, ip_reputation(hardcoded) | tor_exit, botnet_c2, datacenter_ip, asn_reputation | 1.5/5 (30%) |
| Behavioral (UEBA) | login_velocity | off_hours_pattern, privilege_escalation, api_abuse, credential_stuffing, baseline_deviation | 2/6 (33%) |
| Session | session_age, session_hijack(hardcoded) | concurrent_sessions, token_replay, session_fixation | 1/4 (25%) |
| **总计** | **8.5** | **17.5** | **33%** |

**结论**：26 个组件中仅 8.5 个实际工作（33%），与 KB-142 研究的 31% 一致。

---

## 2. 目标架构：统一风险引擎 (URE)

```
┌─────────────────────────────────────────────────────────────┐
│                    Unified Risk Engine                       │
│                   (services/audit/ure/)                       │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │  Signal      │  │  Composite   │  │  Decision    │       │
│  │  Collectors  │→ │  Scorer      │→ │  Policy      │       │
│  │  (20 types)  │  │  (weighted)  │  │  (threshold) │       │
│  └──────┬───────┘  └──────────────┘  └──────┬───────┘       │
│         │                                    │               │
│  ┌──────┴──────────────────────────────────┴───────┐        │
│  │           Signal Registry (per-tenant)          │        │
│  │  weights + thresholds + enabled/disabled        │        │
│  └──────────────────────────────────────────────────┘        │
│         │                                                    │
│  ┌──────┴──────────────────────────────────────────────┐    │
│  │           PostgreSQL Persistence                     │    │
│  │  risk_scores + risk_signals + risk_policies         │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
       ↑              ↑                ↑               ↑
   ITDR Engine    Gateway PDP     Auth Service    Console UI
   (event eval)   (access gate)   (session)       (dashboard)
```

### 设计原则

1. **单一评分源**：所有服务通过 `GET /risk/score/:user_id` 获取统一评分
2. **信号可插拔**：新信号通过实现 `SignalCollector` 接口接入，不需修改引擎核心
3. **租户隔离**：权重和阈值 per-tenant 可配置
4. **持续评估**：CAE 定时 re-eval（15min）+ 特权操作触发

---

## 3. 信号分类法（20 种信号）

### 3.1 Device Trust（6 信号）

| ID | 信号 | 数据源 | 权重默认 |
|----|------|--------|----------|
| `device.trust_level` | 设备信任级别 (managed/compliant/unknown) | device_posture 表 | 15 |
| `device.known` | 设备是否已注册 | auth device_bindings | 5 |
| `device.jailbreak` | 越狱/root 检测 | device postur attestation | 20 |
| `device.attestation` | 硬件证明状态 | platform attestation | 15 |
| `device.compliance` | 合规状态（MDM 策略） | MDM integration | 10 |
| `device.certificate_valid` | 设备证书有效期 | PKI | 5 |

### 3.2 Geo / Anomaly（5 信号）

| ID | 信号 | 数据源 | 权重默认 |
|----|------|--------|----------|
| `geo.impossible_travel` | 不可能旅行 | audit velocity store | 20 |
| `geo.new_country` | 首次从新国家登录 | login history | 10 |
| `geo.vpn_proxy` | VPN/代理检测 | IP geo DB | 10 |
| `geo.asn_mismatch` | ASN 不匹配历史 | ASN registry | 5 |
| `geo.distance_km` | 登录地与通常位置距离 | geo DB | 5 |

### 3.3 Network / Threat Intel（5 信号）

| ID | 信号 | 数据源 | 权重默认 |
|----|------|--------|----------|
| `net.threat_intel_ip` | IP 在威胁情报库中 | threat_indicators 表 | 25 |
| `net.threat_intel_email` | Email 在泄露库中 | threat_indicators 表 | 20 |
| `net.tor_exit` | Tor 出口节点 | tor node list | 15 |
| `net.datacenter_ip` | 数据中心 IP（非浏览器） | ASN DB | 10 |
| `net.bot_signature` | 已知 bot/crawler 签名 | UA fingerprint | 5 |

### 3.4 Behavioral / UEBA（6 信号）

| ID | 信号 | 数据源 | 权重默认 |
|----|------|--------|----------|
| `behavior.login_velocity` | 单小时登录次数 | audit velocity | 15 |
| `behavior.off_hours` | 非工作时间操作 | audit events | 10 |
| `behavior.privilege_escalation` | 权限提升尝试 | ITDR detections | 20 |
| `behavior.api_abuse` | API 调用频率异常 | rate limiter | 10 |
| `behavior.credential_stuffing` | 凭据填充模式 | ITDR credential_stuffing rule | 15 |
| `behavior.baseline_deviation` | 行为基线偏离 | UEBA baseline | 10 |

### 3.5 Session（4 信号）

| ID | 信号 | 数据源 | 权重默认 |
|----|------|--------|----------|
| `session.age` | Session 超过 8 小时 | auth sessions | 5 |
| `session.concurrent` | 多设备并发 session | auth sessions | 10 |
| `session.token_replay` | Token 在多 IP 使用 | token audit | 20 |
| `session.hijack_score` | 会话劫持指标 | hijack_check | 15 |

> **注意**：上面列了 26 个信号位，但部分是"子信号"，URE 统一为 20 个 top-level 信号类型。每类下可配子规则。

---

## 4. 复合加权评分算法

### 4.1 公式

```
risk_score = clamp(Σ(signal_value_i × weight_i × severity_multiplier_i) × 100, 0, 100)
```

其中：
- `signal_value_i`: 0.0 – 1.0（信号触发的程度）
- `weight_i`: 租户配置的权重（默认值见 §3）
- `severity_multiplier_i`: 0.5 / 1.0 / 1.5（信号严重程度倍率）

### 4.2 实现

```go
type SignalResult struct {
    SignalID    string  `json:"signal_id"`
    Value       float64 `json:"value"`        // 0.0-1.0
    Weight      float64 `json:"weight"`       // tenant config
    Severity    string  `json:"severity"`     // low/medium/high/critical
    Detail      string  `json:"detail"`
}

type RiskResult struct {
    UserID      string         `json:"user_id"`
    TenantID    string         `json:"tenant_id"`
    Score       int            `json:"score"`       // 0-100
    Level       string         `json:"level"`       // low/medium/high/critical
    Decision    string         `json:"decision"`    // allow/step_up/step_up_strong/block
    Signals     []SignalResult `json:"signals"`
    EvaluatedAt time.Time      `json:"evaluated_at"`
}

func (e *Engine) Evaluate(ctx context.Context, userID, tenantID uuid.UUID) *RiskResult {
    signals := e.collectSignals(ctx, userID, tenantID)
    
    rawScore := 0.0
    var results []SignalResult
    for _, sig := range signals {
        mult := severityMultiplier(sig.Severity)
        rawScore += sig.Value * sig.Weight * mult
        results = append(results, sig)
    }
    
    score := clampToRange(rawScore * 100 / e.maxWeightSum(tenantID), 0, 100)
    
    return &RiskResult{
        Score:    int(math.Round(score)),
        Level:    scoreToLevel(score),
        Decision: e.policy.Decision(tenantID, score),
        Signals:  results,
    }
}
```

### 4.3 归一化

权重总和按 `Σ(weight_i)` 归一化到 0-100，确保不同租户配置可比较。

---

## 5. 决策策略

### 5.1 默认阈值

| Score 范围 | Level | Decision | CAE Action |
|-----------|-------|----------|------------|
| 0 – 29 | low | `allow` | 无额外验证 |
| 30 – 59 | medium | `step_up` | 要求 MFA |
| 60 – 84 | high | `step_up_strong` | 要求硬件 key / 生物认证 |
| 85 – 100 | critical | `block` | 阻断会话 + SOC 告警 |

### 5.2 Per-Tenant 配置

```json
{
  "tenant_id": "uuid",
  "thresholds": {
    "allow_max": 30,
    "step_up_max": 60,
    "step_up_strong_max": 85
  },
  "signal_overrides": {
    "net.threat_intel_ip": {"weight": 30, "severity_multiplier": 1.5},
    "geo.impossible_travel": {"disabled": true}
  }
}
```

---

## 6. 持续评估 (CAE Integration)

### 6.1 定时 Re-Evaluation

```go
// 每 15 分钟对活跃 session 用户进行风险评估
func (e *Engine) RunContinuousEval(ctx context.Context) {
    ticker := time.NewTicker(15 * time.Minute)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            activeUsers := e.getActiveUsers(ctx)
            for _, userID := range activeUsers {
                result := e.Evaluate(ctx, userID, tenantID)
                e.persist(ctx, result)
                
                // Score increased significantly → trigger CAE action
                if result.Decision != "allow" {
                    e.caeCallback(ctx, result)
                }
            }
        }
    }
}
```

### 6.2 特权操作触发

当 PDP evaluatePolicy 遇到 `risk_score` 条件时，同步调用 URE：

```
PDP → GET /risk/score/:user_id → URE evaluates → returns score → PDP gates access
```

特权操作（role.assign, data.export, admin.api）强制触发即时评估，不等 15min 周期。

---

## 7. ITDR 集成

### 7.1 Threat Intel → Network Signal

B-37b 已实现 `ThreatIntelRule` → ITDR detection。URE 消费 threat_intel_hit detection 作为 `net.threat_intel_ip` 信号源：

```go
func (c *ThreatIntelSignalCollector) Collect(ctx context.Context, userID uuid.UUID) (*SignalResult, error) {
    detections, _ := c.itdrRepo.ListByActor(ctx, userID)
    for _, det := range detections {
        if det.RuleID == "threat_intel_hit" {
            return &SignalResult{
                SignalID: "net.threat_intel_ip",
                Value:    severityToValue(det.Severity),
                Severity: string(det.Severity),
            }, nil
        }
    }
    return nil, nil
}
```

### 7.2 ITDR Detection → Risk Re-Evaluation

任何 ITDR detection（brute_force, credential_stuffing, impossible_travel）触发 URE 即时 re-eval：

```go
func (e *Engine) OnITDRDetection(ctx context.Context, det *domain.Detection) {
    if det.ActorID == nil {
        return
    }
    // Immediate re-evaluation — don't wait for 15min cycle.
    result := e.Evaluate(ctx, *det.ActorID, det.TenantID)
    e.persist(ctx, result)
    
    if result.Decision == "block" {
        e.caeCallback(ctx, result) // session revoke + SOC webhook
    }
}
```

---

## 8. 数据模型

### risk_policies

```sql
CREATE TABLE risk_policies (
    id              UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL UNIQUE,
    thresholds      JSONB NOT NULL DEFAULT '{"allow_max":30,"step_up_max":60,"step_up_strong_max":85}',
    signal_overrides JSONB NOT NULL DEFAULT '{}',
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### risk_signals

```sql
CREATE TABLE risk_signals (
    id              UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    signal_id       TEXT NOT NULL,          -- "net.threat_intel_ip"
    signal_category TEXT NOT NULL,          -- "network" | "device" | "geo" | "behavior" | "session"
    default_weight  REAL NOT NULL DEFAULT 10,
    default_severity TEXT NOT NULL DEFAULT 'medium',
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE(tenant_id, signal_id)
);
```

### risk_scores

```sql
CREATE TABLE risk_scores (
    id              UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    score           INT NOT NULL,           -- 0-100
    level           TEXT NOT NULL,          -- low/medium/high/critical
    decision        TEXT NOT NULL,          -- allow/step_up/step_up_strong/block
    signals         JSONB NOT NULL DEFAULT '[]',
    evaluated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_risk_scores_user ON risk_scores(tenant_id, user_id, evaluated_at DESC);
```

---

## 9. API 契约

### GET /api/v1/risk/score/:user_id — 获取用户风险评分

Response 200:
```json
{
  "user_id": "uuid",
  "tenant_id": "uuid",
  "score": 47,
  "level": "medium",
  "decision": "step_up",
  "signals": [
    {"signal_id": "net.threat_intel_ip", "value": 0.8, "weight": 25, "severity": "high", "detail": "IP 203.0.113.50 in AbuseIPDB"},
    {"signal_id": "behavior.login_velocity", "value": 0.5, "weight": 15, "severity": "medium", "detail": "12 logins in last hour"}
  ],
  "evaluated_at": "2026-07-18T02:00:00Z"
}
```

### POST /api/v1/risk/evaluate — 即时评估

Request:
```json
{"user_id": "uuid", "tenant_id": "uuid", "trigger": "privileged_request"}
```

Response 200: (same as GET)

### PUT /api/v1/risk/policies/:tenant_id — 更新策略

Request:
```json
{
  "thresholds": {"allow_max": 25, "step_up_max": 55, "step_up_strong_max": 80},
  "signal_overrides": {"net.threat_intel_ip": {"weight": 30}}
}
```

### curl 验收命令

```bash
# 获取评分
curl http://localhost:8080/api/v1/risk/score/01234567-89ab-cdef-... \
  -H "Authorization: Bearer $TOKEN"

# 即时评估
curl -X POST http://localhost:8080/api/v1/risk/evaluate \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"user_id":"01234567-...","tenant_id":"...","trigger":"privileged_request"}'

# 更新策略
curl -X PUT http://localhost:8080/api/v1/risk/policies/tenant-uuid \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"thresholds":{"allow_max":25},"signal_overrides":{"net.threat_intel_ip":{"weight":30}}}'
```

---

## 10. 迁移计划

### Phase 1: URE 核心服务（1 周）
1. 创建 `services/audit/ure/` 包（URE Engine + Signal Registry + PG schema）
2. 实现 20 个 SignalCollector（分 5 批）
3. 实现 composite scorer + decision policy
4. 暴露 API endpoints
5. 初始化默认 risk_policies（所有租户继承默认权重）

### Phase 2: 消费者迁移（3 天）
1. Policy ABAC：`env.risk_score` 从 policy handler → URE API
2. Auth session_timeout：从 hardcoded → URE API
3. Console risk dashboard：从 identity handler → URE API
4. Gateway PDP：risk 条件 → URE API（同步调用）

### Phase 3: 持续评估 + ITDR 联动（3 天）
1. 启动 continuous eval goroutine（15min 周期）
2. 注册 ITDR Engine callback → URE.OnITDRDetection
3. 实现 CAE callback（session revoke + SOC webhook）

### Phase 4: 旧引擎废弃（2 天）
1. 标记 audit RiskEngine 为 deprecated（保留 fallback）
2. 删除 identity risk_profile 内存 map
3. 删除 policy risk_score_handler 硬编码因子
4. 从 auth session_timeout 移除本地计算

### Phase 5: 信号补全（持续）
1. 逐个实现 §1.5 中缺失的 17.5 个组件
2. 接入外部数据源（IP geo DB, ASN DB, Tor node list）
3. UEBA baseline 训练管道

**预计总工期：3 周（1 人）**

---

## 11. 反模式禁令

- **禁止内存 map 存储风险评分** — 必须 PostgreSQL risk_scores 表
- **禁止 hardcoded 权重/因子** — 必须从 risk_signals 表读取 per-tenant 配置
- **禁止 hardcoded JSON 假数据** — identity risk_profile 的 hardcoded factors（"admin-level role"）必须替换
- **禁止独立评分** — 所有服务必须调 URE API 获取统一评分，不得本地计算
- **禁止 log.Printf 占位** — SignalCollector 必须真实查询数据源
- **禁止同步阻塞** — continuous eval 必须 async goroutine + context timeout

---

## 12. 测试计划

| 测试 | 验证点 |
|------|--------|
| TestSignalCollector_DeviceTrust | 设备信任信号从 device_posture 表读取 |
| TestSignalCollector_ThreatIntel | threat_intel_hit detection → network signal |
| TestSignalCollector_GeoAnomaly | 不可能旅行 → geo signal |
| TestCompositeScorer_Weighted | 多信号加权评分正确（验证公式） |
| TestCompositeScorer_Normalize | 权重归一化到 0-100 |
| TestDecisionPolicy_Allow | score < 30 → allow |
| TestDecisionPolicy_StepUp | score 30-59 → step_up |
| TestDecisionPolicy_Block | score > 85 → block |
| TestDecisionPolicy_PerTenant | 不同租户不同阈值 |
| TestContinuousEval_Trigger | 15min ticker 触发 re-eval |
| TestITDRDetection_ReEval | ITDR detection → 即时 re-eval → score 升高 |
| TestCAECallback_SessionRevoke | block decision → session revoke |
| TestNilGracefulDegradation | nil repo/checker → 不 panic |
| TestSignalOverride | tenant override 权重 → 评分变化 |
| TestMigrationBackwardCompat | URE API 返回格式与旧 handler 兼容 |
