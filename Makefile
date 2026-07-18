.PHONY: help proto build test test-short test-race coverage lint lint-ci install-hooks docker-run docker-stop docker-build docker-build-allinone docker-push docker-build-services docker-push-services swagger-gen migrate-up migrate-down clean

GGID_ROOT := $(shell pwd)
PROTO_DIR := $(GGID_ROOT)/api/proto

help:
	@echo "GGID - Identity & Access Management Suite"
	@echo ""
	@echo "Targets:"
	@echo "  proto        Generate protobuf + gRPC + OpenAPI code"
	@echo "  build        Build all services"
	@echo "  test         Run all tests"
	@echo "  test-short   Run tests (short mode, 2m timeout)"
	@echo "  coverage     Run tests + generate HTML coverage report"
	@echo "  test-race    Run tests with race detector"
	@echo "  lint         Run golangci-lint"
	@echo "  migrate-up   Run database migrations"
	@echo "  migrate-down Rollback last migration"
	@echo "  lint-ci      CI lint gate (go build + go vet)"
	@echo "  install-hooks  Install git pre-commit hook"
	@echo "  docker-run   Start infrastructure (PostgreSQL, Redis, NATS)"
	@echo "  docker-stop  Stop infrastructure"
	@echo "  docker-build-services  Build all 8 service images (prebuilt binary)"
	@echo "  docker-push-services   Build + push all 8 service images"
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
	go test -timeout 10m -cover $(shell go list ./... | grep -v '/sdk/examples/' | grep -v '/node_modules/' | grep -v '^github.com/ggid/ggid$$')

test-short:
	go test -timeout 2m -short $(shell go list ./... | grep -v '/sdk/examples/' | grep -v '/node_modules/' | grep -v '^github.com/ggid/ggid$$')

coverage:
	go test -timeout 10m -coverprofile=coverage.out $(shell go list ./... | grep -v '/sdk/examples/' | grep -v '/node_modules/' | grep -v '^github.com/ggid/ggid$$')
	go tool cover -func=coverage.out | grep total
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

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

docker-build:
	docker build --platform linux/amd64 -f console/Dockerfile -t ggid/ggid-console:latest .

docker-build-allinone:
	docker build --platform linux/amd64 -f deploy/all-in-one/Dockerfile -t ggid/ggid-all-in-one:latest .

docker-push: docker-build-allinone
	docker tag ggid/ggid-all-in-one:latest registry.iot2.win/ggid/all-in-one:latest
	docker push registry.iot2.win/ggid/all-in-one:latest

docker-build-services:
	@bash scripts/build-all-images.sh --no-push --console

docker-push-services:
	@bash scripts/build-all-images.sh --console

swagger-gen:
	@echo "Generating OpenAPI spec from annotations..."
	@which swag > /dev/null 2>&1 || go install github.com/swaggo/swag/v2/cmd/swag@latest
	swag init -g services/auth/internal/server/http.go --output docs/swagger/auth --parseDependency
	swag init -g services/identity/internal/server/http.go --output docs/swagger/identity --parseDependency
	swag init -g services/oauth/internal/server.go --output docs/swagger/oauth --parseDependency
	@echo "Swagger specs generated under docs/swagger/"

lint-ci:
	@echo "Running CI lint gate (build + vet)..."
	go build ./... && go vet ./... && echo "lint-ci OK"

install-hooks:
	@mkdir -p .git/hooks
	@cp scripts/hooks/pre-commit .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Pre-commit hook installed: build + vet gate"

clean:
	find . -name '*.bin' -delete
	find . -name 'bin' -type d -exec rm -rf {} +

swagger:
	@bash scripts/gen-swagger.sh
	@echo ">> Swagger spec generated at deploy/openapi.yaml"

console-build:
	cd console && npm ci && npm run build

console-deploy: console-build
	docker build --platform linux/amd64 -f console/Dockerfile -t registry.iot2.win/ggid/console:latest .
	docker push registry.iot2.win/ggid/console:latest
