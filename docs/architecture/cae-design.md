# CAE 持续访问评估设计 (Continuous Access Evaluation)

> 状态: Proposed | 作者: IAMExpert | 日期: 2026-07-17 | 来源: arch Round 91 排期
> 关联: ITDR 引擎 (docs/architecture/itdr-engine-design.md §6 Playbook v2)、零信任 CAE (docs/team-backlog.md)

## 1. 问题

NIST SP 800-207 核心原则：访问授权不是一次性登录事件。当会话期间上下文恶化（ITDR critical detection、设备失信任、管理员手动吊销），访问应**立即**撤销——不是等 JWT 自然过期（15-30 分钟窗口）。

GGID 现状审计发现两个致命缺口：

| 缺口 | 现状 | 后果 |
|------|------|------|
| Gateway 不检查 jti 吊销 | JWT 验证只看签名+exp | 已吊销的 token 在 TTL 内仍然有效 |
| jti 黑名单是内存 map | `jtiBlocklist` in-memory | 多实例不同步，auth 实例 1 吊销 → gateway 不知道 → 实例 2 不知道 |

这意味着：**即使是管理员手动吊销会话，该用户的 JWT 在过期前仍然完全有效**。CAE 的目标是将这个窗口从 15-30 分钟压缩到 <1 秒。

## 2. 总体架构

```
ITDR Engine (audit)                    Admin Console
  │ critical detection                   │ POST /auth/sessions/revoke
  ▼                                      ▼
┌─────────────────────────────────────────────────┐
│              NATS: ggid.session.revoke            │
│   { user_id, session_ids[], jtis[], reason }      │
└─────────────────────────────────────────────────┘
        │ subscribe
        ▼
┌─────────────────────────────┐
│     auth service             │
│  SessionRevocationManager    │
│  1. Revoke DB sessions       │
│  2. Add jtis → Redis SET     │
│     key: ggid:revoked_jti    │
│     TTL: JWT exp - now       │
│  3. Delete Redis sessions    │
└─────────────────────────────┘

--- Gateway JWT Verification (CAE Check) ---

  Client Request → Gateway
       │
       ▼
  1. JWT signature + exp (existing)
       │
       ▼
  2. Extract jti from claims
       │
       ▼
  3. Redis SISMEMBER ggid:revoked_jti <jti> (~0.3ms)
       │
       ├─ member exists → 401 "session revoked"
       │
       └─ not member → continue (authorized)
```

## 3. 核心组件

### 3.1 Redis JTI 黑名单（替代内存 map）

**位置**：`pkg/auth/jti_blocklist.go`（所有服务可复用）

```go
// JTIBlocklist 管理已吊销 JWT 的 jti，使用 Redis SET + per-entry TTL。
type JTIBlocklist interface {
    // Revoke 添加 jti 到黑名单，TTL 为 JWT 剩余有效期。
    Revoke(ctx context.Context, jti string, jwtExp time.Time) error
    // RevokeUser 吊销用户全部活跃 jti（查 DB session 表获取 jti 列表）。
    RevokeUser(ctx context.Context, userID uuid.UUID) error
    // IsRevoked 检查单个 jti。O(1)，~0.3ms。
    IsRevoked(ctx context.Context, jti string) (bool, error)
}
```

**Redis 存储设计**：
```
# 使用 Redis Sorted Set（score=exp_unix），非普通 SET
ZADD ggid:revoked_jti <exp_unix_ts> "<jti>"
# 查询
ZSCORE ggid:revoked_jti "<jti>"  # 存在=已吊销
# 清理过期条目（定时 ZREMRANGEBYSCORE 0 <now>）
ZREMRANGEBYSCORE ggid:revoked_jti 0 <current_unix_ts>
```

为什么用 ZSET 而非 SET：自动过期清理，避免黑名单无限膨胀。非 SET + per-key TTL 因为 490 个 jti × 独立 key 会浪费内存。

**多租户**：jti 全局唯一（UUID），无需租户前缀。但 stats 可按租户聚合（revoke 事件带 tenant_id）。

### 3.2 Gateway CAE 中间件

**位置**：`services/gateway/internal/middleware/middleware.go`，在 JWTAuth 之后追加

