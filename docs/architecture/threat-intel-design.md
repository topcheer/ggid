# Threat Intelligence Integration Hub 设计

> 状态: Proposed | 作者: IAMExpert | 日期: 2026-07-18 | 来源: arch 指派 + 第21小时研究
> 关联: ITDR 引擎、CAE risk engine、threat_intel_feed_handler.go (现有 stub 69 行)

## 1. 愿景

GGID ITDR 从纯内部规则检测升级为内外联动 — 接入外部威胁情报源（AlienVault OTX/AbuseIPDB/HaveIBeenPwned/MISP），在 login/access 时实时查询已知恶意 IP/泄露凭据/恶意域名，提升检测准确率并降低误报。

**现有资产**：
- ITDR 引擎 ✓（6+ 检测规则：brute_force/credential_stuffing/impossible_travel/baseline_deviation）
- CAE risk engine ✓
- threat_intel_feed_handler.go stub（69 行，返回 hardcoded 数据）
- audit/threat-feed endpoint ✓（注册但不查外部源）

**缺**：外部情报采集 + indicators 持久化 + ITDR/CAE 联动。

## 2. 数据模型

### threat_intel_sources

```sql
CREATE TABLE threat_intel_sources (
    id              UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    name            TEXT NOT NULL,           -- "AlienVault OTX"
    source_type     TEXT NOT NULL,           -- 'ip' | 'credential' | 'domain' | 'url'
    api_endpoint    TEXT NOT NULL,           -- "https://otx.alienvault.com/api/v1/indicators"
    api_key_ref     TEXT,                    -- Vault/secret broker reference (B-36)
    poll_interval   INTERVAL NOT NULL DEFAULT '1 hour',
    last_poll       TIMESTAMPTZ,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, name)
);
```

### threat_indicators

```sql
CREATE TABLE threat_indicators (
    id              UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    source_id       UUID NOT NULL REFERENCES threat_intel_sources(id),
    indicator_type  TEXT NOT NULL,           -- 'ip' | 'email' | 'credential_hash' | 'domain' | 'url'
    indicator_value TEXT NOT NULL,           -- "192.168.1.100" / "user@domain.com"
    severity        TEXT NOT NULL DEFAULT 'medium',  -- 'low'|'medium'|'high'|'critical'
    confidence      INT NOT NULL DEFAULT 50, -- 0-100
    first_seen      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen       TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ,             -- TTL 过期自动清理
    metadata        JSONB DEFAULT '{}',      -- 源特定字段
    UNIQUE(tenant_id, indicator_type, indicator_value)
);

CREATE INDEX idx_threat_indicators_lookup ON threat_indicators(tenant_id, indicator_type, indicator_value);
CREATE INDEX idx_threat_indicators_expiry ON threat_indicators(expires_at) WHERE expires_at IS NOT NULL;
```

## 3. 端点 API 契约

### GET /api/v1/audit/threat-intel/sources — 列表
Response 200:
```json
{"sources": [{"id":"...","name":"AbuseIPDB","source_type":"ip","enabled":true,"last_poll":"..."}], "total": 2}
```

### POST /api/v1/audit/threat-intel/sources — 创建情报源
Request:
```json
{"name":"AbuseIPDB","source_type":"ip","api_endpoint":"https://api.abuseipdb.com/api/v2/check","api_key_ref":"vault://threat-intel/abuseipdb","poll_interval":"1h"}
```
Response 201:
```json
{"id":"01234567-...","name":"AbuseIPDB","source_type":"ip","enabled":true,"created_at":"..."}
```

### GET /api/v1/audit/threat-intel/indicators?type=ip&page_size=50 — 查询 indicators
Response 200:
```json
{"indicators": [{"id":"...","indicator_type":"ip","indicator_value":"1.2.3.4","severity":"high","confidence":85}], "total": 1500}
```

### POST /api/v1/audit/threat-intel/check — 实时检查
Request:
```json
{"indicator":"192.168.1.100","indicator_type":"ip"}
```
Response 200 (match):
```json
{"matched": true, "severity": "high", "confidence": 90, "source": "AbuseIPDB", "description": "Known SSH brute-force source"}
```
Response 200 (no match):
```json
{"matched": false}
```

### GET /api/v1/audit/threat-intel/stats — 统计
Response 200:
```json
{"sources_enabled": 3, "indicators_total": 5400, "hits_24h": 42, "by_type": {"ip": 3200, "credential": 1800, "domain": 400}}
```

