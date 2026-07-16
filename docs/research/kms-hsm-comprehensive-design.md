# KMS/HSM 全面集成设计 — 云 KMS + 私有部署 + 国密标准

## Date: 2026-07-16

## P0: 硬编码密钥修复
- credential_vault_handler.go: vaultAESKey 从环境变量读取
- biometric_handler.go: bioKey 从环境变量读取
- 所有密钥支持：env var → DB 持久化 → KMS 远程获取（优先级递减）

## P1: 云 KMS 实现
| Provider | 状态 | SDK |
|----------|------|-----|
| AWS KMS | Stub → 实现 | aws-sdk-go-v2/kms |
| Google Cloud KMS | Config only → 实现 | google.golang.org/api/cloudkms |
| Azure Key Vault | Config only → 实现 | github.com/Azure/azure-sdk-for-go |
| HashiCorp Vault | 新增 | github.com/hashicorp/vault/api |
| Alibaba Cloud KMS | 新增 | github.com/aliyun/alibaba-cloud-sdk-go |

## P2: 私有部署 HSM/KMS
| Provider | 说明 |
|----------|------|
| PKCS#11 | 通用 HSM 接口（Thales, Utimaco, YubiHSM） |
| SoftHSM2 | 开源软件 HSM（开发/测试用） |
| Fortanix SDKMS | SaaS HSM |
| Entrust nShield | 企业 HSM |

## P3: 中国国家商用密码标准（国密）
| 标准 | 算法 | 实现 |
|------|------|------|
| SM2 | 椭圆曲线签名/密钥交换 | crypto/ecdsa 兼容 + go-sm2 |
| SM3 | 哈希（256bit） | go-sm3 / crypto/sm3 (Go 1.21+) |
| SM4 | 对称加密（128bit） | go-sm4 |
| SM9 | 标识签名 | go-sm9 |
| GM/T 0018 | 密码设备接口 | 类似 PKCS#11 国密版 |
| GM/T 0016 | 智能密码密钥 | USB Key/硬件令牌 |

### 国密 KeyProvider 设计
- SM2KeyProvider: 签名密钥（替换 RSA/ECDSA）
- SM4KeyProvider: 对称加密（替换 AES）
- 支持 SM2+SM3 签名（类比 RS256→SM2SM3）
- JWT 算法头: `alg: SM2SM3` 或 `alg: SM3SM2`

## P4: 配置持久化
所有 KMS 配置（环境变量 → DB 持久化）：
- kms_configs 表: (provider, config_json, status, created_at, updated_at)
- 启动时优先从 DB 加载，env var 作为 fallback
- 支持运行时切换 KMS provider（需要 key rotation）

## 实现优先级
1. P0: 硬编码密钥修复（立即）
2. P1: HashiCorp Vault（最通用私有部署） + AWS KMS
3. P2: PKCS#11（通用 HSM）
4. P3: 国密 SM2/SM3/SM4 支持
5. P4: 配置持久化 + 运行时切换
