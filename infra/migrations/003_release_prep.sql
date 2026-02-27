SET @has_target_bucket := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'model_registry'
    AND COLUMN_NAME = 'target_bucket'
);
SET @sql_target_bucket := IF(
  @has_target_bucket = 0,
  'ALTER TABLE model_registry ADD COLUMN target_bucket INT NOT NULL DEFAULT 0 AFTER rollout_ratio',
  'SELECT 1'
);
PREPARE stmt_target_bucket FROM @sql_target_bucket;
EXECUTE stmt_target_bucket;
DEALLOCATE PREPARE stmt_target_bucket;

SET @has_samples_model_created_idx := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'samples'
    AND INDEX_NAME = 'idx_samples_model_created'
);
SET @sql_samples_model_created_idx := IF(
  @has_samples_model_created_idx = 0,
  'ALTER TABLE samples ADD INDEX idx_samples_model_created (model_version, created_at)',
  'SELECT 1'
);
PREPARE stmt_samples_model_created_idx FROM @sql_samples_model_created_idx;
EXECUTE stmt_samples_model_created_idx;
DEALLOCATE PREPARE stmt_samples_model_created_idx;

CREATE TABLE IF NOT EXISTS user_sessions (
  session_token VARCHAR(96) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  wechat_code VARCHAR(128) NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_user_sessions_user (user_id),
  INDEX idx_user_sessions_expire (expires_at),
  CONSTRAINT fk_user_sessions_user FOREIGN KEY (user_id) REFERENCES users(user_id)
);

CREATE TABLE IF NOT EXISTS active_learning_tasks (
  task_id VARCHAR(64) PRIMARY KEY,
  task_date DATE NOT NULL,
  sample_id VARCHAR(64) NOT NULL,
  reason VARCHAR(64) NOT NULL,
  confidence DECIMAL(6,5) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
  meta_json JSON DEFAULT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_active_learning_date (task_date),
  INDEX idx_active_learning_status (status),
  CONSTRAINT fk_active_learning_sample FOREIGN KEY (sample_id) REFERENCES samples(sample_id)
);

CREATE TABLE IF NOT EXISTS model_evaluations (
  evaluation_id VARCHAR(64) PRIMARY KEY,
  model_version VARCHAR(64) NOT NULL,
  metrics_json JSON NOT NULL,
  confusion_matrix_json JSON DEFAULT NULL,
  calibration_json JSON DEFAULT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_model_evaluations_model_created (model_version, created_at)
);

CREATE TABLE IF NOT EXISTS risk_events (
  event_id VARCHAR(64) PRIMARY KEY,
  sample_id VARCHAR(64) NOT NULL,
  pain_risk_score DECIMAL(6,5) NOT NULL,
  pain_risk_level VARCHAR(16) NOT NULL,
  evidence_json JSON DEFAULT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_risk_events_sample_created (sample_id, created_at),
  CONSTRAINT fk_risk_events_sample FOREIGN KEY (sample_id) REFERENCES samples(sample_id)
);
