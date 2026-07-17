# PAM JIT Zero Standing Privileges 设计

> 状态: Proposed | 作者: IAMExpert | 日期: 2026-07-17 | 来源: arch Round 91 sprint 排期 #7
> 关联: CAE (docs/architecture/cae-design.md §3.4 — JIT 到期后 session 降权)

## 1. 问题

CSA 2025 主推"杀死常驻权限"（Zero Standing Privileges）：管理员不应永久持有 admin 角色，而是在需要时临时提权（JIT elevation），有时间限制 + 审批 + 全审计。

**GGID 现状审计**：

| 组件 | 现状 | 缺口 |
|------|------|------|
| break-glass | GET history only（内存数组，重启丢失） | 无 activate / 审批 / 到期撤销 / DB 持久化 |
| jit-elevation 页面 | console 页面 + `/api/v1/policy/jit-elevation` | 端点存在但无审批流 / 临时绑定 / 到期解绑 |
| UserRole.ExpiresAt | **已存在** `*time.Time` + repo 查询已过滤过期 | 基础设施齐备，只差上层逻辑 |
| RevokeRole | 已实现（role_service.go:173） | 可复用于到期解绑 |

**好消息**： UserRole 模型已有 ExpiresAt 字段，repo 的 `ListByUser` 已有 `expires_at IS NULL OR expires_at > NOW()` 过滤。基础设施完全就绪，只需补上层 JIT 工作流。

## 2. 总体架构

```
用户请求提权 → [审批流] → 临时角色绑定(expires_at) → 使用 → 到期自动解绑
     ↓                                    ↓                    ↓
  audit事件                          audit事件            audit事件 + CAE session降权
```

### 2.1 JIT Request 生命周期

```
Status: pending → approved/rejected → active → expired/revoked
        ↑              ↑                ↑          ↑
     用户提交       审批者决策       绑定生效    到期/手动撤销
```

### 2.2 组件

| 组件 | 位置 | 职责 |
|------|------|------|
| JITRequestService | policy/internal/service/jit_service.go | 请求 CRUD + 审批 + 角色绑定/解绑 |
| JITRequestRepo | policy/internal/repository/jit_repo.go | Postgres 持久化 |
| BreakGlassService | auth/internal/service/break_glass_service.go | 紧急提权（简化审批 + 强审计） |
| ExpirySweeper | policy/cmd/main.go goroutine | 每分钟扫描到期请求 → RevokeRole + audit |

## 3. 数据模型

### 3.1 migration 014_jit_requests.sql

