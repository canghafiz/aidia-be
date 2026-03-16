FROM golang:1.25-alpine AS builder
WORKDIR /app

RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -o main ./main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/docs ./docs
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /app/db/migrations ./db/migrations

EXPOSE ${APP_PORT}