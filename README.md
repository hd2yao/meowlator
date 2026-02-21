# Meowlator

Monorepo for a WeChat Mini Program that performs edge-first cat intent inference with cloud fallback,
and continuously improves with user feedback.

Current project version: `0.2.0` (see `/Users/dysania/program/meowlator/VERSION`).

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

2. Export ONNX (and INT8 quantized ONNX):

```bash
make export-onnx
```

3. Optional: load priors into inference service:

```bash
export MODEL_PRIORS_PATH=/Users/dysania/program/meowlator/ml/training/artifacts/mobilenetv3-small-v2/intent_priors.json
```

4. Record a functional node after each feature implementation:

```bash
python3 /Users/dysania/program/meowlator/tools/record_node.py \
  --node-id N004 \
  --version 0.2.0 \
  --area inference \
  --functional-node \"describe feature\" \
  --verification \"make test\" \
  --commit <commit_hash>
```

## Compliance defaults

- Image retention default: 7 days
- Entertainment-only output, not medical diagnosis
- V2 pain-risk branch is intentionally disabled in MVP
