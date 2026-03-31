# Meowlator MVP Gap Closure Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 把项目从“核心链路已通，但验收与上线条件不完整”推进到“可稳定联调、可分工并行、可按阶段验收”的状态。

**Architecture:** 先补齐 `M1` 的真实验收能力，再做 `M2` 的 ONNX 集成方案与最小落地，最后补最小可观测性。执行时使用独立 `git worktree`，由主控 agent 冻结接口和验收命令，子 agent 按文件所有权并行推进，避免共享写入面。

**Tech Stack:** WeChat Mini Program + TypeScript, Go `net/http`, Python training scripts, Docker Compose, MySQL, Redis.

---

## Execution Rules

1. 不在当前脏工作区直接执行实现任务。先建独立 worktree。
2. 主控 agent 只做任务编排、接口冻结、集成验证，不在同一时间与实现 agent 改同一批文件。
3. 每个独立任务都必须走完整闭环：最小失败验证 -> 最小实现 -> 快速验证 -> review -> commit。
4. API 协议或共享类型一旦冻结，当轮并行任务内不得随意再改。

## Agent Layout

1. `Controller`：主控 agent，负责 worktree、任务派发、集成验证、最终 code review。
2. `Worker-A`：前端与小程序测试，只改 `apps/wechat-miniprogram/**`。
3. `Worker-B`：API 冒烟与观测，只改 `services/api/**`、`infra/**`、必要的 `Makefile`。
4. `Worker-C`：云推理与 ONNX，只改 `services/inference/**`、`ml/training/**`。
5. `Reviewer-Spec`：只读，检查任务是否符合目标。
6. `Reviewer-Code`：只读，检查回归风险、边界条件、测试缺口。

## Phase Map

1. `Phase 0`：串行。建立干净执行环境和基线。
2. `Phase 1`：可并行。补齐本地联调与测试验收。
3. `Phase 2`：先串行冻结 ONNX 方案，再由 `Worker-C` 实现。
4. `Phase 3`：在 `Phase 1` 稳定后并行补最小可观测性。

---

### Task 0: Isolated Workspace And Baseline

**Files:**
- Modify if needed: `.gitignore`
- Create: `.worktrees/mvp-gap-closure/` via `git worktree`
- Verify: [Makefile](/Users/dysania/program/meowlator/Makefile:1)

**Agent:** `Controller`

**Step 1: 验证 worktree 目录策略**

Run:

```bash
ls -d .worktrees 2>/dev/null || true
git check-ignore -q .worktrees || true
```

Expected: 若 `.worktrees` 不存在或未忽略，先修复后再继续。

**Step 2: 创建独立 worktree**

Run:

```bash
git worktree add .worktrees/mvp-gap-closure -b codex/mvp-gap-closure
```

Expected: 新 worktree 创建成功，当前实现工作转移到该目录。

**Step 3: 运行基线验证**

Run:

```bash
make test
docker compose -f infra/docker-compose.yml up -d --build
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8081/healthz
```

Expected: `make test` 通过，两个 `healthz` 返回 `{"status":"ok"}`。

**Step 4: 记录基线问题**

输出到执行记录中：
- 当前未完成项清单
- 当前脏工作区不纳入本轮提交的文件
- 后续所有任务在新 worktree 中执行

**Step 5: Commit**

这个任务原则上不产生业务代码提交；若修复了 `.gitignore`，单独提交：

```bash
git add .gitignore
git commit -m "chore: ignore local worktree directory"
```

---

### Task 1: Scripted Local Smoke Flow

**Files:**
- Modify: [Makefile](/Users/dysania/program/meowlator/Makefile:1)
- Create: `tools/smoke_local_flow.sh`
- Modify if needed: [docs/api.md](/Users/dysania/program/meowlator/docs/api.md:1)

**Agent:** `Worker-B`

**Step 1: 先写 smoke 脚本骨架**

