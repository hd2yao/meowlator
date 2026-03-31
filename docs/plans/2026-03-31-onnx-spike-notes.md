# 2026-03-31 ONNX Integration Spike Notes

## 结论

1. Go 侧运行时选型冻结为 `github.com/yalue/onnxruntime_go`，先做 CPU 推理，不引入 GPU provider。
2. `ONNX` 模式必须是显式配置模式，不做静默回退。`ONNX` 所需动态库或模型文件缺失时，推理服务启动即失败。
3. 当前 `services/inference` 的容器基线不适合直接承载 `ONNX` 运行时。`Task 5` 需要把推理服务镜像从 `alpine` 切到 `debian:bookworm-slim` 或等价 `glibc` 基线。
4. 当前仓库没有可直接加载的 `.onnx` 实物，只有导出脚本和模型注册元数据。`Task 5` 必须把模型文件挂载/复制到服务可读路径，不能假设仓库内自带产物。

## 现状证据

### 1. 当前云推理仍是启发式，不是真推理

- 当前云推理入口直接依赖 `app.Model`，`POST /v1/inference/predict` 只调用 `model.Predict(...)`。
  - [services/inference/internal/api/handlers.go](/Users/dysania/.config/superpowers/worktrees/meowlator/codex-mvp-gap-closure-plan/services/inference/internal/api/handlers.go:10)
- 当前 `Model` 的输出来自 `imageKey|sceneTag` 哈希、固定意图集合、先验混合和规则派生状态，不读取图片，也不加载模型文件。
  - [services/inference/internal/app/model.go](/Users/dysania/.config/superpowers/worktrees/meowlator/codex-mvp-gap-closure-plan/services/inference/internal/app/model.go:65)

### 2. ONNX 导出物的真实形态

- 训练导出脚本把 `MobileNetV3 Small` 的分类头替换为 `num_classes=8` 的线性层。
  - [ml/training/scripts/export_onnx.py](/Users/dysania/.config/superpowers/worktrees/meowlator/codex-mvp-gap-closure-plan/ml/training/scripts/export_onnx.py:12)
- 导出输入名固定为 `input`，输出名固定为 `logits`，默认 `opset=17`。
  - [ml/training/scripts/export_onnx.py](/Users/dysania/.config/superpowers/worktrees/meowlator/codex-mvp-gap-closure-plan/ml/training/scripts/export_onnx.py:54)
- 输入张量形状是 `[batch, 3, input_size, input_size]`。当前 checkpoint 默认 `input_size=224`。
  - [ml/training/scripts/export_onnx.py](/Users/dysania/.config/superpowers/worktrees/meowlator/codex-mvp-gap-closure-plan/ml/training/scripts/export_onnx.py:28)
- 输出只有单个 `logits` 头，没有 `state` 头、没有 `copy` 头。
  - [ml/training/scripts/export_onnx.py](/Users/dysania/.config/superpowers/worktrees/meowlator/codex-mvp-gap-closure-plan/ml/training/scripts/export_onnx.py:58)

### 3. 仓库里的模型资产状态

- 模型注册目录只有元数据，没有 `.onnx` 实物。
  - [ml/model-registry/mobilenetv3-small-int8-v1.json](/Users/dysania/.config/superpowers/worktrees/meowlator/codex-mvp-gap-closure-plan/ml/model-registry/mobilenetv3-small-int8-v1.json:1)
  - [ml/model-registry/mobilenetv3-small-int8-v2.json](/Users/dysania/.config/superpowers/worktrees/meowlator/codex-mvp-gap-closure-plan/ml/model-registry/mobilenetv3-small-int8-v2.json:1)
- 注册表只记录 `artifact_uri`，例如 `s3://...onnx`，说明服务集成时必须解决模型下发或挂载问题。
  - [ml/model-registry/mobilenetv3-small-int8-v1.json](/Users/dysania/.config/superpowers/worktrees/meowlator/codex-mvp-gap-closure-plan/ml/model-registry/mobilenetv3-small-int8-v1.json:4)
- `Makefile` 已有导出命令，但产物默认落在本地训练目录，不会自动进入 `services/inference`。
  - [Makefile](/Users/dysania/.config/superpowers/worktrees/meowlator/codex-mvp-gap-closure-plan/Makefile:62)

## 运行时方案冻结

### 1. Go 运行时

选型：`github.com/yalue/onnxruntime_go`

原因：

1. 当前项目是 Go 服务，不引入 Python sidecar，优先选择 Go 原生调用链。
2. 该库直接包装官方 ONNX Runtime C API，适合本项目“服务启动时加载模型，进程内复用 session”的模式。
3. 该库要求显式指定 ONNX Runtime 共享库路径，并在初始化前完成环境准备，符合本项目对 fail-fast 的要求。

外部依据：

