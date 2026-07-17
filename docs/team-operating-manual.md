# GGID 团队协作手册

## 角色定义

### 研究类（持续增长 backlog）
- **researcher**（Mac-Studio 远程）— IAM 趋势研究、竞品分析、设计文档、kanban backlog 填充
- **IAMExpert**（macmini）— 安全审计、密码学实现、E2E 验收、设计文档

### 实施类（从 kanban 取任务执行）
- **backend**（macmini）— Go services 实现 + 测试
- **frontend**（macmini）— Console UI 实现 + tsc 验证
- **docs&devops**（macmini）— 文档质量、部署配置
- **techwriter**（macmini）— API 文档、端到端验证

### 协调类
- **arch&pm**（macmini）— kanban 维护、验收、架构决策、自己也做 pkg/sdk/deploy 任务

## 通信规则

### lanchat 礼仪
1. **收到任务**：回一条简短的 "收到，开始做 XXX"（让发送方知道你已开始）
2. **任务完成**：回结果摘要 + commit hash（让 arch 验收）
3. **被阻塞**：回阻塞原因 + 需要什么帮助
4. **禁止**：无意义的确认（"好的""谢谢""明白"）、重复进度汇报、催促

### 消息频率
- 每个成员每轮最多发 2 条消息（1 收到确认 + 1 完成报告）
- arch 每轮最多 1 次 broadcast
- DM 只发给需要知道的人

## Kanban 工作流

### 实施类成员（backend/frontend/docs）
1. 每轮开始：`read_file docs/kanban.md`
2. 从 TODO 取自己角色的最高优先级任务
3. 改状态 IN_PROGRESS + 填 assignee + git commit kanban
4. 开始实现
5. 完成后：改状态 DONE + 填 commit hash + git commit kanban
6. DM arch："XXX 完成，commit XXX"
7. 回到步骤 1 取下一个

### 研究类成员（researcher/IAMExpert）
1. 每轮：研究一个 IAM 主题或审计一个模块
2. 发现 gap → 添加到 kanban TODO（填 ID/priority/scope/acceptance）
3. 设计文档 → docs/architecture/
4. git commit + push（researcher 在远程机器，必须 push）
5. DM arch："新增 N 项 backlog，commit XXX"

### arch
1. 读 kanban → 检查状态
2. 验收 DONE 项（build + test + E2E）
3. 从研究报告填充新 TODO
4. 推动阻塞项
5. 做 pkg/sdk/deploy 任务
6. 每轮 broadcast 简报

## 远程协作（researcher 在 Mac-Studio）

### Git 同步规则
- researcher 工作目录：`/Volumes/new/ggai/ggid`（Mac-Studio）
- 主仓库：`/Users/zhanju/ggai/ggid`（macmini）
- **researcher 每次完成工作后必须 `git push`**
- arch 定期 `git pull` 同步 researcher 的改动
- researcher 研究/设计文档放在 docs/，不修改 services/ 或 console/
- 如果 researcher 需要看最新代码：`git pull` 同步

### 分工避免冲突
- researcher 只写 docs/research/ 和 docs/architecture/ 下的新文件
- 不修改已有文件（避免与 macmini 成员冲突）
- kanban 更新通过 git push/pull 同步

## Backlog 持续增长机制

researcher 的核心职责：**保持 backlog 增长**
- 每轮研究一个 IAM 主题
- 输出：设计文档 + kanban TODO 项
- 确保实施类成员永远不会"没有任务可做"

研究方向轮转：
1. 国密合规深化
2. 零信任架构
3. ITDR/UEBA
4. IGA/PAM
5. 云原生 IAM/CIEM
6. 身份联邦/OIDC
7. AI 驱动安全
8. 竞品分析（Auth0/Okta/Keycloak/Ping）
