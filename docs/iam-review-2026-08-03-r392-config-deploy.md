# R392 — 配置管理（第17轮）+ 部署可靠性（第17轮）深度审计

**审查日期**: 2026-08-03  
**审查范围**: services/*/cmd/, services/*/internal/{conf,config}/, deploy/（Docker/K8s/Helm/migrations）, pkg/, Makefile  
**审查模式**: 只读纯代码审查，逐行检查

---

## 审查路径汇总

### 配置管理代码路径
- `services/auth/internal/conf/conf.go` — Auth Config 结构体 + LoadFromEnv（182行）
- `services/identity/internal/conf/conf.go` — Identity Config 结构体 + DBConfig（31行）
- `services/oauth/internal/conf/conf.go` — OAuth Config + DBConfig（54行）
- `services/gateway/internal/config/config.go` — Gateway Config + envOrDefault + LoadFromEnv（257行）
- `services/policy/internal/config/config.go` — Policy Config + getEnv/getEnvInt（44行）
- `services/audit/internal/config/config.go` — Audit Config + getEnv/getEnvInt（54行）
- `services/{policy,org,audit}/internal/data/db.go` — data.Config DB配置结构体（×3重复）
- `services/*/cmd/main.go` — 全部7个服务入口

### 部署代码路径
- `console/Dockerfile`（44行）
- `services/{identity,auth,gateway,oauth,policy,org,audit}/Dockerfile`（各16-17行）
- `deploy/docker/Dockerfile.service`（41行）
- `deploy/docker-compose.yaml`（360行）
- `deploy/docker-compose.prod.yaml`（518行）
- `deploy/.env.example`（58行）
- `deploy/k8s/mcp-deployment.yaml`（82行）
- `deploy/k8s/db-migrate-job.yaml`（47行）
- `deploy/k8s/console-deployment.yaml`（129行）
- `deploy/helm/ggid/{Chart.yaml, values.yaml, values-prod.yaml}`
- `deploy/migrations/`（53个SQL文件，检查编号冲突）

---

## 审查角度1：配置管理

### C1. conf vs config 包分裂  [P0 — 持续17轮]

**文件/行号**:
- `services/auth/internal/conf/conf.go` — 包名 `conf`
- `services/identity/internal/conf/conf.go` — 包名 `conf`
- `services/oauth/internal/conf/conf.go` — 包名 `conf`
- `services/gateway/internal/config/config.go` — 包名 `config`
- `services/policy/internal/config/config.go` — 包名 `config`
- `services/audit/internal/config/config.go` — 包名 `config`

**问题**: 7个服务中3个使用 `conf`，3个使用 `config`，org 也有 `config`。包命名不一致导致：
1. import路径需要记忆每个服务用的是哪个名字
2. 新开发者需要查文档才知道该用哪个
3. 跨服务复制代码容易引入错误

**风险**: 高 — 降低可维护性，增加新开发者犯错概率。如果合并为统一包名，可减少认知负担。

**建议修复**: 统一为 `config`（Go社区惯例更常用全词）：
```
mv services/auth/internal/conf  services/auth/internal/config
mv services/identity/internal/conf services/identity/internal/config
mv services/oauth/internal/conf services/oauth/internal/config
# 批量更新import路径
```

---

### C2. os.Getenv 散用 — 259处跨73文件  [P1]

**文件/行号**:
- 全库搜索: `os.Getenv(` → 73文件, 259处匹配
- 其中 `envOrDefault` + `getEnv` + `GetEnv` + `GetEnvInt` → 10文件, 126处匹配
- `pkg/middleware/internal_auth.go`: 4处直接 os.Getenv
- `pkg/notification/dispatcher.go`: 5处
- `services/audit/internal/config/config.go`: 15处

**问题**: 大量直接使用 `os.Getenv` 而非统一配置加载函数。分散的环境变量读取导致：
1. 配置项清单不透明 — 无法从单一来源获取所有配置
2. 默认值散落各处
3. 无法统一验证（如必填项检查）

**量化分析**:
| 方式 | 文件数 | 调用数 |
|------|--------|--------|
| os.Getenv | 73 | 259 |
| envOrDefault/getEnv/GetEnv | 10 | 126 |
| **总计** | **~80** | **~385** |

**风险**: 中 — 不直接造成安全漏洞，但增加配置漂移和遗漏验证的风险。

