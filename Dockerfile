FROM golang:1.25-alpine AS builder
WORKDIR /app

# Install git (required for some Go modules)
RUN apk add --no-cache git

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./main.go

# Final image
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy binary and migrations
COPY --from=builder /app/main .
COPY --from=builder /app/docs ./docs
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /app/db/migrations ./db/migrations

# Make migrate executable
RUN chmod +x /usr/local/bin/migrate

# Expose port
EXPOSE ${APP_PORT}

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${APP_PORT}/swagger/index.html || exit 1

# Run the application
CMD ["./main"]