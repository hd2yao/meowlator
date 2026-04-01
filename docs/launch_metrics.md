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

## 当前已落地的观测与告警基础

当前版本已接入本地 Prometheus + Alertmanager + Grafana（通过 `infra/docker-compose.yml` 启动），并保留 API 服务内 `/metrics` 文本端点作为采集入口。

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
8. `finalize_duration_ms_bucket{le="50|100|250|500|1000|2500|5000|+Inf"}`
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

### 本地启动方式

1. （可选）先设置告警通道环境变量：

```bash
export ALERT_TG_BOT_TOKEN=123456:abc
export ALERT_TG_CHAT_ID=123456789
export ALERT_WEBHOOK_URL=http://127.0.0.1:19093/alert
```

2. 启动业务服务：

```bash
make up
```

3. 启动核心观测栈（Prometheus + Alertmanager）：

```bash
make up-observability
```

4. 可选启动 Grafana：

```bash
make up-grafana
```

5. 访问入口：
   - Prometheus: `http://127.0.0.1:9090`
   - Alertmanager: `http://127.0.0.1:9093`
   - Grafana: `http://127.0.0.1:3000`（默认 `admin/admin`）
   - 预置看板：`Meowlator Runtime Overview`

### 已落地告警规则

规则文件：`infra/monitoring/alerts.yml`

1. `MeowlatorAPIErrorRateHigh`
   - 条件：`api_error_rate > 1.5%` 持续 5 分钟
2. `MeowlatorFinalizeP95High`
   - 条件：`finalize_p95_ms > 2500` 持续 5 分钟
3. `MeowlatorCloudFallbackRatioHigh`
   - 条件：`cloud_fallback_ratio > 30%` 持续 10 分钟
4. `MeowlatorCopyTimeoutRatioHigh`
   - 条件：`llm_timeout_ratio > 10%` 持续 10 分钟

### Grafana 预置看板

1. datasource provisioning：`infra/monitoring/grafana/provisioning/datasources/prometheus.yml`
2. dashboard provider：`infra/monitoring/grafana/provisioning/dashboards/meowlator.yml`
3. dashboard JSON：`infra/monitoring/grafana/dashboards/meowlator-runtime-overview.json`
4. 预置看板覆盖：
   - API Requests/s
   - API Error Rate (%)
   - Finalize P95 (ms)
   - Cloud Fallback Ratio (%)
   - Copy Timeout Ratio (%)
   - Finalize Requests/s

### 告警通知通道

1. Alertmanager 模板配置：`infra/monitoring/alertmanager.yml`
2. Alertmanager 运行时配置：`infra/monitoring/alertmanager.runtime.yml`（由渲染脚本生成，不入库）
3. 渲染脚本：`tools/render_alertmanager_config.sh`（`make up*` 会自动先执行）
4. 当前默认把 `severity=critical` 告警转发到 `telegram` 和 `webhook` receiver
5. Telegram 参数：
   - `ALERT_TG_BOT_TOKEN`
   - `ALERT_TG_CHAT_ID`
6. Webhook 参数：
   - `ALERT_WEBHOOK_URL`（Compose 默认值：`http://127.0.0.1:19093/alert`）
7. 本地调试时可在启动前覆盖环境变量，例如：

```bash
ALERT_TG_BOT_TOKEN=123456:abc \
ALERT_TG_CHAT_ID=123456789 \
ALERT_WEBHOOK_URL=http://127.0.0.1:19093/alert \
make up-observability
```

### 现阶段限制

1. 指标仍是进程内内存累计值，服务重启后会清零
2. 当前只接入 Telegram + Webhook，未接飞书/钉钉/邮件专用模板
3. 当前仅预置 1 个运行时总览看板，尚未覆盖业务与成本维度看板

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
