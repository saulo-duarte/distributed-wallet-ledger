.PHONY: build test test-unit test-integration vet fmt sqlc-generate compose-up compose-down compose-config migrate-up migrate-down

DATABASE_URL ?= postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable
MIGRATE ?= migrate

build:
	go build ./...

test:
	go test ./...

test-unit:
	go test ./internal/ledger/domain/...

test-integration:
	go test -tags=integration ./...

vet:
	go vet ./...

fmt:
	gofmt -w ./cmd ./internal

sqlc-generate:
	sqlc generate

compose-up:
	docker compose up -d postgres

compose-down:
	docker compose down

compose-config:
	docker compose config

migrate-up:
	$(MIGRATE) -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	$(MIGRATE) -path migrations -database "$(DATABASE_URL)" down 1
