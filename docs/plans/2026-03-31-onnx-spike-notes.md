# 2026-03-31 ONNX Integration Spike Notes

## 目标

冻结 `services/inference` 接入 ONNX 真推理的最小实现边界，避免 `Task 5` 一边试 runtime 一边改主逻辑。

## 当前现状

1. 当前云推理调用链是 `cmd/inference/main.go -> internal/api/handlers.go -> internal/app/model.go`。
2. `internal/app/model.go` 只有一个 `Model`，内部直接做哈希 + priors 融合，没有 predictor 抽象。
3. 当前配置只有 `INFERENCE_ADDR` 和 `MODEL_PRIORS_PATH`，没有 ONNX 模型路径、runtime 动态库路径，或图片根目录配置。
4. 当前 API 上传接口把文件落盘到 `/tmp/meowlator/uploads/<sampleId>.jpg`，但 inference 服务收到的只有 `imageKey`，例如 `samples/<userId>/<sampleId>.jpg`。
5. 当前 `api` 与 `inference` 容器之间没有共享 uploads volume，因此即使 ORT session 可用，inference 侧也无法按现有协议读取真实图片文件。
6. 当前 `services/inference/Dockerfile` 使用 `golang:1.22-alpine` + `alpine:3.20`，这更适合纯 Go 二进制，不适合第一版带 cgo + glibc 动态库的 ORT 运行时。

## 结论

### 1. Go 运行时选型

选用 `github.com/yalue/onnxruntime_go` 作为 Go 侧 ONNX 绑定。

原因：

1. 它是现成的 Go 封装，支持通过 `SetSharedLibraryPath(...)` 显式指定 ORT 动态库路径，再初始化运行环境和 Session。
2. 它不要求把 ONNX Runtime 源码编进项目，但要求本机或容器里存在与其头文件版本匹配的共享库。
3. 该库 README 明确要求 Go 开启 cgo，并提供正确版本的 `onnxruntime` 共享库路径。

### 2. 版本与部署策略

冻结第一版策略：

1. Go 包固定到一个明确版本。
2. 容器里同时放入与该包头文件版本匹配的 `libonnxruntime` 动态库。
3. 不在第一版尝试 GPU provider，只做 CPU 路径。

当前检查结果：

1. `go list -m -versions github.com/yalue/onnxruntime_go` 可见版本到 `v1.27.0`。
2. `v1.27.0` 的 README 说明它当前使用 `onnxruntime` C API `1.24.1` 头文件，因此共享库也应匹配 `1.24.1`。
3. ONNX Runtime 官方安装文档说明各语言和平台组合需要按安装矩阵选包；Linux 动态库依赖环境变量和系统库正确配置。

冻结决定：

1. `Task 5` 先按 `github.com/yalue/onnxruntime_go v1.27.0` 设计。
2. 共享库按 `onnxruntime 1.24.1` CPU Linux x64/arm64 对应发布包准备。
3. 若后续升级 wrapper 版本，必须同步验证共享库版本，不做“只升 Go 包不升 so”的半升级。

### 3. 配置边界

`Task 5` 新增以下配置字段：

1. `INFERENCE_PREDICTOR_MODE`
   - 可选值：`heuristic`、`onnx`
   - 默认：`heuristic`
2. `ONNX_MODEL_PATH`
   - `onnx` 模式必填
3. `ONNX_SHARED_LIB_PATH`
   - `onnx` 模式必填
4. `INFERENCE_UPLOAD_ROOT`
   - 默认 `/tmp/meowlator/uploads`
   - 用于根据 `sampleId` 还原本地图片路径
5. `MODEL_PRIORS_PATH`
   - 保留；是否继续做 priors 融合，在 `Task 5` 中只允许作为 ONNX 输出后的轻量后处理，不能替代真实推理

### 4. 错误处理策略

冻结如下：

1. `heuristic` 模式：保持当前行为，可在无模型文件时启动。
2. `onnx` 模式：启动时即加载共享库和模型文件；任一缺失、版本不匹配或 Session 初始化失败，服务直接退出，不允许 silent fallback。
3. `healthz` 仍返回进程是否存活；运行时初始化失败则进程不应进入监听状态。
4. API 协议不改：`POST /v1/inference/predict` 保持现有输入输出结构。
5. `onnx` 模式下如果 `imageKey` 无法映射到真实文件，或文件不存在，单次请求返回明确错误，不允许伪造预测结果。

### 5. 容器与本地运行要求

冻结如下：