**建议修复**:
1. 创建 `pkg/config` 统一配置加载包，包含 `GetString(key, fallback)`, `GetInt(key, fallback)`, `GetDuration(key, fallback)` 等
2. 将所有 `os.Getenv` 调用替换为统一函数
3. 在各服务 `config.FromEnv()` 中集中定义所有环境变量

---

### C3. DB配置结构体重复 — 5种不同定义  [P0]

**文件/行号**:

| 服务 | 文件 | 结构体名 | 字段 |
|------|------|----------|------|
| auth | `conf.go:33-37` | `DatabaseConfig` | URL, MaxOpenConns, MaxIdleConns |
| identity | `conf.go:25-31` | `DBConfig` | URL, MaxConns, MinConns, MaxConnLifetime, MaxConnIdleTime |
| oauth | `conf.go:31-37` | `DBConfig` | URL, MaxConns, MinConns, MaxConnLifetime, MaxConnIdleTime |
| policy | `data/db.go:13` | `Config` | Host, Port, User, Password, Database, SSLMode, MaxConns, MinConns, MaxConnLifetime |
| org | `data/db.go:13` | `Config` | 同 policy |
| audit | `data/db.go:13` | `Config` | 同 policy |

**问题**: 存在**5种不同的数据库配置结构体**：
1. auth 用 `URL + MaxOpenConns/MaxIdleConns`（字段名不一致：Open vs 无前缀）
2. identity/oauth 用 `URL + MaxConns/MinConns`（与auth字段名不同）
3. policy/org/audit 用 `Host/Port/User/Password/Database/SSLMode` 拆分字段（与URL模式不兼容）
4. auth 缺少 `MaxConnLifetime/MaxConnIdleTime`
5. 各服务在 cmd/main.go 中手动设置 pool 参数（auth: MaxConns=20/MinConns=5, identity: MaxConns=20/MinConns=2 — 不一致）

**风险**: 高 — 连接池参数不一致可能导致某些服务连接耗尽。配置模式不统一增加运维复杂度。

**建议修复**: 创建 `pkg/db.Config` 统一结构体：
```go
type Config struct {
    URL             string        // 完整连接字符串
    Host            string        // 或拆分字段（二选一）
    Port            int
    User            string
    Password        string
    Database        string
    SSLMode         string
    MaxConns        int32         // 统一字段名
    MinConns        int32
    MaxConnLifetime time.Duration
    MaxConnIdleTime time.Duration
}
```

---

### C4. Redis 配置结构体重复 — 3种定义  [P1]

**文件/行号**:
- `auth/conf.go:39-43` — `RedisConfig{Addr, Password, DB}`
- gateway `config.go:107-121` — 无独立结构体，直接在 main.go 中 `redis.NewClient` 内联
- `deploy/docker-compose.yaml:167` — `REDIS_ADDR` + 无密码字段（dev）

**问题**: auth 有独立 RedisConfig，gateway 在 main.go 内联创建 Redis client，audit/oAuth 通过不同路径获取 Redis 连接。无统一的 Redis 配置结构体。

**风险**: 中 — 连接参数（PoolSize、Timeouts）在 gateway 和 auth 中分别硬编码且值相同（PoolSize=20, MinIdleConns=5），但修改需要多处同步。

---

### C5. 配置验证 — 启动 fail-fast 覆盖不完整  [P0]

**已检查的文件和验证状态**:

| 服务 | 文件 | PASSWORD_PEPPER | HashChainSecret | DB ping | Redis ping | JWT验证 |
|------|------|----------------|----------------|---------|-----------|--------|
| gateway | `cmd/main.go:33-43` | ✅ Fatal if prod | N/A | N/A (no DB) | ⚠️ Warn-only | ✅ Fatal |
| identity | `cmd/main.go:19-29` | ✅ Fatal if prod | N/A | ✅ (server.New) | N/A | N/A |
| auth | `cmd/main.go:64+` | ✅ Fatal if prod | N/A | ✅ Fatal:93 | ✅ Fatal:113 | ✅ |
| oauth | `cmd/main.go` | ⚠️ 待确认 | N/A | ⚠️ 待确认 | ⚠️ 待确认 | ✅ |
| policy | `cmd/main.go:57-68` | ❌ 无验证 | N/A | ✅ Fatal:65 | N/A | N/A |
| org | `cmd/main.go:57-65` | ❌ 无验证 | N/A | ✅ Fatal:62 | N/A | N/A |
| audit | `cmd/main.go:68-92` | ❌ 无验证 | ✅ Fatal:91 | ✅ Fatal:75 | N/A | N/A |

