# Verifiable Credentials 设计 (W3C VC + DID:web)

> 状态: Proposed | 作者: IAMExpert | 日期: 2026-07-17 | 来源: kanban I-11, 第 13 小时研究
> 关联: 国密 SM2/SM3 (pkg/crypto/)、OIDC/OAuth (oauth 服务)、DID handler (identity 服务已有)

## 1. 愿景

W3C Verifiable Credentials (VC) 是去中心化身份的核心标准。持有者（钱包）向验证者证明属性（学历/职业/年龄），无需中心化 IdP 参与。EUDI Wallet（欧盟 2026 强制）、中国 CA 电子证照、金融 KYC 全部收敛到 VC 标准。

**GGID 独特优势**：
- 已有 DID:web resolver（identity 服务 did_resolver.go，129 行）
- 已有 DID handler（注册/解析/列表/停用）
- 已有国密 SM2/SM3 签名能力（pkg/crypto/）
- 已有 OIDC discovery + JWKS + SAML（联邦身份基础设施）
- 只需补充 VC 签发/验证/吊销/呈现层

## 2. 总体架构

```
┌─────────────┐    issue     ┌──────────────┐    present    ┌──────────────┐
│  Issuer      │ ──────────→ │  Holder       │ ───────────→ │  Verifier     │
│  (GGID)      │   VC (JSON)  │  (Wallet/App) │   VP (JSON)  │  (GGID/API)  │
│              │              │               │              │               │
│ DID:web      │              │ DID:key       │              │ verify VP     │
│ SM2/Ed25519  │              │ stores VC     │              │ check status  │
│ sign VC      │              │               │              │ extract claims│
└─────────────┘              └──────────────┘              └──────────────┘
      │                                                           │
      └───────── statusList (吊销检查) ──────────────────────────┘
```

**角色**：GGID 同时作为 Issuer（签发 VC）和 Verifier（验证 VP），也可作为 Holder（持有跨组织 VC）。

## 3. 数据模型

### 3.1 migration 021_verifiable_credentials.sql

```sql
-- 可验证凭证
CREATE TABLE verifiable_credentials (
    id              UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    issuer_did      TEXT NOT NULL,           -- "did:web:iam.ggid.example.com"
    subject_did     TEXT NOT NULL,           -- 持有者 DID
    subject_user_id UUID,                   -- 关联 GGID 用户（可选）
    credential_type TEXT NOT NULL,           -- "UniversityDegree", "EmployeeCredential", "IdentityCard"
    payload         JSONB NOT NULL,          -- VC 完整 JSON（含 @context + type + credentialSubject + proof）
    issuer_key_id   TEXT NOT NULL,           -- 签名密钥 ID（kid）
    algorithm       TEXT NOT NULL,           -- "Ed25519" / "SM2SM3"
    status_list_idx INT,                    -- statusList 中的位置索引
    status_credential_id UUID,              -- 关联的 statusList credential ID
    expires_at      TIMESTAMPTZ,             -- VC 过期时间
    revoked         BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_vc_subject ON verifiable_credentials(tenant_id, subject_did);
CREATE INDEX idx_vc_type ON verifiable_credentials(tenant_id, credential_type);
ALTER TABLE verifiable_credentials ENABLE ROW LEVEL SECURITY;

-- StatusList2021 吊销列表
CREATE TABLE vc_status_lists (
    id              UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    issuer_did      TEXT NOT NULL,
    list_index      INT NOT NULL,            -- 第几个 statusList（每 131072 个 VC 一个）
    bitstring       BYTEA NOT NULL,          -- 压缩位图（每个 bit 对应一个 VC 的吊销状态）
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, issuer_did, list_index)
);

-- VC 类型模板（管理员配置签发模板）
CREATE TABLE vc_templates (
    id              UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    type            TEXT NOT NULL UNIQUE,    -- "UniversityDegree"
    name            TEXT NOT NULL,
    json_template   JSONB NOT NULL,          -- credentialSubject 模板（含 {{user.name}} 占位符）
    required_claims TEXT[] NOT NULL,         -- 签发前必须验证的用户属性
    auto_issue      BOOLEAN NOT NULL DEFAULT FALSE, -- 注册/事件触发自动签发
    ttl_days        INT,                     -- 默认有效期
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, type)
);
```