脚本覆盖：
- `login`
- `upload-url`
- `upload`
- `finalize`
- `feedback`

**Step 2: 验证脚本在当前实现上至少能跑到明确失败点**

Run:

```bash
bash tools/smoke_local_flow.sh
```

Expected: 如果失败，失败点必须明确定位在具体接口，不允许 silent failure。

**Step 3: 补最小实现**

实现要求：
- 使用现有 API，不新增业务接口
- 参数和签名算法复用现有协议
- 输出每一步的结果摘要，便于人工审阅

**Step 4: 把脚本接入 Makefile**

Run:

```bash
make smoke-local
```

Expected: 一条命令完成本地主链路冒烟。

**Step 5: Commit**

```bash
git add Makefile tools/smoke_local_flow.sh docs/api.md
git commit -m "feat: add local smoke flow for end-to-end validation"
```

**Acceptance**
- `make smoke-local` 可执行
- 能证明上传、finalize、feedback 三段主链路都可用
- 失败时能快速定位接口层问题

---

### Task 2: Mini Program Test Harness

**Files:**
- Modify: [apps/wechat-miniprogram/package.json](/Users/dysania/program/meowlator/apps/wechat-miniprogram/package.json:1)
- Create: `apps/wechat-miniprogram/tests/`
- Create: `apps/wechat-miniprogram/tests/index-page.test.ts`
- Create: `apps/wechat-miniprogram/tests/result-page.test.ts`
- Modify if needed: `apps/wechat-miniprogram/tsconfig.json`

**Agent:** `Worker-A`

**Step 1: 建立最小测试框架**

优先使用小程序官方生态工具；目标不是视觉回归，而是页面状态和关键交互验证。

**Step 2: 先写失败测试**

覆盖至少以下场景：
- 首页可触发拍照/相册入口
- 结果页能消费 `lastResult`
- 反馈弹层可打开、选择标签、触发提交逻辑

**Step 3: 运行测试确认失败**

Run:

```bash
cd apps/wechat-miniprogram && npm test
```

Expected: 新增测试最初失败，失败原因与缺失的测试支撑一致。

**Step 4: 补最小实现与脚本**

补充内容：
- 测试脚本
- 测试依赖
- 必要的页面适配，避免测试无法挂载

**Step 5: 再次验证并提交**

Run:

```bash
cd apps/wechat-miniprogram && npm test
cd apps/wechat-miniprogram && npm run typecheck
```

Commit:

```bash
git add apps/wechat-miniprogram/package.json apps/wechat-miniprogram/tsconfig.json apps/wechat-miniprogram/tests
git commit -m "test(miniprogram): add page interaction test harness"
```

**Acceptance**
- 小程序不再只有 `typecheck`
- 首页、结果页、反馈弹层有自动化测试
- 后续 UI 重构有最小回归保护

---

### Task 3: API Smoke And Persistence Verification

**Files:**
- Modify: [services/api/internal/api/handlers_test.go](/Users/dysania/program/meowlator/services/api/internal/api/handlers_test.go:1)
- Create: `services/api/internal/api/flow_test.go`
- Modify if needed: [infra/docker-compose.yml](/Users/dysania/program/meowlator/infra/docker-compose.yml:1)

**Agent:** `Worker-B`

**Step 1: 写失败测试**

覆盖：
- `POST /v1/auth/wechat/login`
- `POST /v1/samples/upload-url`
- `POST /v1/inference/finalize`
- `POST /v1/feedback`

**Step 2: 跑局部测试**

Run:

```bash
cd services/api && go test ./internal/api -run TestFlow -v
```

Expected: 初始失败，能看出缺的是链路级测试而不是业务能力缺失。

**Step 3: 用最小代码补齐测试支撑**

原则：
- 优先复用内存仓储和现有 fake client
- 不为了测试新增业务分支

**Step 4: 跑完整 Go 测试**

Run:

```bash
cd services/api && go test ./...
```

