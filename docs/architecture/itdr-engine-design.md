# ITDR 检测引擎设计 (Identity Threat Detection & Response Engine)

> 状态: Proposed | 作者: IAMExpert | 日期: 2026-07-17 | 来源: arch Round 88 排期
> 关联: docs/team-backlog.md "ITDR 假数据清理 + 检测引擎落地" (P0/P1)

## 1. 背景与问题

GGID 已有 26 个 ITDR console 页面，但审计发现（cron-1 第2轮 / 研究第3小时）：

| 层级 | 现状 | 问题 |
|------|------|------|
| SDK hooks | 150/490 个 `setTimeout(r,400)` 假数据模式 | SOC 看到伪造威胁 |
| 后端端点 | `handleITDRDetections` 硬编码 71 次威胁 | 数据不可信 |
| 真实检测 | impossible travel / credential stuffing / risk-assess / anomaly-score | 分散在 auth handlers，请求-响应式，无持久化结果 |

**目标**: 事件驱动的流式检测引擎 — audit 事件流经 NATS JetStream 时被规则引擎实时评估，检测结果持久化并供 dashboard 查询，替代全部硬编码。

## 2. 总体架构

```
auth / oauth / gateway / identity
        │ publish audit events
        ▼
┌─────────────────────────────────┐
│   NATS JetStream (GGID_AUDIT)   │
└─────────────────────────────────┘
        │ consume (existing EventConsumer)
        ▼
┌──────────────────────────────────────────┐
│            audit service                  │
│  ┌────────────────┐   ┌───────────────┐  │
│  │ persist events │   │ DetectionEngine│  │
│  │ (existing)     │   │     (NEW)      │  │
│  └────────────────┘   └──────┬────────┘  │
│                              │ per event  │
│                   ┌──────────▼─────────┐  │
│                   │ RuleRegistry        │  │
│                   │ 6 built-in rules    │  │
│                   └──────────┬─────────┘  │
│                   match →    │            │
│         ┌────────────────────▼───────┐    │
│         │ itdr_detections (Postgres) │    │
│         └────────────────────┬───────┘    │
│         ┌────────────────────▼───────┐    │
│         │ Response Playbooks (v1:    │    │
│         │ NATS event + webhook;      │    │
│         │ v2: auto lock/revoke)      │    │
│         └────────────────────────────┘    │
└──────────────────────────────────────────┘
        │ query
        ▼
GET /api/v1/audit/itdr/detections[/:id]  (dashboard, SOC)
GET /api/v1/audit/itdr/stats             (dashboard counters)
CRUD /api/v1/audit/itdr/rules[/:id]      (rule config)
        │
        ▼
console ITDR pages + sdk/react hooks (替换假数据)
```

**关键设计决策**:

1. **检测引擎放在 audit 服务**（而非 auth）：事件流天然汇聚点；auth 保持轻量；检测规则可横向扩展。auth 现有的 on-demand 检测端点保留为"主动扫描"入口，引擎落地后改为查询引擎结果。
2. **复用现有 NATS consumer**：`EventConsumer` 已消费 JetStream 并持久化事件。在同一消费循环中增加 `engine.Evaluate(ctx, event)`，无新增基础设施。
3. **状态存 Redis**：有状态规则（滑动窗口计数、geo 历史）使用 Redis（已在所有部署中），引擎无状态可水平扩展。
4. **结果落 Postgres**：`itdr_detections` 表支持 SOC 工作流（ack/resolve）、多租户 RLS、与 audit_events 同库便于 JOIN 取证。

## 3. 核心组件

### 3.1 规则接口

位置：`services/audit/internal/detection/`

```go
// Rule 检测规则接口。无状态规则只看单事件；有状态规则通过 StateStore 访问窗口数据。
type Rule interface {
    ID() string                 // "impossible_travel"
    Name() string               // "Impossible Travel"
    MITRE() string              // "T1078" (可空)
    DefaultSeverity() Severity  // critical/high/medium/low

    // Evaluate 评估单个事件。返回 nil 表示未命中。
    Evaluate(ctx context.Context, evt *domain.AuditEvent, state StateStore, cfg RuleConfig) (*Detection, error)
}

// RuleConfig 每条规则的租户级配置（阈值、窗口、启停、严重级别覆盖）。
type RuleConfig struct {
    RuleID    string
    Enabled   bool
    Severity  string         // 覆盖默认
    Threshold map[string]any // 规则自定义参数, e.g. {"max_speed_kmh": 900}
}

// StateStore 有状态规则的窗口存储（Redis 实现）。
type StateStore interface {
    // AddEvent 记录窗口数据点（ZSET, score=unix_ts）
    AddEvent(ctx context.Context, key string, ts int64, member string, windowTTL time.Duration) error
    // EventsSince 读取窗口内数据点
    EventsSince(ctx context.Context, key string, since int64) ([]string, error)
    // Incr 窗口计数（用于失败次数）
    Incr(ctx context.Context, key string, windowTTL time.Duration) (int64, error)
}

// Detection 一次命中结果。
type Detection struct {
    ID         uuid.UUID
    TenantID   uuid.UUID
    RuleID     string
    ActorID    *uuid.UUID
    Severity   string
    Title      string
    Detail     map[string]any // 证据: 速度、窗口计数、源 IP 列表等
    EventIDs   []uuid.UUID    // 关联的 audit event ids（取证）
    Status     string         // new / acknowledged / resolved / false_positive
    DetectedAt time.Time
}
```

