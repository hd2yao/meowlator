# 更新日志

## [1.0.0] - 2026-02-23

### 新增
- 会话鉴权流程：`POST /v1/auth/wechat/login` + API Bearer 会话校验。
- 请求防护：按用户/按 IP 分钟级限流；`upload-url` 与 `delete` 接口签名校验。
- 白名单上线控制：可选白名单拦截 + 按用户每日配额。
- 模型发布管理接口：`POST /v1/admin/models/register`、`POST /v1/admin/models/rollout`、`POST /v1/admin/models/activate`。
- 客户端配置扩展：`edgeDeviceWhitelist`、`modelRollout`、`riskEnabled`、`abBucketRules`。
- 发布就绪数据迁移：`user_sessions`、`active_learning_tasks`、`model_evaluations`、`risk_events`、模型灰度桶字段与样本索引。
- API 服务每日过期样本清理循环任务。
- v0.6 训练基线能力：确定性切分构建、阈值报告、发布门禁脚本、校准产物（`calibration.json`）。

### 变更
- `edgeRuntime` / `edgeMeta` 协议新增 `modelHash`、`inputShape`、`failureCode`。
- `GET /v1/metrics/client-config` 现返回灰度分流元信息（`rolloutModel`、`selectedModel`、`inRollout`、分桶信息），并按用户稳定分桶生效。
- 小程序请求层改为自动登录鉴权；在需要的接口自动携带签名与鉴权头。
- 小程序首页流程在端侧推理前先拉取 `client-config`，将 `selectedModel` 应用于端侧 runtime 元数据；设备不在白名单时自动降级云端。
- 小程序包版本升级到 `1.0.0`。

## [0.5.0] - 2026-02-22

### 新增
- API 推理结果支持可选 `risk` 字段（`painRiskScore`、`painRiskLevel`、`riskEvidence`、`disclaimer`）。
- 服务层风险评估（`EvaluatePainRisk`）：基于状态 + 意图信号。
- 运行时开关：`PAIN_RISK_ENABLED`。
- 小程序结果页新增风险卡片与固定免责声明（非医疗诊断）。

### 变更
- 文案生成在风险分支存在时强制注入免责声明。
- `POST /v1/inference/finalize` 文档补充风险字段响应示例。

## [0.4.0] - 2026-02-22

### 新增
- 训练脚本 `ml/training/scripts/train.py` 支持 `--resume-checkpoint` 续训。
- 训练产物新增 `confusion_matrix.json`（与 `metrics.json`、`intent_priors.json` 同步输出）。
- 新增反馈清洗脚本：`ml/training/scripts/data_cleaning.py`（去重 + 异常用户降权 + 标签校验）。
- 新增训练清单构建脚本：`ml/training/scripts/build_training_manifest.py`（公开数据 + 反馈加权融合）。
- 新增主动学习任务脚本：`ml/training/scripts/generate_active_learning_tasks.py`（40/40/20 采样）。
- 新增数据清洗、清单构建、主动学习脚本单测。

### 变更
- Makefile 新增可复用流水线目标：`clean-feedback-data`、`build-training-manifest`、`active-learning-daily`、`train-vision-resume`。
- 训练脚本在 checkpoint 与 metrics 中记录可复现元数据（`seed`、`resumed_from`）。

## [0.3.0] - 2026-02-22

### 新增
- API `POST /v1/inference/finalize` 支持 `edgeRuntime` 上报（`engine`、`modelVersion`、`loadMs`、`inferMs`、`deviceModel`、`failureReason`）。
- 推理最终结果新增 `result.edgeMeta` 便于运行时观测（`fallbackUsed`、`usedEdgeResult`）。
- 小程序新增 `EdgeInferenceEngine` 抽象（`loadModel`、`predict`、`getHealth`）与 runtime 上报。

### 变更
- 小程序首页不再调用固定 `mockEdgeResult`，改为走端侧推理并在失败时自动云端兜底。
- 小程序共享类型扩展了 `EdgeRuntime` / `EdgeMeta`。

## [0.2.0] - 2026-02-21

### 新增
- 实现节点追踪文档与记录工具。
- 基于公开数据集的视觉训练流水线（Oxford-IIIT Pet + MobileNetV3）。
- 从 checkpoint 导出 ONNX（支持可选 INT8 量化）。
- 推理服务支持从模型产物加载 `intent_priors`（可选）。

### 变更
- 小程序包版本升级到 `0.2.0`。
- 项目文档补充版本化迭代工作流与训练命令。

## [0.1.1] - 2026-02-21

### 修复
- API Docker 镜像构建补齐 `go.sum` 复制与依赖下载。
- Compose 启动顺序调整，API 依赖 MySQL 健康检查通过后再启动。

## [0.1.0] - 2026-02-21

### 新增
- MVP Monorepo 初始化（小程序、API、推理服务、训练骨架、基础设施）。
- 核心 API 接口、反馈闭环、云端兜底推理流程。
- 本地 compose 栈（MySQL/Redis）与 CI 测试工作流。
