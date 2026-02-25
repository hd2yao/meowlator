# Meowlator 项目深度研究

> 最后更新：2026-02-25
> 版本：2.0.0（事实校正版）
> 研究范围：完整架构、数据流、技术栈、发布流程

---

## 1. 项目核心定义

**Meowlator** 是一个微信小程序猫意图识别系统，核心特点：

- **端侧优先推理**：小程序本地运行启发式/模拟推理，快速响应
- **云端兜底**：低置信度或端侧失败时自动调用云服务复判
- **反馈闭环**：用户反馈实时入库，支持离线训练与灰度发布
- **娱乐定位**：输出搞笑拟人文案，非医疗诊断

---

## 2. 技术栈全景

| 层级 | 技术 | 用途 |
|------|------|------|
| **前端** | 微信原生 + TypeScript | 小程序 UI、端侧推理、图片采集 |
| **API** | Go net/http | 业务编排、会话管理、限流、签名验证 |
| **推理** | Go | 云端复判、先验融合、结果聚合 |
| **训练** | Python + PyTorch + torchvision | MobileNetV3 视觉模型、ONNX 导出 |
| **存储** | MySQL | 业务事实存储（用户、样本、反馈、模型注册） |
| **缓存** | Redis | 文案缓存（降低 LLM 成本） |
| **部署** | Docker Compose | 本地开发与测试 |

---

## 3. 架构与数据流

### 3.1 完整推理链路

```
小程序选图
    ↓
[端侧推理] (启发式/模拟)
    ↓
POST /v1/inference/finalize
    ↓
[阈值决策]
    ├─ confidence >= edgeAccept → 直接用端结果
    ├─ confidence < cloudFallback → 云端复判
    └─ 中间区间 → 云端复判
    ↓
[文案生成] (LLM 可选 + 模板兜底)
    ↓
返回 result + copy + needFeedback
    ↓
用户反馈 POST /v1/feedback
    ↓
MySQL 入库 (training_weight + reliability_score)
```

### 3.2 关键阈值

- `EDGE_ACCEPT_THRESHOLD`：端侧置信度达到此值直接返回（默认 0.7）
- `CLOUD_FALLBACK_THRESHOLD`：置信度低于此值强制云端复判（默认 0.45）
- 中间区间：云端复判但不强制反馈

### 3.3 意图分类体系

8 个意图标签（`IntentLabel`）：

1. `FEEDING` — 进食
2. `SEEK_ATTENTION` — 寻求关注
3. `WANT_PLAY` — 想玩耍
4. `WANT_DOOR_OPEN` — 想开门
5. `DEFENSIVE_ALERT` — 防御警戒
6. `RELAX_SLEEP` — 放松睡眠
7. `CURIOUS_OBSERVE` — 好奇观察
8. `UNCERTAIN` — 不确定

### 3.4 状态 3D 空间

每个推理结果包含三维状态：

```go
type State3D struct {
    Tension Level3  // 紧张度：LOW / MID / HIGH
    Arousal Level3  // 唤醒度：LOW / MID / HIGH
    Comfort Level3  // 舒适度：LOW / MID / HIGH
}
```

---

## 4. 核心模块详解

### 4.1 小程序（`apps/wechat-miniprogram`）

**职责**：
- 图片采集与上传
- 端侧推理执行（`EdgeInferenceEngine`）
- 结果展示与用户反馈

**关键文件**：
- `app.ts` — 应用入口，会话初始化
- `utils/api.ts` — HTTP 请求与签名
- `utils/edge_inference.ts` — 端侧推理引擎（当前启发式）
- `pages/index/index.ts` — 拍照与上传
- `pages/result/result.ts` — 结果展示与反馈

**当前实现**：
- 启发式推理（哈希 + 规则）
- 支持设备白名单检查
- 上报 `edgeRuntime` 元数据（引擎、模型版本、耗时、设备型号）

### 4.2 API 服务（`services/api`）

**职责**：
- 会话管理与鉴权
- 样本上传与签名验证
- 推理结果融合与决策
- 反馈入库与权重计算
- 模型发布管理
- 限流与白名单控制

**核心接口**：

