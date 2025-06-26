# Stage 1: Build the Go application and the migrate tool
FROM golang:1.23 AS builder

# Install migrate tool
# Using a specific version for reproducibility
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1

WORKDIR /app
COPY ../../go.mod ../../go.sum ./
RUN go mod download

COPY ../../ ./
RUN go build -o main ./cmd/api/main.go

# Stage 2: Create the final lightweight image
FROM debian:bookworm-slim
WORKDIR /app

# Install postgresql-client for pg_isready and ca-certificates for https connections if needed by app later
RUN apt-get update && apt-get install -y postgresql-client ca-certificates && rm -rf /var/lib/apt/lists/*

# Copy the compiled application binary from the builder stage
COPY --from=builder /app/main .
# Copy the migrate tool from the builder stage (GOPATH/bin is the default install location)
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
# Copy migration files
COPY ../../migrations ./migrations
# Copy the entrypoint script
COPY ../../scripts/entrypoint.sh /app/entrypoint.sh
# Make the entrypoint script executable
RUN chmod +x /app/entrypoint.sh

# Set the entrypoint for the container
ENTRYPOINT ["/app/entrypoint.sh"]
# Default command to run (will be executed by entrypoint.sh)
CMD ["./main"]