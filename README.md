# Meowlator

Monorepo for a WeChat Mini Program that performs edge-first cat intent inference with cloud fallback,
and continuously improves with user feedback.

## Structure

- `apps/wechat-miniprogram`: WeChat Mini Program (TypeScript)
- `services/api`: Go API gateway/business service
- `services/inference`: Go cloud inference fallback service
- `ml/training`: Training and active-learning pipeline scripts
- `ml/model-registry`: Model release metadata
- `infra`: Migrations and local infra

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

3. Database migrations are in `infra/migrations`.

## Compliance defaults

- Image retention default: 7 days
- Entertainment-only output, not medical diagnosis
- V2 pain-risk branch is intentionally disabled in MVP
