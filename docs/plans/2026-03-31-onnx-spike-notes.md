# 2026-03-31 ONNX Integration Spike Notes

## Goal

把 `services/inference` 从“启发式假推理”推进到“可显式切换的 ONNX 真推理”，并冻结实现边界，避免 `Task 5` 再回头改协议和部署方式。

## Current State

1. 云推理 HTTP 协议已经稳定，只暴露 `POST /v1/inference/predict`，请求体为 `imageKey + sceneTag`，返回 `result`。
   - 代码位置：`services/inference/internal/api/handlers.go`
2. 当前 `services/inference/internal/app/model.go` 仍是哈希 + 先验融合的启发式实现，不加载 `.onnx`，也不做图像前处理。
3. 配置面当前只有：
   - `INFERENCE_ADDR`
   - `MODEL_PRIORS_PATH`
   - 代码位置：`services/inference/internal/config/config.go`
4. 当前容器镜像是 `alpine` 两阶段构建，不包含 ONNX Runtime 共享库，也没有模型挂载约定。
   - 代码位置：`services/inference/Dockerfile`
   - 代码位置：`infra/docker-compose.yml`
5. 仓库内没有现成 `.onnx` 实物，只有模型注册元数据；因此 ONNX smoke 需要外部模型文件或先本地导出。
   - 代码位置：`ml/model-registry/*.json`
   - `find` 结果：仓库内无 `.onnx` 文件

## Training Artifact Findings

1. `ml/training/scripts/export_onnx.py` 已支持从 `MobileNetV3 Small` checkpoint 导出 ONNX，并可选做动态 INT8 量化。
2. 导出脚本固定：
   - 输入名：`input`
   - 输出名：`logits`
   - 输入张量：`1 x 3 x input_size x input_size`
   - 默认 `input_size` 来自 checkpoint，缺省为 `224`
3. 业务侧还原逻辑需要从 logits 推导：
   - `intentTop3`
   - `confidence`
   - `state`
   - `evidence`
   - `copyStyleVersion`
4. 当前训练依赖已经声明并且本机 Python 环境可见：
   - `torch`
   - `onnx`
   - `onnxruntime`
   - 代码位置：`ml/training/requirements.txt`

## Runtime Decision

### Selected Go Runtime

选型：`github.com/yalue/onnxruntime_go`

理由：

1. 它是对 Microsoft ONNX Runtime C API 的 Go 包装，适合当前 Go 推理服务直接加载 `.onnx`。
2. 它要求显式设置 ONNX Runtime 共享库路径，便于我们把“缺库/缺模型时启动失败”做成 fail-fast。
3. 它支持直接创建 session 和张量，足够覆盖 MVP 的单输入单输出 CPU 推理路径。

参考来源：

1. `yalue/onnxruntime_go` README：<https://github.com/yalue/onnxruntime_go>
2. ONNX Runtime C API 文档：<https://onnxruntime.ai/docs/get-started/with-c.html>

### Hard Requirements

1. 需要 `cgo`。
2. 需要随服务提供 ONNX Runtime 共享库。
3. 需要模型文件真实存在于容器或本地文件系统。

## Deployment Decision

### Container Baseline

结论：`Task 5` 不应继续使用当前 `alpine` 运行时镜像。

原因：

1. 选定的 Go wrapper 依赖 ONNX Runtime 共享库。
2. 当前镜像没有任何共享库拷贝或挂载策略。
3. 基于 wrapper README 和官方 release 的典型分发方式，Linux 侧通常直接消费官方共享库；在这种路径下，继续用 `alpine/musl` 风险高，容器内库兼容性不可控。

这里的 `alpine` 风险判断是基于官方文档和 wrapper 的部署要求做出的工程推断，不是当前仓库内已跑通的事实。

冻结决策：

1. `services/inference/Dockerfile` 切到 `debian:bookworm-slim` 或同类 glibc 基线。
2. 通过环境变量传入：
   - ONNX Runtime 共享库路径
   - ONNX 模型路径
3. `docker-compose` 为 inference 服务增加模型和共享库的挂载约定。

## Config Freeze

`Task 5` 统一新增以下配置字段：

1. `INFERENCE_PREDICTOR`
   - 可选值：`heuristic`、`onnx`
   - 默认：`heuristic`
2. `ONNX_MODEL_PATH`
   - 当 `INFERENCE_PREDICTOR=onnx` 时必填
