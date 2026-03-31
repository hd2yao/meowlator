# Meowlator API v1 文档

## 本地主链路冒烟

可直接运行：

```bash
make smoke-local
```

该命令会顺序执行 `login -> upload-url -> upload -> finalize -> feedback -> delete`，用于快速验证本地 API 主链路是否可用。

## POST /v1/auth/wechat/login

请求示例：

```json
{
  "code": "wx-login-code"
}
```

响应示例：

```json
{
  "userId": "user_123abc",
  "sessionToken": "sess_xxx",
  "expiresAt": 1760000000
}
```

受保护接口需携带请求头：
- `Authorization: Bearer <sessionToken>`
- `X-User-Id: <userId>`

当 `WHITELIST_ENABLED=true` 时，只有 `WHITELIST_USERS` 中的用户可访问受保护接口，且受 `WHITELIST_DAILY_QUOTA` 的每日配额限制。

## POST /v1/samples/upload-url

请求示例：

```json
{
  "catId": "cat-default",
  "suffix": ".jpg"
}
```

该接口要求请求签名头：
- `X-Req-Ts`
- `X-Req-Sig`

本地 MVP 调试时，可将图片用 multipart 字段 `file` 上传到返回的 URL。  
API 会通过 `POST /v1/samples/upload/{sampleId}` 接收并临时落盘到 `/tmp/meowlator/uploads`。

响应示例：

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

请求示例：

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

响应包含最终推理结果、文案区块和反馈标记。  
当请求携带 `edgeRuntime` 时，响应中会返回 `result.edgeMeta`：

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

小程序当前使用的 `failureCode`：
- `DEVICE_NOT_WHITELISTED`：设备不在 `edgeDeviceWhitelist`，直接走云端兜底。
- `EDGE_RUNTIME_ERROR`：端侧模型加载/推理失败。

当 `PAIN_RISK_ENABLED=true` 时，响应可能包含 `result.risk`：

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

- `isCorrect=true` 表示确认模型结果，样本权重为 0.6。
- `isCorrect=false` 必须提供 `trueLabel`，样本权重为 1.0。

## POST /v1/copy/generate

输入仅支持结构化推理 JSON，不支持原始图片。

## DELETE /v1/samples/{sampleId}

删除样本及关联反馈记录。  
该接口要求请求签名头（`X-Req-Ts`、`X-Req-Sig`）。

## GET /v1/metrics/client-config

返回阈值、模型版本、AB 配置、端侧白名单与灰度发布元数据。

响应示例：

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

注册候选模型指标（内部接口，需 `X-Admin-Token`）。

## POST /v1/admin/models/rollout

将模型设置为灰度状态（内部接口，需 `X-Admin-Token`）。

## POST /v1/admin/models/activate

激活目标模型，并回滚前一个 ACTIVE/GRAY 模型（内部接口，需 `X-Admin-Token`）。
