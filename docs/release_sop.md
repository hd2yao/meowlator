# Meowlator White-list Release SOP (v1.0.0)

## Preflight Checklist

1. `make test` passed on release commit.
2. Database migrations are applied (`infra/migrations/001~003`).
3. Candidate model is registered by `POST /v1/admin/models/register`.
4. Gate report (`artifacts/pipeline/gate_report.json`) is `pass=true`.
5. Environment variables are set:
   - `ADMIN_TOKEN`
   - `RATE_LIMIT_PER_USER_MIN`
   - `RATE_LIMIT_PER_IP_MIN`
   - `EDGE_DEVICE_WHITELIST`
   - `PAIN_RISK_ENABLED`

## Gray Rollout Steps

1. Set 10% gray:
   - `POST /v1/admin/models/rollout`
   - body: `{\"modelVersion\":\"<candidate>\",\"rolloutRatio\":0.1,\"targetBucket\":0}`
2. Observe 24h:
   - API error rate
   - finalize p95 latency
   - cloud fallback ratio
   - LLM timeout ratio
3. Increase to 30%, 60% with the same 24h observation window.
4. Full activation:
   - `POST /v1/admin/models/activate`
   - body: `{\"modelVersion\":\"<candidate>\"}`

## Rollback Policy

Trigger rollback if any condition meets:

1. Error rate increases by 50%+ over baseline.
2. p95 latency exceeds threshold by 20%+.
3. User complaint volume spikes abnormally.

Rollback steps:

1. Activate previous stable model via `POST /v1/admin/models/activate`.
2. Temporarily tighten `cloudFallbackThreshold` in config.
3. Keep evidence snapshots:
   - model version
   - rollout ratio
   - anomaly window
   - impact scope

## 7-day White-list Observation Targets

1. Share rate `>= 15%`.
2. Valid feedback rate `>= 25%`.
3. System error rate `< 1.5%`.
4. Monthly budget forecast `<= 3000 RMB`.
