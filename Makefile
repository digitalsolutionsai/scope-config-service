.PHONY: proto-gen build-cli build-server build-httpgateway run-server run-httpgateway up down migrate-up migrate-down

# ====================================================================================
# PROTO
# ====================================================================================
proto-gen:
	@echo "Generating gRPC code from proto files..."
	@./scripts/gen-proto.sh

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
	@migrate -database "postgres://$(shell sed -n 's/POSTGRES_USER=//p' .env):$(shell sed -n 's/POSTGRES_PASSWORD=//p' .env)@localhost:5555/$(shell sed -n 's/POSTGRES_DB=//p' .env)?sslmode=disable" -path db/migrations up

migrate-down:
	@echo "Reverting database migrations..."
	@migrate -database "postgres://$(shell sed -n 's/POSTGRES_USER=//p' .env):$(shell sed -n 's/POSTGRES_PASSWORD=//p' .env)@localhost:5555/$(shell sed -n 's/POSTGRES_DB=//p' .env)?sslmode=disable" -path db/migrations down
