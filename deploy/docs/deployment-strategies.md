# GGID 部署策略指南

> 金丝雀、蓝绿、滚动更新三种策略的配置和使用方法。

---

## 概览

| 策略 | 适用场景 | 停机时间 | 回滚速度 | 资源开销 |
|------|---------|---------|---------|---------|
| 滚动更新 (默认) | 常规发布 | 零 | 中（逐步回滚） | 1x |
| 金丝雀 | 高风险变更 | 零 | 快（切回旧版） | 1.1-1.5x |
| 蓝绿 | 需要预览验证 | 零（切换瞬间） | 最快（切回蓝） | 2x |

---

## 1. 滚动更新（默认）

Helm 默认策略，配合 `values-prod.yaml` 使用即可：

```bash
helm upgrade ggid deploy/helm/ggid/ -f deploy/helm/ggid/values-prod.yaml
```

K8s 自动滚动更新，配合 PDB 确保可用性：
```yaml
podDisruptionBudget:
  enabled: true
  minAvailable: 2
```

回滚：
```bash
helm rollback ggid <previous-revision>
```

---

## 2. 金丝雀发布

### 前提条件
- 安装 [Argo Rollouts](https://argoproj.github.io/rollouts/) 或 [Flagger](https://flagger.app/)

### 使用
```bash
# 部署金丝雀配置
helm upgrade ggid deploy/helm/ggid/ \
  -f deploy/helm/ggid/values-prod.yaml \
  -f deploy/helm/ggid/values-canary.yaml
```

### Argo Rollouts 安装
```bash
kubectl create namespace argo-rollouts
kubectl apply -n argo-rollouts -f https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml
```

### 流量策略（values-canary.yaml）
```
新版本 → 10% 流量 (5分钟) → 50% 流量 (10分钟) → 100% 流量
         ↑ 错误率 > 5% 自动回滚
```

### 手动控制
```bash
# 查看金丝雀状态
kubectl argo rollouts get rollout ggid-gateway -n ggid --watch

# 手动 promote（跳过等待）
kubectl argo rollouts promote ggid-gateway -n ggid

# 手动回滚
kubectl argo rollouts abort ggid-gateway -n ggid
```

### Flagger 替代方案
```yaml
# flagger-gateway.yaml
apiVersion: flagger.app/v1beta1
kind: Canary
metadata:
  name: ggid-gateway
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: ggid-gateway
  service:
    port: 8080
  analysis:
    interval: 1m
    threshold: 5
    maxWeight: 100
    stepWeight: 20
    metrics:
      - name: error-rate
        threshold: 2
        query: |
          sum(rate(http_requests_total{status=~"5.."}[1m])) /
          sum(rate(http_requests_total[1m])) * 100
```

---

## 3. 蓝绿部署

### 使用
```bash
# 部署蓝绿配置
helm upgrade ggid deploy/helm/ggid/ \
  -f deploy/helm/ggid/values-prod.yaml \
  -f deploy/helm/ggid/values-bluegreen.yaml
```

### 流程
1. 新版本部署到 "preview" 环境（绿色）
2. 健康检查通过后等待人工确认
3. `autoPromotionEnabled: false` → 手动 promote 切换流量
4. 旧版本（蓝色）保留 60s 后缩容

### 手动切换
```bash
# 查看蓝绿状态
kubectl argo rollouts get rollout ggid-gateway -n ggid

# 切换到新版本（promote）
kubectl argo rollouts promote ggid-gateway -n ggid

# 回滚到旧版本
kubectl argo rollouts undo ggid-gateway -n ggid
```

---

## 4. 混合策略

关键服务（gateway/auth）使用蓝绿，其他服务使用金丝雀：

```bash
# 创建混合配置
cat > values-mixed.yaml <<'EOF'
gateway:
  rollout:
    strategy: blueGreen
    blueGreen:
      autoPromotionEnabled: false

auth:
  rollout:
    strategy: blueGreen
    blueGreen:
      autoPromotionEnabled: false

# 其他服务默认金丝雀
EOF

helm upgrade ggid deploy/helm/ggid/ \
  -f deploy/helm/ggid/values-prod.yaml \
  -f deploy/helm/ggid/values-mixed.yaml
```

---

## 5. 数据库迁移策略

**重要**：DB schema 变更必须向前兼容，兼容蓝绿/金丝雀并行运行：

1. **Expand**：先添加新列/表（向后兼容）
2. **Migrate**：部署新版本代码（读写兼容新旧 schema）
3. **Contract**：确认所有实例已更新后删除旧列

```bash
# Expand
ggid-migrate up

# Deploy new version (blue/green or canary)
helm upgrade ggid ... 

# Contract (after all old instances removed)
ggid-migrate contract
```

---

## 6. 验证清单

- [ ] Argo Rollouts controller 已安装
- [ ] Prometheus metrics 端点可用（用于自动分析）
- [ ] PDB 配置正确（minAvailable >= 2）
- [ ] Health check 端点（/healthz）可靠
- [ ] DB 迁移向前兼容
- [ ] 回滚测试通过
