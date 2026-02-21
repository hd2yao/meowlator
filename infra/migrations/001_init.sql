CREATE TABLE IF NOT EXISTS users (
  user_id VARCHAR(64) PRIMARY KEY,
  wechat_openid VARCHAR(128) UNIQUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cats (
  cat_id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  name VARCHAR(64) NOT NULL,
  breed VARCHAR(64) DEFAULT NULL,
  personal_bias_json JSON DEFAULT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_cats_user FOREIGN KEY (user_id) REFERENCES users(user_id)
);

CREATE TABLE IF NOT EXISTS samples (
  sample_id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  cat_id VARCHAR(64) NOT NULL,
  image_key VARCHAR(255) NOT NULL,
  scene_tag VARCHAR(64) DEFAULT NULL,
  model_version VARCHAR(64) NOT NULL,
  edge_pred_json JSON DEFAULT NULL,
  final_pred_json JSON DEFAULT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expire_at TIMESTAMP NOT NULL,
  INDEX idx_samples_user_created (user_id, created_at),
  INDEX idx_samples_expire (expire_at),
  CONSTRAINT fk_samples_user FOREIGN KEY (user_id) REFERENCES users(user_id),
  CONSTRAINT fk_samples_cat FOREIGN KEY (cat_id) REFERENCES cats(cat_id)
);

CREATE TABLE IF NOT EXISTS feedback (
  feedback_id VARCHAR(64) PRIMARY KEY,
  sample_id VARCHAR(64) NOT NULL,
  user_id VARCHAR(64) NOT NULL,
  is_correct BOOLEAN NOT NULL,
  true_label VARCHAR(32) DEFAULT NULL,
  reliability_score DECIMAL(4,3) NOT NULL,
  training_weight DECIMAL(4,3) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_feedback_sample (sample_id),
  INDEX idx_feedback_user_created (user_id, created_at),
  CONSTRAINT fk_feedback_user FOREIGN KEY (user_id) REFERENCES users(user_id),
  CONSTRAINT fk_feedback_sample FOREIGN KEY (sample_id) REFERENCES samples(sample_id)
);

CREATE TABLE IF NOT EXISTS model_registry (
  model_version VARCHAR(64) PRIMARY KEY,
  task_scope VARCHAR(64) NOT NULL,
  metrics_json JSON NOT NULL,
  status VARCHAR(32) NOT NULL,
  rollout_ratio DECIMAL(5,4) NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS training_runs (
  run_id VARCHAR(64) PRIMARY KEY,
  dataset_version VARCHAR(64) NOT NULL,
  params_json JSON NOT NULL,
  result_json JSON NOT NULL,
  artifact_uri VARCHAR(512) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