### 3.2 内置规则（迁移现有真实检测 + 补充）

| Rule ID | 类型 | 逻辑 | 来源 | MITRE |
|---------|------|------|------|-------|
| `brute_force` | stateful | 同一 user 5min 内 login 失败 ≥10 次 | credential_stuffing_handler 阈值复用 | T1110 |
| `credential_stuffing` | stateful | 同一 IP 10min 内尝试 ≥15 个不同账号 | credential_stuffing_handler 复用 | T1110.004 |
| `impossible_travel` | stateful | 同一 user 相邻成功登录 geo 速度 >900km/h（haversine）| impossible_travel_handler 算法原样迁移 | T1078 |
| `offhours_admin` | stateless | admin 角色在 00:00-06:00 (租户时区) 执行敏感操作 | 新增 | T1078 |
| `new_device_privileged` | stateful | 未见过的 UserAgent+IP 组合首次执行 role.assign/policy.update | 新增 | T1078 |
| `token_replay` | stateful | 同一 refresh token jti 在两个不同 IP 使用 | 新增（auth 已在 Redis 记录 jti） | T1550 |

规则注册表 `RuleRegistry`：启动时注册内置规则，从 `itdr_rules` 表加载租户级启停/阈值覆盖。

### 3.3 评估流程

```go
func (e *Engine) Evaluate(ctx context.Context, evt *domain.AuditEvent) error {
    rules := e.registry.RulesFor(evt.TenantID, evt.Action) // 按 action 索引, 避免全表评估
    for _, r := range rules {
        cfg := e.registry.ConfigFor(evt.TenantID, r.ID())
        if !cfg.Enabled { continue }
        det, err := r.Evaluate(ctx, evt, e.state, cfg)
        if err != nil {
            slog.Warn("rule evaluate error", "rule", r.ID(), "err", err)
            continue // 单规则失败不阻塞事件消费
        }
        if det != nil {
            if err := e.repo.InsertDetection(ctx, det); err != nil { ... }
            e.playbooks.Dispatch(ctx, det) // v1: 仅 critical/high 发 NATS + webhook
        }
    }
    return nil
}
```

- **性能**：规则按 `Action` 索引（"user.login" 只触发 login 相关 3 条规则）；Evaluate 在 consumer 批处理循环内同步执行，Redis RTT ~1ms，目标 P99 事件处理延迟 <50ms。
- **去重**：同一 (rule_id, actor_id, 5min bucket) 只产生一条 detection，后续命中更新 count 和 event_ids（SQL UPSERT）。
- **故障策略**：Redis 不可用时有状态规则跳过（记录 warn），无状态规则继续；引擎 panic 由 consumer 的 recover 兜底，不阻塞审计持久化主链路。

## 4. 数据模型（migration `010_itdr.sql`）