**Step 5: Commit**

```bash
git add services/api/internal/api/handlers_test.go services/api/internal/api/flow_test.go
git commit -m "test(api): add end-to-end flow coverage"
```

**Acceptance**
- API 有主链路级测试，不只剩单点单元测试
- 能验证 feedback 回流逻辑
- 后续 ONNX 接入前可作为回归基线

---

### Task 4: ONNX Integration Decision Spike

**Files:**
- Create: `docs/plans/2026-03-31-onnx-spike-notes.md`
- Inspect: [services/inference/internal/app/model.go](/Users/dysania/program/meowlator/services/inference/internal/app/model.go:1)
- Inspect: [ml/training/scripts/export_onnx.py](/Users/dysania/program/meowlator/ml/training/scripts/export_onnx.py:1)

**Agent:** `Controller` then `Worker-C`

**Step 1: 先做 0.5 天 spike，不直接改主逻辑**

目标：
- 确认 Go 侧 ONNX runtime 选型
- 确认模型文件加载方式
- 确认缺模型时的 fail-fast 行为

**Step 2: 输出结论**

结论至少包含：
- 选用的 Go 运行时方案
- 是否需要 cgo / 动态库
- 本地和容器中的部署要求
- 最小 smoke 命令

**Step 3: 通过后冻结实现边界**

冻结项：
- 运行时依赖
- 配置字段
- 错误处理策略

**Step 4: Commit**

```bash
git add docs/plans/2026-03-31-onnx-spike-notes.md
git commit -m "docs: record onnx integration spike decision"
```

**Acceptance**
- 不再停留在“需要评估”
- ONNX 落地路径明确，可直接进入实现

---

### Task 5: Cloud ONNX Runtime Implementation

**Files:**
- Modify: [services/inference/cmd/inference/main.go](/Users/dysania/program/meowlator/services/inference/cmd/inference/main.go:1)
- Create: `services/inference/internal/app/predictor.go`
- Create: `services/inference/internal/app/heuristic_predictor.go`
- Create: `services/inference/internal/app/onnx_predictor.go`
- Modify or split: [services/inference/internal/app/model.go](/Users/dysania/program/meowlator/services/inference/internal/app/model.go:1)
- Create: `services/inference/internal/app/onnx_predictor_test.go`
- Modify if needed: `services/inference/go.mod`

**Agent:** `Worker-C`

**Step 1: 先写失败测试**

覆盖：
- 模型文件缺失时服务明确失败
- ONNX predictor 成功加载时返回有效 `InferenceResult`
- 启发式模式仍可作为 fallback 或 feature flag

**Step 2: 跑测试确认失败**

Run:

```bash
cd services/inference && go test ./internal/app -run TestONNX -v
```

**Step 3: 实现双 predictor 结构**

要求：
- 保留启发式实现
- ONNX 模式必须是真加载模型，不允许假装接入
- 配置显式控制模式切换

**Step 4: 跑服务级验证**

Run:

```bash
cd services/inference && go test ./...
make test-go
```

**Step 5: Commit**

```bash
git add services/inference/cmd/inference/main.go services/inference/internal/app services/inference/go.mod
git commit -m "feat(inference): add cloud onnx runtime predictor"
```

**Acceptance**
- 云侧至少一条路径切到 ONNX 真推理
- 模型缺失时行为符合预期
- 启发式模式仍可用于本地或回退

---

### Task 6: Minimal Observability

**Files:**
- Modify: [services/api/internal/api/handlers.go](/Users/dysania/program/meowlator/services/api/internal/api/handlers.go:102)
- Modify: [services/api/cmd/api/main.go](/Users/dysania/program/meowlator/services/api/cmd/api/main.go:15)
- Modify: [docs/launch_metrics.md](/Users/dysania/program/meowlator/docs/launch_metrics.md:1)
- Modify if needed: [infra/docker-compose.yml](/Users/dysania/program/meowlator/infra/docker-compose.yml:1)