| 接口 | 方法 | 用途 |
|------|------|------|
| `/v1/auth/wechat/login` | POST | 微信登录，返回 sessionToken |
| `/v1/samples/upload-url` | POST | 获取上传 URL（需签名） |
| `/v1/samples/upload/{sampleId}` | POST | 上传图片 |
| `/v1/inference/finalize` | POST | 融合推理决策 |
| `/v1/feedback` | POST | 用户反馈 |
| `/v1/copy/generate` | POST | 文案生成 |
| `/v1/samples/{sampleId}` | DELETE | 删除样本（需签名） |
| `/v1/metrics/client-config` | GET | 获取客户端配置（阈值、灰度、白名单） |
| `/v1/admin/models/register` | POST | 注册候选模型（需 Admin Token） |
| `/v1/admin/models/rollout` | POST | 灰度发布（需 Admin Token） |
| `/v1/admin/models/activate` | POST | 激活模型（需 Admin Token） |

**关键组件**：

- `domain/types.go` — 数据模型（Intent、State3D、InferenceResult、Feedback）
- `app/service.go` — 业务逻辑（阈值决策、融合推理）
- `app/inference_client.go` — 云推理客户端
- `app/redis_cache.go` — 文案缓存
- `repository/mysql.go` — MySQL 持久化
- `repository/memory.go` — 内存仓储（本地测试用）

### 4.3 推理服务（`services/inference`）

**职责**：
- 云端复判
- 先验融合
- 结果聚合

**当前实现**：
- 哈希 + 规则推理
- 支持加载 `intent_priors.json`
- 返回统一的 `InferenceResult` 结构

### 4.4 训练流水线（`ml/training`）

**核心脚本**：

| 脚本 | 用途 |
|------|------|
| `train.py` | MobileNetV3 训练（Oxford-IIIT Pet + 伪标签） |
| `export_onnx.py` | ONNX 导出（支持 INT8 量化） |
| `data_cleaning.py` | 反馈数据清洗 |
| `build_training_manifest.py` | 构建训练清单（公开数据 + 反馈） |
| `generate_active_learning_tasks.py` | 每日主动学习任务 |
| `build_eval_splits.py` | 构建评估集（train/val/test = 0.7/0.15/0.15） |
| `threshold_report.py` | 阈值分析报告 |
| `evaluate_intent_metrics.py` | 意图指标评估 |
| `gate_model_release.py` | 模型发布门禁检查 |

**关键产物**：

- `metrics.json` — 模型指标（top1、top3、loss）
- `intent_priors.json` — 意图先验（用于云推理融合）
- `confusion_matrix.json` — 混淆矩阵
- `calibration.json` — 置信度校准
- `gate_report.json` — 发布门禁报告（pass/fail）

**数据来源**：

1. `oxford` — Oxford-IIIT Pet（伪标签映射）
2. `fake` — 烟雾测试数据
3. 反馈样本 — 用户反馈经清洗后进入训练

---

## 5. 关键概念深度解析

### 5.1 "启发式/模拟" vs "ONNX 真推理"

**启发式/模拟**：
- 不加载 `.onnx` 文件
- 使用哈希、规则、先验
- 结果确定性、可复现
- 当前端侧与云侧的实现方式

**ONNX 真推理**：
- 加载 `.onnx` 模型文件
- 执行真实张量计算
- 结果依赖模型参数
- 训练脚本已支持导出，但主链路未切换

**判断已切到真推理的标志**：
1. 有 ONNX runtime 会话加载日志
2. 模型文件不可用时接口失败
3. 同图不同模型版本输出有可解释差异

### 5.2 反馈权重与可靠性

**训练权重** (`training_weight`)：
- 确认正确：0.6
- 纠错（提供 trueLabel）：1.0

**可靠性分数** (`reliability_score`)：
- 基于用户历史反馈准确率
- 范围 [0, 1]
- 用于加权训练样本

**入库计算**：
```
effective_weight = training_weight * reliability_score
```

### 5.3 灰度分桶与确定性路由

**分桶逻辑**：
1. 用户 ID 哈希 → 100 个桶（0-99）
2. 根据 `rolloutRatio` 计算目标桶范围
3. 用户桶号在范围内 → 命中灰度模型

**示例**：
- `rolloutRatio = 0.1` → 目标桶 0-9（10%）
- `targetBucket = 0` → 从桶 0 开始
- 用户桶号 5 → 命中灰度