```go
// CAECheck 在 JWT 验证通过后检查 jti 是否已被吊销。
// 延迟预算：Redis SISMEMBER ~0.3ms（同机房），总计 <1ms。
func CAECheck(rdb *redis.Client) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 从 JWTAuth 注入的 context 取 jti
            jti, ok := middleware.JTIFromRequest(r)
            if !ok {
                next.ServeHTTP(w, r) // 无 jti 的 token（少见），放行
                return
            }
            // Redis 检查（pipeline + local cache 1s TTL 减少 Redis QPS）
            score, err := rdb.ZScore(ctx, "ggid:revoked_jti", jti).Result()
            if err == nil && score > 0 {
                writeUnauthorized(w, "session revoked (CAE)")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

**性能优化 — 本地缓存**：
- gateway 进程内 `sync.Map` 做 1 秒 TTL 本地缓存（同一 jti 1 秒内只查 Redis 一次）
- 高频场景（同一用户连续请求）Redis QPS 降 90%
- 本地缓存只能缓存 **not-revoked** 结果（revoked 状态必须实时，1 秒窗口可接受）

**降级策略**：Redis 不可用时 → 放行 + warn 日志（不阻断业务），与 sysconfig hot-reload 的优雅降级一致。

### 3.3 SessionRevocationManager（auth 服务）

**位置**：`services/auth/internal/service/session_revocation_manager.go`

```go
type SessionRevocationManager struct {
    sessionRepo  SessionRepository    // 查 DB 获取用户活跃 session + jti
    jtiBlocklist JTIBlocklist         // Redis 黑名单
    nats         *nats.Conn           // 订阅 revoke 事件
    auditPub     *audit.Publisher     // 写 audit 事件
}

// RevokeUser 吊销用户全部会话（管理员/CAE 触发）。
// 延迟目标：<100ms（DB 查询 + Redis 写入 + audit publish）
func (m *SessionRevocationManager) RevokeUser(ctx context.Context, userID, tenantID uuid.UUID, reason string) error {
    // 1. 查 DB：用户活跃 sessions + jtis
    sessions, err := m.sessionRepo.GetActiveByUser(ctx, tenantID, userID)
    // 2. Redis：批量 ZADD jti 黑名单（TTL = 各 token exp）
    for _, s := range sessions {
        m.jtiBlocklist.Revoke(ctx, s.JTI, s.TokenExp)
    }
    // 3. DB：标记 sessions revoked
    m.sessionRepo.RevokeAllByUser(ctx, tenantID, userID)
    // 4. Redis：删除 session keys（refresh token 无法续期）
    for _, s := range sessions {
        rdb.Del(ctx, "ggid:session:"+s.ID.String())
    }
    // 5. Audit 事件
    m.auditPub.Publish(ctx, audit.Event{
        Action: "session.revoked", ActorType: "system",
        Metadata: map[string]any{"reason": reason, "user_id": userID},
    })
    return nil
}
```

### 3.4 ITDR → CAE 联动

**触发路径**：ITDR playbook v2 新增 `revoke_sessions` 自动响应动作

```
ITDR Engine → critical detection → playbook.Dispatch()
    ├─ playbook action = "revoke_sessions"
    │   → POST /api/v1/auth/internal/revoke-user (NATS 或 HTTP)
    │   → SessionRevocationManager.RevokeUser()
    │   → Redis ZADD + DB revoke + audit
    │
    └─ playbook action = "lock_account"
        → POST /api/v1/identity/internal/lock-user
        → identity 服务标记 user.locked=true
```

**端到端延迟预算（<1s 目标）**：

| 阶段 | 耗时 | 说明 |
|------|------|------|
| ITDR detection → NATS publish | ~5ms | 已有 NATS 基础设施 |
| NATS → auth consumer | ~10ms | JetStream 投递 |
| RevokeUser 执行 | ~50ms | DB 查 + Redis ZADD + audit |
| Redis ZADD 生效 | ~1ms | Redis 写入即对 gateway 可见 |
| Gateway 下次请求检测 | ~0.3ms | ZSCORE 查询 |
| **总计** | **~66ms** | **远低于 1s 目标** |

最坏情况：用户恰好在这一秒发了请求 → 该请求通过（使用了过期前的缓存），下一秒被拦截。1 秒窗口可接受（与本地缓存 TTL 对齐）。

### 3.5 NATS Subject 设计

```
# 吊销事件
Subject: ggid.session.revoke
Payload: { "user_id": "...", "tenant_id": "...", "session_ids": [...], "jtis": [...], "reason": "itdr_critical" }

