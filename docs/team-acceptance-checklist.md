# GGID 交付验收标准 Checklist

> 本文件由团队 retrospective 讨论产生，所有成员必须遵守。

## 任务认领规则

### Frontend 认领前
1. `grep -rn "端点路径" services/*/internal/server/http.go` 确认后端端点已注册
2. 端点不存在 → DM backend 约定 API 契约（URL + request/response JSON）→ 写入 kanban 任务描述
3. 端点存在 → 开始实现
4. 后端未就绪的页面：顶部加 `DEMO DATA` 横幅 + empty fallback

### Backend 交付前自检
1. **DB-backed**：repo 使用 pgxpool.Pool，非内存 map
2. **调用链完整**：从 main.go → mux.HandleFunc → handler → repo → DB 全链路可调用
3. **真实实现**：无 log.Printf 占位、无 "in production would" 注释
4. **测试**：≥3 个单测覆盖核心路径
5. **接线**：handler 注册在 mux 中，main.go 调用链完整（如 SMS provider 的 SetSMSSender）

### 研究文档（researcher/IAMExpert/techwriter）
1. API Design 部分增加"前置条件检查"：哪些 GGID 现有端点可用、哪些需新建
2. 每个 backlog 项包含 Definition of Done（验收标准）
3. 末尾增加"反模式禁令"：明确禁止 log.Printf/内存 map/alg=none 等占位

## 验收流程

1. 实现者完成 → 标记 REVIEW（不是 DONE）
2. arch 查 idle 成员 → DM 验收（含 git pull 提醒）
3. 验收人按 checklist 逐项检查
4. 全部通过 → 标 DONE
5. 任一不通过 → REJECT + 列出具体问题 → 实现者修复后重新提交

## Kanban 任务描述格式

每个 TODO 必须包含：
```
| ID | Task | Priority | Scope | 端点路径 | Acceptance |
```

- 端点路径：HTTP method + URL（前端和 backend 用完全一致的路径）
- Acceptance：curl 命令（验收方可直接运行验证）

## 团队成员改进承诺（本 session retrospective）

- **backend**：交付前自检调用链 + DB-backed + ≥3 测试
- **frontend**：开工前 grep 确认端点 + API 契约先行
- **IAMExpert**：审计前置（IN_PROGRESS 时预检）+ 验收标准文档化
- **docs/techwriter**：验收更果断 REJECT + 部署后验收（rebuild 再验）
- **researcher**：研究文档增加端点验证清单 + 自主巡检已实现功能
- **arch**：任务描述附带端点路径 + 及时验收分配
