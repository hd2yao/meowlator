.PHONY: up down test test-go test-py test-mini run-api run-inference fmt

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

fmt:
	cd services/api && go fmt ./...
	cd services/inference && go fmt ./...