**Agent:** `Worker-B`

**Step 1: 写最小验证**

先定义必须可观测的 4 个量：
- `finalize` 延迟
- 错误率
- `fallbackUsed` 比例
- copy 生成失败/超时

**Step 2: 选择实现路径**

优先：
- Prometheus `/metrics`

回退：
- 结构化日志 + 聚合脚本

不要两条路一起做。

**Step 3: 实现并验证**

Run:

```bash
cd services/api && go test ./...
docker compose -f infra/docker-compose.yml up -d --build
```

Expected: 能稳定产出上述 4 个量，且不影响主链路。

**Step 4: 更新文档**

对齐 [docs/KPI_BASELINE.md](/Users/dysania/program/meowlator/docs/KPI_BASELINE.md:79) 和 [docs/release_sop.md](/Users/dysania/program/meowlator/docs/release_sop.md:5) 中需要观察的指标。

**Step 5: Commit**

```bash
git add services/api/internal/api/handlers.go services/api/cmd/api/main.go docs/launch_metrics.md infra/docker-compose.yml
git commit -m "feat(api): add minimal runtime observability"
```

**Acceptance**
- 可以支撑灰度观察，不再只靠人工读零散日志
- 指标和发布 SOP 对齐

---

### Task 7: Final Integration Gate

**Files:**
- Verify all modified files from Tasks 1-6
- Update if needed: [docs/implementation_nodes.md](/Users/dysania/program/meowlator/docs/implementation_nodes.md:1)

**Agent:** `Controller` + `Reviewer-Spec` + `Reviewer-Code`

**Step 1: 跑全量验证**

Run:

```bash
make test
make smoke-local
docker compose -f infra/docker-compose.yml up -d --build
```

**Step 2: 做规格评审**

核对：
- `M1` 是否已真实可验收
- `M2` 是否已明确并至少云侧落地
- 可观测性是否满足最小发布观察

**Step 3: 做代码评审**

重点查：
- 回归风险
- 错误处理
- 测试缺口
- 文档与实现是否漂移

**Step 4: 记录实现节点**

新增 `N014+`，补齐本轮实现节点与 commit SHA。

**Step 5: 集成收尾**

如需继续发布流程，再进入 PR、灰度、合并，不停在“本地已提交”。

**Acceptance**
- `make test` 通过
- `make smoke-local` 通过
- 规格 review 无阻塞项
- 代码 review 无阻塞项

---

## Parallel Execution Order

1. `Controller` 完成 `Task 0`
2. `Worker-B` 执行 `Task 1`
3. `Worker-A` 执行 `Task 2`
4. `Worker-B` 执行 `Task 3`
5. `Controller` 完成 `Task 4`
6. `Worker-C` 执行 `Task 5`
7. `Worker-B` 执行 `Task 6`
8. `Controller` 执行 `Task 7`

## Do Not Parallelize

1. `Task 0` 不能并行
2. `Task 4` 与 `Task 5` 不能并行
3. `Task 7` 必须等前面都完成
4. 任一共享协议改动必须先由 `Controller` 冻结后再并行

## Reviewer Checklist

1. Spec review 只回答两件事：有没有漏做；有没有多做。
2. Code review 只回答四件事：回归、边界、测试、可维护性。
3. 发现阻塞问题时，回到原实现 agent 修复，不由 reviewer 直接改。

## Done Definition

1. 本轮不再以“代码看起来差不多”为完成标准。
2. 完成标准是：有自动化测试、有 smoke 验证、有 review、有实现节点记录。
3. 若要宣称“模型已上线”，还必须满足 [docs/project_manual.md](/Users/dysania/program/meowlator/docs/project_manual.md:275) 和 [docs/KPI_BASELINE.md](/Users/dysania/program/meowlator/docs/KPI_BASELINE.md:79) 的门禁条件。