```sql
CREATE TABLE jit_requests (
    id              UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,          -- 请求者
    role_id         UUID NOT NULL,          -- 申请的角色
    scope_type      TEXT NOT NULL DEFAULT 'tenant',
    scope_id        UUID,
    reason          TEXT NOT NULL,           -- 业务理由
    duration_min    INT NOT NULL,            -- 申请时长（分钟），max 480(8h)
    status          TEXT NOT NULL DEFAULT 'pending',  -- pending/approved/rejected/active/expired/revoked
    approver_id     UUID,                   -- 审批者
    approved_at     TIMESTAMPTZ,
    activated_at    TIMESTAMPTZ,             -- 角色绑定时间
    expires_at      TIMESTAMPTZ,             -- activated_at + duration
    revoked_at      TIMESTAMPTZ,
    revoked_reason  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_jit_req_user ON jit_requests(tenant_id, user_id, status);
CREATE INDEX idx_jit_req_expiry ON jit_requests(status, expires_at) WHERE status = 'active';
ALTER TABLE jit_requests ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_iso ON jit_requests USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

### 3.2 break_glass_records 迁移（内存 → DB）

```sql
CREATE TABLE break_glass_records (
    id              UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    requester_id    UUID NOT NULL,
    requester_name  TEXT NOT NULL,
    reason          TEXT NOT NULL,
    scope           TEXT NOT NULL,           -- 受影响系统/范围
    duration_min    INT NOT NULL DEFAULT 60,
    status          TEXT NOT NULL DEFAULT 'active', -- active/expired/revoked
    approver_id     UUID,                   -- 可选：双人审批
    activated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_bg_active ON break_glass_records(tenant_id, status, expires_at) WHERE status = 'active';
```

## 4. API 设计

### 4.1 JIT Request（policy 服务）

| Method | Path | 说明 |
|--------|------|------|
| POST | /api/v1/policies/jit/request | 用户提交提权请求（role_id, scope, duration_min, reason） |
| GET | /api/v1/policies/jit/requests?status=&user_id= | 列表（自己/待审批） |
| POST | /api/v1/policies/jit/requests/{id}/approve | 审批者批准 → 绑定临时角色 |
| POST | /api/v1/policies/jit/requests/{id}/reject | 审批者拒绝（body: reason） |
| POST | /api/v1/policies/jit/requests/{id}/revoke | 手动撤销（管理员/自己提前放弃） |
| GET | /api/v1/policies/jit/active | 当前活跃提权（dashboard） |

**审批规则**：
- auto-approve：role 配置 `jit_auto_approve=true` 且 duration ≤ role.jit_max_duration → 直接 active（如 DBA 临时查询 30min）
- manual：需 approver（role.jit_approver_role 指定的角色持有者）批准
- 规则可配置：itdr_rules 同表或新 jit_role_configs 表

### 4.2 Break-Glass（auth 服务）

| Method | Path | 说明 |
|--------|------|------|
| POST | /api/v1/auth/break-glass/activate | 紧急提权（强审计：reason 必填 + webhook 告警 + audit critical） |
| GET | /api/v1/auth/break-glass/history | 历史记录（迁 DB） |
| GET | /api/v1/auth/break-glass/active | 当前活跃紧急会话 |
| POST | /api/v1/auth/break-glass/{id}/revoke | 撤销（安全官） |

**break-glass vs JIT 区别**：
- JIT：有审批流，适合可预见的临时提权（DBA 日常运维）
- break-glass：**无审批或自动审批**，适合紧急故障恢复（生产宕机），代价是**最强审计**（每次触发 SOC webhook + audit critical + ITDR 可选检测）

## 5. ExpirySweeper（到期自动解绑）

```go
// 每分钟扫描到期的 JIT requests 和 break-glass records。
func (s *ExpirySweeper) Run(ctx context.Context) {
    ticker := time.NewTicker(60 * time.Second)
    for {
        select {
        case <-ticker.C:
            // 1. JIT: 找 status=active AND expires_at < now → RevokeRole + status=expired + audit
            expired, _ := s.jitRepo.ListExpired(ctx)
            for _, req := range expired {
                s.roleSvc.RevokeRole(ctx, req.UserID, req.RoleID, req.ScopeType, req.ScopeID)
                s.jitRepo.UpdateStatus(ctx, req.ID, "expired")
                s.auditPub.Publish(ctx, audit.Event{
                    Action: "jit.expired", ActorID: &req.UserID,
                    Metadata: map[string]any{"role_id": req.RoleID, "reason": "auto_expired"},
                })
            }
            // 2. Break-glass: 同理
            // 3. CAE 联动：到期用户的 session 需要 session 降权（可选 v1 — 仅 audit 告警）
        case <-ctx.Done():
            return
        }
    }
}
```

**CAE 联动（v2）**：JIT 到期 → RevokeRole → 如果用户当前 session 的角色集变化 → CAE 检测到权限缩减 → 可选 session revoke 或 step-up 要求。v1 仅 audit 记录差异，v2 联动 CAE。

## 6. 安全要求

| 要求 | 实现 |
|------|------|
| 理由必填 | API 校验 reason != "" |
| 最大时长 | duration_min ≤ 480（8h），超限拒绝 |
| 审计闭环 | 6 个状态转换各写 audit 事件 |
| 审批者不能审批自己 | approver_id != user_id 校验 |
| break-glass webhook | 每次 activate 发 SOC 告警 webhook |
| 到期不可绕过 | ExpirySweeper 每分钟执行；repo 查询已过滤 expires_at |
| 并发安全 | status 转换用 `WHERE status = 'pending'` 乐观锁 |

## 7. 测试计划

| 测试 | 验证点 |
|------|--------|
| JIT request → approve → active | UserRole 绑定 + expires_at 设置 |
| ExpirySweeper 到期 | status → expired + RevokeRole 调用 + audit |
| 手动 revoke | 即时 RevokeRole + status → revoked |
| 自审批拒绝 | approver_id == user_id → 400 |
| 超时拒绝 | duration_min > 480 → 400 |
| break-glass activate | DB 记录 + audit critical + webhook 调用 |
| break-glass 到期 | status → expired |
| 并发审批 | 两个 goroutine 同时 approve → 只有一个成功 |
| E2E：JIT → 使用 → 到期 → CAE | 权限缩减检测 |

## 8. 工作量

| Phase | 内容 | 预估 |
|-------|------|------|
| 1 | migration 014 + 015 + JITRequestRepo + BreakGlassRepo | 0.5d |
| 2 | JITRequestService（request/approve/reject/revoke）+ ExpirySweeper | 1d |
| 3 | API 端点 6 个 + break-glass activate/revoke | 0.5d |
| 4 | BreakGlass 内存→DB 迁移 + 删除旧内存代码 | 0.5d |
| 5 | 测试 + E2E | 0.5d |
| **总计** | | **~3d** |
