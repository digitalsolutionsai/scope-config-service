# Dockerfile for scope-config-service

# ---- Builder Stage ----
# Use the official Go image as a builder.
FROM golang:1.24-alpine AS builder

# Set the working directory inside the container.
WORKDIR /app

# Add Go's bin directory to the PATH.
# This is necessary so that tools installed via `go install` are available.
ENV PATH="/go/bin:${PATH}"

# Install the build tools.
RUN go install github.com/bufbuild/buf/cmd/buf@latest
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Copy Go module and sum files.
COPY go.mod go.sum ./

# Download Go module dependencies.
RUN go mod download

# Copy the rest of the application source code.
COPY . .

# Generate the protobuf Go code.
RUN buf generate

# Build the application binaries.
# CGO_ENABLED=0 is used to build a statically linked binary.
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/config ./cmd/cli

# ---- Final Stage ----
# Use a minimal base image for the final container.
FROM alpine:3

# Add ca-certificates to make TLS connections.
RUN apk add --no-cache ca-certificates

# Set the working directory.
WORKDIR /app

# Copy the migrations. The application runs them on startup.
COPY --from=builder /app/db/migrations ./db/migrations

# Copy the binaries from the builder stage.
COPY --from=builder /app/server /app/server
COPY --from=builder /app/config /app/config

# Create a symlink for the config CLI to make it available in the PATH.
RUN ln -s /app/config /usr/local/bin/config

# This will run the server when the container starts.
CMD ["/app/server"]
