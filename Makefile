.PHONY: setup auth-keys generate generate-check migrate-up dev dev-down test test-unit test-integration check build compose-config

setup:
	cd frontend && npm ci
	cd backend && go mod download

auth-keys:
	cd backend && go run ./cmd/authkeys

generate:
	cd backend && go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.0 --config openapi/oapi-codegen.yaml openapi/openapi.yaml
	cd backend && go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
	cd frontend && npm run generate:api

generate-check: generate
	git diff --exit-code -- backend/openapi/generated backend/db/generated frontend/types/generated

migrate-up:
	docker compose --profile tools run --build --rm migrate up

dev:
	docker compose -f compose.yaml -f compose.dev.yaml up --build

dev-down:
	docker compose -f compose.yaml -f compose.dev.yaml down --remove-orphans

test:
	$(MAKE) test-unit
	$(MAKE) test-integration

test-unit:
	cd backend && go test ./...
	cd frontend && npm test

test-integration:
	cd backend && go test -tags=integration ./integration

check: generate-check
	test -z "$$(cd backend && gofmt -l $$(find . -name '*.go' -not -path './openapi/generated/*' -not -path './db/generated/*'))"
	cd backend && go vet -tags=integration ./...
	cd backend && go test ./...
	cd backend && go test -tags=integration ./integration
	cd frontend && npm run format:check
	cd frontend && npm run lint
	cd frontend && npm run typecheck
	cd frontend && npm test

build:
	cd backend && CGO_ENABLED=0 go build -o bin/api ./cmd/api
	cd frontend && npm run build

compose-config:
	docker compose -f compose.yaml -f compose.dev.yaml config --quiet