**客户端消费**：
- `client-config` 返回 `selectedModel`
- 小程序使用 `selectedModel` 作为推理版本
- 上报时包含 `modelVersion`

### 5.4 模型发布状态机

```
CANDIDATE
    ↓
GRAY (灰度中，rolloutRatio < 1.0)
    ↓
ACTIVE (全量激活)
    ↓
ROLLED_BACK (回滚)
```

**状态转移**：
- `register` → CANDIDATE
- `rollout` → GRAY
- `activate` → ACTIVE（同时回滚前一个 ACTIVE/GRAY）

### 5.5 会话与鉴权

**登录流程**：
1. 小程序调用 `POST /v1/auth/wechat/login`
2. 返回 `userId + sessionToken + expiresAt`
3. 后续请求需携带：
   - `Authorization: Bearer <sessionToken>`
   - `X-User-Id: <userId>`

**签名验证**（MVP 级 FNV32）：
- 上传与删除接口需签名
- 请求头：`X-Req-Ts + X-Req-Sig`

### 5.6 白名单与配额

**设备白名单** (`EDGE_DEVICE_WHITELIST`)：
- 逗号分隔的设备型号列表
- 不在白名单的设备直接走云端兜底
- 小程序上报 `deviceModel`

**用户白名单** (`WHITELIST_ENABLED`)：
- 启用时仅 `WHITELIST_USERS` 中的用户可访问
- 每用户每日配额 `WHITELIST_DAILY_QUOTA`

---

## 6. 发布与灰度流程

### 6.1 发布前检查清单

1. ✅ `make test` 全通过
2. ✅ 数据库迁移执行完成
3. ✅ 候选模型通过 `POST /v1/admin/models/register`
4. ✅ `gate_report.json` 为 `pass=true`
5. ✅ 环境变量配置正确

### 6.2 灰度发布步骤

**第一阶段：10% 灰度**
```bash
POST /v1/admin/models/rollout
{
  "modelVersion": "<candidate>",
  "rolloutRatio": 0.1,
  "targetBucket": 0
}
```
观察 24 小时：
- API 错误率
- `finalize` p95 延迟
- 云端兜底比例
- LLM 超时比例

**第二阶段：30% 灰度**
```bash
POST /v1/admin/models/rollout
{
  "modelVersion": "<candidate>",
  "rolloutRatio": 0.3,
  "targetBucket": 0
}
```

**第三阶段：60% 灰度**
```bash
POST /v1/admin/models/rollout
{
  "modelVersion": "<candidate>",
  "rolloutRatio": 0.6,
  "targetBucket": 0
}
```

**全量激活**
```bash
POST /v1/admin/models/activate
{
  "modelVersion": "<candidate>"
}
```

### 6.3 回滚触发条件

满足任一条件即触发回滚：

1. 错误率较基线上升 50% 以上
2. p95 延迟超过阈值 20% 以上
3. 用户投诉量异常上升

**回滚步骤**：
1. 激活上一稳定版本
2. 临时收紧 `cloudFallbackThreshold`
3. 留存证据快照

---

## 7. 环境变量完整清单

### 7.1 基础运行

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `API_ADDR` | `:8080` | API 监听地址 |
| `INFERENCE_URL` | `http://localhost:8081` | 推理服务 URL |
| `MYSQL_DSN` | 无 | MySQL 连接字符串 |
| `REDIS_ADDR` | `localhost:6379` | Redis 地址 |
| `DEFAULT_RETENTION_DAYS` | `7` | 图片保留天数 |

### 7.2 推理策略

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `MODEL_VERSION` | `mobilenetv3-small-int8-v1` | 当前模型版本 |
| `EDGE_ACCEPT_THRESHOLD` | `0.7` | 端侧接受阈值 |
| `CLOUD_FALLBACK_THRESHOLD` | `0.45` | 云端兜底阈值 |
| `PAIN_RISK_ENABLED` | `false` | 是否启用疑似不适提示 |
| `EDGE_DEVICE_WHITELIST` | 无 | 设备白名单（逗号分隔） |

