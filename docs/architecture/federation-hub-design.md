# Federation Hub 设计 (Cross-Domain Identity Federation)

> 状态: Proposed | 作者: IAMExpert | 日期: 2026-07-17 | 来源: kanban I-12, 第 17 小时研究
> 关联: SAML IdP/SP (oauth)、OIDC (oauth)、DID:web (identity)、SCIM 2.0 (identity)、VC (vc-design.md)

## 1. 愿景

GGID 作为身份联邦中枢（Hub），连接 N 个 IdP（Okta/Entra/ADFS/飞书/钉钉）和 N 个 SP（内部应用/云 SaaS）。用户从任意 IdP 登录 → Hub 转换断言 → 访问任意 SP。产品定位对标 Auth0/Ping Identity。

**现有联邦资产盘点（全部已实现）**：
- SAML SP（ACS + ParseAssertion + ValidateConditions）
- SAML IdP（XMLDSig 签名断言 + BuildSAMLResponse）
- OIDC Discovery + JWKS + Dynamic Registration
- OIDC Federation config handler（存在）
- DID:web resolver + DID handler（注册/解析/列表/停用）
- SCIM 2.0 inbound（IdP → GGID 用户/组同步）
- Social login providers（Google/GitHub/WeChat）
- LDAP/AD integration（authprovider chain）

**缺 3 块拼图**：Trust Chain Registry + Assertion Transformation Engine + Federation Metadata Aggregate。

## 2. 总体架构

```
IdP A (SAML)     IdP B (OIDC)     IdP C (LDAP)     IdP D (DID:web)
    │                │                 │                 │
    ▼                ▼                 ▼                 ▼
┌──────────────────────────────────────────────────────────────┐
│                    GGID Federation Hub                         │
│                                                                │
│  ┌─────────────────┐  ┌──────────────────┐  ┌──────────────┐ │
│  │ Trust Chain      │  │ Assertion        │  │ Discovery    │ │
│  │ Registry         │  │ Transformation   │  │ Service      │ │
│  │ (trusted_idps/  │  │ Engine            │  │ (WAYF +      │ │
│  │  trusted_sps)   │  │ (SAML→OIDC→VC)   │  │  email route)│ │
│  └─────────────────┘  └──────────────────┘  └──────────────┘ │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │ Federation Metadata Aggregate                             │ │
│  │ /api/v1/.well-known/federation-configuration              │ │
│  └──────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
    │                │                 │                 │
    ▼                ▼                 ▼                 ▼
SP 1 (SAML)      SP 2 (OIDC)       SP 3 (GGID app)    SP 4 (VC verify)
```

## 3. Trust Chain Registry

### 3.1 migration 022_federation_entities.sql

```sql
CREATE TABLE federation_entities (
    id              UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    entity_id       TEXT NOT NULL,           -- SAML entityID / OIDC issuer / DID
    entity_name     TEXT NOT NULL,           -- "Okta Production"
    entity_type     TEXT NOT NULL,           -- 'idp' | 'sp' | 'both'
    protocol        TEXT NOT NULL,           -- 'saml' | 'oidc' | 'ldap' | 'did' | 'scim' | 'social'
    
    -- 连接配置
    metadata_url    TEXT,                    -- SAML metadata URL / OIDC discovery URL
    acs_url         TEXT,                    -- SP Assertion Consumer Service URL
    slo_url         TEXT,                    -- Single Logout URL
    issuer          TEXT,                    -- OIDC issuer / SAML entityID
    
    -- 信任
    trust_level     TEXT NOT NULL DEFAULT 'verified', -- 'verified' | 'pending' | 'revoked'
    trust_direction TEXT NOT NULL,           -- 'inbound' | 'outbound' | 'bidirectional'
    
    -- 证书（SAML 签名验证 / OIDC JWKS URL）
    certificates    JSONB NOT NULL DEFAULT '[]', -- [{kid, pem, fingerprint, expires_at}]
    jwks_url        TEXT,                    -- OIDC JWKS endpoint
    
    -- 自动发现
    auto_discovery  BOOLEAN NOT NULL DEFAULT FALSE, -- 自动从 metadata_url 刷新
    
    -- 属性映射引用
    transform_rule_id UUID,                  -- 关联 assertion_transform_rules
    
    -- 过期/状态
    expires_at      TIMESTAMPTZ,             -- 信任关系过期
    last_checked    TIMESTAMPTZ,             -- 最后 metadata 刷新时间
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_by      UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    UNIQUE(tenant_id, entity_id, protocol)
);

CREATE INDEX idx_fed_entities_type ON federation_entities(tenant_id, entity_type, enabled);
CREATE INDEX idx_fed_entities_protocol ON federation_entities(tenant_id, protocol, enabled);
ALTER TABLE federation_entities ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_iso ON federation_entities
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

### 3.2 Trust Chain 验证

```go
type TrustChainValidator struct {
    repo FederationEntityRepo
}