**问题**:
1. **policy/org 缺少 PASSWORD_PEPPER 验证** — 虽然这两个服务可能不直接验证密码，但如果任何代码路径调用 crypto 验证，缺失 pepper 会导致不一致
2. **audit 缺少 PASSWORD_PEPPER 检查** — 同理
3. **gateway Redis 连接是 warn-only**（`main.go:124`）— Redis 不可用时 gateway 使用空 store，但 JTI blocklist 和 RBAC resolver 可能静默失败
4. **NATS 连接全部为 best-effort**（policy/org/audit）— 审计事件可能丢失且无告警

**风险**: 高 — 生产环境中如果关键配置缺失，服务可能以降级模式运行而不报错，违反 fail-fast 原则。

---

### C6. 敏感配置硬编码/日志泄露  [P0]

**文件/行号**:

| 位置 | 问题 | 严重度 |
|------|------|--------|
| `deploy/docker-compose.yaml:8` | `POSTGRES_PASSWORD` 硬编码弱密码 | P0 |
| `deploy/docker-compose.yaml:51` | `LDAP_ADMIN_PASSWORD` 硬编码弱密码 | P0 |
| `deploy/docker-compose.yaml:75-76` | LDAP密码出现在命令行参数 | P0 |
| `deploy/docker-compose.yaml:142` | `DATABASE_URL` 明文包含用户名密码 | P0 |
| `deploy/docker-compose.yaml:326` | OAuth同样明文DB连接串含密码 | P0 |
| `deploy/docker-compose.yaml:173-174` | `LDAP_BIND_PASSWORD` 默认弱密码 | P1 |
| `auth/conf.go:90` | Default DB URL含明文密码 | P1 |
| `oauth/conf.go:48` | 同上 Default DB URL含明文密码 | P1 |
| `identity/cmd/main.go:40` | fallback DB URL含明文密码 | P1 |
| `gateway/config.go:109` | Redis fallback `localhost:6379` 无密码 | P1 |

**日志泄露检查**:
- `auth/cmd/main.go:96` — `log.Println("connected to PostgreSQL")` ✅ 不泄露URL
- `auth/cmd/main.go:115` — `log.Println("connected to Redis")` ✅ 不泄露地址
- `gateway/cmd/main.go:124` — `log.Printf("Warning: Redis not available... %v", err)` ✅ 不泄露密码
- **注意**: `DATABASE_URL` 在日志中未直接打印，但 error message 可能包含连接字符串（如 `pool.Ping(ctx)` 失败时 pgx 错误可能包含完整 URL）

**风险**: 极高 — `docker-compose.yaml` 中的明文密码在版本控制中暴露。虽然 `.prod.yaml` 使用了 `${VAR:?}` 强制环境变量，但 dev compose 文件中的默认密码可能被误用于生产。

**建议修复**:
1. `docker-compose.yaml` 也应使用 `${VAR:-default}` 模式，将默认值限制为明显的开发标识（如 `changeme-dev-only`）
2. 移除所有代码中的默认明文密码，改为空字符串 + fail-fast
3. 确保 pgx 错误日志不暴露连接字符串（使用 `%w` 包装时注意）

---

### C7. envOrDefault/getEnv 重复实现 — 4+ 个变体  [P1]

**文件/行号**:

| 变体 | 位置 | 签名 |
|------|------|------|
| `envOrDefault` | `gateway/config.go:165` | `func(key, fallback string) string` |
| `getEnv` | `policy/config.go:39` | `func(key, def string) string` — wraps httputil.GetEnv |
| `getEnv` | `audit/config.go:48` | 同上 |
| `getEnv` | `org/config.go` | 同上 |
| `GetEnv` | `pkg/httputil/` | 公开版本 |
| `GetEnvInt` | `pkg/httputil/` | 公开版本 |
| `parseIntDefault` | `auth/conf.go:175` | 本地 int 解析 — 不复用 httputil.GetEnvInt |

**问题**: 
1. gateway 有自己的 `envOrDefault`，不使用 `httputil.GetEnv`
2. policy/audit/org 的 `getEnv` 是 `httputil.GetEnv` 的 thin wrapper（多一层间接无价值）
3. auth 的 `parseIntDefault` 不复用 `httputil.GetEnvInt`
4. 存在两种命名风格：`envOrDefault` vs `getEnv` vs `GetEnv`