## 4. VC 签发流程

### 4.1 VC 结构（W3C VC Data Model 2.0 + JSON-LD）

```json
{
  "@context": [
    "https://www.w3.org/ns/credentials/v2",
    "https://w3id.org/security/suites/ed25519-2020/v1"
  ],
  "id": "urn:uuid:01234567-89ab-cdef...",
  "type": ["VerifiableCredential", "EmployeeCredential"],
  "issuer": "did:web:iam.ggid.example.com",
  "issuanceDate": "2026-07-17T12:00:00Z",
  "expirationDate": "2027-07-17T12:00:00Z",
  "credentialSubject": {
    "id": "did:web:wallet.user.example.com",
    "name": "张三",
    "email": "zhangsan@example.com",
    "department": "Engineering",
    "role": "Senior Developer",
    "employeeId": "GGID-2026-001234"
  },
  "credentialStatus": {
    "id": "https://iam.ggid.example.com/vc/status/0",
    "type": "StatusList2021Entry",
    "statusPurpose": "revocation",
    "statusListIndex": "42"
  },
  "proof": {
    "type": "Ed25519Signature2020",
    "created": "2026-07-17T12:00:00Z",
    "verificationMethod": "did:web:iam.ggid.example.com#key-1",
    "proofPurpose": "assertionMethod",
    "proofValue": "z3v..."
  }
}
```

### 4.2 签发服务

位置：`services/identity/internal/service/vc_issuer.go`

```go
type VCIssuer struct {
    didResolver *DIDResolver
    keyProvider crypto.KeyProvider   // 国密 SM2 或 Ed25519
    statusList  *StatusListManager
    templates   *VCTemplateRepo
    repo        *VCRepository
    auditPub    *audit.Publisher
}

// IssueCredential 签发一个 VC。
func (i *VCIssuer) IssueCredential(ctx context.Context, req IssueRequest) (*VerifiableCredential, error) {
    // 1. 获取模板
    tmpl, err := i.templates.Get(req.Type)
    
    // 2. 填充 credentialSubject（模板占位符 → 用户属性）
    subject := fillTemplate(tmpl, req.UserClaims)
    
    // 3. 分配 statusList index
    statusIdx, statusCredID := i.statusList.Allocate(ctx)
    
    // 4. 构建 VC 文档（不含 proof）
    vc := buildVCDoc(req.IssuerDID, subject, statusIdx, statusCredID, req.ExpiresAt)
    
    // 5. 签名（canonicalize → sign）
    proof, err := i.signProof(vc, req.Algorithm) // Ed25519 或 SM2SM3
    
    // 6. 存储
    i.repo.Store(ctx, vc, proof, statusIdx)
    
    // 7. Audit
    i.auditPub.Publish(ctx, audit.Event{Action: "vc.issued", ...})
    
    return vc.WithProof(proof), nil
}
```

### 4.3 国密 SM2 签名支持

proof.type = "SM2Signature2024"（自定义 suite），proofValue = ASN.1 DER SM2 签名（复用 pkg/crypto SigningMethodSM2）。

```go
func (i *VCIssuer) signProof(vc map[string]any, alg string) (*Proof, error) {
    canonicalized := canonicalizeJSONLD(vc)  // URDNA2015 规范化
    hash := sm3.Sum256(canonicalized)        // SM3 哈希
    
    switch alg {
    case "SM2SM3":
        sig, err := i.keyProvider.Signer().Sign(rand.Reader, hash, sm2.NewSM2SignerOption(true, nil))
        return &Proof{Type: "SM2Signature2024", ProofValue: base64url.Encode(sig)}, nil
    case "Ed25519":
        sig := ed25519.Sign(i.keyProvider.Signer().(ed25519.PrivateKey), canonicalized)
        return &Proof{Type: "Ed25519Signature2020", ProofValue: base64url.Encode(sig)}, nil
    }
}
```