### 7.3 安全与控制

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `ADMIN_TOKEN` | 无 | 管理员 Token |
| `RATE_LIMIT_PER_USER_MIN` | `120` | 用户每分钟限流 |
| `RATE_LIMIT_PER_IP_MIN` | `300` | IP 每分钟限流 |
| `WHITELIST_ENABLED` | `false` | 是否启用用户白名单 |
| `WHITELIST_USERS` | 无 | 白名单用户（逗号分隔） |
| `WHITELIST_DAILY_QUOTA` | `100` | 每用户每日配额 |

### 7.4 文案生成

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `COPY_LLM_ENABLED` | `false` | 是否启用 LLM 文案 |
| `COPY_LLM_ENDPOINT` | 无 | LLM 服务端点 |
| `COPY_TIMEOUT_MS` | `1200` | 文案生成超时（毫秒） |

---

## 8. 数据库架构

### 8.1 核心表

**users**
- `user_id` (PK)
- `wechat_openid`
- `created_at`

**user_sessions**
- `session_token` (PK)
- `user_id` (FK)
- `wechat_code`
- `expires_at`
- `created_at`

**samples**
- `sample_id` (PK)
- `user_id` (FK)
- `cat_id` (FK)
- `image_key`
- `scene_tag`
- `model_version`
- `edge_pred_json` (JSON)
- `final_pred_json` (JSON)
- `created_at`
- `expire_at`

**feedback**
- `feedback_id` (PK)
- `sample_id` (FK)
- `user_id` (FK)
- `is_correct`
- `true_label`
- `reliability_score`
- `training_weight`
- `created_at`

**model_registry**
- `model_version` (PK)
- `task_scope`
- `metrics_json` (JSON)
- `status` (CANDIDATE/GRAY/ACTIVE/ROLLED_BACK)
- `rollout_ratio`
- `target_bucket`
- `created_at`

---

## 9. 常见问题与排障

### 9.1 小程序与后端结果不一致

**原因**：可能触发了云端兜底

**排查**：
1. 检查 `finalize` 返回的 `source` 字段
2. 检查 `fallbackUsed` 标记
3. 查看 `edgeMeta.failureCode`

### 9.2 训练脚本后台执行

```bash
nohup make train-vision > ml/training/logs/train_$(date +%F_%H%M%S).log 2>&1 &
echo $!
```

查看状态：
```bash
ps -ef | grep "scripts/train.py"
tail -f ml/training/logs/<your_log>.log
```

### 9.3 本地测试 vs 正式部署差异

| 方面 | 本地测试 | 正式部署 |
|------|---------|---------|
| 存储 | 内存仓储可选 | 必须 MySQL 持久化 |
| 域名 | 127.0.0.1 可用 | 必须 HTTPS 合法域名 |
| 依赖 | 可容忍部分降级 | 必须监控告警齐全 |

---

## 10. 成本控制抓手

1. **提高端侧命中率** → 减少云推理调用
2. **提高文案缓存命中率** → 减少 LLM 成本
3. **原图 7 天清理** → 长期仅留匿名特征与标签

**周报建议字段**：
- 云兜底占比
- 文案缓存命中率
- 存储增长趋势
- 单次识别成本估算

---

## 11. 合规与安全

### 11.1 输出边界

- 娱乐与辅助理解，不作医疗诊断
- 若启用"疑似不适"，必须附免责声明

### 11.2 用户权利

- 支持删除样本与相关记录
- `DELETE /v1/samples/{sampleId}`

### 11.3 数据策略

- 默认 7 天过期清理
- 长期仅留匿名特征与标签

### 11.4 发布留痕

- 模型版本可追踪
- 灰度比例可追踪
- 关键策略变更可追踪

---

## 12. 实现节点时间线

| 节点 | 日期 | 版本 | 功能 | 状态 |
|------|------|------|------|------|
| N001 | 2026-02-21 | 0.1.0 | Monorepo 初始化 | ✅ |
| N002 | 2026-02-21 | 0.1.1 | 运行稳定性 | ✅ |
| N003 | 2026-02-21 | 0.2.0 | 训练与版本化 | ✅ |
| N004 | 2026-02-22 | 0.3.0 | 端侧运行时观测 | ✅ |
| N005 | 2026-02-22 | 0.4.0 | 数据流水线自动化 | ✅ |
| N006 | 2026-02-22 | 0.5.0 | 疑似不适提示 | ✅ |
| N007 | 2026-02-23 | 1.0.0 | 发布就绪 | ✅ |
| N008 | 2026-02-23 | 1.0.0 | 白名单配额 | ✅ |
| N009 | 2026-02-23 | 1.0.0 | 灰度路由 | ✅ |
| N010 | 2026-02-23 | 1.0.0 | 客户端灰度接入 | ✅ |
| N011 | 2026-02-23 | 1.0.0 | 文档知识点 | ✅ |
| N012 | 2026-02-24 | 1.0.0 | 文档重构 | ✅ |
| N013 | 2026-02-24 | 1.0.0 | 文档本地化 | ✅ |