**风险**: 中 — 不直接导致bug，但增加维护成本和新开发者困惑。

---

### C8. 配置热更新覆盖  [P1]

**文件/行号**:
- `gateway/cmd/main.go:122-129` — sysconfig Store 从 Redis 热加载
- `pkg/sysconfig/` — Store 接口支持热更新

**问题**:
1. 只有 gateway 实现了 sysconfig 热更新，其他服务（auth/identity等）不感知 sysconfig 变更
2. gateway 的 sysconfig Store 在 Redis 不可用时降级为 `nil` store（仅默认值），但不会在 Redis 恢复后自动重连
3. 热更新范围不明确 — 哪些配置可以热更新、哪些需要重启，缺乏文档

**风险**: 中 — 配置不一致可能导致 gateway 使用与后端服务不同的配置（如密码策略、session超时等）。

---

### C9. 环境分离  [P1]

**文件/行号**:
- `GGID_ENV` 环境变量用于区分 dev/test/prod
- `gateway/cmd/main.go:37-42` — 检查 `GGID_ENV != "test" && != "dev"`
- `auth/cmd/main.go` — 类似检查
- `identity/cmd/main.go:23-28` — 类似检查
- `audit/cmd/main.go:87-92` — 类似检查

**问题**:
1. `GGID_ENV` 只在少数地方检查，大多数配置加载不区分环境
2. 没有 `staging` 环境支持 — 只有 `dev`/`test`/`prod`（隐含）
3. dev/test 以外的所有值都被视为 production，但无明确枚举验证
4. `docker-compose.yaml` 不设置 `GGID_ENV`，`docker-compose.prod.yaml` 也不设置
5. policy/org 服务完全没有 `GGID_ENV` 检查

**风险**: 中 — 如果误配（如设置 `GGID_ENV=production` 而非 `prod`），fail-fast 检查会意外触发或被跳过。

**建议修复**: 
1. 定义 `pkg/env.Environment` 枚举类型
2. 在所有服务启动时统一验证环境值
3. docker-compose 文件显式设置 `GGID_ENV`

---

## 审查角度2：部署可靠性

### D1. Dockerfile — 非root/多阶段/:latest 评估

**逐文件评估**:

| Dockerfile | 多阶段 | 非root | HEALTHCHECK | :latest tag | CGO_ENABLED=0 |
|------------|--------|--------|-------------|-------------|---------------|
| `console/Dockerfile` | ✅ 3阶段 | ✅ appuser:1001 | ❌ 无 | ✅ 避免 | N/A |
| `services/identity/Dockerfile` | ✅ | ✅ appuser:1001 | ❌ 无 | ✅ | ✅ |
| `services/auth/Dockerfile` | ✅ | ✅ appuser:1001 | ❌ 无 | ✅ | ✅ |
| `services/gateway/Dockerfile` | ✅ | ✅ appuser:1001 | ❌ 无 | ✅ | ✅ |
| `services/oauth/Dockerfile` | ✅ | ✅ app:1001(S) | ❌ 无 | ✅ | ✅ |
| `services/policy/Dockerfile` | ✅ | ✅ app:1001(S) | ❌ 无 | ✅ | ✅ |
| `services/org/Dockerfile` | ✅ | ✅ app:1001(S) | ❌ 无 | ✅ | ✅ |
| `services/audit/Dockerfile` | ✅ | ✅ app:1001(S) | ❌ 无 | ✅ | ✅ |
| `deploy/docker/Dockerfile.service` | ❌ 单阶段 | ✅ appuser:1001 | ✅ :38-39 | ⚠️ 示例用:latest | ✅ |

**问题**:

1. **[P0] 7个服务 Dockerfile 无 HEALTHCHECK 指令** — 虽然 docker-compose.yaml 中有 healthcheck，但 K8s 部署或独立 docker run 时无内置健康检查
2. **[P1] 用户创建不一致** — identity/auth/gateway 使用 `adduser -D -u 1001 appuser`，而 oauth/policy/org/audit 使用 `addgroup -S app && adduser -S app -G app`（系统用户无明确UID）
3. **[P1] oauth/policy/org/audit 缺少 `tzdata` 包** — identity/auth/gateway 也没有（仅 oauth 有 `tzdata`），时间戳相关操作可能使用UTC而非本地时区
4. **[P1] Dockerfile.service 使用 :latest tag**（`deploy/docker/Dockerfile.service:10` 注释示例）— 虽然是注释，但引导用户使用不可变tag
5. **[P2] 各服务 Dockerfile 完全相同（仅服务名/端口不同）** — 7个几乎相同的 Dockerfile 应该用模板统一