1. `onnxruntime_go` README 明确要求 `cgo` 和可用的 ONNX Runtime shared library，并建议显式设置 shared library path。
   - [yalue/onnxruntime_go README](https://github.com/yalue/onnxruntime_go)
2. ONNX Runtime 官方安装文档明确区分语言绑定与底层运行时库，Linux 下需要提供动态库和依赖环境。
   - [Install ONNX Runtime](https://onnxruntime.ai/docs/install/)

### 2. 容器与系统前提

冻结要求：

1. `Task 5` 的推理服务构建必须开启 `cgo`。
2. 推理服务运行镜像切到 `glibc` 基线，避免继续使用当前 `alpine`/`musl` 路线。
3. 容器内必须提供：
   - ONNX Runtime shared library
   - 目标 `.onnx` 模型文件
   - 可选的 `intent_priors.json`

当前需要调整的地方：

1. 当前推理服务 Dockerfile 采用 `golang:1.22-alpine` -> `alpine:3.20` 双阶段构建，不适合作为 ONNX runtime 基线。
   - [services/inference/Dockerfile](/Users/dysania/.config/superpowers/worktrees/meowlator/codex-mvp-gap-closure-plan/services/inference/Dockerfile:1)
2. 当前 compose 只传了 `MODEL_PRIORS_PATH`，还没有模型路径或 runtime 动态库路径配置。
   - [infra/docker-compose.yml](/Users/dysania/.config/superpowers/worktrees/meowlator/codex-mvp-gap-closure-plan/infra/docker-compose.yml:19)

### 3. 配置字段冻结

`Task 5` 采用以下配置边界：

1. `INFERENCE_PREDICTOR_MODE=heuristic|onnx`
2. `ONNX_MODEL_PATH=/app/models/<model>.onnx`
3. `ONNXRUNTIME_SHARED_LIBRARY_PATH=/app/lib/libonnxruntime.so`
4. `MODEL_PRIORS_PATH` 保持现有语义，作为可选后处理输入

行为冻结：

1. `heuristic` 模式：允许没有 ONNX 运行时和模型文件。
2. `onnx` 模式：若共享库、模型文件、session 初始化任一失败，服务直接返回启动错误并退出。
3. `onnx` 模式下不做“默默退回 heuristic”。需要切回启发式时，只能显式改配置。

## 输出映射冻结

当前导出的 ONNX 模型只给出 8 类意图 `logits`，因此服务在 ONNX 模式下仍需补齐业务输出：

1. `intentTop3`
   - 对 `logits` 做 softmax，取 top-3，映射到当前 8 个 `IntentLabel`
   - 标签集合沿用当前服务定义
   - [services/inference/internal/app/model.go](/Users/dysania/.config/superpowers/worktrees/meowlator/codex-mvp-gap-closure-plan/services/inference/internal/app/model.go:11)
2. `confidence`
   - 取 top-1 softmax 概率
3. `state`
   - 现有训练导出没有三轴状态头
   - 第一阶段保持兼容做法：仍由服务端基于 top intent 或概率分布做确定性派生，不在 `Task 5` 扩训模型结构
4. `source`
   - 固定 `"CLOUD"`
5. `evidence`
   - 第一阶段保留规则文案，标识 `"云端 ONNX 复判"` 与 `"视觉 logits 排序"`
6. `copyStyleVersion`
   - 继续固定 `"v1"`

这意味着 `Task 5` 的目标是“云侧真视觉分类接入”，不是一次性把 `state` 也升级成多头神经网络。

## 最小 smoke 方案

### 1. Python 级模型 smoke

仓库现有 `ml/training/requirements.txt` 已包含 `onnx` 和 `onnxruntime`，足以做最小验证。

建议命令：

```bash
cd ml/training
python3 scripts/export_onnx.py \
  --checkpoint ./artifacts/mobilenetv3-small-v2/mobilenetv3-small-v2.pt \
  --output ./artifacts/mobilenetv3-small-v2/mobilenetv3-small-v2.onnx \
  --quantize-int8

python3 - <<'PY'
import numpy as np
import onnxruntime as ort

session = ort.InferenceSession("./artifacts/mobilenetv3-small-v2/mobilenetv3-small-v2.onnx")
output = session.run(["logits"], {"input": np.zeros((1, 3, 224, 224), dtype=np.float32)})[0]
print(output.shape)
PY
```

预期：

1. 导出成功
2. `InferenceSession` 可加载模型
3. 输出 shape 为 `(1, 8)`

### 2. Go 服务级 smoke

`Task 5` 完成后最小启动命令冻结为：

```bash
cd services/inference
INFERENCE_PREDICTOR_MODE=onnx \
ONNX_MODEL_PATH=/absolute/path/to/model.onnx \
ONNXRUNTIME_SHARED_LIBRARY_PATH=/absolute/path/to/libonnxruntime.so \
go run ./cmd/inference
```

最小接口验证：

```bash
curl -s http://127.0.0.1:8081/v1/inference/predict \
  -H 'Content-Type: application/json' \
  -d '{"imageKey":"samples/u1/s1.jpg","sceneTag":"UNKNOWN"}'
```

预期：

1. 服务启动日志出现 ONNX session 初始化成功信息
2. 缺模型或缺动态库时，进程启动失败
3. 同一输入在不同模型版本下可以产生不同 top-3

## Task 5 实现边界

需要新增/拆分：

1. `services/inference/internal/app/predictor.go`
2. `services/inference/internal/app/heuristic_predictor.go`
3. `services/inference/internal/app/onnx_predictor.go`
4. `services/inference/internal/app/onnx_predictor_test.go`
5. `services/inference/internal/config/config.go`
6. `services/inference/cmd/inference/main.go`

不在 `Task 5` 范围内的内容：

1. 自动下载模型
2. GPU provider
3. 多头状态网络
4. 在线热更新模型

## 验收口径

只有同时满足以下条件，才算 ONNX 真推理接入完成：

1. 代码路径真实创建 ONNX Runtime session 并执行张量推理
2. `onnx` 模式下模型文件不可用时服务失败
3. `heuristic` 模式仍可独立运行
4. `go test ./...` 覆盖 ONNX 加载失败与成功路径
