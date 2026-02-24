# Meowlator

这是一个微信小程序猫意图识别项目的 Monorepo：端侧优先推理，低置信度自动云端兜底，并通过用户反馈持续迭代。

当前项目版本：`1.0.0`（见 `/Users/dysania/program/meowlator/VERSION`）。

## 目录结构

- `apps/wechat-miniprogram`：微信小程序（TypeScript）
- `services/api`：Go API 网关与业务服务
- `services/inference`：Go 云端推理兜底服务
- `ml/training`：训练与主动学习流水线脚本
- `ml/model-registry`：模型发布元数据
- `infra`：数据库迁移与本地基础设施
- `docs/implementation_nodes.md`：实现节点时间线
- `tools/record_node.py`：实现节点记录工具

## 快速开始

1. 启动基础设施与服务：

```bash
make up
```

2. 本地 API 在设置 `MYSQL_DSN` 时自动使用 MySQL，否则回退内存仓储。环境变量参考 `/Users/dysania/program/meowlator/.env.example`。

3. 运行测试：

```bash
make test
```

4. 数据库迁移脚本位于 `infra/migrations`。

## 迭代工作流

1. 运行视觉基线训练（Oxford-IIIT Pet + 伪意图映射）：

```bash
make train-vision
```

快速烟雾训练（小型合成数据，快速自检）：

```bash
make train-vision-smoke
```

2. 构建反馈数据流水线（清洗 -> 清单 -> 主动学习任务）：

```bash
make clean-feedback-data
make build-training-manifest
make active-learning-daily
```

构建确定性评估切分与阈值报告：

```bash
make build-eval-splits
make threshold-report
make evaluate-intent
```

基于已有 checkpoint 续训：

```bash
make train-vision-resume
```

3. 导出 ONNX（含 INT8 量化 ONNX）：

```bash
make export-onnx
```

4. 可选：向推理服务加载先验分布：

```bash
export MODEL_PRIORS_PATH=/Users/dysania/program/meowlator/ml/training/artifacts/mobilenetv3-small-v2/intent_priors.json
```

5. 每次功能迭代后记录实现节点：

```bash
python3 /Users/dysania/program/meowlator/tools/record_node.py \
  --node-id N007 \
  --version 1.0.0 \
  --area release-readiness \
  --functional-node "描述功能点" \
  --verification "make test" \
  --commit <commit_hash>
```

6. 模型发布门禁检查：

```bash
make gate-model
```

7. 小程序端侧推理可上报运行时元信息。`POST /v1/inference/finalize` 可携带：

```json
{
  "edgeRuntime": {
    "engine": "wx-heuristic-v1",
    "modelVersion": "mobilenetv3-small-int8-v2",
    "modelHash": "dev-hash-v1",
    "inputShape": "1x3x224x224",
    "loadMs": 12,
    "inferMs": 38,
    "deviceModel": "iPhone15,2",
    "failureCode": "EDGE_RUNTIME_ERROR"
  }
}
```

8. 可选：开启 API 疑似不适风险提示分支（非医疗诊断）：

```bash
export PAIN_RISK_ENABLED=true
```

9. 安全与发布相关环境变量：

```bash
export EDGE_DEVICE_WHITELIST=iPhone15,Android
export RATE_LIMIT_PER_USER_MIN=120
export RATE_LIMIT_PER_IP_MIN=300
export ADMIN_TOKEN=dev-admin-token
export WHITELIST_ENABLED=true
export WHITELIST_USERS=user_a,user_b
export WHITELIST_DAILY_QUOTA=100
```

10. 发布操作文档：
- `/Users/dysania/program/meowlator/docs/release_sop.md`
- `/Users/dysania/program/meowlator/docs/launch_metrics.md`

## 合规默认项

- 图片默认保留 7 天
- 输出为娱乐与辅助理解，不作医疗诊断
- V2 疑似不适分支在 MVP 中默认关闭
