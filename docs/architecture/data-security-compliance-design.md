# 数据安全法合规引擎设计 (Data Security Compliance Engine)

> 状态: Proposed | 作者: IAMExpert | 日期: 2026-07-17 | 来源: arch 下 sprint 排期 #2
> 关联: 国密 SM2/SM3/SM4 (已实现)、ZT PDP ($data.classification)、ITDR 审计链

## 1. 问题

《数据安全法》第 21 条强制要求数据分类分级保护。《网络数据安全管理条例》(2025) + GB/T 43697-2024 数据分类分级国家标准提供实操指引。分类为：**一般 / 重要 / 核心**，每级有差异化的访问控制、加密、审计、出境管控要求。

**GGID 现状审计**：

| 组件 | 现状 | 缺口 |
|------|------|------|
| PIPL data inventory handler | 硬编码 7 行假数据（basic_info/biometric/location...） | 无 DB 持久化、无动态管理 |
| attribute governance handler | 硬编码 6 个属性（ssn/email/phone...） | 同上 |
| compliance config handler | 有 `DataClassificationRules map[string]string` | 仅配置壳，无评估逻辑 |
| audit retention | RetentionPolicy 有（按天数删除） | 无按数据级别差异化保留 |
| PDP (ztp-pdp-design.md) | 设计中 $data.classification 属性 | 未实现 |

**第 6 个"UI 有后端空壳"模式** — 分类分级的概念存在于多个 handler 但全部硬编码，无统一分类引擎。

## 2. 设计目标

统一的数据分类分级标签体系 → 与 PDP 联动（差异化访问控制）→ 差异化审计保留 → 出境管控。

**关键约束**：不做数据扫描/发现（那是 DLP 产品的事），只做**标签管理 + 标签驱动的策略评估**。

## 3. 数据模型

### 3.1 migration 016_data_classification.sql

```sql
-- 数据资源分类标签
CREATE TABLE data_classifications (
    id              UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    resource_type   TEXT NOT NULL,         -- 'user_attribute', 'audit_log', 'api_endpoint', 'custom'
    resource_id     TEXT NOT NULL,          -- 具体资源标识 (e.g. 'users.email', 'audit.events')
    classification   TEXT NOT NULL,         -- 'general' | 'important' | 'core'
    category        TEXT,                   -- 'pii' | 'biometric' | 'financial' | 'health' | 'identity_doc'
    lawful_basis    TEXT,                   -- PIPL 合法基础
    retention_days  INT,                    -- 差异化保留期限
    cross_border    TEXT NOT NULL DEFAULT 'allowed', -- 'allowed' | 'restricted' | 'prohibited'
    mask_rule       TEXT,                   -- 'full_mask' | 'partial_mask' | 'none'
    created_by      UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, resource_type, resource_id)
);
CREATE INDEX idx_data_class_lookup ON data_classifications(tenant_id, resource_type, resource_id);

-- RLS
ALTER TABLE data_classifications ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_iso ON data_classifications
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

### 3.2 分类级别定义

| 级别 | 中文 | 访问要求 | 加密 | 审计保留 | 出境 |
|------|------|---------|------|---------|------|
| general | 一般数据 | 标准 RBAC | AES-256 | 1 年 | 允许 |
| important | 重要数据 | ABAC + MFA | AES-256 + 传输加密 | 3 年 | 限制（需申报） |
| core | 核心数据 | ABAC + JIT + CAE | 国密 SM4（可选）| 7 年 | 禁止 |

## 4. 分类标签 → PDP 联动

扩展 ZT PDP（docs/architecture/ztp-pdp-design.md）的 SecurityContext：

```go
type SecurityContext struct {
    // ... existing fields ...
    DataClassification string  // 当前请求访问数据的分类级别
}
```

ABAC 条件 DSL 新增：

```json
{
  "effect": "allow",
  "conditions": {
    "and": [
      {"$data.classification": {"$in": ["general", "important"]}},
      {"$security.device_trusted": true}
    ]
  }
}
```

**默认策略**（无需显式配置，内置）：
- core 级数据：自动要求 device_trusted + JIT + no ITDR critical + MFA stepup
- important 级数据：自动要求 device_trusted + MFA
- general 级数据：标准 RBAC

## 5. 差异化审计保留

扩展现有 RetentionPolicy（按天数）→ 按数据级别：

```go
type ClassificationRetentionPolicy struct {
    GeneralDays   int  // 365
    ImportantDays int  // 1095
    CoreDays      int  // 2555
}

