#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TRAINING_DIR="${ROOT_DIR}/ml/training"

required_inputs=(
  "${TRAINING_DIR}/data/feedback/raw_feedback.jsonl"
  "${TRAINING_DIR}/data/public/public_manifest.jsonl"
  "${TRAINING_DIR}/data/feedback/candidate_pool.jsonl"
)

missing=()
for input in "${required_inputs[@]}"; do
  if [[ ! -f "${input}" ]]; then
    missing+=("${input}")
  fi
done

if (( ${#missing[@]} > 0 )); then
  echo "[training-daily] skip: missing required inputs"
  for input in "${missing[@]}"; do
    echo "  - ${input}"
  done
  exit 0
fi

cd "${TRAINING_DIR}"

mkdir -p ./artifacts/pipeline

echo "[training-daily] data_cleaning"
python3 scripts/data_cleaning.py \
  --input ./data/feedback/raw_feedback.jsonl \
  --output ./data/feedback/clean_feedback.jsonl \
  --report ./artifacts/pipeline/cleaning_report.json

echo "[training-daily] build_training_manifest"
python3 scripts/build_training_manifest.py \
  --public-manifest ./data/public/public_manifest.jsonl \
  --feedback ./data/feedback/clean_feedback.jsonl \
  --output ./artifacts/pipeline/training_manifest.jsonl \
  --report ./artifacts/pipeline/manifest_report.json

echo "[training-daily] generate_active_learning_tasks"
python3 scripts/generate_active_learning_tasks.py \
  --pool ./data/feedback/candidate_pool.jsonl \
  --daily-budget 100 \
  --output ./artifacts/pipeline/active_learning_tasks.json

baseline_metrics="./artifacts/mobilenetv3-small-v2/metrics.json"
candidate_metrics="./artifacts/mobilenetv3-small-v2-smoke/metrics.json"
if [[ -f "${baseline_metrics}" && -f "${candidate_metrics}" ]]; then
  echo "[training-daily] gate_model_release"
  python3 scripts/gate_model_release.py \
    --baseline "${baseline_metrics}" \
    --candidate "${candidate_metrics}" \
    --output ./artifacts/pipeline/gate_report.json
else
  echo "[training-daily] skip gate_model_release: metrics files missing"
  echo "  - ${baseline_metrics}"
  echo "  - ${candidate_metrics}"
fi

echo "[training-daily] done"
