# GGID 标准发版流程

> 本文档描述 GGID 平台从代码变更到正式发布的完整流程。

---

## 发版触发

推送 `v*` 格式的 Git tag 自动触发 release workflow：

```bash
# 1. 确保 main 分支干净且 CI 通过
git status
gh run list --repo topcheer/ggid --limit 1

# 2. 更新版本号（如果需要）
# - Go: 不需要（Go modules 用 tag）
# - Node SDK: sdk/node/package.json version
# - Python SDK: sdk/python/pyproject.toml version

# 3. 打 tag 并推送
git tag v2.1.0
git push origin v2.1.0
```

## Release Workflow 执行内容

```
push tag v*
     │
     ├─► test-gate (build + go test ./...)
     │       └─ FAIL → 中止发布
     │
     ├─► docker (8 services × multi-arch)
     │       └─ ghcr.io/<owner>/ggid-{svc}:v2.1.0 + :latest
     │
     ├─► sdk-tags (10 SDK 子模块 tag)
     │       └─ sdk/node/v2.1.0 → 触发 npm publish
     │       └─ sdk/python/v2.1.0 → 触发 PyPI publish
     │
     └─► github-release (needs: docker + sdk-tags + test-gate)
             └─ 自动生成 Changelog + GitHub Release
```

## 预发布检查清单

| # | 检查项 | 命令 |
|---|--------|------|
| 1 | Go 编译通过 | `go build ./...` |
| 2 | Go 测试通过 | `go test ./... -count=1` |
| 3 | 无未提交变更 | `git status --short` |
| 4 | CI main 分支绿色 | `gh run list --limit 1` |
| 5 | SDK 版本号已更新 | `grep version sdk/node/package.json sdk/python/pyproject.toml` |
| 6 | CHANGELOG 已更新 | `docs/CHANGELOG.md` |
| 7 | Helm Chart 版本同步 | `deploy/helm/ggid/Chart.yaml appVersion` |

## 版本号规范 (SemVer)

| 格式 | 用途 | 示例 |
|------|------|------|
| `vMAJOR.MINOR.PATCH` | 正式版 | `v2.1.0` |
| `vMAJOR.MINOR.PATCH-rc.N` | 候选版 | `v2.1.0-rc.1` |
| `vMAJOR.MINOR.PATCH-beta.N` | 测试版 | `v2.1.0-beta.1` |

- **MAJOR**: 破坏性 API 变更
- **MINOR**: 新功能，向后兼容
- **PATCH**: Bug 修复

## 回滚

```bash
# 删除 tag（仅限 rc/beta）
git tag -d v2.1.0-rc.1
git push origin :refs/tags/v2.1.0-rc.1

# Helm 回滚
helm rollback ggid <revision>

# Docker 镜像回退
kubectl set image deployment/ggid-gateway \
  ggid-gateway=ghcr.io/topcheer/ggid-gateway:v2.0.0
```

## 发布后验证

```bash
# 1. Docker 镜像可用
docker pull ghcr.io/topcheer/ggid-gateway:v2.1.0

# 2. GitHub Release 已创建
gh release view v2.1.0 --repo topcheer/ggid

# 3. npm 包已更新
npm view @ggid/sdk version

# 4. PyPI 包已更新
pip index versions ggid
```