1. `services/inference/Dockerfile` 需要从 Alpine 迁移到带 glibc 的基础镜像，优先 `golang:1.22-bookworm` builder + `debian:bookworm-slim` runtime。
2. runtime 镜像内需要放置：
   - `/app/models/<model>.onnx`
   - `/app/lib/libonnxruntime.so.<version>`
   - inference 可读的 uploads 目录
3. 启动时通过环境变量显式传入上述路径，不依赖默认搜索路径。
4. `infra/docker-compose.yml` 需要让 `api` 与 `inference` 共享同一个 uploads volume。
5. 第一版只做 CPU provider，不引入 CUDA/TensorRT 等额外依赖。

原因：

1. 当前方案需要 cgo。
2. `onnxruntime_go` 非 Windows 平台通过 `dlopen` 加载共享库。
3. 官方 ONNX Runtime Linux 文档强调动态库路径和依赖环境要正确配置；第一版不值得在 Alpine/musl 上做兼容性试错。

## 图片访问边界

冻结如下：

1. 第一版不改 `POST /v1/inference/predict` 的 JSON 协议，继续只传 `imageKey` 和 `sceneTag`。
2. inference 服务根据 `imageKey` 中的文件后缀，结合 `sampleId` 还原本地路径：`<INFERENCE_UPLOAD_ROOT>/<sampleId><suffix>`。
3. API 上传处理逻辑继续负责把文件落到该共享目录；容器模式下通过 volume 保证 inference 可读。
4. 如果 `imageKey` 无法解析或目标文件不存在，ONNX predictor 直接返回明确错误，不回退到 heuristic。

## 最小实现拆解（Task 5）

1. 把 `internal/app/model.go` 中与启发式推理有关的逻辑迁到 `heuristic_predictor.go`。
2. 新建 `predictor.go`，定义统一接口，例如：
   - `Predict(imageKey string, sceneTag string) (InferenceResult, error)`
   - `Name() string`
3. 新建 `onnx_predictor.go`：
   - 初始化 ORT environment
   - 加载 `.onnx` model
   - 把 `imageKey` 映射为 `INFERENCE_UPLOAD_ROOT/<sampleId><suffix>`
   - 做真实图像前处理与 Session 推理
   - 把输出 logits 映射成现有 `InferenceResult`
4. `cmd/inference/main.go` 根据 `INFERENCE_PREDICTOR_MODE` 构造 predictor。
5. `internal/api/handlers.go` 依赖 predictor 接口，而不是具体 `Model`。
6. 测试至少覆盖：
   - `onnx` 模式缺模型文件时失败
   - `onnx` 模式缺共享库时失败
   - `onnx` 模式 `imageKey` 找不到本地文件时返回明确错误
   - `heuristic` 模式仍可运行

## 最小 smoke 命令

本地 smoke：

```bash
cd services/inference && \
INFERENCE_PREDICTOR_MODE=onnx \
INFERENCE_UPLOAD_ROOT=/tmp/meowlator/uploads \
ONNX_MODEL_PATH=/abs/path/model.onnx \
ONNX_SHARED_LIB_PATH=/abs/path/libonnxruntime.so.1.24.1 \
go run ./cmd/inference
```

预期：

1. 若共享库或模型缺失，进程直接报错退出。
2. 若启动成功，再执行：

```bash
curl -X POST http://127.0.0.1:8081/v1/inference/predict \
  -H 'Content-Type: application/json' \
  -d '{"imageKey":"samples/u1/s1.jpg","sceneTag":"FOOD_BOWL"}'
```

3. 若目标图片不存在，请求返回明确错误。
4. 若目标图片存在，返回结构仍为现有 `result` 包装。

## 现有测试可复用部分

1. `services/inference/internal/app/model_test.go`
   - 可保留 `LoadIntentPriors` 相关测试
   - 启发式确定性测试可迁到 `heuristic_predictor_test.go`
2. `services/api/internal/api/flow_test.go`
   - 作为上游回归基线，无需因为 ONNX 接入修改协议
3. `make test-go`
   - 继续作为服务级回归入口

## 参考依据

1. `services/inference/cmd/inference/main.go`
2. `services/inference/internal/api/handlers.go`
3. `services/inference/internal/app/model.go`
4. `services/inference/internal/config/config.go`
5. `services/inference/Dockerfile`
6. `services/api/internal/api/handlers.go`
7. `ml/training/scripts/export_onnx.py`
8. `github.com/yalue/onnxruntime_go v1.27.0 README`
9. ONNX Runtime official install docs: https://onnxruntime.ai/docs/install/
