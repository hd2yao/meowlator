# Meowlator API v1

## POST /v1/samples/upload-url

Request:

```json
{
  "userId": "demo-user-001",
  "catId": "cat-default",
  "suffix": ".jpg"
}
```

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
    "loadMs": 14,
    "inferMs": 37,
    "deviceModel": "iPhone15,2",
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
      "loadMs": 14,
      "inferMs": 37,
      "deviceModel": "iPhone15,2",
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

- `isCorrect=true` means confirmed label with weight 0.6
- `isCorrect=false` requires `trueLabel` with weight 1.0

## POST /v1/copy/generate

Input only accepts structured inference JSON. Raw images are not supported.

## DELETE /v1/samples/{sampleId}

Deletes sample and related feedback records.

## GET /v1/metrics/client-config

Returns thresholds, model version, and AB config.
