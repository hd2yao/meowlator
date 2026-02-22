# Launch Metrics Dashboard Contract

## Core Business Metrics

1. `dau`
2. `share_rate` (`share_uv / result_uv`)
3. `valid_feedback_rate` (`feedback_valid / result_uv`)

## Core Quality Metrics

1. `intent_top1_offline`
2. `intent_top3_offline`
3. `low_conf_top1`
4. `ece`

## Runtime Metrics

1. `api_error_rate`
2. `finalize_p95_ms`
3. `edge_success_ratio`
4. `cloud_fallback_ratio`
5. `llm_timeout_ratio`

## Cost Metrics

1. `api_compute_cost_rmb`
2. `inference_compute_cost_rmb`
3. `storage_cost_rmb`
4. `llm_cost_rmb`
5. `total_monthly_forecast_rmb`

## Alert Rules

1. `api_error_rate > 1.5%` for 5 min.
2. `finalize_p95_ms > 2500` for 5 min.
3. `cloud_fallback_ratio` increases by 30% compared to 24h baseline.
4. `total_monthly_forecast_rmb > 3000`.
