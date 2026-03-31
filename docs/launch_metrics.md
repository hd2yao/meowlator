# 上线指标看板协议

> 说明：目标阈值用于达标评估，告警阈值用于监控触发，两者不等价。

## 核心业务指标

1. `dau`
2. `share_rate`（`share_uv / result_uv`）
3. `valid_feedback_rate`（`feedback_valid / result_uv`）

## 核心质量指标

1. `intent_top1_offline`
2. `intent_top3_offline`
3. `low_conf_top1`
4. `ece`

## 运行时指标

1. `api_error_rate` — 目标 < 1.5%，告警 > 1.5%
2. `finalize_p95_ms` — 目标 < 500ms，告警 > 2500ms
3. `edge_success_ratio`
4. `cloud_fallback_ratio` — 目标 < 30%
5. `llm_timeout_ratio`

## 当前已落地的最小观测出口

当前版本没有接 Prometheus/Grafana 栈，先在 API 服务内提供轻量级 `/metrics` 文本端点，用于本地联调、灰度观察和后续对接抓取。

### 端点

1. `GET /metrics`
2. 返回类型：`text/plain`
3. 当前不做鉴权，默认面向内网/本地环境使用

### 已实现指标名

1. `api_requests_total`
   - API 已处理请求总数
2. `api_errors_total`
   - API 返回 `4xx/5xx` 的请求数
3. `finalize_requests_total`
   - `POST /v1/inference/finalize` 总次数
4. `finalize_errors_total`
   - `finalize` 失败次数
5. `finalize_fallback_total`
   - `finalize` 结果中 `fallbackUsed=true` 的次数
6. `finalize_duration_ms_count`
7. `finalize_duration_ms_sum`
8. `finalize_duration_ms_bucket{le="50|100|250|500|1000|2500|5000"}`
   - 用于计算 `finalize_p95_ms`
9. `copy_requests_total`
   - copy 生成总请求数，包含缓存命中
10. `copy_failures_total`
   - copy 上游生成失败次数
11. `copy_timeouts_total`
   - copy 上游生成超时次数

### 指标解释

1. `api_error_rate = api_errors_total / api_requests_total`
2. `cloud_fallback_ratio = finalize_fallback_total / finalize_requests_total`
3. `llm_timeout_ratio = copy_timeouts_total / copy_requests_total`
4. `finalize_p95_ms`
   - 当前通过 `finalize_duration_ms_bucket` 近似计算，不在服务内直接输出 p95

### 现阶段限制

1. 指标为进程内内存累计值，服务重启后会清零
2. 还没有多实例聚合
3. 还没有告警推送，只提供观测出口

## 成本指标

1. `api_compute_cost_rmb`
2. `inference_compute_cost_rmb`
3. `storage_cost_rmb`
4. `llm_cost_rmb`
5. `total_monthly_forecast_rmb` — 高峰期 ≤ ¥16,000/月，稳定期 ≤ ¥12,000/月

## 告警规则

1. `api_error_rate > 1.5%` 持续 5 分钟。
2. `finalize_p95_ms > 2500` 持续 5 分钟。
3. `cloud_fallback_ratio` 相比 24 小时基线升高 30%。
4. 高峰期 `total_monthly_forecast_rmb > 16000`。
5. 稳定期 `total_monthly_forecast_rmb > 12000`。