// Apply 对 audit_events 按 resource 的分类级别执行差异化清理。
func (p *ClassificationRetentionPolicy) Apply(ctx context.Context, deleter EventDeleter) (*Result, error) {
    // JOIN data_classifications ON resource_type + resource_id
    // WHERE created_at < now() - retention_days(for that classification)
}
```

## 6. PII 屏蔽规则

```go
// MaskValue 根据数据分类的 mask_rule 屏蔽值。
func MaskValue(value string, rule string) string {
    switch rule {
    case "full_mask":
        return strings.Repeat("*", len(value))  // ssn: ***********
    case "partial_mask":
        if len(value) <= 4 { return "****" }
        return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]  // em**@***om
    default:
        return value
    }
}
```

API 响应序列化时自动应用 mask（基于请求者的权限 + 数据分类标签）。

## 7. API 设计

| Method | Path | 说明 |
|--------|------|------|
| GET/POST | /api/v1/identity/data-classifications | 列表 / 创建分类标签 |
| PUT/DELETE | /api/v1/identity/data-classifications/{id} | 更新 / 删除 |
| GET | /api/v1/identity/data-classifications/lookup?type=&id= | 查询资源分类（PDP 内部调用 + 缓存） |
| GET | /api/v1/audit/compliance/data-inventory | 数据资产清单（替代 pipl_data_inventory 硬编码） |
| GET | /api/v1/audit/compliance/data-security | 数据安全态势（替代 F-17 硬编码）|

**迁移**：pipl_data_inventory_handler.go + attribute_governance_handler.go 硬编码数据 → 查 data_classifications 表。

## 8. 迁移现有硬编码 handler

| 现有 handler | 硬编码内容 | 迁移目标 |
|-------------|-----------|---------|
| pipl_data_inventory_handler | 7 行 category + sensitivity + retention | seed data_classifications 表（按租户） |
| attribute_governance_handler | 6 个用户属性 PII 分类 | resource_type='user_attribute', resource_id=属性名 |
| compliance_config_handler | DataClassificationRules map | 合并到 data_classifications 表 |

## 9. 出境管控

cross_border 字段 + gateway 检查：
- 请求源 IP 地理定位（已有 geo-fencing 基础设施）
- core 级数据 + 非中国大陆 IP → 拒绝 + 审计告警
- important + 非中国大陆 IP → 记录出境日志（供后续申报）

## 10. 与已有组件协同

| 组件 | 协同点 |
|------|--------|
| 国密 SM4 | core 级数据可选 SM4 加密（pkg/crypto/sm4.go 已实现） |
| PDP | $data.classification 内置属性 |
| ITDR | core 级数据异常访问 → critical detection |
| CAE | core 级数据访问 + session risk 高 → step-up |
| 审计 | 差异化保留 + 出境日志 |
| SCIM | 同步分类标签到下游 SP |

## 11. 工作量

| Phase | 内容 | 预估 |
|-------|------|------|
| 1 | migration 016 + DataClassificationRepo + seed | 0.5d |
| 2 | API 端点 + 迁移硬编码 handler → DB | 0.5d |
| 3 | PDP $data.classification 注入 + 默认策略 | 0.5d |
| 4 | 差异化保留 + PII mask | 0.5d |
| 5 | 出境管控 + 测试 | 0.5d |
| **总计** | | **~2.5d** |
