# 零信任统一策略决策点 (Unified PDP)

> 状态: Proposed | 作者: IAMExpert | 日期: 2026-07-17 | 来源: kanban I-05
> 关联: ITDR (docs/architecture/itdr-engine-design.md)、CAE (docs/architecture/cae-design.md)、UEBA (docs/architecture/ueba-design.md)

## 1. 问题

NIST SP 800-207 零信任参考架构核心：统一 PDP（Policy Decision Point）/ PEP（Policy Enforcement Point）。

**GGID 现状**：安全信号分散在各服务，策略评估时无法组合：

| 信号 | 来源服务 | 现状 |
|------|---------|------|
| 用户角色/权限 | policy | CheckPermission（RBAC + ABAC 条件） |
| 设备信任状态 | auth | device_trust_handler（独立查询） |
| ITDR open detections | audit | itdr_detections 表（独立查询） |
| Session 风险评分 | audit | risk_engine.Evaluate（独立调用） |
| UEBA 基线偏差 | audit | user_behavior_profiles（设计阶段） |

**无法表达**："仅当设备可信 AND 无未处理 critical detection AND 工作时间 → 允许导出报表"——因为各维度不在同一评估上下文中。

## 2. 设计目标

扩展现有 policy 服务 CheckPermission，注入运行时安全上下文，ABAC 条件 DSL 增加**内置属性**引用外部信号源。

**关键约束**：不重写 policy 引擎，只扩展输入维度。

## 3. 统一评估上下文

### 3.1 SecurityContext

```go
// SecurityContext 在策略评估时注入，聚合来自多服务的实时信号。
type SecurityContext struct {
    UserID       uuid.UUID
    TenantID     uuid.UUID
    SessionID    uuid.UUID
    
    // Device signals (from auth)
    DeviceTrusted    bool
    DeviceFingerprint string
    
    // ITDR signals (from audit)
    OpenCriticalCount int   // 未处理的 critical detections
    OpenHighCount     int
    
    // Risk signals (from audit risk_engine)
    SessionRiskScore  float64  // 0.0-1.0
    GeoAnomaly        bool
    
    // Temporal
    RequestTime       time.Time
    UserLocalTime     time.Time // 用户时区
    
    // UEBA (optional, when profile exists)
    BaselineDeviation  float64 // 最近偏差分数
}
```

### 3.2 信号收集器（异步预取 + 缓存）

```go
type SignalCollector struct {
    authClient    AuthSignalClient   // GET /api/v1/auth/devices/trusted?user_id=
    auditClient   AuditSignalClient  // GET /api/v1/audit/itdr/stats?user_id=
    riskEngine    *RiskEngine        // 本地（同进程或 Redis 缓存）
    cache         *SecurityCache     // 5s TTL
}

func (sc *SignalCollector) Collect(ctx context.Context, userID, tenantID uuid.UUID) (*SecurityContext, error) {
    // 检查缓存
    if cached, ok := sc.cache.Get(userID); ok {
        return cached, nil
    }
    // 并发收集（errgroup，超时 200ms each）
    var wg errgroup.Group
    var devAuth bool
    var itdrOpen int
    var risk float64
    
    wg.Go(func() error { devAuth = sc.authClient.GetDeviceTrust(ctx, userID); return nil })
    wg.Go(func() error { itdrOpen = sc.auditClient.GetOpenCritical(ctx, userID); return nil })
    wg.Go(func() error { risk = sc.riskEngine.Evaluate(userID, ...).Score; return nil })
    wg.Wait()
    
    secCtx := &SecurityContext{...}
    sc.cache.Set(userID, secCtx, 5*time.Second)
    return secCtx, nil
}
```

**性能**：3 个并发查询 × 200ms 超时 = 总计 <200ms。5s 缓存覆盖同一用户连续请求。降级：单信号源超时 → 默认值（device_trusted=false、itdr_open=0），不阻塞评估。

## 4. ABAC 条件 DSL 扩展

### 4.1 现有条件（保持兼容）

现有 policy 条件基于请求属性（resource_type, action, IP 等）。扩展增加 `$security.*` 内置属性：

### 4.2 新增内置属性

| 属性 | 类型 | 来源 | 示例值 |
|------|------|------|--------|
| `$security.device_trusted` | bool | auth | true / false |
| `$security.itdr_critical_open` | int | audit | 0, 1, 2... |
| `$security.session_risk` | float | audit | 0.0-1.0 |
| `$security.geo_anomaly` | bool | audit | true / false |
| `$security.is_business_hours` | bool | computed | true (9-18h user TZ) |
| `$security.baseline_deviation` | float | audit UEBA | 0.0-1.0 |

### 4.3 条件表达式示例

```json
{
  "effect": "allow",
  "conditions": {
    "and": [
      {"$security.device_trusted": true},
      {"$security.itdr_critical_open": {"$eq": 0}},
      {"$security.session_risk": {"$lt": 0.7}}
    ]
  }
}
```

或：allow if business_hours AND NOT geo_anomaly；deny if itdr_critical_open > 0。

### 4.4 评估流程

```
CheckPermission(userID, resource, action)
    │
    ├─ 1. RBAC: user has role with permission? (existing)
    │
    ├─ 2. ABAC: policy conditions match? (existing + extended)
    │      ├─ request attributes (resource_type, action) — existing
    │      └─ $security.* attributes — NEW: SignalCollector.Collect → inject
    │
    └─ 3. Effect: allow / deny / deny_with_stepup
```

**deny_with_stepup**（新增 effect）：不是直接拒绝，而是要求 MFA 升级后重新评估。适用于 `$security.session_risk > 0.5` 但用户未做错事的灰区场景。

## 5. API 扩展

现有端点不变。新增评估上下文注入：

| Method | Path | 说明 |
|--------|------|------|
| POST | /api/v1/policies/check-with-context | 带显式 SecurityContext 的评估（测试/调试用） |
| GET | /api/v1/policies/security-context/{user_id} | 查看用户当前安全上下文（admin 可视化）|

内部使用：gateway 或 auth 服务在评估权限时自动注入 SignalCollector 结果。

## 6. 实现路径

| Phase | 内容 | 文件 | 预估 |
|-------|------|------|------|
| 1 | SecurityContext 模型 + SignalCollector + 缓存 | services/policy/internal/service/ | 0.5d |
| 2 | ABAC Evaluator 扩展 $security.* 解析 | services/policy/internal/service/condition_groups.go | 0.5d |
| 3 | deny_with_stepup effect + auth 联动 | services/policy/ + services/auth/ | 0.5d |
| 4 | API 端点 + 测试 | services/policy/internal/server/ | 0.5d |
| **总计** | | | **~2d** |

## 7. 与现有组件协同

| 组件 | 关系 |
|------|------|
| CAE | $security.itdr_critical_open 源自 ITDR → critical 时 CAE 也 revoke session，双重防护 |
| UEBA | $security.baseline_deviation 源自 UEBA profile，PDP 自动利用 |
| PAM JIT | JIT 提权时也过 PDP：临时角色 + 当前安全上下文组合评估 |
| Risk Engine | 现有 risk_engine.Evaluate 作为 $security.session_risk 信号源，不替换 |

## 8. 缓存一致性

5s TTL 缓存意味着 ITDR critical detection 后最多 5s 延迟才反映到 PDP 评估。紧急场景（CAE 已在 <1s 内 revoke session）不受影响——PDP 评估的延迟窗口仅影响"允许/拒绝新请求"，已 revoke 的 session 由 gateway CAECheck 拦截。

可配置：高安全租户可设 TTL=0（每次实时查询，延迟 ~200ms）。
