# IAM 系统深度审查报告 — 2026-07-27 (R39)

## 审查范围

全新视角。聚焦：WASM 插件安全、password pepper 配置、KMS 集成、audit hash chain、P1-11 重实现验证、新 commit 审查。

---

## 1. RFC 标准符合性

之前 R36-R38 已覆盖。P1-11 已重实现为验证 JWT 的合规实现。无新差距。

## 2. 竞品迁移成本

之前 R36 已覆盖。无新变化。

## 3. 安全审查

### 3.1 WASM 插件签名验证 — fail-open

**文件**: `services/gateway/internal/middleware/wasm_plugin.go:410-413`

```go
if secret == "" {
    return nil  // skip verification, log warning
}
```

| ID | 严重性 | 描述 |
|----|--------|------|
| P2-58 | P2 | WASM 插件签名验证 fail-open — GGID_INTERNAL_SECRET 未设置时跳过验证仅 log warning，允许未签名 WASM 执行 |

### 3.2 Password pepper — fail-open

**文件**: `services/auth/cmd/main.go:108-113`

```go
pepper := os.Getenv("PASSWORD_PEPPER")
if pepper == "" {
    log.Printf("⚠️ SECURITY WARNING: PASSWORD_PEPPER is not set!")
} else {
    crypto.SetPepper(pepper)
}
```

| ID | 严重性 | 描述 |
|----|--------|------|
| P2-59 | P2 | Password pepper fail-open — 未设置时仅 warning 继续运行，不强制要求（与 P1-9 TOTP fail-closed 模式不一致）|

### 3.3 P1-11 重实现 — ✅ 已修复

**文件**: `token_downscope_handler.go` (commit 1e023913c)

- 现在验证源 JWT ✅
- 使用 OAuthService 颁发实际降权限 token ✅
- scope 子集验证 ✅

### 3.4 Audit hash chain — ✅ 安全

- ComputeHash: HMAC-SHA256 链式哈希 ✅
- VerifyHash: 篡改检测 ✅
- 密钥版本支持 ✅

### 3.5 KMS 集成

- AWS KMS + Vault: 纯 Go 实现，始终可用 ✅
- GCP KMS: 存根（不支持）— 无影响

## 4. 代码质量 — 最近 10 commits

| Commit | 类型 | 质量 |
|--------|------|------|
| c4a28b14a docs | — | — |
| 1cbe27c5a fix | client description 持久化 | ✅ |
| da3a1a106 fix | UX forgot-password focus ring | ✅ |
| 7672a34fd fix | 移除重复 downscope 路由 | ✅ 重要修复（防止 panic）|
| 1e023913c fix | P1-11 downscope 重实现 | ✅ 正确 |
| 556d8c2d0 fix | P1-11 移除存根 | ✅ |
| 其余 | docs | — |

**无新 bug。** P1-11 修复质量好。

---

## 5. 问题汇总

### 本次新发现

| ID | 严重性 | 描述 |
|----|--------|------|
| P2-58 | P2 | WASM 插件签名验证 fail-open |
| P2-59 | P2 | Password pepper fail-open |

### 已修复确认

P1-11 ✅ (1e023913c — 重实现为验证 JWT 的合规实现)

### 活跃问题

| ID | 严重性 | 状态 |
|----|--------|------|
| P1-3 | P1 | 活跃 (MCP JWKS) |
| P1-6 | P1 | 活跃 (JWKS rotation) |
| P2-31~P2-59 | P2 | 活跃 (52 个) |

**总计：54 个活跃问题（P0: 0 / P1: 2 / P2: 52）**