// ValidateTrust 验证来自外部实体的断言/token 的信任链。
func (v *TrustChainValidator) ValidateTrust(ctx context.Context, entityID, protocol string, cert *x509.Certificate) error {
    entity, err := v.repo.GetByEntityID(ctx, entityID, protocol)
    if err != nil || entity == nil {
        return ErrUntrustedEntity
    }
    if !entity.Enabled || entity.TrustLevel == "revoked" {
        return ErrEntityRevoked
    }
    if entity.ExpiresAt != nil && entity.ExpiresAt.Before(time.Now()) {
        return ErrTrustExpired
    }
    // 验证证书在 entity.certificates 列表中（fingerprint 匹配）
    if !entity.HasCertificate(cert) {
        return ErrCertificateNotTrusted
    }
    return nil
}
```

### 3.3 Certificate Expiry Monitoring

每日 cron 检查所有 federation_entities.certificates → 30 天内过期 → 告警 webhook + Console 通知。

## 4. Assertion Transformation Engine

### 4.1 数据模型

```sql
CREATE TABLE assertion_transform_rules (
    id              UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    name            TEXT NOT NULL,           -- "Okta SAML → GGID OIDC"
    source_protocol TEXT NOT NULL,           -- 'saml' | 'oidc' | 'vc' | 'ldap'
    target_protocol TEXT NOT NULL,           -- 'saml' | 'oidc' | 'vc' | 'header'
    rules           JSONB NOT NULL,          -- 映射规则数组
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, name)
);
```

### 4.2 映射 DSL

```json
{
  "rules": [
    {
      "source": "saml:attribute:eduPersonAffiliation",
      "target": "oidc:claim:role",
      "transform": "array_first",
      "fallback": "user"
    },
    {
      "source": "saml:attribute:mail",
      "target": "oidc:claim:email",
      "transform": "lowercase"
    },
    {
      "source": "saml:nameid",
      "target": "oidc:claim:sub",
      "transform": "uuid_from_string"
    },
    {
      "source": "saml:attribute:department",
      "target": "header:X-GGID-Department",
      "transform": "identity"
    },
    {
      "source": "oidc:claim:sub",
      "target": "vc:credentialSubject.id",
      "transform": "did:web:prefix:{domain}"
    }
  ],
  "claim_filters": {
    "oidc:claim:email": {"regex": ".*@trusted-domain\\.com$"}
  }
}
```

**Transform 函数**：`identity` / `lowercase` / `uppercase` / `array_first` / `array_join:","` / `uuid_from_string` / `prefix:` / `suffix:` / `regex:pattern` / `did:web:prefix:{domain}`

### 4.3 引擎

```go
type TransformationEngine struct {
    repo TransformRuleRepo
}