3. `ONNX_SHARED_LIBRARY_PATH`
   - 当 `INFERENCE_PREDICTOR=onnx` 时必填
4. `ONNX_INPUT_SIZE`
   - 默认 `224`
   - 用于前处理与模型导出尺寸对齐

冻结策略：

1. `heuristic` 模式继续保留，作为显式 feature flag。
2. `onnx` 模式下禁止 silent fallback。
3. 如果模型文件、共享库、session 初始化任一失败，服务启动直接 `log.Fatal`。

## External Contract Freeze

`Task 5` 不改以下外部接口：

1. `POST /v1/inference/predict`
2. 请求体：`{"imageKey":"...","sceneTag":"..."}`
3. 返回体：`{"result":{...}}`
4. API 服务对 inference 的 HTTP client 协议保持不变。

这样 ONNX 接入只影响 `services/inference/internal/app` 和启动配置，不波及 `services/api`。

## Output Mapping Freeze

当前导出的 ONNX 模型只有单个 `logits` 输出，因此 `Task 5` 需要在服务内做最小后处理，还原现有业务结构。

冻结规则：

1. `intentTop3`
   - 对 `logits` 做 softmax
   - 取 top-3
   - 标签集合沿用当前 `services/inference/internal/app/model.go` 中的 8 个 `IntentLabel`
2. `confidence`
   - 使用 top-1 softmax 概率
3. `state`
   - 第一阶段不扩展多头模型
   - 继续由服务端做确定性派生，输入只使用 top intent 和 confidence bucket
4. `source`
   - 固定 `"CLOUD"`
5. `evidence`
   - 第一阶段保留规则文案，至少包含 `"云端 ONNX 复判"` 和 `"视觉 logits 排序"`
6. `copyStyleVersion`
   - 固定 `"v1"`

这意味着 `Task 5` 的目标是先把“云侧真视觉分类”接进来，不在这一轮扩成多头状态网络。

## Minimal Smoke Path

### Step 1: 产出或准备模型

如果已有 checkpoint，可先导出：

```bash
cd ml/training
python3 scripts/export_onnx.py \
  --checkpoint ./artifacts/<run>/<model>.pt \
  --output ./artifacts/<run>/<model>.onnx \
  --quantize-int8
```

如果没有 checkpoint，则需要先提供一个现成 `.onnx` 文件；当前仓库本身不包含该文件。

### Step 2: 启动 inference 服务

```bash
cd services/inference
INFERENCE_PREDICTOR=onnx \
ONNX_MODEL_PATH=/absolute/path/to/model.onnx \
ONNX_SHARED_LIBRARY_PATH=/absolute/path/to/libonnxruntime.so \
go run ./cmd/inference
```

### Step 3: 调用最小预测接口

```bash
curl -X POST http://127.0.0.1:8081/v1/inference/predict \
  -H 'Content-Type: application/json' \
  -d '{"imageKey":"samples/cat-001/sample-123.jpg","sceneTag":"UNKNOWN"}'
```

预期：

1. 服务能成功启动并输出 ONNX session 初始化日志。
2. 预测接口返回有效 `result`。
3. 如果 `ONNX_MODEL_PATH` 或 `ONNX_SHARED_LIBRARY_PATH` 错误，服务在启动阶段明确失败。

## Task 5 Implementation Boundary

`Task 5` 的最小实现范围冻结如下：

1. 抽出 `Predictor` 接口，拆分 `heuristic` 与 `onnx` 两个实现。
2. `main.go` 按 `INFERENCE_PREDICTOR` 选择 predictor，并在 `onnx` 模式下 fail-fast。
3. `onnx` predictor 真实加载 `.onnx`，不允许伪接入。
4. 单测至少覆盖：
   - `onnx` 模式缺模型/缺库时报错
   - predictor 能返回有效 `InferenceResult`
   - `heuristic` 模式仍可工作
5. 暂不在本任务内做 GPU、批量推理、多模型热切换。

## Decision Summary

1. 方案已冻结，可以直接进入 `Task 5`。
2. 运行时选型为 `onnxruntime_go + official onnxruntime shared library`。
3. 配置策略为显式 `heuristic|onnx` 切换。
4. 错误策略为 `onnx` 模式启动即 fail-fast，不做 silent fallback。
5. 容器策略应从 `alpine` 转到更适合共享库部署的 glibc 基线。
