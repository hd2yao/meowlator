# Meowlator

Monorepo for a WeChat Mini Program that performs edge-first cat intent inference with cloud fallback,
and continuously improves with user feedback.

Current project version: `1.0.0` (see `/Users/dysania/program/meowlator/VERSION`).

## Structure

- `apps/wechat-miniprogram`: WeChat Mini Program (TypeScript)
- `services/api`: Go API gateway/business service
- `services/inference`: Go cloud inference fallback service
- `ml/training`: Training and active-learning pipeline scripts
- `ml/model-registry`: Model release metadata
- `infra`: Migrations and local infra
- `docs/implementation_nodes.md`: implementation node timeline
- `tools/record_node.py`: append node records

## Quick start

1. Start infra and services:

```bash
make up
```

2. Local API uses MySQL automatically when `MYSQL_DSN` is set, otherwise falls back to in-memory repository.
Use `/Users/dysania/program/meowlator/.env.example` as a reference for environment variables.

3. Run tests:

```bash
make test
```

4. Database migrations are in `infra/migrations`.

## Iteration Workflow

1. Run vision training baseline (Oxford-IIIT Pet + pseudo intent mapping):

```bash
make train-vision
```

Quick smoke training (small synthetic dataset, fast sanity check):

```bash
make train-vision-smoke
```

2. Build feedback data pipeline (clean -> manifest -> active-learning tasks):

```bash
make clean-feedback-data
make build-training-manifest
make active-learning-daily
```

Build deterministic evaluation split and threshold report:

```bash
make build-eval-splits
make threshold-report
make evaluate-intent
```

Resume training from an existing checkpoint:

```bash
make train-vision-resume
```

3. Export ONNX (and INT8 quantized ONNX):

```bash
make export-onnx
```

4. Optional: load priors into inference service:

```bash
export MODEL_PRIORS_PATH=/Users/dysania/program/meowlator/ml/training/artifacts/mobilenetv3-small-v2/intent_priors.json
```

5. Record a functional node after each feature implementation:

```bash
python3 /Users/dysania/program/meowlator/tools/record_node.py \
  --node-id N007 \
  --version 1.0.0 \
  --area release-readiness \
  --functional-node \"describe feature\" \
  --verification \"make test\" \
  --commit <commit_hash>
```

6. Model release gate check:

```bash
make gate-model
```

7. Mini Program edge inference reports runtime metadata. `POST /v1/inference/finalize` can include:

```json
{
  "edgeRuntime": {
    "engine": "wx-heuristic-v1",
    "modelVersion": "mobilenetv3-small-int8-v2",
    "modelHash": "dev-hash-v1",
    "inputShape": "1x3x224x224",
    "loadMs": 12,
    "inferMs": 38,
    "deviceModel": "iPhone15,2",
    "failureCode": "EDGE_RUNTIME_ERROR"
  }
}
```

8. Optional: enable pain-risk reminder branch in API (non-diagnostic):

```bash
export PAIN_RISK_ENABLED=true
```

9. New security/release env vars:

```bash
export EDGE_DEVICE_WHITELIST=iPhone15,Android
export RATE_LIMIT_PER_USER_MIN=120
export RATE_LIMIT_PER_IP_MIN=300
export ADMIN_TOKEN=dev-admin-token
export WHITELIST_ENABLED=true
export WHITELIST_USERS=user_a,user_b
export WHITELIST_DAILY_QUOTA=100
```

10. Release operation docs:
- `/Users/dysania/program/meowlator/docs/release_sop.md`
- `/Users/dysania/program/meowlator/docs/launch_metrics.md`

## Compliance defaults

- Image retention default: 7 days
- Entertainment-only output, not medical diagnosis
- V2 pain-risk branch is intentionally disabled in MVP
