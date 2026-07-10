.PHONY: help proto build test lint docker-run docker-stop migrate-up migrate-down clean

GGID_ROOT := $(shell pwd)
PROTO_DIR := $(GGID_ROOT)/api/proto

help:
	@echo "GGID - Identity & Access Management Suite"
	@echo ""
	@echo "Targets:"
	@echo "  proto        Generate protobuf + gRPC + OpenAPI code"
	@echo "  build        Build all services"
	@echo "  test         Run all tests"
	@echo "  lint         Run golangci-lint"
	@echo "  migrate-up   Run database migrations"
	@echo "  migrate-down Rollback last migration"
	@echo "  docker-run   Start infrastructure (PostgreSQL, Redis, NATS)"
	@echo "  docker-stop  Stop infrastructure"
	@echo "  clean        Clean build artifacts"

proto:
	@for svc in identity auth oauth policy org audit; do \\
		echo "Generating proto for $$svc..."; \\
		buf generate $(PROTO_DIR)/$$svc/v1; \\
	done

build:
	@for svc in gateway identity auth oauth policy org audit; do \\
		echo "Building $$svc..."; \\
		cd services/$$svc && go build -o bin/$$svc ./cmd/ && cd $(GGID_ROOT); \\
	done

test:
	go test -timeout 10m -cover ./...

test-race:
	go test -race -timeout 20m -cover ./...

lint:
	golangci-lint run ./...

migrate-up:
	@migrate -path ./services/identity/migrations \
		-database "postgres://ggid:ggid@localhost:5432/ggid?sslmode=disable" \
		-up

migrate-down:
	@migrate -path ./services/identity/migrations \
		-database "postgres://ggid:ggid@localhost:5432/ggid?sslmode=disable" \
		down 1

docker-run:
	docker compose -f deploy/docker-compose.yaml up -d

docker-stop:
	docker compose -f deploy/docker-compose.yaml down

clean:
	find . -name '*.bin' -delete
	find . -name 'bin' -type d -exec rm -rf {} +
