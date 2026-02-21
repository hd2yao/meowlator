# Changelog

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
