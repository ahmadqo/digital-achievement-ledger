# Stage 1: Build
FROM golang:1.23-alpine AS builder

# Install dependencies yang dibutuhkan
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o main ./cmd/server

# Stage 2: Runtime (minimal image)
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

# Set timezone Asia/Jakarta
ENV TZ=Asia/Jakarta

WORKDIR /app

# Copy binary dan migrations
COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./main"]