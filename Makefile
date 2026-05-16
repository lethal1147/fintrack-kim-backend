BINARY     = bin/api
MAIN       = ./cmd/api
MIGRATE    = $(shell which migrate 2>/dev/null || echo "migrate")
DB_URL     = postgres://fintrack:fintrack@localhost:5433/fintrack?sslmode=disable
MIGRATIONS = migrations

.PHONY: dev build test test-cover lint migrate-up migrate-down docker-up docker-down swagger

dev:
	go run $(MAIN)

build:
	go build -o $(BINARY) $(MAIN)

test:
	go test ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	golangci-lint run

migrate-up:
	$(MIGRATE) -path $(MIGRATIONS) -database "$(DB_URL)" up

migrate-down:
	$(MIGRATE) -path $(MIGRATIONS) -database "$(DB_URL)" down 1

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

swagger:
	swag init -g $(MAIN)/main.go --output docs
