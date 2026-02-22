# Implementation Nodes

This file records every functional implementation node in chronological order.

| Node ID | Date | Version | Area | Functional Node | Verification | Commit |
|---|---|---|---|---|---|---|
| N001 | 2026-02-21 | 0.1.0 | Monorepo Bootstrap | Built MVP architecture: API, inference, mini program skeleton, training pipeline skeleton, infra, CI. | `make test` | `ac6d374` |
| N002 | 2026-02-21 | 0.1.1 | Runtime Stability | Fixed Docker API image dependency copy and compose startup order for MySQL readiness. | `make up && docker compose -f infra/docker-compose.yml ps` | `ffaa1e8` |
| N003 | 2026-02-21 | 0.2.0 | Training + Versioning | Added real dataset training script, real ONNX export, inference priors loading, and implementation node recorder. | `make test` + training/inference smoke checks | `TBD` |
| N004 | 2026-02-22 | 0.3.0 | Edge Runtime Telemetry | Replaced mini program fixed mock call with `EdgeInferenceEngine`, added `edgeRuntime` request and `edgeMeta` response for finalize flow, and preserved cloud fallback on edge failure. | `cd services/api && go test ./...` + `cd apps/wechat-miniprogram && npm run typecheck` | `f2c13d8` |
| N005 | 2026-02-22 | 0.4.0 | Data Pipeline Automation | Added feedback cleaning, weighted manifest builder, daily active-learning task generator, training resume support, and confusion matrix artifact output. | `cd ml/training && python3 -m unittest discover -s scripts -p 'test_*.py'` + `make test` | `1565346` |
| N006 | 2026-02-22 | 0.5.0 | Pain Risk Reminder | Added optional pain-risk scoring branch, API risk output schema, disclaimer enforcement in copy generation, and mini-program risk card rendering (non-diagnostic). | `make test` | `3349df5` |
