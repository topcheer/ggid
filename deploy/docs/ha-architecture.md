# GGID 多区域高可用架构设计

> 版本：1.0 | 日期：2026-07-23
> 状态：设计文档（实际部署需根据云厂商/区域/网络拓扑定制）

---

## 1. 架构概览

```
                    ┌─────────────────┐
                    │  Global DNS     │
                    │  (Route53/CF)   │
                    │  Health-based   │
                    │  failover       │
                    └────┬───────┬────┘
                         │       │
              ┌──────────┘       └──────────┐
              ↓                              ↓
    ┌──────────────────┐          ┌──────────────────┐
    │  Primary Region   │          │  Standby Region   │
    │  (ap-east-1)      │          │  (ap-northeast-1) │
    │                   │          │                   │
    │  ┌─────────────┐  │          │  ┌─────────────┐  │
    │  │  K8s Cluster │  │          │  │  K8s Cluster │  │
    │  │  (3+ nodes)  │  │          │  │  (3+ nodes)  │  │
    │  │              │  │          │  │              │  │
    │  │  GGID Svc    │  │          │  │  GGID Svc    │  │
    │  │  (active)    │  │          │  │  (warm)      │  │
    │  └──────┬───────┘  │          │  └──────┬───────┘  │
    │         │          │          │         │          │
    │  ┌──────┴───────┐  │          │  ┌──────┴───────┐  │
    │  │ PG Primary   │◄─┼──async──┼──┤ PG Replica   │  │
    │  │ (read/write) │  │  stream  │  │ (read-only) │  │
    │  └──────────────┘  │          │  └──────────────┘  │
    │                     │          │                     │
    │  ┌──────────────┐  │          │  ┌──────────────┐  │
    │  │ Redis Primary│◄─┼──replicate┼─►│Redis Replica │  │
    │  │              │  │          │  │              │  │
    │  └──────────────┘  │          │  └──────────────┘  │
    └──────────────────┘          └──────────────────┘
```

## 2. 组件级 HA 策略

### 2.1 PostgreSQL

| 项 | 主区域 | 备区域 |
|----|--------|--------|
| 角色 | Primary (read/write) | Hot Standby (read-only) |
| 复制 | 异步流复制 (physical replication slot) | — |
| Promote | 手动 / pg_auto_failover | `pg_ctl promote` |
| RPO | < 1s（异步复制延迟） | — |
| 连接 | Pooler (PgBouncer) → Primary | Pooler → Standby |

**values-prod.yaml 配置**:
```yaml
postgresql:
  enabled: false  # 使用外部 PG

externalDatabase:
  host: "pg-primary.ap-east-1.internal"
  port: 5432
  username: "ggid"
  password: "${PG_PASSWORD}"
  database: "ggid"
  replicaHost: "pg-replica.ap-northeast-1.internal"  # 读副本
  maxConns: 100
```

**复制设置**:
```ini
# postgresql.conf (primary)
wal_level = replica
max_wal_senders = 10
max_replication_slots = 10
synchronous_commit = on  # 跨区域可用 remote_apply

# postgresql.conf (standby)
hot_standby = on
hot_standby_feedback = on
```

**故障转移流程**:
1. 检测主 PG 不可用（pg_isready timeout 30s）
2. 等待 WAL replay 完成（`pg_wal_replay_wait`）
3. `pg_ctl promote` 提升备库
4. 更新 DNS / 连接池指向新主库
5. 原主恢复后作为新备库重新加入

### 2.2 Redis

| 项 | 主区域 | 备区域 |
|----|--------|--------|
| 模式 | Sentinel 或 Cluster | — |
| 复制 | async replica | — |
| Promote | Sentinel 自动 failover | 手动 |

**values-prod.yaml 配置**:
```yaml
redis:
  enabled: false  # 使用外部 Redis

externalRedis:
  host: "redis-primary.ap-east-1.internal"
  port: 6379
  password: "${REDIS_PASSWORD}"
  sentinel:
    enabled: true
    master: "ggid"
    nodes: "sentinel-0:26379,sentinel-1:26379,sentinel-2:26379"
```

**注意**: GGID 的 revokedTokens/stateStore 已迁移到 Redis（S5 修复）。Sentinel failover 时有 < 5s 窗口不可用，服务层已有内存 fallback。

### 2.3 NATS

