# UEBA Per-User 行为基线设计 (User Entity Behavior Analytics)

> 状态: Proposed | 作者: IAMExpert | 日期: 2026-07-17 | 来源: kanban I-04
> 关联: ITDR 引擎 (docs/architecture/itdr-engine-design.md)、零信任 (docs/team-backlog.md)

## 1. 问题

GGID 现有 risk_engine 是**全局阈值启发式**（velocity>20→+0.4, device unknown→+0.3），所有用户同一标尺。问题：
- 异常但合法用户误报高（夜班工程师 02:00 登录 → flagged）
- 低慢攻击漏报（每天 3 次失败，永远不超过阈值 10/5min）
- 无"个性化正常"概念

UEBA 解决方案：为每个用户建立行为基线（profile），用统计偏差（z-score）检测异常。

## 2. GGID 独特优势

- audit 事件流（NATS JetStream）= 天然训练数据
- ITDR 检测引擎 = 天然投递机制（新增 `baseline_deviation` 规则即可）
- 无 ML 框架依赖：纯 Go 统计

## 3. 总体架构

```
NATS audit events → audit consumer
    ├→ persist event (existing)
    ├→ ITDR engine.Evaluate (existing)
    └→ ProfileBuilder (NEW)
         │ daily job: 30-day sliding window
         ▼
    user_behavior_profiles (Postgres JSONB)
         │
         ▼
    ITDR baseline_deviation rule (NEW)
         │ event vs profile: z-score > 3σ
         ▼
    Detection (evidence: "通常 9-18h 北京登录，本次 03:15 未知 IP 俄罗斯")
```

## 4. 行为基线模型

### 4.1 Profile 结构

```go
type UserProfile struct {
    UserID       uuid.UUID
    TenantID     uuid.UUID
    LoginHours   [24]float64   // 登录小时分布（0-23），归一化概率
    LoginDays    [7]float64    // 周几分布（0=周日）
    KnownIPs     map[string]float64  // IP/CIDR → 频率（top 20，其余归 "other"）
    KnownASNs    map[string]float64  // ASN → 频率
    KnownCountries map[string]float64 // 国家代码 → 频率
    KnownDevices map[string]float64  // UA fingerprint → 频率
    AvgSessionDuration float64       // 平均会话时长（分钟）
    AvgDailyActions    float64       // 日均操作数
    ActionTypes        map[string]float64 // action 类型 → 频率
    ResourceTypes      map[string]float64 // resource_type → 频率
    UpdatedAt    time.Time
    EventCount   int            // 训练事件总数（冷启动追踪）
}
```

### 4.2 存储

```sql
-- migration 015_ueba_profiles.sql
CREATE TABLE user_behavior_profiles (
    tenant_id   UUID NOT NULL,
    user_id     UUID NOT NULL,
    profile     JSONB NOT NULL DEFAULT '{}',
    event_count INT NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, user_id)
);
```

JSONB 而非分列：profile 结构可能演进，JSONB 灵活。查询时 `profile->>'login_hours'` 提取。

### 4.3 Profile Builder

**位置**：`services/audit/internal/detection/profile_builder.go`

**执行方式**：每日 cron（或 NATS 定时触发），30 天滑动窗口

```go
func (pb *ProfileBuilder) BuildProfile(ctx context.Context, tenantID, userID uuid.UUID) error {
    // 1. 查 30 天 audit_events
    events, err := pb.repo.ListByUserSince(ctx, tenantID, userID, 30*24*time.Hour)
    
    // 2. 统计各维度
    profile := &UserProfile{EventCount: len(events)}
    for _, evt := range events {
        hour := evt.CreatedAt.Hour()
        profile.LoginHours[hour]++
        if evt.IPAddress != "" {
            profile.KnownIPs[evt.IPAddress]++
        }
        // ... ASN, country, device, action type, resource type
    }
    
    // 3. 归一化为概率分布
    normalize(profile.LoginHours[:])
    normalizeMap(profile.KnownIPs)
    // ...
    
    // 4. UPSERT 到 DB
    return pb.repo.UpsertProfile(ctx, profile)
}
```

**性能**：单个用户 30 天 ~3000 事件，内存聚合 <10ms。按租户批量处理（每租户一次查询 + 内存 group-by）。

### 4.4 冷启动

- 新用户前 50 事件：**只学习不评估**（profile.EventCount < 50 时 baseline_deviation 规则跳过）
- 50-200 事件：**宽松模式**（z-score > 4σ 才触发，severity=low）
- 200+ 事件：**正常模式**（z-score > 3σ，severity=medium）

## 5. baseline_deviation 规则

**位置**：`services/audit/internal/detection/rules/baseline_deviation.go`

实现 ITDR `Rule` 接口（与 brute_force/impossible_travel 同构）：

