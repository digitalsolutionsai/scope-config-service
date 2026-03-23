.PHONY: proto-gen build-cli build-server build-httpgateway run-server run-httpgateway up down up-sqlite down-sqlite migrate-up migrate-down swagger-gen test test-go test-go-sdk test-ts-sdk docker-build docker-push

# ====================================================================================
# PROTO
# ====================================================================================
proto-gen:
	@echo "Generating gRPC code from proto files..."
	@./scripts/gen-proto.sh

# ====================================================================================
# SWAGGER
# ====================================================================================
swagger-gen:
	@echo "Generating Swagger documentation..."
	@swag init -g pkg/httpgateway/docs.go --output ./docs
	@echo "Swagger documentation generated at ./docs"

# ====================================================================================
# BUILD
# ====================================================================================
build-cli:
	@echo "Building CLI..."
	@go build -o bin/config-cli ./cmd/cli
	@echo "Building CLI done."
build-server:
	@echo "Building server..."
	@go build -o bin/server ./cmd/server
	@echo "Building server done."

build-httpgateway:
	@echo "Building HTTP gateway..."
	@go build -o bin/httpgateway ./cmd/httpgateway
	@echo "Building HTTP gateway done."

# ====================================================================================
# TEST
# ====================================================================================
test: test-go test-go-sdk test-ts-sdk
	@echo "All tests passed."

test-go:
	@echo "Running Go backend tests..."
	@go test -v -short ./pkg/...
	@echo "Go backend tests done."

test-go-sdk:
	@echo "Running Go SDK tests..."
	@cd sdks/go && go test -v -short ./...
	@echo "Go SDK tests done."

test-ts-sdk:
	@echo "Running TypeScript SDK tests..."
	@cd sdks/typescript && npm test
	@echo "TypeScript SDK tests done."

# ====================================================================================
# RUN
# ====================================================================================
run-server: build-server
	@echo "Running server..."
	@./bin/server

run-httpgateway: build-httpgateway
	@echo "Running HTTP gateway..."
	@./bin/httpgateway

# ====================================================================================
# DOCKER
# ====================================================================================
DOCKERHUB_REPO ?= dsailoivo/scope-config
TAG ?= latest

up:
	@echo "Starting services with Docker Compose (PostgreSQL)..."
	@docker compose -f compose.postgres.yml -f compose.yml up -d

up-build:
	@echo "Starting services with Docker Compose (PostgreSQL, rebuild)..."
	@docker compose -f compose.postgres.yml -f compose.yml up -d --build

up-sqlite:
	@echo "Starting services with Docker Compose (SQLite)..."
	@docker compose -f compose.sqlite.yml up -d

up-sqlite-build:
	@echo "Starting services with Docker Compose (SQLite, rebuild)..."
	@docker compose -f compose.sqlite.yml up -d --build

down:
	@echo "Stopping services..."
	@docker compose down

down-sqlite:
	@echo "Stopping SQLite services..."
	@docker compose -f compose.sqlite.yml down

ps:
	@echo "Listing running containers..."
	@docker compose ps

docker-build:
	@echo "Building Docker image $(DOCKERHUB_REPO):$(TAG)..."
	@docker build -t $(DOCKERHUB_REPO):$(TAG) .
	@echo "Docker build done."

docker-push: docker-build
	@echo "Pushing $(DOCKERHUB_REPO):$(TAG) to Docker Hub..."
	@docker push $(DOCKERHUB_REPO):$(TAG)
	@echo "Docker push done."

# ====================================================================================
# DATABASE
# ====================================================================================
migrate-up:
	@echo "Applying database migrations..."
	@migrate -database "postgres://$(shell sed -n 's/POSTGRES_USER=//p' .env):$(shell sed -n 's/POSTGRES_PASSWORD=//p' .env)@localhost:5555/$(shell sed -n 's/POSTGRES_DB=//p' .env)?sslmode=disable" -path db/migrations up

migrate-down:
	@echo "Reverting database migrations..."
	@migrate -database "postgres://$(shell sed -n 's/POSTGRES_USER=//p' .env):$(shell sed -n 's/POSTGRES_PASSWORD=//p' .env)@localhost:5555/$(shell sed -n 's/POSTGRES_DB=//p' .env)?sslmode=disable" -path db/migrations down
