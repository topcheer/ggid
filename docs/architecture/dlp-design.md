# DLP (Data Loss Prevention) 设计 — Identity-Based

> 状态: Proposed | 作者: IAMExpert | 日期: 2026-07-17 | 来源: arch 指派
> 关联: 数据安全法 (data-security-compliance-design.md $data.classification)、ITDR、CAE

## 端点 API 契约

### POST /api/v1/identity/dlp/policies — 创建策略

Request:
```json
{
  "name": "Block core data export by non-admin",
  "trigger": "export",
  "conditions": {
    "and": [
      {"$data.classification": "core"},
      {"$user.role": {"$ne": "admin"}}
    ]
  },
  "action": "block",
  "webhook_url": "https://soc.example.com/dlp-alert"
}
```

Response 201:
```json
{
  "id": "01234567-89ab-cdef-...",
  "name": "Block core data export by non-admin",
  "trigger": "export",
  "conditions": {"and": [...]},
  "action": "block",
  "enabled": true,
  "created_at": "2026-07-17T22:00:00Z"
}
```

### GET /api/v1/identity/dlp/policies?enabled=true — 列表

Response 200:
```json
{"policies": [...], "total": 3}
```

### GET /api/v1/audit/dlp/events?severity=blocked&since=24h — 拦截事件

Response 200:
```json
{
  "events": [
    {
      "id": "...",
      "policy_id": "...",
      "user_id": "...",
      "user_name": "viewer@ggid.dev",
      "trigger": "export",
      "resource_type": "audit_events",
      "data_classification": "core",
      "action_taken": "blocked",
      "reason": "core data export by non-admin",
      "timestamp": "2026-07-17T22:05:00Z"
    }
  ],
  "total": 1
}
```

### POST /api/v1/identity/dlp/policies/{id}/test — 策略模拟

Request:
```json
{
  "user_id": "01234567-...",
  "resource_type": "audit_events",
  "trigger": "export"
}
```

Response 200:
```json
{
  "matched": true,
  "policy_id": "...",
  "action": "block",
  "reason": "core data export by non-admin",
  "data_classification": "core"
}
```

## curl 验收命令

```bash
# 创建 DLP 策略
curl -X POST http://localhost:8080/api/v1/identity/dlp/policies \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{"name":"Block core export","trigger":"export","conditions":{"and":[{"$data.classification":"core"},{"$user.role":{"$ne":"admin"}}]},"action":"block"}'

# 查询拦截事件
curl http://localhost:8080/api/v1/audit/dlp/events?severity=blocked \
  -H "Authorization: Bearer $TOKEN"

# 策略模拟
curl -X POST http://localhost:8080/api/v1/identity/dlp/policies/{id}/test \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"user_id":"...","resource_type":"audit_events","trigger":"export"}'
```

## $data.classification 联动

DLP conditions 中 `$data.classification` 查 data_classifications 表（B-18 已实现）：
- export 请求 → 查 resource_type 的 classification → core + non-admin → block
- important + non-admin → mask（PII 脱敏后导出）
- general → log only

与 PDP（ztp-pdp-design.md）SecurityContext 复用 SignalCollector 注入 $data.classification。

## 反模式禁令

- 禁止 log.Printf 占位代替实际 block/mask 执行
- 禁止内存 map 存策略（必须 PostgreSQL）
- 禁止 alg=none 签名（DLP 不涉及签名但审计链需 HMAC）