## 4. curl 验收命令

```bash
# 注册情报源
curl -X POST http://localhost:8080/api/v1/audit/threat-intel/sources \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" -H "Content-Type: application/json" \
  -d '{"name":"AbuseIPDB","source_type":"ip","api_endpoint":"https://api.abuseipdb.com/api/v2/check","poll_interval":"1h"}'

# 实时检查 IP
curl -X POST http://localhost:8080/api/v1/audit/threat-intel/check \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"indicator":"192.168.1.100","indicator_type":"ip"}'

# 统计
curl http://localhost:8080/api/v1/audit/threat-intel/stats -H "Authorization: Bearer $TOKEN"

# 查询 indicators
curl "http://localhost:8080/api/v1/audit/threat-intel/indicators?type=ip&page_size=50" -H "Authorization: Bearer $TOKEN"
```

## 5. 外部情报源适配器

| 源 | 类型 | API | 适配逻辑 |
|---|---|---|---|
| AlienVault OTX | ip/domain/url | GET /api/v1/indicators/{type}/{value} | 解析 pulses → indicators |
| AbuseIPDB | ip | POST /api/v2/check | abuseConfidenceScore → severity |
| HaveIBeenPwned | credential | GET /range/{hash_prefix} | k-anonymity 模式 |
| MISP | all | POST /events/restSearch | CIRCL OSINT feed |
| Spamhaus | domain | TXT DNS query | domain → DBL lookup |

每个适配器实现 `IntelAdapter` 接口：
```go
type IntelAdapter interface {
    Fetch(ctx context.Context, client *http.Client) ([]ThreatIndicator, error)
    SourceType() string
}
```

## 6. ITDR + CAE 联动

### ITDR 注入点
```go
// 在 ITDR Evaluate 前，查询 threat_indicators
func (e *Engine) checkThreatIntel(ctx context.Context, evt *domain.AuditEvent) *domain.Detection {
    // 查 IP
    if indicator, err := e.threatRepo.Check(ctx, "ip", evt.SourceIP); err == nil && indicator != nil {
        return &domain.Detection{
            RuleID:   "threat_intel_hit",
            Severity: indicator.Severity,
            Reason:   fmt.Sprintf("IP %s in threat feed (%s, confidence %d%%)", evt.SourceIP, indicator.Source, indicator.Confidence),
        }
    }
    // 查 email/credential
    // ...
    return nil
}
```

### CAE 联动
- threat_intel_hit detection → risk_engine 提升 risk score
- high/critical hit → 触发 step-up MFA 或 session revoke
- critical hit → SOC webhook + ITDR 复合规则联动（ransomware precursor）

## 7. 定时采集

```go
// 每 poll_interval 执行一次
func (c *IntelCollector) Run(ctx context.Context) {
    sources, _ := c.repo.ListEnabled(ctx)
    for _, src := range sources {
        adapter := getAdapter(src.SourceType)
        indicators, err := adapter.Fetch(ctx, c.client)
        if err != nil {
            log.Printf("threat intel poll error: %v", err)
            continue
        }
        c.repo.UpsertIndicators(ctx, src.ID, indicators)
        c.repo.UpdateLastPoll(ctx, src.ID)
    }
    // 清理过期 indicators
    c.repo.DeleteExpired(ctx)
}
```

## 8. 反模式禁令

- 禁止 log.Printf 占位代替实际外部 API 调用
- 禁止内存 map 存储 indicators（必须 PostgreSQL threat_indicators 表）
- 禁止 hardcoded JSON 假数据（threat_intel_feed_handler.go 现 stub 必须替换）
- 禁止同步阻塞采集（必须 async goroutine + context timeout）
- 外部 API 必须有 SSRF 保护 + 超时 + 重试

## 9. 测试计划

| 测试 | 验证点 |
|------|--------|
| Source CRUD | 创建/查询/更新/删除情报源 |
| Indicator upsert | 重复 indicator 更新 last_seen 而非报错 |
| Check match | 已知恶意 IP → matched:true |
| Check no match | 未知 IP → matched:false |
| Adapter OTX | mock OTX API → 正确解析 pulses |
| Adapter AbuseIPDB | mock API → abuseConfidenceScore → severity |
| Collector poll | 定时执行 → indicators 写入 |
| TTL expiry | 过期 indicators 自动删除 |
| ITDR injection | threat_intel_hit detection 正确生成 |
| Nil pool fallback | DB 不可用时降级安全 |
