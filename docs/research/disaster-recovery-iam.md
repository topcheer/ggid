# Disaster Recovery for IAM Systems

> **Research Document** — GGID Identity and Access Management Suite
> Topic: Disaster Recovery (DR) strategies, RPO/RTO targets, backup procedures, failover architecture, and DR runbooks for production IAM deployments.

---

## Table of Contents

1. [RPO/RTO Targets for IAM](#1-rporto-targets-for-iam)
2. [PostgreSQL Backup & PITR](#2-postgresql-backup--pitr)
3. [NATS JetStream Backup](#3-nats-jetstream-backup)
4. [Redis Snapshot & Restore](#4-redis-snapshot--restore)
5. [Service Failover Architecture](#5-service-failover-architecture)
6. [Multi-Region Active-Active Design](#6-multi-region-active-active-design)
7. [DR Runbook Templates](#7-dr-runbook-templates)
8. [Backup Testing & DR Drills](#8-backup-testing--dr-drills)
9. [GGID DR Gap Analysis](#9-ggid-dr-gap-analysis)
10. [Implementation Roadmap](#10-implementation-roadmap)

---

## 1. RPO/RTO Targets for IAM

### Why IAM Needs Differentiated DR Targets

Identity and Access Management is a **dependency root** for every downstream service.
When the auth service is down, no user can log in — and every service that validates
JWTs or checks permissions is effectively degraded. However, not every IAM subsystem
has the same urgency:

- **Authentication (login, token issuance)**: Must be available within minutes. A
  prolonged auth outage locks out all users, including administrators who need to
  investigate the failure itself.
- **Audit logging**: Can tolerate hours of delay. Audit events can be buffered in
  NATS JetStream and persisted when the database recovers. Regulatory requirements
  (SOX, GDPR) mandate eventual completeness, not real-time availability.
- **Policy/Org management**: Administrative changes (creating roles, updating org
  structures) are low-frequency operations. A delay of 30-60 minutes is acceptable.

### RPO/RTO Definition Table

| Service | RPO Target | RTO Target | Tier | Justification |
|---------|-----------|-----------|------|---------------|
| **Gateway** | 0 (stateless) | 2 min | Tier 0 | Stateless reverse proxy. Restart from image. Single entry point for all traffic. |
| **Auth** | 0 (Redis-backed) | 5 min | Tier 0 | Login/token issuance. Without auth, all user access fails. Redis session cache + DB is source of truth. |
| **OAuth** | 0 (Redis-backed) | 5 min | Tier 0 | OAuth/OIDC flows are auth-critical. Token refresh and consent depend on this service. |
| **Identity** | 5 min (Postgres WAL) | 10 min | Tier 1 | User CRUD. New user registration can queue; existing JWTs remain valid. |
| **Policy** | 5 min (Postgres WAL) | 15 min | Tier 1 | RBAC/ABAC evaluation. Existing policies are cached in JWT claims; new policy changes can wait. |
| **Org** | 5 min (Postgres WAL) | 15 min | Tier 1 | Organization management. Rarely changed in real-time; admin-only operations. |
| **Audit** | 15 min (NATS buffer) | 30 min | Tier 2 | Audit events buffered in JetStream. Can replay after recovery. Compliance allows eventual consistency. |

### RPO/RTO Decision Framework

```
                       RTO (Recovery Time Objective)
                      <5min          <15min         <60min
                 ┌──────────────┬──────────────┬──────────────┐
   RPO 0  (none) │ Gateway      │              │              │
   (Recovery     │ Auth, OAuth  │              │              │
    Point        │              │              │              │
   Objective)    ├──────────────┼──────────────┼──────────────┤
   RPO <5min     │              │ Identity     │              │
                 │              │ Policy       │              │
                 │              │ Org          │              │
                 ├──────────────┼──────────────┼──────────────┤
   RPO <15min    │              │              │ Audit        │
                 └──────────────┴──────────────┴──────────────┘
```

**Key principle**: IAM services should be designed so that JWTs issued before a failure
remain valid throughout the outage. With a standard 15-minute access token lifetime,
a Tier 0 RTO of 5 minutes ensures most users experience zero disruption. Refresh
tokens (7-30 day lifetime) bridge the gap for long sessions.

---

## 2. PostgreSQL Backup & PITR

### Current State in GGID

GGID uses PostgreSQL 16 as the primary data store. The `docker-compose.yaml` mounts
a Docker volume (`ggid-pgdata`) for persistence, but **no backup or WAL archiving is
configured**. The `docker-compose.prod.yaml` adds `appendonly yes` for Redis but does
not configure PostgreSQL WAL archiving.

### Point-in-Time Recovery (PITR) Architecture

PITR allows recovery to any point in time between a base backup and the current WAL
archive. This is critical for:

- Recovering from accidental `DROP TABLE` or destructive migration
- Rolling back a bad data modification
- Forensic analysis of a security breach

#### WAL Archiving Configuration

```ini
# postgresql.conf — WAL archiving for PITR
wal_level = replica
archive_mode = on
archive_command = 'wal-g wal-push %p'
archive_timeout = '60s'        # force WAL segment rotation every 60s
max_wal_senders = 5            # for streaming replication
wal_keep_size = '1GB'          # retain WAL for replica catch-up
```

#### Base Backup with WAL-G

WAL-G is the recommended tool (superset of WAL-E with better compression and parallel
processing):

```bash
#!/bin/bash
# backup-postgres.sh — Daily base backup + continuous WAL archiving

export WALG_S3_PREFIX="s3://ggid-backups/postgres"
export AWS_REGION="us-east-1"
export WALG_COMPRESSION_METHOD=lz4
export WALG_UPLOAD_CONCURRENCY=16

# 1. Verify WAL archiving is working
WAL_COUNT=$(psql -c "SELECT count(*) FROM pg_stat_archiver WHERE failed = false" -t)
if [ "$WAL_COUNT" -eq 0 ]; then
  echo "FATAL: WAL archiving is not functioning"
  exit 1
fi

# 2. Create base backup
wal-g backup-push /var/lib/postgresql/data

# 3. Delete old base backups (retain 7 days, minimum 3 backups)
wal-g delete retain 7d 3 --confirm

# 4. Verify backup integrity
wal-g backup-list | tail -5
echo "Backup complete: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

#### Recovery to a Specific Timestamp

```bash
#!/bin/bash
# restore-postgres.sh — PITR recovery to a specific timestamp

TARGET_TIME="${1:-$(date -u -d '5 minutes ago' +%Y-%m-%dT%H:%M:%SZ)}"
PGDATA="/var/lib/postgresql/data"

echo "Restoring PostgreSQL to: $TARGET_TIME"

# 1. Stop PostgreSQL
pg_ctl -D "$PGDATA" stop -m fast

# 2. Move existing data directory aside (for safety)
mv "$PGDATA" "${PGDATA}.corrupt.$(date +%s)"

# 3. Fetch and restore latest base backup
export WALG_S3_PREFIX="s3://ggid-backups/postgres"
wal-g backup-fetch "$PGDATA" LATEST

# 4. Create recovery configuration
cat > "$PGDATA/recovery.signal" <<EOF
EOF

cat > "$PGDATA/postgresql.auto.conf" <<EOF
restore_command = 'wal-g wal-fetch "%f" "%p"'
recovery_target_time = '$TARGET_TIME'
recovery_target_action = 'promote'
EOF

# 5. Start PostgreSQL — it will enter recovery mode automatically
pg_ctl -D "$PGDATA" start

# 6. Monitor recovery progress
tail -f "$PGDATA/log/postgresql.log" &
# Wait for "recovery complete" message
```

#### PITR in Go — Automated Recovery Trigger

```go
// Package dr provides disaster recovery automation for PostgreSQL.
package dr

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// PostgresRecovery configures a PITR restore operation.
type PostgresRecovery struct {
	PGDataDir  string
	S3Prefix   string
	TargetTime time.Time
	Promote    bool // if true, promote after recovery
}

// Restore performs a point-in-time recovery of PostgreSQL.
func (r *PostgresRecovery) Restore(ctx context.Context) error {
	// 1. Stop PostgreSQL
	if err := r.runCmd(ctx, "pg_ctl", "-D", r.PGDataDir, "stop", "-m", "fast"); err != nil {
		return fmt.Errorf("stop postgres: %w", err)
	}

	// 2. Fetch latest base backup
	fetch := exec.CommandContext(ctx, "wal-g", "backup-fetch", r.PGDataDir, "LATEST")
	fetch.Env = append(fetch.Env, "WALG_S3_PREFIX="+r.S3Prefix)
	if out, err := fetch.CombinedOutput(); err != nil {
		return fmt.Errorf("backup-fetch: %w\n%s", err, out)
	}

	// 3. Create recovery.signal
	if err := r.runCmd(ctx, "touch", r.PGDataDir+"/recovery.signal"); err != nil {
		return fmt.Errorf("create recovery.signal: %w", err)
	}

	// 4. Write recovery configuration
	action := "shutdown"
	if r.Promote {
		action = "promote"
	}
	conf := fmt.Sprintf(`restore_command = 'wal-g wal-fetch "%%f" "%%p"'
recovery_target_time = '%s'
recovery_target_action = '%s'
`, r.TargetTime.UTC().Format("2006-01-02T15:04:05Z"), action)

	if err := r.runCmd(ctx, "sh", "-c", fmt.Sprintf(
		"echo '%s' > %s/postgresql.auto.conf", conf, r.PGDataDir)); err != nil {
		return fmt.Errorf("write recovery conf: %w", err)
	}

	// 5. Start PostgreSQL in recovery mode
	if err := r.runCmd(ctx, "pg_ctl", "-D", r.PGDataDir, "start"); err != nil {
		return fmt.Errorf("start postgres: %w", err)
	}

	return nil
}

func (r *PostgresRecovery) runCmd(ctx context.Context, name string, args ...string) error {
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w\n%s", name, err, buf.String())
	}
	return nil
}
```

### Streaming Replication for Hot Standby

For near-zero RTO, a hot standby replica provides instant failover:

```ini
# Primary: postgresql.conf
wal_level = replica
max_wal_senders = 10
max_replication_slots = 10

# Primary: pg_hba.conf
host replication replicator 10.0.0.0/8 md5
```

```bash
# Standby: create replica
pg_basebackup \
  -h primary.postgres.internal \
  -U replicator \
  -D /var/lib/postgresql/data \
  -Fp -Xs -P -R

# The -R flag creates standby.signal and primary_conninfo automatically
# Start PostgreSQL — it will begin streaming WAL from primary
pg_ctl -D /var/lib/postgresql/data start
```

#### Failover with pg_promote

```bash
#!/bin/bash
# failover-postgres.sh — Promote standby to primary

PGDATA="/var/lib/postgresql/data"

echo "=== PostgreSQL Failover ==="

# 1. Promote standby to primary
pg_ctl -D "$PGDATA" promote

# 2. Wait for promotion to complete
for i in $(seq 1 30); do
  IS_PRIMARY=$(psql -tAc "SELECT pg_is_in_recovery()")
  if [ "$IS_PRIMARY" = "f" ]; then
    echo "Promotion successful"
    break
  fi
  echo "Waiting for promotion... ($i/30)"
  sleep 2
done

# 3. Update connection strings in services
# (handled by service discovery / config reload)
echo "Failover complete: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

---

## 3. NATS JetStream Backup

### GGID NATS Usage

GGID uses NATS JetStream as the audit event bus. The `audit` service's
`EventConsumer` (in `services/audit/internal/consumer/nats_consumer.go`) creates a
JetStream with **file-based storage** (`jetstream.FileStorage`) and a `LimitsPolicy`
retention. Events flow from publishers (auth, identity, policy) through JetStream to
the audit service consumer which persists them to PostgreSQL.

### Stream Snapshots to Disk

```go
// Package natsdr provides NATS JetStream backup and restore utilities.
package natsdr

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// StreamSnapshot represents a serialized JetStream state.
type StreamSnapshot struct {
	StreamName  string                 `json:"stream_name"`
	Config      jetstream.StreamConfig `json:"config"`
	State       jetstream.StreamState  `json:"state"`
	Messages    [][]byte               `json:"messages"`
	Consumers   []ConsumerSnapshot     `json:"consumers"`
}

// ConsumerSnapshot captures consumer state for recovery.
type ConsumerSnapshot struct {
	Name   string                  `json:"name"`
	Config jetstream.ConsumerConfig `json:"config"`
	State  jetstream.ConsumerInfo   `json:"state"`
}

// BackupStream exports all messages and consumer state from a JetStream.
func BackupStream(ctx context.Context, nc *nats.Conn, streamName string) (*StreamSnapshot, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("create jetstream context: %w", err)
	}

	stream, err := js.Stream(ctx, streamName)
	if err != nil {
		return nil, fmt.Errorf("get stream %s: %w", streamName, err)
	}

	// Snapshot stream config and state
	info, err := stream.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("stream info: %w", err)
	}

	snapshot := &StreamSnapshot{
		StreamName: streamName,
		Config:     info.Config,
		State:      info.State,
	}

	// Read all messages from the stream
	consumer, err := js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Durable:   "backup-consumer",
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("create backup consumer: %w", err)
	}
	defer js.DeleteConsumer(ctx, streamName, "backup-consumer")

	for {
		batch, err := consumer.FetchNoWait(ctx, 100)
		if err != nil {
			if err == jetstream.ErrNoMessages {
				break
			}
			return nil, fmt.Errorf("fetch messages: %w", err)
		}
		for msg := range batch.Messages() {
			snapshot.Messages = append(snapshot.Messages, msg.Data())
			msg.Ack()
		}
	}

	// Snapshot all durable consumers
	for name, cfg := range listDurableConsumers(ctx, js, streamName) {
		ci, _ := js.Consumer(ctx, streamName, name)
		if ci != nil {
			snapshot.Consumers = append(snapshot.Consumers, ConsumerSnapshot{
				Name:   name,
				Config: cfg,
			})
		}
	}

	return snapshot, nil
}

// SaveSnapshot writes the stream snapshot to a file (local or mounted S3 volume).
func SaveSnapshot(snap *StreamSnapshot, path string) error {
	data, err := json.Marshal(snap)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write snapshot file: %w", err)
	}
	return nil
}

// RestoreStream recreates a JetStream from a snapshot file.
func RestoreStream(ctx context.Context, nc *nats.Conn, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read snapshot: %w", err)
	}

	var snap StreamSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return fmt.Errorf("unmarshal snapshot: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("create jetstream context: %w", err)
	}

	// Recreate stream from snapshot config
	_, err = js.CreateOrUpdateStream(ctx, snap.Config)
	if err != nil {
		return fmt.Errorf("restore stream: %w", err)
	}

	// Replay all messages
	for _, msgData := range snap.Messages {
		_, err = js.Publish(ctx, snap.Config.Subjects[0], msgData)
		if err != nil {
			return fmt.Errorf("replay message: %w", err)
		}
	}

	// Recreate consumers
	for _, c := range snap.Consumers {
		_, err = js.CreateOrUpdateConsumer(ctx, snap.StreamName, c.Config)
		if err != nil {
			return fmt.Errorf("restore consumer %s: %w", c.Name, err)
		}
	}

	return nil
}

func listDurableConsumers(ctx context.Context, js jetstream.JetStream, stream string) map[string]jetstream.ConsumerConfig {
	// In production, use js.ListConsumers or ConsumerNames
	return make(map[string]jetstream.ConsumerConfig)
}
```

### File-Based vs Memory Storage

| Aspect | FileStorage | MemoryStorage |
|--------|-------------|---------------|
| **Durability** | Survives restart | Lost on restart |
| **Performance** | Slower (disk I/O) | Faster (RAM) |
| **Max Size** | Disk-bound | RAM-bound |
| **DR Suitability** | Excellent | Poor |
| **GGID Default** | Yes (audit stream) | — |

GGID already uses `jetstream.FileStorage` for the audit stream, which is the correct
choice for DR. Messages persist to the `ggid-nats-data` volume (in prod compose).

### Stream Copy to Remote NATS Cluster

```bash
#!/bin/bash
# nats-mirror.sh — Mirror JetStream to a remote NATS cluster for DR

SOURCE_URL="nats://nats:4222"
DEST_URL="nats://nats-dr.secondary-region.internal:4222"
STREAM="AUDIT_EVENTS"

nats stream copy \
  --source "$SOURCE_URL" \
  --dest "$DEST_URL" \
  "$STREAM"

echo "Stream mirror complete: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

---

## 4. Redis Snapshot & Restore

### GGID Redis Usage

Redis is used by the **auth** and **oauth** services for:
- Session token caching (active login sessions)
- Rate limiting counters
- OAuth authorization code and PKCE challenge storage
- Refresh token rotation tracking

The `docker-compose.prod.yaml` configures `--appendonly yes` for durability and sets
a 128MB memory limit with `allkeys-lru` eviction policy.

### RDB Snapshots vs AOF

| Feature | RDB Snapshot | AOF (Append-Only File) |
|---------|-------------|----------------------|
| **Recovery Granularity** | Point-in-time | All writes up to failure |
| **Performance Impact** | Fork-based, periodic | Per-write fsync |
| **File Size** | Compact | Larger (rewrites help) |
| **Data Loss Window** | Snapshot interval (minutes) | Configurable (1s - always) |
| **Best For** | Cold backups, DR images | Hot recovery, minimal RPO |

**Recommended**: Use both. RDB for off-site backup images, AOF for local crash recovery.

### Redis Configuration for DR

```conf
# redis.conf — Production DR configuration

# AOF: fsync every second (balances durability and performance)
appendonly yes
appendfsync everysec
no-appendfsync-on-rewrite no
auto-aof-rewrite-percentage 100
auto-aof-rewrite-min-size 64mb

# RDB: snapshot every 5 minutes if 100+ keys changed
save 300 100
save 60 10000
dbfilename dump.rdb
dir /data

# Replication (for hot standby)
replicaof redis-primary.internal 6379
replica-read-only yes
```

### Automated Backup with Timestamps

```bash
#!/bin/bash
# backup-redis.sh — Create RDB snapshot and upload to S3

REDIS_HOST="${REDIS_HOST:-redis:6379}"
REDIS_PASS="${REDIS_PASSWORD:-}"
S3_BUCKET="ggid-backups/redis"
TIMESTAMP=$(date -u +%Y%m%dT%H%M%SZ)
BACKUP_FILE="/tmp/redis-${TIMESTAMP}.rdb"

# 1. Trigger BGSAVE (non-blocking)
if [ -n "$REDIS_PASS" ]; then
  redis-cli -h "${REDIS_HOST%%:*}" -p "${REDIS_HOST##*:}" -a "$REDIS_PASS" BGSAVE
else
  redis-cli -h "${REDIS_HOST%%:*}" -p "${REDIS_HOST##*:}" BGSAVE
fi

# 2. Wait for BGSAVE to complete
while true; do
  STATUS=$(redis-cli -h "${REDIS_HOST%%:*}" -p "${REDIS_HOST##*:}" -a "$REDIS_PASS" INFO persistence 2>/dev/null \
    | grep rdb_bgsave_in_progress | cut -d: -f2 | tr -d '\r')
  if [ "$STATUS" = "0" ]; then
    break
  fi
  echo "Waiting for BGSAVE... ($STATUS)"
  sleep 2
done

# 3. Copy RDB file
docker cp ggid-redis:/data/dump.rdb "$BACKUP_FILE"

# 4. Upload to S3
aws s3 cp "$BACKUP_FILE" "s3://${S3_BUCKET}/" --sse AES256
rm "$BACKUP_FILE"

# 5. Delete old backups (retain 30 days)
aws s3 ls "s3://${S3_BUCKET}/" | awk '{print $4}' | \
  while read -r f; do
    DATE_PART=$(echo "$f" | grep -oE '[0-9]{8}T[0-9]{6}Z' | head -1)
    if [ -n "$DATE_PART" ]; then
      AGE_DAYS=$(( ( $(date +%s) - $(date -d "${DATE_PART:0:8}" +%s) ) / 86400 ))
      if [ "$AGE_DAYS" -gt 30 ]; then
        aws s3 rm "s3://${S3_BUCKET}/$f"
        echo "Deleted old backup: $f ($AGE_DAYS days old)"
      fi
    fi
  done

echo "Redis backup complete: $TIMESTAMP"
```

### Redis Backup Verification in Go

```go
// Package redisdr provides Redis backup verification utilities.
package redisdr

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// VerifyBackup connects to a restored Redis instance and checks data integrity.
func VerifyBackup(ctx context.Context, addr, password string) (*BackupReport, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
	})

	report := &BackupReport{CheckedAt: time.Now().UTC()}

	// 1. Ping
	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		return report, fmt.Errorf("ping failed: %w", err)
	}
	report.PingResult = pong

	// 2. Check DB size
	dbSize, err := rdb.DBSize(ctx).Result()
	if err != nil {
		return report, fmt.Errorf("dbsize: %w", err)
	}
	report.KeyCount = dbSize

	// 3. Sample random keys to verify data integrity
	if dbSize > 0 {
		sampleKeys, err := rdb.RandomKey(ctx).Result()
		if err == nil && sampleKeys != "" {
			report.SampleKey = sampleKeys
			report.SampleType, _ = rdb.Type(ctx, sampleKeys).Result()
		}
	}

	// 4. Check for critical GGID keys
	criticalPatterns := []string{
		"session:*",
		"rate_limit:*",
		"oauth:auth_code:*",
		"refresh_token:*",
	}

	for _, pattern := range criticalPatterns {
		keys, err := rdb.Keys(ctx, pattern).Result()
		if err != nil {
			report.Warnings = append(report.Warnings,
				fmt.Sprintf("could not scan pattern %s: %v", pattern, err))
			continue
		}
		report.PatternCounts[pattern] = len(keys)
	}

	return report, nil
}

// BackupReport summarizes backup verification results.
type BackupReport struct {
	CheckedAt     time.Time         `json:"checked_at"`
	PingResult    string            `json:"ping"`
	KeyCount      int64             `json:"key_count"`
	SampleKey     string            `json:"sample_key,omitempty"`
	SampleType    string            `json:"sample_type,omitempty"`
	PatternCounts map[string]int    `json:"pattern_counts"`
	Warnings      []string          `json:"warnings,omitempty"`
}

func init() {
	// Initialize PatternCounts to avoid nil map
}
```

### Sentinel for Automatic Failover

Redis Sentinel provides automated master election and client notification:

```conf
# sentinel.conf — Monitor GGID Redis primary

port 26379
sentinel monitor ggid-redis redis-primary.internal 6379 2
sentinel down-after-milliseconds ggid-redis 5000
sentinel failover-timeout ggid-redis 30000
sentinel parallel-syncs ggid-redis 1

# Notification script for DR alerts
sentinel notification-script ggid-redis /opt/ggid/scripts/redis-failover-alert.sh
```

---

## 5. Service Failover Architecture

### Active-Active vs Active-Passive

| Strategy | Services | RTO | Complexity | GGID Suitability |
|----------|----------|-----|-----------|-----------------|
| **Active-Active** | Gateway, Auth, OAuth | ~0 | High | Tier 0 — immediate failover |
| **Active-Passive** | Identity, Policy, Org | 2-5 min | Medium | Tier 1 — hot standby |
| **Cold Standby** | Audit (query) | 15-30 min | Low | Tier 2 — can replay from NATS |

### Health-Check-Based Failover

GGID already implements a sophisticated health check system in the gateway
(`services/gateway/internal/healthcheck/healthcheck.go`) with three probe modes:

- **`/healthz?mode=live`** — Liveness probe: process is alive (no backend checks)
- **`/healthz?mode=ready`** — Readiness probe: all backends healthy
- **`/healthz`** — Full aggregated status with per-service detail

The `AggregatedStatus` struct reports `healthy` and `unhealthy` counts, returning HTTP
503 if any service is down. This is the ideal contract for load balancer health checks.

### Load Balancer Configuration

```yaml
# nginx — Health-check-based upstream failover for GGID
upstream ggid_gateway {
  # Active region (primary)
  server gateway-primary.internal:8080 max_fails=3 fail_timeout=10s;

  # DR region (standby) — receives traffic only when primary is down
  server gateway-dr.secondary.internal:8080 backup max_fails=3 fail_timeout=30s;
}

server {
  listen 443 ssl http2;
  server_name iam.example.com;

  location / {
    proxy_pass http://ggid_gateway;
    proxy_next_upstream error timeout http_502 http_503 http_504;
    proxy_connect_timeout 3s;
    proxy_read_timeout 30s;

    # Health check: if gateway returns 503, try DR region
    proxy_intercept_errors on;
    error_page 502 503 504 = @failover;
  }

  location @failover {
    proxy_pass https://gateway-dr.secondary.internal:8080;
  }

  # Separate health endpoint for LB
  location /healthz {
    proxy_pass http://ggid_gateway/healthz?mode=ready;
    access_log off;
  }
}
```

### DNS-Based Failover

DNS TTL determines how quickly clients redirect to a DR region:

| Record Type | TTL | Use Case |
|------------|-----|----------|
| A record (primary) | 30s | Short TTL for fast DNS failover |
| A record (DR) | 30s | Same TTL for symmetric failover |
| CNAME (traffic manager) | 60s | Route53 latency/routing policy |

```json
// AWS Route53 health check + failover record
{
  "RecordType": "A",
  "SetIdentifier": "primary",
  "Failover": "PRIMARY",
  "TTL": 30,
  "ResourceRecords": ["10.0.1.10"],
  "HealthCheckId": "abc123-primary-gateway"
},
{
  "RecordType": "A",
  "SetIdentifier": "dr",
  "Failover": "SECONDARY",
  "TTL": 30,
  "ResourceRecords": ["10.1.1.10"]
}
```

### Graceful Shutdown for Zero-Downtime Deploys

GGID's `audit/cmd/main.go` already implements graceful shutdown correctly:

```go
// Existing pattern in GGID — graceful shutdown
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
<-sigCh

log.Println("Audit Service: shutting down...")
grpcServer.GracefulStop()
if natsConsumer != nil {
    natsConsumer.Close()
}
httpServer.Shutdown(context.Background())
log.Println("Audit Service: stopped")
```

**Enhanced graceful shutdown with connection draining:**

```go
// Package shutdown provides enhanced graceful shutdown with connection draining.
package shutdown

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// GracefulManager coordinates shutdown of HTTP and gRPC servers.
type GracefulManager struct {
	httpServers  map[string]*http.Server
	shutdownFuncs []func(context.Context) error
	drainTimeout time.Duration
}

func New() *GracefulManager {
	return &GracefulManager{
		httpServers:  make(map[string]*http.Server),
		drainTimeout: 30 * time.Second,
	}
}

// Wait blocks until SIGINT/SIGTERM, then gracefully shuts down all servers.
func (m *GracefulManager) Wait() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	log.Printf("Received %s, draining connections (timeout: %s)...", sig, m.drainTimeout)

	ctx, cancel := context.WithTimeout(context.Background(), m.drainTimeout)
	defer cancel()

	var wg sync.WaitGroup
	for name, srv := range m.httpServers {
		wg.Add(1)
		go func(name string, srv *http.Server) {
			defer wg.Done()
			if err := srv.Shutdown(ctx); err != nil {
				log.Printf("  %s: shutdown error: %v", name, err)
			} else {
				log.Printf("  %s: drained successfully", name)
			}
		}(name, srv)
	}

	for _, fn := range m.shutdownFuncs {
		wg.Add(1)
		go func(fn func(context.Context) error) {
			defer wg.Done()
			if err := fn(ctx); err != nil {
				log.Printf("  cleanup error: %v", err)
			}
		}(fn)
	}

	wg.Wait()
	log.Println("All connections drained, exiting")
}
```

---

## 6. Multi-Region Active-Active Design

### Architecture Overview

```
                    ┌──────────────────────────────────────────────────┐
                    │              Global Load Balancer                 │
                    │     (Route53 / Cloudflare latency routing)       │
                    │                                                  │
                    │   ┌─────────────────┐  ┌─────────────────┐      │
                    │   │  iam-us.example  │  │  iam-eu.example  │     │
                    │   │  (30s TTL)       │  │  (30s TTL)       │     │
                    │   └────────┬─────────┘  └────────┬────────┘     │
                    └────────────┼──────────────────────┼─────────────┘
                                 │                      │
              ┌──────────────────┼──────────────────────┼──────────────┐
              │     US-EAST REGION         │      EU-WEST REGION       │
              │                            │                           │
              │  ┌─────────────────┐       │   ┌─────────────────┐    │
              │  │  Gateway (x3)   │       │   │  Gateway (x3)   │    │
              │  └────────┬────────┘       │   └────────┬────────┘    │
              │           │                │            │              │
              │  ┌────────┴────────────────┴────────────┴────────┐    │
              │  │  Auth  │ OAuth  │ Identity │ Policy │ Org     │    │
              │  └───┬────┴───┬────┴────┬─────┴───┬────┴───┬────┘    │
              │      │        │         │         │        │          │
              │  ┌───┴────┐ ┌─┴──┐ ┌────┴────┐ ┌──┴───┐ ┌─┴──────┐   │
              │  │Postgres│ │Redis│ │  NATS   │ │ LDAP │ │Audit   │   │
              │  │Primary │ │     │ │JetStream│ │      │ │Consumer│   │
              │  └───┬────┘ └────┘ └─────────┘ └──────┘ └────────┘   │
              │      │                                                      │
              │      │     Logical Replication      │                    │
              │      └──────────────────────────────┘                    │
              │             (bidirectional, conflict-aware)              │
              │                                                            │
              │  ┌────────┐ ┌──────┐ ┌──────────┐ ┌──────┐ ┌───────┐    │
              │  │Postgres│ │Redis │ │  NATS    │ │ LDAP │ │Audit  │    │
              │  │Primary │ │      │ │JetStream │ │      │ │Consumer│   │
              │  └────────┘ └──────┘ └──────────┘ └──────┘ └───────┘    │
              └────────────────────────────────────────────────────────┘
                          EU-WEST REGION
```

### Cross-Region PostgreSQL Replication

Physical streaming replication requires the standby to be in the same region for
acceptable latency. For multi-region active-active, **logical replication** is required:

```sql
-- Create publication on both primaries (bidirectional)
CREATE PUBLICATION ggid_multi_region
  FOR TABLE users, roles, permissions, organizations, org_members,
              oauth_clients, audit_events, refresh_tokens;

-- Create subscription on the other region's primary
CREATE SUBSCRIPTION ggid_us_to_eu
  CONNECTION 'host=us-postgres.internal port=5432 user=replicator password=...'
  PUBLICATION ggid_multi_region;
```

### Conflict Resolution for User Data

Bidirectional replication creates conflict risk. Strategy:

```go
// Package conflict provides last-write-wins conflict resolution for multi-region IAM.
package conflict

import (
	"database/sql"
	"time"
)

// ResolveUserUpdate implements last-write-wins with updated_at comparison.
// Called by a BEFORE UPDATE trigger on the users table.
func ResolveUserUpdate(oldUpdatedAt, newUpdatedAt time.Time, oldRegion, newRegion string) (accept bool) {
	// Accept the update if:
	// 1. New update is more recent (LWW)
	// 2. Same timestamp but this region is the designated "tie-breaker"
	if newUpdatedAt.After(oldUpdatedAt) {
		return true
	}
	if newUpdatedAt.Equal(oldUpdatedAt) {
		// Tie-breaker: alphabetical region name wins deterministically
		return newRegion <= oldRegion
	}
	return false // reject stale update
}
```

```sql
-- Trigger: conflict resolution for multi-region replication
CREATE OR REPLACE FUNCTION resolve_user_conflict()
RETURNS TRIGGER AS $$
BEGIN
  -- Only applies to replicated updates (not local app writes)
  IF NEW.updated_at < OLD.updated_at THEN
    -- Stale replicated write — reject
    RETURN NULL;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER user_conflict_trigger
  BEFORE UPDATE ON users
  FOR EACH ROW
  EXECUTE FUNCTION resolve_user_conflict();
```

### Session Continuity Across Regions

JWT tokens are self-contained and can be validated in any region that has the public
key. This is a major DR advantage:

```go
// Token validation works in any region — no shared session store needed
// As long as both regions share the same RSA key pair (via shared secret mount)
// and JWKS endpoint, any JWT issued in Region A is valid in Region B.

// The gateway validates JWTs using the public key:
// 1. Load /configs/rsa_public.pem (shared volume across regions)
// 2. Verify signature, expiry, tenant_id claim
// 3. No Redis call needed for basic validation
```

**Refresh token rotation** requires coordination. Use a distributed lock:

```go
// RotateRefreshToken uses Redis SETNX as a distributed lock to prevent
// concurrent rotation across regions.
func RotateRefreshToken(ctx context.Context, rdb *redis.Client, oldToken string) (string, error) {
	lockKey := "refresh_lock:" + oldToken

	// Acquire lock with 5s TTL
	acquired, err := rdb.SetNX(ctx, lockKey, "1", 5*time.Second).Result()
	if err != nil {
		return "", fmt.Errorf("acquire lock: %w", err)
	}
	if !acquired {
		return "", ErrConcurrentRotation
	}
	defer rdb.Del(ctx, lockKey)

	// Generate new token, invalidate old, persist to DB
	// Both regions' Redis instances are replicated, so the lock is visible everywhere
	newToken := generateToken()
	return newToken, nil
}
```

---

## 7. DR Runbook Templates

### Runbook: PostgreSQL Primary Failure

```
INCIDENT: PostgreSQL Primary Failure
SEVERITY: P0
ON-CALL:  Database team

PRECONDITIONS:
  - Hot standby replica is running and receiving WAL
  - WAL-G backups are current (verify S3 bucket)

STEPS:
  1. CONFIRM: pg_isready -h postgres-primary.internal
     If timeout → primary is down.

  2. PROMOTE STANDBY:
     pg_ctl -D /var/lib/postgresql/data promote
     Wait for "database system is ready to accept connections"

  3. VERIFY: psql -h postgres-standby.internal -c "SELECT pg_is_in_recovery()"
     Expected: f (not in recovery — promoted to primary)

  4. UPDATE DNS: Change postgres.internal → new primary IP
     TTL: 30s
     Wait for DNS propagation (check with dig)

  5. RESTART SERVICES that hold DB connection pools:
     - identity, policy, org, audit, auth, oauth
     Or trigger graceful reconnection.

  6. VERIFY GGID: curl https://iam.example.com/healthz?mode=ready
     Expected: {"status":"healthy","unhealthy":0}

  7. SETUP NEW STANDBY from the promoted primary:
     pg_basebackup -h <new-primary> -U replicator -D /var/lib/postgresql/data -R

VERIFICATION CHECKLIST:
  [ ] All services report healthy via /healthz
  [ ] New user registration works (POST /api/v1/auth/register)
  [ ] Login works (POST /api/v1/auth/login)
  [ ] JWT validation works (GET /api/v1/users with Bearer token)
  [ ] Audit events are flowing (POST test event, query /api/v1/audit)
  [ ] Replication lag is 0 (SELECT * FROM pg_stat_replication)

COMMUNICATION:
  - Post status page update: "Database failover completed"
  - Notify stakeholders via #incident channel
  - Create postmortem within 48 hours
```

### Runbook: NATS JetStream Loss

```
INCIDENT: NATS JetStream Unavailable
SEVERITY: P1 (audit events buffered, no data loss expected)

STEPS:
  1. CONFIRM: wget -qO- http://nats:8222/healthz
     If empty or 503 → NATS is down.

  2. CHECK if services are still publishing (they retry automatically):
     - Publishers use nats.MaxReconnects(-1) with 2s reconnect wait
     - Messages buffer in publisher memory during outage

  3. RESTART NATS:
     docker restart ggid-nats
     If persistent volume is intact, JetStream recovers from disk.

  4. IF DATA VOLUME CORRUPT — restore from snapshot:
     # Restore from backup
     go run cmd/nats-restore/main.go \
       --snapshot /backups/nats/audit-events-latest.json \
       --url nats://nats:4222

  5. VERIFY consumer resumed:
     Check audit service logs: "NATS consumer started"
     Query: SELECT count(*) FROM audit_events WHERE created_at > NOW() - INTERVAL '5 min'

  6. IF MESSAGES WERE LOST (volume destroyed, no snapshot):
     - This is expected to be RARE (file storage should survive restarts)
     - Note gap in audit log: "Audit gap: <start> to <end> due to NATS data loss"
     - For compliance: document the gap and root cause
```

### Runbook: Redis Loss

```
INCIDENT: Redis Unavailable
SEVERITY: P0 (auth sessions and rate limiting affected)

IMPACT:
  - Active sessions: users may need to re-login (session cache lost)
  - Rate limiting: resets to zero (brief vulnerability window)
  - OAuth flows: in-flight authorization codes lost

STEPS:
  1. CONFIRM: redis-cli -h redis:6379 ping
     If "MISCONF Redis is configured to save RDB snapshots but not able to persist"
     → disk full, check disk space.
     If "NOAUTH" → password changed (check config).
     If connection refused → Redis process dead.

  2. RESTART REDIS:
     docker restart ggid-redis
     AOF file will replay on startup (check logs for "DB loaded from append-only file")

  3. IF AOF CORRUPT:
     redis-check-aof --fix /data/appendonly.aof
     # Then restart

  4. IF DATA LOST (no AOF/RDB):
     - Sessions are gone: users must re-login
     - This is acceptable — JWTs remain valid (they're self-contained)
     - Only refresh token rotation state is lost (double-spend risk window)

  5. VERIFY:
     redis-cli -h redis:6379 ping → PONG
     Login with test account → success
     Check rate limiting: make 6 rapid login attempts → 429 on 6th
```

### Runbook: Full Region Failure

```
INCIDENT: Primary Region Down (us-east)
SEVERITY: P0
TRIGGER: Health checks failing for all services in us-east

STEPS:
  1. ACTIVATE DR REGION:
     # Update DNS to point to eu-west
     aws route53 change-resource-record-sets \
       --hosted-zone-id Z123 \
       --change-batch '{"Changes":[{"Action":"UPSERT",...}]}'
     TTL: 30s

  2. VERIFY eu-west is receiving traffic:
     curl https://iam-eu.example.com/healthz?mode=ready

  3. PROMOTE eu-west PostgreSQL to primary (if not already):
     See PostgreSQL failover runbook above.

  4. VERIFY cross-region replication was current:
     Compare row counts between regions (if old region partially available):
     SELECT count(*) FROM users;
     SELECT count(*) FROM audit_events;

  5. COMMUNICATE:
     - Status page: "IAM service degraded in us-east, failover to eu-west complete"
     - Stakeholder email: "All auth requests now served from eu-west region"
     - Monitor: Watch for 5x normal error rate (indicates replication lag issues)

  6. RECOVERY OF us-east (when region comes back):
     - Do NOT immediately repoint DNS back
     - Re-establish replication: us-east as standby of eu-west
     - Run data consistency checks
     - Perform controlled DNS switchback during low-traffic window
```

---

## 8. Backup Testing & DR Drills

### Why Untested Backups Are Worse Than No Backups

An untested backup creates false confidence. Teams believe they can recover but
discover at the worst possible moment that:

- The backup job was silently failing for months
- The restore process requires dependencies not available in DR
- The backup format is incompatible with the current schema version
- Encrypted backups can't be decrypted (lost keys)

### Automated Restore Testing in CI

```yaml
# .github/workflows/backup-restore-test.yaml
name: Backup Restore Test
on:
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM UTC
  workflow_dispatch:

jobs:
  test-postgres-restore:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Start PostgreSQL
        run: |
          docker run -d --name pg-test \
            -e POSTGRES_PASSWORD=test \
            -v ./backups/latest:/backups:ro \
            postgres:16-alpine

      - name: Restore from backup
        run: |
          docker exec pg-test pg_restore \
            -U postgres -d postgres \
            -1 /backups/ggid-latest.dump

      - name: Verify data integrity
        run: |
          USER_COUNT=$(docker exec pg-test psql -U postgres -tAc \
            "SELECT count(*) FROM users")
          if [ "$USER_COUNT" -lt 1 ]; then
            echo "FAIL: No users after restore"
            exit 1
          fi

          # Verify referential integrity
          docker exec pg-test psql -U postgres -c \
            "SELECT * FROM users u LEFT JOIN organizations o ON o.id = u.org_id WHERE o.id IS NULL LIMIT 5"
          # Should return 0 orphaned rows

      - name: Alert on failure
        if: failure()
        run: |
          curl -X POST "$SLACK_WEBHOOK" \
            -d '{"text":"BACKUP RESTORE TEST FAILED in CI!"}'

  test-redis-restore:
    runs-on: ubuntu-latest
    steps:
      - name: Start Redis
        run: docker run -d --name redis-test -p 6379:6379 redis:7-alpine

      - name: Restore RDB
        run: |
          docker cp backups/redis-latest.rdb redis-test:/data/dump.rdb
          docker restart redis-test
          sleep 3

      - name: Verify
        run: |
          COUNT=$(redis-cli ping && redis-cli dbsize)
          echo "Redis keys after restore: $COUNT"
```

### Game Day Exercises

Quarterly DR game days validate end-to-end recovery:

| Exercise | Frequency | Duration | What We Learn |
|----------|-----------|----------|---------------|
| PostgreSQL failover | Quarterly | 30 min | Actual failover time, service reconnection |
| Redis loss simulation | Quarterly | 15 min | Session impact, rate limit behavior |
| Full region failover | Bi-annual | 2 hours | DNS propagation, cross-region consistency |
| Backup restore from cold storage | Monthly | 1 hour | Backup integrity, decryption, schema compat |
| Chaos: Kill random service | Monthly | 45 min | Service resilience, fallback behavior |

### Chaos Engineering for IAM

```yaml
# chaos-experiment.yaml — Kill auth service pod during peak traffic
apiVersion: litmuschaos.io/v1alpha1
kind: ChaosEngine
metadata:
  name: auth-service-kill
spec:
  experiments:
    - name: pod-delete
      spec:
        components:
          env:
            - name: TOTAL_CHAOS_DURATION
              value: "30"
            - name: CHAOS_INTERVAL
              value: "10"
            - name: TARGET_PODS
              value: "ggid-auth-.*"
        probe:
          - name: gateway-still-healthy
            type: httpProbe
            httpProbe/inputs:
              url: "http://gateway/healthz?mode=ready"
              expectedResponseCode: "200"
```

### DR Metrics Dashboard

Track these KPIs continuously:

| Metric | Target | Alert Threshold |
|--------|--------|----------------|
| Backup success rate | 100% | < 100% for 2 consecutive runs |
| Backup duration (PostgreSQL) | < 10 min | > 30 min |
| Restore time (PostgreSQL) | < 15 min | > 30 min |
| WAL archive lag | 0 | > 60s |
| Replica lag | < 1s | > 10s |
| DR drill pass rate | 100% | < 80% |

---

## 9. GGID DR Gap Analysis

### What GGID Currently Has

| Capability | Status | Location |
|-----------|--------|----------|
| Docker volume persistence | Present | `docker-compose.yaml` — `ggid-pgdata`, `ggid-configs` |
| Redis AOF | Present | `docker-compose.prod.yaml` — `appendonly yes`, `ggid-redis-data` volume |
| NATS file storage | Present | `nats_consumer.go` — `jetstream.FileStorage`, `ggid-nats-data` volume |
| Health check endpoints | Present | All services expose `/healthz` and `/readyz` |
| Gateway health aggregation | Present | `healthcheck.go` — `CheckAll()`, `ReadyHandler()`, `DeepHandler()` |
| Graceful shutdown | Present | All service `cmd/main.go` — `signal.Notify`, `GracefulStop()`, `Shutdown()` |
| Network isolation | Present | `docker-compose.prod.yaml` — `frontend-net`, `backend-net`, `data-net` |
| Resource limits | Present | `docker-compose.prod.yaml` — memory/CPU limits per service |
| Restart policies | Present | `docker-compose.prod.yaml` — `restart: unless-stopped` |

### What GGID Is Missing

| Gap | Severity | Description |
|-----|----------|-------------|
| **No PostgreSQL backups** | P0 | No pg_dump, no WAL archiving, no WAL-G/WAL-E. Volume is the only copy. |
| **No PostgreSQL replication** | P0 | No streaming replica, no hot standby. Single point of failure. |
| **No Redis replication** | P1 | No replica, no Sentinel. Single Redis instance. |
| **No Redis password in dev** | P1 | Dev `docker-compose.yaml` has no Redis auth. Prod has it. |
| **No NATS cluster** | P1 | Single NATS node. No clustering or leaf nodes for DR. |
| **No automated backup scripts** | P0 | No cron jobs for PostgreSQL/Redis/NATS backups. |
| **No backup verification** | P0 | No CI job to test restore from backups. |
| **No multi-region config** | P1 | Single-region only. No cross-region replication. |
| **No DNS failover** | P1 | No Route53/Cloudflare failover configuration. |
| **No DR runbook** | P0 | No documented recovery procedures. |
| **No LDAP backup** | P2 | LDAP data in volume, no LDIF export schedule. |
| **No config/secret backup** | P1 | RSA keys in `ggid-configs` volume, no off-site copy. |
| **No chaos testing** | P2 | No game day exercises, no chaos engineering. |

### Risk Assessment

```
 SINGLE POINT OF FAILURE ANALYSIS

 PostgreSQL ──── P0 CRITICAL ──── All 7 services depend on it
     │                               No backup, no replica
     │
 Redis ──────── P1 HIGH ────────── Auth + OAuth depend on it
     │                               No replica, no Sentinel
     │
 NATS ────────── P1 HIGH ────────── Audit depends on it
     │                               No cluster, single node
     │
 LDAP ────────── P2 MODERATE ────── Auth fallback (local provider)
                                   No backup, but local auth still works
```

**If PostgreSQL is lost and no backup exists, the entire IAM system must be rebuilt
from scratch.** All user accounts, roles, permissions, organizations, OAuth clients,
and audit history are irrecoverably destroyed. This is an existential risk.

---

## 10. Implementation Roadmap

### Priority-Ordered Action Items

| # | Action Item | Effort | Priority | Dependency |
|---|------------|--------|----------|------------|
| 1 | **Implement PostgreSQL WAL-G backup pipeline** | 2-3 days | P0 | S3 bucket |
| 2 | **Deploy PostgreSQL streaming replica + auto-failover** | 3-5 days | P0 | Item 1 |
| 3 | **Create DR runbook and on-call rotation** | 1-2 days | P0 | Items 1-2 |
| 4 | **Deploy Redis Sentinel + replica** | 1-2 days | P1 | Redis instance |
| 5 | **Add NATS clustering (3-node) + stream mirroring** | 2-3 days | P1 | NATS instance |
| 6 | **Build CI-based backup restore test** | 1-2 days | P1 | Items 1, 4 |
| 7 | **Configure DNS failover (Route53 health checks)** | 1 day | P1 | Items 1-2 |
| 8 | **Implement multi-region logical replication** | 5-10 days | P2 | Items 1-2 |
| 9 | **First DR game day exercise** | 0.5 days | P2 | Items 1-5 |

### Detailed Breakdown

#### Phase 1: Eliminate Data Loss Risk (Week 1-2)

```bash
# 1. Install WAL-G in PostgreSQL container
# 2. Configure S3 bucket for WAL archive
# 3. Set up daily base backup cron
# 4. Verify PITR works in staging

# Deploy replica:
docker-compose -f docker-compose.prod.yaml \
  -f docker-compose.replica.yaml up -d
```

#### Phase 2: Eliminate Single Points of Failure (Week 3-4)

```yaml
# docker-compose.replica.yaml (new)
services:
  postgres-replica:
    image: postgres:16-alpine
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      PRIMARY_HOST: postgres
      REPL_USER: replicator
      REPL_PASSWORD: ${REPL_PASSWORD}
    entrypoint: ""
    command:
      - sh
      - -c
      - |
        until pg_isready -h postgres -U ggid; do sleep 2; done
        pg_basebackup -h postgres -U replicator -D $$PGDATA -Fp -Xs -R
        postgres
    volumes:
      - ggid-pgdata-replica:/var/lib/postgresql/data
```

#### Phase 3: Automated Validation (Week 5-6)

```bash
# Weekly restore test in staging:
# 1. Spin up fresh PostgreSQL
# 2. Restore latest WAL-G backup
# 3. Run schema validation queries
# 4. Alert if any check fails
# 5. Report: restore time, data integrity, schema version
```

#### Phase 4: Multi-Region (Month 2-3)

- Stand up secondary region with full GGID stack
- Configure logical replication (bidirectional)
- DNS failover with Route53 health checks
- First cross-region DR drill

### Success Criteria

| Criterion | Measurement |
|-----------|------------|
| PostgreSQL RPO | < 60 seconds (WAL archive interval) |
| PostgreSQL RTO | < 15 minutes (automated failover) |
| Redis RPO | < 1 second (AOF everysec) |
| Redis RTO | < 30 seconds (Sentinel failover) |
| Backup test pass rate | 100% weekly |
| DR drill completion | Quarterly, 80%+ procedures succeed |
| Audit event loss | 0 events during failover (JetStream durability) |

---

## Appendix: Quick Reference Commands

```bash
# PostgreSQL backup
wal-g backup-push /var/lib/postgresql/data

# PostgreSQL restore (PITR)
wal-g backup-fetch /var/lib/postgresql/data LATEST
echo "recovery_target_time = '2025-01-15T10:30:00Z'" >> recovery.conf

# PostgreSQL failover
pg_ctl -D /var/lib/postgresql/data promote

# Redis backup
redis-cli BGSAVE
docker cp ggid-redis:/data/dump.rdb redis-backup-$(date +%s).rdb

# Redis restore
docker cp redis-backup.rdb ggid-redis:/data/dump.rdb
docker restart ggid-redis

# NATS stream info
nats stream info AUDIT_EVENTS

# NATS stream purge (not backup — use carefully)
nats stream purge AUDIT_EVENTS

# Gateway health check (all services)
curl -s https://iam.example.com/healthz | jq .

# Gateway readiness (for LB)
curl -sf https://iam.example.com/healthz?mode=ready && echo "READY"
```

---

*Document version: 1.0 | Last updated: 2025 | GGID IAM Suite*
