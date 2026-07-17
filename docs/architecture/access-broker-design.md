# Access Broker / Identity-Aware Proxy 设计 (ZTNA)

> 状态: Proposed | 作者: IAMExpert | 日期: 2026-07-17 | 来源: arch 下 sprint 最高优先级
> 关联: Gateway (reverse proxy + JWT + ABAC)、CAE (docs/architecture/cae-design.md)、PDP (docs/architecture/ztp-pdp-design.md)、ITDR

## 1. 愿景

GGID 从"IAM 平台"升级为"Zero Trust Access Platform"。任何内部应用（Grafana/Jenkins/Jupyter/internal dashboard/SSH/Web Terminal）放在 GGID 身份层后面，按身份+设备+上下文+ITDR 授权访问，**完全替代 VPN**。

**为什么 GGID 能做**：gateway 已有 reverse proxy + JWT 验证 + ABAC 条件 + CAE 实时吊销 + device trust + ITDR + 健康检查 + 限流 + stats。全部组件就绪，只差"受保护应用注册"层。

## 2. 总体架构

```
用户浏览器/CLI
    │ HTTPS
    ▼
┌─────────────────────────────────────────────────────────┐
│  GGID Gateway                                            │
│                                                          │
│  1. 请求路径匹配 → 查 protected_apps 路由表              │
│  2. JWT 验证（CAECheck jti 黑名单）                     │
│  3. PDP 评估：$app.access_policy + $security.*           │
│     ├─ allow → 注入 header → 代理到 upstream            │
│     ├─ deny → 403                                        │
│     └─ stepup → 402 Require-MFA → 重验证后放行          │
│  4. audit 事件（app.access + 异常检测联动 ITDR）         │
│                                                          │
│  受保护应用示例：                                        │
│  /app/grafana/  → http://grafana.internal:3000          │
│  /app/jenkins/  → http://jenkins.internal:8080          │
│  /app/jupyter/  → http://jupyter.internal:8888          │
│  /app/ssh/*     → WebSocket → SSH bastion              │
└─────────────────────────────────────────────────────────┘
         │ header injection
         ▼
┌─────────────────────────┐
│  Backend Application     │
│  (配置 trusted header    │
│   模式，信任 GGID 注入   │
│   的身份 header)         │
└─────────────────────────┘
```

## 3. 数据模型

### 3.1 migration 018_protected_apps.sql

```sql
CREATE TABLE protected_apps (
    id              UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    name            TEXT NOT NULL,              -- 'Grafana', 'Jenkins'
    slug            TEXT NOT NULL,              -- 'grafana' → URL /app/grafana/
    upstream_url    TEXT NOT NULL,              -- 'http://grafana.internal:3000'
    icon            TEXT,                       -- emoji or URL
    description     TEXT,

    -- 认证模式
    auth_mode       TEXT NOT NULL DEFAULT 'jwt', -- 'jwt' | 'session_header' | 'anonymous'

    -- 访问策略（ABAC 条件 JSON）
    access_policy   JSONB NOT NULL DEFAULT '{}',
    -- 示例: {"conditions":{"and":[{"$security.device_trusted":true},{"$security.itdr_critical_open":0},{"$app.role":"sre"}]}}

    -- Header 注入配置
    inject_headers  JSONB NOT NULL DEFAULT '[]',
    -- 示例: [{"name":"X-WebAuth-User","value":"$user.email"},{"name":"X-WebAuth-Roles","value":"$user.roles_csv"}]

    -- 健康检查
    health_check_path   TEXT DEFAULT '/health',
    health_check_interval INT DEFAULT 30,       -- 秒
    health_status       TEXT DEFAULT 'unknown', -- 'healthy'/'unhealthy'/'unknown'

    -- 限流（per-app）
    rate_limit_per_min  INT DEFAULT 100,

    -- 状态
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_by      UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE(tenant_id, slug)
);

CREATE INDEX idx_protected_apps_slug ON protected_apps(tenant_id, slug) WHERE enabled = TRUE;
ALTER TABLE protected_apps ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_iso ON protected_apps
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- 访问日志（每次受保护应用请求）
CREATE TABLE app_access_logs (
    id              UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    app_id          UUID NOT NULL,
    user_id         UUID,
    user_name       TEXT,
    method          TEXT NOT NULL,
    path            TEXT NOT NULL,
    status_code     INT NOT NULL,
    response_time_ms INT,
    ip_address      TEXT,
    user_agent      TEXT,
    pdp_decision    TEXT,                       -- 'allow'/'deny'/'stepup'
    pdp_reason      TEXT,                       -- 条件不满足的详情
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_app_logs_app_time ON app_access_logs(tenant_id, app_id, created_at DESC);
CREATE INDEX idx_app_logs_user ON app_access_logs(tenant_id, user_id, created_at DESC);
```

