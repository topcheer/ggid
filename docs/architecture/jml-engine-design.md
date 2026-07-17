# JML 身份编排引擎设计 (Joiner-Mover-Leaver Orchestration)

> 状态: Proposed | 作者: IAMExpert | 日期: 2026-07-17 | 来源: arch 下 sprint 排期 #1
> 关联: ITDR 引擎模式 (docs/architecture/itdr-engine-design.md)、CAE (docs/architecture/cae-design.md)、PAM JIT

## 1. 问题

Identity Orchestration 是 2025 IAM 增长引擎。核心能力：HR 系统变更 → 自动触发 JML 规则 → 级联开通/变更/回收权限 + 通知 + 审计。

**GGID 现状审计**：

| 组件 | 现状 | 缺口 |
|------|------|------|
| LifecycleRule 模型 | trigger + actions 定义完整 | 内存 map 存储，不持久化 |
| lifecycle CRUD 端点 | 有（create/list/preview） | 规则没人执行 |
| provision_webhook | 接收外部 HR 事件（user.created/deleted/role_changed） | 收到后只写 log，不触发规则 |
| deprovision handler | 手动 API 调用 | 不自动触发 |
| CAE RevokeUser | 已实现（NATS → Redis jti → gateway 401） | leaver 应联动但未连 |

**核心缺口**：规则定义存在，但 **trigger→action 执行链未接线**。

## 2. 设计目标

复用 ITDR 引擎模式（规则注册 + 事件评估 + NATS 消费），实现 HR 事件 → JML 规则匹配 → 自动执行 actions → 审计闭环。

## 3. 总体架构

```
HR System (Workday/BambooHR/飞书)
    │ POST /api/v1/users/provision-webhook
    │ { event: "user.created", user: {...}, department: "engineering" }
    ▼
┌─────────────────────────────────────────────┐
│  identity: provision_webhook handler         │
│  1. Parse event type                         │
│  2. Publish to NATS: ggid.lifecycle.event    │
└─────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────┐
│  identity: JML Engine (NEW)                  │
│  Consumer: ggid.lifecycle.event              │
│  1. Map event → trigger (created→joiner,     │
│     role_changed→mover, deleted→leaver)      │
│  2. Find matching LifecycleRules             │
│  3. Execute actions sequentially             │
│  4. Each action writes audit event           │
└─────────────────────────────────────────────┘
    │
    ├── assign_role → policy service API
    ├── revoke_access → policy revoke + CAE RevokeUser (NATS ggid.session.revoke)
    ├── notify_manager → webhook/email
    ├── create_account → identity CreateUserFromSocial
    └── disable_account → identity deprovision
```

## 4. 数据模型

### 4.1 migration 017_jml_rules.sql

```sql
CREATE TABLE lifecycle_rules (
    id          UUID PRIMARY KEY,
    tenant_id   UUID NOT NULL,
    name        TEXT NOT NULL,
    trigger     TEXT NOT NULL,              -- 'joiner', 'mover', 'leaver'
    conditions  JSONB NOT NULL DEFAULT '{}', -- {"department": "engineering", "role": "developer"}
    actions     JSONB NOT NULL DEFAULT '[]', -- [{"type": "assign_role", "params": {"role_id": "..."}}, {"type": "notify_manager"}]
    priority    INT NOT NULL DEFAULT 100,    -- 低优先级先执行
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_jml_rules_trigger ON lifecycle_rules(tenant_id, trigger) WHERE enabled = TRUE;
ALTER TABLE lifecycle_rules ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_iso ON lifecycle_rules
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- 执行日志
CREATE TABLE lifecycle_executions (
    id          UUID PRIMARY KEY,
    tenant_id   UUID NOT NULL,
    rule_id     UUID NOT NULL,
    user_id     UUID NOT NULL,
    trigger     TEXT NOT NULL,
    action_type TEXT NOT NULL,
    action_params JSONB,
    result      TEXT NOT NULL,              -- 'success', 'failed', 'skipped'
    error       TEXT,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_jml_exec_user ON lifecycle_executions(tenant_id, user_id, executed_at DESC);
```

### 4.2 条件匹配

conditions 是 JSONB，匹配 provision webhook 事件中的用户属性：

```json
{
  "department": "engineering",
  "seniority": "senior+",
  "source_idp": "workday"
}
```

匹配逻辑：事件 payload 中的用户属性满足所有 conditions 键值对。支持通配符 `*`。

## 5. Action 执行器

