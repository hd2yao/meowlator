# 上线指标看板协议

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

1. `api_error_rate`
2. `finalize_p95_ms`
3. `edge_success_ratio`
4. `cloud_fallback_ratio`
5. `llm_timeout_ratio`

## 成本指标

1. `api_compute_cost_rmb`
2. `inference_compute_cost_rmb`
3. `storage_cost_rmb`
4. `llm_cost_rmb`
5. `total_monthly_forecast_rmb`

## 告警规则

1. `api_error_rate > 1.5%` 持续 5 分钟。
2. `finalize_p95_ms > 2500` 持续 5 分钟。
3. `cloud_fallback_ratio` 相比 24 小时基线升高 30%。
4. `total_monthly_forecast_rmb > 3000`。