### 3.2 access_policy JSON 格式

复用 ZT PDP（docs/architecture/ztp-pdp-design.md）的 ABAC 条件 DSL：

```json
{
  "effect": "allow",
  "conditions": {
    "and": [
      {"$security.device_trusted": true},
      {"$security.itdr_critical_open": {"$eq": 0}},
      {"$security.is_business_hours": true},
      {"$user.role": {"$in": ["sre", "platform-eng"]}}
    ]
  },
  "deny_effect": "stepup",
  "deny_message": "此应用要求 SRE 角色 + 可信设备 + 工作时间访问"
}
```

- `$user.*`：从 JWT claims 提取（sub/roles/email/department）
- `$security.*`：从 PDP SignalCollector 获取（device/ITDR/risk/UEBA）
- `$app.*`：当前应用的元数据

## 4. Gateway 动态路由

### 4.1 路由表构建

```go
// ProtectedAppRouter 管理动态注册的受保护应用路由。
type ProtectedAppRouter struct {
    apps     map[string]*ProtectedAppProxy  // slug → proxy
    mu       sync.RWMutex
    repo     ProtectedAppRepo
    pdp      *SignalCollector               // PDP for access policy eval
    auditor  *audit.Publisher
}

type ProtectedAppProxy struct {
    App     *ProtectedApp
    Proxy   *httputil.ReverseProxy
    Policy  *AccessPolicy                   // 预编译的条件树
}
```

### 4.2 请求处理流程

```go
func (pr *ProtectedAppRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 1. 路径匹配：/app/{slug}/...
    slug := extractSlug(r.URL.Path)
    appProxy, ok := pr.getProxy(slug)
    if !ok {
        next.ServeHTTP(w, r) // 不是受保护应用，走正常路由
        return
    }

    // 2. JWT 验证（复用现有 JWTAuth + CAECheck）
    // — 已在 gateway 中间件链完成，此处只需读 context

    // 3. PDP 评估
    secCtx, _ := pr.pdp.Collect(r.Context(), userID, tenantID)
    decision := pr.pdp.Evaluate(appProxy.Policy, secCtx, userClaims)

    switch decision.Effect {
    case "deny":
        writeAccessDenied(w, decision.Reason)
        pr.logAccess(r, appProxy.App, "deny", decision.Reason)
        return
    case "stepup":
        writeRequireMFA(w, decision.Reason)
        pr.logAccess(r, appProxy.App, "stepup", decision.Reason)
        return
    case "allow":
        // 4. Header 注入
        pr.injectHeaders(r, appProxy.App, userClaims)
        // 5. 重写路径（去掉 /app/{slug} 前缀）
        r.URL.Path = strings.TrimPrefix(r.URL.Path, "/app/"+slug)
        // 6. 代理
        appProxy.Proxy.ServeHTTP(w, r)
        pr.logAccess(r, appProxy.App, "allow", "")
    }
}
```

### 4.3 热加载

protected_apps 表变更 → NATS 事件 `ggid.apps.changed` → gateway 订阅 → 重建路由表。

或 admin API：`POST /api/v1/gateway/apps/reload` → 触发全量刷新。

## 5. Session Passthrough Header 注入

### 5.1 支持的传统应用集成

