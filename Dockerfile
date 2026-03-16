# ---- Builder Stage ----
# Use the official Go image as a builder.
FROM golang:1.24-alpine AS builder

# Set the working directory inside the container.
WORKDIR /app

# Copy the Go modules files.
COPY go.mod go.sum ./

# Download the Go modules.
# This is done as a separate step to leverage Docker layer caching.
RUN go mod download

# Copy the rest of the source code.
COPY . .

# Build the application binaries.
# CGO_ENABLED=0 is used to build a statically linked binary.
# The server now includes both gRPC and HTTP gateway
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/config-cli ./cmd/cli

# ---- Final Stage ----
# Use a minimal base image for the final container.
FROM alpine:3.22

# Add ca-certificates to make TLS connections.
RUN apk add --no-cache ca-certificates

# Set the working directory.
WORKDIR /app

# Copy the migrations. The application runs them on startup.
COPY --from=builder /app/db/migrations ./db/migrations

# Copy the SQLite schema init file for fallback database support.
COPY --from=builder /app/db/sqlite_init.sql ./db/sqlite_init.sql

# Copy the seed templates. The application imports them on startup.
COPY --from=builder /app/templates ./templates

# Copy the binaries from the builder stage.
COPY --from=builder /app/server /app/server
COPY --from=builder /app/config-cli /app/config-cli

# Create a symlink for the config CLI to make it available in the PATH.
RUN ln -s /app/config-cli /usr/local/bin/config-cli

# Create the data directory for SQLite persistence.
RUN mkdir -p /app/data

# This will run the server when the container starts.
CMD ["/app/server"]
