.PHONY: proto build-cli build-server run-server up down migrate-up migrate-down

# ====================================================================================
# PROTO
# ====================================================================================
proto:
	@echo "Generating gRPC code from proto files..."
	@./scripts/gen-proto.sh

# ====================================================================================
# BUILD
# ====================================================================================
build-cli:
	@echo "Building CLI..."
	@go build -o bin/config-cli ./cmd/cli

build-server:
	@echo "Building server..."
	@go build -o bin/server ./cmd/server


# ====================================================================================
# RUN
# ====================================================================================
run-server: build-server
	@echo "Running server..."
	@./bin/server

# ====================================================================================
# DOCKER
# ====================================================================================
up:
	@echo "Starting services with Docker Compose..."
	@docker compose -f compose.postgres.yml -f compose.yml up -d --build

down:
	@echo "Stopping services..."
	@docker compose -f compose.postgres.yml -f compose.yml down

# ====================================================================================
# DATABASE
# ====================================================================================
migrate-up:
	@echo "Applying database migrations..."
	@migrate -database "postgres://$(shell sed -n 's/POSTGRES_USER=//p' .env):$(shell sed -n 's/POSTGRES_PASSWORD=//p' .env)@localhost:5432/$(shell sed -n 's/POSTGRES_DB=//p' .env)?sslmode=disable" -path db/migrations up

migrate-down:
	@echo "Reverting database migrations..."
	@migrate -database "postgres://$(shell sed -n 's/POSTGRES_USER=//p' .env):$(shell sed -n 's/POSTGRES_PASSWORD=//p' .env)@localhost:5432/$(shell sed -n 's/POSTGRES_DB=//p' .env)?sslmode=disable" -path db/migrations down