---

## 13. 快速命令参考

```bash
# 启停
make up
make down

# 测试
make test
make test-go
make test-py
make test-mini

# 单服务运行
make run-api
make run-inference

# 训练与导出
make train-vision
make train-vision-smoke
make train-vision-resume
make export-onnx

# 数据流水线
make clean-feedback-data
make build-training-manifest
make active-learning-daily
make build-eval-splits
make threshold-report
make evaluate-intent
make gate-model
```

---

## 14. 知识点自测清单

- [ ] 能解释本地测试与正式部署差异
- [ ] 能解释启发式推理和 ONNX 真推理差异
- [ ] 能解释 MySQL 与 Redis 分工
- [ ] 能解释反馈实时入库与离线训练发布关系
- [ ] 能解释续训与重训触发条件
- [ ] 能解释阈值如何决定端云路径
- [ ] 能解释灰度分桶与 `selectedModel`
- [ ] 能给出一套最小可执行的灰度回滚流程

---

## 15. 关键文件导航

| 文件 | 用途 |
|------|------|
| `README.md` | 项目概览 |
| `docs/project_manual.md` | 详细项目手册 |
| `docs/api.md` | API 文档 |
| `docs/release_sop.md` | 发布 SOP |
| `docs/launch_metrics.md` | 上线指标 |
| `docs/implementation_nodes.md` | 实现节点记录 |
| `Makefile` | 命令集合 |
| `services/api/internal/domain/types.go` | 核心数据模型 |
| `services/api/internal/app/service.go` | 业务逻辑 |
| `ml/training/scripts/train.py` | 训练脚本 |
| `apps/wechat-miniprogram/utils/edge_inference.ts` | 端侧推理 |

---

## 16. 架构决策与权衡

### 16.1 为什么是启发式推理而非实时 ONNX

**权衡**：
- 启发式：快速、确定性、易调试
- ONNX：准确、真实模型能力、但需模型文件

**当前选择**：启发式（MVP 阶段）
**未来方向**：逐步切换到 ONNX 真推理

### 16.2 为什么反馈不是实时训练

**权衡**：
- 实时训练：快速适应
- 离线训练：稳定、可审计、易回滚

**当前选择**：离线训练（每日/每周）
**原因**：
- 实时训练放大噪声标签风险
- 难做可审计与稳定回滚
- MVP 成本与复杂度不划算

### 16.3 为什么用 Redis 缓存文案

**权衡**：
- 缓存：降低 LLM 成本与时延
- 无缓存：简单但成本高

**当前选择**：Redis 缓存
**降级策略**：Redis 挂了不拖垮主推理链路

---

## 17. 性能与可观测性

### 17.1 关键指标

- **API 错误率** — 目标 < 1.5%，告警 > 1.5%（持续 5 分钟）
- **finalize p95 延迟** — 目标 < 500ms，告警 > 2500ms（持续 5 分钟）
- **云端兜底比例** — 目标 < 30%
- **文案缓存命中率** — 目标 > 60%
- **LLM 超时率** — 目标 < 5%

### 17.2 上线后 7 天观察目标

- 分享率 >= 15%
- 有效反馈率 >= 25%
- 系统错误率 < 1.5%
- 月成本预测：高峰期 <= ¥16,000/月，稳定期 <= ¥12,000/月

---

## 18. UI/UX 设计规范

### 18.1 设计令牌 (Design Tokens)

**品牌色彩**：
- `Primary`: #FF8A4C (活力橙 - 核心行为按钮、高亮意图文本)
- `Primary Dark`: #E06B2E (按钮按压态)
- `Secondary`: #FCD34D (辅助暖黄色)
- `Surface`: #FFFFFF (卡片白)
- `Background`: #FAFAFA (底层背景灰/白)
- `Text Primary`: #1F2937 (主标题、强调文案)
- `Text Secondary`: #6B7280 (次要提示信息)
- `Success`: #10B981 (准确反馈按钮)
- `Danger`: #EF4444 (报错、不准反馈按钮)

