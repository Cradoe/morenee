# Stage 1: Build the Go binary
FROM golang:1.23 AS builder
WORKDIR /app

# Install Air for live reloading
RUN go install github.com/cosmtrek/air@v1.43.0

# Copy Go modules and install dependencies
COPY go.mod go.sum ./
RUN go mod download


# Copy the rest of the application
COPY . .

# Build the application
RUN go build -o /app/bin/api ./cmd/api

# Stage 2: Runtime image
FROM golang:1.23 AS runtime
WORKDIR /app

# Copy built binary and Air binary for hot-reloading
COPY --from=builder /app/bin/api ./bin/api
COPY --from=builder /go/bin/air /usr/local/bin/air

# Expose app port
EXPOSE 4444

# Default command for production (non-reloading)
CMD ["/app/bin/api"]

# Optional hot-reloading mode (used with docker-compose)
ENV AIR_CONFIG=.air.toml

ENTRYPOINT ["air", "--build.cmd=go build -o /tmp/bin/api ./cmd/api", "--build.bin=/tmp/bin/api", "--build.kill_delay=3000"]




