# Meowlator API v1

## POST /v1/auth/wechat/login

Request:

```json
{
  "code": "wx-login-code"
}
```

Response:

```json
{
  "userId": "user_123abc",
  "sessionToken": "sess_xxx",
  "expiresAt": 1760000000
}
```

Protected APIs require headers:
- `Authorization: Bearer <sessionToken>`
- `X-User-Id: <userId>`

If `WHITELIST_ENABLED=true`, only users in `WHITELIST_USERS` can call protected APIs, with per-user daily quota controlled by `WHITELIST_DAILY_QUOTA`.

## POST /v1/samples/upload-url

Request:

```json
{
  "catId": "cat-default",
  "suffix": ".jpg"
}
```

This endpoint requires request signature headers:
- `X-Req-Ts`
- `X-Req-Sig`

For local MVP debugging, upload the image to the returned URL using multipart field `file`.
The API service accepts it at `POST /v1/samples/upload/{sampleId}` and stores a temp copy under `/tmp/meowlator/uploads`.

Response:

```json
{
  "sampleId": "sample_xxx",
  "imageKey": "samples/demo-user-001/sample_xxx.jpg",
  "uploadUrl": "https://upload.example.local/put?key=...",
  "expiresInSeconds": 600,
  "retentionDeadline": 1730112000
}
```

## POST /v1/inference/finalize

Request:

```json
{
  "sampleId": "sample_xxx",
  "deviceCapable": true,
  "sceneTag": "UNKNOWN",
  "edgeResult": {
    "intentTop3": [{"label":"FEEDING","prob":0.62}],
    "state": {"tension":"MID","arousal":"MID","comfort":"LOW"},
    "confidence": 0.62,
    "source": "EDGE",
    "evidence": ["靠近食盆区域"],
    "copyStyleVersion": "v1"
  },
  "edgeRuntime": {
    "engine": "wx-heuristic-v1",
    "modelVersion": "mobilenetv3-small-int8-v2",
    "modelHash": "dev-hash-v1",
    "inputShape": "1x3x224x224",
    "loadMs": 14,
    "inferMs": 37,
    "deviceModel": "iPhone15,2",
    "failureCode": "EDGE_RUNTIME_ERROR",
    "failureReason": ""
  }
}
```

Response includes final result, copy block, and feedback flag.
When `edgeRuntime` is provided, `result.edgeMeta` will be returned:

```json
{
  "sampleId": "sample_xxx",
  "result": {
    "source": "EDGE",
    "edgeMeta": {
      "engine": "wx-heuristic-v1",
      "modelVersion": "mobilenetv3-small-int8-v2",
      "modelHash": "dev-hash-v1",
      "inputShape": "1x3x224x224",
      "loadMs": 14,
      "inferMs": 37,
      "deviceModel": "iPhone15,2",
      "failureCode": "EDGE_RUNTIME_ERROR",
      "fallbackUsed": false,
      "usedEdgeResult": true
    }
  },
  "fallbackUsed": false,
  "needFeedback": false
}
```

When `PAIN_RISK_ENABLED=true`, response may include `result.risk`:

```json
{
  "result": {
    "risk": {
      "painRiskScore": 0.78,
      "painRiskLevel": "HIGH",
      "riskEvidence": ["紧张度高", "舒适度低", "结合视觉行为证据"],
      "disclaimer": "非医疗诊断，仅作风险提示；若持续异常请咨询兽医。"
    }
  }
}
```

## POST /v1/feedback

- `isCorrect=true` means confirmed label with weight 0.6.
- `isCorrect=false` requires `trueLabel` with weight 1.0

## POST /v1/copy/generate

Input only accepts structured inference JSON. Raw images are not supported.

## DELETE /v1/samples/{sampleId}

Deletes sample and related feedback records.
This endpoint requires request signature headers (`X-Req-Ts`, `X-Req-Sig`).

## GET /v1/metrics/client-config

Returns thresholds, model version, AB config, whitelist and rollout metadata.

Response sample:

```json
{
  "edgeAcceptThreshold": 0.7,
  "cloudFallbackThreshold": 0.45,
  "copyStyleVersion": "v1",
  "modelVersion": "mobilenetv3-small-int8-v2",
  "abBucket": 1,
  "edgeDeviceWhitelist": ["iPhone15", "Android"],
  "modelRollout": {
    "activeModel": "mobilenetv3-small-int8-v1",
    "rolloutModel": "mobilenetv3-small-int8-v2",
    "selectedModel": "mobilenetv3-small-int8-v2",
    "rolloutRatio": 0.1,
    "targetBucket": 30,
    "totalBuckets": 100,
    "userBucket": 34,
    "inRollout": true
  },
  "riskEnabled": false,
  "abBucketRules": {
    "totalBuckets": 3
  }
}
```

## POST /v1/admin/models/register

Registers candidate model metrics (internal endpoint, requires `X-Admin-Token`).

## POST /v1/admin/models/rollout

Sets model to GRAY rollout (internal endpoint, requires `X-Admin-Token`).

## POST /v1/admin/models/activate

Activates target model and rolls back previous active/gray model (internal endpoint, requires `X-Admin-Token`).