**建议修复**:
```dockerfile
# 在每个服务 Dockerfile 的 final stage 添加：
HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=10s \
  CMD wget -qO- http://localhost:${PORT}/healthz || exit 1
```

---

### D2. K8s Manifest — Resource Limits / Probes 评估

**检查文件**: `deploy/k8s/mcp-deployment.yaml`, `console-deployment.yaml`, `db-migrate-job.yaml`

| Manifest | Resources | Liveness | Readiness | SecurityContext |
|----------|-----------|----------|-----------|-----------------|
| mcp-deployment | ❌ 无 limits/requests | ✅ :52-57 | ✅ :58-63 | ✅ runAsNonRoot |
| console-deployment | ✅ :62-68 | ✅ :69-75 | ✅ :76-82 | ✅ :42-46 |
| db-migrate-job | N/A (Job) | N/A | N/A | ❌ 无 |

**问题**:

1. **[P0] mcp-deployment 无 resource limits**（`k8s/mcp-deployment.yaml:26-63`）— MCP 服务可能消耗过多内存导致节点不稳定
2. **[P1] mcp readOnlyRootFilesystem: false**（`:33`）— 应为 true（console也是同样问题 `:46`）
3. **[P1] 缺少主服务（gateway/identity/auth等）的 K8s manifest** — k8s/ 目录只有 mcp/console/db-migrate，核心微服务的 K8s 部署依赖 Helm chart
4. **[P1] db-migrate-job 无 securityContext**（`k8s/db-migrate-job.yaml:12-31`）— 容器以 root 运行
5. **[P2] db-migrate-job 无 resource limits** — Job 可能被 OOM kill 且无重试限制
6. **[P2] console livenessProbe 使用 path: /** — 应使用更精确的 `/api/health` 或类似端点，`/` 返回200可能不反映真实健康状态

---

### D3. Migration — 版本冲突 / Tracking 表  [P0]

**文件**: `deploy/migrations/`（53个SQL文件）

**编号冲突检测**:
```
044_encrypt_totp_secrets.sql
044_refresh_token_families.sql   ← 冲突！
046_alert_rules_db.sql
046_tenant_idp_configs.sql       ← 冲突！
047_api_keys.sql
047_zt_posture_history.sql       ← 冲突！
```

**3组编号冲突**，确认存在 migration 版本号重复。

**Tracking 表问题**:
- `deploy/k8s/db-migrate-job.yaml:27-30` — 逐文件 `psql -f` 执行，**无 migration tracking 表**
- `deploy/docker-compose.yaml:118-124` — 检查 `information_schema.tables` 行数，如果>0就跳过**所有** migration
- 两者都**不使用专业的 migration 工具**（如 golang-migrate, goose, buf migration）

**问题**:
1. **[P0] 3组编号冲突** — 同一编号的 migration 只会执行一个（按文件名字母排序），另一个被跳过
2. **[P0] 无 migration tracking 表** — compose 的 "tables count > 0 → skip all" 逻辑意味着：
   - 首次部署后，新 migration 永远不会执行
   - 如果手动创建了任何表，所有 migration 被跳过
3. **[P1] 无回滚 migration** — 只有 forward SQL，无 down/rollback
4. **[P1] `cat /migrations/*.sql | psql` 模式**（compose:122）— 将所有SQL合并为一个事务，一个失败全部回滚（可能是好事也可能是坏事取决于场景）

**风险**: 极高 — 生产环境中数据库 schema 可能与代码期望不一致，导致运行时错误。

**建议修复**:
1. 立即重命名冲突 migration（044→054, 046→055, 047→056）
2. 引入 `golang-migrate` 或 `goose` 实现 versioned migration + tracking 表
3. 废弃 `cat *.sql | psql` 模式

---

### D4. 健康检查 — /healthz /readyz 检查 DB  [P0]

**逐服务检查**:

| 服务 | /healthz | /readyz | DB Ping in /readyz | 文件位置 |
|------|----------|---------|---------------------|----------|
| gateway | ✅ (router) | ❌ 无 | N/A | `gateway/internal/router/` |
| identity | ✅ | ❌ 无 | ❌ | `identity/internal/server/http.go` |
| auth | ✅ | ❌ 无 | ❌ | `auth/internal/server/http.go` |
| oauth | ✅ | ❌ 无 | ❌ | `oauth/internal/server/http.go` |
| policy | ✅ :121-124 | ✅ :125-128 | ❌ 只返回 "ready" | `policy/cmd/main.go` |
| org | ✅ :126-129 | ✅ :130-133 | ❌ 只返回 "ready" | `org/cmd/main.go` |
| audit | ✅ :166-169 | ✅ :170-182 | ✅ db.Ping(ctx) | `audit/cmd/main.go` |
| mcp | ✅ | ✅ | N/A | `k8s/mcp-deployment.yaml:52-63` |

**问题**:
1. **[P0] 只有 audit 的 /readyz 真正检查 DB**（`audit/cmd/main.go:171-178`）— 其他服务的 /readyz 返回固定 "ready"，不反映依赖健康
2. **[P0] policy/org 的 /readyz 不检查 DB**（`policy/cmd/main.go:125-128`, `org/cmd/main.go:130-133`）— DB 宕机时 readiness probe 仍返回 200
3. **[P1] gateway/identity/auth/oauth 无 /readyz** — K8s readinessProbe 无法区分服务是否准备好接受流量
4. **[P1] gateway /healthz 不检查 Redis** — gateway 依赖 Redis（JTI blocklist, RBAC resolver），但 Redis 宕机时 /healthz 仍返回 200

**风险**: 高 — K8s/Docker 会将流量路由到不健康的实例。

**建议修复**: 所有服务的 /readyz 应检查关键依赖（DB, Redis, NATS）：
```go
mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
    defer cancel()
    if err := db.Ping(ctx); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte("not ready"))
        return
    }
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("ready"))
})
```

---

### D5. Graceful Shutdown  [P1]

**逐服务检查**:

| 服务 | SIGINT/SIGTERM | HTTP Shutdown | gRPC GracefulStop | Background ctx cancel | 超时 |
|------|----------------|---------------|--------------------|-----------------------|------|
| gateway | ✅ :179-180 | ✅ :183+190 | N/A | ✅ :195 | ✅ 30s :187 |
| identity | ✅ :57 | ✅ (server.Run) | ✅ (server.Run) | ✅ | ⚠️ 待确认 |
| auth | ⚠️ 待确认 | ⚠️ 待确认 | N/A | ⚠️ | ⚠️ |
| oauth | ⚠️ 待确认 | ⚠️ 待确认 | N/A | ⚠️ | ⚠️ |
| policy | ✅ (隐含) | ✅ shutdown.New | ✅ 待确认 | ✅ | ⚠️ 待确认 |
| org | ✅ :166-168 | ✅ :171 | ✅ :174 | ⚠️ 无显式ctx cancel | ❌ 无超时 |
| audit | ✅ | ✅ | ✅ | ✅ | ⚠️ 待确认 |

**问题**:
1. **[P1] org 缺少 shutdown 超时**（`org/cmd/main.go:170-175`）— `shutdown.New()` 后直接 GracefulStop，无最大等待时间
2. **[P1] gateway 双重 shutdown**（`gateway/cmd/main.go:183,190`）— 先调用 `shutdown.New(...).Execute()` 再调用 `srv.Shutdown()`，可能重复关闭
3. **[P2] auth main.go 截断无法完整评估** — 从可见代码看，auth 使用 `signal.NotifyContext` 模式，但 shutdown 逻辑在截断部分

---

### D6. 连接管理 — Fail-Fast 评估  [P0]

**逐服务检查**:

| 服务 | DB 连接 | Fail-Fast | Redis | Fail-Fast | NATS | Fail-Fast |
|------|---------|-----------|-------|-----------|------|-----------|
| gateway | N/A (no direct DB) | N/A | ✅ :111-121 | ⚠️ Warn-only:124 | N/A | N/A |
| identity | ✅ server.New | ✅ Fatal | N/A | N/A | N/A | N/A |
| auth | ✅ :87-91 | ✅ Fatal:89 | ✅ :99-111 | ✅ Fatal:113 | ✅ best-effort | ⚠️ Warn:89 |
| oauth | ⚠️ | ⚠️ | ⚠️ | ⚠️ | N/A | N/A |
| policy | ✅ :63 | ✅ Fatal:65 | N/A | N/A | ✅ :88 | ⚠️ Warn:89 |
| org | ✅ :61 | ✅ Fatal:63 | N/A | N/A | ✅ :90 | ⚠️ Warn:91 |
| audit | ✅ :72 | ✅ Fatal:75 | ✅ (ITDR) :119 | ⚠️ 无检查 | ✅ :108 | ✅ Fatal:110 |

**问题**:
1. **[P0] gateway Redis 连接不 fail-fast**（`gateway/cmd/main.go:123-129`）— Redis 不可用时 gateway 以降级模式运行（无 sysconfig, JTI blocklist 不可用），可能允许已撤销的token通过
2. **[P0] audit NATS consumer 启动 fail-fast 但 publisher 不fail-fast** — audit 服务的 consumer 启动失败是 Fatal（`:110`），但 NATS publisher 在其他服务中是 best-effort
3. **[P1] Redis 连接参数硬编码不一致** — gateway/main.go 和 auth/main.go 中的 Redis pool 参数虽然值相同（PoolSize=20, MinIdleConns=5），但分散在两个文件中
4. **[P1] audit Redis 连接（ITDR）无 ping 检查**（`audit/cmd/main.go:119-126`）— 创建 Redis client 后不验证连接

---

### D7. TLS / 证书  [P1]

**检查路径**:
- gRPC TLS: `policy/cmd/main.go:31-51`, `org/cmd/main.go:31-51`, `audit/cmd/main.go:41-61`
- Redis TLS: `auth/cmd/main.go:49-62`, `gateway/cmd/main.go:120`
- HTTP TLS: 无（所有服务 HTTP 都是明文）
- `deploy/helm/ggid/values-prod.yaml` — 通过 nginx ingress 做 TLS 终止

**问题**:
1. **[P1] 所有 HTTP 服务无 TLS** — gateway, identity, auth, oauth, policy, org, audit 的 HTTP 端口都是明文。虽然 K8s/Helm 通过 ingress 做了 TLS 终止，但 docker-compose 部署中流量是明文的
2. **[P1] gRPC TLS 是可选的**（`GRPC_TLS_ENABLED` 默认不设）— 内部 gRPC 流量默认明文
3. **[P2] Redis TLS skipVerify 在生产中被拒绝**（`auth/cmd/main.go:54-57`）— ✅ 已有防护
4. **[P2] docker-compose.prod.yaml 中 PostgreSQL 连接 `sslmode=disable`**（`:140, 173, 212`）— 生产环境应使用 `sslmode=require` 或 `verify-full`

---

### D8. Helm Chart 安全默认值  [P1]

**文件**: `deploy/helm/ggid/values.yaml`, `values-prod.yaml`

**检查结果**:

| 安全项 | values.yaml | values-prod.yaml | 评估 |
|--------|-------------|------------------|------|
| PodSecurityContext | ❌ 未定义 | ❌ 未定义 | P0 |
| runAsNonRoot | ❌ 未定义 | ❌ 未定义 | P0 |
| readOnlyRootFilesystem | ❌ 未定义 | ❌ 未定义 | P1 |
| NetworkPolicy | ✅ :93 enabled | ✅ :154-155 enabled | ✅ |
| PDB | ✅ :89 enabled | ✅ :157-159 | ✅ |
| Resource limits | ✅ 所有服务 | ✅ 所有服务（加大） | ✅ |
| Image pullPolicy | ✅ IfNotPresent | ✅ IfNotPresent | ✅ |
| ServiceAccount | ❌ 未定义 | ❌ 未定义 | P1 |
| Secret management | ⚠️ 空字符串 | ⚠️ 空字符串+注释 | P1 |
| Probes | ❌ 未在values定义 | ❌ 未在values定义 | P1 |

**问题**:
1. **[P0] Helm chart 无 PodSecurityContext 定义** — `values.yaml` 和 `values-prod.yaml` 都没有 `podSecurityContext` 配置，如果 deployment 模板不强制设置，容器可能以 root 运行
2. **[P1] 密钥默认为空字符串** — `postgresql.auth.password: ""`, `redis.auth.password: ""`, `config.passwordPepper: ""` — 应该使用 `required` 函数或提供生成命令
3. **[P1] 无 ServiceAccount 配置** — 应定义专用 SA 而非使用 default
4. **[P2] Helm chart templates 目录未检查** — 只看了 values，需要确认 templates/ 中 deployment.yaml 是否包含 securityContext

---

### D9. MCP 健康端点  [P1]

**文件**: `deploy/k8s/mcp-deployment.yaml:52-63`

**问题**:
1. **[P1] MCP /healthz 和 /readyz 是否真实检查依赖未知** — K8s manifest 配置了 liveness/readiness probe 指向 `/healthz` 和 `/readyz`，但 MCP 服务的实际实现需要检查 `services/mcp/` 代码
2. **[P1] MCP 无 resource limits**（见 D2）

---

### D10. Docker HEALTHCHECK  [P0]

**检查结果**:

| Dockerfile | HEALTHCHECK | docker-compose healthcheck |
|------------|-------------|---------------------------|
| console/Dockerfile | ❌ 无 | ❌ 无 |
| 7个服务 Dockerfile | ❌ 无 | ✅ 6个有（identity/auth/policy/org/audit/oauth）|
| deploy/docker/Dockerfile.service | ✅ :38-39 | N/A |

**问题**:
1. **[P0] 7个服务 Dockerfile 无 HEALTHCHECK** — 只有 `Dockerfile.service` 有，但这是预编译版本的模板
2. **[P0] console 无 healthcheck** — Dockerfile 和 docker-compose.yaml 都没有 console 的 healthcheck
3. **[P1] gateway 在 docker-compose.yaml 中无 healthcheck**（`:187-219`）— gateway 被 console depends_on，但只检查 `service_started` 而非 `service_healthy`
4. **[P1] keygen/migrate one-shot 服务无 healthcheck** — 合理（one-shot 不需要），但应确保 depends_on 正确

**注意**: `docker-compose.prod.yaml` 中 gateway 有 healthcheck（隐含通过其他服务的 health check），但自身无 healthcheck 定义（:421-471 无 healthcheck 字段）。

---

## 发现汇总

### 按严重度

| 级别 | 数量 | 关键发现 |
|------|------|----------|
| **P0** | 10 | migration编号冲突(3组) | DB配置结构体5种重复 | HEALTHCHECK缺失 | /readyz不查DB | 敏感配置硬编码 | 无migration tracking表 | gateway Redis不fail-fast | Helm无securityContext |
| **P1** | 14 | conf vs config分裂 | os.Getenv散用259处 | envOrDefault重复4+种 | 环境分离不完整 | Dockerfile不一致 | K8s缺resource limits | HTTP无TLS | Graceful shutdown超时缺失 | Helm密钥空默认值 |
| **P2** | 5 | Dockerfile重复 | DB连接参数分散 | 时间戳tzdata缺失 | SA未定义 | Console liveness路径 |

### 按审查角度

**配置管理**（8项）:
- C1: conf vs config 包分裂 [P0] — 3+3分裂，17轮未修复
- C2: os.Getenv 散用 259处/73文件 [P1]
- C3: DB配置结构体 5种重复 [P0]
- C4: Redis配置无统一结构体 [P1]
- C5: fail-fast 覆盖不完整 — policy/org/audit 缺 PASSWORD_PEPPER检查 [P0]
- C6: 敏感配置硬编码 — docker-compose.yaml 明文密码 [P0]
- C7: envOrDefault 4+种变体重复 [P1]
- C8: 热更新只有gateway，Redis断线不重连 [P1]
- C9: 环境分离不完整 — 无枚举，policy/org不检查 [P1]

**部署可靠性**（10项）:
- D1: 7个服务Dockerfile无HEALTHCHECK [P0]
- D2: mcp-deployment无resource limits [P0]
- D3: Migration编号3组冲突 + 无tracking表 [P0]
- D4: /readyz不检查DB — 仅audit真正检查 [P0]
- D5: org缺少shutdown超时，gateway双重shutdown [P1]
- D6: gateway Redis不fail-fast，允许已撤销token [P0]
- D7: HTTP无TLS，gRPC TLS可选，prod sslmode=disable [P1]
- D8: Helm chart无PodSecurityContext [P0]
- D9: MCP健康端点实现待确认 [P1]
- D10: Docker HEALTHCHECK全面缺失 [P0]

### 修复优先级（Top 5）

1. **[P0] Migration编号冲突** — 044/046/047重复，立即重命名
2. **[P0] /readyz检查DB** — 所有服务添加依赖检查
3. **[P0] 敏感配置移除硬编码** — docker-compose.yaml 密码改为环境变量
4. **[P0] Docker HEALTHCHECK** — 7个服务Dockerfile添加
5. **[P0] DB配置结构体统一** — 创建 pkg/db.Config

---

**审查统计**: R392配置管理第17轮+部署可靠性第17轮  
**P0**: 10项 | **P1**: 14项 | **P2**: 5项 | **总计**: 29项发现
