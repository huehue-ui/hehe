FROM golang:1.23 AS builder
WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -o main .

FROM debian:bookworm-slim
WORKDIR /app

RUN apt-get update && apt-get install -y postgresql-client && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/main .
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["./main"]