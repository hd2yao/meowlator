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