```go
type ActionExecutor struct {
    policyClient  PolicyClient   // HTTP: assign/revoke role
    authClient    AuthClient     // HTTP: CAE revoke-user
    identityRepo  IdentityRepo   // local: create/disable user
    webhookSender WebhookSender  // notify
    auditPub      *audit.Publisher
}

func (e *ActionExecutor) Execute(ctx context.Context, action LifecycleAction, userID, tenantID uuid.UUID) error {
    switch action.Type {
    case "assign_role":
        roleID := uuid.Parse(action.Params["role_id"].(string))
        return e.policyClient.AssignRole(ctx, userID, roleID, tenantID)
    
    case "revoke_access":
        // 1. 撤销所有角色
        e.policyClient.RevokeAllRoles(ctx, userID, tenantID)
        // 2. CAE: 撤销所有会话
        e.authClient.RevokeUser(ctx, userID, "lifecycle_leaver")
    
    case "notify_manager":
        e.webhookSender.Send(action.Params["webhook_url"], map[string]any{
            "event": "lifecycle_action", "user_id": userID, ...
        })
    
    case "create_account":
        // 已有 CreateUserFromSocial，复用
    case "disable_account":
        // 已有 deprovision handler，复用
    }
    // 写 audit 事件
    e.auditPub.Publish(ctx, audit.Event{
        Action: "lifecycle." + action.Type, ...
    })
}
```

## 6. 事件 → Trigger 映射

| provision_webhook event | JML trigger | 示例 |
|------------------------|-------------|------|
| user.created | joiner | 新员工入职 → 分配部门默认角色 |
| user.role_changed | mover | 调岗 → 撤销旧角色 + 分配新角色 |
| user.deactivated | leaver (soft) | 休假 → 暂停权限（不删除） |
| user.deleted | leaver (hard) | 离职 → 撤销全部 + CAE session revoke |
| user.reactivated | rejoiner | 销假 → 恢复权限 |

## 7. CAE 联动（leaver 场景）

```
HR: user.deleted → provision_webhook → NATS ggid.lifecycle.event
→ JML Engine: trigger=leaver → revoke_access action
  ├─ policy: RevokeAllRoles(userID)
  ├─ NATS ggid.session.revoke → CAE RevokeUser → Redis ZADD jtis
  └─ gateway: CAECheck → 401 (离职者 <1s 被踢)
```

**端到端延迟**：HR 事件 → 用户失去访问 <3s（webhook 接收 100ms + NATS 10ms + 规则匹配 50ms + revoke 100ms + CAE 1ms）。

## 8. API 设计

| Method | Path | 说明 |
|--------|------|------|
| GET/POST | /api/v1/users/lifecycle/rules | 列表 / 创建规则（迁 DB） |
| PUT/DELETE | /api/v1/users/lifecycle/rules/{id} | 更新 / 删除 |
| GET | /api/v1/users/lifecycle/executions | 执行日志 |
| POST | /api/v1/users/lifecycle/dry-run | 预览：给定事件 → 会触发哪些规则和 actions |
| POST | /api/v1/users/provision-webhook | HR 事件入口（已有，改造为发布 NATS） |

## 9. 迁移现有代码

| 现有文件 | 改动 |
|---------|------|
| lifecycle_handler.go | 内存 map → lifecycle_rules DB；CRUD 不变 |
| provision_webhook_handler.go | 收到事件后发布 NATS ggid.lifecycle.event |
| 新增 lifecycle_engine.go | NATS consumer + 规则匹配 + action 执行 |
| 新增 lifecycle_executor.go | Action 执行器 |

## 10. 测试

| 测试 | 验证点 |
|------|--------|
| joiner: user.created + 匹配规则 | assign_role 执行 + audit |
| leaver: user.deleted → revoke_all + CAE | session 401 <3s |
| 条件不匹配 | 0 actions 执行 |
| 多规则同 trigger | 按 priority 顺序执行 |
| action 失败 | 记录 failed + 继续后续 action |
| dry-run | 预览匹配规则不执行 |

## 11. 工作量

| Phase | 内容 | 预估 |
|-------|------|------|
| 1 | migration 017 + LifecycleRuleRepo + 执行日志表 | 0.5d |
| 2 | JML Engine（NATS consumer + 规则匹配） | 0.5d |
| 3 | ActionExecutor（5 种 action + CAE 联动） | 1d |
| 4 | provision_webhook 改造 + API 端点 | 0.5d |
| 5 | 迁移内存 lifecycle_handler → DB + 测试 | 0.5d |
| **总计** | | **~3d** |
