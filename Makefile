.PHONY: up down up-observability up-grafana down-observability test test-go test-py test-mini smoke-local run-api run-inference fmt train-vision train-vision-smoke train-vision-resume export-onnx clean-feedback-data build-training-manifest active-learning-daily build-eval-splits threshold-report evaluate-intent gate-model

RESUME_CHECKPOINT ?= ./artifacts/mobilenetv3-small-v2/mobilenetv3-small-v2.pt

up:
	docker compose -f infra/docker-compose.yml up --build -d

down:
	docker compose -f infra/docker-compose.yml down

up-observability:
	docker compose -f infra/docker-compose.yml up -d prometheus alertmanager

up-grafana:
	docker compose -f infra/docker-compose.yml up -d grafana

down-observability:
	docker compose -f infra/docker-compose.yml stop prometheus alertmanager grafana

test: test-go test-py test-mini

test-go:
	cd services/api && go test ./...
	cd services/inference && go test ./...

test-py:
	cd ml/training && python3 -m unittest discover -s scripts -p 'test_*.py'

test-mini:
	cd apps/wechat-miniprogram && npm run typecheck

smoke-local:
	bash tools/smoke_local_flow.sh

run-api:
	cd services/api && go run ./cmd/api

run-inference:
	cd services/inference && go run ./cmd/inference

train-vision:
	cd ml/training && python3 scripts/train.py --dataset-root ./data/oxford_pet --output-dir ./artifacts/mobilenetv3-small-v2 --download --epochs 3

train-vision-smoke:
	cd ml/training && python3 scripts/train.py --dataset fake --dataset-root ./data/oxford_pet --output-dir ./artifacts/mobilenetv3-small-v2-smoke --epochs 1 --batch-size 16

train-vision-resume:
	cd ml/training && python3 scripts/train.py --dataset-root ./data/oxford_pet --output-dir ./artifacts/mobilenetv3-small-v2 --resume-checkpoint $(RESUME_CHECKPOINT) --epochs 1 --batch-size 32

clean-feedback-data:
	cd ml/training && python3 scripts/data_cleaning.py --input ./data/feedback/raw_feedback.jsonl --output ./data/feedback/clean_feedback.jsonl --report ./artifacts/pipeline/cleaning_report.json

build-training-manifest:
	cd ml/training && python3 scripts/build_training_manifest.py --public-manifest ./data/public/public_manifest.jsonl --feedback ./data/feedback/clean_feedback.jsonl --output ./artifacts/pipeline/training_manifest.jsonl --report ./artifacts/pipeline/manifest_report.json

active-learning-daily:
	cd ml/training && python3 scripts/generate_active_learning_tasks.py --pool ./data/feedback/candidate_pool.jsonl --daily-budget 100 --output ./artifacts/pipeline/active_learning_tasks.json

build-eval-splits:
	cd ml/training && python3 scripts/build_eval_splits.py --input ./artifacts/pipeline/training_manifest.jsonl --output-dir ./artifacts/pipeline/splits --seed 42 --train-ratio 0.7 --val-ratio 0.15

threshold-report:
	cd ml/training && python3 scripts/threshold_report.py --input ./data/feedback/candidate_pool.jsonl --output ./artifacts/pipeline/threshold_report.json

evaluate-intent:
	cd ml/training && python3 scripts/evaluate_intent_metrics.py --input ./artifacts/pipeline/predictions_eval.jsonl --output ./artifacts/pipeline/intent_eval_report.json --low-conf-threshold 0.45

gate-model:
	cd ml/training && python3 scripts/gate_model_release.py --baseline ./artifacts/mobilenetv3-small-v2/metrics.json --candidate ./artifacts/mobilenetv3-small-v2-smoke/metrics.json --output ./artifacts/pipeline/gate_report.json

export-onnx:
	cd ml/training && python3 scripts/export_onnx.py --checkpoint ./artifacts/mobilenetv3-small-v2/mobilenetv3-small-v2.pt --output ./artifacts/mobilenetv3-small-v2/mobilenetv3-small-v2.onnx --quantize-int8

fmt:
	cd services/api && go fmt ./...
	cd services/inference && go fmt ./...
