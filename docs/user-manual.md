# GGID 用户手册

> GGID IAM Suite · v1.0 · 2026-07-22

本手册面向最终用户（终端用户、租户管理员、平台管理员），涵盖 Console 操作全流程。

---

## 目录

1. [登录与认证](#1-登录与认证)
2. [个人中心](#2-个人中心profile)
3. [安全设置](#3-安全设置security)
4. [设备管理](#4-设备管理devices)
5. [用户管理](#5-用户管理users)
6. [角色与权限](#6-角色与权限roles)
7. [组织管理](#7-组织管理orgs)
8. [审计日志](#8-审计日志audit)
9. [OAuth 客户端](#9-oauth-客户端oauth-clients)
10. [系统设置](#10-系统设置settings)
11. [MFA 多因素认证](#11-mfa-多因素认证)
12. [密码策略](#12-密码策略)
13. [品牌化定制](#13-品牌化定制)
14. [常见问题](#14-常见问题)

---

## 1. 登录与认证

### 1.1 通过 Console 登录

1. 打开 Console 地址：`https://ggid-console.iot2.win/login`
2. 输入：
   - **租户 ID**：从管理员处获取（或输入 `default` 自动解析）
   - **用户名**：如 `admin`
   - **密码**：初始密码由管理员分配
3. 点击 **Sign In**

Console 使用 OAuth2 Password Grant 认证：
```
POST /oauth/token
  grant_type=password
  username=admin
  password=********
  client_id=ggid-console
  scope=openid profile email offline_access
```

登录成功后：
- Access token 存入浏览器 localStorage
- Refresh token 存入 localStorage（用于自动续期）
- Token 过期后自动刷新，无需重新登录

### 1.2 Passkey 登录

在登录页面点击 **Continue with Passkey**，浏览器弹出 WebAuthn 选择器，选择已注册的 Passkey 即可完成免密码登录。

### 1.3 SSO 登录

点击 **Sign in with GGID SSO**，重定向到 GGID OAuth2 授权页面完成登录。

### 1.4 会话管理

- Access token 有效期：15 分钟
- Refresh token 自动轮换（RFC 6749 family rotation）
- 退出登录：点击右上角用户头像 → Logout，清除所有 token

---

## 2. 个人中心（Profile）

路径：`Console → Identity → My Profile`

### 2.1 个人信息

- 修改姓名、邮箱、电话
- 上传头像
- 设置时区、语言偏好

### 2.2 Security 标签页

点击 **Security** 标签页进入安全设置：

- **修改密码**：输入当前密码 + 新密码 + 确认密码，实时显示密码强度
- **MFA 管理**：启用/禁用 TOTP、查看已注册的 MFA 设备
- **Linked Accounts**：查看关联的第三方账户（Google、GitHub 等）

### 2.3 Devices 标签页

查看所有已登录设备：
- 设备名称、类型、最后登录时间、IP 地址
- 可远程注销指定设备

---

## 3. 安全设置（Security）

路径：`Console → Security`

| 菜单 | 功能 |
|------|------|
| Sessions | 查看所有活跃会话，可强制下线 |
| CAE Monitor | Continuous Access Evaluation 监控 |
| Risk Score | 用户风险评分仪表盘 |
| Posture | 安全态势总览 |
| Conditional Access | 条件访问策略（IF-THEN 规则） |
| Password Policy | 密码策略配置 |
| MFA | MFA 全局策略配置 |
| Passkeys | Passkey 管理与策略 |

---

## 4. 设备管理（Devices）

路径：`Console → Identity → My Sessions`

查看当前会话和已注册设备，支持远程注销。

---

## 5. 用户管理（Users）

路径：`Console → Identity → Users`

### 5.1 查看用户列表

- 搜索、筛选用户（按状态、角色、组织）
- 分页浏览

### 5.2 创建用户

1. 点击 **Create User**
2. 填写用户名、邮箱、初始密码
3. 分配角色
4. 分配组织
5. 可选：发送欢迎邮件

### 5.3 编辑用户

- 修改个人信息
- 启用/禁用账户
- 重置密码
- 管理角色分配
- 查看用户活动时间线

### 5.4 批量导入

路径：`Console → Settings → Import Wizard`

支持 CSV/JSON 格式，含 dry-run 验证模式。

---

## 6. 角色与权限（Roles）

路径：`Console → Identity → Roles`

### 6.1 内置角色

| 角色 | 说明 |
|------|------|
| `platform:admin` | 平台管理员，可跨租户管理 |
| `tenant:admin` | 租户管理员，管理本租户所有资源 |
| `user` | 普通用户 |

### 6.2 创建自定义角色

1. 点击 **Create Role**
2. 输入角色名称和描述
3. 选择权限（如 `users:read`, `orders:write`, `audit:read`）
4. 保存

### 6.3 权限命名规则

```
<resource>:<action>
```

示例：
- `users:read` — 查看用户
- `users:write` — 创建/修改用户
- `users:delete` — 删除用户
- `orders:approve` — 审批订单
- `admin` — 超级权限（等价于所有权限）

通配符：`users:*` 匹配 users 下所有操作。

---

## 7. 组织管理（Orgs）

路径：`Console → Identity → Organizations`

- 创建组织树（支持多层级）
- 将用户分配到组织
- 组织级权限隔离

---

## 8. 审计日志（Audit）

路径：`Console → Audit`

### 8.1 审计日志

- 查看所有操作记录（登录、用户变更、权限修改等）
- 按时间、用户、操作类型筛选
- 导出审计报告

### 8.2 告警

- 配置审计告警规则
- 实时异常检测通知

### 8.3 访问评审（Access Reviews）

- 创建定期访问评审计划
- 评审用户权限是否合规
- 自动撤销未通过的权限

---

## 9. OAuth 客户端（OAuth Clients）

路径：`Console → Settings → OAuth Clients`

### 9.1 创建 OAuth 客户端

1. 点击 **Create OAuth Client**
2. 填写：
   - 客户端名称
   - 重定向 URI
   - 支持的 Grant Types（authorization_code, client_credentials, refresh_token 等）
   - 允许的 Scopes
3. 生成 Client Secret
4. 保存

### 9.2 常用 Grant Types

| Grant Type | 适用场景 |
|------------|---------|
| `authorization_code` | Web 应用、SPA（配合 PKCE） |
| `password` | 第一方可信应用（如 Console） |
| `client_credentials` | 服务间 M2M 调用 |
| `refresh_token` | Token 续期 |
| `urn:ietf:params:oauth:grant-type:device_code` | IoT/无浏览器设备 |
| `urn:ietf:params:oauth:grant-type:token-exchange` | Token 降权/委派 |

---

## 10. 系统设置（Settings）

路径：`Console → Settings`

### 10.1 功能一览

| 设置页 | 路径 | 功能 |
|--------|------|------|
| Password Policy | /settings/password-policy | 密码复杂度、过期、历史 |
| MFA | /settings/mfa | MFA 方式配置 |
| Conditional Access | /settings/conditional-access | 条件访问策略 |
| Branding | /settings/branding | Logo、颜色、自定义 CSS |
| Feature Flags | /settings/feature-flags | 功能开关 |
| SCIM | /settings/scim | SCIM 2.0 用户同步 |
| LDAP | /settings/ldap-config | LDAP 目录同步 |
| SAML | /settings/saml-config | SAML 2.0 SSO |
| Webhooks | /settings/webhooks | 事件通知回调 |
| OAuth Clients | /settings/oauth-clients | OAuth 客户端管理 |
| Passkeys | /settings/passkey-management | Passkey 策略 |
| Notifications | /settings/notifications | 通知偏好 |

---

## 11. MFA 多因素认证

### 11.1 支持的 MFA 方式

| 方式 | 说明 |
|------|------|
| TOTP | Google Authenticator / Microsoft Authenticator |
| Passkey | WebAuthn / FIDO2（指纹、Face ID、安全密钥） |
| RSA SecurID | RSA 硬件令牌 |
| YubiKey | Yubico OTP |
| Backup Codes | 一次性备用恢复码 |

### 11.2 启用 TOTP

1. 进入 `Profile → Security → MFA Methods`
2. 点击 **Enable TOTP**
3. 用 Authenticator App 扫描 QR 码
4. 输入 6 位验证码确认
5. 保存生成的备用码

### 11.3 启用 Passkey

1. 在登录页面或 Profile 中点击 **Register Passkey**
2. 浏览器弹出 WebAuthn 注册流程
3. 按提示完成指纹/面纹/PIN 验证
4. Passkey 注册成功后可用于免密码登录

---

## 12. 密码策略

路径：`Console → Settings → Password Policy`

### 12.1 可配置项

| 配置 | 说明 | 默认值 |
|------|------|--------|
| 最小长度 | 密码最少字符数 | 12 |
| 大写字母 | 必须包含大写字母 | 是 |
| 小写字母 | 必须包含小写字母 | 是 |
| 数字 | 必须包含数字 | 是 |
| 特殊字符 | 必须包含特殊符号 | 是 |
| 禁止用户名 | 密码不能包含用户名 | 是 |
| 禁止常见密码 | 拒绝 Top 20 常见密码 | 否 |
| 密码历史 | 不可重用最近 N 个密码 | 5 |
| 过期天数 | 密码最大有效期 | 90 |

### 12.2 实时验证

在密码策略页面右侧可输入测试密码，实时查看是否满足所有规则。

---

## 13. 品牌化定制

路径：`Console → Settings → Branding`

### 13.1 可定制内容

- **Logo**：上传或填写 URL（SVG/PNG，512x512 以下）
- **主色调**：颜色选择器
- **辅助色**：颜色选择器
- **字体**：Inter / Roboto / Open Sans / Lato / Poppins / Noto Sans SC
- **圆角**：0-20px 范围调整
- **自定义 CSS**：注入到 Console 页面的样式代码
- **暗色模式**：默认主题模式

### 13.2 邮件模板预览

在 Email 标签页可预览验证邮件、密码重置邮件、欢迎邮件的渲染效果。

---

## 14. 常见问题

### Q: 忘记密码怎么办？

联系租户管理员重置密码，或通过 `/forgot-password` 页面自助重置（需配置邮件服务）。

### Q: Token 过期后需要重新登录吗？

不需要。Console 自动使用 refresh token 续期。只有当 refresh token 也过期或被撤销时才需重新登录。

### Q: 如何获取我的租户 ID？

登录后，租户 ID 存储在 JWT claims 中。可在 Console 开发者工具 → Application → Local Storage → `ggid_tenant_id` 查看。也可通过 `GET /api/v1/tenants/resolve?slug=default` 查询。

### Q: 支持哪些语言？

Console 支持中文（简体/繁体）、英语、日语、韩语、法语、德语、西班牙语、葡萄牙语、俄语、阿拉伯语。在右上角切换。

### Q: 如何创建 API Key？

路径：`Console → API Keys`，点击 Create API Key，选择权限范围，生成后立即保存（仅显示一次）。

---

> 技术支持：查看 [接入手册](./integration-manual.md) 获取 API 集成指导。
> 部署指南：查看 [部署文档](./deployment-guide.md) 了解自托管部署。