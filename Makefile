.PHONY: setup generate generate-check migrate-up dev test check build compose-config

setup:
	cd frontend && npm ci
	cd backend && go mod download

generate:
	cd backend && go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.0 --config openapi/oapi-codegen.yaml openapi/openapi.yaml
	cd backend && go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
	cd frontend && npm run generate:api

generate-check: generate
	git diff --exit-code -- backend/openapi/generated backend/db/generated frontend/types/generated

migrate-up:
	docker compose --profile tools run --rm migrate up

dev:
	docker compose -f compose.yaml -f compose.dev.yaml up --build

test:
	cd backend && go test ./...
	cd frontend && npm test

check: generate-check
	test -z "$$(cd backend && gofmt -l $$(find . -name '*.go' -not -path './openapi/generated/*' -not -path './db/generated/*'))"
	cd backend && go vet ./...
	cd backend && go test ./...
	cd frontend && npm run format:check
	cd frontend && npm run lint
	cd frontend && npm run typecheck
	cd frontend && npm test

build:
	cd backend && CGO_ENABLED=0 go build -o bin/api ./cmd/api
	cd frontend && npm run build

compose-config:
	docker compose -f compose.yaml -f compose.dev.yaml config --quiet
