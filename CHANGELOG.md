# Changelog

## [1.0.0] - 2026-02-23

### Added
- Session auth flow: `POST /v1/auth/wechat/login` + API bearer session validation.
- Request protection: per-user/per-IP rate limiting and signed request verification for sample upload-url / delete APIs.
- White-list launch control: optional whitelist gating and per-user daily quota.
- Admin model release APIs: `POST /v1/admin/models/register`, `POST /v1/admin/models/rollout`, `POST /v1/admin/models/activate`.
- Client config extensions: `edgeDeviceWhitelist`, `modelRollout`, `riskEnabled`, `abBucketRules`.
- Data migrations for release readiness: `user_sessions`, `active_learning_tasks`, `model_evaluations`, `risk_events`, model rollout bucket column and sample indexes.
- Daily expired-sample cleanup loop in API service.
- Training v0.6 baseline additions: deterministic split builder, threshold report, release gate script, and calibration artifact output (`calibration.json`).

### Changed
- `edgeRuntime` / `edgeMeta` schema extended with `modelHash`, `inputShape`, `failureCode`.
- `GET /v1/metrics/client-config` now returns rollout selection metadata (`rolloutModel`, `selectedModel`, `inRollout`, rollout bucket info) and applies user-level gray routing deterministically.
- Mini Program request layer now auto-login/authenticate, sends signed requests where required, and attaches auth headers for uploads.
- Mini Program index flow now pulls `client-config` before edge inference, applies `selectedModel` to edge runtime metadata, and falls back to cloud when device model is not in edge whitelist.
- Mini Program package version bumped to `1.0.0`.

## [0.5.0] - 2026-02-22

### Added
- API inference result supports optional `risk` block (`painRiskScore`, `painRiskLevel`, `riskEvidence`, `disclaimer`).
- Service-level risk evaluation (`EvaluatePainRisk`) based on state + intent signals.
- Runtime feature flag: `PAIN_RISK_ENABLED` for enabling risk branch.
- Mini Program result page adds risk card and fixed non-diagnostic disclaimer rendering.

### Changed
- Copy generation now enforces risk disclaimer when risk branch is present.
- `POST /v1/inference/finalize` docs include risk response example.

## [0.4.0] - 2026-02-22

### Added
- Training script `ml/training/scripts/train.py` now supports `--resume-checkpoint` incremental training.
- Training artifacts now include `confusion_matrix.json` alongside `metrics.json` and `intent_priors.json`.
- New feedback data cleaning script: `ml/training/scripts/data_cleaning.py` (dedup + suspicious user down-weight + label checks).
- New manifest builder: `ml/training/scripts/build_training_manifest.py` (public + feedback weighted merge).
- New active-learning task generator: `ml/training/scripts/generate_active_learning_tasks.py` (40/40/20 strategy output).
- New unit tests for data cleaning, manifest build, and active-learning task generation.

### Changed
- Makefile adds reusable pipeline targets: `clean-feedback-data`, `build-training-manifest`, `active-learning-daily`, `train-vision-resume`.
- Training script now records reproducibility metadata (`seed`, `resumed_from`) in checkpoint and metrics outputs.

## [0.3.0] - 2026-02-22

### Added
- API `POST /v1/inference/finalize` now accepts `edgeRuntime` payload (`engine`, `modelVersion`, `loadMs`, `inferMs`, `deviceModel`, `failureReason`).
- Final inference result includes `result.edgeMeta` for runtime observability (`fallbackUsed`, `usedEdgeResult`).
- Mini Program added `EdgeInferenceEngine` abstraction (`loadModel`, `predict`, `getHealth`) and runtime reporting.

### Changed
- Mini Program index flow no longer calls fixed `mockEdgeResult`; it now uses edge inference output and falls back to cloud automatically when edge inference fails.
- Mini Program shared types expanded for `EdgeRuntime` / `EdgeMeta`.

## [0.2.0] - 2026-02-21

### Added
- Implementation node tracking docs and recording tool.
- Dataset-based vision training pipeline (Oxford-IIIT Pet + MobileNetV3).
- Real ONNX export path from checkpoint with optional INT8 quantization.
- Inference service support for optional intent priors loaded from model artifacts.

### Changed
- Mini program package version bumped to `0.2.0`.
- Project docs updated with versioned iteration workflow and training commands.

## [0.1.1] - 2026-02-21

### Fixed
- Docker image build for API now copies `go.sum` and downloads dependencies.
- Compose startup ordering for API now waits for MySQL health.

## [0.1.0] - 2026-02-21

### Added
- MVP monorepo bootstrap (Mini Program, API service, inference service, training skeleton, infra).
- Core API endpoints, feedback loop, cloud fallback inference flow.
- Local compose stack with MySQL/Redis and CI test workflow.
