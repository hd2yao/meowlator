# Changelog

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