# auth 服务订阅
Consumer: auth-revocation-listener
```

auth 服务的 NATS consumer 在 ITDR `revoke_sessions` playbook 触发时接收事件。与 audit NATS consumer 并行运行（不同 subject，不同 consumer group）。

## 4. API 端点

| Method | Path | 说明 | 认证 |
|--------|------|------|------|
| POST | /api/v1/auth/sessions/revoke-user | 按 userID 吊销全部会话 | JWT admin |
| POST | /api/v1/auth/internal/revoke-user | 内部调用（ITDR playbook） | InternalAuth |
| GET | /api/v1/auth/sessions/revocations/stats | 吊销统计（24h count、active blocklist size） | JWT admin |

## 5. 迁移路径

### 5.1 替换内存 jtiBlocklist

现有 `impersonation.go` 的内存 jtiBlocklist → 替换为 Redis 实现。调用点：
- `RevokeAllUserSessions(jtis)` → `jtiBlocklist.Revoke(ctx, jti, exp)` 循环
- `IsJTIRevoked(jti)` → `jtiBlocklist.IsRevoked(ctx, jti)`

兼容期：Redis 不可用时 fallback 到内存 map（dev 模式）。

### 5.2 Gateway 中间件链更新

```
现有：PanicRecovery → SecurityHeaders → ... → JWTAuth → proxy
新增：... → JWTAuth → CAECheck → proxy
```

`JWTAuth` 需要将 jti 注入 context（当前只注入 user_id/tenant_id，需补 jti）。

### 5.3 Session 表增加 jti 列

```sql
-- migration 013 (或合入下一个 migration)
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS jti TEXT;
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS token_exp TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_sessions_jti ON sessions(jti) WHERE jti IS NOT NULL;
```

登录时 token_service.IssueAccessToken 返回的 jti 需写回 session 记录。

## 6. 测试计划

| 测试 | 验证点 |
|------|--------|
| 单元：JTIBlocklist Redis 实现 | Revoke → IsRevoked=true；TTL 过期 → false |
| 单元：SessionRevocationManager.RevokeUser | DB sessions revoked + Redis ZADD + audit published |
| 单元：CAECheck 中间件 | revoked jti → 401；valid jti → 200；无 jti → 放行 |
| 单元：Redis 降级 | Redis 断开 → 放行 + warn 日志 |
| E2E：ITDR → CAE 全链路 | 发 10 次失败登录 → brute_force detection → playbook revoke_sessions → 用户下次请求 401 |
| E2E：延迟测量 | detection timestamp → gateway 401 timestamp < 1000ms |
| 并发：本地缓存正确性 | 100 并发同 jti → Redis 只查 1 次 |
| 回归：正常请求无影响 | 非 revoked token 请求延迟增加 < 1ms |

## 7. 工作量

| Phase | 内容 | 文件 | 预估 |
|-------|------|------|------|
| 1 | pkg/auth/jti_blocklist.go (Redis ZSET) + 测试 | pkg/auth/ | 0.5d |
| 2 | SessionRevocationManager + NATS consumer | services/auth/ | 0.5d |
| 3 | migration (jti/token_exp) + session repo + token_service 写 jti | services/auth/ | 0.5d |
| 4 | Gateway CAECheck 中间件 + JWTAuth jti 注入 + 本地缓存 | services/gateway/ | 0.5d |
| 5 | ITDR playbook revoke_sessions action | services/audit/ | 0.5d |
| 6 | E2E 测试 + 延迟验证 | test/ | 0.5d |
| **总计** | | | **~3d** |

## 8. 风险与缓解

| 风险 | 缓解 |
|------|------|
| Redis ZSCORE 每请求增加延迟 | 本地 1s 缓存 + pipeline → P99 增加 <1ms |
| Redis 不可用 | 降级放行（与现有 sysconfig 模式一致）；内存 fallback |
| 黑名单 ZSET 膨胀 | ZREMRANGEBYSCORE 定时清理（cron 或 consumer 附带） |
| 多 gateway 实例缓存不一致 | 1 秒 TTL 是可接受窗口；紧急场景管理员可调 /sessions/revocations/flush-cache |
| ITDR → revoke 误报（合法用户被踢） | playbook severity=high 仍用 NATS 通知不自动 revoke，仅 critical 自动 revoke + audit 留证 |