NATS 用于审计事件传递。跨区域使用 NATS Leafnode 或 JetStream 跨集群：

```yaml
externalNats:
  url: "nats://nats.ap-east-1.internal:4222"
  leafnode:
    remotes:
      - url: "nats://nats.ap-northeast-1.internal:7422"
```

### 2.4 应用服务

所有 GGID 微服务已支持多副本（values-prod.yaml 配置 3+ replicas + HPA）。跨区域部署：

| 项 | 主区域 | 备区域 |
|----|--------|--------|
| 状态 | Active（承接流量） | Warm（部署就绪，不接流量） |
| 数据 | 读写 PG Primary | 可读 PG Standby |
| 切换 | DNS failover 到备区域 | — |

## 3. DNS 故障转移

### Route53 配置
```
ggid.iot2.win → CNAME → primary-alb.ap-east-1.elb.amazonaws.com
  (health check: GET /healthz, interval 10s, timeout 5s)
  failover: PRIMARY

ggid.iot2.win → CNAME → standby-alb.ap-northeast-1.elb.amazonaws.com
  failover: SECONDARY
```

### Cloudflare 配置
```
ggid.iot2.win → Load Balancer Pool
  origin-1: primary-k8s (health: GET /healthz)
  origin-2: standby-k8s (health: GET /healthz)
  policy: fallback (primary → standby on failure)
```

## 4. 故障转移 Runbook

### 自动故障转移（DNS 层）
1. Route53/CF 健康检查失败（连续 3 次，30s）
2. DNS 自动指向备区域 ALB
3. 备区域开始承接流量（TTL 60s 内全球生效）

### 手动 PG Promote（数据层）
```bash
# 1. 确认主库不可用
pg_isready -h pg-primary.ap-east-1.internal

# 2. 等待备库 WAL replay
psql -h pg-replica.ap-northeast-1.internal -c "SELECT pg_last_wal_replay_lsn();"

# 3. 提升备库
pg_ctl promote -D $PGDATA

# 4. 更新 Helm values 指向新主库
helm upgrade ggid deploy/helm/ggid/ -f values-prod.yaml \
  --set externalDatabase.host=pg-replica.ap-northeast-1.internal

# 5. 验证
kubectl rollout restart deployment/ggid-identity deployment/ggid-auth deployment/ggid-oauth -n ggid
```

### 故障恢复（Reverse）
1. 修复原主区域
2. 将原主设置为备（pg_rewind + 重新建立复制）
3. 切换 DNS 回主区域
4. 恢复原始复制拓扑

## 5. values-ha.yaml 多区域配置层

```yaml
# 多区域高可用配置
# Usage: helm install ggid deploy/helm/ggid/ -f values-prod.yaml -f values-ha.yaml

# 外部数据库（主备 PG）
postgresql:
  enabled: false

externalDatabase:
  host: "pg-primary.ap-east-1.internal"
  replicaHost: "pg-replica.ap-northeast-1.internal"
  port: 5432

# 外部 Redis（Sentinel）
redis:
  enabled: false

externalRedis:
  sentinel:
    enabled: true
    master: "ggid"
    nodes: "sentinel-0:26379,sentinel-1:26379,sentinel-2:26379"

# Pod 反亲和（跨 node/zone 分散）
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchLabels:
              app.kubernetes.io/name: ggid
          topologyKey: kubernetes.io/hostname
      - weight: 50
        podAffinityTerm:
          labelSelector:
            matchLabels:
              app.kubernetes.io/name: ggid
          topologyKey: topology.kubernetes.io/zone

# 拓扑分布约束
topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: topology.kubernetes.io/zone
    whenUnsatisfiable: ScheduleAnyway
    labelSelector:
      matchLabels:
        app.kubernetes.io/name: ggid

# PDB 确保始终有足够副本
podDisruptionBudget:
  enabled: true
  minAvailable: 2
```

## 6. 验证清单

- [ ] PG 流复制延迟 < 1s（`pg_stat_replication`）
- [ ] Sentinel quorum ≥ 2
- [ ] DNS failover 测试：关停主区域 ALB → 30s 内备区域接管
- [ ] 应用只读模式测试：备区域 PG Standby 可接受读请求
- [ ] 审计链连续性：跨区域切换后 audit events 无丢失
- [ ] Redis token revocation 跨区域同步
