# Implementation Nodes

This file records every functional implementation node in chronological order.

| Node ID | Date | Version | Area | Functional Node | Verification | Commit |
|---|---|---|---|---|---|---|
| N001 | 2026-02-21 | 0.1.0 | Monorepo Bootstrap | Built MVP architecture: API, inference, mini program skeleton, training pipeline skeleton, infra, CI. | `make test` | `ac6d374` |
| N002 | 2026-02-21 | 0.1.1 | Runtime Stability | Fixed Docker API image dependency copy and compose startup order for MySQL readiness. | `make up && docker compose -f infra/docker-compose.yml ps` | `ffaa1e8` |
| N003 | 2026-02-21 | 0.2.0 | Training + Versioning | Added real dataset training script, real ONNX export, inference priors loading, and implementation node recorder. | `make test` + training/inference smoke checks | `TBD` |
