INSERT INTO model_registry (model_version, task_scope, metrics_json, status, rollout_ratio)
VALUES (
  'mobilenetv3-small-int8-v1',
  'intent_state_mvp',
  JSON_OBJECT('top1', 0.55, 'top3', 0.80, 'latency_p95_ms', 2500),
  'ACTIVE',
  1.0
)
ON DUPLICATE KEY UPDATE
  metrics_json = VALUES(metrics_json),
  status = VALUES(status),
  rollout_ratio = VALUES(rollout_ratio);