## 5. VC 验证流程

位置：`services/identity/internal/service/vc_verifier.go`

```go
func (v *VCVerifier) VerifyCredential(ctx context.Context, vc *VerifiableCredential) (*VerificationResult, error) {
    // 1. 解析 issuer DID → 获取 verificationMethod → 公钥
    didDoc, err := v.didResolver.ResolveDID(vc.Issuer)
    pubKey := didDoc.GetVerificationMethod(vc.Proof.VerificationMethod)
    
    // 2. 规范化 VC（去掉 proof）+ SM3/SHA-256 哈希
    canonicalized := canonicalizeJSONLD(vc.WithoutProof())
    
    // 3. 验签
    switch vc.Proof.Type {
    case "SM2Signature2024":
        ok := sm2.VerifyASN1WithSM2(pubKey, nil, canonicalized, sig)
    case "Ed25519Signature2020":
        ok := ed25519.Verify(pubKey, canonicalized, sig)
    }
    
    // 4. 检查过期
    if vc.ExpirationDate.Before(time.Now()) { return "expired" }
    
    // 5. 检查吊销（StatusList2021）
    revoked := v.statusList.CheckRevoked(ctx, vc.CredentialStatus.StatusListIndex, vc.CredentialStatus.ID)
    
    // 6. 返回结果 + 提取 claims
    return &VerificationResult{
        Valid: ok && !revoked && !expired,
        Claims: vc.CredentialSubject,
        Checks: []string{"signature", "expiry", "status"},
    }, nil
}
```

## 6. StatusList2021 吊销

W3C StatusList2021 标准：一个 VC 的吊销状态用一个 bit 表示（0=有效, 1=吊销）。131072 个 VC 共享一个 statusList credential（位图压缩为 base64url GZIP）。

```go
// Revoke 标记一个 VC 为已吊销。
func (s *StatusListManager) Revoke(ctx context.Context, vcID uuid.UUID) error {
    vc, _ := s.repo.Get(vcID)
    list, _ := s.statusListRepo.Get(vc.StatusListCredentialID)
    
    // 设置对应 bit = 1
    s.setBit(list.Bitstring, vc.StatusListIndex, 1)
    s.statusListRepo.Update(ctx, list)
    
    // 更新 VC revoked 标记
    s.repo.MarkRevoked(ctx, vcID)
    
    // Audit
    s.auditPub.Publish(ctx, audit.Event{Action: "vc.revoked", ...})
}

// CheckRevoked 检查 statusList credential。
// 验证者 GET https://iam.ggid.example.com/vc/status/{index} → 返回 statusList VC（公开端点）
// 本地解压位图 → 查对应 bit
func (s *StatusListManager) CheckRevoked(ctx context.Context, idx int, statusCredURL string) bool {
    resp, _ := http.Get(statusCredURL)
    statusVC := decode(resp)
    bitstring := gunzip(base64urlDecode(statusVC.EncodedList))
    return getBit(bitstring, idx) == 1
}
```

**公开端点**：`GET /vc/status/{index}` 返回 statusList VC（无需认证，标准化要求）。

## 7. VP (Verifiable Presentation) 验证

持有者向验证者提交 VP（包含一个或多个 VC + 持有者签名证明）：

```json
{
  "@context": ["https://www.w3.org/ns/credentials/v2"],
  "type": "VerifiablePresentation",
  "holder": "did:web:wallet.user.example.com",
  "verifiableCredential": [ /* VC array */ ],
  "proof": { /* holder 签名证明持有 */ }
}
```

GGID 验证端点：`POST /api/v1/identity/vc/verify-presentation`

## 8. API 设计