**字体排版**：
- 大标题 (Hero Title): 28pt/36px, Extrabold
- 卡片标题 (Card Title): 18pt/24px, Bold
- 正文内容 (Body): 15pt/22px, Regular
- 微小提示 (Caption): 12pt/16px, Regular

**圆角与投影**：
- 卡片/按钮圆角: 16px、24px 或 32px
- 卡片投影 (Soft Shadow): `0 10px 40px -10px rgba(0,0,0,0.08)`

### 18.2 核心页面设计

**首页 (Home Screen)**
- 隐藏原生 Navigation Bar，采用自定义空 Header
- 大字号 Slogan: "读懂猫咪的每一个动作"（品牌色高亮关键词）
- 操作区大卡片（4:5 比例），中间放置猫咪 Emoji
- 两个核心按钮：
  - `📸 立即拍照` (Primary Color 填充 + 强投影)
  - `🖼️ 从相册选择` (浅灰背景)
- 底部合规提示：灰色小字（12px）"非医疗诊断，照片7天销毁"

**结果页 (Result Screen)**
- 上半部分 (45%)：用户上传的猫咪照片满宽展示，左上角后退按钮
- 照片左下角：透明状态标签"端侧极速推理 12ms"
- 下半部分：从底部升起的大圆角（32px）白色卡片
  - 意图标签：全大写，主色点睛（如 "WANT_PLAY"）
  - 翻译文本标题：放大中文翻译（如 "我想玩耍！"）
  - 置信度提示：卡片右上角小块强调（如 95%）
  - AI 拟人文案块：浅灰底色卡片存放搞笑文本
  - 验证反馈区：底部两个大按钮（✅ 超准 和 ❌ 瞎说）

**纠错反馈弹窗 (Feedback Action Sheet)**
- 背板处理：原页面 40% 黑色半透明遮罩
- Title: "纠正意图"（带关闭 ✕ 按钮）
- 选择网格：2列 × 4行 标签网格，枚举 8 类意图
  - 🍖 要吃的 (FEEDING)
  - 👋 求抚摸 (SEEK_ATTENTION)
  - ⚽️ 想玩耍 (WANT_PLAY)
  - 🚪 要开门 (WANT_DOOR_OPEN)
  - 😾 警惕防御 (DEFENSIVE_ALERT)
  - 💤 放松睡觉 (RELAX_SLEEP)
  - 👀 好奇观察 (CURIOUS_OBSERVE)
  - ❓ 摸鱼/不确定 (UNCERTAIN)
- 提交按钮：吸底全宽主按钮 "提交反馈"

### 18.3 设计资源

**文件清单**：
- `docs/ui_design.html` — Tailwind HTML 源码（可直接在浏览器打开）
- `docs/ui_design_spec.md` — 详细设计规范文档
- `docs/ui_design_home.svg` — 首页设计稿
- `docs/ui_design_result.svg` — 结果页设计稿
- `docs/ui_design_feedback.svg` — 反馈弹窗设计稿

**Figma 导入方式**：
1. 使用 Chrome 打开 `docs/ui_design.html`
2. 在 Figma 安装插件：[Figma to HTML, CSS, React & more!](https://www.figma.com/community/plugin/849822798253640248)
3. 在插件界面输入本地 `localhost` 或 `file://` 地址
4. 插件自动解析 DOM、颜色、边距和文字属性，生成可编辑的 Auto Layout 涂层

---

## 19. 总结

Meowlator 是一个**端云融合、反馈驱动、灰度可控**的猫意图识别系统。核心创新点：

1. **端侧优先** — 快速响应，降低延迟
2. **云端兜底** — 保证准确性
3. **反馈闭环** — 持续改进
4. **灰度发布** — 风险可控
5. **成本意识** — 缓存、清理、限流

当前处于 **1.0.0 MVP 阶段**，已具备完整的发布工程能力，下一步重点是：

- 切换到 ONNX 真推理
- 扩大用户规模与反馈量
- 优化成本与性能指标
- 完善合规与安全链路