// Transform 将源协议的断言属性转换为目标协议的 claims。
func (e *TransformationEngine) Transform(ctx context.Context, sourceProto, targetProto string, sourceAttrs map[string][]string) (map[string]any, error) {
    rules, err := e.repo.FindRules(ctx, sourceProto, targetProto)
    
    result := make(map[string]any)
    for _, rule := range rules {
        values, ok := sourceAttrs[rule.Source]
        if !ok || len(values) == 0 {
            if rule.Fallback != "" {
                result[rule.Target] = rule.Fallback
            }
            continue
        }
        transformed := applyTransform(rule.Transform, values)
        if rule.Filter != nil && !rule.Filter.Match(transformed) {
            return nil, ErrClaimValidationFailed
        }
        result[rule.Target] = transformed
    }
    return result, nil
}
```

## 5. Federation Metadata Aggregate

### 5.1 统一发现端点

`GET /api/v1/.well-known/federation-configuration`

```json
{
  "issuer": "https://iam.ggid.example.com",
  "federation_version": "1.0",
  "trusted_idps": [
    {
      "entity_id": "https://okta.example.com",
      "entity_name": "Okta Production",
      "protocol": "saml",
      "metadata_url": "https://okta.example.com/metadata",
      "trust_level": "verified",
      "logo_url": "https://...",
      "login_url": "https://iam.ggid.example.com/sso/okta"
    },
    {
      "entity_id": "https://login.microsoftonline.com/...",
      "entity_name": "Azure AD",
      "protocol": "oidc",
      "discovery_url": "https://login.microsoftonline.com/.../.well-known/openid-configuration"
    }
  ],
  "trusted_sps": [
    {
      "entity_id": "https://grafana.internal",
      "entity_name": "Grafana",
      "protocol": "header",
      "acs_url": "https://grafana.internal/login/generic_oauth"
    }
  ],
  "capabilities": {
    "saml_idp": true,
    "saml_sp": true,
    "oidc_provider": true,
    "oidc_client": true,
    "scim_inbound": true,
    "vc_issuer": true,
    "vc_verifier": true,
    "did_resolver": true
  }
}
```

### 5.2 自动刷新

auto_discovery=true 的 entity → 每日 cron 拉 metadata_url → 更新 certificates + entity config → 检测变化告警。

## 6. Discovery Service (WAYF)

### 6.1 Email Domain 路由

```go
// DiscoverIdP 根据 email domain 自动路由到对应 IdP。
func (s *DiscoveryService) DiscoverIdP(ctx context.Context, email string) (*FederationEntity, error) {
    domain := strings.Split(email, "@")[1]
    
    // 查 domain → entity 映射
    entity, err := s.repo.GetByDomain(ctx, domain)
    if err == nil {
        return entity, nil  // 自动路由
    }
    
    // 无匹配 → 返回 WAYF picker
    return nil, ErrWayfRequired
}
```

### 6.2 WAYF Picker API

`GET /api/v1/auth/discovery?return_to=...`

返回 HTML 页面列出所有 enabled IdP（logo + 名称 + 协议），用户选择后跳转。

### 6.3 Domain → Entity 映射表

```sql
CREATE TABLE federation_domain_routes (
    tenant_id   UUID NOT NULL,
    domain      TEXT NOT NULL,     -- "engineering.corp.com"
    entity_id   UUID NOT NULL,     -- → federation_entities.id
    PRIMARY KEY(tenant_id, domain)
);
```

## 7. 安全要求

| 要求 | 实现 |
|------|------|
| 信任锚验证 | TrustChainValidator.ValidateTrust — entityID + certificate fingerprint + trust_level + expiry |
| Certificate expiry 告警 | 每日 cron 检查 → 30d 预警 webhook |
| Entity 防伪造 | SAML: signature 验证 + cert fingerprint 匹配；OIDC: JWKS 验签 + issuer 匹配；DID:web: HTTPS TLS 验证 |
| Transform 过滤 | claim_filters regex（阻止非法 domain email 进入 OIDC claims） |
| Metadata 签名 | Federation Metadata Aggregate 自身签名（Hub private key），SP 下载后验签 |
| 审计 | 每次联邦登录/断言转换写 audit 事件 |

## 8. Console 页面

| 页面 | 功能 |
|------|------|
| Federation Dashboard | 信任拓扑图（IdP ← Hub → SP，颜色=trust_level，绿色=verified）|
| Entity 注册 | entity_id/protocol/metadata_url → 自动拉取解析 → 预览 → 注册 |
| Transform 配置器 | 可视化映射规则编辑器 + 测试（输入 SAML attribute → 输出 OIDC claim）|
| Discovery 配置 | domain route 列表 + WAYF picker 预览 |
| Certificate 管理 | 过期时间线 + 手动轮换 |

## 9. 与现有组件集成

| 组件 | 集成点 |
|------|--------|
| SAML SP（/saml/acs） | TrustChainValidator 在 ParseAssertion 前验证 IdP entity trust |
| SAML IdP（/saml/idp/sso） | 构建 SAML Response 时查 SP trust → 限制 audience |
| OIDC /authorize | client_id → federation_entities 查 client trust |
| SCIM /scim/v2 | SCIM source → federation_entities 查 IdP trust + Transform 映射属性 |
| Social login | 每个 social provider 注册为 federation_entity |
| DID:web | DID resolver 验证信任 → federation_entities trust_level |
| Access Broker | ProtectedApp 自动注册为 federation_entity (type=sp) |
| JML | IdP provision-webhook → Transform 引擎映射属性 → 用户创建 |

## 10. 测试计划

| 测试 | 验证点 |
|------|--------|
| TrustChain valid entity | ValidateTrust → nil |
| TrustChain revoked entity | → ErrEntityRevoked |
| TrustChain expired trust | → ErrTrustExpired |
| TrustChain unknown cert | → ErrCertificateNotTrusted |
| Transform SAML→OIDC | attribute 正确映射到 claim |
| Transform missing source | fallback 值使用 |
| Transform filter reject | 非法 domain → error |
| Metadata aggregate | 全部 enabled entity 返回 |
| Discovery email route | domain 匹配 → auto route |
| Discovery WAYF | 无匹配 → picker 页面 |
| Auto-refresh | mock metadata URL → cert 更新 |

## 11. 工作量

| Phase | 内容 | 预估 |
|-------|------|------|
| 1 | migration 022 + FederationEntityRepo + domain routes | 1d |
| 2 | TrustChainValidator + cert expiry cron | 1d |
| 3 | TransformationEngine（DSL 解析 + transform 函数 + 测试） | 2d |
| 4 | Federation Metadata Aggregate + auto-refresh | 1d |
| 5 | Discovery Service（email route + WAYF picker） | 0.5d |
| 6 | SAML/OIDC/SCIM 集成（TrustChain 注入） | 1d |
| 7 | Console（拓扑图 + 注册 + Transform 配置器） | 1.5d |
| **总计** | | **~8d** |
