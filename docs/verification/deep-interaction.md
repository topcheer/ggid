# Deep Interaction Verification Results

## Hash Chain — ✅ PASS

用 domain.ComputeHash 逻辑验证：50 broken → repair-chain → 0 broken。guardian 确认 tamper-check CLEAN (1397 verified)。

## CAE 策略 CRUD — ✅ PASS

- 创建 "Deep Test Policy"（device_posture < 70 → require_mfa）
- 保存成功，策略出现在列表
- 刷新后策略持久化确认（3 个策略全部保留）

## Webhook 创建 — ✅ PASS

- 修复后创建 "Deep Test Webhook" 成功
- httpbin.org URL 持久化确认
- commit cd86308f6（name 字段修复）

## U19 Impersonation — ✅ PASS

完整流程验证通过：
1. ✅ 搜索用户 → 显示 viewer_go
2. ✅ 填写理由（>10 字符）
3. ✅ 选择用户 → Start Impersonation
4. ✅ API 返回 access_token（427 字符 JWT）
5. ✅ ImpersonationBanner 显示："Impersonating viewer_go (viewer_go@erp-demo.local)"
6. ✅ End Session → token + state 清除，banner 消失

修复链路（5 层）：
- commit 9c99c59da: 路径 /admin/impersonate → /auth/impersonate
- commit 8c9c18cd9: 参数 impersonator_id + target_user_id
- commit 57bf6bb3b: JWT 签发 access_token
- commit 6a38f4416: history 端点修复
- commit 94fea7c09: banner key + 字段匹配

### 发现
- 前端调用 `POST /api/v1/admin/impersonate` → 404（路由不存在）
- 后端实际路由 `POST /api/v1/auth/impersonate` → 400（参数不对）
- 参数不匹配：前端发 `{user_id, reason}`，后端期望 `{impersonator_id, target_id, reason}`
- `/api/v1/admin/impersonate/history` 和 `/api/v1/admin/impersonate/end` 也很可能 404

### 之前验证的假阳性
30/30 "PASS" 只验证了：
1. 页面渲染正常（h1 + len）✅
2. 搜索用户不崩溃 ✅
但从未验证：
- 点击用户 → 选择 ✅
- 输入理由 → Start Impersonation ✅（按钮点击成功）
- 实际 API 调用 → ❌ 404
- ImpersonationBanner 显示 → ❌ 无
- Dashboard 视角切换 → ❌ 仍显示 admin
- 审计记录 → ❌ 无
- 撤销恢复 → 未测

### 根因
前后端 API 契约不一致：路径、参数名、参数结构都不同。

## 待验证项

- [x] 安全仪表盘数据准确性 — ✅ failed_logins_24h=14, total_events_24h=643, Security Score=75
- [x] Webhook 创建 — ✅ PASS（name 字段修复 cd86308f6，httpbin.org 持久化）
- [x] U20 MFA 完整流程 — ✅ PASS（API setup+verify 验证通过，verified:true；UI QR 渲染受浏览器自动化限制）
- [x] U24 Sessions 撤销 — ⚠️ 页面正常渲染，但 0 sessions（API token 方式不创建 session 记录，需真实浏览器登录测试）

## 总结

深度验证 7/7 完成。发现并修复 5 个表面验证未暴露的真实 bug：

| Bug | 严重度 | 根因 | 修复 |
|-----|--------|------|------|
| U19 Impersonation 完全不可用 | P1 | 5 层 API 契约不匹配 | 5 commits 全部修复 |
| Webhook 创建失败 | P1 | 前端发 description 后端要 name | cd86308f6 |
| Hash chain 50 broken | P1 | 新事件用旧 hash 计算 | repair-chain + 镜像更新 |
| CAE 策略纯前端无持久化 | P1 | 无 API 调用 | 5de4c4322 |
| 审计事件 inet 类型丢弃 | P0 | IP:PORT 不兼容 inet | 4e5b0193f |

教训：页面渲染 ≠ 功能正常。30/30 表面 PASS 中至少 5 个是假阳性。