| Method | Path | 说明 | 认证 |
|--------|------|------|------|
| POST | /api/v1/identity/vc/issue | 签发 VC | JWT admin |
| POST | /api/v1/identity/vc/verify | 验证单个 VC | JWT（任何已认证用户）|
| POST | /api/v1/identity/vc/verify-presentation | 验证 VP | 公开（或 API key）|
| POST | /api/v1/identity/vc/{id}/revoke | 吊销 VC | JWT admin |
| GET | /api/v1/identity/vc | 列表（issuer 视角）| JWT admin |
| GET | /vc/status/{index} | StatusList VC（公开）| 无认证 |
| GET/POST | /api/v1/identity/vc/templates | VC 模板管理 | JWT admin |
| GET | /api/v1/identity/did/{did} | DID 解析（已有）| 无认证 |

## 9. Console 页面

| 页面 | 路径 | 功能 |
|------|------|------|
| VC 模板管理 | /security/vc/templates | 类型列表 + 创建/编辑模板 + 自动签发配置 |
| VC 签发 | /security/vc/issue | 选择用户 + 模板 → 预览 → 签发 + 二维码（钱包扫描导入）|
| VC 验证 | /security/vc/verify | 粘贴 VC/VP JSON → 验证结果（签名/过期/吊销）|
| VC 列表 | /security/vc/credentials | 已签发 VC 列表 + 吊销按钮 + 状态灯 |
| DID 管理 | /security/did（已有）| DID 文档查看 + 密钥轮换 |

## 10. 安全要求

| 要求 | 实现 |
|------|------|
| 签发私钥保护 | KeyProvider（local/HSM/PKCS11），支持 SM2（国密合规）|
| statusList 公开但不可篡改 | statusList VC 本身签名（issuer 签），验证者验签 |
| VC 不可伪造 | issuer 签名验证 + DID 解析获取可信公钥 |
| 隐私（选择性披露） | v2: SD-JWT based VC（每个 claim 独立签名，持有者选择展示）|
| DID:web TLS | DID 文档通过 HTTPS 获取（did:web 规范要求）|
| 审计 | 签发/吊销/验证全写 audit 事件 |

## 11. 与已有组件协同

| 组件 | 协同点 |
|------|--------|
| DID resolver (已有) | issuer/holder DID 解析 → 获取公钥 |
| 国密 SM2/SM3 (已实现) | proof.type="SM2Signature2024"，国密合规 VC |
| OIDC/OAuth | OID4VCI 协议（OAuth auth code flow + authorization_details RAR → 签发 VC）|
| SAML | VC 可替代 SAML assertion（跨域联邦场景）|
| 数据安全法 | VC 中的 credentialSubject 字段联动 data_classification |
| ITDR | 异常 VC 签发模式检测（批量签发/非工作时间签发）|

## 12. 测试计划

| 测试 | 验证点 |
|------|--------|
| Issue Ed25519 VC | proof 验证通过 |
| Issue SM2SM3 VC | proof 验证通过（国密合规）|
| Verify valid VC | result.valid=true |
| Verify expired VC | result.valid=false, check="expiry" |
| Revoke + verify | result.valid=false, check="status" |
| Tampered VC | signature 验证失败 |
| StatusList 压缩/解压 | 131072 bits → base64url gzip |
| DID:web resolve | HTTPS 获取 DID 文档 |
| VP verify | holder 签名 + 内含 VC 全部有效 |
| 模板占位符填充 | {{user.name}} → 实际值 |

## 13. 工作量

| Phase | 内容 | 预估 |
|-------|------|------|
| 1 | migration 021 + VCTemplate + VCRepo + StatusListManager | 1d |
| 2 | VCIssuer（签名 Ed25519+SM2 + 模板填充 + statusList 分配）| 1d |
| 3 | VCVerifier（DID 解析 + 验签 + 过期/吊销检查）| 0.5d |
| 4 | API 端点 + 公开 statusList 端点 | 0.5d |
| 5 | Console 页面（模板/签发/验证/列表）| 1d |
| 6 | OID4VCI 协议集成（OAuth → VC）| 1d（可选 v2）|
| 7 | 测试 | 0.5d |
| **总计** | | **~5d**（含 OID4VCI ~6d）|