| 应用 | 注入的 Header | 说明 |
|------|--------------|------|
| Grafana | `X-WebAuth-User: $user.email` + `X-WebAuth-Role: Admin/Viewer` | [Auth Proxy](https://grafana.com/docs/grafana/latest/auth/auth-proxy/) 模式 |
| Jenkins | `X-Forwarded-User: $user.username` + `X-Forwarded-Roles: $user.roles_csv` | Reverse Proxy Auth 模式 |
| Jupyter | `Jupyter-User: $user.email` | `c.Spawner.auth = HeaderAuth` |
| 通用 | `X-GGID-User-ID` + `X-GGID-User-Name` + `X-GGID-User-Email` + `X-GGID-Tenant-ID` + `X-GGID-Roles` | GGID 标准 header |

### 5.2 安全要求

- 注入的 header 值从 JWT claims 提取（不可被客户端伪造）
- **清除入站同名 header**：`r.Header.Del("X-WebAuth-User")` — 防止用户伪造
- 下游应用配置为仅接受 GGID gateway 的请求（网络隔离或 InternalAuth）

## 6. Console 管理页面

### 6.1 页面结构

| 页面 | 路径 | 功能 |
|------|------|------|
| 应用列表 | /security/apps | 所有受保护应用卡片网格（名称/图标/健康状态/访问次数/最近访问） |
| 注册应用 | /security/apps/new | 表单：name/slug/upstream_url/auth_mode/health_check + access_policy 构建器 + header 注入配置 |
| 编辑应用 | /security/apps/{id} | 同上 + 启停开关 |
| 访问日志 | /security/apps/{id}/logs | 实时访问日志（user/time/status/pdp_decision/response_time）+ filter |
| 实时监控 | /security/apps/{id}/monitor | 实时 QPS / latency / 4xx-5xx 率 / 活跃用户数（WebSocket 或 polling） |
| 策略配置器 | 嵌入 | 可视化条件构建器：IF device.trusted AND role IN [sre] AND business_hours THEN allow ELSE stepup |

### 6.2 UX 要求（生产级）

- 应用卡片：健康指示灯（green/red/yellow）+ 实时 QPS badge + 最近 5 条访问 mini-list
- 注册表单：upstream_url 填入后实时健康检查（试连 + 返回状态）
- 策略构建器：拖拽条件 + 实时预览 JSON + "测试此策略"（输入测试用户 → 输出 allow/deny）
- 访问日志：虚拟滚动（不卡顿）+ CSV 导出 + 时间线视图
- 空态："还没有受保护应用。添加你的第一个应用，开始零信任之旅。"

## 7. 与现有组件联动

| 组件 | 联动点 |
|------|--------|
| CAE | 用户 session 被 CAE 吊销 → 下次访问受保护应用 → gateway CAECheck → 401 |
| ITDR | critical detection → CAE revoke → 受保护应用访问自动阻断 |
| Device Trust | $security.device_trusted 注入 PDP → 不可信设备无法访问 Grafana |
| UEBA | $security.baseline_deviation 高 → stepup MFA |
| PAM JIT | $user.role 检查 JIT 提权是否 active → 无 active JIT → deny admin app |
| 数据安全法 | $data.classification 标记受保护应用处理的数据级别 → core 级要求额外条件 |
| Audit | 每次访问写 app_access_logs + audit 事件 → ITDR 可检测异常访问模式（新用户突然访问所有应用） |

## 8. WebSocket / SSH 支持

- WebSocket：httputil.ReverseProxy 原生支持（Upgrade 透传），无需额外开发
- SSH Bastion：`/app/ssh/{host}` → WebSocket → gateway 转发到 SSH bastion（后续 v2，需 terminal proxy）
- RDP/VNC：通过 Apache Guacamole 集成（v3）

## 9. API 设计

| Method | Path | 说明 |
|--------|------|------|
| GET | /api/v1/gateway/apps | 受保护应用列表 |
| POST | /api/v1/gateway/apps | 注册新应用 |
| GET | /api/v1/gateway/apps/{id} | 应用详情 |
| PUT | /api/v1/gateway/apps/{id} | 更新 |
| DELETE | /api/v1/gateway/apps/{id} | 删除 |
| POST | /api/v1/gateway/apps/{id}/test-policy | 测试 access_policy（输入测试用户 → 输出决策） |
| POST | /api/v1/gateway/apps/reload | 热加载路由表 |
| GET | /api/v1/gateway/apps/{id}/logs?since=&user=&limit= | 访问日志 |
| GET | /api/v1/gateway/apps/{id}/stats | 实时统计（QPS/latency/error_rate/active_users） |
| GET | /api/v1/gateway/apps/{id}/health | 健康检查状态 |

## 10. 测试计划

| 测试 | 验证点 |
|------|--------|
| 路由匹配 | /app/grafana/ → 正确 proxy；/app/unknown/ → 404 |
| PDP allow | 满足条件 → 代理 + header 注入 |
| PDP deny | 条件不满足 → 403 + 原因 |
| PDP stepup | 灰区 → 402 Require-MFA |
| header 清除 | 客户端伪造 X-WebAuth-User → 被清除 |
| CAE 联动 | session 吊销 → 访问受保护应用 → 401 |
| 健康检查 | upstream 不可达 → health=unhealthy → 应用卡片变红 |
| 热加载 | DB 新增 app → NATS → gateway 路由更新 → 立即可用 |
| 并发 | 100 并发请求 → 无 race |
| 租户隔离 | tenant A 的 app 对 tenant B 不可见 |

## 11. 工作量

| Phase | 内容 | 文件 | 预估 |
|-------|------|------|------|
| 1 | migration 018 + ProtectedAppRepo + app_access_logs | migrations/, services/gateway/ | 1d |
| 2 | ProtectedAppRouter（路由 + PDP 评估 + header 注入） | services/gateway/internal/router/ | 2d |
| 3 | 热加载（NATS 订阅 + admin reload API） | services/gateway/ | 0.5d |
| 4 | API 端点 10 个 + test-policy + stats + logs | services/gateway/internal/handler/ | 1.5d |
| 5 | Console 页面（应用列表/注册/编辑/日志/监控/策略构建器） | console/src/app/security/apps/ | 1.5d |
| **总计** | | | **~6.5d** |
