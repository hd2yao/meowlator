# Meowlator 白名单发布 SOP（v1.0.0）

> 指标口径：目标阈值用于发布达标评估；告警阈值用于运行时触发（见 `docs/KPI_BASELINE.md`）。

## 发布前检查

1. 发布提交上 `make test` 全通过。
2. 数据库迁移已执行（`infra/migrations/001~003`）。
3. 候选模型已通过 `POST /v1/admin/models/register` 注册。
4. 门禁报告 `artifacts/pipeline/gate_report.json` 为 `pass=true`。
5. 环境变量已配置：
   - `ADMIN_TOKEN`
   - `RATE_LIMIT_PER_USER_MIN`
   - `RATE_LIMIT_PER_IP_MIN`
   - `EDGE_DEVICE_WHITELIST`
   - `PAIN_RISK_ENABLED`

## 灰度发布步骤

1. 先灰度 10%：
   - `POST /v1/admin/models/rollout`
   - 请求体：`{"modelVersion":"<candidate>","rolloutRatio":0.1,"targetBucket":0}`
2. 观察 24 小时：
   - API 错误率
   - `finalize` p95 延迟
   - 云端兜底比例
   - LLM 超时比例
   - 月成本预测（按阶段：高峰期 `<= ¥16,000/月`，稳定期 `<= ¥12,000/月`）
3. 按同样 24 小时观察窗口，逐步扩大到 30%、60%。
4. 全量激活：
   - `POST /v1/admin/models/activate`
   - 请求体：`{"modelVersion":"<candidate>"}`

## 回滚策略

满足任一条件即触发回滚：

1. 错误率较基线上升 50% 以上。
2. p95 延迟超过阈值 20% 以上。
3. 用户投诉量异常上升。

回滚步骤：

1. 通过 `POST /v1/admin/models/activate` 激活上一稳定版本。
2. 临时收紧配置中的 `cloudFallbackThreshold`。
3. 留存证据快照：
   - 模型版本
   - 灰度比例
   - 异常时间窗
   - 影响范围

## 白名单发布后 7 天观察目标

1. 分享率 `>= 15%`。
2. 有效反馈率 `>= 25%`。
3. 系统错误率 `< 1.5%`。
4. 月成本预测：高峰期 `<= ¥16,000/月`，稳定期 `<= ¥12,000/月`。
