# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy source code
COPY . .

# Download dependencies
RUN go mod tidy

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Final stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Set timezone to WIB
ENV TZ=Asia/Makassar

# Copy binary from builder
COPY --from=builder /app/main .

# Copy web templates and static files
COPY --from=builder /app/web ./web

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Copy config example
COPY --from=builder /app/config.yaml.example ./config.yaml.example

# Expose port
EXPOSE 8080

# Run
CMD ["./main"]