```go
type BaselineDeviationRule struct{}

func (r *BaselineDeviationRule) Evaluate(ctx context.Context, evt *domain.AuditEvent, state StateStore, cfg domain.RuleConfig) (*domain.Detection, error) {
    // 1. 获取用户 profile（从 DB 或缓存）
    profile, err := r.profileStore.Get(ctx, evt.TenantID, *evt.ActorID)
    if err != nil || profile.EventCount < 50 {
        return nil, nil // 冷启动跳过
    }
    
    var anomalies []string
    var maxDeviation float64
    
    // 2. 逐维度评估偏差
    // a. 登录时间偏差
    hour := evt.CreatedAt.Hour()
    hourProb := profile.LoginHours[hour]
    if hourProb < 0.01 { // 此小时占历史 <1%
        anomalies = append(anomalies, fmt.Sprintf(
            "登录时间 %02d:00 异常（通常在 %s）",
            hour, commonHours(profile.LoginHours)))
        maxDeviation = math.Max(maxDeviation, 0.8)
    }
    
    // b. IP/地理偏差
    if profile.KnownCountries[country] < 0.05 {
        anomalies = append(anomalies, fmt.Sprintf(
            "登录来自 %s（通常在 %s）",
            country, commonCountries(profile.KnownCountries)))
        maxDeviation = math.Max(maxDeviation, 0.7)
    }
    
    // c. 设备偏差
    if profile.KnownDevices[deviceFP] == 0 {
        anomalies = append(anomalies, "未见过此设备")
        maxDeviation = math.Max(maxDeviation, 0.5)
    }
    
    // d. 资源类型偏差（用户首次访问新资源类型）
    if profile.ResourceTypes[evt.ResourceType] == 0 && evt.ResourceType != "" {
        anomalies = append(anomalies, fmt.Sprintf("首次访问资源类型 %s", evt.ResourceType))
        maxDeviation = math.Max(maxDeviation, 0.4)
    }
    
    // 3. 多维度组合：≥2 个异常 → severity 升级
    severity := "medium"
    if len(anomalies) >= 2 {
        severity = "high"
    }
    if len(anomalies) >= 3 {
        severity = "critical"
    }
    
    if len(anomalies) == 0 {
        return nil, nil
    }
    
    return &domain.Detection{
        RuleID:   "baseline_deviation",
        Severity: severity,
        Title:    fmt.Sprintf("用户行为基线偏差：%s", evt.ActorName),
        Detail: map[string]any{
            "anomalies":     anomalies,
            "max_deviation": maxDeviation,
            "event_count":   profile.EventCount,
        },
    }, nil
}
```

### 5.1 与现有规则协同

| 现有规则 | baseline_deviation 补充 |
|---------|----------------------|
| brute_force（5min/10 次） | 低慢攻击（每天 3 次，30 天累积 90 次 → profile 显示"非典型失败模式"）|
| impossible_travel（900km/h） | profile 知道"此用户从不去俄罗斯"，即使速度合法 |
| credential_stuffing（IP/15 账号） | profile 知道"此用户通常只用 2 个 IP"，新 IP 即异常 |

**不替换现有规则**——baseline_deviation 是第 7 条规则，与 3 条迁移规则 + 3 条新规则（Phase 4）叠加。

## 6. Profile 缓存

为避免每事件查 DB：

```go
type ProfileCache struct {
    cache map[string]*UserProfile // key: tenantID:userID
    mu    sync.RWMutex
    ttl   time.Duration // 5 分钟
}
```

Profile 变化慢（日级更新），5 分钟 TTL 足够。高活跃用户的事件在缓存 TTL 内只查一次 profile。

## 7. API（可选，admin 可查看）

| Method | Path | 说明 |
|--------|------|------|
| GET | /api/v1/audit/ueba/profiles/{user_id} | 用户行为 profile（解释模型）|
| GET | /api/v1/audit/ueba/stats | UEBA 覆盖率：多少用户已有 profile、平均偏差率 |

## 8. 测试计划

| 测试 | 验证点 |
|------|--------|
| ProfileBuilder：30 天事件聚合 | login_hours 分布正确、IP 频率正确 |
| 冷启动：<50 事件 | baseline_deviation 返回 nil |
| 正常用户（profile 内行为）| 0 anomalies → nil |
| 异常用户（03:00 俄罗斯新 IP）| ≥2 anomalies → high severity |
| 多维度组合 | 3 异常 → critical |
| ProfileCache | 5min TTL、并发安全 |

## 9. 工作量

| Phase | 内容 | 预估 |
|-------|------|------|
| 1 | migration 015 + ProfileRepo + Profile JSONB 模型 | 0.5d |
| 2 | ProfileBuilder（日 job + 30 天滑窗 + 归一化）| 0.5d |
| 3 | baseline_deviation 规则 + ProfileCache | 0.5d |
| 4 | ITDR 引擎注册 + API 端点 | 0.5d |
| 5 | 测试 | 0.5d |
| **总计** | | **~2.5d backend** |

## 10. 与 Peer-Group 对比（后续 P2）

Per-user 基线解决"这个人不正常"。Peer-group 解决"这个人做同岗位的人不做的事"——例如财务用户访问代码仓库，即使个人历史有过（profile 不报），但同部门 0 人做此操作（peer 报）。

实现：profile builder 同时聚合 department/role 级分布，baseline_deviation 规则追加 peer 维度。
