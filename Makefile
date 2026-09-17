.PHONY: build test test-all test-unit test-integration test-fuzz test-mutation vet fmt sqlc-generate infra-up infra-down infra-reset compose-up compose-down compose-config migrate-up migrate-down

DATABASE_URL ?= postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable
MIGRATIONS_PATH ?= migrations
MIGRATE ?= migrate

build:
	go build ./...

test:
	go test ./...

test-all:
	scripts\test-all.cmd

test-unit:
	go test ./internal/ledger/domain/...

test-integration:
	scripts\test-integration.cmd

test-fuzz:
	scripts\test-fuzz.cmd

test-mutation:
	scripts\test-mutation.cmd

vet:
	go vet ./...

fmt:
	gofmt -w ./cmd ./internal

sqlc-generate:
	scripts\sqlc-generate.cmd

infra-up:
	docker compose up -d --wait postgres

infra-down:
	docker compose down

infra-reset:
	docker compose down -v

compose-up: infra-up

compose-down: infra-down

compose-config:
	docker compose config

migrate-up:
	$(MIGRATE) -path "$(MIGRATIONS_PATH)" -database "$(DATABASE_URL)" up

migrate-down:
	$(MIGRATE) -path "$(MIGRATIONS_PATH)" -database "$(DATABASE_URL)" down 1
