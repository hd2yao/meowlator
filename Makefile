.PHONY: up down test test-go test-py test-mini run-api run-inference fmt train-vision export-onnx

up:
	docker compose -f infra/docker-compose.yml up --build -d

down:
	docker compose -f infra/docker-compose.yml down

test: test-go test-py test-mini

test-go:
	cd services/api && go test ./...
	cd services/inference && go test ./...

test-py:
	cd ml/training && python3 -m unittest discover -s scripts -p 'test_*.py'

test-mini:
	cd apps/wechat-miniprogram && npm run typecheck

run-api:
	cd services/api && go run ./cmd/api

run-inference:
	cd services/inference && go run ./cmd/inference

train-vision:
	cd ml/training && python3 scripts/train.py --dataset-root ./data/oxford_pet --output-dir ./artifacts/mobilenetv3-small-v2 --download --epochs 3

export-onnx:
	cd ml/training && python3 scripts/export_onnx.py --checkpoint ./artifacts/mobilenetv3-small-v2/mobilenetv3-small-v2.pt --output ./artifacts/mobilenetv3-small-v2/mobilenetv3-small-v2.onnx --quantize-int8

fmt:
	cd services/api && go fmt ./...
	cd services/inference && go fmt ./...
