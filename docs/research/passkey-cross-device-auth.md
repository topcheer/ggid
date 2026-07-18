# Passkey Cross-Device Authentication & Hybrid Transport

> 状态: Research | 作者: arch_pm | 日期: 2026-07-18 | 来源: WebAuthn enterprise research + FIDO Alliance spec

## 1. 现状

GGID 已有完整的 WebAuthn/Passkey 支持：
- 注册流程（register/begin + finish）
- 认证流程（login/begin + finish）
- AAGUID 允许列表（KB-078）
- TAP 临时访问（KB-076）
- Conditional UI（passkeyAutofill 端点）

**缺口**：无跨设备认证（hybrid transport）。用户在桌面浏览器无法使用手机上的 passkey。

## 2. FIDO Cross-Device Authentication (CDA)

### 2.1 协议概述
- 用户在桌面浏览器发起 WebAuthn 登录
- 浏览器显示 QR 码
- 用户用手机扫描 QR 码
- 手机作为 authenticator 完成认证
- 通过 BLE（蓝牙低功耗）建立安全通道

### 2.2 GGID 需要做的
- **后端**：`generateAssertionOptions()` 时设置 `mediation: "conditional"` 和 `userVerification: "required"` — WebAuthn server 端无需改动，CDA 是客户端/浏览器功能
- **前端**：确保 `navigator.credentials.get()` 调用支持 hybrid transport（浏览器原生支持）
- **限制**：CDA 需要浏览器支持（Chrome 118+, Safari 17+, Edge 118+）

### 2.3 实现检查
```
grep -rn "mediation\|conditional\|hybrid\|crossDevice" services/auth/
```
WebAuthn server 端不需要特殊改动。只需确保 assertion options 正确配置。

## 3. Device Public Key (DPK) — 番茄工作法增强

FIDO 2.1 新增 Device Public Key：
- 每个设备生成独立的密钥对（不同于 passkey credential）
- DPK 用于证明设备身份（device attestation）
- 即使 passkey 同步到新设备，DPK 不同 → 可检测设备变化

### 3.1 GGID 集成
1. WebAuthn registration 时提取 DPK
2. 存储 DPK 在 `webauthn_device_public_keys` 表
3. 登录时验证 DPK — 如果与注册时不同 → 标记为新设备 + 触发 CAE 评估
4. 端点：POST /api/v1/auth/webauthn/dpk/verify

## 4. Backlog

| ID | Title | Owner | Priority | Est |
|----|-------|-------|----------|-----|
| KB-260 | Conditional UI autofill 配置增强（mediation: conditional） | security | P2 | 1d |
| KB-261 | Device Public Key 提取与存储 | security | P1 | 3d |
| KB-262 | DPK 变更检测 → CAE 设备变化信号 | security | P1 | 2d |
| KB-263 | Cross-device QR 认证文档与浏览器兼容性测试 | qa | P2 | 1d |

## 5. 反模式禁令
- 不实现自定义 QR 协议（使用浏览器原生 CDA）
- DPK 不可替代 passkey credential（两者独立）
- DPK 验证失败不阻止登录（降级为 step_up MFA）