```sql
CREATE TABLE itdr_rules (
    id          TEXT NOT NULL,              -- rule_id, e.g. "brute_force"
    tenant_id   UUID NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    severity    TEXT,                        -- 覆盖默认 severity
    threshold   JSONB NOT NULL DEFAULT '{}',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (id, tenant_id)
);

CREATE TABLE itdr_detections (
    id           UUID PRIMARY KEY,
    tenant_id    UUID NOT NULL,
    rule_id      TEXT NOT NULL,
    actor_id     UUID,
    severity     TEXT NOT NULL,
    title        TEXT NOT NULL,
    detail       JSONB NOT NULL DEFAULT '{}',
    event_ids    UUID[] NOT NULL DEFAULT '{}',
    status       TEXT NOT NULL DEFAULT 'new',  -- new/acknowledged/resolved/false_positive
    hit_count    INT NOT NULL DEFAULT 1,       -- 去重累计
    detected_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_itdr_det_tenant_time ON itdr_detections (tenant_id, detected_at DESC);
CREATE INDEX idx_itdr_det_status ON itdr_detections (tenant_id, status, severity);
-- RLS 与 audit_events 同策略
ALTER TABLE itdr_detections ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON itdr_detections
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

## 5. API 设计（audit 服务新增，gateway 已代理 /api/v1/audit/）

| Method | Path | 说明 | 替代 |
|--------|------|------|------|
| GET | `/api/v1/audit/itdr/detections?severity=&status=&since=&rule_id=&page=` | 分页查询检测结果 | itdr-dashboard / threat-hunting 页面数据源 |
| GET | `/api/v1/audit/itdr/detections/{id}` | 单条详情（含 event_ids 取证链） | — |
| POST | `/api/v1/audit/itdr/detections/{id}/acknowledge` | SOC 确认 | — |
| POST | `/api/v1/audit/itdr/detections/{id}/resolve` | 关闭（body: resolution note / false_positive） | — |
| GET | `/api/v1/audit/itdr/stats?window=24h` | dashboard 计数：按 severity/status/rule 聚合 | 替代 `handleITDRDetections` 硬编码 |
| GET/POST | `/api/v1/audit/itdr/rules` | 规则列表 / 创建租户覆盖 | detection-rules 页面 |
| PUT/DELETE | `/api/v1/audit/itdr/rules/{id}` | 更新阈值/启停 / 恢复默认 | — |

auth 服务现有端点的迁移：
- `handleITDRDetections`（硬编码）→ **删除**，前端改调 `/api/v1/audit/itdr/*`
- `handleGoldenTicketDetect` / `handleLateralMovementDetect`（静态）→ 标记 deprecated，页面改查引擎（golden ticket 属于 AD/Kerberos 检测，GGID 无 KDC 数据源，规则暂不实现，页面从目录下架或标注"需外部 SIEM 数据源"）
- `handleDetectImpossibleTravel` / `handleDetectCredentialStuffing`（on-demand 真逻辑）→ 保留为"主动分析"工具端点；算法本体迁移为引擎规则（共享 `internal/detection` 包或复制算法，避免 auth→audit 服务依赖）

## 6. Response Playbooks

**v1（本设计范围）**：
- detection 产生时按 severity 分发：
  - critical/high → 发布 NATS 事件 `ggid.itdr.detection`（subject）+ 调用租户配置的 webhook（`itdr_rules` 同表加 webhook_url 或 system config）
  - medium/low → 仅落库
- webhook payload: detection JSON，HMAC-SHA256 签名头（复用 webhook 签名基础设施）

**v2（后续迭代，本设计仅留接口）**：
- 自动响应动作：lock_account（调 identity admin API）、revoke_sessions（调 auth admin API）、require_stepup（写 Redis 风险标记，auth 登录时检查）
- playbook 定义表 + 执行审计（每步结果写回 detection.detail.execution）

## 7. 前端改造点（与 P0 假数据清理配合）

| 页面/hook | 改造 |
|-----------|------|
| `useITDRDashboard` | 接 `GET /audit/itdr/stats` + `/detections` |
| threat-hunting-workbench | 接 `/detections` 查询 + 自定义 filter |
| anomaly-detect-dashboard | 接 `/detections?rule_id=anomaly_*` |
| golden-ticket/lateral-movement 页面 | 标注"需 Kerberos/AD 数据源"或下架（无真实数据源） |
| 150 个假 hooks | 按 frontend 的 P0 清理计划分批接线/标注 |

## 8. 实施计划

| Phase | 内容 | 文件 | 预估 |
|-------|------|------|------|
| 1 | migration 010_itdr.sql + Detection 模型 + repo | migrations/, services/audit/internal/{domain,repository}/ | 0.5d |
| 2 | Engine + StateStore(Redis) + RuleRegistry + 3 条迁移规则（brute_force/credential_stuffing/impossible_travel） | services/audit/internal/detection/ | 1.5d |
| 3 | consumer 接线 Evaluate + API handlers + 路由 | consumer/, server/, handler/ | 1d |
| 4 | 剩余 3 条新规则 + playbook v1 webhook | detection/, playbooks | 1d |
| 5 | auth 硬编码 handler 删除/deprecate + 单测/E2E | services/auth/ | 0.5d |
| 6 | frontend hooks 接线（与 P0 清理协同） | sdk/react/, console/ | 由 frontend 排期 |

总计 backend ~4.5 天。测试要求：每条规则 ≥3 单测（命中/未命中/边界），engine 并发消费测试，API handler 测试，build+make test 全绿。

## 9. 风险与缓解

| 风险 | 缓解 |
|------|------|
| 引擎拖慢审计消费主链路 | Evaluate 同步但有 50ms 预算；超预算规则可通过配置移到异步队列 |
| Redis 状态丢失导致漏检 | 窗口数据本身是易失的（TTL ≤24h），可接受；detection 结果落 PG 不丢 |
| 误报洪水（如 brute_force 阈值过低）| 去重 bucket + 默认阈值来自现有生产逻辑 + 租户可配 |
| golden-ticket 等无数据源规则误导用户 | 页面明确标注数据源要求，不伪造检测结果 |
